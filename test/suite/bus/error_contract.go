package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBusErrorResponseCarriesCode(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	pr, err := sdk.Publish(env.Kit, ctx, messages.ToolCallMsg{Name: "nonexistent-tool"})
	require.NoError(t, err)

	ch := make(chan json.RawMessage, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
		ch <- json.RawMessage(msg.Payload)
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		var resp struct {
			Error   string         `json:"error"`
			Code    string         `json:"code"`
			Details map[string]any `json:"details"`
		}
		require.NoError(t, json.Unmarshal(payload, &resp))
		assert.NotEmpty(t, resp.Error)
		assert.Equal(t, "NOT_FOUND", resp.Code)
		if resp.Details != nil {
			assert.Equal(t, "nonexistent-tool", resp.Details["name"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for error response")
	}
}

func testResultMetaIncludesCode(t *testing.T, _ *suite.TestEnv) {
	meta := messages.ResultMeta{Error: "not found", Code: "NOT_FOUND", Details: map[string]any{"resource": "tool"}}
	data, err := json.Marshal(meta)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "NOT_FOUND", decoded["code"])
	assert.Equal(t, "not found", decoded["error"])
}
