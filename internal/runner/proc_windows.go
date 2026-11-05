//go:build windows

package runner

import (
	"os/exec"
	"strconv"
	"syscall"
)

func configure(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// killTree terminates the child and its descendants with taskkill /T.
func killTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	k := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
	if err := k.Run(); err != nil {
		_ = cmd.Process.Kill()
	}
}
