package main

import (
	"fmt"
	"net"
	"strconv"

	"github.com/caarlos0/env/v11"
)

// config holds the settings read from environment variables.
// Gin validates GIN_MODE itself when the package loads; this only changes its default to release.
type config struct {
	GinMode string `env:"GIN_MODE" envDefault:"release"`
	Port    uint16 `env:"PORT"     envDefault:"8080"`
}

// loadConfig parses and validates environ, the process environment as a map.
func loadConfig(environ map[string]string) (config, error) {
	cfg, err := env.ParseAsWithOptions[config](env.Options{Environment: environ})
	if err != nil {
		return config{}, fmt.Errorf("parse env: %w", err)
	}

	return cfg, nil
}

func (c config) listenAddr() string {
	return net.JoinHostPort("", strconv.FormatUint(uint64(c.Port), 10))
}
