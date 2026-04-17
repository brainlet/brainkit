package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/internal/types"
	schedulesmod "github.com/brainlet/brainkit/modules/schedules"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// ── Deploy persistence ──────────────────────────────────────────────────

func testDeploySurvivesRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	// Kernel 1: deploy a service
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test",
		Store:     store1,
	})
	require.NoError(t, err)

	ctx := context.Background()
	testutil.Deploy(t, k1, "greeter-persist.ts", `bus.on("greet", (msg) => { msg.reply({ hello: "world" }); });`)

	// Verify service works
	time.Sleep(100 * time.Millisecond)
	sendPR, _ := sdk.SendToService(k1, ctx, "greeter-persist.ts", "greet", map[string]bool{"x": true})
	replyCh := make(chan bool, 1)
	replyUnsub, _ := k1.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) { replyCh <- true })
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

	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test",
		Store:     store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	// Service should be running
	time.Sleep(200 * time.Millisecond)
	sendPR2, _ := sdk.SendToService(k2, ctx, "greeter-persist.ts", "greet", map[string]bool{"x": true})
	replyCh2 := make(chan bool, 1)
	replyUnsub2, _ := k2.SubscribeRaw(ctx, sendPR2.ReplyTo, func(msg sdk.Message) { replyCh2 <- true })
	defer replyUnsub2()

	select {
	case <-replyCh2:
		// auto-redeployed and responded
	case <-time.After(5 * time.Second):
		t.Fatal("redeployed service did not respond")
	}
}

func testTeardownRemovesFromStore(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")
	store, _ := brainkit.NewSQLiteStore(storePath)

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test",
		Store:     store,
	})
	require.NoError(t, err)

	testutil.Deploy(t, k, "temp-persist.ts", `bus.on("x", (m) => m.reply({}));`)

	// Teardown
	testutil.Teardown(t, k, "temp-persist.ts")
	k.Close()

	// Kernel 2: should have NO deployments
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, _ := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test",
		Store:     store2,
	})
	defer k2.Close()

	deployments := testutil.ListDeployments(t, k2)
	assert.Empty(t, deployments, "torn-down deployment should not persist")
}

func testOrderPreserved(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")
	store, _ := brainkit.NewSQLiteStore(storePath)

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test",
		Store:     store,
	})
	require.NoError(t, err)

	for _, name := range []string{"first-persist.ts", "second-persist.ts", "third-persist.ts"} {
		testutil.Deploy(t, k, name, `bus.on("ping", (m) => m.reply({}));`)
	}
	k.Close()

	// Verify order in store
	store2, _ := brainkit.NewSQLiteStore(storePath)
	deps, _ := store2.LoadDeployments()
	store2.Close()

	require.Len(t, deps, 3)
	assert.Equal(t, "first-persist.ts", deps[0].Source)
	assert.Equal(t, "second-persist.ts", deps[1].Source)
	assert.Equal(t, "third-persist.ts", deps[2].Source)
	assert.Less(t, deps[0].Order, deps[1].Order)
	assert.Less(t, deps[1].Order, deps[2].Order)
}

func testFailedRedeployDoesNotBlock(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")
	store, _ := brainkit.NewSQLiteStore(storePath)

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test",
		Store:     store,
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Deploy a working service
	testutil.Deploy(t, k, "good-persist.ts", `bus.on("ping", (msg) => { msg.reply({ ok: true }); });`)

	// Persist a broken deployment directly into the store
	store.SaveDeployment(types.PersistedDeployment{
		Source: "broken-persist.ts", Code: `throw new Error("intentional failure");`,
		Order: 99, DeployedAt: time.Now(),
	})

	k.Close()

	// Kernel 2: should start even though broken-persist.ts fails to redeploy
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test",
		Store:     store2,
	})
	require.NoError(t, err, "Kernel should start even with a broken persisted deployment")
	defer k2.Close()

	// The good service should still work
	time.Sleep(200 * time.Millisecond)
	sendPR, _ := sdk.SendToService(k2, ctx, "good-persist.ts", "ping", map[string]bool{"x": true})
	replyCh := make(chan bool, 1)
	replyUnsub, _ := k2.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) { replyCh <- true })
	defer replyUnsub()

	select {
	case <-replyCh:
		// good-persist.ts works despite broken-persist.ts failure
	case <-time.After(5 * time.Second):
		t.Fatal("good-persist.ts should work even when broken-persist.ts fails to redeploy")
	}
}

// ── Metadata persistence ────────────────────────────────────────────────

