// Ported from: packages/ai/src/ui-message-stream/ui-message-stream-response-init.ts
package uimessagestream

import "net/http"

// UIMessageStreamResponseInit contains options for creating a UI message stream response.
// It extends the standard ResponseInit with additional streaming options.
type UIMessageStreamResponseInit struct {
	// Status is the HTTP status code for the response.
	Status int

	// StatusText is the HTTP status text for the response.
	StatusText string

	// Headers contains additional HTTP headers to include in the response.
	Headers http.Header

	// ConsumeSseStream is an optional callback to consume a copy of the SSE stream independently.
	// This is useful for logging, debugging, or processing the stream in parallel.
	// The callback receives a channel of SSE-formatted strings and does not block the response.
	ConsumeSseStream func(stream <-chan string)
}
