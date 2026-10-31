package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/beambeambeam/claimflow/apps/api/env"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
)

// run serves the API with cfg until ctx is canceled.
func run(ctx context.Context, cfg env.Env, logger *slog.Logger) error {
	gin.SetMode(cfg.GinMode)

	addr := listenAddr(cfg)

	listener, err := new(net.ListenConfig).Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	return serve(ctx, listener, newRouter(logger), logger)
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

func newRouter(logger *slog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(requestLogger(logger), recoverer(logger))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}

func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		logger.LogAttrs(c.Request.Context(), slog.LevelInfo, "request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}

func recoverer(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		logger.LogAttrs(c.Request.Context(), slog.LevelError, "panic recovered",
			slog.Any("panic", recovered),
			slog.String("path", c.Request.URL.Path),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
