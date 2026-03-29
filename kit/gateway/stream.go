package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk/messages"
)

func (gw *Gateway) handleStream(w http.ResponseWriter, r *http.Request, matched *route, pathParams map[string]string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	payload, err := buildPayload(r, matched, pathParams)
	if err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	reqID := requestID(r)
	replyTo := matched.Topic + ".reply." + reqID

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	eventCh := make(chan messages.Message, 16)
	unsub, err := gw.rt.SubscribeRaw(ctx, replyTo, func(msg messages.Message) {
		select {
		case eventCh <- msg:
		default:
		}
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer unsub()

	pubCtx := messaging.WithPublishMeta(ctx, reqID, replyTo)
	if _, err := gw.rt.PublishRaw(pubCtx, matched.Topic, payload); err != nil {
		http.Error(w, "publish failed", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for {
		select {
		case msg := <-eventCh:
			terminal := writeSSEEvent(w, flusher, msg.Payload, msg.Metadata)
			if terminal {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// writeSSEEvent writes one SSE event. Returns true if terminal (end/error type or done metadata).
func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, payload []byte, metadata map[string]string) bool {
	// Try typed stream envelope first (streaming protocol wire format)
	var envelope struct {
		Type  string          `json:"type"`
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}

	if json.Unmarshal(payload, &envelope) == nil && envelope.Type != "" {
		eventName := envelope.Type
		if envelope.Type == "event" && envelope.Event != "" {
			eventName = envelope.Event
		}
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, string(envelope.Data))
		flusher.Flush()
		return envelope.Type == "end" || envelope.Type == "error"
	}

	// Raw payload — no type field
	fmt.Fprintf(w, "data: %s\n\n", string(payload))
	flusher.Flush()

	// Check metadata done flag for raw payloads
	return metadata != nil && metadata["done"] == "true"
}
