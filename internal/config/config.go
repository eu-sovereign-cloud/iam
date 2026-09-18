// Package config loads iamd's configuration from environment variables.
package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config is iamd's full runtime configuration, populated from environment
// variables (see ADR 0007). All fields have safe defaults except JWTIssuer.
type Config struct {
	ListenAddr     string `env:"IAM_LISTEN_ADDR" envDefault:":8080"`
	Namespace      string `env:"IAM_NAMESPACE" envDefault:"iam-system"`
	JWTIssuer      string `env:"IAM_JWT_ISSUER,required"`
	JWTAudience    string `env:"IAM_JWT_AUDIENCE"`
	KubeconfigPath string `env:"KUBECONFIG"`
	LogLevel       string `env:"IAM_LOG_LEVEL" envDefault:"info"`
}

// Load reads and validates Config from the process environment.
func Load() (Config, error) {
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}
