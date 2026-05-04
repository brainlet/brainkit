//go:build windows

package jsbridge

import "os/exec"

func configureCommandProcessGroup(*exec.Cmd) {}

func killCommandTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
}
