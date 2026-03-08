// Ported from: packages/mcp/src/tool/mcp-http-transport.ts
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// HttpMCPTransportConfig is the configuration for creating an HTTP MCP transport.
type HttpMCPTransportConfig struct {
	URL          string
	Headers      map[string]string
	AuthProvider OAuthClientProvider
}

// reconnectionOptions holds reconnection configuration for inbound SSE.
type reconnectionOptions struct {
	initialReconnectionDelay    time.Duration
	maxReconnectionDelay        time.Duration
	reconnectionDelayGrowFactor float64
	maxRetries                  int
}

// HttpMCPTransport implements MCPTransport using the Streamable HTTP style.
//
// Client transport for Streamable HTTP: this implements the MCP Streamable HTTP
// transport specification. It will connect to a server using HTTP POST for
// sending messages and HTTP GET with Server-Sent Events for receiving messages.
type HttpMCPTransport struct {
	mu                       sync.Mutex
	serverURL                *url.URL
	headers                  map[string]string
	authProvider             OAuthClientProvider
	resourceMetadataURL      *url.URL
	sessionID                string
	ctx                      context.Context
	cancel                   context.CancelFunc
	started                  bool
	inboundSSECancel         context.CancelFunc
	lastInboundEventID       string
	inboundReconnectAttempts int
	reconnOpts               reconnectionOptions

	onclose   func()
	onerror   func(error)
	onmessage func(JSONRPCMessage)
}

// NewHttpMCPTransport creates a new HTTP MCP transport.
func NewHttpMCPTransport(config HttpMCPTransportConfig) *HttpMCPTransport {
	u, _ := url.Parse(config.URL)
	return &HttpMCPTransport{
		serverURL:    u,
		headers:      config.Headers,
		authProvider: config.AuthProvider,
		reconnOpts: reconnectionOptions{
			initialReconnectionDelay:    1 * time.Second,
			maxReconnectionDelay:        30 * time.Second,
			reconnectionDelayGrowFactor: 1.5,
			maxRetries:                  2,
		},
	}
}

func (t *HttpMCPTransport) SetOnClose(handler func())              { t.onclose = handler }
func (t *HttpMCPTransport) SetOnError(handler func(error))         { t.onerror = handler }
func (t *HttpMCPTransport) SetOnMessage(handler func(JSONRPCMessage)) { t.onmessage = handler }

func (t *HttpMCPTransport) commonHeaders(base map[string]string) (map[string]string, error) {
	headers := make(map[string]string)
	for k, v := range t.headers {
		headers[k] = v
	}
	for k, v := range base {
		headers[k] = v
	}
	headers["mcp-protocol-version"] = LatestProtocolVersion

	t.mu.Lock()
	sessionID := t.sessionID
	t.mu.Unlock()

	if sessionID != "" {
		headers["mcp-session-id"] = sessionID
	}

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

// Start initializes the transport.
func (t *HttpMCPTransport) Start() error {
	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return NewMCPClientError(MCPClientErrorOptions{
			Message: "MCP HTTP Transport Error: Transport already started. Note: client.connect() calls start() automatically.",
		})
	}
	t.ctx, t.cancel = context.WithCancel(context.Background())
	t.started = true
	t.mu.Unlock()

	// Best-effort open inbound SSE
	go t.openInboundSSE(false, "")

	return nil
}

// Close closes the transport.
func (t *HttpMCPTransport) Close() error {
	t.mu.Lock()
	if t.inboundSSECancel != nil {
		t.inboundSSECancel()
	}
	sessionID := t.sessionID
	cancel := t.cancel
	ctx := t.ctx
	serverURL := t.serverURL
	t.mu.Unlock()

	// Try to send DELETE to terminate the session
	if sessionID != "" && ctx != nil {
		headers, err := t.commonHeaders(map[string]string{})
		if err == nil {
			req, err := http.NewRequestWithContext(ctx, http.MethodDelete, serverURL.String(), nil)
			if err == nil {
				for k, v := range headers {
					req.Header.Set(k, v)
				}
				resp, err := http.DefaultClient.Do(req)
				if err == nil {
					resp.Body.Close()
				}
			}
		}
	}

	if cancel != nil {
		cancel()
	}

	if t.onclose != nil {
		t.onclose()
	}

	return nil
}

// Send sends a JSON-RPC message via HTTP POST.
func (t *HttpMCPTransport) Send(message JSONRPCMessage) error {
	return t.sendAttempt(message, false)
}

