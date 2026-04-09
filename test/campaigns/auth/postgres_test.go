// Package auth_test runs auth-method campaigns for PostgreSQL.
// Each test starts a real Postgres container with specific auth configuration,
// creates a Kernel, and verifies the JS driver can connect and perform CRUD.
package auth_test

import (
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

func waitForTCP(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("TCP probe failed: %s not accepting connections after %v", addr, timeout)
}

func newKit(t *testing.T, envVars map[string]string) *brainkit.Kit {
	t.Helper()
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test",
		CallerID:  "auth-test",
		FSRoot:    tmpDir,
		EnvVars:   envVars,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })
	return k
}

// evalStore deploys a minimal .ts that connects to a store, writes a thread,
// reads it back, and returns the result as JSON.
func evalStore(t *testing.T, k *brainkit.Kit, storeType, storeCode string) string {
	t.Helper()

	code := `
		try {
			var embed = globalThis.__agent_embed;
			` + storeCode + `
			await store.init();
			var thread = { id: "auth-test-1", resourceid: "user1", metadata: {}, createdat: new Date(), updatedat: new Date() };
			if (typeof store.saveThread === "function") {
				await store.saveThread(thread);
			}
			var threads = [];
			if (typeof store.getThreadsByResourceId === "function") {
				threads = await store.getThreadsByResourceId({ resourceId: "user1" });
			} else if (typeof store.listThreads === "function") {
				threads = await store.listThreads({});
			}
			return JSON.stringify({ ok: true, backend: "` + storeType + `", connected: true, threads: threads ? threads.length : -1 });
		} catch(e) {
			return JSON.stringify({ error: e.message.substring(0, 300), backend: "` + storeType + `" });
		}
	`
	result := testutil.EvalTS(t, k, "__auth_test.ts", code)
	t.Logf("[%s] %s", storeType, result)
	return result
}

func TestPostgres_SCRAM_SHA256(t *testing.T) {
	campaigns.RequirePodman(t)
	testutil.CleanupOrphanedContainers(t)

	addr := testutil.StartContainer(t, "pgvector/pgvector:pg16", "5432/tcp", nil,
		wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second),
		"POSTGRES_USER=scramuser", "POSTGRES_PASSWORD=scrampass", "POSTGRES_DB=authtest")
	waitForTCP(t, addr, 15*time.Second)

	k := newKit(t, map[string]string{
		"POSTGRES_URL": "postgresql://scramuser:scrampass@" + addr + "/authtest",
	})

	result := evalStore(t, k, "postgres-scram-sha256", `
		var store = new embed.PostgresStore({
			id: "pg-scram-test",
			connectionString: process.env.POSTGRES_URL,
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

func TestPostgres_MD5(t *testing.T) {
	campaigns.RequirePodman(t)
	testutil.CleanupOrphanedContainers(t)

	addr := testutil.StartContainer(t, "postgres:16", "5432/tcp", nil,
		wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second),
		"POSTGRES_USER=md5user", "POSTGRES_PASSWORD=md5pass", "POSTGRES_DB=authtest",
		"POSTGRES_HOST_AUTH_METHOD=md5")
	waitForTCP(t, addr, 15*time.Second)

	k := newKit(t, map[string]string{
		"POSTGRES_URL": "postgresql://md5user:md5pass@" + addr + "/authtest",
	})

	result := evalStore(t, k, "postgres-md5", `
		var store = new embed.PostgresStore({
			id: "pg-md5-test",
			connectionString: process.env.POSTGRES_URL,
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

func TestPostgres_Trust(t *testing.T) {
	campaigns.RequirePodman(t)
	testutil.CleanupOrphanedContainers(t)

	addr := testutil.StartContainer(t, "postgres:16", "5432/tcp", nil,
		wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second),
		"POSTGRES_USER=trustuser", "POSTGRES_HOST_AUTH_METHOD=trust")
	waitForTCP(t, addr, 15*time.Second)

	k := newKit(t, map[string]string{
		"POSTGRES_URL": "postgresql://trustuser@" + addr + "/trustuser?sslmode=disable",
	})

	result := evalStore(t, k, "postgres-trust", `
		var store = new embed.PostgresStore({
			id: "pg-trust-test",
			connectionString: process.env.POSTGRES_URL,
		});
	`)
	require.Contains(t, result, `"ok":true`)
}
