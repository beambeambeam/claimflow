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

	return serve(ctx, listener, newRouter(logger, cfg.AppVersion, pool), logger)
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

// server composes the feature handlers into the generated strict server interface.
type server struct {
	system.Handler
}

// newRouter builds the router: request ID and log, problem responses, then the contract's operations.
func newRouter(logger *slog.Logger, version string, pool db.DBTX) *gin.Engine {
	router := gin.New()
	router.Use(httperr.RequestID(), requestLogger(logger), httperr.Handle(logger))
	router.NoRoute(httperr.NotFound)

	handler := api.NewStrictHandlerWithOptions(server{
		Handler: system.NewHandler(version, db.New(pool)),
	}, nil, api.StrictGinServerOptions{
		HandlerErrorFunc:         httperr.Report,
		ResponseErrorHandlerFunc: httperr.Report,
	})

	api.RegisterHandlersWithOptions(router, handler, api.GinServerOptions{BaseURL: basePath})

	return router
}

// newLogger writes JSON lines with the field names Cloud Logging reads (`severity`, `message`).
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
				case slog.MessageKey:
					attr.Key = "message"
				}
			}

			return attr
		},
	}))
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
