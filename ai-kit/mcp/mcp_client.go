// Ported from: packages/mcp/src/tool/mcp-client.ts
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
)

const clientVersion = "1.0.0"

// MCPClientConfig is the configuration for creating an MCP client.
type MCPClientConfig struct {
	// Transport can be either an MCPTransport or an MCPTransportConfig.
	// If MCPTransportConfig is provided, a transport will be created from it.
	Transport interface{}

	// OnUncaughtError is an optional callback for uncaught errors.
	OnUncaughtError func(error)

	// Name is the optional client name, defaults to "ai-sdk-mcp-client".
	Name string

	// Version is the optional client version, defaults to "1.0.0".
	Version string

	// Capabilities are optional client capabilities to advertise during initialization.
	Capabilities *ClientCapabilities
}

// MCPClient is the interface for an MCP client.
type MCPClient interface {
	// ListTools lists available tools from the MCP server.
	ListTools(ctx context.Context, params *PaginatedRequestParams) (*ListToolsResult, error)

	// CallTool calls a tool on the MCP server.
	CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error)

	// ListResources lists available resources from the MCP server.
	ListResources(ctx context.Context, params *PaginatedRequestParams) (*ListResourcesResult, error)

	// ReadResource reads a resource from the MCP server.
	ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error)

	// ListResourceTemplates lists resource templates from the MCP server.
	ListResourceTemplates(ctx context.Context) (*ListResourceTemplatesResult, error)

	// ListPrompts lists prompts from the MCP server (experimental).
	ListPrompts(ctx context.Context, params *PaginatedRequestParams) (*ListPromptsResult, error)

	// GetPrompt gets a prompt from the MCP server (experimental).
	GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*GetPromptResult, error)

	// OnElicitationRequest registers a handler for elicitation requests.
	OnElicitationRequest(handler func(ElicitationRequest) (*ElicitResult, error))

	// Close closes the client connection.
	Close() error
}

// CreateMCPClient creates and initializes a new MCP client.
func CreateMCPClient(config MCPClientConfig) (MCPClient, error) {
	client := newDefaultMCPClient(config)
	if err := client.init(); err != nil {
		return nil, err
	}
	return client, nil
}

// defaultMCPClient is the default implementation of MCPClient.
type defaultMCPClient struct {
	transport          MCPTransport
	onUncaughtError    func(error)
	clientInfo         Configuration
	clientCapabilities ClientCapabilities
	requestMessageID   atomic.Int64
	responseHandlers   sync.Map // map[int64]chan responseOrError
	serverCapabilities ServerCapabilities
	isClosed           atomic.Bool
	elicitationHandler func(ElicitationRequest) (*ElicitResult, error)
}

// responseOrError holds either a response or an error for the response channel.
type responseOrError struct {
	response *JSONRPCResponse
	err      error
}

func newDefaultMCPClient(config MCPClientConfig) *defaultMCPClient {
	name := config.Name
	if name == "" {
		name = "ai-sdk-mcp-client"
	}
	version := config.Version
	if version == "" {
		version = clientVersion
	}

	c := &defaultMCPClient{
		onUncaughtError: config.OnUncaughtError,
		clientInfo: Configuration{
			Name:    name,
			Version: version,
		},
	}
	if config.Capabilities != nil {
		c.clientCapabilities = *config.Capabilities
	}
	c.isClosed.Store(true)

	// Set up transport
	switch t := config.Transport.(type) {
	case MCPTransport:
		c.transport = t
	case MCPTransportConfig:
		transport, err := CreateMCPTransport(t)
		if err != nil {
			// Store error to be returned during init
			c.transport = nil
			return c
		}
		c.transport = transport
	default:
		c.transport = nil
	}

	if c.transport != nil {
		c.transport.SetOnClose(func() { c.onClose() })
		c.transport.SetOnError(func(err error) { c.onError(err) })
		c.transport.SetOnMessage(func(msg JSONRPCMessage) {
			if msg.HasMethod() {
				if msg.HasID() {
					c.onRequestMessage(msg)
				} else {
					c.onError(NewMCPClientError(MCPClientErrorOptions{
						Message: "Unsupported message type",
					}))
				}
				return
			}
			c.onResponse(msg)
		})
	}

	return c
}

