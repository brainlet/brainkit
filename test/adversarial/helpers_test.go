package adversarial_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
)

// sendAndReceive publishes a typed message and waits for the raw response.
func sendAndReceive(t *testing.T, rt sdk.Runtime, msg messages.BrainkitMessage, timeout time.Duration) (json.RawMessage, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, msg)
	if err != nil {
		t.Logf("publish failed: %v", err)
		return nil, false
	}

	ch := make(chan json.RawMessage, 1)
	unsub, err := rt.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	if err != nil {
		t.Logf("subscribe failed: %v", err)
		return nil, false
	}
	defer unsub()

	select {
	case payload := <-ch:
		return payload, true
	case <-ctx.Done():
		return nil, false
	}
}

// responseCode extracts the error code from a bus response payload.
func responseCode(payload json.RawMessage) string {
	var resp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	json.Unmarshal(payload, &resp)
	return resp.Code
}

// responseHasError checks if a bus response contains an error field.
func responseHasError(payload json.RawMessage) bool {
	var resp struct {
		Error string `json:"error"`
	}
	json.Unmarshal(payload, &resp)
	return resp.Error != ""
}
