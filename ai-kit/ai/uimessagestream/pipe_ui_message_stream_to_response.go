// Ported from: packages/ai/src/ui-message-stream/pipe-ui-message-stream-to-response.ts
package uimessagestream

import (
	"io"
	"net/http"

	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// PipeUIMessageStreamToResponseOptions configures PipeUIMessageStreamToResponse.
type PipeUIMessageStreamToResponseOptions struct {
	// Response is the http.ResponseWriter to write to.
	Response http.ResponseWriter

	// Stream is the UI message chunk channel to send.
	Stream <-chan UIMessageChunk

	UIMessageStreamResponseInit
}

// PipeUIMessageStreamToResponse pipes a UI message stream to an http.ResponseWriter.
// The stream is transformed to Server-Sent Events (SSE) format.
//
// In TypeScript this writes to a Node.js ServerResponse. In Go it writes to
// an http.ResponseWriter.
func PipeUIMessageStreamToResponse(opts PipeUIMessageStreamToResponseOptions) {
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

	statusText := opts.StatusText

	// Write headers using WriteToServerResponse pattern
	headerMap := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			headerMap[key] = values[0]
		}
	}

	// Set headers on response
	for key, value := range headerMap {
		opts.Response.Header().Set(key, value)
	}

	if statusText != "" {
		// Go's net/http doesn't support custom status text directly,
		// but we set the status code
		_ = statusText
	}
	opts.Response.WriteHeader(status)

	// Stream the SSE data
	flusher, canFlush := opts.Response.(http.Flusher)
	for s := range mainStream {
		_, _ = io.WriteString(opts.Response, s)
		if canFlush {
			flusher.Flush()
		}
	}
}
