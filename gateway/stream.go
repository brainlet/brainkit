package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"github.com/brainlet/brainkit/internal/syncx"
	"time"

	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/google/uuid"
)

// --- Stream Session ---

// streamSession manages one SSE stream lifecycle: bus subscription,
// event buffering, SSE writing, reconnection, and safety caps.
type streamSession struct {
	id            string // unique stream token (UUID)
	replyTo       string // bus topic for this stream
	correlationID string

	mu         syncx.RWMutex
	buffer     []bufferedEvent // append-only replay buffer for reconnection
	nextID     int             // sequential SSE id counter
	eventCount int             // total data events written (for MaxEvents cap)
	terminal   bool            // true after end/error/timeout
	terminalAt time.Time       // when terminal event occurred (for grace period)
	hasWriter  bool            // true when an HTTP handler goroutine is active

	eventCh chan streamEvent // bus messages arrive here
	unsub   func()          // bus subscription cancel

	config StreamConfig
}

type bufferedEvent struct {
	id   int    // SSE id
	data []byte // full SSE line
}

type streamEvent struct {
	Type     string          `json:"type"`
	Seq      int             `json:"seq"`
	Total    int             `json:"total,omitempty"` // set on terminal events (end/error)
	Event    string          `json:"event,omitempty"`
	Data     json.RawMessage `json:"data"`
	Payload  []byte          // raw message payload
	Metadata map[string]string
}

func parseStreamEvent(payload []byte, metadata map[string]string) streamEvent {
	var evt streamEvent
	evt.Payload = payload
	evt.Metadata = metadata
	json.Unmarshal(payload, &evt)
	return evt
}

func (e *streamEvent) isTerminal() bool {
	return e.Type == "end" || e.Type == "error"
}

func (e *streamEvent) isDoneMetadata() bool {
	return e.Metadata != nil && e.Metadata["done"] == "true"
}

// --- Stream ID encoding ---

func formatStreamID(token string, seq int) string {
	return token + ":" + strconv.Itoa(seq)
}

func parseStreamID(lastEventID string) (token string, lastSeq int) {
	idx := strings.LastIndex(lastEventID, ":")
	if idx < 0 {
		return lastEventID, -1
	}
	token = lastEventID[:idx]
	seq, err := strconv.Atoi(lastEventID[idx+1:])
	if err != nil {
		return lastEventID, -1
	}
	return token, seq
}

// --- SSE Writing Helpers ---

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, id, eventName string, data []byte) []byte {
	line := []byte(fmt.Sprintf("id: %s\nevent: %s\ndata: %s\n\n", id, eventName, string(data)))
	w.Write(line)
	flusher.Flush()
	return line
}

func writeKeepalive(w http.ResponseWriter, flusher http.Flusher) {
	w.Write([]byte(":keepalive\n\n"))
	flusher.Flush()
}

func writeSyntheticError(w http.ResponseWriter, flusher http.Flusher, id, reason, msg string) {
	errData, _ := json.Marshal(map[string]string{"message": msg, "reason": reason})
	w.Write([]byte(fmt.Sprintf("id: %s\nevent: error\ndata: %s\n\n", id, string(errData))))
	flusher.Flush()
}

// --- Stream Session Methods ---

func newStreamSession(gw *Gateway, replyTo, correlationID string) (*streamSession, error) {
	id := uuid.NewString()
	cfg := gw.streamConfig

	session := &streamSession{
		id:            id,
		replyTo:       replyTo,
		correlationID: correlationID,
		eventCh:       make(chan streamEvent, cfg.MaxEvents),
		config:        cfg,
	}

	// Subscribe to replyTo — bus subscriber goroutine pushes to eventCh.
	// This subscription lives for the entire session lifetime (survives client disconnect).
	// Uses a long-lived context — cancelled only when session is cleaned up.
	unsub, err := gw.rt.SubscribeRaw(context.Background(), replyTo, func(msg messages.Message) {
		evt := parseStreamEvent(msg.Payload, msg.Metadata)
		session.eventCh <- evt
	})
	if err != nil {
		return nil, err
	}
	session.unsub = unsub

	gw.registerSession(session)
	return session, nil
}

