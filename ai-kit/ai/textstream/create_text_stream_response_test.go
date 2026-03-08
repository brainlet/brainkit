// Ported from: packages/ai/src/text-stream/create-text-stream-response.test.ts
package textstream

import (
	"io"
	"net/http/httptest"
	"testing"
)

func TestCreateTextStreamResponse(t *testing.T) {
	t.Run("should create a response with correct headers and encoded stream", func(t *testing.T) {
		textStream := make(chan string, 1)
		textStream <- "test-data"
		close(textStream)

		recorder := httptest.NewRecorder()
		CreateTextStreamResponse(recorder, CreateTextStreamResponseOptions{
			Status: 200,
			Headers: map[string]string{
				"Custom-Header": "test",
			},
			TextStream: textStream,
		})

		result := recorder.Result()
		defer result.Body.Close()

		// Verify status
		if result.StatusCode != 200 {
			t.Errorf("expected status 200, got %d", result.StatusCode)
		}

		// Verify headers
		ct := result.Header.Get("Content-Type")
		if ct != "text/plain; charset=utf-8" {
			t.Errorf("expected Content-Type 'text/plain; charset=utf-8', got %q", ct)
		}

		ch := result.Header.Get("Custom-Header")
		if ch != "test" {
			t.Errorf("expected Custom-Header 'test', got %q", ch)
		}

		// Verify body
		body, err := io.ReadAll(result.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if string(body) != "test-data" {
			t.Errorf("expected body 'test-data', got %q", string(body))
		}
	})
}
