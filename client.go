package brainkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// BusClient sends bus commands to a running brainkit instance over HTTP.
// The instance exposes a control API on a local port.
type BusClient struct {
	baseURL    string
	httpClient *http.Client
}

// busRequestPayload is the JSON body for POST /api/bus.
type busRequestPayload struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

// busResponsePayload is the JSON response from POST /api/bus.
type busResponsePayload struct {
	Payload json.RawMessage `json:"payload"`
	Error   string          `json:"error,omitempty"`
}

// NewClient creates a BusClient that connects to a running instance over HTTP.
func NewClient(baseURL string) *BusClient {
	return &BusClient{
		baseURL: baseURL,
		httpClient: &http.Client{},
	}
}

// Request sends a typed bus command and returns the raw response payload.
func (c *BusClient) Request(ctx context.Context, topic string, payload json.RawMessage) (json.RawMessage, error) {
	body, err := json.Marshal(busRequestPayload{Topic: topic, Payload: payload})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/bus", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to brainkit instance at %s: %w\nHint: is `brainkit start` running?", c.baseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp busResponsePayload
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, fmt.Errorf("%s", errResp.Error)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result busResponsePayload
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Response might be raw payload (not wrapped)
		return respBody, nil
	}
	if result.Error != "" {
		return nil, fmt.Errorf("%s", result.Error)
	}
	return result.Payload, nil
}

// StreamEvent is one event from the NDJSON stream.
type StreamEvent struct {
	Payload json.RawMessage `json:"payload"`
	Done    bool            `json:"done"`
	Error   string          `json:"error,omitempty"`
}

// Stream sends a bus command and returns a channel of streaming events.
// The channel closes after the terminal event (done=true) or on error.
func (c *BusClient) Stream(ctx context.Context, topic string, payload json.RawMessage) (<-chan StreamEvent, error) {
	body, err := json.Marshal(busRequestPayload{Topic: topic, Payload: payload})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/stream", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to brainkit instance at %s\nHint: is `brainkit start` running?", c.baseURL)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamEvent, 100)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		dec := json.NewDecoder(resp.Body)
		for dec.More() {
			var evt StreamEvent
			if err := dec.Decode(&evt); err != nil {
				ch <- StreamEvent{Error: err.Error(), Done: true}
				return
			}
			ch <- evt
			if evt.Done {
				return
			}
		}
	}()
	return ch, nil
}

// Close is a no-op for HTTP client (no persistent connection).
func (c *BusClient) Close() error { return nil }
