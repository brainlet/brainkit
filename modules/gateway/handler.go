package gateway

import (
	"context"
	"errors"
	"net/http"

	"github.com/brainlet/brainkit/sdk"
)

// callerHolder is implemented by *brainkit.Kit so the gateway can acquire
// the shared-inbox Caller without importing the brainkit package (import
// cycle avoidance).
type callerHolder interface {
	Caller() *sdk.Caller
}

func (gw *Gateway) handleRequest(w http.ResponseWriter, r *http.Request, matched *route, pathParams map[string]string) {
	payload, err := buildPayload(r, matched, pathParams)
	if err != nil {
		writePayloadReadError(w, err)
		return
	}

	holder, ok := gw.rt.(callerHolder)
	if !ok {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	c := holder.Caller()
	if c == nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), gw.config.Timeout)
	defer cancel()

	reply, err := c.Call(ctx, matched.Topic, payload, sdk.CallerConfig{})
	if err != nil {
		var tErr *sdk.CallTimeoutError
		if errors.As(err, &tErr) {
			if r.Context().Err() != nil {
				return
			}
			http.Error(w, `{"error":"gateway timeout"}`, http.StatusGatewayTimeout)
			return
		}
		var cErr *sdk.CallCancelledError
		if errors.As(err, &cErr) {
			return
		}
		http.Error(w, "publish failed", http.StatusBadGateway)
		return
	}

	status := http.StatusOK
	if matched.Config.statusMapper != nil {
		status = matched.Config.statusMapper(reply, nil)
	} else {
		status = mapHTTPStatus(reply, nil)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if status >= 400 {
		w.Write(sanitizeErrorPayload(reply))
	} else {
		w.Write(reply)
	}
}
