package gateway

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/sdk"
	"github.com/google/uuid"
)

// registerBusAPIRoutes adds the built-in POST /api/bus and
// POST /api/stream handlers to mux. The bus API exposes the Kit's
// request→reply bus surface over HTTP — the canonical external
// entry point used by the brainkit CLI and any downstream tool
// that wants to drive a running Kit without linking against
// brainkit directly.
//
// Disable via Config.NoBusAPI=true.
func registerBusAPIRoutes(mux *http.ServeMux, rt sdk.Runtime) {
	mux.HandleFunc("POST /api/bus", busAPIHandler(rt))
	mux.HandleFunc("POST /api/stream", busAPIStreamHandler(rt))
}

// busAPIHandler handles POST /api/bus — generic bus request-reply
// over HTTP.
//
// Body:     {"topic":"kit.health","payload":{...}}
// Response: {"payload":{...}} or {"error":"...","code":"..."}
//
// The client's request context controls how long to wait — there
// is no server-side timeout cap on top.
func busAPIHandler(rt sdk.Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeBusJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var req struct {
			Topic   string          `json:"topic"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeBusJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
			return
		}
		if req.Topic == "" {
			writeBusJSON(w, http.StatusBadRequest, map[string]string{"error": "topic is required"})
			return
		}

		ctx := r.Context()
		correlationID := uuid.NewString()
		replyTo := req.Topic + ".reply." + correlationID

		replyCh := make(chan sdk.Message, 1)
		unsub, err := rt.SubscribeRaw(ctx, replyTo, func(msg sdk.Message) {
			select {
			case replyCh <- msg:
			default:
			}
		})
		if err != nil {
			writeBusJSON(w, http.StatusInternalServerError, map[string]string{"error": "subscribe: " + err.Error()})
			return
		}
		defer unsub()

		pubCtx := transport.WithPublishMeta(ctx, correlationID, replyTo)
		if _, err := rt.PublishRaw(pubCtx, req.Topic, req.Payload); err != nil {
			writeBusJSON(w, http.StatusBadGateway, map[string]string{"error": "publish: " + err.Error()})
			return
		}

		select {
		case msg := <-replyCh:
			writeBusJSON(w, http.StatusOK, map[string]json.RawMessage{"payload": msg.Payload})
		case <-ctx.Done():
			writeBusJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "timeout waiting for response"})
		}
	}
}

// busAPIStreamHandler handles POST /api/stream — bus publish +
// stream every reply event as NDJSON. Each intermediate reply
// (`done=false`) writes one line and flushes; the terminal reply
// (`done=true`) writes last and closes the response.
func busAPIStreamHandler(rt sdk.Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeBusJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var req struct {
			Topic   string          `json:"topic"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			writeBusJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
			return
		}
		if req.Topic == "" {
			writeBusJSON(w, http.StatusBadRequest, map[string]string{"error": "topic is required"})
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeBusJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
			return
		}

		ctx := r.Context()
		correlationID := uuid.NewString()
		replyTo := req.Topic + ".reply." + correlationID

		eventCh := make(chan sdk.Message, 100)
		unsub, err := rt.SubscribeRaw(ctx, replyTo, func(msg sdk.Message) {
			select {
			case eventCh <- msg:
			default:
			}
		})
		if err != nil {
			writeBusJSON(w, http.StatusInternalServerError, map[string]string{"error": "subscribe: " + err.Error()})
			return
		}
		defer unsub()

		pubCtx := transport.WithPublishMeta(ctx, correlationID, replyTo)
		if _, err := rt.PublishRaw(pubCtx, req.Topic, req.Payload); err != nil {
			writeBusJSON(w, http.StatusBadGateway, map[string]string{"error": "publish: " + err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		enc := json.NewEncoder(w)
		for {
			select {
			case msg := <-eventCh:
				done := msg.Metadata != nil && msg.Metadata["done"] == "true"
				_ = enc.Encode(map[string]any{
					"payload": json.RawMessage(msg.Payload),
					"done":    done,
				})
				flusher.Flush()
				if done {
					return
				}
			case <-ctx.Done():
				_ = enc.Encode(map[string]string{"error": "timeout"})
				flusher.Flush()
				return
			}
		}
	}
}

func writeBusJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
