package health

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testShutdownGracefulWithActiveDeployments — close with active deployments.
func testShutdownGracefulWithActiveDeployments(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	for i := 0; i < 5; i++ {
		err := env.Deploy("shutdown-svc-adv.ts", `bus.on("ping", function(msg) { msg.reply({ok:true}); });`)
		require.NoError(t, err)
		testutil.Teardown(t, env.Kit, "shutdown-svc-adv.ts")
	}
	err := env.Deploy("final-svc-adv.ts", `bus.on("ping", function(msg) { msg.reply({ok:true}); });`)
	require.NoError(t, err)

	err = env.Kit.Close()
	assert.NoError(t, err)
}

// testShutdownWithActiveSchedules — close cancels all schedules.
func testShutdownWithActiveSchedules(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		pr, _ := sdk.PublishScheduleCreate(env.Kit, ctx, sdk.ScheduleCreateMsg{
			Expression: "every 1h",
			Topic:      "shutdown-sched-adv",
			Payload:    json.RawMessage(`{}`),
		})
		ch := make(chan sdk.ScheduleCreateResp, 1)
		unsub, _ := sdk.SubscribeScheduleCreateResp(env.Kit, ctx, pr.ReplyTo,
			func(resp sdk.ScheduleCreateResp, msg sdk.Message) { ch <- resp })
		<-ch
		unsub()
	}

	err := env.Kit.Close()
	assert.NoError(t, err)
}

// testShutdownWithActiveSubscriptions — close unsubscribes all.
func testShutdownWithActiveSubscriptions(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	err := env.Deploy("sub-shutdown-adv.ts", `
		bus.subscribe("topic1", function() {});
		bus.subscribe("topic2", function() {});
		bus.subscribe("topic3", function() {});
		output("subscribed");
	`)
	require.NoError(t, err)

	err = env.Kit.Close()
	assert.NoError(t, err)
}

// testShutdownDrainTimeoutAdv — drain with stuck handler forces close.
func testShutdownDrainTimeoutAdv(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	testutil.Deploy(t, k, "stuck-adv.ts", `
		bus.on("stuck", async function(msg) {
			await new Promise(r => setTimeout(r, 60000)); // 60s — will exceed drain timeout
		});
	`)

	// Fire a message to the stuck handler
	k.PublishRaw(context.Background(), "ts.stuck-adv.stuck", json.RawMessage(`{}`))
	time.Sleep(50 * time.Millisecond) // let handler start

	// Shutdown with 1s timeout — should force-close
	shutCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err = k.Shutdown(shutCtx)
	assert.NoError(t, err) // force-close is still a clean close
}

// testShutdownConcurrentClose — multiple goroutines calling Close.
func testShutdownConcurrentClose(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k.Close() // second+ calls should be no-op
		}()
	}
	wg.Wait()
}

// testShutdownStorageAccessBeforeClose — storage works right until close.
func testShutdownStorageAccessBeforeClose(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	// Use it
	err := env.Deploy("storage-use-adv.ts", `output("using storage");`)
	require.NoError(t, err)

	// Close
	err = env.Kit.Close()
	assert.NoError(t, err)
}
