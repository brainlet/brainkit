// Ported from: packages/mcp/src/error/oauth-error.ts
package mcp

import "fmt"

// MCPClientOAuthError represents an error that occurred with the MCP client
// within the OAuth flow.
type MCPClientOAuthError struct {
	Name    string
	Message string
	Cause   error
}

func NewMCPClientOAuthError(opts MCPClientOAuthErrorOptions) *MCPClientOAuthError {
	name := "MCPClientOAuthError"
	if opts.Name != "" {
		name = opts.Name
	}
	return &MCPClientOAuthError{
		Name:    name,
		Message: opts.Message,
		Cause:   opts.Cause,
	}
}

type MCPClientOAuthErrorOptions struct {
	Name    string
	Message string
	Cause   error
}

func (e *MCPClientOAuthError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Name, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

func (e *MCPClientOAuthError) Unwrap() error {
	return e.Cause
}

// ServerError represents an OAuth server_error.
type ServerError struct {
	MCPClientOAuthError
}

const ServerErrorCode = "server_error"

func NewServerError(opts MCPClientOAuthErrorOptions) *ServerError {
	opts.Name = "ServerError"
	return &ServerError{MCPClientOAuthError: *NewMCPClientOAuthError(opts)}
}

// InvalidClientError represents an OAuth invalid_client error.
type InvalidClientError struct {
	MCPClientOAuthError
}

const InvalidClientErrorCode = "invalid_client"

func NewInvalidClientError(opts MCPClientOAuthErrorOptions) *InvalidClientError {
	opts.Name = "InvalidClientError"
	return &InvalidClientError{MCPClientOAuthError: *NewMCPClientOAuthError(opts)}
}

// InvalidGrantError represents an OAuth invalid_grant error.
type InvalidGrantError struct {
	MCPClientOAuthError
}

const InvalidGrantErrorCode = "invalid_grant"

func NewInvalidGrantError(opts MCPClientOAuthErrorOptions) *InvalidGrantError {
	opts.Name = "InvalidGrantError"
	return &InvalidGrantError{MCPClientOAuthError: *NewMCPClientOAuthError(opts)}
}

// UnauthorizedClientError represents an OAuth unauthorized_client error.
type UnauthorizedClientError struct {
	MCPClientOAuthError
}

const UnauthorizedClientErrorCode = "unauthorized_client"

func NewUnauthorizedClientError(opts MCPClientOAuthErrorOptions) *UnauthorizedClientError {
	opts.Name = "UnauthorizedClientError"
	return &UnauthorizedClientError{MCPClientOAuthError: *NewMCPClientOAuthError(opts)}
}

// OAuthErrorConstructor is a function that constructs an MCPClientOAuthError subtype.
type OAuthErrorConstructor func(opts MCPClientOAuthErrorOptions) *MCPClientOAuthError

// OAuthErrors maps error codes to their constructor functions.
var OAuthErrors = map[string]func(opts MCPClientOAuthErrorOptions) error{
	ServerErrorCode:           func(opts MCPClientOAuthErrorOptions) error { return NewServerError(opts) },
	InvalidClientErrorCode:    func(opts MCPClientOAuthErrorOptions) error { return NewInvalidClientError(opts) },
	InvalidGrantErrorCode:     func(opts MCPClientOAuthErrorOptions) error { return NewInvalidGrantError(opts) },
	UnauthorizedClientErrorCode: func(opts MCPClientOAuthErrorOptions) error { return NewUnauthorizedClientError(opts) },
}
