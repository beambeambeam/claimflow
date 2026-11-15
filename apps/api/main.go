package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// main serves the API, or runs `api migrate up|down|status` or `api seed`.
func main() { // coverage-ignore
	if err := runUntilSignal(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "api:", err)
		os.Exit(1)
	}
}

// runUntilSignal runs the command in args until it ends, or until SIGINT or SIGTERM.
func runUntilSignal(args []string) error { // coverage-ignore
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(args) > 0 && args[0] == "migrate" {
		return migrate(ctx, cfg.DatabaseUrl, args[1:], os.Stdout)
	}

	if len(args) > 0 && args[0] == "seed" {
		password := ""
		if cfg.SeedUserPassword != nil {
			password = *cfg.SeedUserPassword
		}

		return seedCommand(ctx, cfg.DatabaseUrl, password, int(cfg.BcryptCost), os.Stdout)
	}

	return run(ctx, &cfg, newLogger(os.Stdout, cfg.LogLevel))
}
