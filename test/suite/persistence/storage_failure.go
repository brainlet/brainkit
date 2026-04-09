package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// testStorageFailureMidQuery starts Postgres, deploys a .ts that writes to storage,
// kills Postgres mid-operation, and observes the error behavior.
// Goal: understand what Mastra does when the backend dies. Does it retry? Hang? Error?

// testPostgresStorageDeath is the real test. It needs Podman.
func testPostgresStorageDeath(t *testing.T, _ *suite.TestEnv) {
	if !testutil.PodmanAvailable() {
		t.Skip("Podman not available")
	}
	testutil.LoadEnv(t)

	// ── Start Postgres container ──
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	if os.Getenv("DOCKER_HOST") == "" {
		if out, err := exec.Command("podman", "machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}").Output(); err == nil {
			os.Setenv("DOCKER_HOST", "unix://"+string(out[:len(out)-1]))
		}
	}

	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16",
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second),
			Env: map[string]string{
				"POSTGRES_USER":     "test",
				"POSTGRES_PASSWORD": "test",
				"POSTGRES_DB":       "brainkit",
			},
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("failed to start postgres container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	mappedPort, _ := container.MappedPort(ctx, nat.Port("5432/tcp"))
	pgConnStr := fmt.Sprintf("postgresql://test:test@%s:%s/brainkit", host, mappedPort.Port())
	t.Logf("Postgres at %s", pgConnStr)

	// ── Create Kit with Postgres storage ──
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: "memory",
		Namespace: "test-storage-fail",
		CallerID:  "test",
		FSRoot:    tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.PostgresStorage(pgConnStr),
		},
		Store: mustStore(t, filepath.Join(tmpDir, "kit.db")),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer k.Close()

	// ── Deploy .ts that writes to Postgres storage ──
	tsCode := `
		import { bus, storage } from "kit";
		import { Memory } from "agent";

		const store = storage("default");
		const mem = new Memory({ storage: store });

		bus.on("write", async (msg) => {
			try {
				await mem.saveThread({ thread: {
					id: "thread-" + Date.now(),
					title: msg.payload.title || "test",
					resourceId: "test-resilience",
					createdAt: new Date(),
					updatedAt: new Date(),
					metadata: {},
				}});
				msg.reply({ ok: true });
			} catch(e) {
				msg.reply({ ok: false, error: e.message || String(e) });
			}
		});

		bus.on("read", async (msg) => {
			try {
				// Use a simple SQL query to verify the store is reachable
				const result = await store.getThread("nonexistent");
				msg.reply({ ok: true, found: result != null });
			} catch(e) {
				// getThread on nonexistent returns null, not error — any error here means connection failed
				msg.reply({ ok: false, error: e.message || String(e) });
			}
		});
	`

	testutil.Deploy(t, k, "storage-fail-persist.ts", tsCode)

	// ── Phase 1: Verify storage works while Postgres is alive ──
	t.Log("Phase 1: Writing to storage with Postgres alive...")
	resp := busRoundTrip(t, k, "ts.storage-fail-persist.write", map[string]any{"title": "before-kill"})
	if !resp.OK {
		t.Fatalf("Write before kill failed: %s", resp.Error)
	}
	t.Log("  Write succeeded")

	resp = busRoundTrip(t, k, "ts.storage-fail-persist.read", nil)
	t.Logf("  Read result: ok=%v, error=%q", resp.OK, resp.Error)

	// ── Phase 2: Kill Postgres container ──
	t.Log("Phase 2: Killing Postgres container...")
	if err := container.Stop(ctx, nil); err != nil {
		t.Fatalf("Stop container: %v", err)
	}
	t.Log("  Postgres stopped")

	// ── Phase 3: Attempt operations with dead Postgres ──
	t.Log("Phase 3: Writing to storage with Postgres dead...")
	writeStart := time.Now()
	resp = busRoundTripWithTimeout(t, k, "ts.storage-fail-persist.write", map[string]any{"title": "after-kill"}, 15*time.Second)
	writeDuration := time.Since(writeStart)
	t.Logf("  Write result: ok=%v, error=%q, duration=%s", resp.OK, resp.Error, writeDuration.Round(time.Millisecond))

	t.Log("Phase 3b: Reading from storage with Postgres dead...")
	readStart := time.Now()
	resp = busRoundTripWithTimeout(t, k, "ts.storage-fail-persist.read", nil, 15*time.Second)
	readDuration := time.Since(readStart)
	t.Logf("  Read result: ok=%v, error=%q, duration=%s", resp.OK, resp.Error, readDuration.Round(time.Millisecond))

	// ── Phase 4: Restart Postgres and verify recovery ──
	t.Log("Phase 4: Restarting Postgres container...")
	if err := container.Start(ctx); err != nil {
		t.Fatalf("Restart container: %v", err)
	}
	// Wait for Postgres to be ready
	time.Sleep(3 * time.Second)

	t.Log("Phase 4b: Writing to storage after Postgres restart...")
	resp = busRoundTripWithTimeout(t, k, "ts.storage-fail-persist.write", map[string]any{"title": "after-restart"}, 15*time.Second)
	t.Logf("  Write result: ok=%v, error=%q", resp.OK, resp.Error)

	resp = busRoundTripWithTimeout(t, k, "ts.storage-fail-persist.read", nil, 15*time.Second)
	t.Logf("  Read result: ok=%v, error=%q", resp.OK, resp.Error)

	// ── Assessment: Log what we learned ──
	t.Log("=== Storage Failure Assessment ===")
	t.Logf("Write with dead Postgres: took %s", writeDuration.Round(time.Millisecond))
	t.Logf("Read with dead Postgres: took %s", readDuration.Round(time.Millisecond))
	if writeDuration > 10*time.Second || readDuration > 10*time.Second {
		t.Log("FINDING: Operations hang for >10s when Postgres is dead — needs timeout or circuit breaker")
	} else if resp.OK {
		t.Log("FINDING: Operations succeed after restart — Mastra reconnects automatically")
	} else {
		t.Log("FINDING: Operations fail fast — Mastra does NOT retry/reconnect automatically")
	}
}

type storageResp struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Count int    `json:"count"`
}

func busRoundTrip(t *testing.T, k *brainkit.Kit, topic string, payload any) storageResp {
	t.Helper()
	return busRoundTripWithTimeout(t, k, topic, payload, 10*time.Second)
}

func busRoundTripWithTimeout(t *testing.T, k *brainkit.Kit, topic string, payload any, timeout time.Duration) storageResp {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	payloadJSON, _ := json.Marshal(payload)
	correlationID := fmt.Sprintf("test-%d", time.Now().UnixNano())
	replyTo := topic + ".reply." + correlationID

	replyCh := make(chan sdk.Message, 1)
	unsub, err := k.SubscribeRaw(ctx, replyTo, func(msg sdk.Message) {
		select {
		case replyCh <- msg:
		default:
		}
	})
	if err != nil {
		return storageResp{Error: "subscribe: " + err.Error()}
	}
	defer unsub()

	pubCtx := transport.WithPublishMeta(ctx, correlationID, replyTo)
	_, err = k.PublishRaw(pubCtx, topic, payloadJSON)
	if err != nil {
		return storageResp{Error: "publish: " + err.Error()}
	}

	select {
	case msg := <-replyCh:
		var resp storageResp
		json.Unmarshal(msg.Payload, &resp)
		return resp
	case <-ctx.Done():
		return storageResp{Error: "timeout after " + timeout.String()}
	}
}

func mustStore(t *testing.T, path string) *brainkit.SQLiteStore {
	t.Helper()
	store, err := brainkit.NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}