func (t *HttpMCPTransport) sendAttempt(message JSONRPCMessage, triedAuth bool) error {
	headers, err := t.commonHeaders(map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json, text/event-stream",
	})
	if err != nil {
		return err
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(t.ctx, http.MethodPost, t.serverURL.String(), bytes.NewReader(body))
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

	// Capture session ID
	if sid := resp.Header.Get("mcp-session-id"); sid != "" {
		t.mu.Lock()
		t.sessionID = sid
		t.mu.Unlock()
	}

	if resp.StatusCode == http.StatusUnauthorized && t.authProvider != nil && !triedAuth {
		resp.Body.Close()
		t.resourceMetadataURL = ExtractResourceMetadataURL(resp)
		result, authErr := Auth(t.authProvider, AuthOptions{
			ServerURL:           t.serverURL,
			ResourceMetadataURL: t.resourceMetadataURL,
		})
		if authErr != nil {
			if t.onerror != nil {
				t.onerror(authErr)
			}
			return authErr
		}
		if result != AuthResultAuthorized {
			unauthErr := NewUnauthorizedError("")
			if t.onerror != nil {
				t.onerror(unauthErr)
			}
			return unauthErr
		}
		return t.sendAttempt(message, true)
	}

	// 202 Accepted: message acknowledged, no response body
	if resp.StatusCode == http.StatusAccepted {
		resp.Body.Close()
		// Try to (re)start inbound SSE if not already open
		t.mu.Lock()
		hasInbound := t.inboundSSECancel != nil
		t.mu.Unlock()
		if !hasInbound {
			go t.openInboundSSE(false, "")
		}
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		text, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		errorMessage := fmt.Sprintf("MCP HTTP Transport Error: POSTing to endpoint (HTTP %d): %s", resp.StatusCode, string(text))
		if resp.StatusCode == http.StatusNotFound {
			errorMessage += ". This server does not support HTTP transport. Try using `sse` transport instead"
		}
		sendErr := NewMCPClientError(MCPClientErrorOptions{Message: errorMessage})
		if t.onerror != nil {
			t.onerror(sendErr)
		}
		return sendErr
	}

	// Notifications (messages without 'id') don't expect a JSON-RPC response
	isNotification := !message.HasID()
	if isNotification {
		resp.Body.Close()
		return nil
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		// Could be a single message or an array
		var raw json.RawMessage
		if err := json.Unmarshal(respBody, &raw); err != nil {
			return err
		}

		// Check if it's an array
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			var messages []json.RawMessage
			if err := json.Unmarshal(trimmed, &messages); err != nil {
				return err
			}
			for _, m := range messages {
				msg, err := ParseJSONRPCMessage(m)
				if err != nil {
					if t.onerror != nil {
						t.onerror(err)
					}
					continue
				}
				if t.onmessage != nil {
					t.onmessage(msg)
				}
			}
		} else {
			msg, err := ParseJSONRPCMessage(trimmed)
			if err != nil {
				return err
			}
			if t.onmessage != nil {
				t.onmessage(msg)
			}
		}
		return nil
	}

	if strings.Contains(contentType, "text/event-stream") {
		if resp.Body == nil {
			sendErr := NewMCPClientError(MCPClientErrorOptions{
				Message: "MCP HTTP Transport Error: text/event-stream response without body",
			})
			if t.onerror != nil {
				t.onerror(sendErr)
			}
			return sendErr
		}

		go t.processSSEResponse(resp.Body)
		return nil
	}

	resp.Body.Close()
	sendErr := NewMCPClientError(MCPClientErrorOptions{
		Message: fmt.Sprintf("MCP HTTP Transport Error: Unexpected content type: %s", contentType),
	})
	if t.onerror != nil {
		t.onerror(sendErr)
	}
	return sendErr
}

func (t *HttpMCPTransport) processSSEResponse(body io.ReadCloser) {
	defer body.Close()
	t.processSSEEvents(body, false)
}

func (t *HttpMCPTransport) getNextReconnectionDelay(attempt int) time.Duration {
	delay := float64(t.reconnOpts.initialReconnectionDelay) *
		math.Pow(t.reconnOpts.reconnectionDelayGrowFactor, float64(attempt))
	if time.Duration(delay) > t.reconnOpts.maxReconnectionDelay {
		return t.reconnOpts.maxReconnectionDelay
	}
	return time.Duration(delay)
}

