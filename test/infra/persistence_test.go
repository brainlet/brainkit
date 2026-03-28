package infra_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistence_DeploySurvivesRestart(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	// Kernel 1: deploy a service
	store1, err := kit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k1, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "test",
		CallerID:  "test",
		Store:     store1,
	})
	require.NoError(t, err)

	ctx := context.Background()
	pr, err := sdk.Publish(k1, ctx, messages.KitDeployMsg{
		Source: "greeter.ts",
		Code:   `bus.on("greet", (msg) => { msg.reply({ hello: "world" }); });`,
	})
	require.NoError(t, err)
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](k1, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()

	// Verify service works
	time.Sleep(100 * time.Millisecond)
	sendPR, _ := sdk.SendToService(k1, ctx, "greeter.ts", "greet", map[string]bool{"x": true})
	replyCh := make(chan bool, 1)
	replyUnsub, _ := k1.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) { replyCh <- true })
	select {
	case <-replyCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
	replyUnsub()

	// Close Kernel 1
	k1.Close()

	// Kernel 2: same store — service should auto-redeploy
	store2, err := kit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k2, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "test",
		CallerID:  "test",
		Store:     store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	// Service should be running
	time.Sleep(200 * time.Millisecond)
	sendPR2, _ := sdk.SendToService(k2, ctx, "greeter.ts", "greet", map[string]bool{"x": true})
	replyCh2 := make(chan bool, 1)
	replyUnsub2, _ := k2.SubscribeRaw(ctx, sendPR2.ReplyTo, func(msg messages.Message) { replyCh2 <- true })
	defer replyUnsub2()

	select {
	case <-replyCh2:
		// auto-redeployed and responded
	case <-time.After(5 * time.Second):
		t.Fatal("redeployed service did not respond")
	}
}

func TestPersistence_TeardownRemovesFromStore(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")
	store, _ := kit.NewSQLiteStore(storePath)

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "test",
		CallerID:  "test",
		Store:     store,
	})
	require.NoError(t, err)

	ctx := context.Background()
	pr, _ := sdk.Publish(k, ctx, messages.KitDeployMsg{Source: "temp.ts", Code: `bus.on("x", (m) => m.reply({}));`})
	deployCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { deployCh <- struct{}{} })
	<-deployCh
	unsub()

	// Teardown
	tpr, _ := sdk.Publish(k, ctx, messages.KitTeardownMsg{Source: "temp.ts"})
	tdCh := make(chan struct{}, 1)
	tunsub, _ := sdk.SubscribeTo[messages.KitTeardownResp](k, ctx, tpr.ReplyTo, func(_ messages.KitTeardownResp, _ messages.Message) { tdCh <- struct{}{} })
	<-tdCh
	tunsub()
	k.Close()

	// Kernel 2: should have NO deployments
	store2, _ := kit.NewSQLiteStore(storePath)
	k2, _ := kit.NewKernel(kit.KernelConfig{
		Namespace: "test",
		CallerID:  "test",
		Store:     store2,
	})
	defer k2.Close()

	deployments := k2.ListDeployments()
	assert.Empty(t, deployments, "torn-down deployment should not persist")
}

func TestPersistence_OrderPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")
	store, _ := kit.NewSQLiteStore(storePath)

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "test",
		CallerID:  "test",
		Store:     store,
	})
	require.NoError(t, err)

	ctx := context.Background()
	for _, name := range []string{"first.ts", "second.ts", "third.ts"} {
		pr, _ := sdk.Publish(k, ctx, messages.KitDeployMsg{
			Source: name,
			Code:   `bus.on("ping", (m) => m.reply({}));`,
		})
		ch := make(chan struct{}, 1)
		unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { ch <- struct{}{} })
		<-ch
		unsub()
	}
	k.Close()

	// Verify order in store
	store2, _ := kit.NewSQLiteStore(storePath)
	deps, _ := store2.LoadDeployments()
	store2.Close()

	require.Len(t, deps, 3)
	assert.Equal(t, "first.ts", deps[0].Source)
	assert.Equal(t, "second.ts", deps[1].Source)
	assert.Equal(t, "third.ts", deps[2].Source)
	assert.Less(t, deps[0].Order, deps[1].Order)
	assert.Less(t, deps[1].Order, deps[2].Order)
}

func TestPersistence_FailedRedeployDoesNotBlock(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")
	store, _ := kit.NewSQLiteStore(storePath)

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "test",
		CallerID:  "test",
		Store:     store,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Deploy a working service
	pr1, _ := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "good.ts",
		Code:   `bus.on("ping", (msg) => { msg.reply({ ok: true }); });`,
	})
	ch1 := make(chan struct{}, 1)
	u1, _ := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, pr1.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) { ch1 <- struct{}{} })
	<-ch1
	u1()

	// Persist a broken deployment directly into the store
	store.SaveDeployment(kit.PersistedDeployment{
		Source: "broken.ts", Code: `throw new Error("intentional failure");`,
		Order: 99, DeployedAt: time.Now(),
	})

	k.Close()

	// Kernel 2: should start even though broken.ts fails to redeploy
	store2, _ := kit.NewSQLiteStore(storePath)
	k2, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "test",
		CallerID:  "test",
		Store:     store2,
	})
	require.NoError(t, err, "Kernel should start even with a broken persisted deployment")
	defer k2.Close()

	// The good service should still work
	time.Sleep(200 * time.Millisecond)
	sendPR, _ := sdk.SendToService(k2, ctx, "good.ts", "ping", map[string]bool{"x": true})
	replyCh := make(chan bool, 1)
	replyUnsub, _ := k2.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) { replyCh <- true })
	defer replyUnsub()

	select {
	case <-replyCh:
		// good.ts works despite broken.ts failure
	case <-time.After(5 * time.Second):
		t.Fatal("good.ts should work even when broken.ts fails to redeploy")
	}
}
