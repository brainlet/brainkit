//go:build !unix

package browser

import (
	"os"
	"os/exec"
)

func configureBrowserCommand(*exec.Cmd) {}

func terminateBrowserProcess(process *os.Process) error {
	if process == nil {
		return nil
	}
	return process.Kill()
}

func killBrowserProcess(process *os.Process) error {
	if process == nil {
		return nil
	}
	return process.Kill()
}
