package gateway

import "net/http"

func (gw *Gateway) handleWebhook(w http.ResponseWriter, r *http.Request, matched *route, pathParams map[string]string) {
	payload, err := buildPayload(r, matched, pathParams)
	if err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := gw.rt.PublishRaw(r.Context(), matched.Topic, payload); err != nil {
		http.Error(w, "publish failed", http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}
