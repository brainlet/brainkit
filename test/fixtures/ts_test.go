package fixtures_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/kit"
	provreg "github.com/brainlet/brainkit/kit/registry"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
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
		"memory-postgres":       true, // needs Postgres container
		"memory-postgres-scram": true, // needs Postgres container
		"memory-mongodb":        true, // needs MongoDB container
		"vector-pgvector":       true, // needs Postgres with pgvector
		"vector-mongodb":        true, // needs MongoDB container
		"memory-semantic-recall": true, // needs libsql-server with vector32()
		"memory-working":        true, // needs libsql-server with vector32()
		"vector-methods":        true, // needs libsql-server with vector32()
		"rag-vector-query-tool": true, // needs libsql-server with vector32()
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
			t.Logf("Postgres container started: %s", pgURL)

			// Start MongoDB without auth — our QuickJS crypto polyfills don't fully support
			// SCRAM-SHA-256 (pbkdf2Sync + saslprep). No-auth mode lets us test the driver
			// wire protocol, connection pooling, and CRUD operations end-to-end.
			mongoAddr := testutil.StartContainer(t, "mongo:7", "27017/tcp", nil,
				wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second))
			os.Setenv("MONGODB_URL", "mongodb://"+mongoAddr)
			t.Logf("MongoDB container started: %s", mongoAddr)

			// TCP health probe — verify ports actually accept connections
			// Container log messages can appear before the service is truly ready.
			waitForTCP(t, mongoAddr, 15*time.Second)
			t.Logf("MongoDB TCP probe passed")

			// Start libsql-server for vector extension fixtures (memory-semantic-recall,
			// memory-working, vector-methods, rag-vector-query-tool).
			// The embedded libsql bridge uses modernc.org/sqlite which lacks vector32().
			// The containerized libsql-server has full vector extension support.
			libsqlAddr := testutil.StartContainer(t,
				"ghcr.io/tursodatabase/libsql-server:latest",
				"8080/tcp",
				[]string{"sqld", "--http-listen-addr", "0.0.0.0:8080"},
				wait.ForHTTP("/health").WithStartupTimeout(30*time.Second))
			os.Setenv("LIBSQL_VECTOR_URL", "http://"+libsqlAddr)
			t.Logf("libsql-server container started: %s", libsqlAddr)
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
				if !hasPodman {
					t.Skipf("needs Podman for libsql-server container")
				}
				startContainers()
			}
			if name == "memory-upstash" && os.Getenv("UPSTASH_REDIS_REST_URL") == "" {
				t.Skipf("needs UPSTASH_REDIS_REST_URL in .env")
			}

			// 1. Read raw .ts source — Deploy handles transpile + import strip
			tsSource := loadTSFixtureRaw(t, name)

			// 2. Create kernel — mcp-tools gets a kernel with an in-process MCP server
			var tk *testutil.TestKernel
			if name == "mcp-tools" {
				tk = newTestKernelWithMCP(t)
			} else {
				tk = testutil.NewTestKernelFull(t)
			}
			registerFixtureTools(t, tk, name)

			// 3. Inject infra URLs into env (process.env reads from os.Getenv)
			if strings.HasPrefix(name, "memory-libsql") {
				// memory-libsql* fixtures use the embedded bridge (no vector extensions needed)
				libsqlURL := tk.StorageURL("default")
				if libsqlURL != "" {
					os.Setenv("LIBSQL_URL", libsqlURL)
				}
			}
			if fixtureNeedsVectorExtensions(name) {
				// Vector extension fixtures use the containerized libsql-server which has vector32()
				vectorURL := os.Getenv("LIBSQL_VECTOR_URL")
				if vectorURL != "" {
					os.Setenv("LIBSQL_URL", vectorURL)
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

// waitForTCP probes a TCP address until it accepts a connection or timeout expires.
// This catches the gap between container log readiness and actual port availability.
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// newTestKernelWithMCP creates a Kernel with an in-process MCP server for the mcp-tools fixture.
// The MCP server exposes a single "echo" tool and is served over HTTP using mcp-go's
// StreamableHTTPServer. The Kernel connects to it via the HTTP transport.
func newTestKernelWithMCP(t *testing.T) *testutil.TestKernel {
	t.Helper()

	// 1. Create MCP server with an echo tool
	s := mcpserver.NewMCPServer("testmcp", "1.0.0")
	s.AddTool(
		mcp.Tool{
			Name:        "echo",
			Description: "Echoes the input message",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"message": map[string]any{"type": "string", "description": "Message to echo"},
				},
			},
		},
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			msg, _ := args["message"].(string)
			result, _ := json.Marshal(map[string]string{"echoed": msg, "server": "testmcp"})
			return mcp.NewToolResultText(string(result)), nil
		},
	)

	// 2. Serve over HTTP using StreamableHTTPServer
	httpServer := mcpserver.NewStreamableHTTPServer(s)
	ts := httptest.NewServer(httpServer)
	t.Cleanup(ts.Close)

	mcpURL := ts.URL + "/mcp"
	t.Logf("MCP server started at %s", mcpURL)

	// 3. Create kernel with MCP server config
	testutil.LoadEnv(t)
	tmpDir := t.TempDir()

	aiProviders := make(map[string]provreg.AIProviderRegistration)
	envVars := make(map[string]string)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		aiProviders["openai"] = provreg.AIProviderRegistration{
			Type:   provreg.AIProviderOpenAI,
			Config: provreg.OpenAIProviderConfig{APIKey: key},
		}
		envVars["OPENAI_API_KEY"] = key
	}

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-mcp",
		WorkspaceDir: tmpDir,
		AIProviders:  aiProviders,
		EnvVars:      envVars,
		EmbeddedStorages: map[string]kit.EmbeddedStorageConfig{
			"default": {Path: filepath.Join(tmpDir, "brainkit.db")},
		},
		MCPServers: map[string]mcppkg.ServerConfig{
			"test": {URL: mcpURL},
		},
	})
	if err != nil {
		t.Fatalf("NewKernel with MCP: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	return &testutil.TestKernel{Kernel: k}
}
