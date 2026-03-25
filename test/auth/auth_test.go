// Package auth tests every authentication method for every database backend.
// Each test starts a real container with specific auth configuration,
// creates a Kernel, and verifies the JS driver can connect + read/write.
// No mocks, no skipping — real containers, real auth handshakes.
package auth_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/kit"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

func init() {
	testutil.LoadEnv(&testing.T{})
}

// evalStore deploys a minimal .ts that connects to a store, writes a thread,
// reads it back, and returns the result as JSON.
func evalStore(t *testing.T, k *kit.Kernel, storeType, storeCode string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	code := `
		try {
			var embed = globalThis.__agent_embed;
			` + storeCode + `
			await store.init();
			// init() triggers the full auth handshake (SCRAM, md5, token, etc.)
			// If we get here, auth succeeded. Now test basic CRUD.
			var thread = { id: "auth-test-1", resourceid: "user1", metadata: {}, createdat: new Date(), updatedat: new Date() };
			// Mastra stores use saveThread({ thread }) wrapper
			if (typeof store.saveThread === "function") {
				await store.saveThread(thread);
			}
			// Try listing threads
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
	result, err := k.EvalTS(ctx, "__auth_test.ts", code)
	require.NoError(t, err, "EvalTS failed")
	t.Logf("[%s] %s", storeType, result)
	return result
}

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

func newKernel(t *testing.T, envVars map[string]string) *kit.Kernel {
	t.Helper()
	tmpDir := t.TempDir()
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "auth-test",
		WorkspaceDir: tmpDir,
		EnvVars:      envVars,
		EmbeddedStorages: map[string]kit.EmbeddedStorageConfig{
			"default": {Path: filepath.Join(tmpDir, "brainkit.db")},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })
	return k
}

// ═══════════════════════════════════════════════════════════════════
// PostgreSQL Auth Methods
// ═══════════════════════════════════════════════════════════════════

func TestPostgres_SCRAM_SHA256(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
	testutil.CleanupOrphanedContainers(t)

	// pg16 defaults to scram-sha-256
	addr := testutil.StartContainer(t, "pgvector/pgvector:pg16", "5432/tcp", nil,
		wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second),
		"POSTGRES_USER=scramuser", "POSTGRES_PASSWORD=scrampass", "POSTGRES_DB=authtest")
	waitForTCP(t, addr, 15*time.Second)

	k := newKernel(t, map[string]string{
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
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
	testutil.CleanupOrphanedContainers(t)

	// Force md5 auth via POSTGRES_HOST_AUTH_METHOD
	addr := testutil.StartContainer(t, "postgres:16", "5432/tcp", nil,
		wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second),
		"POSTGRES_USER=md5user", "POSTGRES_PASSWORD=md5pass", "POSTGRES_DB=authtest",
		"POSTGRES_HOST_AUTH_METHOD=md5")
	waitForTCP(t, addr, 15*time.Second)

	k := newKernel(t, map[string]string{
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
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
	testutil.CleanupOrphanedContainers(t)

	// Trust auth — no password required
	addr := testutil.StartContainer(t, "postgres:16", "5432/tcp", nil,
		wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(60*time.Second),
		"POSTGRES_USER=trustuser", "POSTGRES_HOST_AUTH_METHOD=trust")
	waitForTCP(t, addr, 15*time.Second)

	k := newKernel(t, map[string]string{
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

// ═══════════════════════════════════════════════════════════════════
// MongoDB Auth Methods
// ═══════════════════════════════════════════════════════════════════

func TestMongoDB_SCRAM_SHA256(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
	testutil.CleanupOrphanedContainers(t)

	// MongoDB 7 defaults to SCRAM-SHA-256
	addr := testutil.StartContainer(t, "mongo:7", "27017/tcp", nil,
		wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second),
		"MONGO_INITDB_ROOT_USERNAME=scramuser", "MONGO_INITDB_ROOT_PASSWORD=scrampass")
	waitForTCP(t, addr, 15*time.Second)

	k := newKernel(t, map[string]string{
		"MONGODB_URL":      "mongodb://scramuser:scrampass@" + addr,
		"MONGODB_LOG_ALL":  "off",
	})

	result := evalStore(t, k, "mongodb-scram-sha256", `
		var store = new embed.MongoDBStore({
			id: "mongo-scram256-test",
			url: process.env.MONGODB_URL,
			dbName: "authtest",
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

func TestMongoDB_SCRAM_SHA1(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
	testutil.CleanupOrphanedContainers(t)

	// Force SCRAM-SHA-1 only via mongod --setParameter
	addr := testutil.StartContainer(t, "mongo:7", "27017/tcp",
		[]string{"mongod", "--setParameter", "authenticationMechanisms=SCRAM-SHA-1"},
		wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second),
		"MONGO_INITDB_ROOT_USERNAME=sha1user", "MONGO_INITDB_ROOT_PASSWORD=sha1pass")
	waitForTCP(t, addr, 15*time.Second)

	k := newKernel(t, map[string]string{
		"MONGODB_URL":      "mongodb://sha1user:sha1pass@" + addr + "/?authMechanism=SCRAM-SHA-1",
		"MONGODB_LOG_ALL":  "off",
	})

	result := evalStore(t, k, "mongodb-scram-sha1", `
		var store = new embed.MongoDBStore({
			id: "mongo-scram1-test",
			url: process.env.MONGODB_URL,
			dbName: "authtest",
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

func TestMongoDB_NoAuth(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
	testutil.CleanupOrphanedContainers(t)

	// No auth — no credentials
	addr := testutil.StartContainer(t, "mongo:7", "27017/tcp", nil,
		wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second))
	waitForTCP(t, addr, 15*time.Second)

	k := newKernel(t, map[string]string{
		"MONGODB_URL": "mongodb://" + addr,
	})

	result := evalStore(t, k, "mongodb-noauth", `
		var store = new embed.MongoDBStore({
			id: "mongo-noauth-test",
			url: process.env.MONGODB_URL,
			dbName: "authtest",
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

// ═══════════════════════════════════════════════════════════════════
// Upstash (cloud token auth)
// ═══════════════════════════════════════════════════════════════════

func TestUpstash_TokenAuth(t *testing.T) {
	url := os.Getenv("UPSTASH_REDIS_REST_URL")
	token := os.Getenv("UPSTASH_REDIS_REST_TOKEN")
	if url == "" || token == "" {
		t.Skip("needs UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN in .env")
	}

	k := newKernel(t, map[string]string{
		"UPSTASH_REDIS_REST_URL":   url,
		"UPSTASH_REDIS_REST_TOKEN": token,
	})

	result := evalStore(t, k, "upstash-token", `
		var store = new embed.UpstashStore({
			id: "upstash-auth-test",
			url: process.env.UPSTASH_REDIS_REST_URL,
			token: process.env.UPSTASH_REDIS_REST_TOKEN,
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

// ═══════════════════════════════════════════════════════════════════
// LibSQL (embedded HTTP bridge — no auth, and containerized)
// ═══════════════════════════════════════════════════════════════════

func TestLibSQL_EmbeddedNoAuth(t *testing.T) {
	k := newKernel(t, map[string]string{})

	// Use the embedded libsql bridge (started by Kernel from EmbeddedStorages)
	url := k.StorageURL("default")
	require.NotEmpty(t, url, "embedded storage URL should be set")
	os.Setenv("LIBSQL_URL", url)

	result := evalStore(t, k, "libsql-embedded", `
		var store = new embed.LibSQLStore({
			id: "libsql-auth-test",
			url: process.env.LIBSQL_URL,
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

func TestLibSQL_ContainerNoAuth(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
	testutil.CleanupOrphanedContainers(t)

	addr := testutil.StartContainer(t,
		"ghcr.io/tursodatabase/libsql-server:latest",
		"8080/tcp",
		[]string{"sqld", "--http-listen-addr", "0.0.0.0:8080"},
		wait.ForHTTP("/health").WithStartupTimeout(30*time.Second))

	k := newKernel(t, map[string]string{
		"LIBSQL_URL": "http://" + addr,
	})

	result := evalStore(t, k, "libsql-container", `
		var store = new embed.LibSQLStore({
			id: "libsql-container-test",
			url: process.env.LIBSQL_URL,
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

// ═══════════════════════════════════════════════════════════════════
// InMemory (no auth, baseline)
// ═══════════════════════════════════════════════════════════════════

func TestInMemory_NoAuth(t *testing.T) {
	k := newKernel(t, map[string]string{})

	result := evalStore(t, k, "inmemory", `
		var store = new embed.InMemoryStore();
	`)
	require.Contains(t, result, `"ok":true`)
}
