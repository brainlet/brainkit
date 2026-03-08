// Ported from: packages/mcp/src/tool/mcp-transport.ts
package mcp

// MCPTransport is the transport interface for MCP (Model Context Protocol)
// communication. Maps to the Transport interface in the MCP spec.
type MCPTransport interface {
	// Start initializes and starts the transport.
	Start() error

	// Send sends a JSON-RPC message through the transport.
	Send(message JSONRPCMessage) error

	// Close cleans up and closes the transport.
	Close() error

	// SetOnClose sets the handler for transport closure.
	SetOnClose(handler func())

	// SetOnError sets the handler for transport errors.
	SetOnError(handler func(error))

	// SetOnMessage sets the handler for received messages.
	SetOnMessage(handler func(JSONRPCMessage))
}

// MCPTransportConfig is the configuration for creating built-in transports.
type MCPTransportConfig struct {
	// Type is "sse" or "http".
	Type string

	// URL is the URL of the MCP server.
	URL string

	// Headers contains additional HTTP headers to be sent with requests.
	Headers map[string]string

	// AuthProvider is an optional OAuth client provider for authentication.
	AuthProvider OAuthClientProvider
}

// CreateMCPTransport creates an MCPTransport from a transport configuration.
func CreateMCPTransport(config MCPTransportConfig) (MCPTransport, error) {
	switch config.Type {
	case "sse":
		return NewSseMCPTransport(SseMCPTransportConfig{
			URL:          config.URL,
			Headers:      config.Headers,
			AuthProvider: config.AuthProvider,
		}), nil
	case "http":
		return NewHttpMCPTransport(HttpMCPTransportConfig{
			URL:          config.URL,
			Headers:      config.Headers,
			AuthProvider: config.AuthProvider,
		}), nil
	default:
		return nil, NewMCPClientError(MCPClientErrorOptions{
			Message: "Unsupported or invalid transport configuration. If you are using a custom transport, make sure it implements the MCPTransport interface.",
		})
	}
}

// IsCustomMCPTransport checks if a value implements the MCPTransport interface.
// In Go, this is handled naturally by the type system, but we provide this
// as a convenience for code ported from TS.
func IsCustomMCPTransport(transport interface{}) bool {
	_, ok := transport.(MCPTransport)
	return ok
}
