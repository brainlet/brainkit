// Ported from: packages/ai/src/ui-message-stream/pipe-ui-message-stream-to-response.test.ts
package uimessagestream

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPipeUIMessageStreamToResponse(t *testing.T) {
	t.Run("should write to ResponseWriter with correct headers and encoded stream", func(t *testing.T) {
		chunks := []UIMessageChunk{
			{Type: "text-start", ID: "1"},
			{Type: "text-delta", ID: "1", Delta: "test-data"},
			{Type: "text-end", ID: "1"},
		}
		stream := sliceToChan(chunks)

		rec := httptest.NewRecorder()

		PipeUIMessageStreamToResponse(PipeUIMessageStreamToResponseOptions{
			Response: rec,
			Stream:   stream,
			UIMessageStreamResponseInit: UIMessageStreamResponseInit{
				Status:     200,
				StatusText: "OK",
			},
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
		if got := result.Header.Get("X-Vercel-Ai-Ui-Message-Stream"); got != "v1" {
			t.Errorf("expected v1, got %q", got)
		}

		body := rec.Body.String()

		// Should contain SSE data for each chunk
		for _, chunk := range chunks {
			expectedJSON, _ := json.Marshal(chunk)
			expectedSSE := "data: " + string(expectedJSON) + "\n\n"
			if !strings.Contains(body, expectedSSE) {
				t.Errorf("body missing expected SSE data for %s.\ngot: %q\nwant contains: %q", chunk.Type, body, expectedSSE)
			}
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

		PipeUIMessageStreamToResponse(PipeUIMessageStreamToResponseOptions{
			Response: rec,
			Stream:   stream,
			UIMessageStreamResponseInit: UIMessageStreamResponseInit{
				Status: 200,
			},
		})

		body := rec.Body.String()
		if !strings.Contains(body, `"errorText":"Custom error message"`) {
			t.Errorf("body missing error text: %q", body)
		}
		if !strings.Contains(body, "data: [DONE]\n\n") {
			t.Error("body missing [DONE] sentinel")
		}
	})
}