func testPackageNameSurvivesRestart(t *testing.T, _ *suite.TestEnv) {
	storePath := filepath.Join(t.TempDir(), "pkg-restart.db")

	// Phase 1: deploy with package name
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store1,
	})
	require.NoError(t, err)

	err = testutil.DeployWithOpts(k1, "svc-persist.ts",
		`bus.on("ping", (msg) => msg.reply({ ok: true }));`,
		"my-package",
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

func testRedeployPreservesMetadata(t *testing.T, _ *suite.TestEnv) {
	storePath := filepath.Join(t.TempDir(), "redeploy-meta.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store,
	})
	require.NoError(t, err)
	defer k.Close()

	err = testutil.DeployWithOpts(k, "svc-redeploy-persist.ts",
		`bus.on("v1", (msg) => msg.reply({ v: 1 }));`,
		"my-pkg",
	)
	require.NoError(t, err)

	// Redeploy with new code — metadata (packageName) is preserved by
	// passing the same name explicitly. Under the Package-as-unit model,
	// packageName is part of the deploy request, not metadata inferred
	// from the source.
	err = testutil.DeployWithOpts(k, "svc-redeploy-persist.ts",
		`bus.on("v2", (msg) => msg.reply({ v: 2 }));`,
		"my-pkg",
	)
	require.NoError(t, err)

	// Check store directly
	deps, _ := store.LoadDeployments()
	require.Len(t, deps, 1)
	assert.Equal(t, "my-pkg", deps[0].PackageName, "packageName must survive redeploy")
}

func testWithRestoringSkipsPersist(t *testing.T, _ *suite.TestEnv) {
	storePath := filepath.Join(t.TempDir(), "restoring.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store,
	})
	require.NoError(t, err)
	defer k.Close()

	// WithRestoring is an internal flag (not exposed via bus). Bus deploy always persists.
	// This test now verifies that a bus deploy DOES persist (the inverse of the old test).
	// The internal WithRestoring behavior is tested at the engine level.
	testutil.Deploy(t, k, "ephemeral-persist.ts",
		`bus.on("x", (msg) => msg.reply({}));`,
	)

	// Bus deploy persists — store should have the deployment
	deps, _ := store.LoadDeployments()
	assert.Len(t, deps, 1, "bus deploy should persist")
}

// ── Schedule persistence (from persistence_test.go) ─────────────────────

func testScheduleCatchUpOnRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "schedule-restart.db")

	// Kernel 1: create a one-time schedule that fires 100ms from now
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store1,
		Modules: []brainkit.Module{schedulesmod.NewModule(schedulesmod.Config{Store: store1})},
	})
	require.NoError(t, err)

	// Create a schedule that fires "in 100ms"
	schedID := testutil.Schedule(t, k1, "in 100ms", "test.catchup.persist", json.RawMessage(`{"caught":"up"}`))
	require.NotEmpty(t, schedID)

	// Close immediately — the schedule hasn't fired yet
	k1.Close()

	// Wait past the fire time
	time.Sleep(200 * time.Millisecond)

	// Kernel 2: same store — should fire the missed schedule on startup
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store2,
		Modules: []brainkit.Module{schedulesmod.NewModule(schedulesmod.Config{Store: store2})},
	})
	require.NoError(t, err)
	defer k2.Close()

	// The one-time schedule should have been fired and deleted
	schedules := listSchedules(t, k2)
	assert.Empty(t, schedules, "one-time schedule should be deleted after catch-up fire")
}

func testRecurringScheduleRestartsCorrectly(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "recurring-restart.db")

	// Kernel 1: create recurring schedule
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store1,
		Modules: []brainkit.Module{schedulesmod.NewModule(schedulesmod.Config{Store: store1})},
	})
	require.NoError(t, err)

	schedID := testutil.Schedule(t, k1, "every 1h", "test.heartbeat.persist", json.RawMessage(`{"beat":true}`))
	require.NotEmpty(t, schedID)

	// Verify schedule exists
	schedules := listSchedules(t, k1)
	require.Len(t, schedules, 1)
	k1.Close()

	// Kernel 2: schedule should be restored and active
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store2,
		Modules: []brainkit.Module{schedulesmod.NewModule(schedulesmod.Config{Store: store2})},
	})
	require.NoError(t, err)
	defer k2.Close()

	schedules = listSchedules(t, k2)
	require.Len(t, schedules, 1, "recurring schedule should be restored")
	assert.Equal(t, "every 1h", schedules[0].Expression)
	assert.Equal(t, "test.heartbeat.persist", schedules[0].Topic)
	assert.False(t, schedules[0].OneTime, "should be recurring, not one-time")
}

