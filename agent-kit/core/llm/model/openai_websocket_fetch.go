// Ported from: packages/core/src/llm/model/openai-websocket-fetch.ts
package model

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// ---------------------------------------------------------------------------
// CreateOpenAIWebSocketFetchOptions
// ---------------------------------------------------------------------------

// CreateOpenAIWebSocketFetchOptions holds options for creating an OpenAI
// WebSocket fetch function.
type CreateOpenAIWebSocketFetchOptions struct {
	// URL is the WebSocket endpoint URL.
	// Default: "wss://api.openai.com/v1/responses"
	URL string `json:"url,omitempty"`
	// Headers contains additional headers for the WebSocket connection.
	// Authorization and OpenAI-Beta are managed internally.
	Headers map[string]string `json:"headers,omitempty"`
}

// ---------------------------------------------------------------------------
// OpenAIWebSocketFetch
// ---------------------------------------------------------------------------

// OpenAIWebSocketFetch is a fetch-compatible function that routes OpenAI
// Responses API streaming requests through a persistent WebSocket connection
// instead of HTTP.
type OpenAIWebSocketFetch struct {
	mu            sync.Mutex
	wsURL         string
	baseHeaders   map[string]string
	conn          *websocket.Conn
	connecting    chan struct{} // nil when not connecting
	connectionKey string
	busy          bool
}

// NewOpenAIWebSocketFetch creates a new OpenAIWebSocketFetch.
func NewOpenAIWebSocketFetch(opts *CreateOpenAIWebSocketFetchOptions) *OpenAIWebSocketFetch {
	wsURL := "wss://api.openai.com/v1/responses"
	var headers map[string]string

	if opts != nil {
		if opts.URL != "" {
			wsURL = opts.URL
		}
		headers = opts.Headers
	}

	return &OpenAIWebSocketFetch{
		wsURL:       wsURL,
		baseHeaders: headers,
	}
}

// Close closes the underlying WebSocket connection.
func (f *OpenAIWebSocketFetch) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.conn != nil {
		f.conn.Close()
		f.conn = nil
	}
	f.connectionKey = ""
	f.connecting = nil
}

