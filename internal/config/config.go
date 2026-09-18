// Package config loads iamd's configuration from environment variables.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config is iamd's full runtime configuration, populated from environment
// variables (see ADR 0007). All fields have safe defaults except JWTIssuer.
type Config struct {
	ListenAddr     string        `env:"IAM_LISTEN_ADDR" envDefault:":8080"`
	Namespace      string        `env:"IAM_NAMESPACE" envDefault:"iam-system"`
	JWTIssuer      string        `env:"IAM_JWT_ISSUER,required"`
	JWTAudience    string        `env:"IAM_JWT_AUDIENCE"`
	AccessTokenTTL time.Duration `env:"IAM_ACCESS_TOKEN_TTL" envDefault:"15m"`
	KubeconfigPath string        `env:"KUBECONFIG"`
	LogLevel       string        `env:"IAM_LOG_LEVEL" envDefault:"info"`
}

// Load reads and validates Config from the process environment.
func Load() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("loading config: %w", err)
	}
	if cfg.AccessTokenTTL <= 0 {
		return Config{}, fmt.Errorf("IAM_ACCESS_TOKEN_TTL must be positive, got %s", cfg.AccessTokenTTL)
	}
	return cfg, nil
}
