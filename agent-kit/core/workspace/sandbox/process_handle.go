// Ported from: packages/core/src/workspace/sandbox/process-manager/process-handle.ts
package sandbox

import (
	"fmt"
	"io"
	"sync"
)

// =============================================================================
// Process Handle
// =============================================================================

// ProcessHandle represents a handle to a spawned process.
//
// The base struct handles stdout/stderr accumulation, callback dispatch via
// EmitStdout/EmitStderr, and lazy reader/writer getters.
//
// For consumers:
//   - handle.Stdout — poll accumulated output
//   - handle.Wait() — wait for exit, optionally with streaming callbacks
//   - handle.Reader / handle.Writer — io.Reader/io.Writer interop
//
// For implementors: Call EmitStdout(data) / EmitStderr(data) from
// your transport callback to dispatch data.
type ProcessHandle struct {
	// PID is the process ID.
	PID int
	// ExitCode is the exit code, nil while the process is still running.
	ExitCode *int
	// Command is the command that was spawned (set by the process manager).
	Command string

	// WaitFunc is the implementation-specific wait function.
	// Set by the creator of the ProcessHandle.
	WaitFunc func() (*CommandResult, error)
	// KillFunc is the implementation-specific kill function.
	KillFunc func() (bool, error)
	// SendStdinFunc is the implementation-specific stdin write function.
	SendStdinFunc func(data string) error

	mu              sync.RWMutex
	stdout          string
	stderr          string
	stdoutListeners []func(data string)
	stderrListeners []func(data string)
	reader          *processReader
	writer          *processWriter
}

// NewProcessHandle creates a new ProcessHandle.
func NewProcessHandle(pid int, options *SpawnProcessOptions) *ProcessHandle {
	h := &ProcessHandle{
		PID: pid,
	}
	if options != nil {
		if options.OnStdout != nil {
			h.stdoutListeners = append(h.stdoutListeners, options.OnStdout)
		}
		if options.OnStderr != nil {
			h.stderrListeners = append(h.stderrListeners, options.OnStderr)
		}
	}
	return h
}

// Stdout returns the accumulated stdout so far.
func (h *ProcessHandle) Stdout() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.stdout
}

// Stderr returns the accumulated stderr so far.
func (h *ProcessHandle) Stderr() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.stderr
}

// EmitStdout emits stdout data — accumulates, dispatches to user callbacks,
// and pushes to reader.
func (h *ProcessHandle) EmitStdout(data string) {
	h.mu.Lock()
	h.stdout += data
	listeners := make([]func(string), len(h.stdoutListeners))
	copy(listeners, h.stdoutListeners)
	h.mu.Unlock()

	for _, listener := range listeners {
		listener(data)
	}

	if h.reader != nil {
		h.reader.push([]byte(data))
	}
}

// EmitStderr emits stderr data — accumulates and dispatches to user callbacks.
func (h *ProcessHandle) EmitStderr(data string) {
	h.mu.Lock()
	h.stderr += data
	listeners := make([]func(string), len(h.stderrListeners))
	copy(listeners, h.stderrListeners)
	h.mu.Unlock()

	for _, listener := range listeners {
		listener(data)
	}
}

// WaitOptions holds options for Wait.
type WaitOptions struct {
	OnStdout func(data string)
	OnStderr func(data string)
}

// Wait waits for the process to finish and returns the result.
// Optionally pass OnStdout/OnStderr callbacks to stream output chunks while waiting.
func (h *ProcessHandle) Wait(options *WaitOptions) (*CommandResult, error) {
	if h.WaitFunc == nil {
		return nil, fmt.Errorf("ProcessHandle.WaitFunc not set")
	}

	// Add temporary listeners
	if options != nil {
		h.mu.Lock()
		if options.OnStdout != nil {
			h.stdoutListeners = append(h.stdoutListeners, options.OnStdout)
		}
		if options.OnStderr != nil {
			h.stderrListeners = append(h.stderrListeners, options.OnStderr)
		}
		h.mu.Unlock()
	}

	result, err := h.WaitFunc()

	// Remove temporary listeners
	if options != nil {
		h.mu.Lock()
		if options.OnStdout != nil {
			h.removeStdoutListener(options.OnStdout)
		}
		if options.OnStderr != nil {
			h.removeStderrListener(options.OnStderr)
		}
		h.mu.Unlock()
	}

	return result, err
}

// Kill kills the running process. Returns true if killed, false if not found.
func (h *ProcessHandle) Kill() (bool, error) {
	if h.KillFunc == nil {
		return false, fmt.Errorf("ProcessHandle.KillFunc not set")
	}
	return h.KillFunc()
}

// SendStdin sends data to the process's stdin.
func (h *ProcessHandle) SendStdin(data string) error {
	if h.SendStdinFunc == nil {
		return fmt.Errorf("ProcessHandle.SendStdinFunc not set")
	}
	return h.SendStdinFunc(data)
}

// Reader returns an io.Reader for stdout (for use with pipes, etc.).
func (h *ProcessHandle) Reader() io.Reader {
	if h.reader == nil {
		h.reader = newProcessReader()
		// When process exits, close the reader
		go func() {
			if h.WaitFunc != nil {
				_, _ = h.WaitFunc()
			}
			h.reader.close()
		}()
	}
	return h.reader
}

// Writer returns an io.Writer for stdin (for use with pipes, etc.).
func (h *ProcessHandle) Writer() io.Writer {
	if h.writer == nil {
		h.writer = &processWriter{handle: h}
	}
	return h.writer
}

// removeStdoutListener removes a specific stdout listener (must be called under lock).
func (h *ProcessHandle) removeStdoutListener(fn func(string)) {
	for i, listener := range h.stdoutListeners {
		if &listener == &fn {
			h.stdoutListeners = append(h.stdoutListeners[:i], h.stdoutListeners[i+1:]...)
			return
		}
	}
}

// removeStderrListener removes a specific stderr listener (must be called under lock).
func (h *ProcessHandle) removeStderrListener(fn func(string)) {
	for i, listener := range h.stderrListeners {
		if &listener == &fn {
			h.stderrListeners = append(h.stderrListeners[:i], h.stderrListeners[i+1:]...)
			return
		}
	}
}

// =============================================================================
// Process Reader (io.Reader for stdout)
// =============================================================================

// processReader is an io.Reader that receives data from EmitStdout.
type processReader struct {
	mu     sync.Mutex
	buf    []byte
	closed bool
	ch     chan struct{}
}

func newProcessReader() *processReader {
	return &processReader{
		ch: make(chan struct{}, 1),
	}
}

func (r *processReader) push(data []byte) {
	r.mu.Lock()
	r.buf = append(r.buf, data...)
	r.mu.Unlock()
	select {
	case r.ch <- struct{}{}:
	default:
	}
}

func (r *processReader) close() {
	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
	select {
	case r.ch <- struct{}{}:
	default:
	}
}

func (r *processReader) Read(p []byte) (int, error) {
	for {
		r.mu.Lock()
		if len(r.buf) > 0 {
			n := copy(p, r.buf)
			r.buf = r.buf[n:]
			r.mu.Unlock()
			return n, nil
		}
		if r.closed {
			r.mu.Unlock()
			return 0, io.EOF
		}
		r.mu.Unlock()
		<-r.ch
	}
}

// =============================================================================
// Process Writer (io.Writer for stdin)
// =============================================================================

// processWriter is an io.Writer that sends data to the process's stdin.
type processWriter struct {
	handle *ProcessHandle
}

func (w *processWriter) Write(p []byte) (int, error) {
	err := w.handle.SendStdin(string(p))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
