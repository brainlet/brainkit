package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/brainlet/brainkit/sdk"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// handleWebSocketAudio opens a duplex binary WebSocket and
// wires it to a pair of bus topics:
//
//   - inbound frames (text or binary) publish on route.Topic
//     with payload {sessionId, mime, bytes_b64, text?}.
//   - every message received on route.OutTopic is written back
//     to the client as a binary frame (or text if the payload
//     is a JSON string).
//
// Designed for continuous bidirectional audio (realtime voice)
// — the connection stays open until either side drops.
func (gw *Gateway) handleWebSocketAudio(w http.ResponseWriter, r *http.Request, matched *route) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		gw.logger.Error("websocket-audio accept error", slog.String("error", err.Error()))
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	// Unlimited read — audio payloads can be bigger than the
	// default 32 KiB limit.
	conn.SetReadLimit(-1)

	sessionID := uuid.NewString()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Tell the .ts side we connected so it can initialize a
	// realtime session for this client.
	connectPayload, _ := json.Marshal(map[string]any{
		"sessionId": sessionID,
		"type":      "connect",
	})
	if _, err := gw.rt.PublishRaw(context.Background(), matched.Topic, connectPayload); err != nil {
		gw.logger.Error("websocket-audio publish connect", slog.String("error", err.Error()))
	}

	unsub, err := gw.rt.SubscribeRaw(ctx, matched.OutTopic+"."+sessionID, func(msg sdk.Message) {
		// Outbound payload shape: {"binary":true, "bytes_b64":"..."} for audio,
		// or {"binary":false, "text":"..."} for JSON/control. Fall back to
		// writing the raw payload as a text frame when neither field is set.
		var out struct {
			Binary   bool   `json:"binary"`
			BytesB64 string `json:"bytes_b64"`
			Text     string `json:"text"`
		}
		if jerr := json.Unmarshal(msg.Payload, &out); jerr == nil && (out.BytesB64 != "" || out.Text != "") {
			if out.Binary && out.BytesB64 != "" {
				raw, derr := base64.StdEncoding.DecodeString(out.BytesB64)
				if derr == nil {
					_ = conn.Write(ctx, websocket.MessageBinary, raw)
					return
				}
			}
			if out.Text != "" {
				_ = conn.Write(ctx, websocket.MessageText, []byte(out.Text))
				return
			}
		}
		_ = conn.Write(ctx, websocket.MessageText, msg.Payload)
	})
	if err != nil {
		gw.logger.Error("websocket-audio subscribe", slog.String("error", err.Error()))
		return
	}
	defer unsub()

	for {
		typ, data, rerr := conn.Read(ctx)
		if rerr != nil {
			disconnect, _ := json.Marshal(map[string]any{
				"sessionId": sessionID,
				"type":      "disconnect",
			})
			_, _ = gw.rt.PublishRaw(context.Background(), matched.Topic, disconnect)
			return
		}

		var payload []byte
		if typ == websocket.MessageBinary {
			payload, _ = json.Marshal(map[string]any{
				"sessionId": sessionID,
				"type":      "audio",
				"binary":    true,
				"bytes_b64": base64.StdEncoding.EncodeToString(data),
			})
		} else {
			payload, _ = json.Marshal(map[string]any{
				"sessionId": sessionID,
				"type":      "text",
				"binary":    false,
				"text":      string(data),
			})
		}
		if _, err := gw.rt.PublishRaw(context.Background(), matched.Topic, payload); err != nil {
			gw.logger.Error("websocket-audio publish", slog.String("error", err.Error()))
		}
	}
}
