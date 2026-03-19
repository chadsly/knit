package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/getlantern/systray"

	"knit/internal/app"
)

func main() {
	if os.Getenv(trayServerOnlyEnv) == "1" {
		runServerOnly()
		return
	}
	cleanupLog := setupTrayLogging()
	defer cleanupLog()
	defer recoverTrayPanic()

	cfg := app.DefaultConfig()
	if err := ensureServerRunning(cfg.HTTPListenAddr); err != nil {
		log.Fatalf("failed to ensure daemon is running: %v", err)
	}
	systray.Run(func() { onReady(cfg.HTTPListenAddr, cfg.ControlToken) }, onExit)
}

func onReady(addr, token string) {
	systray.SetTitle("Knit")
	systray.SetTooltip("Local multimodal feedback tray controller")

	openUI := systray.AddMenuItem("Open UI", "Open local review UI")
	pause := systray.AddMenuItem("Pause Capture", "Pause active capture")
	resume := systray.AddMenuItem("Resume Capture", "Resume capture")
	kill := systray.AddMenuItem("Kill Capture", "Immediate capture stop")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit Tray", "Close the tray and leave the daemon running")

	baseURL := "http://" + addr
	go func() {
		for {
			select {
			case <-openUI.ClickedCh:
				_ = openBrowser(baseURL)
			case <-pause.ClickedCh:
				_ = postJSON(baseURL+"/api/session/pause", []byte("{}"), token)
			case <-resume.ClickedCh:
				_ = postJSON(baseURL+"/api/session/resume", []byte("{}"), token)
			case <-kill.ClickedCh:
				_ = postJSON(baseURL+"/api/capture/kill", []byte("{}"), token)
			case <-quit.ClickedCh:
				log.Printf("tray shutdown requested; leaving daemon running")
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {}

func postJSON(url string, body []byte, token string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Knit-Token", token)
	}
	req.Header.Set("X-Knit-Nonce", fmt.Sprintf("%d-%d", time.Now().UTC().UnixNano(), rand.Int()))
	req.Header.Set("X-Knit-Timestamp", fmt.Sprintf("%d", time.Now().UTC().UnixMilli()))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	return nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