// Do performs a request. If the request is a POST to a /responses endpoint
// with streaming enabled, it routes through the WebSocket connection.
// Otherwise, it falls back to standard HTTP.
func (f *OpenAIWebSocketFetch) Do(ctx context.Context, method, url string, body map[string]any, headers map[string]string) (*WebSocketFetchResponse, error) {
	// Only intercept POST to /responses with stream=true
	if method != "POST" || !strings.HasSuffix(url, "/responses") {
		return f.doHTTP(ctx, method, url, body, headers)
	}

	stream, _ := body["stream"].(bool)
	if !stream {
		return f.doHTTP(ctx, method, url, body, headers)
	}

	f.mu.Lock()
	if f.busy {
		f.mu.Unlock()
		// Fall back to HTTP for concurrent requests
		return f.doHTTP(ctx, method, url, body, headers)
	}
	f.busy = true
	f.mu.Unlock()

	normalized := normalizeHeaderMap(headers)
	authorization := normalized["authorization"]

	conn, err := f.getConnection(authorization, normalized)
	if err != nil {
		f.mu.Lock()
		f.busy = false
		f.mu.Unlock()
		return nil, err
	}

	// Remove "stream" from body for WebSocket
	requestBody := make(map[string]any, len(body))
	for k, v := range body {
		if k != "stream" {
			requestBody[k] = v
		}
	}
	requestBody["type"] = "response.create"

	msgBytes, err := json.Marshal(requestBody)
	if err != nil {
		f.mu.Lock()
		f.busy = false
		f.mu.Unlock()
		return nil, fmt.Errorf("failed to marshal WebSocket request: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		f.mu.Lock()
		f.busy = false
		f.mu.Unlock()
		return nil, fmt.Errorf("failed to send WebSocket message: %w", err)
	}

	// Return a response that reads from the WebSocket
	return &WebSocketFetchResponse{
		StatusCode: 200,
		Headers:    map[string]string{"content-type": "text/event-stream"},
		conn:       conn,
		fetch:      f,
		ctx:        ctx,
	}, nil
}

// WebSocketFetchResponse wraps a WebSocket connection as an HTTP-like response.
type WebSocketFetchResponse struct {
	StatusCode int
	Headers    map[string]string
	conn       *websocket.Conn
	fetch      *OpenAIWebSocketFetch
	ctx        context.Context
}

// ReadEvents reads SSE events from the WebSocket connection.
// The returned channel is closed when the stream completes or an error occurs.
func (r *WebSocketFetchResponse) ReadEvents() <-chan string {
	ch := make(chan string)

	go func() {
		defer close(ch)
		defer func() {
			r.fetch.mu.Lock()
			r.fetch.busy = false
			r.fetch.mu.Unlock()
		}()

		for {
			select {
			case <-r.ctx.Done():
				return
			default:
			}

			_, message, err := r.conn.ReadMessage()
			if err != nil {
				return
			}

			text := string(message)
			ch <- fmt.Sprintf("data: %s\n\n", text)

			// Check if this is a terminal event
			var event struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(message, &event) == nil {
				if event.Type == "response.completed" || event.Type == "error" {
					ch <- "data: [DONE]\n\n"
					return
				}
			}
		}
	}()

	return ch
}

func (f *OpenAIWebSocketFetch) getConnection(authorization string, headers map[string]string) (*websocket.Conn, error) {
	normalizedHeaders := mergeHeaders(normalizeHeaderMap(f.baseHeaders), headers)
	delete(normalizedHeaders, "authorization")
	delete(normalizedHeaders, "openai-beta")
	nextKey := buildConnectionKey(authorization, normalizedHeaders)

	f.mu.Lock()
	defer f.mu.Unlock()

	// Reuse existing connection if same key
	if f.conn != nil && f.connectionKey == nextKey {
		return f.conn, nil
	}

	// Close existing connection if different key
	if f.conn != nil && f.connectionKey != nextKey {
		f.conn.Close()
		f.conn = nil
		f.connectionKey = ""
	}

	// Create new connection
	connHeaders := make(http.Header)
	for k, v := range normalizedHeaders {
		connHeaders.Set(k, v)
	}
	connHeaders.Set("Authorization", authorization)
	connHeaders.Set("OpenAI-Beta", "responses_websockets=2026-02-06")

	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(f.wsURL, connHeaders)
	if err != nil {
		return nil, fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	f.conn = conn
	f.connectionKey = nextKey

	return conn, nil
}

func (f *OpenAIWebSocketFetch) doHTTP(ctx context.Context, method, url string, body map[string]any, headers map[string]string) (*WebSocketFetchResponse, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = strings.NewReader(string(bodyBytes))
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	return &WebSocketFetchResponse{
		StatusCode: resp.StatusCode,
		Headers:    map[string]string{"content-type": resp.Header.Get("Content-Type")},
		conn:       nil,
		fetch:      f,
		ctx:        ctx,
	}, fmt.Errorf("HTTP response: %d %s", resp.StatusCode, string(respBody))
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

func buildConnectionKey(authorization string, headers map[string]string) string {
	// Sort headers for consistent key
	type kv struct{ K, V string }
	pairs := make([]kv, 0, len(headers))
	for k, v := range headers {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].K < pairs[j].K })

	data := map[string]any{
		"authorization": authorization,
		"headers":       pairs,
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func normalizeHeaderMap(headers map[string]string) map[string]string {
	if headers == nil {
		return make(map[string]string)
	}
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		result[strings.ToLower(k)] = v
	}
	return result
}

func mergeHeaders(base, overlay map[string]string) map[string]string {
	result := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overlay {
		result[k] = v
	}
	return result
}
