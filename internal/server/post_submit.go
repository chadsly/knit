package server

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"knit/internal/operatorstate"
)

const defaultPostSubmitTimeout = 600 * time.Second

type postSubmitResult struct {
	Enabled        bool                  `json:"enabled"`
	Workdir        string                `json:"workdir,omitempty"`
	TimeoutSeconds int                   `json:"timeout_seconds"`
	Rebuild        *postSubmitStepResult `json:"rebuild,omitempty"`
	Verify         *postSubmitStepResult `json:"verify,omitempty"`
}

type postSubmitStepResult struct {
	Command    string `json:"command"`
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	ExitCode   int    `json:"exit_code,omitempty"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

func runPostSubmitAutomation() *postSubmitResult {
	return runPostSubmitAutomationFor(operatorstate.RuntimeCodex{
		CodexWorkdir:      strings.TrimSpace(os.Getenv("KNIT_CODEX_WORKDIR")),
		PostSubmitRebuild: strings.TrimSpace(os.Getenv("KNIT_POST_SUBMIT_REBUILD_CMD")),
		PostSubmitVerify:  strings.TrimSpace(os.Getenv("KNIT_POST_SUBMIT_VERIFY_CMD")),
		PostSubmitTimeout: readPostSubmitTimeoutFromEnv(),
	})
}

func runPostSubmitAutomationFor(rc operatorstate.RuntimeCodex) *postSubmitResult {
	rebuildCmd := strings.TrimSpace(rc.PostSubmitRebuild)
	verifyCmd := strings.TrimSpace(rc.PostSubmitVerify)
	if rebuildCmd == "" && verifyCmd == "" {
		return nil
	}

	timeout := defaultPostSubmitTimeout
	if rc.PostSubmitTimeout > 0 {
		timeout = time.Duration(rc.PostSubmitTimeout) * time.Second
	}

	workdir := strings.TrimSpace(rc.CodexWorkdir)
	if workdir == "" {
		if wd, err := os.Getwd(); err == nil {
			workdir = wd
		}
	}

	result := &postSubmitResult{
		Enabled:        true,
		Workdir:        workdir,
		TimeoutSeconds: int(timeout / time.Second),
	}
	if rebuildCmd != "" {
		result.Rebuild = runPostSubmitStep("rebuild", rebuildCmd, workdir, timeout)
	}
	if verifyCmd != "" {
		if result.Rebuild != nil && result.Rebuild.Status != "success" {
			result.Verify = &postSubmitStepResult{
				Command: verifyCmd,
				Status:  "skipped",
				Error:   "skipped because rebuild step failed",
			}
		} else {
			result.Verify = runPostSubmitStep("verify", verifyCmd, workdir, timeout)
		}
	}
	return result
}

func readPostSubmitTimeoutFromEnv() int {
	if v := strings.TrimSpace(os.Getenv("KNIT_POST_SUBMIT_TIMEOUT_SECONDS")); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			return sec
		}
	}
	return 0
}

func runPostSubmitStep(step, command, workdir string, timeout time.Duration) *postSubmitStepResult {
	start := time.Now()
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	cmd := shellCommand(ctx, command)
	if workdir != "" {
		cmd.Dir = workdir
	}
	out, err := cmd.CombinedOutput()

	res := &postSubmitStepResult{
		Command:    command,
		DurationMS: time.Since(start).Milliseconds(),
	}
	output := truncateTail(strings.TrimSpace(string(out)), 8000)
	if output != "" {
		res.Output = output
	}

	if err == nil {
		res.Status = "success"
		return res
	}
	if errorsIsTimeout(ctx) {
		res.Status = "timeout"
		res.Error = fmt.Sprintf("%s step timed out after %s", step, timeout)
		res.ExitCode = -1
		return res
	}

	res.Status = "failed"
	res.Error = err.Error()
	if exitErr, ok := err.(*exec.ExitError); ok {
		res.ExitCode = exitErr.ExitCode()
	}
	return res
}

func shellCommand(ctx context.Context, raw string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", raw)
	}
	return exec.CommandContext(ctx, "sh", "-lc", raw)
}

func errorsIsTimeout(ctx context.Context) bool {
	return ctx != nil && ctx.Err() != nil && ctx.Err() == context.DeadlineExceeded
}

func truncateTail(text string, limit int) string {
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[len(text)-limit:]
}
