//go:build e2e

package harness

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type eventCollector struct {
	mu      sync.Mutex
	events  []HarnessEvent
	waiters map[HarnessEventType][]chan HarnessEvent
}

func newEventCollector() *eventCollector {
	return &eventCollector{
		waiters: make(map[HarnessEventType][]chan HarnessEvent),
	}
}

func (c *eventCollector) handler(event HarnessEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)

	if chs, ok := c.waiters[event.Type]; ok {
		for _, ch := range chs {
			select {
			case ch <- event:
			default:
			}
		}
		delete(c.waiters, event.Type)
	}
}

func (c *eventCollector) WaitFor(typ HarnessEventType, timeout time.Duration) (HarnessEvent, error) {
	c.mu.Lock()
	for _, e := range c.events {
		if e.Type == typ {
			c.mu.Unlock()
			return e, nil
		}
	}
	ch := make(chan HarnessEvent, 1)
	c.waiters[typ] = append(c.waiters[typ], ch)
	c.mu.Unlock()

	select {
	case e := <-ch:
		return e, nil
	case <-time.After(timeout):
		return HarnessEvent{}, fmt.Errorf("timeout waiting for %s after %v", typ, timeout)
	}
}

func (c *eventCollector) AllOfType(typ HarnessEventType) []HarnessEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	var result []HarnessEvent
	for _, e := range c.events {
		if e.Type == typ {
			result = append(result, e)
		}
	}
	return result
}

func (c *eventCollector) Count(typ HarnessEventType) int {
	return len(c.AllOfType(typ))
}

func (c *eventCollector) Has(typ HarnessEventType) bool {
	return c.Count(typ) > 0
}

func (c *eventCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = nil
}

func sendWithTimeout(t *testing.T, h *Harness, content string, timeout time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- h.SendMessage(content) }()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		h.Abort()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		t.Fatalf("SendMessage timed out after %v (aborted)", timeout)
		return nil
	}
}

func assertJSON(t *testing.T, method, url string, body string, expectedStatus int, expectedJSON string) {
	t.Helper()
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("HTTP %s %s: status = %d, want %d (body: %s)", method, url, resp.StatusCode, expectedStatus, string(respBody))
		return
	}

	if expectedJSON != "" {
		respBody, _ := io.ReadAll(resp.Body)
		var expected, actual any
		json.Unmarshal([]byte(expectedJSON), &expected)
		json.Unmarshal(respBody, &actual)
		expectedB, _ := json.Marshal(expected)
		actualB, _ := json.Marshal(actual)
		if string(expectedB) != string(actualB) {
			t.Errorf("HTTP %s %s: body = %s, want %s", method, url, string(respBody), expectedJSON)
		}
	}
}

func assertStatus(t *testing.T, method, url, body string, expectedStatus int) {
	t.Helper()
	assertJSON(t, method, url, body, expectedStatus, "")
}

func waitForPort(t *testing.T, port int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("port %d not available after %v", port, timeout)
}

func assertHTTPReachable(t *testing.T, method, url string, body string, expectedStatus int) {
	t.Helper()
	var req *http.Request
	var err error
	if body != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("create request: %v", err)
		}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	if expectedStatus > 0 && resp.StatusCode != expectedStatus {
		t.Errorf("HTTP %s %s: status = %d, want %d", method, url, resp.StatusCode, expectedStatus)
	}
}

func assertMathResult(t *testing.T, url string, a, b, expected float64) {
	t.Helper()
	body := fmt.Sprintf(`{"a":%v,"b":%v}`, a, b)
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP POST %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Errorf("HTTP POST %s {a:%v,b:%v}: status = %d (body: %s)", url, a, b, resp.StatusCode, string(respBody))
		return
	}

	var result map[string]any
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		t.Errorf("HTTP POST %s: invalid JSON: %s", url, string(respBody))
		return
	}

	var got float64
	var found bool
	for _, key := range []string{"result", "answer", "sum", "difference", "quotient", "value"} {
		if v, ok := result[key]; ok {
			if n, ok := v.(float64); ok {
				got = n
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("HTTP POST %s {a:%v,b:%v}: no result field in %s", url, a, b, string(respBody))
		return
	}
	if got != expected {
		t.Errorf("HTTP POST %s {a:%v,b:%v}: result = %v, want %v", url, a, b, got, expected)
	}
}