func (c *defaultMCPClient) init() error {
	if c.transport == nil {
		return NewMCPClientError(MCPClientErrorOptions{
			Message: "Transport is nil or invalid",
		})
	}

	if err := c.transport.Start(); err != nil {
		_ = c.Close()
		return err
	}
	c.isClosed.Store(false)

	ctx := context.Background()
	params := map[string]interface{}{
		"protocolVersion": LatestProtocolVersion,
		"capabilities":    c.clientCapabilities,
		"clientInfo":      c.clientInfo,
	}

	result, err := c.request(ctx, "initialize", params)
	if err != nil {
		_ = c.Close()
		return err
	}

	if result == nil {
		_ = c.Close()
		return NewMCPClientError(MCPClientErrorOptions{
			Message: "Server sent invalid initialize result",
		})
	}

	var initResult InitializeResult
	resultBytes, err := json.Marshal(result)
	if err != nil {
		_ = c.Close()
		return err
	}
	if err := json.Unmarshal(resultBytes, &initResult); err != nil {
		_ = c.Close()
		return NewMCPClientError(MCPClientErrorOptions{
			Message: "Failed to parse initialize result",
			Cause:   err,
		})
	}

	supported := false
	for _, v := range SupportedProtocolVersions {
		if v == initResult.ProtocolVersion {
			supported = true
			break
		}
	}
	if !supported {
		_ = c.Close()
		return NewMCPClientError(MCPClientErrorOptions{
			Message: fmt.Sprintf("Server's protocol version is not supported: %s", initResult.ProtocolVersion),
		})
	}

	c.serverCapabilities = initResult.Capabilities

	// Complete initialization handshake
	if err := c.notification("notifications/initialized", nil); err != nil {
		_ = c.Close()
		return err
	}

	return nil
}

// Close closes the client connection.
func (c *defaultMCPClient) Close() error {
	if c.isClosed.Load() {
		return nil
	}
	if c.transport != nil {
		if err := c.transport.Close(); err != nil {
			return err
		}
	}
	c.onClose()
	return nil
}

func (c *defaultMCPClient) assertCapability(method string) error {
	switch method {
	case "initialize":
		// Always allowed
	case "tools/list", "tools/call":
		if c.serverCapabilities.Tools == nil {
			return NewMCPClientError(MCPClientErrorOptions{
				Message: "Server does not support tools",
			})
		}
	case "resources/list", "resources/read", "resources/templates/list":
		if c.serverCapabilities.Resources == nil {
			return NewMCPClientError(MCPClientErrorOptions{
				Message: "Server does not support resources",
			})
		}
	case "prompts/list", "prompts/get":
		if c.serverCapabilities.Prompts == nil {
			return NewMCPClientError(MCPClientErrorOptions{
				Message: "Server does not support prompts",
			})
		}
	default:
		return NewMCPClientError(MCPClientErrorOptions{
			Message: fmt.Sprintf("Unsupported method: %s", method),
		})
	}
	return nil
}

func (c *defaultMCPClient) request(ctx context.Context, method string, params map[string]interface{}) (interface{}, error) {
	if c.isClosed.Load() {
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Attempted to send a request from a closed client",
		})
	}

	if err := c.assertCapability(method); err != nil {
		return nil, err
	}

	// Check context
	select {
	case <-ctx.Done():
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Request was aborted",
			Cause:   ctx.Err(),
		})
	default:
	}

	messageID := c.requestMessageID.Add(1) - 1
	responseCh := make(chan responseOrError, 1)

	c.responseHandlers.Store(messageID, responseCh)

	reqMsg, err := MakeRequestMessage(JSONRPCRequest{
		Method: method,
		Params: params,
		ID:     messageID,
	})
	if err != nil {
		c.responseHandlers.Delete(messageID)
		return nil, err
	}

	if err := c.transport.Send(reqMsg); err != nil {
		c.responseHandlers.Delete(messageID)
		return nil, err
	}

	// Wait for response or context cancellation
	select {
	case <-ctx.Done():
		c.responseHandlers.Delete(messageID)
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Request was aborted",
			Cause:   ctx.Err(),
		})
	case r := <-responseCh:
		if r.err != nil {
			return nil, r.err
		}
		return r.response.Result, nil
	}
}

// ListTools lists available tools from the MCP server.
func (c *defaultMCPClient) ListTools(ctx context.Context, params *PaginatedRequestParams) (*ListToolsResult, error) {
	var p map[string]interface{}
	if params != nil {
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &p)
	}

	result, err := c.request(ctx, "tools/list", p)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var listResult ListToolsResult
	if err := json.Unmarshal(resultBytes, &listResult); err != nil {
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Failed to parse server response",
			Cause:   err,
		})
	}
	return &listResult, nil
}

