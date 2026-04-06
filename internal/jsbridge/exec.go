package jsbridge

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"github.com/brainlet/brainkit/internal/syncx"

	quickjs "github.com/buke/quickjs-go"
)

// spawnedProcess tracks a spawned child process for streaming reads and stdin writes.
type spawnedProcess struct {
	cmd       *exec.Cmd
	lines     chan string
	linesDone chan struct{}
	waitErr   chan error
	stdinPipe io.WriteCloser // stdin pipe for writing to the process
	chunks    chan string     // raw stdout chunks (for LSP/JSON-RPC)
}

// ExecPolyfill provides child_process.exec and child_process.spawn.
type ExecPolyfill struct {
	mu     syncx.Mutex
	nextID int
	procs  map[int]*spawnedProcess
	bridge *Bridge
}

func (p *ExecPolyfill) SetBridge(b *Bridge) { p.bridge = b }

// Exec creates a child process execution polyfill.
func Exec() *ExecPolyfill {
	return &ExecPolyfill{procs: map[int]*spawnedProcess{}, nextID: 1}
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

		// Set up cwd if provided as 3rd arg
		if len(args) >= 3 && args[2].ToString() != "" {
			cmd.Dir = args[2].ToString()
		}

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("spawn: stdout pipe: %w", err))
		}

		stdinPipe, err := cmd.StdinPipe()
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("spawn: stdin pipe: %w", err))
		}

		if err := cmd.Start(); err != nil {
			return ctx.ThrowError(fmt.Errorf("spawn: start: %w", err))
		}

		proc := &spawnedProcess{
			cmd:       cmd,
			lines:     make(chan string, 256),
			linesDone: make(chan struct{}),
			waitErr:   make(chan error, 1),
			stdinPipe: stdinPipe,
			chunks:    make(chan string, 256),
		}

		// Read stdout in two modes: line-based (for spawn_read) and raw chunks (for LSP)
		go func() {
			reader := bufio.NewReader(stdoutPipe)
			for {
				// Read raw bytes (up to 64KB) for chunk mode
				buf := make([]byte, 65536)
				n, readErr := reader.Read(buf)
				if n > 0 {
					chunk := string(buf[:n])
					// Send to chunk channel (non-blocking — drop if full)
					select {
					case proc.chunks <- chunk:
					default:
					}
					// Also split into lines for line-based mode
					lines := strings.Split(chunk, "\n")
					for i, line := range lines {
						if i == len(lines)-1 && line == "" {
							continue // skip trailing empty from split
						}
						select {
						case proc.lines <- line:
						default:
						}
					}
				}
				if readErr != nil {
					break
				}
			}
			close(proc.lines)
			close(proc.chunks)
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

	// Write to process stdin (for LSP JSON-RPC communication)
	ctx.Globals().Set("__go_spawn_write", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return ctx.ThrowError(fmt.Errorf("spawn_write: id and data arguments required"))
		}
		id := int(args[0].ToInt32())
		data := args[1].ToString()

		p.mu.Lock()
		proc, ok := p.procs[id]
		p.mu.Unlock()
		if !ok {
			return ctx.ThrowError(fmt.Errorf("spawn_write: no process with id %d", id))
		}

		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			p.bridge.Go(func(goCtx context.Context) {
				_, err := io.WriteString(proc.stdinPipe, data)
				if goCtx.Err() != nil {
					return
				}
				if err != nil {
					ctx.Schedule(func(ctx *quickjs.Context) {
						errVal := ctx.NewError(fmt.Errorf("spawn_write: %w", err))
						defer errVal.Free()
						reject(errVal)
					})
					return
				}
				ctx.Schedule(func(ctx *quickjs.Context) {
					resolve(ctx.NewBool(true))
				})
			})
		})
	}))

	// Read raw chunk from process stdout (for LSP — not line-based)
	ctx.Globals().Set("__go_spawn_read_chunk", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("spawn_read_chunk: id argument required"))
		}
		id := int(args[0].ToInt32())

		p.mu.Lock()
		proc, ok := p.procs[id]
		p.mu.Unlock()
		if !ok {
			return ctx.ThrowError(fmt.Errorf("spawn_read_chunk: no process with id %d", id))
		}

		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			p.bridge.Go(func(goCtx context.Context) {
				chunk, ok := <-proc.chunks
				if goCtx.Err() != nil {
					return
				}
				ctx.Schedule(func(ctx *quickjs.Context) {
					if !ok {
						resolve(ctx.NewNull())
					} else {
						resolve(ctx.NewString(chunk))
					}
				})
			})
		})
	}))

	// Kill a spawned process
	ctx.Globals().Set("__go_spawn_kill", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return ctx.ThrowError(fmt.Errorf("spawn_kill: id argument required"))
		}
		id := int(args[0].ToInt32())

		p.mu.Lock()
		proc, ok := p.procs[id]
		p.mu.Unlock()
		if !ok {
			return ctx.Undefined()
		}

		if proc.cmd.Process != nil {
			_ = proc.cmd.Process.Kill()
		}
		if proc.stdinPipe != nil {
			_ = proc.stdinPipe.Close()
		}
		p.mu.Lock()
		delete(p.procs, id)
		p.mu.Unlock()

		return ctx.Undefined()
	}))

	// __go_exec_sync(command) → JSON { stdout, stderr, exitCode }
	ctx.Globals().Set("__go_exec_sync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("execSync: command argument required"))
		}
		command := args[0].ToString()

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
				return qctx.ThrowError(fmt.Errorf("execSync: %w", err))
			}
		}

		b, _ := json.Marshal(map[string]interface{}{
			"stdout":   stdoutBuf.String(),
			"stderr":   stderrBuf.String(),
			"exitCode": exitCode,
		})
		return qctx.NewString(string(b))
	}))

	// __go_exec_file_sync(file, argsJSON, cwd) → JSON { stdout, stderr, exitCode }
	ctx.Globals().Set("__go_exec_file_sync", ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 1 {
			return qctx.ThrowError(fmt.Errorf("execFileSync: file argument required"))
		}
		file := args[0].ToString()
		var cmdArgs []string
		if len(args) >= 2 && args[1].ToString() != "[]" {
			json.Unmarshal([]byte(args[1].ToString()), &cmdArgs)
		}

		cmd := exec.Command(file, cmdArgs...)
		if len(args) >= 3 && args[2].ToString() != "" {
			cmd.Dir = args[2].ToString()
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
				return qctx.ThrowError(fmt.Errorf("execFileSync %s: %w", file, err))
			}
		}

		b, _ := json.Marshal(map[string]interface{}{
			"stdout":   stdoutBuf.String(),
			"stderr":   stderrBuf.String(),
			"exitCode": exitCode,
		})
		return qctx.NewString(string(b))
	}))

	return evalJS(ctx, `
globalThis.child_process = {
  async exec(command) { return JSON.parse(await __go_exec(command)); },
  execSync: function(command) {
    var result = JSON.parse(__go_exec_sync(command));
    if (result.exitCode !== 0) {
      var err = new Error("Command failed: " + command + "\n" + result.stderr);
      err.status = result.exitCode;
      err.stderr = result.stderr;
      err.stdout = result.stdout;
      throw err;
    }
    return typeof Buffer !== "undefined" ? Buffer.from(result.stdout) : result.stdout;
  },
  execFileSync: function(file, args, options) {
    var cwd = (options && options.cwd) || "";
    var result = JSON.parse(__go_exec_file_sync(file, JSON.stringify(args || []), cwd));
    if (result.exitCode !== 0) {
      var err = new Error("Command failed: " + file + "\n" + result.stderr);
      err.status = result.exitCode;
      err.stderr = result.stderr;
      err.stdout = result.stdout;
      throw err;
    }
    return typeof Buffer !== "undefined" ? Buffer.from(result.stdout) : result.stdout;
  },
  spawnSync: function(command, args, options) {
    var cwd = (options && options.cwd) || "";
    var result = JSON.parse(__go_exec_file_sync(command, JSON.stringify(args || []), cwd));
    return { stdout: result.stdout, stderr: result.stderr, status: result.exitCode, error: null };
  },
  spawn(command, args, cwd) {
    const id = __go_spawn(command, args ? JSON.stringify(args) : '[]', cwd || '');
    return {
      pid: id,
      async readLine() { return await __go_spawn_read(id); },
      async readChunk() { return await __go_spawn_read_chunk(id); },
      async write(data) { return await __go_spawn_write(id, data); },
      async wait() { return await __go_spawn_wait(id); },
      kill() { __go_spawn_kill(id); },
    };
  },
};
`)
}
