package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// ServerConfig configures one external MCP server connection.
type ServerConfig = types.MCPServerConfig

// ToolInfo describes a tool from an MCP server.
type ToolInfo struct {
	ServerName  string          `json:"serverName"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// MCPManager manages connections to multiple MCP servers.
type MCPManager struct {
	mu        syncx.RWMutex
	closeMu   sync.Mutex
	clients   map[string]*mcpClientEntry
	tools     map[string][]ToolInfo
	newClient mcpClientFactory
	closing   bool
	closed    bool
}

type mcpClient interface {
	Initialize(context.Context, mcp.InitializeRequest) (*mcp.InitializeResult, error)
	ListTools(context.Context, mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	CallTool(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	Close() error
}

type mcpClientEntry struct {
	client    mcpClient
	closing   bool
	closeDone chan mcpCloseResult
}

type mcpCloseResult struct {
	err error
}

type mcpClientFactory func(context.Context, string, ServerConfig) (mcpClient, error)

type mcpCloseTarget struct {
	name  string
	entry *mcpClientEntry
}

var (
	errMCPManagerClosed          = errors.New("mcp: manager closed")
	errMCPServerAlreadyConnected = errors.New("mcp: server already connected")
)

// NewManager creates a new MCPManager.
func NewManager() *MCPManager {
	return newManagerWithFactory(defaultMCPClientFactory)
}

func newManagerWithFactory(factory mcpClientFactory) *MCPManager {
	if factory == nil {
		factory = defaultMCPClientFactory
	}
	return &MCPManager{
		clients:   make(map[string]*mcpClientEntry),
		tools:     make(map[string][]ToolInfo),
		newClient: factory,
	}
}

// Connect establishes a connection to an MCP server.
func (m *MCPManager) Connect(ctx context.Context, name string, cfg ServerConfig) error {
	m.mu.Lock()
	m.ensureInitializedLocked()
	factory := m.newClient
	if m.closed || m.closing {
		m.mu.Unlock()
		return fmt.Errorf("mcp: connect %q: %w", name, errMCPManagerClosed)
	}
	if _, ok := m.clients[name]; ok {
		m.mu.Unlock()
		return fmt.Errorf("mcp: connect %q: %w", name, errMCPServerAlreadyConnected)
	}
	m.mu.Unlock()

	c, err := factory(ctx, name, cfg)
	if err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("mcp: create client %q: nil client", name)
	}

	// Initialize the MCP protocol
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "brainkit",
		Version: "1.0.0",
	}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}

	if _, err := c.Initialize(ctx, initReq); err != nil {
		initErr := fmt.Errorf("mcp: initialize %q: %w", name, err)
		if closeErr := c.Close(); closeErr != nil {
			return errors.Join(initErr, fmt.Errorf("mcp: close %q after initialize failure: %w", name, closeErr))
		}
		return initErr
	}

	// Fetch and cache tools
	var tools []ToolInfo
	toolsResult, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err == nil {
		tools = make([]ToolInfo, 0, len(toolsResult.Tools))
		for _, t := range toolsResult.Tools {
			schema, _ := json.Marshal(t.InputSchema)
			tools = append(tools, ToolInfo{
				ServerName:  name,
				Name:        t.Name,
				Description: t.Description,
				InputSchema: schema,
			})
		}
	}

	entry := &mcpClientEntry{client: c}
	m.mu.Lock()
	m.ensureInitializedLocked()
	if m.closed || m.closing {
		m.mu.Unlock()
		closedErr := fmt.Errorf("mcp: connect %q: %w", name, errMCPManagerClosed)
		if closeErr := c.Close(); closeErr != nil {
			return errors.Join(closedErr, fmt.Errorf("mcp: close %q after manager close: %w", name, closeErr))
		}
		return closedErr
	}
	if _, ok := m.clients[name]; ok {
		m.mu.Unlock()
		connectedErr := fmt.Errorf("mcp: connect %q: %w", name, errMCPServerAlreadyConnected)
		if closeErr := c.Close(); closeErr != nil {
			return errors.Join(connectedErr, fmt.Errorf("mcp: close duplicate client %q: %w", name, closeErr))
		}
		return connectedErr
	}
	m.clients[name] = entry
	m.tools[name] = tools
	m.mu.Unlock()

	return nil
}

// ListTools returns all tools from all connected servers.
func (m *MCPManager) ListTools() []ToolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var all []ToolInfo
	for _, tools := range m.tools {
		all = append(all, tools...)
	}
	return all
}

// ListToolsForServer returns tools from a specific server.
func (m *MCPManager) ListToolsForServer(name string) []ToolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]ToolInfo(nil), m.tools[name]...)
}

// CallTool calls a tool on a specific MCP server.
func (m *MCPManager) CallTool(ctx context.Context, serverName, toolName string, args json.RawMessage) (json.RawMessage, error) {
	m.mu.RLock()
	entry, ok := m.clients[serverName]
	m.mu.RUnlock()

	if !ok {
		return nil, &sdk.NotFoundError{Resource: "mcp-server", Name: serverName}
	}
	c := entry.client

	var argsMap map[string]interface{}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &argsMap); err != nil {
			return nil, fmt.Errorf("mcp: unmarshal args: %w", err)
		}
	}

	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = toolName
	callReq.Params.Arguments = argsMap

	result, err := c.CallTool(ctx, callReq)
	if err != nil {
		return nil, fmt.Errorf("mcp: call %s.%s: %w", serverName, toolName, err)
	}

	// Extract text content from result
	var texts []string
	for _, content := range result.Content {
		if tc, ok := content.(mcp.TextContent); ok {
			texts = append(texts, tc.Text)
		}
	}

	if len(texts) == 1 {
		if json.Valid([]byte(texts[0])) {
			return json.RawMessage(texts[0]), nil
		}
		return json.Marshal(texts[0])
	}
	return json.Marshal(texts)
}

// Disconnect closes a specific server connection.
func (m *MCPManager) Disconnect(name string) error {
	m.closeMu.Lock()
	defer m.closeMu.Unlock()

	m.mu.Lock()
	m.ensureInitializedLocked()
	entry, ok := m.clients[name]
	if ok {
		m.closing = true
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}
	err := m.closeTargetsContext(context.Background(), []mcpCloseTarget{{name: name, entry: entry}})
	m.mu.Lock()
	m.closing = false
	m.mu.Unlock()
	return err
}

// Close closes all connections.
func (m *MCPManager) Close() error {
	return m.CloseContext(context.Background())
}

func (m *MCPManager) CloseContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	m.closeMu.Lock()
	defer m.closeMu.Unlock()

	m.mu.Lock()
	m.ensureInitializedLocked()
	targets := make([]mcpCloseTarget, 0, len(m.clients))
	for name, entry := range m.clients {
		targets = append(targets, mcpCloseTarget{name: name, entry: entry})
	}
	m.closing = true
	m.closed = true
	m.mu.Unlock()
	err := m.closeTargetsContext(ctx, targets)
	m.mu.Lock()
	m.closing = false
	m.mu.Unlock()
	return err
}

func (m *MCPManager) ensureInitializedLocked() {
	if m.clients == nil {
		m.clients = make(map[string]*mcpClientEntry)
	}
	if m.tools == nil {
		m.tools = make(map[string][]ToolInfo)
	}
	if m.newClient == nil {
		m.newClient = defaultMCPClientFactory
	}
}

func (m *MCPManager) closeTargetsContext(ctx context.Context, targets []mcpCloseTarget) error {
	if ctx == nil {
		ctx = context.Background()
	}
	waits := make([]mcpCloseWait, 0, len(targets))
	for _, target := range targets {
		if target.entry == nil || target.entry.client == nil {
			continue
		}
		if done := m.startTargetClose(target); done != nil {
			waits = append(waits, mcpCloseWait{name: target.name, done: done})
		}
	}

	var err error
	for _, wait := range waits {
		select {
		case result := <-wait.done:
			if result.err != nil {
				err = errors.Join(err, fmt.Errorf("mcp: close server %q: %w", wait.name, result.err))
			}
		case <-ctx.Done():
			err = errors.Join(err, fmt.Errorf("mcp: close server %q: %w", wait.name, ctx.Err()))
		}
	}
	return err
}

type mcpCloseWait struct {
	name string
	done <-chan mcpCloseResult
}

func (m *MCPManager) startTargetClose(target mcpCloseTarget) <-chan mcpCloseResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureInitializedLocked()
	entry := target.entry
	if entry == nil || entry.client == nil {
		return nil
	}
	if current := m.clients[target.name]; current != entry {
		return nil
	}
	if entry.closing && entry.closeDone != nil {
		return entry.closeDone
	}
	done := make(chan mcpCloseResult, 1)
	entry.closing = true
	entry.closeDone = done
	client := entry.client
	go func() {
		closeErr := client.Close()
		m.finishTargetClose(target.name, entry, closeErr)
		done <- mcpCloseResult{err: closeErr}
		close(done)
	}()
	return done
}

func (m *MCPManager) finishTargetClose(name string, entry *mcpClientEntry, closeErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureInitializedLocked()
	if current := m.clients[name]; current != entry {
		return
	}
	if closeErr != nil {
		entry.closing = false
		entry.closeDone = nil
		return
	}
	delete(m.clients, name)
	delete(m.tools, name)
}

func defaultMCPClientFactory(ctx context.Context, name string, cfg ServerConfig) (mcpClient, error) {
	var c *client.Client

	if cfg.Command != "" {
		// Stdio transport — convert env map to []string{"KEY=VALUE"}
		var envSlice []string
		for k, v := range cfg.Env {
			envSlice = append(envSlice, k+"="+v)
		}
		t := transport.NewStdio(cfg.Command, envSlice, cfg.Args...)
		c = client.NewClient(t)
		if err := c.Start(ctx); err != nil {
			return nil, fmt.Errorf("mcp: start stdio client %q: %w", name, err)
		}
	} else if cfg.URL != "" {
		// HTTP/Streamable HTTP transport
		t, err := transport.NewStreamableHTTP(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("mcp: create HTTP transport for %q: %w", name, err)
		}
		c = client.NewClient(t)
		if err := c.Start(ctx); err != nil {
			return nil, fmt.Errorf("mcp: start HTTP client %q: %w", name, err)
		}
	} else {
		return nil, fmt.Errorf("mcp: server %q has no command or url", name)
	}
	return c, nil
}
