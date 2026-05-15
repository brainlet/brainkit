//go:build unix

package browser

import (
	"os"
	"os/exec"
	"syscall"
)

func configureBrowserCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateBrowserProcess(process *os.Process) error {
	if process == nil {
		return nil
	}
	err := syscall.Kill(-process.Pid, syscall.SIGTERM)
	if err == syscall.ESRCH {
		return nil
	}
	return err
}

func killBrowserProcess(process *os.Process) error {
	if process == nil {
		return nil
	}
	err := syscall.Kill(-process.Pid, syscall.SIGKILL)
	if err == syscall.ESRCH {
		return nil
	}
	return err
}
