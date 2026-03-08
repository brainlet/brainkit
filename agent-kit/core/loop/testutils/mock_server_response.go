// Ported from: packages/core/src/loop/test-utils/mock-server-response.ts
package testutils

import (
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// MockServerResponse
// ---------------------------------------------------------------------------

// MockServerResponse simulates an HTTP ServerResponse for testing stream
// writing. It mirrors the TS MockServerResponse class that implements a
// subset of Node's http.ServerResponse.
type MockServerResponse struct {
	mu             sync.Mutex
	WrittenChunks  [][]byte
	Headers        map[string]string
	StatusCode     int
	StatusMessage  string
	Ended          bool
	eventListeners map[string][]func(args ...any)
	endCh          chan struct{} // signals when End() is called
}

// NewMockServerResponse creates a new MockServerResponse.
func NewMockServerResponse() *MockServerResponse {
	return &MockServerResponse{
		Headers:        make(map[string]string),
		eventListeners: make(map[string][]func(args ...any)),
		endCh:          make(chan struct{}),
	}
}

// Write appends a chunk to the response.
func (m *MockServerResponse) Write(chunk []byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(chunk))
	copy(cp, chunk)
	m.WrittenChunks = append(m.WrittenChunks, cp)
	return true
}

// End optionally writes a final chunk and marks the response as ended.
func (m *MockServerResponse) End(chunk ...[]byte) {
	m.mu.Lock()
	if len(chunk) > 0 && chunk[0] != nil {
		cp := make([]byte, len(chunk[0]))
		copy(cp, chunk[0])
		m.WrittenChunks = append(m.WrittenChunks, cp)
	}
	m.Ended = true
	m.mu.Unlock()

	m.Emit("close")

	// Unblock WaitForEnd().
	select {
	case <-m.endCh:
	default:
		close(m.endCh)
	}
}

// WriteHead sets status code, message, and headers.
func (m *MockServerResponse) WriteHead(statusCode int, statusMessage string, headers map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StatusCode = statusCode
	m.StatusMessage = statusMessage
	for k, v := range headers {
		m.Headers[k] = v
	}
}

// Once registers a listener that fires once for the given event.
func (m *MockServerResponse) Once(event string, listener func(args ...any)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var wrapper func(args ...any)
	wrapper = func(args ...any) {
		listener(args...)
		m.Off(event, wrapper)
	}
	m.eventListeners[event] = append(m.eventListeners[event], wrapper)
}

// On registers a listener for the given event.
func (m *MockServerResponse) On(event string, listener func(args ...any)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventListeners[event] = append(m.eventListeners[event], listener)
}

// Off removes a listener for the given event.
func (m *MockServerResponse) Off(event string, listener func(args ...any)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	listeners := m.eventListeners[event]
	for i, l := range listeners {
		// Compare function pointers (works for closures captured in same variable).
		if &l == &listener {
			m.eventListeners[event] = append(listeners[:i], listeners[i+1:]...)
			return
		}
	}
}

// Emit fires all listeners for the given event.
func (m *MockServerResponse) Emit(event string, args ...any) bool {
	m.mu.Lock()
	listeners := make([]func(args ...any), len(m.eventListeners[event]))
	copy(listeners, m.eventListeners[event])
	m.mu.Unlock()

	for _, l := range listeners {
		l(args...)
	}
	return len(listeners) > 0
}

// Body returns all written chunks joined into a single string.
func (m *MockServerResponse) Body() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var sb strings.Builder
	for _, chunk := range m.WrittenChunks {
		sb.Write(chunk)
	}
	return sb.String()
}

// GetDecodedChunks returns each written chunk decoded as a string.
func (m *MockServerResponse) GetDecodedChunks() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.WrittenChunks))
	for i, chunk := range m.WrittenChunks {
		result[i] = string(chunk)
	}
	return result
}

// WaitForEnd blocks until End() has been called.
func (m *MockServerResponse) WaitForEnd() {
	<-m.endCh
}

// CreateMockServerResponse creates a new MockServerResponse.
// This mirrors the TS createMockServerResponse() factory function.
func CreateMockServerResponse() *MockServerResponse {
	return NewMockServerResponse()
}
