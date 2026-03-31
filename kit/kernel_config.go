package kit

import (
	"encoding/json"
	"time"

	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/internal/messaging"
	toolreg "github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/kit/rbac"
	"github.com/brainlet/brainkit/kit/registry"
	"github.com/brainlet/brainkit/kit/secrets"
	"github.com/brainlet/brainkit/kit/tracing"
)

// RetryPolicy configures retry behavior for failed bus handlers.
type RetryPolicy struct {
	MaxRetries      int           // 0 = no retry
	InitialDelay    time.Duration // delay before first retry (default: 1s)
	MaxDelay        time.Duration // cap on exponential backoff (default: 30s)
	BackoffFactor   float64       // multiplier per retry (default: 2.0)
	DeadLetterTopic string        // where exhausted messages go ("" = discard)
}

// ScheduleConfig configures a new schedule via Kernel.Schedule().
type ScheduleConfig struct {
	ID         string          // optional — auto-generated if empty
	Expression string          // "every 5m" or "in 30s"
	Topic      string          // bus topic to publish to
	Payload    json.RawMessage // message payload
	Source     string          // deployment source (auto-set from .ts)
}

// KernelConfig configures the local runtime.
// The Kernel is a resource pool — the Go developer fills it with providers,
// storages, and vectors. Deployed .ts code picks from the pool via
// storage("name"), vectorStore("name"), and model("provider", "id").
type KernelConfig struct {
	// Identity
	Namespace string
	CallerID  string

	// AI providers — explicit config for custom base URLs.
	// For simple API key usage, leave empty and set the env var
	// (e.g., OPENAI_API_KEY) — auto-detected from os.Getenv.
	AIProviders map[string]registry.AIProviderRegistration

	// EnvVars overrides os.Getenv for specific keys.
	// process.env already reads os.Getenv directly, so this is only needed
	// to override a key for THIS Kernel (e.g., different API key than OS default).
	EnvVars map[string]string

	// Storage pool — deployments pick via storage("name").
	// Multiple backends, multiple instances. SQLite backends auto-start
	// a libsql HTTP bridge transparently.
	Storages map[string]StorageConfig

	// Vector pool — deployments pick via vectorStore("name").
	Vectors map[string]VectorConfig

	// Filesystem sandbox root — deployments access via fs.read/write/list.
	FSRoot string

	// Secrets
	SecretStore secrets.SecretStore // pluggable secret backend; nil = auto-create from SecretKey
	SecretKey   string             // master key for EncryptedKVStore; "" = unencrypted dev mode

	// RBAC — bus-level role-based access control
	Roles       map[string]rbac.Role // named permission sets; nil = no enforcement
	DefaultRole string               // applied when Deploy doesn't specify a role; "" = "service"

	// Tracing
	TraceStore      tracing.TraceStore // nil = no tracing
	TraceSampleRate float64            // 0.0-1.0, default 1.0

	// Infrastructure
	MaxStackSize  int
	SharedTools   *toolreg.ToolRegistry
	MCPServers    map[string]mcppkg.ServerConfig
	Observability ObservabilityConfig
	Store         KitStore
	Probe         registry.ProbeConfig

	// RetryPolicies maps topic glob patterns to retry configurations.
	// When a bus handler throws, the matching policy determines retry behavior.
	// If no policy matches, an error response is sent immediately to the caller.
	RetryPolicies map[string]RetryPolicy

	// LogHandler receives tagged log entries from .ts Compartments, WASM modules,
	// and the Kernel. Called concurrently from multiple goroutines — must be safe.
	// nil = default (print to stdout via log.Printf).
	LogHandler func(LogEntry)

	// Transport is an optional external transport. If set, Kernel uses it instead of
	// creating its own internal GoChannel transport. Used by Node to inject NATS.
	Transport *messaging.Transport

	// DeferRouterStart skips starting the router during NewKernel.
	// Used by Node to register node-specific command bindings before starting.
	DeferRouterStart bool

	// MaxConcurrency limits concurrent bus handler invocations.
	// 0 = unlimited (default). Recommended: 100-1000 for production.
	MaxConcurrency int

	// BusRateLimits maps RBAC role names to publish rate limits (requests/second).
	// When a deployment's role exceeds its limit, bus.publish throws an error.
	// Roles not in this map have no rate limit. Example: {"service": 100, "gateway": 50}.
	BusRateLimits map[string]float64

	// ProviderKeyMapping maps secret names to AI provider names for rotation cache invalidation.
	// When secrets.rotate updates a key matching this map, the JS-side provider cache is refreshed.
	// If nil, uses built-in defaults (OPENAI_API_KEY→openai, ANTHROPIC_API_KEY→anthropic, etc.).
	ProviderKeyMapping map[string]string

	// Plugin registries — searched in order for packages.search/install.
	// Defaults to the official brainlet registry if empty.
	PluginRegistries []RegistryConfig

	// Local plugin cache directory. Defaults to <FSRoot>/plugins/ if FSRoot is set.
	PluginDir string
}

// RegistryConfig configures a plugin registry source.
type RegistryConfig struct {
	Name      string // "official", "company", "community"
	URL       string // "https://raw.githubusercontent.com/brainlet/plugins-registry/main/v1"
	AuthToken string // optional — sent as Authorization: Bearer <token>
}

// DefaultRegistry is the official brainlet plugin registry.
var DefaultRegistry = RegistryConfig{
	Name: "official",
	URL:  "https://raw.githubusercontent.com/brainlet/plugins-registry/main/v1",
}

// ObservabilityConfig configures the tracing/observability system.
type ObservabilityConfig struct {
	Enabled     *bool
	Strategy    string
	ServiceName string
}
