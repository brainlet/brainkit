// Ported from: packages/mcp/src/tool/mcp-sse-transport.ts
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// SseMCPTransportConfig is the configuration for creating an SSE MCP transport.
type SseMCPTransportConfig struct {
	URL          string
	Headers      map[string]string
	AuthProvider OAuthClientProvider
}

// SseMCPTransport implements MCPTransport using Server-Sent Events (SSE)
// for the legacy SSE transport style.
type SseMCPTransport struct {
	mu                  sync.Mutex
	endpoint            *url.URL
	serverURL           *url.URL
	connected           bool
	cancel              context.CancelFunc
	ctx                 context.Context
	headers             map[string]string
	authProvider        OAuthClientProvider
	resourceMetadataURL *url.URL

	onclose   func()
	onerror   func(error)
	onmessage func(JSONRPCMessage)
}

// NewSseMCPTransport creates a new SSE MCP transport.
func NewSseMCPTransport(config SseMCPTransportConfig) *SseMCPTransport {
	u, _ := url.Parse(config.URL)
	return &SseMCPTransport{
		serverURL:    u,
		headers:      config.Headers,
		authProvider: config.AuthProvider,
	}
}

func (t *SseMCPTransport) SetOnClose(handler func())              { t.onclose = handler }
func (t *SseMCPTransport) SetOnError(handler func(error))         { t.onerror = handler }
func (t *SseMCPTransport) SetOnMessage(handler func(JSONRPCMessage)) { t.onmessage = handler }

func (t *SseMCPTransport) commonHeaders(base map[string]string) (map[string]string, error) {
	headers := make(map[string]string)
	for k, v := range t.headers {
		headers[k] = v
	}
	for k, v := range base {
		headers[k] = v
	}
	headers["mcp-protocol-version"] = LatestProtocolVersion

	if t.authProvider != nil {
		tokens, err := t.authProvider.Tokens()
		if err != nil {
			return nil, err
		}
		if tokens != nil && tokens.AccessToken != "" {
			headers["Authorization"] = "Bearer " + tokens.AccessToken
		}
	}

	return headers, nil
}

// Start initializes the SSE connection.
func (t *SseMCPTransport) Start() error {
	t.mu.Lock()
	if t.connected {
		t.mu.Unlock()
		return nil
	}
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.mu.Unlock()

	return t.establishConnection(false)
}

