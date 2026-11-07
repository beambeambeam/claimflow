package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/beambeambeam/claimflow/apps/api/env"
)

// minBcryptCost is the lowest bcrypt cost NFR-01 allows.
const minBcryptCost = 10

// loadConfig reads the env that `varlock run` injects. Varlock validates it against .env.schema first,
// and loadConfig checks the security limits again, so the API never starts below them.
func loadConfig() (env.Env, error) {
	cfg, err := env.Load()
	if err != nil {
		return env.Env{}, fmt.Errorf("load env: %w", err)
	}

	if cfg.BcryptCost < minBcryptCost {
		return env.Env{}, fmt.Errorf("load env: BCRYPT_COST must be at least %d, got %d", minBcryptCost, cfg.BcryptCost)
	}

	return cfg, nil
}

func listenAddr(port int64) string {
	return net.JoinHostPort("", strconv.FormatInt(port, 10))
}
