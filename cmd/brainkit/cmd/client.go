package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// busClient talks to a running brainkit server's gateway over
// POST /api/bus + POST /api/stream. Replaces the pidfile-based
// control HTTP that lived under cmd/brainkit/config/.
type busClient struct {
	endpoint string
	http     *http.Client
}

// newBusClient returns a busClient pointed at the resolved endpoint.
// resolveEndpoint picks, in priority order: the explicit --endpoint
// flag, $BRAINKIT_ENDPOINT, gateway.listen from ./brainkit.yaml, and
// finally the 127.0.0.1:8080 default — so running `brainkit start`
// with a non-default `gateway.listen:` and then `brainkit deploy`
// from the same directory targets the right port without a flag.
func newBusClient(endpoint string) *busClient {
	return &busClient{
		endpoint: resolveEndpoint(endpoint),
		http:     &http.Client{Timeout: 0}, // caller owns the deadline
	}
}

// call sends a bus request and returns the raw JSON payload. The
// gateway returns `{"payload": ...}` on success; the payload is
// unwrapped if it's envelope-shaped (`{ok:true, data:...}`).
func (c *busClient) call(ctx context.Context, topic string, payload json.RawMessage) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{
		"topic":   topic,
		"payload": payload,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/api/bus", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post /api/bus: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bus error %d: %s", resp.StatusCode, string(raw))
	}

	var outer struct {
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(raw, &outer); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, string(raw))
	}
	return unwrapEnvelope(outer.Payload), nil
}

// stream sends a bus request and yields each NDJSON event to
// onEvent until the terminal event (`done=true`) or ctx expires.
// Returns the terminal event payload.
func (c *busClient) stream(ctx context.Context, topic string, payload json.RawMessage, onEvent func(raw json.RawMessage)) (json.RawMessage, error) {
	body, err := json.Marshal(map[string]any{
		"topic":   topic,
		"payload": payload,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/api/stream", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post /api/stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stream error %d: %s", resp.StatusCode, string(raw))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 4096), 1<<20)
	var terminal json.RawMessage
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var ev struct {
			Payload json.RawMessage `json:"payload"`
			Done    bool            `json:"done"`
		}
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, fmt.Errorf("decode ndjson: %w", err)
		}
		if onEvent != nil {
			onEvent(ev.Payload)
		}
		if ev.Done {
			terminal = ev.Payload
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan ndjson: %w", err)
	}
	return unwrapEnvelope(terminal), nil
}

// unwrapEnvelope peels `{ok:true, data:...}` if present. Non-
// envelope payloads pass through unchanged.
func unwrapEnvelope(payload json.RawMessage) json.RawMessage {
	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(payload, &env) == nil && env.OK && env.Data != nil {
		return env.Data
	}
	return payload
}

// withTimeout returns ctx with the cmd's --timeout flag applied.
func withTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}
