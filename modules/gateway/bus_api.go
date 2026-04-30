package gateway

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk"
)

// registerBusAPIRoutes adds the built-in POST /api/bus and
// POST /api/stream handlers to mux. The bus API exposes the Kit's
// request→reply bus surface over HTTP — the canonical external
// entry point used by the brainkit CLI and any downstream tool
// that wants to drive a running Kit without linking against
// brainkit directly.
//
// Disable via Config.NoBusAPI=true.
func registerBusAPIRoutes(mux *http.ServeMux, caller bkmodule.RequestCaller) {
	mux.HandleFunc("POST /api/bus", busAPIHandler(caller))
	mux.HandleFunc("POST /api/stream", busAPIStreamHandler(caller))
}

// busAPIHandler handles POST /api/bus — generic bus request-reply
// over HTTP.
//
// Body:     {"topic":"kit.health","payload":{...}}
// Response: {"payload":{...}} or {"error":"...","code":"..."}
//
// The client's request context controls how long to wait — there
// is no server-side timeout cap on top.
func busAPIHandler(caller bkmodule.RequestCaller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			if isRequestBodyTooLarge(err) {
				writeBusJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
				return
			}
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
		if caller == nil {
			writeBusJSON(w, http.StatusInternalServerError, map[string]string{"error": "caller is not configured"})
			return
		}

		reply, err := caller.Call(r.Context(), req.Topic, req.Payload, sdk.CallerConfig{})
		if err != nil {
			var timeoutErr *sdk.CallTimeoutError
			if errors.As(err, &timeoutErr) {
				writeBusJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "timeout waiting for response"})
				return
			}
			var cancelledErr *sdk.CallCancelledError
			if errors.As(err, &cancelledErr) {
				writeBusJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "timeout waiting for response"})
				return
			}
			writeBusJSON(w, http.StatusBadGateway, map[string]string{"error": "call: " + err.Error()})
			return
		}
		writeBusJSON(w, http.StatusOK, map[string]json.RawMessage{"payload": reply})
	}
}

// busAPIStreamHandler handles POST /api/stream — bus publish +
// stream every reply event as NDJSON. Each intermediate reply
// (`done=false`) writes one line and flushes; the terminal reply
// (`done=true`) writes last and closes the response.
func busAPIStreamHandler(caller bkmodule.RequestCaller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			if isRequestBodyTooLarge(err) {
				writeBusJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
				return
			}
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
		if caller == nil {
			writeBusJSON(w, http.StatusInternalServerError, map[string]string{"error": "caller is not configured"})
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeBusJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
			return
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		enc := json.NewEncoder(w)
		reply, err := caller.Call(r.Context(), req.Topic, req.Payload, sdk.CallerConfig{
			StreamHandler: func(msg sdk.Message) error {
				if err := enc.Encode(map[string]any{
					"payload": json.RawMessage(msg.Payload),
					"done":    false,
				}); err != nil {
					return err
				}
				flusher.Flush()
				return nil
			},
		})
		if err != nil {
			_ = enc.Encode(map[string]string{"error": "call: " + err.Error()})
			flusher.Flush()
			return
		}
		_ = enc.Encode(map[string]any{
			"payload": reply,
			"done":    true,
		})
		flusher.Flush()
	}
}

func writeBusJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
