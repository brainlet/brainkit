package types

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/brainlet/brainkit/internal/tools"
)

// KernelConfig configures the local runtime.
type KernelConfig struct {
	// Identity
	ClusterID string
	RuntimeID string
	Namespace string
	CallerID  string

	// AI providers
	AIProviders map[string]AIProviderRegistration

	// EnvVars overrides os.Getenv for specific keys.
	EnvVars map[string]string

	// Storage pool
	Storages map[string]StorageConfig

	// Vector pool
	Vectors map[string]VectorConfig

	// Filesystem sandbox root
	FSRoot string

	// Secrets
	SecretStore SecretStore
	SecretKey   string

	// Tracing
	TraceStore      TraceStore
	TraceSampleRate float64

	// Infrastructure
	MaxStackSize       int
	SharedTools        *tools.ToolRegistry
	Observability      ObservabilityConfig
	Store              KitStore
	Probe              ProbeConfig
	RetryPolicies      map[string]RetryPolicy
	Logger             *slog.Logger
	LogHandler         func(LogEntry)
	ErrorHandler       func(error, ErrorContext)
	Transport          any // *transport.Transport — uses any to avoid import cycle
	DeferRouterStart   bool
	MaxConcurrency     int
	ProviderKeyMapping map[string]string
	Modules            []any // engine.Module — uses any to avoid import cycle
}

// ObservabilityConfig configures the tracing/observability system.
type ObservabilityConfig struct {
	Enabled     *bool
	Strategy    string
	ServiceName string
}

// RetryPolicy configures retry behavior for failed bus handlers.
type RetryPolicy struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	DeadLetterTopic string
}

// ScheduleConfig configures a new schedule via Kernel.Schedule().
type ScheduleConfig struct {
	ID         string
	Expression string
	Topic      string
	Payload    json.RawMessage
	Source     string
}

// NodeConfig configures a transport-connected runtime node.
type NodeConfig struct {
	Kernel    KernelConfig
	Messaging MessagingConfig
	NodeID    string
	Namespace string
	Plugins   []PluginConfig
}

// MessagingConfig configures the transport-backed runtime host.
type MessagingConfig struct {
	Transport    string
	NATSURL      string
	NATSName     string
	AMQPURL      string
	RedisURL     string
	NATSStoreDir string // JetStream store for embedded NATS. Empty = ephemeral.
}

