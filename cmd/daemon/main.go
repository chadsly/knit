package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"knit/internal/app"
	"knit/internal/config"
	"knit/internal/processlog"
)

func main() {
	cfg := app.DefaultConfig()
	cleanupLog, logPath, err := processlog.Setup("daemon", cfg.DataDir)
	if err != nil {
		log.Printf("daemon logging setup failed: %v", err)
	} else {
		defer cleanupLog()
		log.Printf("daemon effective data dir: %s", cfg.DataDir)
		log.Printf("daemon runtime log path: %s", logPath)
	}
	for _, line := range startupBannerLines(cfg) {
		log.Print(line)
	}
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("daemon panic: %v\n%s", rec, processlog.StackTrace())
			os.Exit(1)
		}
	}()
	a, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		log.Printf("daemon shutdown requested: %v", ctx.Err())
	}()

	if err := a.Run(ctx); err != nil {
		log.Printf("daemon stopped with error: %v", err)
		os.Exit(1)
	}
	log.Printf("daemon stopped cleanly")
}

func startupBannerLines(cfg config.Config) []string {
	return []string{
		"Welcome to Knit.",
		"Version: " + startupVersion(cfg),
		"Open the local UI: " + startupUIURL(cfg),
	}
}

func startupVersion(cfg config.Config) string {
	if v := strings.TrimSpace(cfg.BuildID); v != "" {
		return v
	}
	if v := strings.TrimSpace(cfg.VersionPin); v != "" {
		return v
	}
	return "dev"
}

func startupUIURL(cfg config.Config) string {
	addr := strings.TrimSpace(cfg.HTTPListenAddr)
	if addr == "" {
		return "http://127.0.0.1:7777"
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	host = strings.TrimSpace(host)
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return "http://" + net.JoinHostPort(host, port)
}