func (t *HttpMCPTransport) scheduleInboundSSEReconnection() {
	if t.reconnOpts.maxRetries > 0 && t.inboundReconnectAttempts >= t.reconnOpts.maxRetries {
		if t.onerror != nil {
			t.onerror(NewMCPClientError(MCPClientErrorOptions{
				Message: fmt.Sprintf("MCP HTTP Transport Error: Maximum reconnection attempts (%d) exceeded.", t.reconnOpts.maxRetries),
			}))
		}
		return
	}

	delay := t.getNextReconnectionDelay(t.inboundReconnectAttempts)
	t.inboundReconnectAttempts++
	time.AfterFunc(delay, func() {
		if t.ctx.Err() != nil {
			return
		}
		t.openInboundSSE(false, t.lastInboundEventID)
	})
}

func (t *HttpMCPTransport) openInboundSSE(triedAuth bool, resumeToken string) {
	headers, err := t.commonHeaders(map[string]string{
		"Accept": "text/event-stream",
	})
	if err != nil {
		if t.onerror != nil {
			t.onerror(err)
		}
		return
	}
	if resumeToken != "" {
		headers["last-event-id"] = resumeToken
	}

	sseCtx, sseCancel := context.WithCancel(t.ctx)
	t.mu.Lock()
	t.inboundSSECancel = sseCancel
	t.mu.Unlock()

	req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, t.serverURL.String(), nil)
	if err != nil {
		sseCancel()
		if t.onerror != nil {
			t.onerror(err)
		}
		return
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		sseCancel()
		if t.ctx.Err() != nil {
			return // parent context cancelled
		}
		if t.onerror != nil {
			t.onerror(err)
		}
		t.scheduleInboundSSEReconnection()
		return
	}

	if sid := resp.Header.Get("mcp-session-id"); sid != "" {
		t.mu.Lock()
		t.sessionID = sid
		t.mu.Unlock()
	}

	if resp.StatusCode == http.StatusUnauthorized && t.authProvider != nil && !triedAuth {
		resp.Body.Close()
		sseCancel()
		t.resourceMetadataURL = ExtractResourceMetadataURL(resp)
		result, authErr := Auth(t.authProvider, AuthOptions{
			ServerURL:           t.serverURL,
			ResourceMetadataURL: t.resourceMetadataURL,
		})
		if authErr != nil {
			if t.onerror != nil {
				t.onerror(authErr)
			}
			return
		}
		if result != AuthResultAuthorized {
			if t.onerror != nil {
				t.onerror(NewUnauthorizedError(""))
			}
			return
		}
		t.openInboundSSE(true, resumeToken)
		return
	}

	if resp.StatusCode == http.StatusMethodNotAllowed {
		resp.Body.Close()
		sseCancel()
		return
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || resp.Body == nil {
		if resp.Body != nil {
			resp.Body.Close()
		}
		sseCancel()
		if t.onerror != nil {
			t.onerror(NewMCPClientError(MCPClientErrorOptions{
				Message: fmt.Sprintf("MCP HTTP Transport Error: GET SSE failed: %d %s", resp.StatusCode, resp.Status),
			}))
		}
		return
	}

	t.inboundReconnectAttempts = 0

	go func() {
		defer resp.Body.Close()
		defer sseCancel()
		t.processSSEEvents(resp.Body, true)

		if t.ctx.Err() == nil {
			t.scheduleInboundSSEReconnection()
		}
	}()
}

// processSSEEvents reads and processes SSE events from a reader.
// If trackEventID is true, it updates lastInboundEventID.
func (t *HttpMCPTransport) processSSEEvents(r io.Reader, trackEventID bool) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	var eventType string
	var dataLines []string
	var eventID string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")

				if trackEventID && eventID != "" {
					t.lastInboundEventID = eventID
				}

				if eventType == "message" {
					msg, err := ParseJSONRPCMessage([]byte(data))
					if err != nil {
						if t.onerror != nil {
							t.onerror(NewMCPClientError(MCPClientErrorOptions{
								Message: "MCP HTTP Transport Error: Failed to parse message",
								Cause:   err,
							}))
						}
					} else if t.onmessage != nil {
						t.onmessage(msg)
					}
				}

				eventType = ""
				dataLines = nil
				eventID = ""
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		} else if strings.HasPrefix(line, "id:") {
			eventID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		}
	}

	if err := scanner.Err(); err != nil {
		if t.ctx.Err() != nil {
			return
		}
		if t.onerror != nil {
			t.onerror(err)
		}
	}
}
