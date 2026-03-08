// Ported from: packages/mcp/src/tool/mcp-stdio/mcp-stdio-transport.ts
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os/exec"
	"sync"
)

// StdioConfig is the configuration for a stdio-based MCP transport.
type StdioConfig struct {
	Command string
	Args    []string
	Env     map[string]string
	Cwd     string
	// Stderr handling: if empty, stderr is inherited (os.Stderr).
	// Set to "pipe" to capture stderr, or "null" to discard.
	Stderr string
}

// StdioMCPTransport implements MCPTransport over stdin/stdout of a child process.
type StdioMCPTransport struct {
	mu           sync.Mutex
	process      *exec.Cmd
	ctx          context.Context
	cancel       context.CancelFunc
	readBuffer   *readBuffer
	serverParams StdioConfig
	stdin        io.WriteCloser

	onclose   func()
	onerror   func(error)
	onmessage func(JSONRPCMessage)
}

// NewStdioMCPTransport creates a new stdio MCP transport.
func NewStdioMCPTransport(config StdioConfig) *StdioMCPTransport {
	return &StdioMCPTransport{
		readBuffer:   newReadBuffer(),
		serverParams: config,
	}
}

func (t *StdioMCPTransport) SetOnClose(handler func())              { t.onclose = handler }
func (t *StdioMCPTransport) SetOnError(handler func(error))         { t.onerror = handler }
func (t *StdioMCPTransport) SetOnMessage(handler func(JSONRPCMessage)) { t.onmessage = handler }

// Start starts the child process and begins reading from stdout.
func (t *StdioMCPTransport) Start() error {
	t.mu.Lock()
	if t.process != nil {
		t.mu.Unlock()
		return NewMCPClientError(MCPClientErrorOptions{
			Message: "StdioMCPTransport already started.",
		})
	}

	t.ctx, t.cancel = context.WithCancel(context.Background())
	cmd := CreateChildProcess(t.serverParams, t.ctx)

	// Set up stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.mu.Unlock()
		return err
	}
	t.stdin = stdin

	// Set up stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.mu.Unlock()
		return err
	}

	// Configure stderr
	switch t.serverParams.Stderr {
	case "null":
		// Discard stderr
	case "pipe":
		// Could capture stderr if needed
	default:
		// Inherit stderr (default Go behavior when Stderr is nil is to discard,
		// but the TS code defaults to inherit)
		cmd.Stderr = nil // In practice, set to os.Stderr if you want inherit
	}

	t.process = cmd
	t.mu.Unlock()

	// Start the process
	if err := cmd.Start(); err != nil {
		if t.onerror != nil {
			t.onerror(err)
		}
		return err
	}

	// Read stdout in background
	go t.readStdout(stdout)

	// Monitor process exit in background
	go func() {
		_ = cmd.Wait()
		t.mu.Lock()
		t.process = nil
		t.mu.Unlock()
		if t.onclose != nil {
			t.onclose()
		}
	}()

	return nil
}

func (t *StdioMCPTransport) readStdout(stdout io.ReadCloser) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		msg, err := DeserializeStdioMessage(line)
		if err != nil {
			if t.onerror != nil {
				t.onerror(err)
			}
			continue
		}
		if t.onmessage != nil {
			t.onmessage(msg)
		}
	}

	if err := scanner.Err(); err != nil {
		if t.ctx.Err() != nil {
			// Context cancelled, not an error
			return
		}
		if t.onerror != nil {
			t.onerror(err)
		}
	}
}

// Close stops the child process.
func (t *StdioMCPTransport) Close() error {
	t.mu.Lock()
	if t.cancel != nil {
		t.cancel()
	}
	t.process = nil
	t.readBuffer.clear()
	t.mu.Unlock()
	return nil
}

// Send sends a JSON-RPC message to the child process via stdin.
func (t *StdioMCPTransport) Send(message JSONRPCMessage) error {
	t.mu.Lock()
	stdin := t.stdin
	t.mu.Unlock()

	if stdin == nil {
		return NewMCPClientError(MCPClientErrorOptions{
			Message: "StdioClientTransport not connected",
		})
	}

	serialized := SerializeStdioMessage(message)
	_, err := io.WriteString(stdin, serialized)
	return err
}

// readBuffer accumulates bytes and extracts newline-delimited JSON-RPC messages.
type readBuffer struct {
	buf bytes.Buffer
}

func newReadBuffer() *readBuffer {
	return &readBuffer{}
}

func (rb *readBuffer) append(data []byte) {
	rb.buf.Write(data)
}

func (rb *readBuffer) readMessage() (JSONRPCMessage, bool) {
	data := rb.buf.Bytes()
	idx := bytes.IndexByte(data, '\n')
	if idx == -1 {
		return JSONRPCMessage{}, false
	}

	line := string(data[:idx])
	rb.buf.Next(idx + 1)

	msg, err := DeserializeStdioMessage(line)
	if err != nil {
		return JSONRPCMessage{}, false
	}
	return msg, true
}

func (rb *readBuffer) clear() {
	rb.buf.Reset()
}

// SerializeStdioMessage serializes a JSONRPCMessage for stdio transport (JSON + newline).
func SerializeStdioMessage(message JSONRPCMessage) string {
	data, err := json.Marshal(message)
	if err != nil {
		return "{}\n"
	}
	return string(data) + "\n"
}

// DeserializeStdioMessage deserializes a JSON line into a JSONRPCMessage.
func DeserializeStdioMessage(line string) (JSONRPCMessage, error) {
	return ParseJSONRPCMessage([]byte(line))
}
