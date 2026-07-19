//go:build !linux

package selenium

import "os/exec"

func configureCmd(cmd *exec.Cmd) {}

func killCmd(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
