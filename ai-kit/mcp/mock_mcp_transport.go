// Ported from: packages/mcp/src/tool/mock-mcp-transport.ts
package mcp

import (
	"encoding/json"
	"fmt"
	"time"
)

// MockMCPTransportConfig is configuration for the mock transport.
type MockMCPTransportConfig struct {
	OverrideTools          []MCPTool
	Resources              []MCPResource
	Prompts                []MCPPrompt
	PromptResults          map[string]MockPromptResult
	ResourceTemplates      []ResourceTemplate
	ResourceContents       []ResourceContents
	FailOnInvalidToolParams bool
	InitializeResult       map[string]interface{}
	SendError              bool
	ToolCallResults        map[string]CallToolResult
}

// MockPromptResult holds a prompt result for the mock transport.
type MockPromptResult struct {
	Description string                   `json:"description,omitempty"`
	Messages    []map[string]interface{} `json:"messages"`
}

// MockMCPTransport is a mock transport for testing purposes.
type MockMCPTransport struct {
	tools                   []MCPTool
	resources               []MCPResource
	resourceTemplates       []ResourceTemplate
	resourceContents        []ResourceContents
	prompts                 []MCPPrompt
	promptResults           map[string]MockPromptResult
	failOnInvalidToolParams bool
	initializeResult        map[string]interface{}
	sendError               bool
	toolCallResults         map[string]CallToolResult

	onclose   func()
	onerror   func(error)
	onmessage func(JSONRPCMessage)
}

var defaultMockTools = []MCPTool{
	{
		Name:        "mock-tool",
		Description: "A mock tool for testing",
		InputSchema: MCPToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"foo": map[string]interface{}{"type": "string"},
			},
		},
	},
	{
		Name:        "mock-tool-no-args",
		Description: "A mock tool for testing",
		InputSchema: MCPToolInputSchema{
			Type: "object",
		},
	},
}

// NewMockMCPTransport creates a new mock MCP transport with the given configuration.
func NewMockMCPTransport(config *MockMCPTransportConfig) *MockMCPTransport {
	m := &MockMCPTransport{
		tools: defaultMockTools,
		resources: []MCPResource{
			{
				URI:         "file:///mock/resource.txt",
				Name:        "resource.txt",
				Description: "Mock resource",
				MimeType:    "text/plain",
			},
		},
		prompts: []MCPPrompt{
			{
				Name:        "code_review",
				Title:       "Request Code Review",
				Description: "Asks the LLM to analyze code quality and suggest improvements",
				Arguments: []PromptArgument{
					{Name: "code", Description: "The code to review", Required: boolPtr(true)},
				},
			},
		},
		promptResults: map[string]MockPromptResult{
			"code_review": {
				Description: "Code review prompt",
				Messages: []map[string]interface{}{
					{
						"role": "user",
						"content": map[string]interface{}{
							"type": "text",
							"text": "Please review this code:\nfunction add(a, b) { return a + b; }",
						},
					},
				},
			},
		},
		resourceTemplates: []ResourceTemplate{
			{
				URITemplate: "file:///{path}",
				Name:        "mock-template",
				Description: "Mock template",
			},
		},
		resourceContents: []ResourceContents{
			{
				URI:      "file:///mock/resource.txt",
				Text:     "Mock resource content",
				MimeType: "text/plain",
			},
		},
		toolCallResults: make(map[string]CallToolResult),
	}

	if config != nil {
		if config.OverrideTools != nil {
			m.tools = config.OverrideTools
		}
		if config.Resources != nil {
			m.resources = config.Resources
		}
		if config.Prompts != nil {
			m.prompts = config.Prompts
		}
		if config.PromptResults != nil {
			m.promptResults = config.PromptResults
		}
		if config.ResourceTemplates != nil {
			m.resourceTemplates = config.ResourceTemplates
		}
		if config.ResourceContents != nil {
			m.resourceContents = config.ResourceContents
		}
		m.failOnInvalidToolParams = config.FailOnInvalidToolParams
		if config.InitializeResult != nil {
			m.initializeResult = config.InitializeResult
		}
		m.sendError = config.SendError
		if config.ToolCallResults != nil {
			m.toolCallResults = config.ToolCallResults
		}
	}

	return m
}

func boolPtr(b bool) *bool { return &b }

func (m *MockMCPTransport) SetOnClose(handler func())              { m.onclose = handler }
func (m *MockMCPTransport) SetOnError(handler func(error))         { m.onerror = handler }
func (m *MockMCPTransport) SetOnMessage(handler func(JSONRPCMessage)) { m.onmessage = handler }

