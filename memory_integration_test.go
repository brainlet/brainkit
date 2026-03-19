//go:build integration

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

// TestMemoryIntegration_ThreadManagement tests the full Memory thread management API:
// saveThread, getThreadById, listThreads, updateThread, deleteThread, saveMessages, recall, deleteMessages.
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

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("Thread management: %v", out)

	check := func(key, want string) {
		got := fmt.Sprintf("%v", out[key])
		if got != want {
			t.Errorf("%s: got %v, want %v", key, got, want)
		}
	}
	check("saveThread", "ok")
	check("getThreadById", "t1")
	check("getThreadTitle", "Thread One")
	check("updateThread", "Updated Title")
	check("saveMessages", "ok")
	check("deleteThread", "deleted")

	// listThreads should be 2 initially
	if v, ok := out["listThreads"].(float64); !ok || v < 2 {
		t.Errorf("listThreads: got %v, want >= 2", out["listThreads"])
	}
	// After delete should be 1
	if v, ok := out["listAfterDelete"].(float64); !ok || v != 1 {
		t.Errorf("listAfterDelete: got %v, want 1", out["listAfterDelete"])
	}
	// Recall should find messages
	if v, ok := out["recallCount"].(float64); !ok || v < 1 {
		t.Errorf("recallCount: got %v, want >= 1", out["recallCount"])
	}
}

