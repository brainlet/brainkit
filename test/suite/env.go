// Package suite provides the TestEnv abstraction for brainkit test suites.
// Each domain (bus, deploy, etc.) exports a Run(t, env) function.
// Standalone _test.go files create the right env for the memory fast path.
// Campaigns call Run() with different envs for backend combinations.
package suite

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/types"
	mcppkg "github.com/brainlet/brainkit/internal/mcp"
)

// TestEnv is the shared test environment for all suite domains.
type TestEnv struct {
	Kit    *brainkit.Kit
	Nodes  []*brainkit.Kit
	Config EnvConfig
	T      *testing.T
}

// EnvConfig describes what infrastructure a TestEnv provides.
type EnvConfig struct {
	Transport      string
	Storages       map[string]brainkit.StorageConfig
	Vectors        map[string]brainkit.VectorConfig
	Persistence    string // "none", "sqlite"
	FSRoot         bool
	NodeCount      int // 0=kernel-only, 1=node, 2+=crosskit
	Plugins        []brainkit.PluginConfig
	SecretKey      string
	AIProviders    bool // auto-detect from .env
	MCPServers     map[string]mcppkg.ServerConfig
	Tools          bool // register echo+add
	Tracing        bool
	BusRateLimits  map[string]float64
	RetryPolicies  map[string]brainkit.RetryPolicy
	LogHandler     func(brainkit.LogEntry)
	MaxConcurrency int
}

// EnvOption modifies an EnvConfig before TestEnv creation.
type EnvOption func(*EnvConfig)

func WithPersistence() EnvOption {
	return func(c *EnvConfig) { c.Persistence = "sqlite" }
}

func WithTracing() EnvOption {
	return func(c *EnvConfig) { c.Tracing = true }
}

func WithTransport(backend string) EnvOption {
	return func(c *EnvConfig) { c.Transport = backend }
}

func WithStorage(name string, cfg brainkit.StorageConfig) EnvOption {
	return func(c *EnvConfig) {
		if c.Storages == nil {
			c.Storages = make(map[string]brainkit.StorageConfig)
		}
		c.Storages[name] = cfg
	}
}

func WithVector(name string, cfg brainkit.VectorConfig) EnvOption {
	return func(c *EnvConfig) {
		if c.Vectors == nil {
			c.Vectors = make(map[string]brainkit.VectorConfig)
		}
		c.Vectors[name] = cfg
	}
}

func WithNodes(count int) EnvOption {
	return func(c *EnvConfig) { c.NodeCount = count }
}

func WithMCP(servers map[string]mcppkg.ServerConfig) EnvOption {
	return func(c *EnvConfig) { c.MCPServers = servers }
}

func WithSecretKey(key string) EnvOption {
	return func(c *EnvConfig) { c.SecretKey = key }
}

func WithRateLimits(limits map[string]float64) EnvOption {
	return func(c *EnvConfig) { c.BusRateLimits = limits }
}

func WithRetryPolicies(policies map[string]brainkit.RetryPolicy) EnvOption {
	return func(c *EnvConfig) { c.RetryPolicies = policies }
}

func WithLogHandler(fn func(brainkit.LogEntry)) EnvOption {
	return func(c *EnvConfig) { c.LogHandler = fn }
}

func WithMaxConcurrency(n int) EnvOption {
	return func(c *EnvConfig) { c.MaxConcurrency = n }
}

func WithFSRoot() EnvOption {
	return func(c *EnvConfig) { c.FSRoot = true }
}

func WithPlugins(configs ...brainkit.PluginConfig) EnvOption {
	return func(c *EnvConfig) { c.Plugins = configs }
}

func WithAI() EnvOption {
	return func(c *EnvConfig) { c.AIProviders = true }
}

// Full creates a fully-configured TestEnv matching what NewTestKitFull does:
// storage (SQLite + InMemory), vectors (SQLite), AI auto-detect, FSRoot, echo+add tools.
func Full(t *testing.T, opts ...EnvOption) *TestEnv {
	t.Helper()
	testutil.LoadEnv(t)
	tmpDir := t.TempDir()

	cfg := EnvConfig{
		FSRoot:      true,
		AIProviders: true,
		Tools:       true,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "brainkit.db")),
			"mem":     brainkit.InMemoryStorage(),
		},
		Vectors: map[string]brainkit.VectorConfig{
			"default": brainkit.SQLiteVector(filepath.Join(tmpDir, "brainkit.db")),
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	return NewEnv(t, cfg)
}

// Minimal creates a bare Kit with no storage, no vectors, no AI, no tools.
func Minimal(t *testing.T, opts ...EnvOption) *TestEnv {
	t.Helper()
	testutil.LoadEnv(t)

	cfg := EnvConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	return NewEnv(t, cfg)
}

