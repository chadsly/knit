package app

import (
	"net/http"
	"strings"
	"time"
)

func ProbeDaemon(addr string, timeout time.Duration) bool {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return false
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func WaitForDaemon(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ProbeDaemon(addr, 2*time.Second) {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return ProbeDaemon(addr, 2*time.Second)
}
