package fixtures_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestTSFixturesTranspile verifies every TS fixture transpiles to valid JS.
// Fast sanity check — no Kernel needed. Always runs.
func TestTSFixturesTranspile(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts"))
	require.NoError(t, err)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			js := loadTSFixture(t, name)
			require.NotEmpty(t, js, "transpiled output should not be empty")
		})
	}
}

// fixtureNeedsAI returns true if the fixture name suggests it needs an AI API key.
func fixtureNeedsAI(name string) bool {
	aiPrefixes := []string{
		"ai-", "agent-", "full-composition", "workflow-with-agent",
		"memory-", "observability-", "vector-", "rag-vector",
	}
	for _, prefix := range aiPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// fixtureNeedsContainer returns true if the fixture needs an external container (Podman).
func fixtureNeedsContainer(name string) bool {
	containers := map[string]bool{
		"memory-postgres":      true, // needs Postgres
		"memory-postgres-scram": true, // needs Postgres
		"memory-mongodb":       true, // needs MongoDB
		"memory-upstash":       true, // needs Upstash cloud creds
		"vector-pgvector":      true, // needs Postgres with pgvector
		"vector-mongodb":       true, // needs MongoDB
	}
	return containers[name]
}

// fixtureNeedsVectorExtensions returns true if the fixture needs libsql with vector support.
// The embedded libsql bridge uses modernc.org/sqlite which doesn't have vector extensions.
func fixtureNeedsVectorExtensions(name string) bool {
	vectorExts := map[string]bool{
		"memory-semantic-recall": true, // LibSQLVector needs vector32()
		"memory-working":        true, // LibSQLVector needs vector32()
		"vector-methods":        true, // LibSQLVector needs vector32()
		"rag-vector-query-tool": true, // LibSQLVector + embed
	}
	return vectorExts[name]
}

// TestTSFixturesE2E runs the full pipeline: transpile → deploy → get output → assert.
// Uses real OpenAI API key from .env. Starts containers once upfront (shared across subtests).
func TestTSFixturesE2E(t *testing.T) {
	testutil.LoadEnv(t)

	entries, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts"))
	require.NoError(t, err)

	hasAI := testutil.HasAIKey()
	hasPodman := testutil.PodmanAvailable()

	// Lazy container startup — only start when the first fixture that needs them runs.
	var containersOnce sync.Once
	startContainers := func() {
		containersOnce.Do(func() {
			if !hasPodman {
				return
			}
			testutil.CleanupOrphanedContainers(t)

			pgURL := testutil.StartPgVectorContainer(t)
			os.Setenv("POSTGRES_URL", pgURL)
			t.Logf("Postgres ready: %s", pgURL)

			mongoAddr := testutil.StartContainer(t, "mongo:7", "27017/tcp", nil,
				wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second),
				"MONGO_INITDB_ROOT_USERNAME=test", "MONGO_INITDB_ROOT_PASSWORD=test")
			os.Setenv("MONGODB_URL", "mongodb://test:test@"+mongoAddr)
			t.Logf("MongoDB ready: %s", mongoAddr)
		})
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			if fixtureNeedsAI(name) && !hasAI {
				t.Skipf("needs OPENAI_API_KEY")
			}
			if fixtureNeedsContainer(name) {
				if !hasPodman {
					t.Skipf("needs Podman for containers")
				}
				startContainers()
			}
			if fixtureNeedsVectorExtensions(name) {
				t.Skipf("needs libsql-server with vector extensions")
			}
			if name == "mcp-tools" {
				t.Skipf("needs running MCP server")
			}
			if name == "memory-upstash" {
				t.Skipf("needs Upstash cloud credentials")
			}

			// 1. Read raw .ts source — Deploy handles transpile + import strip
			tsSource := loadTSFixtureRaw(t, name)

			// 3. Create kernel with tools the fixtures expect
			tk := testutil.NewTestKernelFull(t)
			registerFixtureTools(t, tk, name)

			// 3. Inject infra URLs into env (process.env reads from os.Getenv)
			if strings.HasPrefix(name, "memory-libsql") || name == "vector-methods" || name == "rag-vector-query-tool" {
				libsqlURL := tk.StorageURL("default")
				if libsqlURL != "" {
					os.Setenv("LIBSQL_URL", libsqlURL)
				}
			}
			if name == "vector-pgvector" || name == "vector-mongodb" {
				// POSTGRES_URL and MONGODB_URL already set upfront from containers
			}

			// 5. Deploy .ts directly — Kernel detects .ts, transpiles, strips imports
			timeout := 15 * time.Second
			if fixtureNeedsAI(name) {
				timeout = 60 * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			_, err := tk.Deploy(ctx, name+".ts", tsSource)
			if err != nil {
				if fixtureNeedsAI(name) && !hasAI {
					t.Skipf("deploy needs AI key: %v", err)
				}
				t.Fatalf("deploy %s: %v", name, err)
			}

			// 4. Read output
			raw, err := tk.EvalTS(ctx, "__read_output.ts",
				`return typeof globalThis.__module_result !== "undefined" ? globalThis.__module_result : ""`)
			require.NoError(t, err, "read output")

			if raw == "" {
				// Some fixtures (bus-subscribe, tools-register-list) set output;
				// others are deploy-only (no output). Both are valid.
				t.Logf("%s: no output (deploy-only fixture)", name)
				return
			}

			// 5. Parse output
			var actual map[string]any
			if err := json.Unmarshal([]byte(raw), &actual); err != nil {
				// Output might be a plain string or number
				t.Logf("%s output (raw): %s", name, raw)
				return
			}
			t.Logf("%s output: %s", name, truncate(raw, 200))

			// 6. Assert against expect.json if present
			expect := loadExpect(t, "ts", name)
			if expect == nil {
				return
			}

			for key, expected := range expect {
				actualVal, exists := actual[key]
				if !exists {
					t.Errorf("missing key %q in output", key)
					continue
				}
				switch ev := expected.(type) {
				case bool:
					assert.Equal(t, ev, actualVal, "key %s", key)
				case float64:
					assert.InDelta(t, ev, actualVal, 0.01, "key %s", key)
				case string:
					if ev == "*" {
						// Wildcard: just check the key exists (any non-nil value)
						assert.NotNil(t, actualVal, "key %s should exist", key)
					} else if strings.HasPrefix(ev, "~") {
						// Contains check: ~substring
						assert.Contains(t, actualVal, ev[1:], "key %s", key)
					} else {
						assert.Equal(t, ev, actualVal, "key %s", key)
					}
				default:
					assert.Equal(t, expected, actualVal, "key %s", key)
				}
			}
		})
	}
}

