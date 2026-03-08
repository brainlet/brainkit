// Ported from: packages/mcp/src/tool/json-rpc-message.ts
package mcp

import (
	"encoding/json"
	"fmt"
)

const jsonRPCVersion = "2.0"

// JSONRPCRequest represents a JSON-RPC 2.0 request message.
type JSONRPCRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"` // string or int
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response message.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"` // string or int
	Result  interface{} `json:"result"`
}

// JSONRPCErrorDetail represents the error object in a JSON-RPC error response.
type JSONRPCErrorDetail struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error response message.
type JSONRPCError struct {
	JSONRPC string             `json:"jsonrpc"`
	ID      interface{}        `json:"id"` // string or int
	Error   JSONRPCErrorDetail `json:"error"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification message (no id).
type JSONRPCNotification struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// JSONRPCMessage is a union type that can represent any JSON-RPC 2.0 message.
// It uses a raw JSON representation internally and provides methods to determine
// the message type.
type JSONRPCMessage struct {
	raw json.RawMessage
}

// NewJSONRPCMessage creates a JSONRPCMessage from a raw JSON byte slice.
func NewJSONRPCMessage(data json.RawMessage) JSONRPCMessage {
	return JSONRPCMessage{raw: data}
}

// MarshalJSON implements the json.Marshaler interface.
func (m JSONRPCMessage) MarshalJSON() ([]byte, error) {
	return m.raw, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (m *JSONRPCMessage) UnmarshalJSON(data []byte) error {
	m.raw = make(json.RawMessage, len(data))
	copy(m.raw, data)
	return nil
}

// AsMap returns the message as a generic map.
func (m JSONRPCMessage) AsMap() (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(m.raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// HasMethod returns true if the message contains a "method" field.
func (m JSONRPCMessage) HasMethod() bool {
	mp, err := m.AsMap()
	if err != nil {
		return false
	}
	_, ok := mp["method"]
	return ok
}

// HasID returns true if the message contains an "id" field.
func (m JSONRPCMessage) HasID() bool {
	mp, err := m.AsMap()
	if err != nil {
		return false
	}
	_, ok := mp["id"]
	return ok
}

// HasResult returns true if the message contains a "result" field.
func (m JSONRPCMessage) HasResult() bool {
	mp, err := m.AsMap()
	if err != nil {
		return false
	}
	_, ok := mp["result"]
	return ok
}

// HasError returns true if the message contains an "error" field.
func (m JSONRPCMessage) HasError() bool {
	mp, err := m.AsMap()
	if err != nil {
		return false
	}
	_, ok := mp["error"]
	return ok
}

// AsRequest parses the message as a JSONRPCRequest.
func (m JSONRPCMessage) AsRequest() (*JSONRPCRequest, error) {
	var req JSONRPCRequest
	if err := json.Unmarshal(m.raw, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

// AsResponse parses the message as a JSONRPCResponse.
func (m JSONRPCMessage) AsResponse() (*JSONRPCResponse, error) {
	var resp JSONRPCResponse
	if err := json.Unmarshal(m.raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AsError parses the message as a JSONRPCError.
func (m JSONRPCMessage) AsError() (*JSONRPCError, error) {
	var errResp JSONRPCError
	if err := json.Unmarshal(m.raw, &errResp); err != nil {
		return nil, err
	}
	return &errResp, nil
}

// AsNotification parses the message as a JSONRPCNotification.
func (m JSONRPCMessage) AsNotification() (*JSONRPCNotification, error) {
	var notif JSONRPCNotification
	if err := json.Unmarshal(m.raw, &notif); err != nil {
		return nil, err
	}
	return &notif, nil
}

// MakeRequestMessage creates a JSONRPCMessage from a JSONRPCRequest.
func MakeRequestMessage(req JSONRPCRequest) (JSONRPCMessage, error) {
	req.JSONRPC = jsonRPCVersion
	data, err := json.Marshal(req)
	if err != nil {
		return JSONRPCMessage{}, err
	}
	return JSONRPCMessage{raw: data}, nil
}

// MakeResponseMessage creates a JSONRPCMessage from a JSONRPCResponse.
func MakeResponseMessage(resp JSONRPCResponse) (JSONRPCMessage, error) {
	resp.JSONRPC = jsonRPCVersion
	data, err := json.Marshal(resp)
	if err != nil {
		return JSONRPCMessage{}, err
	}
	return JSONRPCMessage{raw: data}, nil
}

// MakeErrorMessage creates a JSONRPCMessage from a JSONRPCError.
func MakeErrorMessage(errResp JSONRPCError) (JSONRPCMessage, error) {
	errResp.JSONRPC = jsonRPCVersion
	data, err := json.Marshal(errResp)
	if err != nil {
		return JSONRPCMessage{}, err
	}
	return JSONRPCMessage{raw: data}, nil
}

// MakeNotificationMessage creates a JSONRPCMessage from a JSONRPCNotification.
func MakeNotificationMessage(notif JSONRPCNotification) (JSONRPCMessage, error) {
	notif.JSONRPC = jsonRPCVersion
	data, err := json.Marshal(notif)
	if err != nil {
		return JSONRPCMessage{}, err
	}
	return JSONRPCMessage{raw: data}, nil
}

// ParseJSONRPCMessage parses a raw JSON byte slice into a JSONRPCMessage,
// performing basic validation.
func ParseJSONRPCMessage(data []byte) (JSONRPCMessage, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return JSONRPCMessage{}, fmt.Errorf("failed to parse JSON-RPC message: %w", err)
	}
	v, ok := raw["jsonrpc"]
	if !ok {
		return JSONRPCMessage{}, fmt.Errorf("missing jsonrpc field in message")
	}
	if vs, ok := v.(string); !ok || vs != jsonRPCVersion {
		return JSONRPCMessage{}, fmt.Errorf("invalid jsonrpc version: %v", v)
	}
	return JSONRPCMessage{raw: json.RawMessage(data)}, nil
}
