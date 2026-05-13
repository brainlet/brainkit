package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	jsruntimemod "github.com/brainlet/brainkit/modules/jsruntime"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/require"
)

func TestJSRuntimeUnmountCancelsActiveAsyncHandlerAndRemounts(t *testing.T) {
	k := newRemountTestKit(t, Config{Transport: Memory()})
	ctx := context.Background()

	require.NoError(t, k.Mount(ctx, jsruntimemod.New()))
	assertLifecycleComponentMounted(t, k, "jsruntime")
	deployCtx, deployCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer deployCancel()
	_, err := k.kernel.Deploy(deployCtx, "runtime-unmount-active.ts", `
		bus.on("hold", async (msg) => {
			await new Promise((resolve) => setTimeout(resolve, 5000));
			msg.reply({ ok: true });
		});
	`)
	require.NoError(t, err)

	callCtx, callCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer callCancel()
	callDone := make(chan error, 1)
	go func() {
		_, err := Call[sdk.CustomMsg, json.RawMessage](k, callCtx, sdk.CustomMsg{
			Topic:   "ts.runtime-unmount-active.hold",
			Payload: json.RawMessage(`{}`),
		}, WithCallTimeout(5*time.Second))
		callDone <- err
	}()

	require.Eventually(t, func() bool {
		snapshot := k.lifecycleDebugSnapshot().Runtime
		return snapshot.ActiveHandlers == 1 && snapshot.Transport.CallerPendingCalls == 1
	}, 2*time.Second, 20*time.Millisecond)

	unmountCtx, unmountCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer unmountCancel()
	require.NoError(t, k.Unmount(unmountCtx, "jsruntime"))
	require.False(t, k.kernel.HasJSRuntime())
	requireNoLifecycleComponent(t, k.lifecycleDebugSnapshot(), "jsruntime")
	require.Eventually(t, func() bool {
		return k.lifecycleDebugSnapshot().Runtime.ActiveHandlers == 0
	}, 2*time.Second, 20*time.Millisecond)

	callCancel()
	select {
	case err := <-callDone:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("in-flight call did not finish after runtime unmount and caller cancellation")
	}
	requireLifecycleNoTransientWork(t, k)

	require.NoError(t, k.Mount(ctx, jsruntimemod.New()))
	assertLifecycleComponentMounted(t, k, "jsruntime")
	remountDeployCtx, remountDeployCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer remountDeployCancel()
	_, err = k.kernel.Deploy(remountDeployCtx, "runtime-remount-fresh.ts", `
		bus.on("ping", (msg) => {
			msg.reply({ pong: msg.payload.value + "-fresh" });
		});
	`)
	require.NoError(t, err)
	reply, err := Call[sdk.CustomMsg, map[string]string](k, ctx, sdk.CustomMsg{
		Topic:   "ts.runtime-remount-fresh.ping",
		Payload: json.RawMessage(`{"value":"remount"}`),
	}, WithCallTimeout(2*time.Second))
	require.NoError(t, err)
	require.Equal(t, "remount-fresh", reply["pong"])
	requireLifecycleNoTransientWork(t, k)
}
