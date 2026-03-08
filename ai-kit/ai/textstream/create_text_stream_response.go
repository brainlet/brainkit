// Ported from: packages/ai/src/text-stream/create-text-stream-response.ts
package textstream

import (
	"io"
	"net/http"
)

// CreateTextStreamResponseOptions are the options for creating a text stream response.
type CreateTextStreamResponseOptions struct {
	// Status is the HTTP status code (default: 200).
	Status int
	// StatusText is the HTTP status text.
	StatusText string
	// Headers are additional response headers.
	Headers map[string]string
	// TextStream is a channel that yields string chunks.
	TextStream <-chan string
}

// CreateTextStreamResponse creates an http.Handler that streams text chunks.
// Each text chunk is encoded as UTF-8 and sent as a separate chunk.
// Sets a Content-Type header to text/plain; charset=utf-8.
func CreateTextStreamResponse(w http.ResponseWriter, opts CreateTextStreamResponseOptions) {
	status := opts.Status
	if status == 0 {
		status = http.StatusOK
	}

	// Set default content-type header
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	// Set additional headers
	for k, v := range opts.Headers {
		w.Header().Set(k, v)
	}

	w.WriteHeader(status)

	flusher, canFlush := w.(http.Flusher)

	for chunk := range opts.TextStream {
		_, _ = io.WriteString(w, chunk)
		if canFlush {
			flusher.Flush()
		}
	}
}