// TestAgentOptionsPassthrough tests that generate() options are passed through to Mastra:
// temperature, maxSteps, onStepFinish, onFinish, per-call instructions.
func TestAgentOptionsPassthrough(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/agent-options-passthrough.js")
	result, err := kit.EvalModule(context.Background(), "agent-options-passthrough.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]string
	json.Unmarshal([]byte(result), &out)
	t.Logf("Agent options: %v", out)

	for _, key := range []string{"temperature", "onStepFinish", "onFinish", "maxSteps"} {
		val := out[key]
		if val == "" || strings.HasPrefix(val, "error:") {
			t.Errorf("%s: %v", key, val)
		}
	}
}

// TestWorkspaceToolRemapping tests workspace tool name remapping and enable/disable.
func TestWorkspaceToolRemapping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	wsPath := t.TempDir()

	// Create a test file so workspace has content
	os.WriteFile(filepath.Join(wsPath, "hello.txt"), []byte("Hello World"), 0644)

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"WORKSPACE_PATH": wsPath,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/workspace-tool-remapping.js")
	result, err := kit.EvalModule(context.Background(), "workspace-tool-remapping.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]string
	json.Unmarshal([]byte(result), &out)
	t.Logf("Tool remapping: %v", out)

	if out["error"] != "" {
		t.Fatalf("workspace error: %s", out["error"])
	}
	if out["create"] != "ok" {
		t.Errorf("create: %v", out["create"])
	}
	if out["readFileRenamed"] != "view" {
		t.Errorf("readFile should be renamed to 'view', got %v", out["readFileRenamed"])
	}
	if out["editFileDisabled"] != "ok" {
		t.Errorf("editFile should be disabled, got %v", out["editFileDisabled"])
	}
	if out["planModeWrite"] != "disabled" {
		t.Errorf("plan mode write should be disabled, got %v", out["planModeWrite"])
	}
	if out["buildModeWrite"] != "enabled" {
		t.Errorf("build mode write should be enabled, got %v", out["buildModeWrite"])
	}
}

// TestWorkspaceBM25Search tests workspace BM25 keyword search.
// Works with embedded SQLite bridge (no vector extensions needed).
func TestWorkspaceBM25Search(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	wsPath := t.TempDir()

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"WORKSPACE_PATH": wsPath,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/workspace-bm25-search.js")
	result, err := kit.EvalModule(context.Background(), "workspace-bm25-search.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("BM25 search: %v", out)

	if out["error"] != nil {
		t.Fatalf("search error: %v", out["error"])
	}
	if out["status"] != "ok" {
		t.Errorf("status: %v", out["status"])
	}
	if v, ok := out["rustCount"].(float64); !ok || v < 1 {
		t.Errorf("rustCount: got %v, want >= 1", out["rustCount"])
	}
	if out["rustHasScore"] != "ok" {
		t.Errorf("results should have scores: %v", out["rustHasScore"])
	}
}

// TestWorkspaceVectorSearch tests workspace vector + hybrid search against a real libsql-server.
// Requires Docker/Podman for testcontainer.
func TestWorkspaceVectorSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ensurePodmanSocket(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "ghcr.io/tursodatabase/libsql-server:latest",
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor:   wait.ForHTTP("/health").WithPort("8080/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Skipf("could not start LibSQL container: %v", err)
	}
	defer container.Terminate(ctx)

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "8080")
	libsqlURL := fmt.Sprintf("http://%s:%s", host, port.Port())
	t.Logf("LibSQL container at %s", libsqlURL)

	key := requireKey(t)
	wsPath := t.TempDir()

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"WORKSPACE_PATH": wsPath,
			"LIBSQL_URL":     libsqlURL,
			"OPENAI_API_KEY": key,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/workspace-vector-search.js")
	result, err := kit.EvalModule(context.Background(), "workspace-vector-search.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("Vector search: %v", out)

	if out["error"] != nil {
		t.Fatalf("vector search error: %v\nstack: %v", out["error"], out["stack"])
	}
	if out["status"] != "ok" {
		t.Errorf("status: %v", out["status"])
	}
	if v, ok := out["vectorCount"].(float64); !ok || v < 1 {
		t.Errorf("vectorCount: got %v, want >= 1", out["vectorCount"])
	}
	if v, ok := out["hybridCount"].(float64); !ok || v < 1 {
		t.Errorf("hybridCount: got %v, want >= 1", out["hybridCount"])
	}
}

// TestWorkspaceDynamicFactory tests dynamic workspace factory resolved per generate() call.
func TestWorkspaceDynamicFactory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	wsPath := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(wsPath, "README.md"), []byte("# Test Project"), 0644)
	os.WriteFile(filepath.Join(wsPath, "main.go"), []byte("package main"), 0644)

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"WORKSPACE_PATH": wsPath,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/workspace-dynamic-factory.js")
	result, err := kit.EvalModule(context.Background(), "workspace-dynamic-factory.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("Dynamic factory: %v", out)

	if out["error"] != nil {
		t.Fatalf("dynamic factory error: %v\nstack: %v", out["error"], out["stack"])
	}
	if out["factoryCalled"] != true {
		t.Errorf("factory should have been called, got %v", out["factoryCalled"])
	}
	if out["hasResponse"] != true {
		t.Errorf("should have a response, got %v", out["hasResponse"])
	}
}

// TestWorkspaceSkillsConfig tests skills discovery via workspace config.
func TestWorkspaceSkillsConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	wsPath := t.TempDir()

	// Create a skills directory with a SKILL.md
	skillDir := filepath.Join(wsPath, "skills", "code-review")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: code-review
description: Review code for quality and security
---

## Instructions

When reviewing code:
1. Check for security vulnerabilities
2. Verify error handling
3. Look for performance issues
`), 0644)

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"WORKSPACE_PATH": wsPath,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/workspace-skills-config.js")
	result, err := kit.EvalModule(context.Background(), "workspace-skills-config.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]string
	json.Unmarshal([]byte(result), &out)
	t.Logf("Skills config: %v", out)

	if out["error"] != "" {
		t.Fatalf("skills error: %s", out["error"])
	}
	if out["create"] != "ok" {
		t.Errorf("create: %v", out["create"])
	}
}

// TestWorkspaceAllowedPaths tests LocalFilesystem.setAllowedPaths() runtime update.
func TestWorkspaceAllowedPaths(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	wsPath := t.TempDir()
	extraPath := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(wsPath, "test.txt"), []byte("base content"), 0644)
	os.WriteFile(filepath.Join(extraPath, "extra.txt"), []byte("extra content"), 0644)

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
		EnvVars: map[string]string{
			"WORKSPACE_PATH": wsPath,
			"EXTRA_PATH":     extraPath,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/workspace-allowed-paths.js")
	result, err := kit.EvalModule(context.Background(), "workspace-allowed-paths.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]string
	json.Unmarshal([]byte(result), &out)
	t.Logf("Allowed paths: %v", out)

	if out["error"] != "" {
		t.Fatalf("error: %s", out["error"])
	}
	if !strings.HasPrefix(out["readBase"], "ok:") {
		t.Errorf("readBase should succeed: %v", out["readBase"])
	}
	if !strings.HasPrefix(out["readExtraBefore"], "blocked:") {
		t.Errorf("readExtra before setAllowedPaths should be blocked: %v", out["readExtraBefore"])
	}
	if out["setAllowedPaths"] != "ok" {
		t.Errorf("setAllowedPaths: %v", out["setAllowedPaths"])
	}
	if !strings.HasPrefix(out["readExtraAfter"], "ok:") {
		t.Errorf("readExtra after setAllowedPaths should succeed: %v", out["readExtraAfter"])
	}
}

// TestRunEvals tests batch evaluation with runEvals().
func TestRunEvals(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/eval-run-evals.js")
	result, err := kit.EvalModule(context.Background(), "eval-run-evals.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("runEvals: %v", out)

	if out["error"] != nil {
		t.Fatalf("runEvals error: %v\nstack: %v", out["error"], out["stack"])
	}
	if out["status"] != "ok" {
		t.Errorf("status: %v", out["status"])
	}
	if out["hasScores"] != "ok" {
		t.Errorf("hasScores: %v", out["hasScores"])
	}
	if v, ok := out["totalItems"].(float64); !ok || v != 3 {
		t.Errorf("totalItems: got %v, want 3", out["totalItems"])
	}
	if out["hasAccuracy"] != "ok" {
		t.Errorf("accuracy score should be a number: %v", out["hasAccuracy"])
	}
	if v, ok := out["accuracyScore"].(float64); ok {
		t.Logf("accuracy score: %.2f (1.0 = all items matched ground truth)", v)
	}
}

// TestProcessorsBuiltin tests that built-in processors are available and constructible.
func TestProcessorsBuiltin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/processors-builtin.js")
	result, err := kit.EvalModule(context.Background(), "processors-builtin.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("Processors: %v", out)

	if out["unicodeNormalizer"] != "ok" {
		t.Errorf("UnicodeNormalizer: %v", out["unicodeNormalizer"])
	}
	if out["tokenLimiter"] != "ok" {
		t.Errorf("TokenLimiterProcessor: %v", out["tokenLimiter"])
	}
	if out["toolCallFilter"] != "ok" {
		t.Errorf("ToolCallFilter: %v", out["toolCallFilter"])
	}
	if out["batchParts"] != "ok" {
		t.Errorf("BatchPartsProcessor: %v", out["batchParts"])
	}
	if v, ok := out["availableCount"].(float64); !ok || v < 11 {
		t.Errorf("expected 11 processors available, got %v", out["availableCount"])
	}
	t.Logf("Available: %v", out["availableList"])
}

// TestAgentSubagents tests agent networks / sub-agent delegation.
func TestAgentSubagents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/agent-subagents.js")
	result, err := kit.EvalModule(context.Background(), "agent-subagents.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("Subagents: %v", out)

	if out["error"] != nil {
		t.Fatalf("subagent error: %v\nstack: %v", out["error"], out["stack"])
	}
	if out["status"] != "ok" {
		t.Errorf("status: %v", out["status"])
	}
	if out["hasAnswer"] != "ok" {
		t.Errorf("should contain 105: %v", out["hasAnswer"])
	}
}

// TestAgentConstrainedSubagents tests the createSubagent() + subagents config pattern.
func TestAgentConstrainedSubagents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/agent-constrained-subagents.js")
	result, err := kit.EvalModule(context.Background(), "agent-constrained-subagents.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	t.Logf("Constrained subagents: %v", out)

	if out["error"] != nil {
		t.Fatalf("error: %v\nstack: %v", out["error"], out["stack"])
	}
	if out["status"] != "ok" {
		t.Errorf("status: %v", out["status"])
	}
	if out["hasResponse"] != "ok" {
		t.Errorf("hasResponse: %v", out["hasResponse"])
	}
	if out["hasStartEvent"] != "ok" {
		t.Errorf("should have start event: %v", out["hasStartEvent"])
	}
	if out["hasEndEvent"] != "ok" {
		t.Errorf("should have end event: %v", out["hasEndEvent"])
	}
	if out["explorerUsed"] != "ok" {
		t.Errorf("explorer subagent should have been used: %v", out["explorerUsed"])
	}
	t.Logf("Events: %v", out["eventCount"])
}
