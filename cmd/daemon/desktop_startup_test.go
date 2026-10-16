package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopEnvironmentDefaultsAndOverrides(t *testing.T) {
	keys := []string{"KNIT_DATA_DIR", "KNIT_ADDR", "KNIT_CONFIG_PATH", "KNIT_SQLITE_PATH"}
	for _, key := range keys {
		t.Setenv(key, "")
	}
	base, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := configureDesktopEnvironment(); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(base, "knit", "desktop")
	if os.Getenv("KNIT_DATA_DIR") != data || os.Getenv("KNIT_CONFIG_PATH") != filepath.Join(data, "knit.toml") || os.Getenv("KNIT_SQLITE_PATH") != filepath.Join(data, "knit.db") || os.Getenv("KNIT_ADDR") != "127.0.0.1:7778" {
		t.Fatal("unstable desktop defaults")
	}
	for _, key := range keys {
		t.Setenv(key, "explicit-"+key)
	}
	if err := configureDesktopEnvironment(); err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if os.Getenv(key) != "explicit-"+key {
			t.Fatal("overwrote explicit configuration", key)
		}
	}
}