func (t *SseMCPTransport) establishConnection(triedAuth bool) error {
	headers, err := t.commonHeaders(map[string]string{
		"Accept": "text/event-stream",
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(t.ctx, http.MethodGet, t.serverURL.String(), nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if t.onerror != nil {
			t.onerror(err)
		}
		return err
	}

	if resp.StatusCode == http.StatusUnauthorized && t.authProvider != nil && !triedAuth {
		t.resourceMetadataURL = ExtractResourceMetadataURL(resp)
		result, err := Auth(t.authProvider, AuthOptions{
			ServerURL:           t.serverURL,
			ResourceMetadataURL: t.resourceMetadataURL,
		})
		if err != nil {
			if t.onerror != nil {
				t.onerror(err)
			}
			return err
		}
		if result != AuthResultAuthorized {
			err := NewUnauthorizedError("")
			if t.onerror != nil {
				t.onerror(err)
			}
			return err
		}
		return t.establishConnection(true)
	}

	if resp.StatusCode != http.StatusOK || resp.Body == nil {
		errorMessage := fmt.Sprintf("MCP SSE Transport Error: %d %s", resp.StatusCode, resp.Status)
		if resp.StatusCode == http.StatusMethodNotAllowed {
			errorMessage += ". This server does not support SSE transport. Try using `http` transport instead"
		}
		err := NewMCPClientError(MCPClientErrorOptions{Message: errorMessage})
		if t.onerror != nil {
			t.onerror(err)
		}
		resp.Body.Close()
		return err
	}

	// Process SSE events in background
	go t.processSSEStream(resp.Body)

	return nil
}

func (t *SseMCPTransport) processSSEStream(body io.ReadCloser) {
	defer body.Close()
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var eventType string
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line = end of event
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				t.handleSSEEvent(eventType, data)
				eventType = ""
				dataLines = nil
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
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

	t.mu.Lock()
	wasConnected := t.connected
	t.connected = false
	t.mu.Unlock()

	if wasConnected {
		err := NewMCPClientError(MCPClientErrorOptions{
			Message: "MCP SSE Transport Error: Connection closed unexpectedly",
		})
		if t.onerror != nil {
			t.onerror(err)
		}
	}
}

func (t *SseMCPTransport) handleSSEEvent(eventType, data string) {
	switch eventType {
	case "endpoint":
		endpoint, err := url.Parse(data)
		if err != nil {
			if t.onerror != nil {
				t.onerror(NewMCPClientError(MCPClientErrorOptions{
					Message: fmt.Sprintf("MCP SSE Transport Error: invalid endpoint URL: %s", data),
				}))
			}
			return
		}
		// Resolve relative URL against server URL
		resolved := t.serverURL.ResolveReference(endpoint)
		if resolved.Host != t.serverURL.Host || resolved.Scheme != t.serverURL.Scheme {
			if t.onerror != nil {
				t.onerror(NewMCPClientError(MCPClientErrorOptions{
					Message: fmt.Sprintf("MCP SSE Transport Error: Endpoint origin does not match connection origin: %s", resolved.String()),
				}))
			}
			return
		}

		t.mu.Lock()
		t.endpoint = resolved
		t.connected = true
		t.mu.Unlock()

	case "message":
		msg, err := ParseJSONRPCMessage([]byte(data))
		if err != nil {
			if t.onerror != nil {
				t.onerror(NewMCPClientError(MCPClientErrorOptions{
					Message: "MCP SSE Transport Error: Failed to parse message",
					Cause:   err,
				}))
			}
			return
		}
		if t.onmessage != nil {
			t.onmessage(msg)
		}
	}
}

// Close closes the SSE transport.
func (t *SseMCPTransport) Close() error {
	t.mu.Lock()
	t.connected = false
	if t.cancel != nil {
		t.cancel()
	}
	t.mu.Unlock()

	if t.onclose != nil {
		t.onclose()
	}
	return nil
}

// Send sends a JSON-RPC message via HTTP POST to the endpoint.
func (t *SseMCPTransport) Send(message JSONRPCMessage) error {
	t.mu.Lock()
	endpoint := t.endpoint
	connected := t.connected
	t.mu.Unlock()

	if endpoint == nil || !connected {
		return NewMCPClientError(MCPClientErrorOptions{
			Message: "MCP SSE Transport Error: Not connected",
		})
	}

	return t.sendAttempt(endpoint, message, false)
}

func (t *SseMCPTransport) sendAttempt(endpoint *url.URL, message JSONRPCMessage, triedAuth bool) error {
	headers, err := t.commonHeaders(map[string]string{
		"Content-Type": "application/json",
	})
	if err != nil {
		return err
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(t.ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if t.onerror != nil {
			t.onerror(err)
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized && t.authProvider != nil && !triedAuth {
		t.resourceMetadataURL = ExtractResourceMetadataURL(resp)
		result, authErr := Auth(t.authProvider, AuthOptions{
			ServerURL:           t.serverURL,
			ResourceMetadataURL: t.resourceMetadataURL,
		})
		if authErr != nil {
			if t.onerror != nil {
				t.onerror(authErr)
			}
			return nil
		}
		if result != AuthResultAuthorized {
			err := NewUnauthorizedError("")
			if t.onerror != nil {
				t.onerror(err)
			}
			return nil
		}
		return t.sendAttempt(endpoint, message, true)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		text, _ := io.ReadAll(resp.Body)
		sendErr := NewMCPClientError(MCPClientErrorOptions{
			Message: fmt.Sprintf("MCP SSE Transport Error: POSTing to endpoint (HTTP %d): %s", resp.StatusCode, string(text)),
		})
		if t.onerror != nil {
			t.onerror(sendErr)
		}
	}

	return nil
}

// DeserializeSSEMessage parses a JSON string into a JSONRPCMessage.
func DeserializeSSEMessage(line string) (JSONRPCMessage, error) {
	return ParseJSONRPCMessage([]byte(line))
}
