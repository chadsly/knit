package processlog

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
)

func Setup(processName, dataDir string) (func(), string, error) {
	name := strings.TrimSpace(processName)
	if name == "" {
		name = "process"
	}
	if strings.TrimSpace(dataDir) == "" {
		dataDir = ".knit"
	}
	absDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(absDir, 0o700); err != nil {
		return nil, "", err
	}
	logPath := filepath.Join(absDir, name+".log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, "", err
	}
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.LUTC)
	log.SetOutput(io.MultiWriter(os.Stderr, f))
	log.Printf("%s logging initialized: %s", name, logPath)
	return func() {
		log.Printf("%s logging closed", name)
		_ = f.Close()
	}, logPath, nil
}

func StackTrace() string {
	return string(debug.Stack())
}
