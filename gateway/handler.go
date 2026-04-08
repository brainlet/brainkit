package gateway

import (
	"context"
	"net/http"

	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/sdk"
)

func (gw *Gateway) handleRequest(w http.ResponseWriter, r *http.Request, matched *route, pathParams map[string]string) {
	payload, err := buildPayload(r, matched, pathParams)
	if err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	reqID := requestID(r)
	replyTo := matched.Topic + ".reply." + reqID

	ctx, cancel := context.WithTimeout(r.Context(), gw.config.Timeout)
	defer cancel()

	replyCh := make(chan sdk.Message, 1)
	unsub, err := gw.rt.SubscribeRaw(ctx, replyTo, func(msg sdk.Message) {
		select {
		case replyCh <- msg:
		default:
		}
	})
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer unsub()

	pubCtx := transport.WithPublishMeta(ctx, reqID, replyTo)
	if _, err := gw.rt.PublishRaw(pubCtx, matched.Topic, payload); err != nil {
		http.Error(w, "publish failed", http.StatusBadGateway)
		return
	}

	select {
	case msg := <-replyCh:
		status := http.StatusOK
		if matched.Config.statusMapper != nil {
			status = matched.Config.statusMapper(msg.Payload, nil)
		} else {
			status = mapHTTPStatus(msg.Payload, nil)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		// Sanitize error responses before sending to HTTP clients
		if status >= 400 {
			w.Write(sanitizeErrorPayload(msg.Payload))
		} else {
			w.Write(msg.Payload)
		}
	case <-ctx.Done():
		if r.Context().Err() != nil {
			return
		}
		http.Error(w, `{"error":"gateway timeout"}`, http.StatusGatewayTimeout)
	}
}