// terminate marks the session as terminal.
func (s *streamSession) terminate(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.terminal {
		return
	}
	s.terminal = true
	s.terminalAt = time.Now()
}

// writeLoop reads from eventCh, classifies messages, writes SSE, manages timers.
func (s *streamSession) writeLoop(w http.ResponseWriter, flusher http.Flusher, r *http.Request) {
	s.mu.Lock()
	s.hasWriter = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.hasWriter = false
		s.mu.Unlock()
	}()

	heartbeatTimer := time.NewTimer(s.config.HeartbeatTimeout)
	defer heartbeatTimer.Stop()

	maxDurationTimer := time.NewTimer(s.config.MaxDuration)
	defer maxDurationTimer.Stop()

	ctx := r.Context()

	for {
		select {
		case evt, ok := <-s.eventCh:
			if !ok {
				return
			}

			heartbeatTimer.Reset(s.config.HeartbeatTimeout)

			// Heartbeat — send keepalive, don't buffer
			if evt.Type == "heartbeat" {
				writeKeepalive(w, flusher)
				continue
			}

			// Terminal: typed end/error OR untyped done=true.
			if evt.isTerminal() || (evt.Type == "" && evt.isDoneMetadata()) {
				if evt.isTerminal() && evt.Total > 0 {
					// Sequenced terminal: the producer sent `total` events before
					// this terminal. Wait until we've written all of them. This
					// handles GoChannel's async delivery (goroutine per Publish)
					// where events arrive out of order under CPU pressure.
					// Wait for all data events before writing terminal.
					// Safety: heartbeat timeout catches producer death,
					// maxDuration caps total wait, ctx.Done() handles
					// client disconnect. No bare channel reads.
					written := s.eventCount
					reassemblyTimeout := time.NewTimer(s.config.HeartbeatTimeout)
					defer reassemblyTimeout.Stop()
				reassemble:
					for written < evt.Total {
						select {
						case next := <-s.eventCh:
							reassemblyTimeout.Reset(s.config.HeartbeatTimeout)
							if next.Type == "heartbeat" {
								continue
							}
							if next.isTerminal() || (next.Type == "" && next.isDoneMetadata()) {
								continue
							}
							eName := next.Type
							if next.Type == "event" && next.Event != "" {
								eName = next.Event
							}
							s.mu.Lock()
							did := formatStreamID(s.id, s.nextID)
							dline := writeSSEEvent(w, flusher, did, eName, next.Data)
							s.buffer = append(s.buffer, bufferedEvent{id: s.nextID, data: dline})
							s.nextID++
							s.eventCount++
							written = s.eventCount
							s.mu.Unlock()
						case <-reassemblyTimeout.C:
							break reassemble
						case <-maxDurationTimer.C:
							break reassemble
						case <-ctx.Done():
							break reassemble
						}
					}
					// All data events written. Now write the terminal.
					s.mu.Lock()
					id := formatStreamID(s.id, s.nextID)
					line := writeSSEEvent(w, flusher, id, evt.Type, evt.Data)
					s.buffer = append(s.buffer, bufferedEvent{id: s.nextID, data: line})
					s.nextID++
					s.mu.Unlock()
					s.terminate(evt.Type)
				} else if evt.isTerminal() {
					// Unsequenced terminal (total=0): legacy or no prior events.
					// Write immediately — no events to wait for.
					s.mu.Lock()
					id := formatStreamID(s.id, s.nextID)
					line := writeSSEEvent(w, flusher, id, evt.Type, evt.Data)
					s.buffer = append(s.buffer, bufferedEvent{id: s.nextID, data: line})
					s.nextID++
					s.mu.Unlock()
					s.terminate(evt.Type)
				} else {
					// Untyped done=true (from handleHandlerFailure)
					s.mu.Lock()
					id := formatStreamID(s.id, s.nextID)
					s.nextID++
					s.mu.Unlock()
					errMsg := string(evt.Payload)
					var parsed struct{ Error string `json:"error"` }
					if json.Unmarshal(evt.Payload, &parsed) == nil && parsed.Error != "" {
						errMsg = parsed.Error
					}
					writeSyntheticError(w, flusher, id, "producer_error", errMsg)
					s.terminate("producer_error")
				}
				return
			}

			// Data event
			eventName := evt.Type
			if evt.Type == "event" && evt.Event != "" {
				eventName = evt.Event
			}

			s.mu.Lock()
			id := formatStreamID(s.id, s.nextID)
			line := writeSSEEvent(w, flusher, id, eventName, evt.Data)
			s.buffer = append(s.buffer, bufferedEvent{id: s.nextID, data: line})
			s.nextID++
			s.eventCount++
			count := s.eventCount
			s.mu.Unlock()

			// Max events cap
			if count >= s.config.MaxEvents {
				s.mu.RLock()
				capID := formatStreamID(s.id, s.nextID)
				s.mu.RUnlock()
				writeSyntheticError(w, flusher, capID, "max_events",
					fmt.Sprintf("stream event limit exceeded (%d)", s.config.MaxEvents))
				s.terminate("max_events")
				return
			}

		case <-heartbeatTimer.C:
			s.mu.RLock()
			id := formatStreamID(s.id, s.nextID)
			s.mu.RUnlock()
			writeSyntheticError(w, flusher, id, "heartbeat_timeout",
				"stream timeout: producer not responding")
			s.terminate("heartbeat_timeout")
			return

		case <-maxDurationTimer.C:
			s.mu.RLock()
			id := formatStreamID(s.id, s.nextID)
			s.mu.RUnlock()
			writeSyntheticError(w, flusher, id, "max_duration",
				fmt.Sprintf("stream duration limit exceeded (%s)", s.config.MaxDuration))
			s.terminate("max_duration")
			return

		case <-ctx.Done():
			// Client disconnected — session stays alive for reconnection
			return
		}
	}
}

