# Go Config — API Reference

> `import "github.com/brainlet/brainkit/kit"`
> `import "github.com/brainlet/brainkit/kit/registry"`

## KernelConfig

```go
type KernelConfig struct {
    Name             string
    Namespace        string                                    // default: "user"
    CallerID         string                                    // default: Namespace
    EnvVars          map[string]string                         // injected into JS process.env
    MaxStackSize     int                                       // QuickJS stack size (bytes)
    SharedTools      *toolreg.ToolRegistry                     // shared across pool instances
    MCPServers       map[string]mcppkg.ServerConfig            // MCP server connections
    Observability    ObservabilityConfig
    Store            KitStore                                  // WASM module/shard persistence
    FSRoot           string                                    // sandboxed fs root

    Storages         map[string]StorageConfig                  // SQLite, Postgres, etc.
    Vectors          map[string]VectorConfig                   // PgVector, LibSQL, etc.

    Probe            registry.ProbeConfig
    Transport        *messaging.Transport                      // injected by Node, nil for standalone
    DeferRouterStart bool                                      // set by Node
    LogHandler       func(LogEntry)                            // nil = log.Printf
}
```

## NodeConfig

```go
type NodeConfig struct {
    Kernel    KernelConfig
    Messaging MessagingConfig
    NodeID    string              // auto-generated UUID if empty
    Namespace string              // overrides Kernel.Namespace if set
    Plugins   []PluginConfig
}
```

## MessagingConfig

```go
type MessagingConfig struct {
    Transport   string // "memory", "nats", "amqp", "redis", "sql-postgres", "sql-sqlite"
    NATSURL     string
    NATSName    string // durable consumer prefix
    AMQPURL     string
    RedisURL    string
    PostgresURL string
    SQLitePath  string
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
    Source  string    // "my-service.ts", "wasm:my-shard", "kernel"
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
var ErrNoWorkspace error       // FSRoot not configured
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
    Base         NodeConfig
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
    SaveModule(name string, binary []byte, info WASMModuleInfo) error
    LoadModules() (map[string]*WASMModule, error)
    DeleteModule(name string) error
    SaveShard(name string, desc ShardDescriptor) error
    LoadShards() (map[string]ShardDescriptor, error)
    DeleteShard(name string) error
    SaveState(shardName, key string, state map[string]string) error
    LoadState(shardName, key string) (map[string]string, error)
    DeleteState(shardName string) error
    Close() error
}

func NewSQLiteStore(path string) (*SQLiteStore, error)
```
