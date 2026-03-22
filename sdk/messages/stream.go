package messages

import "encoding/json"

// StreamChunk represents one chunk of a streaming response.
// Streaming = multiple messages on a scoped topic with a terminal marker.
type StreamChunk struct {
	StreamID string          `json:"streamId"`
	Seq      int             `json:"seq"`
	Delta    string          `json:"delta,omitempty"`
	Done     bool            `json:"done"`
	Final    json.RawMessage `json:"final,omitempty"`
}

func (StreamChunk) BusTopic() string { return "stream.chunk" }
