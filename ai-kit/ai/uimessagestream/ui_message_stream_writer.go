// Ported from: packages/ai/src/ui-message-stream/ui-message-stream-writer.ts
package uimessagestream

// UIMessageStreamWriter provides methods to write chunks to a UI message stream.
type UIMessageStreamWriter struct {
	// write sends a chunk to the stream's internal channel.
	write func(chunk UIMessageChunk)

	// merge merges the contents of another stream (channel) into this stream.
	merge func(stream <-chan UIMessageChunk)

	// OnError is the error handler used by the writer.
	// This is intended for forwarding when merging streams
	// to prevent duplicated error masking.
	OnError func(err error) string
}

// Write appends a UI message chunk to the stream.
func (w *UIMessageStreamWriter) Write(part UIMessageChunk) {
	w.write(part)
}

// Merge merges the contents of another stream channel into this stream.
func (w *UIMessageStreamWriter) Merge(stream <-chan UIMessageChunk) {
	w.merge(stream)
}
