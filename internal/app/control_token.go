package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const controlTokenFilename = "control_token"

func ResolveOrCreateControlToken(dataDir string) (string, error) {
	if v := strings.TrimSpace(os.Getenv("KNIT_CONTROL_TOKEN")); v != "" {
		return v, nil
	}
	dir := strings.TrimSpace(dataDir)
	if dir == "" {
		dir = ".knit"
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create control token dir: %w", err)
	}
	path := filepath.Join(dir, controlTokenFilename)
	if raw, err := os.ReadFile(path); err == nil {
		if token := strings.TrimSpace(string(raw)); token != "" {
			return token, nil
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read control token: %w", err)
	}
	token := generateControlToken()
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write control token: %w", err)
	}
	return token, nil
}
