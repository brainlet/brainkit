//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestWorkspaceToolRemapping tests workspace tool name remapping and enable/disable.
func TestWorkspaceToolRemapping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	key := requireKey(t)
	wsPath := t.TempDir()

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
