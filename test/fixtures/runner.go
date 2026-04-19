package fixtures

import (
	"context"
	"encoding/json"
	"io/fs"
	"net"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	mcppkg "github.com/brainlet/brainkit/modules/mcp"
	"github.com/brainlet/brainkit/sdk"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/testcontainers/testcontainers-go/wait"
)

// skipCategories are fixture categories that have their own specialized runners
// (cross-kit, plugin) and must be excluded from the general runner.
var skipCategories = map[string]bool{
	"cross-kit": true,
	"plugin":    true,
}

// Runner discovers and executes TypeScript fixture tests.
// It replaces the 2-level os.ReadDir + 3 switch-case classifiers with
// filepath.WalkDir + path-based classification maps.
type Runner struct {
	root       string // absolute path to fixtures/ts/
	kitFactory func(t *testing.T, needs FixtureNeeds) *brainkit.Kit

	// toolRegistrar allows callers to register custom Go tools per fixture.
	// If nil, the default registerFixtureTools is used.
	toolRegistrar func(t *testing.T, k *brainkit.Kit, relPath string)
}

// NewRunner creates a runner for the given fixtures root.
// root should point to the fixtures/ directory (NOT fixtures/ts/).
func NewRunner(root string) *Runner {
	return &Runner{
		root: filepath.Join(root, "ts"),
	}
}

// WithKitFactory sets a custom Kit factory for campaigns.
// The factory receives the classified FixtureNeeds and must return a fully
// configured Kit. The runner will NOT call Close — use t.Cleanup in the factory.
func (r *Runner) WithKitFactory(f func(t *testing.T, needs FixtureNeeds) *brainkit.Kit) *Runner {
	r.kitFactory = f
	return r
}

// WithToolRegistrar sets a custom tool registration function.
// relPath is the fixture's relative path from ts/ (e.g. "tools/call-from-ts").
func (r *Runner) WithToolRegistrar(f func(t *testing.T, k *brainkit.Kit, relPath string)) *Runner {
	r.toolRegistrar = f
	return r
}

// RunAll discovers and runs all fixtures under the ts/ root.
func (r *Runner) RunAll(t *testing.T) {
	t.Helper()
	r.run(t, nil)
}

// RunMatching discovers and runs fixtures matching glob patterns.
// Patterns are matched against the relative path from ts/ (e.g. "agent/*", "memory/postgres-*").
func (r *Runner) RunMatching(t *testing.T, patterns ...string) {
	t.Helper()
	r.run(t, patterns)
}

// fixtureEntry represents a discovered fixture ready for execution.
type fixtureEntry struct {
	relPath string       // e.g. "agent/generate/basic"
	needs   FixtureNeeds // classified infrastructure requirements
}

// run is the core execution engine.
func (r *Runner) run(t *testing.T, patterns []string) {
	t.Helper()
	testutil.LoadEnv(t)

	// Discover fixtures
	fixtures := r.discover(t, patterns)
	if len(fixtures) == 0 {
		t.Skipf("no fixtures found matching patterns %v", patterns)
		return
	}

	t.Logf("discovered %d fixtures", len(fixtures))

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

	// Execute fixtures
	for _, fix := range fixtures {
		fix := fix // capture
		t.Run(fix.relPath, func(t *testing.T) {
			r.runFixture(t, fix, hasAI, hasPodman, startContainers)
		})
	}
}

