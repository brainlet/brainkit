// Ported from: packages/ai/src/test/mock-server-response.ts
package testutil

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// MockServerResponse is a test double for http.ResponseWriter that records
// all written data for assertions. It corresponds to the TS MockServerResponse
// which extends Node's ServerResponse.
type MockServerResponse struct {
	mu            sync.Mutex
	WrittenChunks [][]byte
	HeaderMap     http.Header
	StatusCode    int
	StatusMessage string
	Ended         bool

	endCh chan struct{}
}

// NewMockServerResponse creates a new MockServerResponse.
func NewMockServerResponse() *MockServerResponse {
	return &MockServerResponse{
		WrittenChunks: [][]byte{},
		HeaderMap:     http.Header{},
		endCh:         make(chan struct{}),
	}
}

// Header implements http.ResponseWriter.
func (m *MockServerResponse) Header() http.Header {
	return m.HeaderMap
}

// Write implements http.ResponseWriter.
func (m *MockServerResponse) Write(data []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	chunk := make([]byte, len(data))
	copy(chunk, data)
	m.WrittenChunks = append(m.WrittenChunks, chunk)
	return len(data), nil
}

// WriteHeader implements http.ResponseWriter.
func (m *MockServerResponse) WriteHeader(statusCode int) {
	m.StatusCode = statusCode
}

// WriteHeadFull sets the status code, status message, and headers.
// This matches the TS writeHead(statusCode, statusMessage, headers) method.
func (m *MockServerResponse) WriteHeadFull(statusCode int, statusMessage string, headers map[string]string) {
	m.StatusCode = statusCode
	m.StatusMessage = statusMessage
	for k, v := range headers {
		m.HeaderMap.Set(k, v)
	}
}

// End marks the response as ended.
func (m *MockServerResponse) End() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.Ended {
		m.Ended = true
		close(m.endCh)
	}
}

// Body returns all written chunks combined as a single string.
func (m *MockServerResponse) Body() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var sb strings.Builder
	for _, chunk := range m.WrittenChunks {
		sb.Write(chunk)
	}
	return sb.String()
}

// GetDecodedChunks returns the written chunks as decoded strings.
func (m *MockServerResponse) GetDecodedChunks() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.WrittenChunks))
	for i, chunk := range m.WrittenChunks {
		result[i] = string(chunk)
	}
	return result
}

// WaitForEnd blocks until End() is called or the timeout expires.
// Returns true if the response ended, false on timeout.
func (m *MockServerResponse) WaitForEnd(timeout time.Duration) bool {
	select {
	case <-m.endCh:
		return true
	case <-time.After(timeout):
		return false
	}
}
