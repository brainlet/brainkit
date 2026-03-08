// Ported from: packages/ai/src/ui-message-stream/json-to-sse-transform-stream.ts
package uimessagestream

import (
	"encoding/json"
	"fmt"
)

// JsonToSseTransform converts a channel of UIMessageChunks into a channel of
// SSE-formatted strings. Each chunk is serialized to JSON and wrapped in
// "data: ...\n\n" format. When the input channel closes, a "data: [DONE]\n\n"
// sentinel is sent.
//
// This is the Go equivalent of the TypeScript JsonToSseTransformStream which
// extends TransformStream<unknown, string>.
func JsonToSseTransform(input <-chan UIMessageChunk) <-chan string {
	output := make(chan string)
	go func() {
		defer close(output)
		for chunk := range input {
			data, err := json.Marshal(chunk)
			if err != nil {
				// Best-effort: skip chunks that can't be serialized
				continue
			}
			output <- fmt.Sprintf("data: %s\n\n", string(data))
		}
		output <- "data: [DONE]\n\n"
	}()
	return output
}
