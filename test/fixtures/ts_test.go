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
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestTSFixturesTranspile verifies every TS fixture transpiles to valid JS.
// Walks fixtures/ts/<category>/<fixture>/ two levels deep.
func TestTSFixturesTranspile(t *testing.T) {
	categories, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts"))
	require.NoError(t, err)

	for _, catEntry := range categories {
		if !catEntry.IsDir() {
			continue
		}
		category := catEntry.Name()
		fixtures, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts", category))
		if err != nil {
			continue
		}
		for _, fixEntry := range fixtures {
			if !fixEntry.IsDir() {
				continue
			}
			name := fixEntry.Name()
			t.Run(category+"/"+name, func(t *testing.T) {
				js := loadTSFixture(t, category, name)
				require.NotEmpty(t, js, "transpiled output should not be empty")
			})
		}
	}
}

// fixtureNeedsAI returns true if the fixture needs an AI API key.
func fixtureNeedsAI(category, name string) bool {
	// These categories always need AI
	switch category {
	case "agent", "ai", "observability", "composition":
		return true
	case "memory":
		// Storage-only tests that use Agent for conversation need AI
		switch name {
		case "inmemory-basic", "libsql-basic", "libsql-local",
			"postgres-basic", "postgres-scram",
			"mongodb-basic", "mongodb-scram",
			"upstash-basic", "semantic-recall", "working-memory":
			return true
		}
	case "workflow":
		return name == "with-agent-step"
	case "rag":
		return name == "vector-query-tool"
	}
	return false
}

// fixtureNeedsContainer returns true if the fixture needs an external container (Podman).
func fixtureNeedsContainer(category, name string) bool {
	switch category {
	case "memory":
		switch name {
		case "postgres-basic", "postgres-scram",
			"mongodb-basic", "mongodb-scram",
			"semantic-recall", "working-memory":
			return true
		}
	case "agent":
		return name == "with-memory-postgres" || name == "with-memory-mongodb"
	case "vector":
		return true // all vector fixtures need containers
	case "rag":
		return name == "vector-query-tool"
	}
	return false
}

// fixtureNeedsVectorExtensions returns true if the fixture needs libsql with vector support.
func fixtureNeedsVectorExtensions(category, name string) bool {
	switch category {
	case "memory":
		return name == "semantic-recall" || name == "working-memory"
	case "vector":
		return strings.HasPrefix(name, "libsql")
	case "rag":
		return name == "vector-query-tool"
	}
	return false
}

