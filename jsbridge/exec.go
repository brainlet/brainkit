package jsbridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	quickjs "github.com/buke/quickjs-go"
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
	mu     sync.Mutex
	nextID int
	procs  map[int]*spawnedProcess
	bridge *Bridge
}

func (p *ExecPolyfill) SetBridge(b *Bridge) { p.bridge = b }

// Exec creates a child process execution polyfill.
func Exec() *ExecPolyfill {
	return &ExecPolyfill{procs: map[int]*spawnedProcess{}}
}

func (p *ExecPolyfill) Name() string { return "exec" }

func (p *ExecPolyfill) Setup(ctx *quickjs.Context) error {
	// Async exec: shell command runs in a separate goroutine.
	// The bridge is NOT held during command execution.
	ctx.Globals().Set("__go_exec", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("exec: command argument required"))
		}
		command := args[0].ToString()

		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			polyfill := p
			polyfill.bridge.Go(func(goCtx context.Context) {
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
						ctx.Schedule(func(ctx *quickjs.Context) {
							errVal := ctx.NewError(fmt.Errorf("exec: %w", err))
							defer errVal.Free()
							reject(errVal)
						})
						return
					}
				}

				b, _ := json.Marshal(map[string]interface{}{
					"stdout":   stdoutBuf.String(),
					"stderr":   stderrBuf.String(),
					"exitCode": exitCode,
				})
				resultJSON := string(b)

				ctx.Schedule(func(ctx *quickjs.Context) {
					resolve(ctx.NewString(resultJSON))
				})
			})
		})
	}))

	// Spawn stays sync for setup (returns PID immediately).
	// The actual I/O (readLine, wait) becomes async.
	ctx.Globals().Set("__go_spawn", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("spawn: command argument required"))
		}
		command := args[0].ToString()
		var cmdArgs []string
		if len(args) >= 2 {
			if err := json.Unmarshal([]byte(args[1].ToString()), &cmdArgs); err != nil {
				return ctx.ThrowError(fmt.Errorf("spawn: json unmarshal args: %w", err))
			}
		}

		cmd := exec.Command(command, cmdArgs...)
		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("spawn: stdout pipe: %w", err))
		}

		if err := cmd.Start(); err != nil {
			return ctx.ThrowError(fmt.Errorf("spawn: start: %w", err))
		}

		proc := &spawnedProcess{
			cmd:       cmd,
			lines:     make(chan string, 256),
			linesDone: make(chan struct{}),
			waitErr:   make(chan error, 1),
		}

		go func() {
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				proc.lines <- scanner.Text()
			}
			close(proc.lines)
			close(proc.linesDone)
		}()

		go func() {
			<-proc.linesDone
			proc.waitErr <- cmd.Wait()
		}()

		p.mu.Lock()
		id := p.nextID
		p.nextID++
		p.procs[id] = proc
		p.mu.Unlock()

		return ctx.NewInt32(int32(id))
	}))

	// Async readLine: reading from process stdout can block.
	ctx.Globals().Set("__go_spawn_read", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("spawn_read: id argument required"))
		}
		id := int(args[0].ToInt32())

		p.mu.Lock()
		proc, ok := p.procs[id]
		p.mu.Unlock()
		if !ok {
			return ctx.ThrowError(fmt.Errorf("spawn_read: no process with id %d", id))
		}

		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			p.bridge.Go(func(goCtx context.Context) {
				line, ok := <-proc.lines
				if goCtx.Err() != nil { return }
				ctx.Schedule(func(ctx *quickjs.Context) {
					if !ok {
						resolve(ctx.NewNull())
					} else {
						resolve(ctx.NewString(line))
					}
				})
			})
		})
	}))

	// Async wait: waiting for process exit can block.
	ctx.Globals().Set("__go_spawn_wait", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("spawn_wait: id argument required"))
		}
		id := int(args[0].ToInt32())

		p.mu.Lock()
		proc, ok := p.procs[id]
		p.mu.Unlock()
		if !ok {
			return ctx.ThrowError(fmt.Errorf("spawn_wait: no process with id %d", id))
		}

		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			p.bridge.Go(func(goCtx context.Context) {
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

				if goCtx.Err() != nil { return }
				ctx.Schedule(func(ctx *quickjs.Context) {
					resolve(ctx.NewInt32(int32(exitCode)))
				})
			})
		})
	}))

	return evalJS(ctx, `
globalThis.child_process = {
  async exec(command) { return JSON.parse(await __go_exec(command)); },
  spawn(command, args) {
    const id = __go_spawn(command, args ? JSON.stringify(args) : '[]');
    return {
      async readLine() { return await __go_spawn_read(id); },
      async wait() { return await __go_spawn_wait(id); },
    };
  },
};
`)
}
