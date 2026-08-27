// Package config loads process-level configuration from environment variables.
package config

import (
	"fmt"
	"os"
)

// AppConfig holds startup configuration read from the environment.
type AppConfig struct {
	GitHubToken string
	DataDir     string
	Port        string
}

// Load reads AppConfig from the environment, failing fast if GITHUB_TOKEN is unset.
func Load() (AppConfig, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return AppConfig{}, fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return AppConfig{GitHubToken: token, DataDir: dataDir, Port: port}, nil
}
