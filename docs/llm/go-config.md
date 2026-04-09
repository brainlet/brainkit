# Go Config — API Reference

> `import "github.com/brainlet/brainkit"`
> `import "github.com/brainlet/brainkit/sdk"`

## Config

```go
type Config struct {
    Namespace        string                                    // default: "user"
    ClusterID        string                                    // cluster identity for horizontal scaling
    CallerID         string                                    // default: Namespace
    RuntimeID        string                                    // auto-generated per process
    AIProviders      map[string]registry.AIProviderRegistration // explicit provider configs (auto-detected from env if empty)
    EnvVars          map[string]string                         // injected into JS process.env
    Storages         map[string]StorageConfig                  // SQLite, Postgres, etc.
    Vectors          map[string]VectorConfig                   // PgVector, LibSQL, etc.
    FSRoot           string                                    // sandboxed fs root
    SecretStore      secrets.SecretStore                       // pluggable secret backend
    SecretKey        string                                    // master key for EncryptedKVStore
    Roles            map[string]rbac.Role                      // RBAC role definitions
    DefaultRole      string                                    // default role for deployments
    TraceStore       tracing.TraceStore                        // nil = no tracing
    TraceSampleRate  float64                                   // 0.0-1.0, default 1.0
    MaxStackSize     int                                       // QuickJS stack size (bytes)
    SharedTools      *toolreg.ToolRegistry                     // shared across pool instances
    MCPServers       map[string]mcppkg.ServerConfig            // MCP server connections
    Observability    ObservabilityConfig
    Store            KitStore                                  // deployment/schedule persistence
    Probe            registry.ProbeConfig
    RetryPolicies    map[string]RetryPolicy                    // topic glob → retry config
    LogHandler       func(LogEntry)                            // nil = log.Printf
    ErrorHandler     func(error, ErrorContext)                 // nil = log.Printf
    MaxConcurrency   int                                       // concurrent bus handlers (0 = unlimited)
    BusRateLimits    map[string]float64                        // role → requests/sec
    PluginRegistries []RegistryConfig                          // plugin registry sources
    PluginDir        string                                    // local plugin cache
    // Transport fields (flattened from former MessagingConfig)
    Transport   string // "memory", "embedded" (default), "nats", "amqp", "redis"
    NATSURL     string
    NATSName    string // durable consumer prefix
    AMQPURL     string
    RedisURL    string
    // Plugin configuration
    Plugins     []PluginConfig
}
```

## PluginConfig

```go
type PluginConfig struct {
    Name            string
    Binary          string            // path to plugin binary
    Args            []string
    Env             map[string]string
    Config          json.RawMessage   // passed via BRAINKIT_PLUGIN_CONFIG env
    AutoRestart     bool              // restart on crash
    MaxRestarts     int               // default: 5
    StartTimeout    time.Duration     // default: 10s
    ShutdownTimeout time.Duration     // default: 5s
}
```

## StorageConfig

```go
// StorageConfig is created via constructor functions:
func SQLiteStorage(path string) StorageConfig     // local SQLite (libsql HTTP bridge)
func PostgresStorage(url string) StorageConfig     // Postgres connection string
func MongoDBStorage(uri, dbName string) StorageConfig
func InMemoryStorage() StorageConfig
func UpstashStorage(url, token string) StorageConfig
func LibSQLStorage(url, authToken string) StorageConfig // remote libsql/Turso
```

## VectorConfig

```go
// VectorConfig is created via constructor functions:
func PgVectorStore(url string) VectorConfig
func LibSQLVectorStore(url, authToken string) VectorConfig
func MongoDBVectorStore(uri, dbName string) VectorConfig
func PineconeVectorStore(apiKey, environment, index string) VectorConfig
func QdrantVectorStore(url, apiKey, collection string) VectorConfig
```

## ObservabilityConfig

```go
type ObservabilityConfig struct {
    Enabled     *bool   // default: true
    Strategy    string  // "realtime" or "batch"
    ServiceName string  // default: "brainkit"
}
```

## LogEntry

```go
type LogEntry struct {
    Source  string    // "my-service.ts", "kernel"
    Level  string    // "log", "warn", "error", "debug", "info"
    Message string
    Time   time.Time
}
```

## ProbeConfig

```go
// kit/registry/probe.go
type ProbeConfig struct {
    PeriodicInterval time.Duration // 0 = no periodic probing
    Timeout          time.Duration // per-probe timeout
}
```

## AI Providers

AI providers are auto-detected from environment variables (`os.Getenv`). Explicit registration is no longer required for standard providers — if `OPENAI_API_KEY` is set in the environment, the OpenAI provider is available automatically.

Supported providers (15): openai, anthropic, google, mistral, cohere, groq, perplexity, deepseek, fireworks, togetherai, xai, cerebras, azure, huggingface, bedrock.

## MCP Server Config

```go
// internal/mcp/client.go
type ServerConfig struct {
    Command string            // stdio transport: binary path
    Args    []string          // stdio transport: arguments
    Env     map[string]string // stdio transport: environment
    URL     string            // HTTP transport: endpoint URL
}
```

## Kit Sentinels

```go
// kit/errors.go
var ErrMCPNotConfigured error  // no MCP servers registered
var ErrCommandTopic error      // event emitted on command topic
```

## Scaling

```go
type InstanceManager struct{}
func NewInstanceManager() *InstanceManager
func (im *InstanceManager) SpawnPool(name string, cfg PoolConfig) error
func (im *InstanceManager) Scale(name string, delta int) error
func (im *InstanceManager) KillPool(name string) error
func (im *InstanceManager) PoolInfo(name string) (PoolInfo, error)
func (im *InstanceManager) Pools() []string
func (im *InstanceManager) EvaluateAndScale()

type PoolConfig struct {
    Base         brainkit.Config
    InitialCount int
    Min, Max     int
    Strategy     ScalingStrategy
}

type PoolInfo struct { Name string; Current, Min, Max, Pending int }

type ScalingStrategy interface {
    Evaluate(metrics messaging.MetricsSnapshot, pool PoolInfo) ScalingDecision
}
type ScalingDecision struct { Action string; Delta int; Reason string }

func NewStaticStrategy(target int) *StaticStrategy
func NewThresholdStrategy(scaleUp, scaleDown int) *ThresholdStrategy
```

## KitStore

```go
type KitStore interface {
    SaveDeployment(d PersistedDeployment) error
    LoadDeployments() ([]PersistedDeployment, error)
    DeleteDeployment(source string) error
    SaveSchedule(s PersistedSchedule) error
    LoadSchedules() ([]PersistedSchedule, error)
    DeleteSchedule(id string) error
    SaveInstalledPlugin(p InstalledPlugin) error
    LoadInstalledPlugins() ([]InstalledPlugin, error)
    DeleteInstalledPlugin(name string) error
    SaveRunningPlugin(p RunningPluginRecord) error
    LoadRunningPlugins() ([]RunningPluginRecord, error)
    DeleteRunningPlugin(name string) error
    SavePluginState(pluginID, key, value string) error
    LoadPluginState(pluginID, key string) (string, error)
    DeletePluginState(pluginID string) error
    Close() error
}

func NewSQLiteStore(path string) (*SQLiteStore, error)
```