func testDeployOrderPreservedExactly(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "order-exact.db")

	// Kernel 1: deploy A, B, C in order
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", Store: store1,
	})
	require.NoError(t, err)

	for _, name := range []string{"alpha-persist.ts", "beta-persist.ts", "gamma-persist.ts"} {
		testutil.Deploy(t, k1, name, `bus.on("x", (msg) => msg.reply({}));`)
	}
	k1.Close()

	// Kernel 2: verify load order matches deploy order
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	deps, err := store2.LoadDeployments()
	require.NoError(t, err)
	require.Len(t, deps, 3)

	// LoadDeployments returns ORDER BY deploy_order
	assert.Equal(t, "alpha-persist.ts", deps[0].Source)
	assert.Equal(t, "beta-persist.ts", deps[1].Source)
	assert.Equal(t, "gamma-persist.ts", deps[2].Source)
	assert.True(t, deps[0].Order < deps[1].Order, "alpha should have lower order than beta")
	assert.True(t, deps[1].Order < deps[2].Order, "beta should have lower order than gamma")

	store2.Close()
}

// ── Edge cases — corrupt store recovery ─────────────────────────────────

func testCorruptDeploymentTable(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "corrupt.db")

	// Create valid store with a deployment
	store, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
	})
	require.NoError(t, err)
	testutil.Deploy(t, k, "valid-persist.ts", `output("valid");`)
	k.Close()
	store.Close()

	// Open DB directly and corrupt data
	db, err := sql.Open("sqlite", storePath)
	require.NoError(t, err)

	// Inject deployment with code that throws during restore
	db.Exec(`INSERT OR REPLACE INTO deployments (source, code, deploy_order, deployed_at, package_name, role)
		VALUES ('evil-persist.ts', 'throw new Error("corrupt restore");', 0, '2026-01-01T00:00:00Z', '', 'service')`)

	// Inject deployment with binary garbage as code
	db.Exec(`INSERT OR REPLACE INTO deployments (source, code, deploy_order, deployed_at, package_name, role)
		VALUES ('binary-persist.ts', X'DEADBEEF', 1, '2026-01-01T00:00:00Z', '', 'service')`)

	// Inject deployment with enormous code
	bigCode := make([]byte, 1024*1024) // 1MB of garbage
	for i := range bigCode {
		bigCode[i] = byte('A' + (i % 26))
	}
	db.Exec(`INSERT OR REPLACE INTO deployments (source, code, deploy_order, deployed_at, package_name, role)
		VALUES ('huge-persist.ts', ?, 2, '2026-01-01T00:00:00Z', '', 'service')`, string(bigCode))

	db.Close()

	// Reopen — kernel should handle corrupt deployments gracefully
	store2, _ := brainkit.NewSQLiteStore(storePath)
	var errors []error
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
		ErrorHandler: func(err error) {
			errors = append(errors, err)
		},
	})
	require.NoError(t, err)
	defer k2.Close()

	assert.True(t, testutil.Alive(t, k2), "kernel should survive corrupt deployments")
	t.Logf("Errors during restore: %d", len(errors))

	// The valid deployment should still work
	deps := testutil.ListDeployments(t, k2)
	found := false
	for _, d := range deps {
		if d.Source == "valid-persist.ts" {
			found = true
		}
	}
	assert.True(t, found, "valid deployment should survive corrupt siblings")
}

func testCorruptScheduleTable(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "sched-corrupt.db")

	store, _ := brainkit.NewSQLiteStore(storePath)
	store.SaveSchedule(types.PersistedSchedule{
		ID: "valid-sched", Expression: "every 1h", Duration: time.Hour,
		Topic: "valid.topic.persist", Payload: json.RawMessage(`{}`),
		Source: "test", CreatedAt: time.Now(), NextFire: time.Now().Add(time.Hour),
	})
	// Inject corrupt schedule
	store.SaveSchedule(types.PersistedSchedule{
		ID: "corrupt-sched", Expression: "invalid-expression", Duration: 0,
		Topic: "", Payload: json.RawMessage(`not-json`),
		Source: "", CreatedAt: time.Time{}, NextFire: time.Time{},
	})
	// Inject schedule with negative duration
	store.SaveSchedule(types.PersistedSchedule{
		ID: "neg-sched", Expression: "every -1h", Duration: -time.Hour,
		Topic: "neg.topic.persist", Payload: json.RawMessage(`{}`),
		Source: "test", CreatedAt: time.Now(), NextFire: time.Now().Add(-time.Hour),
	})
	store.Close()

	store2, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k.Close()

	assert.True(t, testutil.Alive(t, k), "kernel should survive corrupt schedules")
}