// discover walks the fixtures tree and returns classified entries.
func (r *Runner) discover(t *testing.T, patterns []string) []fixtureEntry {
	t.Helper()

	var fixtures []fixtureEntry

	err := filepath.WalkDir(r.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "index.ts" {
			return nil
		}

		// Compute relative path from ts/ root to the fixture directory.
		// path is like /abs/fixtures/ts/agent/generate/basic/index.ts
		// We want "agent/generate/basic"
		dir := filepath.Dir(path)
		relPath, err := filepath.Rel(r.root, dir)
		if err != nil {
			return err
		}
		// Normalize separators for Windows compatibility
		relPath = filepath.ToSlash(relPath)

		// Check if this is in a skipped category
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) > 0 && skipCategories[parts[0]] {
			return nil
		}

		// Match against patterns if any
		if len(patterns) > 0 && !matchesAny(relPath, patterns) {
			return nil
		}

		needs := ClassifyFixture(relPath)
		fixtures = append(fixtures, fixtureEntry{
			relPath: relPath,
			needs:   needs,
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk fixtures: %v", err)
	}

	return fixtures
}

// runFixture executes a single fixture: skip checks, Kit creation, deploy, eval, assert.
func (r *Runner) runFixture(t *testing.T, fix fixtureEntry, hasAI, hasPodman bool, startContainers func()) {
	t.Helper()

	needs := fix.needs

	// Skip checks
	if needs.AI && !hasAI {
		t.Skipf("needs OPENAI_API_KEY")
	}
	if needs.Container != "" && !hasPodman {
		t.Skipf("needs Podman for %s container", needs.Container)
	}
	if needs.LibSQLServer && !hasPodman {
		t.Skipf("needs Podman for libsql-server container")
	}
	if needs.Credential != "" && os.Getenv(needs.Credential) == "" {
		t.Skipf("needs %s in .env", needs.Credential)
	}

	// Start containers if needed
	if needs.Container != "" || needs.LibSQLServer {
		startContainers()
	}

	// 1. Read raw .ts source
	tsSource := LoadTSFixtureRaw(t, fix.relPath)

	// 2. Create Kit
	var k *brainkit.Kit
	if r.kitFactory != nil {
		k = r.kitFactory(t, needs)
	} else {
		k = r.defaultKit(t, needs)
	}

	// 3. Register fixture-specific tools
	if r.toolRegistrar != nil {
		r.toolRegistrar(t, k, fix.relPath)
	} else {
		registerFixtureTools(t, k, fix.relPath)
	}

	// 4. Inject infra URLs into env
	r.injectInfraEnv(t, k, fix)

	// 5. Deploy .ts
	name := filepath.Base(fix.relPath)
	if err := testutil.DeployErr(k, name+".ts", tsSource); err != nil {
		if needs.AI && !hasAI {
			t.Skipf("deploy needs AI key: %v", err)
		}
		t.Fatalf("deploy %s: %v", fix.relPath, err)
	}

	// 6. Read output
	raw := testutil.EvalTS(t, k, "__read_output.ts",
		`return typeof globalThis.__module_result !== "undefined" ? globalThis.__module_result : ""`)

	if raw == "" {
		t.Logf("%s: no output (deploy-only fixture)", fix.relPath)
		return
	}

	// 7. Parse output
	var actual map[string]any
	if err := json.Unmarshal([]byte(raw), &actual); err != nil {
		t.Logf("%s output (raw): %s", fix.relPath, raw)
		return
	}
	t.Logf("%s output: %s", fix.relPath, Truncate(raw, 2000))

	// 8. Assert against expect.json if present
	expect := LoadExpect(t, fix.relPath)
	if expect == nil {
		return
	}
	AssertExpect(t, fix.relPath, actual, expect)
}

// defaultKit creates a Kit with the appropriate configuration for the fixture's needs.
func (r *Runner) defaultKit(t *testing.T, needs FixtureNeeds) *brainkit.Kit {
	t.Helper()

	if needs.MCP {
		return newKitWithMCP(t)
	}

	tk := testutil.NewTestKitFull(t)
	return tk.Kit
}

// injectInfraEnv sets environment variables that specific fixtures expect
// (e.g. LIBSQL_URL for memory/agent libsql fixtures).
func (r *Runner) injectInfraEnv(t *testing.T, k *brainkit.Kit, fix fixtureEntry) {
	t.Helper()

	_ = k // Kit doesn't expose StorageURL; URL comes from container env vars.

	parts := strings.SplitN(fix.relPath, "/", 2)
	category := ""
	name := ""
	if len(parts) >= 1 {
		category = parts[0]
	}
	if len(parts) >= 2 {
		name = filepath.Base(fix.relPath)
	}

	// For libsql fixtures the URL comes from the LIBSQL_VECTOR_URL env var
	// set by the container startup, not from the Kit.
	if (category == "memory" || category == "agent") && strings.Contains(name, "libsql") {
		if url := os.Getenv("LIBSQL_VECTOR_URL"); url != "" {
			os.Setenv("LIBSQL_URL", url)
		}
	}
	if fix.needs.LibSQLServer {
		vectorURL := os.Getenv("LIBSQL_VECTOR_URL")
		if vectorURL != "" {
			os.Setenv("LIBSQL_URL", vectorURL)
		}
	}
}

// ── Kit factories ─────────────────────────────────────────────────────────

// newKitWithMCP creates a Kit with an in-process MCP server.
func newKitWithMCP(t *testing.T) *brainkit.Kit {
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

	var providers []brainkit.ProviderConfig
	envVars := make(map[string]string)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		providers = append(providers, brainkit.OpenAI(key))
		envVars["OPENAI_API_KEY"] = key
	}

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-mcp",
		FSRoot:    tmpDir,
		Providers: providers,
		EnvVars:   envVars,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
		},
		Modules: []brainkit.Module{
			mcppkg.New(map[string]mcppkg.ServerConfig{
				"test": {URL: mcpURL},
			}),
		},
	})
	if err != nil {
		t.Fatalf("New Kit with MCP: %v", err)
	}
	t.Cleanup(func() { k.Close() })

	return k
}