// CallTool calls a tool on the MCP server.
func (c *defaultMCPClient) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	result, err := c.request(ctx, "tools/call", params)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var callResult CallToolResult
	if err := json.Unmarshal(resultBytes, &callResult); err != nil {
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Failed to parse server response",
			Cause:   err,
		})
	}
	return &callResult, nil
}

// ListResources lists available resources from the MCP server.
func (c *defaultMCPClient) ListResources(ctx context.Context, params *PaginatedRequestParams) (*ListResourcesResult, error) {
	var p map[string]interface{}
	if params != nil {
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &p)
	}

	result, err := c.request(ctx, "resources/list", p)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var listResult ListResourcesResult
	if err := json.Unmarshal(resultBytes, &listResult); err != nil {
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Failed to parse server response",
			Cause:   err,
		})
	}
	return &listResult, nil
}

// ReadResource reads a resource from the MCP server.
func (c *defaultMCPClient) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	params := map[string]interface{}{
		"uri": uri,
	}

	result, err := c.request(ctx, "resources/read", params)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var readResult ReadResourceResult
	if err := json.Unmarshal(resultBytes, &readResult); err != nil {
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Failed to parse server response",
			Cause:   err,
		})
	}
	return &readResult, nil
}

// ListResourceTemplates lists resource templates from the MCP server.
func (c *defaultMCPClient) ListResourceTemplates(ctx context.Context) (*ListResourceTemplatesResult, error) {
	result, err := c.request(ctx, "resources/templates/list", nil)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var listResult ListResourceTemplatesResult
	if err := json.Unmarshal(resultBytes, &listResult); err != nil {
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Failed to parse server response",
			Cause:   err,
		})
	}
	return &listResult, nil
}

// ListPrompts lists prompts from the MCP server.
func (c *defaultMCPClient) ListPrompts(ctx context.Context, params *PaginatedRequestParams) (*ListPromptsResult, error) {
	var p map[string]interface{}
	if params != nil {
		b, _ := json.Marshal(params)
		_ = json.Unmarshal(b, &p)
	}

	result, err := c.request(ctx, "prompts/list", p)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var listResult ListPromptsResult
	if err := json.Unmarshal(resultBytes, &listResult); err != nil {
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Failed to parse server response",
			Cause:   err,
		})
	}
	return &listResult, nil
}

// GetPrompt gets a prompt from the MCP server.
func (c *defaultMCPClient) GetPrompt(ctx context.Context, name string, arguments map[string]interface{}) (*GetPromptResult, error) {
	params := map[string]interface{}{
		"name": name,
	}
	if arguments != nil {
		params["arguments"] = arguments
	}

	result, err := c.request(ctx, "prompts/get", params)
	if err != nil {
		return nil, err
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var getResult GetPromptResult
	if err := json.Unmarshal(resultBytes, &getResult); err != nil {
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Failed to parse server response",
			Cause:   err,
		})
	}
	return &getResult, nil
}

// OnElicitationRequest registers a handler for elicitation requests from the server.
func (c *defaultMCPClient) OnElicitationRequest(handler func(ElicitationRequest) (*ElicitResult, error)) {
	c.elicitationHandler = handler
}

func (c *defaultMCPClient) notification(method string, params map[string]interface{}) error {
	msg, err := MakeNotificationMessage(JSONRPCNotification{
		Method: method,
		Params: params,
	})
	if err != nil {
		return err
	}
	return c.transport.Send(msg)
}

