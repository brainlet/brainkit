// Ported from: packages/provider/src/language-model/v3/language-model-v3-stream-result.ts
package languagemodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// StreamResult is the result of a language model doStream call.
type StreamResult struct {
	// Stream is a channel that delivers stream parts.
	// In TS this is ReadableStream<StreamPart>; in Go we use a receive-only channel.
	Stream <-chan StreamPart

	// Request contains optional request information for telemetry and debugging.
	Request *StreamResultRequest

	// Response contains optional response data.
	Response *StreamResultResponse
}

// StreamResultRequest contains request information.
type StreamResultRequest struct {
	// Body is the request HTTP body that was sent to the provider API.
	Body any
}

// StreamResultResponse contains response information.
type StreamResultResponse struct {
	// Headers are the response headers.
	Headers shared.Headers
}