// registerFixtureTools registers Go tools that specific fixtures expect.
func registerFixtureTools(t *testing.T, tk *testutil.TestKernel, name string) {
	t.Helper()

	switch name {
	case "tools-call":
		// Fixture calls tools.call("uppercase", { text: "hello brainlet" })
		registry.Register(tk.Tools, "uppercase", registry.TypedTool[struct {
			Text string `json:"text"`
		}]{
			Description: "converts text to uppercase",
			Execute: func(ctx context.Context, input struct {
				Text string `json:"text"`
			}) (any, error) {
				return map[string]string{"text": strings.ToUpper(input.Text)}, nil
			},
		})
	case "agent-with-registered-tool":
		// Fixture uses tool("multiply") — needs a multiply tool
		registry.Register(tk.Tools, "multiply", registry.TypedTool[struct {
			A float64 `json:"a"`
			B float64 `json:"b"`
		}]{
			Description: "multiplies two numbers",
			Execute: func(ctx context.Context, input struct {
				A float64 `json:"a"`
				B float64 `json:"b"`
			}) (any, error) {
				return map[string]float64{"result": input.A * input.B}, nil
			},
		})
	case "full-composition":
		// Fixture calls tools.call("reverse", { text: "..." })
		registry.Register(tk.Tools, "reverse", registry.TypedTool[struct {
			Text string `json:"text"`
		}]{
			Description: "reverses a string",
			Execute: func(ctx context.Context, input struct {
				Text string `json:"text"`
			}) (any, error) {
				runes := []rune(input.Text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				return map[string]string{"result": string(runes)}, nil
			},
		})
	}
}

// Containers are started once in TestTSFixturesE2E() before subtests run.
// No per-fixture container setup needed.

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
