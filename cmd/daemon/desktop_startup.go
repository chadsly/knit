package main

import (
	"os"
	"path/filepath"
)

// Desktop packages keep durable state outside versioned plugin installation folders.
// Explicit environment overrides remain authoritative for isolated development/tests.
func configureDesktopEnvironment() error {
	base, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	data := os.Getenv("KNIT_DATA_DIR")
	if data == "" {
		data = filepath.Join(base, "knit", "desktop")
	}
	defaults := map[string]string{
		"KNIT_DATA_DIR":    data,
		"KNIT_ADDR":        "127.0.0.1:7778",
		"KNIT_CONFIG_PATH": filepath.Join(data, "knit.toml"),
		"KNIT_SQLITE_PATH": filepath.Join(data, "knit.db"),
	}
	for key, value := range defaults {
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}
