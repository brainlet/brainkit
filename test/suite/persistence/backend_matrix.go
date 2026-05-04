package persistence

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	schedulesmod "github.com/brainlet/brainkit/modules/schedules"
	"github.com/brainlet/brainkit/modules/schedules/schedulemsg"
	secretsmod "github.com/brainlet/brainkit/modules/secrets"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/stores"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDeployPersistRestart — deploy persistence across kernel restart.
// Ported from adversarial/backend_matrix_test.go:TestBackendMatrix_DeployPersistRestart.
func testDeployPersistRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-matrix.db")
	store, err := stores.NewSQLite(storePath)
	require.NoError(t, err)

	// Phase 1: Deploy
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test",
		FSRoot: tmpDir, Store: store,
		Modules: packageModules(),
	})
	require.NoError(t, err)

	testutil.Deploy(t, k1, "persist-test-matrix.ts", `output("persisted");`)
	k1.Close()

	// Phase 2: Reopen with same store — deployment should be restored
	store2, err := stores.NewSQLite(storePath)
	require.NoError(t, err)

	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test",
		FSRoot: tmpDir, Store: store2,
		Modules: packageModules(),
	})
	require.NoError(t, err)
	defer k2.Close()

	deps := testutil.ListDeployments(t, k2)
	found := false
	for _, d := range deps {
		if d.Source == "persist-test-matrix.ts" {
			found = true
		}
	}
	assert.True(t, found, "persist-test-matrix.ts should survive restart")
}

// testSecretsSurviveRestart — secrets survive kernel restart.
// Ported from adversarial/persistence_matrix_test.go:TestPersistence_Secrets.
func testSecretsSurviveRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-secrets.db")

	// Phase 1: Set secrets
	store1, err := stores.NewSQLite(storePath)
	require.NoError(t, err)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: "test-master-key-1234567890",
		Modules: []bkmodule.Module{secretsmod.New()},
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = secretmsg.CallSecretsSet(k1, ctx, secretmsg.SecretsSetMsg{Name: "persist-secret-matrix", Value: "secret-value-123"}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	k1.Close()

	// Phase 2: Reopen — secret should be retrievable
	store2, _ := stores.NewSQLite(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2, SecretKey: "test-master-key-1234567890",
		Modules: []bkmodule.Module{secretsmod.New()},
	})
	require.NoError(t, err)
	defer k2.Close()

	resp, err := secretmsg.CallSecretsGet(k2, ctx, secretmsg.SecretsGetMsg{Name: "persist-secret-matrix"}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	assert.Equal(t, "secret-value-123", resp.Value, "secret should survive restart")
}

// testMultiDeployOrderAndMetadata — multiple deployments restart with metadata.
// Ported from adversarial/persistence_matrix_test.go:TestPersistence_MultiDeployOrderAndMetadata.
func testMultiDeployOrderAndMetadata(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-multi.db")

	store1, err := stores.NewSQLite(storePath)
	require.NoError(t, err)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store1,
		Modules: packageModules(),
	})
	require.NoError(t, err)

	testutil.Deploy(t, k1, "first-matrix.ts", `output("first");`)
	err = testutil.DeployWithOpts(k1, "second-matrix.ts", `output("second");`, "")
	require.NoError(t, err)
	// packageName is the package identity; runtime source derives from it
	// (name + ext). Passing "my-pkg" deploys as "my-pkg.ts".
	err = testutil.DeployWithOpts(k1, "third-matrix.ts", `output("third");`, "my-pkg")
	require.NoError(t, err)
	k1.Close()

	// Phase 2: Reopen
	store2, _ := stores.NewSQLite(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store2,
		Modules: packageModules(),
	})
	require.NoError(t, err)
	defer k2.Close()

	deps := testutil.ListDeployments(t, k2)
	sources := make([]string, len(deps))
	for i, d := range deps {
		sources[i] = d.Source
	}
	assert.Contains(t, sources, "first-matrix.ts")
	assert.Contains(t, sources, "second-matrix.ts")
	assert.Contains(t, sources, "my-pkg.ts") // packageName drives the runtime source
}

// testMultipleSchedulesSurvive — multiple schedules survive restart.
// Ported from adversarial/persistence_matrix_test.go:TestPersistence_MultipleSchedules.
func testMultipleSchedulesSurvive(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-sched.db")

	store1, err := stores.NewSQLite(storePath)
	require.NoError(t, err)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store1,
		Modules: []bkmodule.Module{schedulesmod.NewModule(schedulesmod.Config{Store: store1})},
	})
	require.NoError(t, err)

	testutil.Schedule(t, k1, "every 1h", "sched.hourly.matrix", json.RawMessage(`{}`))
	testutil.Schedule(t, k1, "every 5m", "sched.fivemin.matrix", json.RawMessage(`{}`))
	testutil.Schedule(t, k1, "in 24h", "sched.onetime.matrix", json.RawMessage(`{}`))
	k1.Close()

	// Reopen
	store2, _ := stores.NewSQLite(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store2,
		Modules: []bkmodule.Module{schedulesmod.NewModule(schedulesmod.Config{Store: store2})},
	})
	require.NoError(t, err)
	defer k2.Close()

	scheds := listSchedules(t, k2)
	assert.GreaterOrEqual(t, len(scheds), 2, "at least 2 schedules should survive (one-time may have fired)")
}

// testDeployWithBusHandlerSurvivesRestart — handler re-subscribes after restart.
// Ported from adversarial/persistence_matrix_test.go:TestPersistence_DeployWithBusHandler.
func testDeployWithBusHandlerSurvivesRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-handler.db")

	// Phase 1: Deploy with bus handler
	store1, _ := stores.NewSQLite(storePath)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store1,
		Modules: packageModules(),
	})
	require.NoError(t, err)

	testutil.Deploy(t, k1, "handler-matrix.ts", `
		bus.on("ping", function(msg) { msg.reply({alive: true}); });
	`)
	k1.Close()

	// Phase 2: Reopen — handler should be active again
	store2, _ := stores.NewSQLite(storePath)
	k2, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store2,
		Modules: packageModules(),
	})
	require.NoError(t, err)
	defer k2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	payload := callPersistService(t, k2, ctx, "handler-matrix.ts", "ping", json.RawMessage(`{}`))
	assert.Contains(t, string(payload), "alive")
}

// listSchedules queries the schedule list via bus command.
func listSchedules(t *testing.T, k *brainkit.Kit) []schedulemsg.ScheduleInfo {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := sdk.Call[schedulemsg.ScheduleListMsg, schedulemsg.ScheduleListResp](k, ctx, schedulemsg.ScheduleListMsg{})
	require.NoError(t, err)
	return resp.Schedules
}
