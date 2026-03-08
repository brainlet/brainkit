// Ported from: packages/mcp/src/error/mcp-client-error.ts
package mcp

import "fmt"

// MCPClientError represents an error that occurred with the MCP client.
type MCPClientError struct {
	Name    string
	Message string
	Cause   error
	Data    interface{}
	Code    *int
}

func NewMCPClientError(opts MCPClientErrorOptions) *MCPClientError {
	name := "MCPClientError"
	if opts.Name != "" {
		name = opts.Name
	}
	return &MCPClientError{
		Name:    name,
		Message: opts.Message,
		Cause:   opts.Cause,
		Data:    opts.Data,
		Code:    opts.Code,
	}
}

type MCPClientErrorOptions struct {
	Name    string
	Message string
	Cause   error
	Data    interface{}
	Code    *int
}

func (e *MCPClientError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Name, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

func (e *MCPClientError) Unwrap() error {
	return e.Cause
}
