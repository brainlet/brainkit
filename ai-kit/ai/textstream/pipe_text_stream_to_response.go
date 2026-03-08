// Ported from: packages/ai/src/text-stream/pipe-text-stream-to-response.ts
package textstream

import (
	"io"
	"net/http"
)

// PipeTextStreamToResponseOptions are the options for piping a text stream to a response.
type PipeTextStreamToResponseOptions struct {
	// Response is the http.ResponseWriter to write to.
	Response http.ResponseWriter
	// Status is the HTTP status code.
	Status int
	// StatusText is the HTTP status text.
	StatusText string
	// Headers are additional response headers.
	Headers map[string]string
	// TextStream is a channel that yields string chunks.
	TextStream <-chan string
}

// PipeTextStreamToResponse writes a text stream to an http.ResponseWriter.
// Each text chunk is encoded as UTF-8 and written as a separate chunk.
// Sets a Content-Type header to text/plain; charset=utf-8.
func PipeTextStreamToResponse(opts PipeTextStreamToResponseOptions) {
	w := opts.Response

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
