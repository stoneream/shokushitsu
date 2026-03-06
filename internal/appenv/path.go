package appenv

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultHomeDirName = ".shoku"
	defaultDBFileName  = "shoku.db"
	defaultConfigFile  = "config.toml"
)

// HomeDir returns SHOKU_HOME if present, otherwise ~/.shoku.
func HomeDir() (string, error) {
	if v, ok := os.LookupEnv("SHOKU_HOME"); ok && v != "" {
		return filepath.Clean(v), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(home, defaultHomeDirName), nil
}

// DBPath ensures the home directory exists and returns a SQLite database path.
func DBPath() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(home, 0o755); err != nil {
		return "", fmt.Errorf("create shoku home directory %q: %w", home, err)
	}

	return filepath.Join(home, defaultDBFileName), nil
}

// ConfigPath returns the config file path (~/.shoku/config.toml by default).
func ConfigPath() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, defaultConfigFile), nil
}