// TestTSFixturesE2E runs the full pipeline: transpile → deploy → get output → assert.
// Walks fixtures/ts/<category>/<fixture>/ two levels deep.
func TestTSFixturesE2E(t *testing.T) {
	testutil.LoadEnv(t)

	categories, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts"))
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

			// MongoDB with SCRAM-SHA-256 authentication
			mongoAddr := testutil.StartContainer(t, "mongo:7", "27017/tcp", nil,
				wait.ForLog("Waiting for connections").WithStartupTimeout(60*time.Second),
				"MONGO_INITDB_ROOT_USERNAME=test", "MONGO_INITDB_ROOT_PASSWORD=test")
			os.Setenv("MONGODB_URL", "mongodb://test:test@"+mongoAddr)
			os.Setenv("MONGODB_LOG_ALL", "off")
			t.Logf("MongoDB container started: %s", mongoAddr)

			waitForTCP(t, mongoAddr, 15*time.Second)
			t.Logf("MongoDB TCP probe passed")

			// libsql-server for vector extension fixtures
			libsqlAddr := testutil.StartContainer(t,
				"ghcr.io/tursodatabase/libsql-server:latest",
				"8080/tcp",
				[]string{"sqld", "--http-listen-addr", "0.0.0.0:8080"},
				wait.ForHTTP("/health").WithStartupTimeout(30*time.Second))
			os.Setenv("LIBSQL_VECTOR_URL", "http://"+libsqlAddr)
			t.Logf("libsql-server container started: %s", libsqlAddr)
		})
	}

	// Skip categories that have their own runners (cross-kit, plugin)
	skipCategories := map[string]bool{
		"cross-kit": true,
		"plugin":    true,
	}

	for _, catEntry := range categories {
		if !catEntry.IsDir() {
			continue
		}
		category := catEntry.Name()
		if skipCategories[category] {
			continue
		}

		fixtures, err := os.ReadDir(filepath.Join(fixturesRoot(t), "ts", category))
		if err != nil {
			continue
		}

		for _, fixEntry := range fixtures {
			if !fixEntry.IsDir() {
				continue
			}
			name := fixEntry.Name()
			fullName := category + "/" + name

			t.Run(fullName, func(t *testing.T) {
				if fixtureNeedsAI(category, name) && !hasAI {
					t.Skipf("needs OPENAI_API_KEY")
				}
				if fixtureNeedsContainer(category, name) {
					if !hasPodman {
						t.Skipf("needs Podman for containers")
					}
					startContainers()
				}
				if fixtureNeedsVectorExtensions(category, name) {
					if !hasPodman {
						t.Skipf("needs Podman for libsql-server container")
					}
					startContainers()
				}
				// Skip fixtures needing cloud credentials
				if strings.Contains(name, "upstash") && os.Getenv("UPSTASH_REDIS_REST_URL") == "" {
					t.Skipf("needs UPSTASH_REDIS_REST_URL in .env")
				}

				// 1. Read raw .ts source
				tsSource := loadTSFixtureRaw(t, category, name)

				// 2. Create kernel — mcp fixtures get a kernel with an in-process MCP server
				var tk *testutil.TestKernel
				if category == "mcp" {
					tk = newTestKernelWithMCP(t)
				} else {
					tk = testutil.NewTestKernelFull(t)
				}
				registerFixtureTools(t, tk, category, name)

				// 3. Inject infra URLs into env
				if (category == "memory" || category == "agent") && strings.Contains(name, "libsql") {
					libsqlURL := tk.StorageURL("default")
					if libsqlURL != "" {
						os.Setenv("LIBSQL_URL", libsqlURL)
					}
				}
				if fixtureNeedsVectorExtensions(category, name) {
					vectorURL := os.Getenv("LIBSQL_VECTOR_URL")
					if vectorURL != "" {
						os.Setenv("LIBSQL_URL", vectorURL)
					}
				}

				// 4. Deploy .ts
				timeout := 15 * time.Second
				if fixtureNeedsAI(category, name) {
					timeout = 60 * time.Second
				}
				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()

				_, err := tk.Deploy(ctx, name+".ts", tsSource)
				if err != nil {
					if fixtureNeedsAI(category, name) && !hasAI {
						t.Skipf("deploy needs AI key: %v", err)
					}
					t.Fatalf("deploy %s: %v", fullName, err)
				}

				// 5. Read output
				raw, err := tk.EvalTS(ctx, "__read_output.ts",
					`return typeof globalThis.__module_result !== "undefined" ? globalThis.__module_result : ""`)
				require.NoError(t, err, "read output")

				if raw == "" {
					t.Logf("%s: no output (deploy-only fixture)", fullName)
					return
				}

				// 6. Parse output
				var actual map[string]any
				if err := json.Unmarshal([]byte(raw), &actual); err != nil {
					t.Logf("%s output (raw): %s", fullName, raw)
					return
				}
				t.Logf("%s output: %s", fullName, truncate(raw, 2000))

				// 7. Assert against expect.json if present
				expect := loadExpect(t, category, name)
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
							assert.NotNil(t, actualVal, "key %s should exist", key)
						} else if strings.HasPrefix(ev, "~") {
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
}

// registerFixtureTools registers Go tools that specific fixtures expect.
func registerFixtureTools(t *testing.T, tk *testutil.TestKernel, category, name string) {
	t.Helper()
	fullName := category + "/" + name

	switch fullName {
	case "tools/call-from-ts":
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
	case "agent/with-registered-tool":
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
	case "agent/hitl-bus-approval":
		// Go-side auto-approver: subscribes to the approval topic and auto-approves.
		// Uses sdk.Reply — the clean Go equivalent of JS msg.reply().
		cancel, subErr := sdk.SubscribeTo[json.RawMessage](tk, context.Background(), "test.approvals",
			func(payload json.RawMessage, msg messages.Message) {
				t.Logf("HITL: approval request received — auto-approving via sdk.Reply")
				sdk.Reply(tk, context.Background(), msg, map[string]bool{"approved": true})
			})
		if subErr != nil {
			t.Logf("HITL: failed to subscribe: %v", subErr)
		} else {
			t.Cleanup(func() { cancel() })
		}
	case "composition/full-agent-workflow-memory":
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

// newTestKernelWithMCP creates a Kernel with an in-process MCP server.
func newTestKernelWithMCP(t *testing.T) *testutil.TestKernel {
	t.Helper()

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

	httpServer := mcpserver.NewStreamableHTTPServer(s)
	ts := httptest.NewServer(httpServer)
	t.Cleanup(ts.Close)

	mcpURL := ts.URL + "/mcp"
	t.Logf("MCP server started at %s", mcpURL)

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