// Start starts the mock transport.
func (m *MockMCPTransport) Start() error {
	if m.sendError && m.onerror != nil {
		m.onerror(fmt.Errorf("Unknown error"))
	}
	return nil
}

// Send processes a message and sends mock responses.
func (m *MockMCPTransport) Send(message JSONRPCMessage) error {
	mp, err := message.AsMap()
	if err != nil {
		return err
	}

	method, hasMethod := mp["method"].(string)
	id, hasID := mp["id"]

	if !hasMethod || !hasID {
		return nil
	}

	// Small delay to simulate network latency
	time.Sleep(10 * time.Millisecond)

	params, _ := mp["params"].(map[string]interface{})

	switch method {
	case "initialize":
		result := m.initializeResult
		if result == nil {
			capabilities := map[string]interface{}{}
			if len(m.tools) > 0 {
				capabilities["tools"] = map[string]interface{}{}
			}
			if len(m.resources) > 0 {
				capabilities["resources"] = map[string]interface{}{}
			}
			if len(m.prompts) > 0 {
				capabilities["prompts"] = map[string]interface{}{}
			}
			result = map[string]interface{}{
				"protocolVersion": "2025-06-18",
				"serverInfo": map[string]interface{}{
					"name":    "mock-mcp-server",
					"version": "1.0.0",
				},
				"capabilities": capabilities,
			}
		}
		m.sendResponse(id, result)

	case "resources/list":
		m.sendResponse(id, map[string]interface{}{
			"resources": m.resources,
		})

	case "resources/read":
		uri, _ := params["uri"].(string)
		var contents []ResourceContents
		for _, c := range m.resourceContents {
			if c.URI == uri {
				contents = append(contents, c)
			}
		}
		if len(contents) == 0 {
			m.sendError_(id, -32002, fmt.Sprintf("Resource %s not found", uri), nil)
			return nil
		}
		m.sendResponse(id, map[string]interface{}{
			"contents": contents,
		})

	case "resources/templates/list":
		m.sendResponse(id, map[string]interface{}{
			"resourceTemplates": m.resourceTemplates,
		})

	case "prompts/list":
		m.sendResponse(id, map[string]interface{}{
			"prompts": m.prompts,
		})

	case "prompts/get":
		name, _ := params["name"].(string)
		result, ok := m.promptResults[name]
		if !ok {
			m.sendError_(id, -32602, fmt.Sprintf("Invalid params: Unknown prompt %s", name), nil)
			return nil
		}
		m.sendResponse(id, result)

	case "tools/list":
		if len(m.tools) == 0 {
			m.sendError_(id, -32000, "Method not supported", nil)
			return nil
		}
		m.sendResponse(id, map[string]interface{}{
			"tools": m.tools,
		})

	case "tools/call":
		toolName, _ := params["name"].(string)
		var found *MCPTool
		for i := range m.tools {
			if m.tools[i].Name == toolName {
				found = &m.tools[i]
				break
			}
		}

		if found == nil {
			names := make([]string, len(m.tools))
			for i, t := range m.tools {
				names[i] = t.Name
			}
			m.sendError_(id, -32601, fmt.Sprintf("Tool %s not found", toolName), map[string]interface{}{
				"availableTools": names,
				"requestedTool":  toolName,
			})
			return nil
		}

		if m.failOnInvalidToolParams {
			args, _ := params["arguments"]
			argsJSON, _ := json.Marshal(args)
			m.sendError_(id, -32602, fmt.Sprintf("Invalid tool inputSchema: %s", string(argsJSON)), map[string]interface{}{
				"expectedSchema":    found.InputSchema,
				"receivedArguments": args,
			})
			return nil
		}

		if customResult, ok := m.toolCallResults[toolName]; ok {
			m.sendResponse(id, customResult)
			return nil
		}

		m.sendResponse(id, map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "Mock tool call result",
				},
			},
		})
	}

	return nil
}

func (m *MockMCPTransport) sendResponse(id interface{}, result interface{}) {
	data, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
	if err != nil {
		return
	}
	msg := NewJSONRPCMessage(data)
	if m.onmessage != nil {
		m.onmessage(msg)
	}
}

func (m *MockMCPTransport) sendError_(id interface{}, code int, message string, data interface{}) {
	errObj := map[string]interface{}{
		"code":    code,
		"message": message,
	}
	if data != nil {
		errObj["data"] = data
	}
	respData, err := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   errObj,
	})
	if err != nil {
		return
	}
	msg := NewJSONRPCMessage(respData)
	if m.onmessage != nil {
		m.onmessage(msg)
	}
}

// Close closes the mock transport.
func (m *MockMCPTransport) Close() error {
	if m.onclose != nil {
		m.onclose()
	}
	return nil
}
