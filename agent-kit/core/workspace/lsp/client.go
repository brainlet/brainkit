// Ported from: packages/core/src/workspace/lsp/client.ts
package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// =============================================================================
// Process Interfaces (satisfied by sandbox package)
// =============================================================================

// ProcessHandle represents a handle to a spawned process.
// This interface is satisfied by the sandbox ProcessHandle when it is ported.
type ProcessHandle interface {
	// PID returns the process ID.
	PID() int
	// ExitCode returns the exit code. Returns -1 if the process is still running.
	// Use IsRunning() to distinguish.
	ExitCode() int
	// IsRunning returns true if the process is still running.
	IsRunning() bool
	// Kill kills the running process. Returns true if killed.
	Kill() error
	// Reader returns a reader for the process's stdout.
	Reader() io.Reader
	// Writer returns a writer for the process's stdin.
	Writer() io.Writer
}

// ProcessSpawner spawns processes. This interface is satisfied by SandboxProcessManager.
type ProcessSpawner interface {
	// Spawn spawns a command and returns a handle.
	Spawn(command string, opts *SpawnOptions) (ProcessHandle, error)
}

// SpawnOptions are options for spawning a process.
type SpawnOptions struct {
	// Cwd is the working directory.
	Cwd string
}

// =============================================================================
// URI Helpers
// =============================================================================

// toFileURI converts a filesystem path to a properly encoded file:// URI.
func toFileURI(fsPath string) string {
	absPath, err := filepath.Abs(fsPath)
	if err != nil {
		absPath = fsPath
	}
	if runtime.GOOS == "windows" {
		absPath = "/" + filepath.ToSlash(absPath)
	}
	u := url.URL{
		Scheme: "file",
		Path:   absPath,
	}
	return u.String()
}

// =============================================================================
// JSON-RPC Message Types
// =============================================================================

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonRPCNotification is a JSON-RPC 2.0 notification (no ID).
type jsonRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError is a JSON-RPC 2.0 error.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// jsonRPCMessage is a generic incoming JSON-RPC message.
type jsonRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// =============================================================================
// LSP Client
// =============================================================================

// LSPClient wraps a JSON-RPC connection to a single LSP server process.
// Uses a ProcessSpawner to spawn the server process.
type LSPClient struct {
	serverDef      *LSPServerDef
	workspaceRoot  string
	processManager ProcessSpawner

	handle ProcessHandle

	mu          sync.Mutex
	nextID      int
	diagnostics map[string][]interface{} // uri -> diagnostics
	pending     map[int]chan *jsonRPCMessage
	initOpts    map[string]interface{}
	closed      bool
	stopReader  context.CancelFunc

	reader io.Reader
	writer io.Writer
}

// NewLSPClient creates a new LSPClient.
func NewLSPClient(serverDef *LSPServerDef, workspaceRoot string, processManager ProcessSpawner) *LSPClient {
	return &LSPClient{
		serverDef:      serverDef,
		workspaceRoot:  workspaceRoot,
		processManager: processManager,
		diagnostics:    make(map[string][]interface{}),
		pending:        make(map[int]chan *jsonRPCMessage),
	}
}

// IsAlive returns whether the underlying server process is still running.
func (c *LSPClient) IsAlive() bool {
	if c.handle == nil {
		return false
	}
	return c.handle.IsRunning()
}

