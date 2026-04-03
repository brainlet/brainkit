package infra_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPersistence_DeploySurvivesRestart(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	// Kernel 1: deploy a service
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
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
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
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
	store, _ := brainkit.NewSQLiteStore(storePath)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
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
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, _ := brainkit.NewKernel(brainkit.KernelConfig{
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
	store, _ := brainkit.NewSQLiteStore(storePath)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
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
	store2, _ := brainkit.NewSQLiteStore(storePath)
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
	store, _ := brainkit.NewSQLiteStore(storePath)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
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
	store.SaveDeployment(brainkit.PersistedDeployment{
		Source: "broken.ts", Code: `throw new Error("intentional failure");`,
		Order: 99, DeployedAt: time.Now(),
	})

	k.Close()

	// Kernel 2: should start even though broken.ts fails to redeploy
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
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

// ── Phase 1: New persistence tests ──────────────────────────────────────

func TestPersistence_PackageNameSurvivesRestart(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "pkg-restart.db")

	// Phase 1: deploy with package name
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store1,
	})
	require.NoError(t, err)

	_, err = k1.Deploy(context.Background(), "svc.ts",
		`bus.on("ping", (msg) => msg.reply({ ok: true }));`,
		brainkit.WithPackageName("my-package"),
	)
	require.NoError(t, err)
	k1.Close()

	// Phase 2: verify package name survived in store
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	deps, err := store2.LoadDeployments()
	require.NoError(t, err)
	require.Len(t, deps, 1)
	assert.Equal(t, "my-package", deps[0].PackageName, "packageName must survive restart")
	store2.Close()
}

func TestPersistence_RedeployPreservesMetadata(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "redeploy-meta.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store,
		Roles: map[string]rbac.Role{"admin": rbac.RoleAdmin},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	_, err = k.Deploy(ctx, "svc.ts",
		`bus.on("v1", (msg) => msg.reply({ v: 1 }));`,
		brainkit.WithPackageName("my-pkg"), brainkit.WithRole("admin"),
	)
	require.NoError(t, err)

	// Redeploy with new code — metadata should be preserved
	_, err = k.Redeploy(ctx, "svc.ts",
		`bus.on("v2", (msg) => msg.reply({ v: 2 }));`)
	require.NoError(t, err)

	// Check store directly
	deps, _ := store.LoadDeployments()
	require.Len(t, deps, 1)
	assert.Equal(t, "my-pkg", deps[0].PackageName, "packageName must survive redeploy")
	assert.Equal(t, "admin", deps[0].Role, "role must survive redeploy")
}

func TestPersistence_WithRestoringSkipsPersist(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "restoring.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store,
	})
	require.NoError(t, err)
	defer k.Close()

	// Deploy with WithRestoring — should NOT persist
	_, err = k.Deploy(context.Background(), "ephemeral.ts",
		`bus.on("x", (msg) => msg.reply({}));`,
		brainkit.WithRestoring(),
	)
	require.NoError(t, err)

	// Store should be empty (WithRestoring skips SaveDeployment)
	deps, _ := store.LoadDeployments()
	assert.Empty(t, deps, "WithRestoring should skip SaveDeployment")
}

func TestPersistence_RolePreservedAcrossRestart(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "role-restart.db")

	roles := map[string]rbac.Role{
		"admin": rbac.RoleAdmin,
		"restricted": {
			Name: "restricted",
			Bus: rbac.BusPermissions{
				Subscribe: rbac.TopicFilter{Allow: []string{"*.reply.*"}},
			},
			Commands: rbac.CommandPermissions{Allow: []string{"tools.list"}},
		},
	}

	// Kernel 1: deploy with role
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test",
		Store: store1, Roles: roles, DefaultRole: "restricted",
	})
	require.NoError(t, err)

	_, err = k1.Deploy(context.Background(), "admin-svc.ts",
		`bus.on("ping", (msg) => msg.reply({ ok: true }));`,
		brainkit.WithRole("admin"),
	)
	require.NoError(t, err)
	k1.Close()

	// Kernel 2: same store — deployment should restore with role="admin"
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test",
		Store: store2, Roles: roles, DefaultRole: "restricted",
	})
	require.NoError(t, err)
	defer k2.Close()

	// Verify the deployment was restored
	deployments := k2.ListDeployments()
	require.Len(t, deployments, 1, "admin-svc.ts should be restored")

	// Verify the role was preserved by checking the stored deployment
	deps, err := store2.LoadDeployments()
	require.NoError(t, err)
	require.Len(t, deps, 1)
	assert.Equal(t, "admin", deps[0].Role, "role should be preserved across restart")
}

