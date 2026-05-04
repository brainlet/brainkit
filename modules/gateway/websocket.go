package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/brainlet/brainkit/sdk"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

func (gw *Gateway) handleWebSocket(w http.ResponseWriter, r *http.Request, matched *route, pathParams map[string]string) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		gw.logger.Error("websocket accept error", slog.String("error", err.Error()))
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	sessionID := uuid.NewString()
	ctx := r.Context()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			disconnectPayload, _ := json.Marshal(map[string]any{
				"sessionId": sessionID,
				"type":      "disconnect",
			})
			gw.rt.PublishRaw(context.Background(), matched.Topic, disconnectPayload)
			return
		}

		var msgData json.RawMessage = data
		payload, _ := json.Marshal(map[string]any{
			"sessionId": sessionID,
			"data":      msgData,
			"type":      "message",
		})

		caller := gw.requestCaller()
		if caller == nil {
			continue
		}
		reply, callErr := caller.Call(ctx, matched.Topic, payload, sdk.CallerConfig{})
		if callErr != nil {
			continue
		}
		if err := conn.Write(ctx, websocket.MessageText, reply); err != nil {
			return
		}
	}
}