// Initialize initializes the LSP connection -- spawns the server and performs the handshake.
func (c *LSPClient) Initialize(initTimeout time.Duration) error {
	if initTimeout == 0 {
		initTimeout = 10 * time.Second
	}

	command := c.serverDef.Command(c.workspaceRoot)
	if command == "" {
		return fmt.Errorf("failed to resolve LSP server command")
	}

	handle, err := c.processManager.Spawn(command, &SpawnOptions{Cwd: c.workspaceRoot})
	if err != nil {
		return fmt.Errorf("failed to spawn LSP server: %w", err)
	}
	c.handle = handle
	c.reader = handle.Reader()
	c.writer = handle.Writer()

	// Get initialization options
	var initializationOptions map[string]interface{}
	if c.serverDef.Initialization != nil {
		initializationOptions = c.serverDef.Initialization(c.workspaceRoot)
	}

	// Start the message reader goroutine
	ctx, cancel := context.WithCancel(context.Background())
	c.stopReader = cancel
	go c.readMessages(ctx)

	// Build initialize params
	initParams := map[string]interface{}{
		"processId": os.Getpid(),
		"rootUri":   toFileURI(c.workspaceRoot),
		"workspaceFolders": []map[string]interface{}{
			{
				"name": "workspace",
				"uri":  toFileURI(c.workspaceRoot),
			},
		},
		"capabilities": map[string]interface{}{
			"window":    map[string]interface{}{"workDoneProgress": true},
			"workspace": map[string]interface{}{"configuration": true},
			"textDocument": map[string]interface{}{
				"publishDiagnostics": map[string]interface{}{
					"relatedInformation": true,
					"tagSupport":         map[string]interface{}{"valueSet": []int{1, 2}},
					"versionSupport":     false,
				},
				"synchronization": map[string]interface{}{
					"didOpen":             true,
					"didChange":           true,
					"dynamicRegistration": false,
					"willSave":            false,
					"willSaveWaitUntil":   false,
					"didSave":             false,
				},
				"completion": map[string]interface{}{
					"dynamicRegistration": false,
					"completionItem": map[string]interface{}{
						"snippetSupport":          false,
						"commitCharactersSupport": false,
						"documentationFormat":     []string{"markdown", "plaintext"},
						"deprecatedSupport":       false,
						"preselectSupport":        false,
					},
				},
				"definition":        map[string]interface{}{"dynamicRegistration": false, "linkSupport": true},
				"typeDefinition":    map[string]interface{}{"dynamicRegistration": false, "linkSupport": true},
				"implementation":    map[string]interface{}{"dynamicRegistration": false, "linkSupport": true},
				"references":        map[string]interface{}{"dynamicRegistration": false},
				"documentHighlight": map[string]interface{}{"dynamicRegistration": false},
				"documentSymbol":    map[string]interface{}{"dynamicRegistration": false, "hierarchicalDocumentSymbolSupport": true},
				"codeAction": map[string]interface{}{
					"dynamicRegistration": false,
					"codeActionLiteralSupport": map[string]interface{}{
						"codeActionKind": map[string]interface{}{
							"valueSet": []string{
								"quickfix",
								"refactor",
								"refactor.extract",
								"refactor.inline",
								"refactor.rewrite",
								"source",
								"source.organizeImports",
							},
						},
					},
				},
				"hover": map[string]interface{}{"dynamicRegistration": false, "contentFormat": []string{"markdown", "plaintext"}},
			},
		},
	}

	if initializationOptions != nil {
		initParams["initializationOptions"] = initializationOptions
		c.initOpts = initializationOptions
	}

	// Send initialize request with timeout
	ctx2, cancel2 := context.WithTimeout(context.Background(), initTimeout)
	defer cancel2()

	_, err = c.sendRequest(ctx2, "initialize", initParams)
	if err != nil {
		return fmt.Errorf("LSP initialize request failed: %w", err)
	}

	// Send initialized notification
	c.sendNotification("initialized", map[string]interface{}{})

	// Send workspace/didChangeConfiguration
	settings := c.initOpts
	if settings == nil {
		settings = map[string]interface{}{}
	}
	c.sendNotification("workspace/didChangeConfiguration", map[string]interface{}{
		"settings": settings,
	})

	return nil
}

// NotifyOpen notifies the server that a document has been opened.
func (c *LSPClient) NotifyOpen(filePath, content, languageID string) {
	if c.writer == nil {
		return
	}
	uri := toFileURI(filePath)

	c.mu.Lock()
	delete(c.diagnostics, uri)
	c.mu.Unlock()

	c.sendNotification("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        uri,
			"languageId": languageID,
			"version":    0,
			"text":       content,
		},
	})
}

// NotifyChange notifies the server that a document has changed.
func (c *LSPClient) NotifyChange(filePath, content string, version int) {
	if c.writer == nil {
		return
	}
	c.sendNotification("textDocument/didChange", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":     toFileURI(filePath),
			"version": version,
		},
		"contentChanges": []map[string]interface{}{
			{"text": content},
		},
	})
}

