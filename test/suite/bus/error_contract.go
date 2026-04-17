package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBusErrorResponseCarriesCode(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.ToolCallMsg{Name: "nonexistent-tool"})
	require.NoError(t, err)

	ch := make(chan json.RawMessage, 1)
	unsub, err := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(msg sdk.Message) {
		ch <- json.RawMessage(msg.Payload)
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		assert.True(t, suite.ResponseHasError(payload))
		assert.Equal(t, "NOT_FOUND", suite.ResponseCode(payload))
		assert.NotEmpty(t, suite.ResponseErrorMessage(payload))
		if d := suite.ResponseErrorDetails(payload); d != nil {
			assert.Equal(t, "nonexistent-tool", d["name"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for error response")
	}
}

