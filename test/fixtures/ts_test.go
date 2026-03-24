package fixtures_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// fixtureNeedsInfra returns true if the fixture needs external containers.
func fixtureNeedsInfra(name string) bool {
	infraPrefixes := []string{
		"memory-libsql", "memory-postgres", "memory-mongodb", "memory-upstash",
		"memory-semantic", "memory-working",
		"vector-", "rag-vector", "observability-",
	}
	for _, prefix := range infraPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// TestTSFixturesE2E runs the full pipeline: transpile → deploy → get output → assert.
// Uses real OpenAI API key from .env. Skips fixtures that need missing infra.
func TestTSFixturesE2E(t *testing.T) {
	testutil.LoadEnv(t)

	entries, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts"))
	require.NoError(t, err)

	hasAI := testutil.HasAIKey()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			// Skip fixtures that need infra we don't have
			if fixtureNeedsAI(name) && !hasAI {
				t.Skipf("needs OPENAI_API_KEY")
			}
			if fixtureNeedsInfra(name) {
				t.Skipf("needs external containers")
			}
			// MCP fixtures need a running MCP server
			if name == "mcp-tools" {
				t.Skipf("needs MCP server")
			}

			// 1. Transpile
			js := loadTSFixture(t, name)

			// 2. Create kernel with tools the fixtures expect
			tk := testutil.NewTestKernelFull(t)
			registerFixtureTools(t, tk, name)

			// 3. Deploy with generous timeout (AI calls can be slow)
			timeout := 15 * time.Second
			if fixtureNeedsAI(name) {
				timeout = 60 * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			_, err := tk.Deploy(ctx, name+".ts", js)
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
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