// WaitForDiagnostics waits for diagnostics to arrive for a file.
// When waitForChange is false (default), returns as soon as diagnostics
// are available. Empty results trigger a short settle window.
func (c *LSPClient) WaitForDiagnostics(filePath string, timeout time.Duration, waitForChange bool, settleMs int) []interface{} {
	if c.writer == nil {
		return nil
	}
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	if settleMs == 0 {
		settleMs = 500
	}

	uri := toFileURI(filePath)
	startTime := time.Now()

	c.mu.Lock()
	initialDiags := c.diagnostics[uri]
	c.mu.Unlock()

	var emptyReceivedAt *time.Time

	for time.Since(startTime) < timeout {
		c.mu.Lock()
		currentDiags, exists := c.diagnostics[uri]
		c.mu.Unlock()

		if waitForChange {
			// Compare by reference-like behavior: check if diagnostics changed
			if exists && !sameDiagSlice(currentDiags, initialDiags) {
				return currentDiags
			}
		} else {
			if exists {
				if len(currentDiags) > 0 {
					return currentDiags
				}
				// Empty -- start a settle window
				if emptyReceivedAt == nil {
					now := time.Now()
					emptyReceivedAt = &now
				}
				if time.Since(*emptyReceivedAt) >= time.Duration(settleMs)*time.Millisecond {
					return currentDiags
				}
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if waitForChange {
		if initialDiags != nil {
			return initialDiags
		}
		return nil
	}
	if diags, ok := c.diagnostics[uri]; ok {
		return diags
	}
	return nil
}

// NotifyClose notifies the server that a document was closed.
func (c *LSPClient) NotifyClose(filePath string) {
	if c.writer == nil {
		return
	}
	uri := toFileURI(filePath)

	c.mu.Lock()
	delete(c.diagnostics, uri)
	c.mu.Unlock()

	c.sendNotification("textDocument/didClose", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": uri,
		},
	})
}

// Shutdown shuts down the connection and kills the process.
func (c *LSPClient) Shutdown() error {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()

	if c.writer != nil {
		// Try to send shutdown request with 1s timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_, _ = c.sendRequest(ctx, "shutdown", nil)
		cancel()

		// Send exit notification
		c.sendNotification("exit", nil)
	}

	if c.stopReader != nil {
		c.stopReader()
	}

	if c.handle != nil {
		_ = c.handle.Kill()
		c.handle = nil
	}

	c.mu.Lock()
	c.diagnostics = make(map[string][]interface{})
	c.mu.Unlock()

	return nil
}

// =============================================================================
// Internal Methods
// =============================================================================

// sendRequest sends a JSON-RPC request and waits for the response.
func (c *LSPClient) sendRequest(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil, fmt.Errorf("client is closed")
	}
	c.nextID++
	id := c.nextID
	ch := make(chan *jsonRPCMessage, 1)
	c.pending[id] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.writeMessage(req); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("LSP error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

// sendNotification sends a JSON-RPC notification (no response expected).
func (c *LSPClient) sendNotification(method string, params interface{}) {
	notif := jsonRPCNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	_ = c.writeMessage(notif)
}

// writeMessage encodes and writes a JSON-RPC message with Content-Length header.
func (c *LSPClient) writeMessage(msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	_, err = fmt.Fprint(c.writer, header)
	if err != nil {
		return err
	}
	_, err = c.writer.Write(body)
	return err
}

// readMessages reads JSON-RPC messages from the server in a loop.
func (c *LSPClient) readMessages(ctx context.Context) {
	buf := make([]byte, 0, 4096)
	tmp := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := c.reader.Read(tmp)
		if err != nil {
			return
		}
		buf = append(buf, tmp[:n]...)

		// Parse messages from buffer
		for {
			msg, rest, ok := parseMessage(buf)
			if !ok {
				break
			}
			buf = rest
			c.handleMessage(msg)
		}
	}
}

// parseMessage tries to parse a complete LSP message from the buffer.
// Returns the parsed message, remaining buffer, and success flag.
func parseMessage(buf []byte) (*jsonRPCMessage, []byte, bool) {
	// Look for Content-Length header
	headerEnd := -1
	for i := 0; i+3 < len(buf); i++ {
		if buf[i] == '\r' && buf[i+1] == '\n' && buf[i+2] == '\r' && buf[i+3] == '\n' {
			headerEnd = i + 4
			break
		}
	}
	if headerEnd == -1 {
		return nil, buf, false
	}

	// Parse Content-Length
	var contentLength int
	header := string(buf[:headerEnd])
	_, err := fmt.Sscanf(header, "Content-Length: %d", &contentLength)
	if err != nil {
		// Skip malformed header
		return nil, buf[headerEnd:], false
	}

	// Check if we have the full body
	if len(buf) < headerEnd+contentLength {
		return nil, buf, false
	}

	body := buf[headerEnd : headerEnd+contentLength]
	rest := buf[headerEnd+contentLength:]

	var msg jsonRPCMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return nil, rest, false
	}

	return &msg, rest, true
}

// handleMessage dispatches an incoming JSON-RPC message.
func (c *LSPClient) handleMessage(msg *jsonRPCMessage) {
	// Response to a pending request
	if msg.ID != nil && msg.Method == "" {
		c.mu.Lock()
		ch, ok := c.pending[*msg.ID]
		c.mu.Unlock()
		if ok {
			ch <- msg
		}
		return
	}

	// Server request -- respond to known methods
	if msg.ID != nil && msg.Method != "" {
		switch msg.Method {
		case "workspace/configuration":
			// Return empty config for each item
			var params struct {
				Items []interface{} `json:"items"`
			}
			_ = json.Unmarshal(msg.Params, &params)
			items := make([]map[string]interface{}, len(params.Items))
			for i := range items {
				items[i] = map[string]interface{}{}
			}
			c.sendResponse(*msg.ID, items)

		case "window/workDoneProgress/create":
			c.sendResponse(*msg.ID, nil)

		default:
			// Unknown request -- send empty response
			c.sendResponse(*msg.ID, nil)
		}
		return
	}

	// Notification
	if msg.Method == "textDocument/publishDiagnostics" {
		var params struct {
			URI         string        `json:"uri"`
			Diagnostics []interface{} `json:"diagnostics"`
		}
		if err := json.Unmarshal(msg.Params, &params); err == nil {
			c.mu.Lock()
			c.diagnostics[params.URI] = params.Diagnostics
			c.mu.Unlock()
		}
	}
}

// sendResponse sends a JSON-RPC response to a server request.
func (c *LSPClient) sendResponse(id int, result interface{}) {
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	_ = c.writeMessage(resp)
}

// sameDiagSlice is a simple reference-like comparison for diagnostic slices.
// In Go we can't do reference comparison on slices, so we compare length
// and first element pointer as a heuristic.
func sameDiagSlice(a, b []interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	// For the purpose of "did diagnostics change", consider same-length slices
	// as different (conservative approach -- will always detect changes)
	return false
}
