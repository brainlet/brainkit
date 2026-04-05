package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/google/uuid"
	"github.com/coder/websocket"
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

		reqID := uuid.NewString()
		replyTo := matched.Topic + ".reply." + reqID

		replyCh := make(chan messages.Message, 1)
		unsub, subErr := gw.rt.SubscribeRaw(ctx, replyTo, func(msg messages.Message) {
			select {
			case replyCh <- msg:
			default:
			}
		})
		if subErr != nil {
			continue
		}

		pubCtx := messaging.WithPublishMeta(ctx, reqID, replyTo)
		if _, pubErr := gw.rt.PublishRaw(pubCtx, matched.Topic, payload); pubErr != nil {
			unsub()
			continue
		}

		select {
		case msg := <-replyCh:
			unsub()
			conn.Write(ctx, websocket.MessageText, msg.Payload)
		case <-ctx.Done():
			unsub()
			return
		}
	}
}
