// Ported from: packages/core/src/workspace/sandbox/local-process-manager.ts
package sandbox

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// =============================================================================
// Local Process Manager
// =============================================================================

// LocalProcessManager is a local implementation of process management using os/exec.
// Spawns processes and tracks them in-memory since there's no server to query.
type LocalProcessManager struct {
	*BaseProcessManager
	// sandbox is the local sandbox this process manager belongs to.
	localSandbox *LocalSandbox
}

// NewLocalProcessManager creates a new LocalProcessManager.
func NewLocalProcessManager(env map[string]string) *LocalProcessManager {
	lpm := &LocalProcessManager{
		BaseProcessManager: NewBaseProcessManager(env),
	}
	// Set the spawn implementation
	lpm.SpawnImpl = lpm.spawnImpl
	lpm.ListImpl = lpm.listImpl
	return lpm
}

// SetLocalSandbox sets the local sandbox reference.
func (lpm *LocalProcessManager) SetLocalSandbox(sandbox *LocalSandbox) {
	lpm.localSandbox = sandbox
}

// spawnImpl is the implementation-specific spawn function.
func (lpm *LocalProcessManager) spawnImpl(command string, options *SpawnProcessOptions) (*ProcessHandle, error) {
	cwd := ""
	if options != nil && options.Cwd != "" {
		cwd = options.Cwd
	} else if lpm.localSandbox != nil {
		cwd = lpm.localSandbox.WorkingDir
	}

	env := lpm.buildEnv(options)
	wrapped := lpm.wrapCommand(command)

	var cmd *exec.Cmd
	if len(wrapped.Args) > 0 {
		cmd = exec.Command(wrapped.Command, wrapped.Args...)
	} else {
		// No isolation — use shell
		cmd = exec.Command("sh", "-c", wrapped.Command)
	}

	if cwd != "" {
		cmd.Dir = cwd
	}

	// Set environment
	envSlice := make([]string, 0, len(env))
	for k, v := range env {
		envSlice = append(envSlice, k+"="+v)
	}
	cmd.Env = envSlice

	// Create a new process group so we can kill the entire tree
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Set up pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("process failed to spawn: %w", err)
	}

	pid := cmd.Process.Pid
	startTime := time.Now()

	handle := NewProcessHandle(pid, options)

	// Set up wait channel
	var waitResult *CommandResult
	waitDone := make(chan struct{})

	// Set up timeout
	var timedOut bool
	var timeoutTimer *time.Timer
	if options != nil && options.Timeout > 0 {
		timeoutTimer = time.AfterFunc(time.Duration(options.Timeout)*time.Millisecond, func() {
			timedOut = true
			// Kill the entire process group
			_ = syscall.Kill(-pid, syscall.SIGTERM)
		})
	}

	// Stream stdout
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				handle.EmitStdout(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	// Stream stderr
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				handle.EmitStderr(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for process completion in background
	go func() {
		err := cmd.Wait()
		if timeoutTimer != nil {
			timeoutTimer.Stop()
		}

		exitCode := 0
		killed := false

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					if status.Signaled() {
						killed = true
						exitCode = 128 + int(status.Signal())
					}
				}
			} else {
				exitCode = 1
			}
		}

		if timedOut {
			timeoutMsg := fmt.Sprintf("\nProcess timed out after %dms", options.Timeout)
			handle.EmitStderr(timeoutMsg)
			exitCode = 124
		}

		handle.ExitCode = &exitCode

		waitResult = &CommandResult{
			ExecutionResult: ExecutionResult{
				Success:         exitCode == 0,
				ExitCode:        exitCode,
				Stdout:          handle.Stdout(),
				Stderr:          handle.Stderr(),
				ExecutionTimeMs: time.Since(startTime).Milliseconds(),
				Killed:          killed,
				TimedOut:        timedOut,
			},
		}
		close(waitDone)
	}()

	handle.WaitFunc = func() (*CommandResult, error) {
		<-waitDone
		return waitResult, nil
	}

	handle.KillFunc = func() (bool, error) {
		if handle.ExitCode != nil {
			return false, nil
		}
		// Kill the entire process group
		err := syscall.Kill(-pid, syscall.SIGKILL)
		if err != nil {
			// Fallback to direct kill
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
		return true, nil
	}

	handle.SendStdinFunc = func(data string) error {
		if handle.ExitCode != nil {
			return fmt.Errorf("process %d has already exited with code %d", pid, *handle.ExitCode)
		}
		_, err := stdin.Write([]byte(data))
		return err
	}

	lpm.TrackHandle(handle)
	return handle, nil
}

// buildEnv builds the environment for execution.
func (lpm *LocalProcessManager) buildEnv(options *SpawnProcessOptions) map[string]string {
	if lpm.localSandbox != nil {
		var additionalEnv map[string]string
		if options != nil {
			additionalEnv = options.Env
		}
		return lpm.localSandbox.BuildEnv(additionalEnv)
	}

	// Fallback: merge base env with option env
	env := make(map[string]string)
	for k, v := range lpm.Env {
		env[k] = v
	}
	if options != nil {
		for k, v := range options.Env {
			env[k] = v
		}
	}
	return env
}

// wrapCommand wraps a command with isolation if configured.
func (lpm *LocalProcessManager) wrapCommand(command string) struct {
	Command string
	Args    []string
} {
	if lpm.localSandbox != nil {
		return lpm.localSandbox.WrapCommandForIsolation(command)
	}
	return struct {
		Command string
		Args    []string
	}{Command: command}
}

// listImpl returns info about all tracked processes.
func (lpm *LocalProcessManager) listImpl() ([]ProcessInfo, error) {
	lpm.mu.RLock()
	defer lpm.mu.RUnlock()

	result := make([]ProcessInfo, 0, len(lpm.Tracked))
	for _, handle := range lpm.Tracked {
		info := ProcessInfo{
			PID:     handle.PID,
			Running: handle.ExitCode == nil,
		}
		if handle.ExitCode != nil {
			code := *handle.ExitCode
			info.ExitCode = &code
		}
		result = append(result, info)
	}
	return result, nil
}
