//go:build !linux

package sauce

import "os/exec"

func configureQuitOnParentExit(cmd *exec.Cmd) {}
