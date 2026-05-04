package persistence

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/require"
)

func callPersistService(t *testing.T, rt sdk.CallerRuntime, ctx context.Context, source, topic string, payload json.RawMessage) json.RawMessage {
	t.Helper()
	resp, err := sdk.Call[sdk.CustomMsg, json.RawMessage](rt, ctx, sdk.CustomMsg{
		Topic:   persistenceServiceTopic(source, topic),
		Payload: payload,
	}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	return resp
}

func persistenceServiceTopic(source, topic string) string {
	name := strings.TrimSuffix(source, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}
