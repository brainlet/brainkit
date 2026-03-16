package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ensurePodmanSocket sets DOCKER_HOST for testcontainers if using Podman.
func ensurePodmanSocket(t *testing.T) {
	t.Helper()
	if os.Getenv("DOCKER_HOST") != "" {
		return // already set
	}
	// Try podman machine inspect to get socket path
	out, err := exec.Command("podman", "machine", "inspect", "--format", "{{.ConnectionInfo.PodmanSocket.Path}}").Output()
	if err != nil {
		return // no podman
	}
	sock := strings.TrimSpace(string(out))
	if sock != "" {
		os.Setenv("DOCKER_HOST", "unix://"+sock)
		os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true") // Ryuk doesn't work well with Podman
		t.Logf("Using Podman socket: %s", sock)
	}
}

// TestMemoryIntegration_LibSQL spins up a real LibSQL server via testcontainers
// and tests agent memory persistence against it.
func TestMemoryIntegration_LibSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "ghcr.io/tursodatabase/libsql-server:latest",
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForHTTP("/health").WithPort("8080/tcp").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("could not start LibSQL container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "8080")
	libsqlURL := fmt.Sprintf("http://%s:%s", host, port.Port())
	t.Logf("LibSQL container running at %s", libsqlURL)

	// Create Kit with LIBSQL_URL injected into the JS runtime
	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"LIBSQL_URL": libsqlURL,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()
	code := loadFixture(t, "testdata/ts/memory-libsql.js")

	result, err := kit.EvalModule(context.Background(), "memory-libsql.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out struct {
		Text      string `json:"text"`
		Remembers bool   `json:"remembers"`
		Store     string `json:"store"`
		URL       string `json:"url"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Remembers {
		t.Errorf("agent didn't remember with LibSQL: %q", out.Text)
	}
	t.Logf("LibSQL integration: %q remembers=%v store=%s url=%s", out.Text, out.Remembers, out.Store, out.URL)
}

// TestMemoryIntegration_LibSQLLocal uses the Kit's embedded storage (pure Go SQLite).
// No Docker/Podman needed. No URL needed in JS — LibSQLStore auto-connects.
func TestMemoryIntegration_LibSQLLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	dbPath := filepath.Join(t.TempDir(), "brainkit-test.db")

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		Storages: map[string]StorageConfig{
			"default": {Path: dbPath},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/memory-libsql-local.js")
	result, err := kit.EvalModule(context.Background(), "memory-libsql-local.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out struct {
		Text           string `json:"text"`
		RemembersColor bool   `json:"remembersColor"`
		RemembersDog   bool   `json:"remembersDog"`
		Store          string `json:"store"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.RemembersColor {
		t.Errorf("agent didn't remember color: %q", out.Text)
	}
	if !out.RemembersDog {
		t.Errorf("agent didn't remember dog: %q", out.Text)
	}
	t.Logf("Local SQLite: %q color=%v dog=%v", out.Text, out.RemembersColor, out.RemembersDog)
}

// TestMemoryIntegration_ThreadManagement tests the Memory thread management API
// accessible via agent.memory (getThreadById, listThreads, saveThread, updateThread, deleteThread).
func TestMemoryIntegration_ThreadManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	dbPath := filepath.Join(t.TempDir(), "thread-mgmt.db")

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		Storages: map[string]StorageConfig{
			"default": {Path: dbPath},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/memory-thread-management.js")
	result, err := kit.EvalModule(context.Background(), "memory-thread-management.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out struct {
		SaveThread    string `json:"saveThread"`
		GetThreadById string `json:"getThreadById"`
		ListThreads   any    `json:"listThreads"`
	}
	json.Unmarshal([]byte(result), &out)

	t.Logf("Thread management: save=%v get=%v list=%v", out.SaveThread, out.GetThreadById, out.ListThreads)

	if out.SaveThread != "ok" {
		t.Errorf("saveThread: got %v, want ok", out.SaveThread)
	}
	if out.GetThreadById != "t1" {
		t.Errorf("getThreadById: got %v, want t1", out.GetThreadById)
	}
}
