package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

// testStoreBackendSQLiteViaConfig proves that Config.StoreBackend="sqlite"
// creates a working store through the factory path (not explicit Store field).
func testStoreBackendSQLiteViaConfig(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()

	// Kit 1: deploy via StoreBackend config
	k1, err := brainkit.New(brainkit.Config{
		Transport:    "memory",
		Namespace:    "test",
		CallerID:     "test",
		FSRoot:       tmpDir,
		StoreBackend: "sqlite",
		StoreURL:     tmpDir + "/backend-test.db",
	})
	require.NoError(t, err)

	testutil.Deploy(t, k1, "backend-persist.ts", `
		bus.on("ping", function(msg) { msg.reply({pong: true}); });
	`)
	k1.Close()

	// Kit 2: restart with same store — deployment should survive
	k2, err := brainkit.New(brainkit.Config{
		Transport:    "memory",
		Namespace:    "test",
		CallerID:     "test",
		FSRoot:       tmpDir,
		StoreBackend: "sqlite",
		StoreURL:     tmpDir + "/backend-test.db",
	})
	require.NoError(t, err)
	defer k2.Close()

	deps := testutil.ListDeployments(t, k2)
	found := false
	for _, d := range deps {
		if d.Source == "backend-persist.ts" {
			found = true
		}
	}
	assert.True(t, found, "deployment should survive restart via StoreBackend=sqlite")
}

// testStoreBackendSQLiteAuditViaConfig proves that the audit store is
// auto-created and records events when StoreBackend is set.
func testStoreBackendSQLiteAuditViaConfig(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()

	k, err := brainkit.New(brainkit.Config{
		Transport:    "memory",
		Namespace:    "test",
		CallerID:     "test",
		FSRoot:       tmpDir,
		StoreBackend: "sqlite",
		StoreURL:     tmpDir + "/audit-backend-test.db",
	})
	require.NoError(t, err)
	defer k.Close()

	// Deploy something — should generate audit event
	testutil.Deploy(t, k, "audit-backend.ts", `
		bus.on("ping", function(msg) { msg.reply({pong: true}); });
	`)

	// Query audit — should find the deploy event
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	replyTo := fmt.Sprintf("audit.query.reply.%d", time.Now().UnixNano())
	ch := make(chan json.RawMessage, 1)
	unsub, _ := k.SubscribeRaw(ctx, replyTo, func(m sdk.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	defer unsub()

	sdk.Publish(k, ctx, sdk.AuditQueryMsg{Category: "deploy"}, sdk.WithReplyTo(replyTo))

	select {
	case resp := <-ch:
		var result sdk.AuditQueryResp
		json.Unmarshal(resp, &result)
		assert.GreaterOrEqual(t, len(result.Events), 1, "audit should have deploy events")
		t.Logf("audit events: %d", len(result.Events))
	case <-ctx.Done():
		t.Fatal("timeout querying audit")
	}
}

// testStoreBackendPostgresViaConfig proves that Config.StoreBackend="postgres"
// creates a working store. Requires Podman.
func testStoreBackendPostgresViaConfig(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)

	pgAddr := testutil.StartContainer(t, "postgres:16", "5432/tcp", nil,
		wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second),
		"POSTGRES_USER=test", "POSTGRES_PASSWORD=test", "POSTGRES_DB=brainkit",
	)
	pgURL := fmt.Sprintf("postgres://test:test@%s/brainkit?sslmode=disable", pgAddr)

	// Kit 1: deploy via StoreBackend=postgres
	k1, err := brainkit.New(brainkit.Config{
		Transport:    "memory",
		Namespace:    "test",
		CallerID:     "test",
		FSRoot:       t.TempDir(),
		StoreBackend: "postgres",
		StoreURL:     pgURL,
	})
	require.NoError(t, err)

	testutil.Deploy(t, k1, "pg-persist.ts", `
		bus.on("ping", function(msg) { msg.reply({pong: true}); });
	`)
	k1.Close()

	// Kit 2: restart with same Postgres — deployment should survive
	k2, err := brainkit.New(brainkit.Config{
		Transport:    "memory",
		Namespace:    "test",
		CallerID:     "test",
		FSRoot:       t.TempDir(),
		StoreBackend: "postgres",
		StoreURL:     pgURL,
	})
	require.NoError(t, err)
	defer k2.Close()

	deps := testutil.ListDeployments(t, k2)
	found := false
	for _, d := range deps {
		if d.Source == "pg-persist.ts" {
			found = true
		}
	}
	assert.True(t, found, "deployment should survive restart via StoreBackend=postgres")
}