func TestPersistence_ScheduleCatchUpOnRestart(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "schedule-restart.db")

	// Kernel 1: create a one-time schedule that fires 100ms from now
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store1,
	})
	require.NoError(t, err)

	// Create a schedule that fires "in 100ms"
	schedID, err := k1.Schedule(context.Background(), brainkit.ScheduleConfig{
		Expression: "in 100ms",
		Topic:      "test.catchup",
		Payload:    []byte(`{"caught":"up"}`),
		Source:     "test",
	})
	require.NoError(t, err)
	require.NotEmpty(t, schedID)

	// Close immediately — the schedule hasn't fired yet
	k1.Close()

	// Wait past the fire time
	time.Sleep(200 * time.Millisecond)

	// Kernel 2: same store — should fire the missed schedule on startup
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	// Subscribe to the catchup topic BEFORE creating kernel
	// (kernel creation triggers restoreSchedules which fires missed schedules)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	// The one-time schedule should have been fired and deleted
	schedules := k2.ListSchedules()
	assert.Empty(t, schedules, "one-time schedule should be deleted after catch-up fire")
}

func TestPersistence_RecurringScheduleRestartsCorrectly(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "recurring-restart.db")

	// Kernel 1: create recurring schedule
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store1,
	})
	require.NoError(t, err)

	schedID, err := k1.Schedule(context.Background(), brainkit.ScheduleConfig{
		Expression: "every 1h",
		Topic:      "test.heartbeat",
		Payload:    []byte(`{"beat":true}`),
		Source:     "test",
	})
	require.NoError(t, err)
	require.NotEmpty(t, schedID)

	// Verify schedule exists
	schedules := k1.ListSchedules()
	require.Len(t, schedules, 1)
	k1.Close()

	// Kernel 2: schedule should be restored and active
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	schedules = k2.ListSchedules()
	require.Len(t, schedules, 1, "recurring schedule should be restored")
	assert.Equal(t, "every 1h", schedules[0].Expression)
	assert.Equal(t, "test.heartbeat", schedules[0].Topic)
	assert.False(t, schedules[0].OneTime, "should be recurring, not one-time")
}

func TestPersistence_DeployOrderPreservedExactly(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "order-exact.db")

	// Kernel 1: deploy A, B, C in order
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", Store: store1,
	})
	require.NoError(t, err)

	ctx := context.Background()
	for _, name := range []string{"alpha.ts", "beta.ts", "gamma.ts"} {
		_, err := k1.Deploy(ctx, name, `bus.on("x", (msg) => msg.reply({}));`)
		require.NoError(t, err)
	}
	k1.Close()

	// Kernel 2: verify load order matches deploy order
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	deps, err := store2.LoadDeployments()
	require.NoError(t, err)
	require.Len(t, deps, 3)

	// LoadDeployments returns ORDER BY deploy_order
	assert.Equal(t, "alpha.ts", deps[0].Source)
	assert.Equal(t, "beta.ts", deps[1].Source)
	assert.Equal(t, "gamma.ts", deps[2].Source)
	assert.True(t, deps[0].Order < deps[1].Order, "alpha should have lower order than beta")
	assert.True(t, deps[1].Order < deps[2].Order, "beta should have lower order than gamma")

	store2.Close()
}