func (c *defaultMCPClient) onRequestMessage(msg JSONRPCMessage) {
	req, err := msg.AsRequest()
	if err != nil {
		c.onError(err)
		return
	}

	if req.Method != "elicitation/create" {
		errMsg, _ := MakeErrorMessage(JSONRPCError{
			ID: req.ID,
			Error: JSONRPCErrorDetail{
				Code:    -32601,
				Message: fmt.Sprintf("Unsupported request method: %s", req.Method),
			},
		})
		_ = c.transport.Send(errMsg)
		return
	}

	if c.elicitationHandler == nil {
		errMsg, _ := MakeErrorMessage(JSONRPCError{
			ID: req.ID,
			Error: JSONRPCErrorDetail{
				Code:    -32601,
				Message: "No elicitation handler registered on client",
			},
		})
		_ = c.transport.Send(errMsg)
		return
	}

	// Parse the elicitation request
	var elicitReq ElicitationRequest
	elicitReq.Method = req.Method
	if req.Params != nil {
		paramsBytes, err := json.Marshal(req.Params)
		if err != nil {
			errMsg, _ := MakeErrorMessage(JSONRPCError{
				ID: req.ID,
				Error: JSONRPCErrorDetail{
					Code:    -32602,
					Message: fmt.Sprintf("Invalid elicitation request: %v", err),
				},
			})
			_ = c.transport.Send(errMsg)
			return
		}
		if err := json.Unmarshal(paramsBytes, &elicitReq.Params); err != nil {
			errMsg, _ := MakeErrorMessage(JSONRPCError{
				ID: req.ID,
				Error: JSONRPCErrorDetail{
					Code:    -32602,
					Message: fmt.Sprintf("Invalid elicitation request: %v", err),
				},
			})
			_ = c.transport.Send(errMsg)
			return
		}
	}

	result, err := c.elicitationHandler(elicitReq)
	if err != nil {
		errMsg, _ := MakeErrorMessage(JSONRPCError{
			ID: req.ID,
			Error: JSONRPCErrorDetail{
				Code:    -32603,
				Message: err.Error(),
			},
		})
		_ = c.transport.Send(errMsg)
		c.onError(err)
		return
	}

	respMsg, _ := MakeResponseMessage(JSONRPCResponse{
		ID:     req.ID,
		Result: result,
	})
	_ = c.transport.Send(respMsg)
}

func (c *defaultMCPClient) onClose() {
	if c.isClosed.Load() {
		return
	}
	c.isClosed.Store(true)

	closedErr := NewMCPClientError(MCPClientErrorOptions{
		Message: "Connection closed",
	})

	c.responseHandlers.Range(func(key, value interface{}) bool {
		ch := value.(chan responseOrError)
		ch <- responseOrError{err: closedErr}
		c.responseHandlers.Delete(key)
		return true
	})
}

func (c *defaultMCPClient) onError(err error) {
	if c.onUncaughtError != nil {
		c.onUncaughtError(err)
	}
}

func (c *defaultMCPClient) onResponse(msg JSONRPCMessage) {
	mp, err := msg.AsMap()
	if err != nil {
		c.onError(NewMCPClientError(MCPClientErrorOptions{
			Message: "Failed to parse response message",
			Cause:   err,
		}))
		return
	}

	// Get the message ID
	rawID, ok := mp["id"]
	if !ok {
		c.onError(NewMCPClientError(MCPClientErrorOptions{
			Message: fmt.Sprintf("Protocol error: Received a response without an ID: %v", mp),
		}))
		return
	}

	// Convert ID to int64
	var messageID int64
	switch id := rawID.(type) {
	case float64:
		messageID = int64(id)
	case json.Number:
		v, _ := id.Int64()
		messageID = v
	default:
		c.onError(NewMCPClientError(MCPClientErrorOptions{
			Message: fmt.Sprintf("Protocol error: Received a response with a non-numeric ID: %v", rawID),
		}))
		return
	}

	raw, ok := c.responseHandlers.Load(messageID)
	if !ok {
		c.onError(NewMCPClientError(MCPClientErrorOptions{
			Message: fmt.Sprintf("Protocol error: Received a response for an unknown message ID: %v", mp),
		}))
		return
	}

	c.responseHandlers.Delete(messageID)
	ch := raw.(chan responseOrError)

	// Check if it's an error response
	if msg.HasError() {
		errResp, _ := msg.AsError()
		if errResp != nil {
			ch <- responseOrError{
				err: NewMCPClientError(MCPClientErrorOptions{
					Message: errResp.Error.Message,
					Code:    intPtr(errResp.Error.Code),
					Data:    errResp.Error.Data,
				}),
			}
			return
		}
	}

	// It's a success response
	resp, respErr := msg.AsResponse()
	if respErr != nil {
		ch <- responseOrError{
			err: NewMCPClientError(MCPClientErrorOptions{
				Message: "Failed to parse server response",
				Cause:   respErr,
			}),
		}
		return
	}

	ch <- responseOrError{response: resp}
}

func intPtr(i int) *int { return &i }