// NewEnv creates a TestEnv from a fully specified config.
func NewEnv(t *testing.T, cfg EnvConfig) *TestEnv {
	t.Helper()

	tmpDir := t.TempDir()

	// Build providers from env
	var providers []brainkit.ProviderConfig
	envVars := make(map[string]string)
	if cfg.AIProviders {
		if key := os.Getenv("OPENAI_API_KEY"); key != "" {
			providers = append(providers, brainkit.OpenAI(key))
			envVars["OPENAI_API_KEY"] = key
		}
	}

	fsRoot := ""
	if cfg.FSRoot {
		fsRoot = tmpDir
	}

	kitCfg := brainkit.Config{
		Namespace:      "test",
		CallerID:       "test-caller",
		FSRoot:         fsRoot,
		Providers:      providers,
		EnvVars:        envVars,
		Storages:       cfg.Storages,
		Vectors:        cfg.Vectors,
		RetryPolicies:  cfg.RetryPolicies,
		LogHandler:     cfg.LogHandler,
		MaxConcurrency: cfg.MaxConcurrency,
	}

	// Persistence: KitStore backed by SQLite
	if cfg.Persistence == "sqlite" {
		storePath := filepath.Join(tmpDir, "kitstore.db")
		store, err := brainkit.NewSQLiteStore(storePath)
		if err != nil {
			t.Fatalf("suite.NewEnv: open store: %v", err)
		}
		t.Cleanup(func() { store.Close() })
		kitCfg.Store = store
	}

	// Secret key
	if cfg.SecretKey != "" {
		kitCfg.SecretKey = cfg.SecretKey
	}

	// Tracing
	if cfg.Tracing {
		kitCfg.TraceStore = tracing.NewMemoryTraceStore(1000)
	}

	// MCP servers
	if len(cfg.MCPServers) > 0 {
		kitCfg.MCPServers = cfg.MCPServers
	}

	// Transport: default to memory (fast GoChannel) for suite tests.
	// Campaigns override with WithTransport("nats"), WithTransport("embedded"), etc.
	if cfg.Transport == "" || cfg.Transport == "memory" {
		kitCfg.Transport = brainkit.Memory()
	} else {
		tcfg := testutil.TransportConfigForBackend(t, cfg.Transport)

		// "embedded" needs no container — skip probe
		if cfg.Transport != "embedded" {
			probe := testutil.MustCreateTransport(t, tcfg)
			testutil.WaitForBackendReady(t, probe)
			probe.Close()
		}

		kitCfg.Transport = testutil.BrainkitTransport(tcfg)
	}

	kit, err := brainkit.New(kitCfg)
	if err != nil {
		t.Fatalf("suite.NewEnv: brainkit.New: %v", err)
	}
	t.Cleanup(func() { kit.Close() })

	// Register test tools
	if cfg.Tools {
		if err := brainkit.RegisterTool(kit, "echo", tools.TypedTool[testutil.EchoInput]{
			Description: "echoes the input message",
			Execute: func(ctx context.Context, input testutil.EchoInput) (any, error) {
				return map[string]string{"echoed": input.Message}, nil
			},
		}); err != nil {
			t.Fatalf("suite.NewEnv: register echo: %v", err)
		}

		if err := brainkit.RegisterTool(kit, "add", tools.TypedTool[testutil.AddInput]{
			Description: "adds two numbers",
			Execute: func(ctx context.Context, input testutil.AddInput) (any, error) {
				return map[string]int{"sum": input.A + input.B}, nil
			},
		}); err != nil {
			t.Fatalf("suite.NewEnv: register add: %v", err)
		}
	}

	return &TestEnv{
		Kit:    kit,
		Config: cfg,
		T:      t,
	}
}

// --- Shared test helpers ---

// Deploy deploys .ts code via bus command and returns any error.
func (e *TestEnv) Deploy(source, code string) error {
	return testutil.DeployErr(e.Kit, source, code)
}

// EvalTS evaluates TypeScript code and returns the result string.
func (e *TestEnv) EvalTS(code string) (string, error) {
	return testutil.EvalTSErr(e.Kit, "__suite_eval.ts", code)
}

// PublishAndWait publishes a typed message and waits for the reply payload.
func (e *TestEnv) PublishAndWait(t *testing.T, msg sdk.BrainkitMessage, timeout time.Duration) (json.RawMessage, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	pr, err := sdk.Publish(e.Kit, ctx, msg)
	if err != nil {
		return nil, err
	}

	ch := make(chan json.RawMessage, 1)
	unsub, err := e.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) {
		select {
		case ch <- json.RawMessage(m.Payload):
		default:
		}
	})
	if err != nil {
		return nil, err
	}
	defer unsub()

	select {
	case payload := <-ch:
		return payload, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SendAndReceive publishes a typed message and waits for the raw response.
func (e *TestEnv) SendAndReceive(t *testing.T, msg sdk.BrainkitMessage, timeout time.Duration) (json.RawMessage, bool) {
	t.Helper()
	payload, err := e.PublishAndWait(t, msg, timeout)
	if err != nil {
		t.Logf("SendAndReceive: %v", err)
		return nil, false
	}
	return payload, true
}

// ResponseCode extracts the error code from a bus response payload.
func ResponseCode(payload json.RawMessage) string {
	var resp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	json.Unmarshal(payload, &resp)
	return resp.Code
}

// ResponseHasError checks if a bus response contains an error field.
func ResponseHasError(payload json.RawMessage) bool {
	var resp struct {
		Error string `json:"error"`
	}
	json.Unmarshal(payload, &resp)
	return resp.Error != ""
}

// RequireAI skips the test if OPENAI_API_KEY is not set.
func (e *TestEnv) RequireAI(t *testing.T) {
	t.Helper()
	if !testutil.HasAIKey() {
		t.Skip("needs OPENAI_API_KEY")
	}
}

// RequirePodman skips the test if Podman is not available.
func (e *TestEnv) RequirePodman(t *testing.T) {
	t.Helper()
	if !testutil.PodmanAvailable() {
		t.Skip("needs Podman")
	}
}

// NewSQLiteStoreForTest creates a SQLite KitStore in tmpDir. Exposed for persistence tests.
func NewSQLiteStoreForTest(t *testing.T) types.KitStore {
	t.Helper()
	storePath := filepath.Join(t.TempDir(), "test-store.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	if err != nil {
		t.Fatalf("NewSQLiteStoreForTest: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}