// ── Fixture tool registration ─────────────────────────────────────────────

// registerFixtureTools registers Go tools that specific fixtures expect.
// relPath is the fixture's relative path from ts/ (e.g. "tools/call-from-ts").
func registerFixtureTools(t *testing.T, k *brainkit.Kit, relPath string) {
	t.Helper()

	switch relPath {
	case "tools/call-from-ts":
		brainkit.RegisterTool(k, "uppercase", brainkit.TypedTool[struct {
			Text string `json:"text"`
		}]{
			Description: "converts text to uppercase",
			Execute: func(ctx context.Context, input struct {
				Text string `json:"text"`
			}) (any, error) {
				return map[string]string{"text": strings.ToUpper(input.Text)}, nil
			},
		})
	case "agent/tools/with-registered-tool":
		brainkit.RegisterTool(k, "multiply", brainkit.TypedTool[struct {
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
	case "agent/hitl/bus-approval":
		// Go-side auto-approver: subscribes to the approval topic and auto-approves.
		// Uses sdk.Reply — the clean Go equivalent of JS msg.reply().
		cancel, subErr := sdk.SubscribeTo[json.RawMessage](k, context.Background(), "test.approvals",
			func(payload json.RawMessage, msg sdk.Message) {
				t.Logf("HITL: approval request received — auto-approving via sdk.Reply")
				sdk.Reply(k, context.Background(), msg, map[string]bool{"approved": true})
			})
		if subErr != nil {
			t.Logf("HITL: failed to subscribe: %v", subErr)
		} else {
			t.Cleanup(func() { cancel() })
		}
	case "composition/agent-workflow-memory":
		brainkit.RegisterTool(k, "reverse", brainkit.TypedTool[struct {
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

// ── Infrastructure helpers ────────────────────────────────────────────────

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

// matchesAny returns true if relPath matches any of the given glob patterns.
func matchesAny(relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, relPath)
		if err == nil && matched {
			return true
		}
		// Also try matching against individual segments for partial patterns
		// e.g. pattern "agent/*" should match "agent/generate/basic"
		if strings.Contains(pattern, "/") {
			// Direct match attempted above
			continue
		}
		// Pattern without / — match against the category
		parts := strings.SplitN(relPath, "/", 2)
		if len(parts) > 0 {
			catMatched, err := filepath.Match(pattern, parts[0])
			if err == nil && catMatched {
				return true
			}
		}
	}
	return false
}
