package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// ServerConfig defines an MCP server connection.
type ServerConfig struct {
	// Stdio transport: command + args (e.g., "npx", ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"])
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`

	// HTTP transport: URL (e.g., "http://localhost:8080/mcp")
	URL string `json:"url,omitempty"`
}

// ToolInfo describes a tool from an MCP server.
type ToolInfo struct {
	ServerName  string          `json:"serverName"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// MCPManager manages connections to multiple MCP servers.
type MCPManager struct {
	mu      sync.RWMutex
	clients map[string]*client.Client
	tools   map[string][]ToolInfo
}

// New creates a new MCPManager.
func New() *MCPManager {
	return &MCPManager{
		clients: make(map[string]*client.Client),
		tools:   make(map[string][]ToolInfo),
	}
}

// Connect establishes a connection to an MCP server.
func (m *MCPManager) Connect(ctx context.Context, name string, cfg ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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
			return fmt.Errorf("mcp: start stdio client %q: %w", name, err)
		}
	} else if cfg.URL != "" {
		// HTTP/Streamable HTTP transport
		t, err := transport.NewStreamableHTTP(cfg.URL)
		if err != nil {
			return fmt.Errorf("mcp: create HTTP transport for %q: %w", name, err)
		}
		c = client.NewClient(t)
		if err := c.Start(ctx); err != nil {
			return fmt.Errorf("mcp: start HTTP client %q: %w", name, err)
		}
	} else {
		return fmt.Errorf("mcp: server %q has no command or url", name)
	}

	// Initialize the MCP protocol
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "brainkit",
		Version: "1.0.0",
	}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}

	_, err := c.Initialize(ctx, initReq)
	if err != nil {
		c.Close()
		return fmt.Errorf("mcp: initialize %q: %w", name, err)
	}

	m.clients[name] = c

	// Fetch and cache tools
	toolsResult, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		m.tools[name] = nil
		return nil
	}

	tools := make([]ToolInfo, 0, len(toolsResult.Tools))
	for _, t := range toolsResult.Tools {
		schema, _ := json.Marshal(t.InputSchema)
		tools = append(tools, ToolInfo{
			ServerName:  name,
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		})
	}
	m.tools[name] = tools

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
	return m.tools[name]
}

// CallTool calls a tool on a specific MCP server.
func (m *MCPManager) CallTool(ctx context.Context, serverName, toolName string, args json.RawMessage) (json.RawMessage, error) {
	m.mu.RLock()
	c, ok := m.clients[serverName]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("mcp: server %q not connected", serverName)
	}

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
	m.mu.Lock()
	defer m.mu.Unlock()

	if c, ok := m.clients[name]; ok {
		c.Close()
		delete(m.clients, name)
		delete(m.tools, name)
	}
	return nil
}

// Close closes all connections.
func (m *MCPManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, c := range m.clients {
		c.Close()
		delete(m.clients, name)
		delete(m.tools, name)
	}
	return nil
}
