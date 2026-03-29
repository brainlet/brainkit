package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk/messages"
)

// streamEvent is a parsed stream protocol message with sequence number.
type streamEvent struct {
	Type  string          `json:"type"`
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
	Seq   int             `json:"seq"`

	// Raw message data for non-typed payloads
	rawPayload []byte
	metadata   map[string]string
}

func (e *streamEvent) isTerminal() bool {
	return e.Type == "end" || e.Type == "error"
}

func (e *streamEvent) isRawDone() bool {
	return e.metadata != nil && e.metadata["done"] == "true"
}

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

	// Seq-based ordered delivery.
	// Messages arrive with seq field. We write them in order.
	// If a message arrives out of order, buffer it until the gap fills.
	// Stream completes when all seqs 0..terminalSeq have been written.
	ra := newReassembler(w, flusher)

	for {
		select {
		case msg := <-eventCh:
			evt := parseStreamEvent(msg.Payload, msg.Metadata)
			if ra.receive(evt) {
				return // stream complete — all seqs delivered in order
			}
		case <-ctx.Done():
			// Client disconnected or timeout — flush what we have in order
			ra.flushAll()
			return
		}
	}
}

// reassembler accumulates stream events and writes them in seq order.
// No timers, no arbitrary waits — purely seq-driven completion.
type reassembler struct {
	w       http.ResponseWriter
	flusher http.Flusher

	nextSeq     int              // next seq expected to write
	terminalSeq int              // seq of end/error event (-1 = not seen)
	buffer      map[int]*streamEvent // out-of-order events waiting for their turn
	hasSeq      bool             // true if any message had seq field (typed protocol)
}

func newReassembler(w http.ResponseWriter, flusher http.Flusher) *reassembler {
	return &reassembler{
		w:           w,
		flusher:     flusher,
		terminalSeq: -1,
		buffer:      make(map[int]*streamEvent),
	}
}

// receive processes one event. Returns true if the stream is complete
// (all seqs 0..terminalSeq written).
func (ra *reassembler) receive(evt *streamEvent) bool {
	// Untyped payload (no seq) — legacy path, write immediately
	if evt.Type == "" && evt.Seq == 0 && !ra.hasSeq {
		ra.writeRaw(evt)
		return evt.isRawDone()
	}

	ra.hasSeq = true

	// Track terminal
	if evt.isTerminal() {
		ra.terminalSeq = evt.Seq
	}

	if evt.Seq == ra.nextSeq {
		// In order — write immediately and flush any consecutive buffered events
		ra.writeEvent(evt)
		ra.nextSeq++
		ra.flushConsecutive()
	} else if evt.Seq > ra.nextSeq {
		// Out of order — buffer until gap fills
		ra.buffer[evt.Seq] = evt
	}
	// seq < nextSeq → duplicate, ignore

	return ra.isComplete()
}

// flushConsecutive writes any buffered events that are now in sequence.
func (ra *reassembler) flushConsecutive() {
	for {
		evt, ok := ra.buffer[ra.nextSeq]
		if !ok {
			break
		}
		delete(ra.buffer, ra.nextSeq)
		ra.writeEvent(evt)
		ra.nextSeq++
	}
}

// isComplete returns true when all seqs up to and including the terminal have been written.
func (ra *reassembler) isComplete() bool {
	return ra.terminalSeq >= 0 && ra.nextSeq > ra.terminalSeq
}

// flushAll writes any remaining buffered events in seq order (best effort on disconnect).
func (ra *reassembler) flushAll() {
	if len(ra.buffer) == 0 {
		return
	}
	seqs := make([]int, 0, len(ra.buffer))
	for seq := range ra.buffer {
		seqs = append(seqs, seq)
	}
	sort.Ints(seqs)
	for _, seq := range seqs {
		ra.writeEvent(ra.buffer[seq])
	}
}

func (ra *reassembler) writeEvent(evt *streamEvent) {
	eventName := evt.Type
	if evt.Type == "event" && evt.Event != "" {
		eventName = evt.Event
	}
	fmt.Fprintf(ra.w, "event: %s\ndata: %s\n\n", eventName, string(evt.Data))
	ra.flusher.Flush()
}

func (ra *reassembler) writeRaw(evt *streamEvent) {
	fmt.Fprintf(ra.w, "data: %s\n\n", string(evt.rawPayload))
	ra.flusher.Flush()
}

// parseStreamEvent extracts the typed envelope with seq from a bus message payload.
func parseStreamEvent(payload []byte, metadata map[string]string) *streamEvent {
	var evt streamEvent
	if json.Unmarshal(payload, &evt) == nil && evt.Type != "" {
		evt.metadata = metadata
		return &evt
	}
	// Untyped payload — raw passthrough
	return &streamEvent{
		rawPayload: payload,
		metadata:   metadata,
	}
}