// replayAndResume replays buffered events from lastSeq+1, then enters writeLoop.
func (s *streamSession) replayAndResume(w http.ResponseWriter, flusher http.Flusher, r *http.Request, lastSeq int) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Replay buffered events after lastSeq
	s.mu.RLock()
	for _, evt := range s.buffer {
		if evt.id > lastSeq {
			w.Write(evt.data)
			flusher.Flush()
		}
	}
	isTerminal := s.terminal
	s.mu.RUnlock()

	if isTerminal {
		return // Stream ended during disconnect — replay was enough
	}

	s.writeLoop(w, flusher, r)
}

// --- Gateway handleStream ---

func (gw *Gateway) handleStream(w http.ResponseWriter, r *http.Request, matched *route, pathParams map[string]string) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Reconnection: check for Last-Event-Id header
	if lastEventID := r.Header.Get("Last-Event-Id"); lastEventID != "" {
		token, lastSeq := parseStreamID(lastEventID)
		if lastSeq < 0 {
			http.Error(w, `{"error":"invalid Last-Event-Id format"}`, http.StatusBadRequest)
			return
		}
		session := gw.findSession(token)
		if session == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusGone)
			w.Write([]byte(`{"error":"stream session expired","reason":"session_expired"}`))
			return
		}
		// Wait for previous writer to exit (client disconnect in-flight)
		writerReady := false
		deadline := time.NewTimer(500 * time.Millisecond)
		tick := time.NewTicker(10 * time.Millisecond)
	waitWriter:
		for {
			select {
			case <-tick.C:
				session.mu.RLock()
				busy := session.hasWriter
				session.mu.RUnlock()
				if !busy {
					writerReady = true
					break waitWriter
				}
			case <-deadline.C:
				break waitWriter
			}
		}
		tick.Stop()
		deadline.Stop()
		if !writerReady {
			http.Error(w, `{"error":"stream already has active writer"}`, http.StatusConflict)
			return
		}
		session.replayAndResume(w, flusher, r, lastSeq)
		return
	}

	// New stream
	payload, err := buildPayload(r, matched, pathParams)
	if err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	reqID := requestID(r)
	replyTo := matched.Topic + ".reply." + reqID

	session, err := newStreamSession(gw, replyTo, reqID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	pubCtx := transport.WithPublishMeta(r.Context(), reqID, replyTo)
	if _, err := gw.rt.PublishRaw(pubCtx, matched.Topic, payload); err != nil {
		http.Error(w, "publish failed", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	session.writeLoop(w, flusher, r)
}
