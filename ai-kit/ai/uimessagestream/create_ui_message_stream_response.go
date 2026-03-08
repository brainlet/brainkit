// Ported from: packages/ai/src/ui-message-stream/create-ui-message-stream-response.ts
package uimessagestream

import (
	"io"
	"net/http"

	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// CreateUIMessageStreamResponseOptions configures CreateUIMessageStreamResponse.
type CreateUIMessageStreamResponseOptions struct {
	UIMessageStreamResponseInit

	// Stream is the UI message chunk channel to send.
	Stream <-chan UIMessageChunk
}

// CreateUIMessageStreamResponse writes a UI message stream response to an
// http.ResponseWriter. The stream is transformed to Server-Sent Events (SSE) format.
//
// In TypeScript this returns a Response object. In Go, we write directly to the
// ResponseWriter since Go's HTTP model is push-based.
func CreateUIMessageStreamResponse(w http.ResponseWriter, opts CreateUIMessageStreamResponseOptions) {
	sseStream := JsonToSseTransform(opts.Stream)

	// When consumeSseStream is provided, tee the stream
	var mainStream <-chan string
	if opts.ConsumeSseStream != nil {
		ch1 := make(chan string)
		ch2 := make(chan string)
		go func() {
			defer close(ch1)
			defer close(ch2)
			for s := range sseStream {
				ch1 <- s
				ch2 <- s
			}
		}()
		mainStream = ch1
		go opts.ConsumeSseStream(ch2)
	} else {
		mainStream = sseStream
	}

	// Prepare headers
	headers := opts.Headers
	if headers == nil {
		headers = make(http.Header)
	}
	headers = util.PrepareHeaders(headers, UIMessageStreamHeaders)

	status := opts.Status
	if status == 0 {
		status = 200
	}

	// Set headers
	for key, values := range headers {
		for _, v := range values {
			w.Header().Set(key, v)
		}
	}

	w.WriteHeader(status)

	// Stream the SSE data
	flusher, canFlush := w.(http.Flusher)
	for s := range mainStream {
		_, _ = io.WriteString(w, s)
		if canFlush {
			flusher.Flush()
		}
	}
}
