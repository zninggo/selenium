//go:build linux

package sauce

import (
	"os/exec"
	"syscall"
)

// configureQuitOnParentExit asks the kernel to SIGKILL the child if this
// parent process dies.
func configureQuitOnParentExit(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}
}
