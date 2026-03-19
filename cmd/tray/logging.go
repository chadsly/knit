package main

import (
	"log"
	"os"

	"knit/internal/app"
	"knit/internal/processlog"
)

func setupTrayLogging() func() {
	dataDir := trayDataDir()
	cleanupLog, logPath, err := processlog.Setup("tray", dataDir)
	if err != nil {
		log.Printf("tray logging setup failed: %v", err)
		return func() {}
	}
	log.Printf("tray effective data dir: %s", dataDir)
	log.Printf("tray runtime log path: %s", logPath)
	return cleanupLog
}

func setupChildDaemonLogging(dataDir string) (func(), string, error) {
	return processlog.Setup("daemon", dataDir)
}

func trayDataDir() string {
	return app.DefaultConfig().DataDir
}

func recoverTrayPanic() {
	if rec := recover(); rec != nil {
		log.Printf("tray panic: %v\n%s", rec, processlog.StackTrace())
		os.Exit(1)
	}
}

func recoverTrayChildPanic() {
	if rec := recover(); rec != nil {
		log.Printf("daemon panic: %v\n%s", rec, processlog.StackTrace())
		os.Exit(1)
	}
}
