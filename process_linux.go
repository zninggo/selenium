//go:build linux

package selenium

import (
	"os/exec"
	"syscall"
)

// configureCmd puts the child in its own process group and asks the kernel to
// deliver SIGKILL if this parent dies (avoids orphaned chromedriver/Xvfb).
func configureCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGKILL,
	}
}

// killCmd terminates the service process group when possible.
func killCmd(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	// Negative PID targets the process group set by configureCmd.
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		return cmd.Process.Kill()
	}
	return nil
}
