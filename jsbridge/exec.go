package jsbridge

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/fastschema/qjs"
)

// spawnedProcess tracks a spawned child process for streaming reads.
type spawnedProcess struct {
	cmd       *exec.Cmd
	lines     chan string
	linesDone chan struct{}
	waitErr   chan error
}

// ExecPolyfill provides child_process.exec and child_process.spawn.
type ExecPolyfill struct {
	mu      sync.Mutex
	nextID  int
	procs   map[int]*spawnedProcess
}

// Exec creates a child process execution polyfill.
func Exec() *ExecPolyfill {
	return &ExecPolyfill{procs: map[int]*spawnedProcess{}}
}

func (p *ExecPolyfill) Name() string { return "exec" }

func (p *ExecPolyfill) Setup(ctx *qjs.Context) error {
	ctx.SetFunc("__go_exec", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("exec: command argument required")
		}
		command := args[0].String()

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", command)
		} else {
			cmd = exec.Command("sh", "-c", command)
		}

		var stdoutBuf, stderrBuf strings.Builder
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf

		exitCode := 0
		err := cmd.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return nil, fmt.Errorf("exec: %w", err)
			}
		}

		b, err := json.Marshal(map[string]interface{}{
			"stdout":   stdoutBuf.String(),
			"stderr":   stderrBuf.String(),
			"exitCode": exitCode,
		})
		if err != nil {
			return nil, fmt.Errorf("exec: json marshal: %w", err)
		}
		return this.Context().NewString(string(b)), nil
	})

	ctx.SetFunc("__go_spawn", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("spawn: command argument required")
		}
		command := args[0].String()
		var cmdArgs []string
		if len(args) >= 2 {
			if err := json.Unmarshal([]byte(args[1].String()), &cmdArgs); err != nil {
				return nil, fmt.Errorf("spawn: json unmarshal args: %w", err)
			}
		}

		cmd := exec.Command(command, cmdArgs...)
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return nil, fmt.Errorf("spawn: stdout pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("spawn: start: %w", err)
		}

		proc := &spawnedProcess{
			cmd:       cmd,
			lines:     make(chan string, 256),
			linesDone: make(chan struct{}),
			waitErr:   make(chan error, 1),
		}

		// Read all stdout lines into the channel.
		// Must complete before cmd.Wait() because Wait closes pipes.
		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				proc.lines <- scanner.Text()
			}
			close(proc.lines)
			close(proc.linesDone)
		}()

		// Wait after stdout is fully drained.
		go func() {
			<-proc.linesDone
			proc.waitErr <- cmd.Wait()
		}()

		p.mu.Lock()
		id := p.nextID
		p.nextID++
		p.procs[id] = proc
		p.mu.Unlock()

		return this.Context().NewInt32(int32(id)), nil
	})

	ctx.SetFunc("__go_spawn_read", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("spawn_read: id argument required")
		}
		id := int(args[0].Int32())

		p.mu.Lock()
		proc, ok := p.procs[id]
		p.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("spawn_read: no process with id %d", id)
		}

		line, ok := <-proc.lines
		if !ok {
			return this.Context().NewNull(), nil
		}
		return this.Context().NewString(line), nil
	})

	ctx.SetFunc("__go_spawn_wait", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) < 1 {
			return nil, fmt.Errorf("spawn_wait: id argument required")
		}
		id := int(args[0].Int32())

		p.mu.Lock()
		proc, ok := p.procs[id]
		p.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("spawn_wait: no process with id %d", id)
		}

		waitErr := <-proc.waitErr
		exitCode := 0
		if waitErr != nil {
			if exitErr, ok := waitErr.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			}
		}

		p.mu.Lock()
		delete(p.procs, id)
		p.mu.Unlock()

		return this.Context().NewInt32(int32(exitCode)), nil
	})

	return evalJS(ctx, `
globalThis.child_process = {
  exec(command) { return JSON.parse(__go_exec(command)); },
  spawn(command, args) {
    const id = __go_spawn(command, args ? JSON.stringify(args) : '[]');
    return {
      readLine() { return __go_spawn_read(id); },
      wait() { return __go_spawn_wait(id); },
    };
  },
};
`)
}
