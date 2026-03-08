// Ported from: packages/ai/src/ui-message-stream/create-ui-message-stream-response.test.ts
package uimessagestream

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestCreateUIMessageStreamResponse(t *testing.T) {
	t.Run("should create a response with correct headers and encoded stream", func(t *testing.T) {
		chunks := []UIMessageChunk{
			{Type: "text-delta", ID: "1", Delta: "test-data"},
		}
		stream := sliceToChan(chunks)

		rec := httptest.NewRecorder()
		headers := make(http.Header)
		headers.Set("Custom-Header", "test")

		CreateUIMessageStreamResponse(rec, CreateUIMessageStreamResponseOptions{
			UIMessageStreamResponseInit: UIMessageStreamResponseInit{
				Status:  200,
				Headers: headers,
			},
			Stream: stream,
		})

		result := rec.Result()
		if result.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", result.StatusCode)
		}

		// Verify headers
		if got := result.Header.Get("Content-Type"); got != "text/event-stream" {
			t.Errorf("expected text/event-stream, got %q", got)
		}
		if got := result.Header.Get("Cache-Control"); got != "no-cache" {
			t.Errorf("expected no-cache, got %q", got)
		}
		if got := result.Header.Get("Connection"); got != "keep-alive" {
			t.Errorf("expected keep-alive, got %q", got)
		}
		if got := result.Header.Get("X-Vercel-Ai-Ui-Message-Stream"); got != "v1" {
			t.Errorf("expected v1, got %q", got)
		}
		if got := result.Header.Get("X-Accel-Buffering"); got != "no" {
			t.Errorf("expected no, got %q", got)
		}
		if got := result.Header.Get("Custom-Header"); got != "test" {
			t.Errorf("expected test, got %q", got)
		}

		body := rec.Body.String()
		// Should contain SSE data for the chunk
		expectedJSON, _ := json.Marshal(chunks[0])
		expectedSSE := "data: " + string(expectedJSON) + "\n\n"
		if !strings.Contains(body, expectedSSE) {
			t.Errorf("body missing expected SSE data.\ngot: %q\nwant contains: %q", body, expectedSSE)
		}
		if !strings.Contains(body, "data: [DONE]\n\n") {
			t.Error("body missing [DONE] sentinel")
		}
	})

	t.Run("should handle errors in the stream", func(t *testing.T) {
		chunks := []UIMessageChunk{
			{Type: "error", ErrorText: "Custom error message"},
		}
		stream := sliceToChan(chunks)

		rec := httptest.NewRecorder()
		CreateUIMessageStreamResponse(rec, CreateUIMessageStreamResponseOptions{
			UIMessageStreamResponseInit: UIMessageStreamResponseInit{
				Status: 200,
			},
			Stream: stream,
		})

		body := rec.Body.String()
		if !strings.Contains(body, `"errorText":"Custom error message"`) {
			t.Errorf("body missing error text: %q", body)
		}
		if !strings.Contains(body, "data: [DONE]\n\n") {
			t.Error("body missing [DONE] sentinel")
		}
	})

	t.Run("should call consumeSseStream with a teed stream", func(t *testing.T) {
		chunks := []UIMessageChunk{
			{Type: "text-delta", ID: "1", Delta: "test-data-1"},
			{Type: "text-delta", ID: "1", Delta: "test-data-2"},
		}
		stream := sliceToChan(chunks)

		var consumedData []string
		var mu sync.Mutex
		consumeDone := make(chan struct{})

		rec := httptest.NewRecorder()
		CreateUIMessageStreamResponse(rec, CreateUIMessageStreamResponseOptions{
			UIMessageStreamResponseInit: UIMessageStreamResponseInit{
				Status: 200,
				ConsumeSseStream: func(sseStream <-chan string) {
					defer close(consumeDone)
					for s := range sseStream {
						mu.Lock()
						consumedData = append(consumedData, s)
						mu.Unlock()
					}
				},
			},
			Stream: stream,
		})

		<-consumeDone

		mu.Lock()
		defer mu.Unlock()
		if len(consumedData) == 0 {
			t.Error("consumeSseStream received no data")
		}

		// Verify the response stream still works correctly
		body := rec.Body.String()
		if !strings.Contains(body, "test-data-1") {
			t.Errorf("response body missing test-data-1: %q", body)
		}
		if !strings.Contains(body, "test-data-2") {
			t.Errorf("response body missing test-data-2: %q", body)
		}
		if !strings.Contains(body, "data: [DONE]\n\n") {
			t.Error("body missing [DONE] sentinel")
		}
	})

	t.Run("should handle synchronous consumeSseStream", func(t *testing.T) {
		chunks := []UIMessageChunk{
			{Type: "text-delta", ID: "1", Delta: "sync-test"},
		}
		stream := sliceToChan(chunks)

		called := false
		consumeDone := make(chan struct{})

		rec := httptest.NewRecorder()
		CreateUIMessageStreamResponse(rec, CreateUIMessageStreamResponseOptions{
			UIMessageStreamResponseInit: UIMessageStreamResponseInit{
				Status: 200,
				ConsumeSseStream: func(sseStream <-chan string) {
					defer close(consumeDone)
					called = true
					for range sseStream {
						// consume
					}
				},
			},
			Stream: stream,
		})

		<-consumeDone

		if !called {
			t.Error("consumeSseStream was not called")
		}

		body := rec.Body.String()
		if !strings.Contains(body, "sync-test") {
			t.Errorf("response body missing sync-test: %q", body)
		}
	})
}

// sliceToChan converts a slice of UIMessageChunk to a channel.
func sliceToChan(chunks []UIMessageChunk) <-chan UIMessageChunk {
	ch := make(chan UIMessageChunk)
	go func() {
		defer close(ch)
		for _, c := range chunks {
			ch <- c
		}
	}()
	return ch
}
