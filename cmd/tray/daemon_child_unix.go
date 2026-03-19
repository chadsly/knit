//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func startDetachedProcess(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	return cmd.Start()
}
