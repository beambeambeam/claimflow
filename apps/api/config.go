package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/beambeambeam/karupan/apps/api/env"
)

// loadConfig reads the env that `varlock run` injects. Varlock validates it against .env.schema first.
func loadConfig() (env.Env, error) {
	cfg, err := env.Load()
	if err != nil {
		return env.Env{}, fmt.Errorf("load env: %w", err)
	}

	return cfg, nil
}

func listenAddr(cfg env.Env) string {
	return net.JoinHostPort("", strconv.FormatInt(cfg.Port, 10))
}
