package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beambeambeam/claimflow/apps/api/env"
	"github.com/beambeambeam/claimflow/apps/api/internal/api"
	"github.com/beambeambeam/claimflow/apps/api/internal/auth"
	"github.com/beambeambeam/claimflow/apps/api/internal/claim"
	"github.com/beambeambeam/claimflow/apps/api/internal/db"
	"github.com/beambeambeam/claimflow/apps/api/internal/httperr"
	"github.com/beambeambeam/claimflow/apps/api/internal/system"
)

const (
	basePath          = "/api/v1"
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// run serves the API with cfg until ctx is canceled.
func run(ctx context.Context, cfg *env.Env, logger *slog.Logger) error {
	gin.SetMode(cfg.GinMode)

	// The pool connects lazily, so the API starts while the database is down and /ready reports it.
	pool, err := pgxpool.New(ctx, cfg.DatabaseUrl)
	if err != nil {
		return fmt.Errorf("DATABASE_URL: %w", err)
	}
	defer pool.Close()

	addr := listenAddr(cfg.Port)

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	return serve(ctx, listener, newRouter(logger, cfg, pool), logger)
}

// serve handles requests on listener until ctx is canceled, then shuts down gracefully.
func serve(ctx context.Context, listener net.Listener, handler http.Handler, logger *slog.Logger) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()
	logger.Info("server started", slog.String("addr", listener.Addr().String()))

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil { // coverage-ignore
		return fmt.Errorf("shut down server: %w", err)
	}

	if err := <-serveErr; !errors.Is(err, http.ErrServerClosed) { // coverage-ignore
		return fmt.Errorf("serve: %w", err)
	}

	logger.Info("server stopped")

	return nil
}

// authHandler and claimHandler name the feature handlers here, because both are called Handler and
// embedding them under that name would clash.
type (
	authHandler  = auth.Handler
	claimHandler = claim.Handler
)

// server composes the feature handlers into the generated strict server interface.
type server struct {
	system.Handler
	authHandler
	claimHandler
}

// newRouter builds the router: request ID and log, problem responses, cross-origin guard, session user,
// the session check for every route but the public operations, then the contract's operations with their route permissions.
func newRouter(logger *slog.Logger, cfg *env.Env, pool claim.Pool) *gin.Engine {
	return newRouterAt(logger, cfg, pool, time.Now)
}

// newRouterAt is newRouter with the clock that the auth and claim rules read. Tests pass a fixed clock.
func newRouterAt(logger *slog.Logger, cfg *env.Env, pool claim.Pool, now func() time.Time) *gin.Engine {
	queries := db.New(pool)
	authService := auth.NewService(queries, int(cfg.BcryptCost), now)

	router := gin.New()
	// Handlers get the gin context; the fallback lets them read values that middleware put in the request context.
	router.ContextWithFallback = true
	router.Use(httperr.RequestID(), requestLogger(logger), httperr.Handle(logger), crossOriginGuard(),
		auth.Session(authService, cfg.CookieSecure), auth.RequireSession(contractPublicRoutes()), requestValidation(contractSpec(), validatedOperations))
	router.NoRoute(httperr.NotFound)

	handler := api.NewStrictHandlerWithOptions(server{
		Handler:      system.NewHandler(cfg.AppVersion, queries),
		authHandler:  auth.NewHandler(authService, cfg.CookieSecure),
		claimHandler: claim.NewHandler(claim.NewService(pool, now)),
	}, []api.StrictMiddlewareFunc{auth.OperationPermissions(operationPermissions)}, api.StrictGinServerOptions{
		RequestErrorHandlerFunc:  httperr.ReportInvalid,
		HandlerErrorFunc:         httperr.Report,
		ResponseErrorHandlerFunc: httperr.Report,
	})

	api.RegisterHandlersWithOptions(router, handler, api.GinServerOptions{
		BaseURL: basePath,
		// A query or path parameter that does not bind, such as page=abc, is a 400 problem like any other bad request.
		ErrorHandler: func(c *gin.Context, err error, _ int) { httperr.ReportInvalid(c, err) },
	})

	return router
}

// newLogger writes JSON lines with the field names Cloud Logging reads (`severity`, `message`).
// Cloud Logging knows DEBUG, INFO, and ERROR, but calls the warn level WARNING.
func newLogger(w io.Writer, level string) *slog.Logger {
	var minLevel slog.Level
	_ = minLevel.UnmarshalText([]byte(level)) // varlock has already checked the value

	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: minLevel,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if len(groups) == 0 {
				switch attr.Key {
				case slog.LevelKey:
					attr.Key = "severity"
					if level, ok := attr.Value.Any().(slog.Level); ok && level == slog.LevelWarn {
						attr.Value = slog.StringValue("WARNING")
					}
				case slog.MessageKey:
					attr.Key = "message"
				}
			}

			return attr
		},
	}))
}

// crossOriginGuard refuses unsafe browser requests that come from another origin (F-01 §1.5, CSRF).
// The zero CrossOriginProtection trusts no origin. Safe methods always pass.
func crossOriginGuard() gin.HandlerFunc {
	var protection http.CrossOriginProtection

	return func(c *gin.Context) {
		if err := protection.Check(c.Request); err != nil {
			httperr.Report(c, httperr.ErrForbidden)
			c.Abort()

			return
		}

		c.Next()
	}
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		logger.LogAttrs(c.Request.Context(), slog.LevelInfo, "request",
			slog.String("requestId", httperr.RequestIDFrom(c)),
			slog.String("method", c.Request.Method),
			slog.String("route", c.FullPath()),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
		)
	}
}
