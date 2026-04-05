package scheduling

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testScheduleFireOnTransport — schedule fires and delivers payload.
// Ported from adversarial/backend_advanced_test.go:TestBackendAdvanced_ScheduleFire.
func testScheduleFireOnTransport(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fired := make(chan []byte, 1)
	unsub, err := env.Kernel.SubscribeRaw(ctx, "sched.transport.fire.suite", func(m messages.Message) {
		fired <- m.Payload
	})
	require.NoError(t, err)
	defer unsub()

	_, err = env.Kernel.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "in 200ms",
		Topic:      "sched.transport.fire.suite",
		Payload:    json.RawMessage(`{"scheduled":"suite"}`),
	})
	require.NoError(t, err)

	select {
	case p := <-fired:
		assert.Contains(t, string(p), "suite")
	case <-ctx.Done():
		t.Fatal("schedule didn't fire")
	}
}
