package scheduling

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
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
	unsub, err := env.Kit.SubscribeRaw(ctx, "sched.transport.fire.suite", func(m sdk.Message) {
		fired <- m.Payload
	})
	require.NoError(t, err)
	defer unsub()

	testutil.Schedule(t, env.Kit, "in 200ms", "sched.transport.fire.suite", json.RawMessage(`{"scheduled":"suite"}`))

	select {
	case p := <-fired:
		assert.Contains(t, string(p), "suite")
	case <-ctx.Done():
		t.Fatal("schedule didn't fire")
	}
}
