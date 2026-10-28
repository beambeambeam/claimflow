package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v11"
)

func main() { // coverage-ignore
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := runUntilSignal(logger); err != nil {
		logger.Error("server stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

// runUntilSignal runs the server until SIGINT or SIGTERM.
func runUntilSignal(logger *slog.Logger) error { // coverage-ignore
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return run(ctx, env.ToMap(os.Environ()), logger)
}
