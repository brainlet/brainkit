package brainkit

import (
	"github.com/brainlet/brainkit/internal/engine"
	"github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/types"
)

// ── Core Types ───────────────────────────────────────────────────────────────

type Kernel = engine.Kernel
type Node = engine.Node

// ── Config Types ─────────────────────────────────────────────────────────────

type KernelConfig = types.KernelConfig
type NodeConfig = types.NodeConfig
type MessagingConfig = types.MessagingConfig
type StorageConfig = types.StorageConfig
type VectorConfig = types.VectorConfig
type PluginConfig = types.PluginConfig
type RetryPolicy = types.RetryPolicy
type ScheduleConfig = types.ScheduleConfig
type ObservabilityConfig = types.ObservabilityConfig
type RegistryConfig = types.RegistryConfig
type MCPServerConfig = types.MCPServerConfig

// ── Storage Convenience Constructors ─────────────────────────────────────────

var (
	SQLiteStorage      = types.SQLiteStorage
	PostgresStorage    = types.PostgresStorage
	MongoDBStorage     = types.MongoDBStorage
	UpstashStorage     = types.UpstashStorage
	InMemoryStorage    = types.InMemoryStorage
	SQLiteVector       = types.SQLiteVector
	PgVectorStore      = types.PgVectorStore
	MongoDBVectorStore = types.MongoDBVectorStore
)

var DefaultRegistry = types.DefaultRegistry

// ── Store Types ──────────────────────────────────────────────────────────────

type KitStore = types.KitStore
type SQLiteStore = types.SQLiteStore
type PersistedDeployment = types.PersistedDeployment
type PersistedSchedule = types.PersistedSchedule

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	return types.NewSQLiteStore(path)
}

// ── RBAC Types ───────────────────────────────────────────────────────────────

type Role = types.Role
type BusPermissions = types.BusPermissions
type TopicFilter = types.TopicFilter
type CommandPermissions = types.CommandPermissions
type RegistrationPermissions = types.RegistrationPermissions

var (
	RoleAdmin    = types.RoleAdmin
	RoleService  = types.RoleService
	RoleGateway  = types.RoleGateway
	RoleObserver = types.RoleObserver
)

// ── Secrets ──────────────────────────────────────────────────────────────────

type SecretStore = types.SecretStore
type SecretMeta = types.SecretMeta

// ── Tracing ──────────────────────────────────────────────────────────────────

type TraceStore = types.TraceStore
type Span = types.Span

// ── Provider Types ───────────────────────────────────────────────────────────

type AIProviderRegistration = types.AIProviderRegistration
type AIProviderType = types.AIProviderType

const (
	AIProviderOpenAI      = types.AIProviderOpenAI
	AIProviderAnthropic   = types.AIProviderAnthropic
	AIProviderGoogle      = types.AIProviderGoogle
	AIProviderMistral     = types.AIProviderMistral
	AIProviderCohere      = types.AIProviderCohere
	AIProviderGroq        = types.AIProviderGroq
	AIProviderPerplexity  = types.AIProviderPerplexity
	AIProviderDeepSeek    = types.AIProviderDeepSeek
	AIProviderFireworks   = types.AIProviderFireworks
	AIProviderTogetherAI  = types.AIProviderTogetherAI
	AIProviderXAI         = types.AIProviderXAI
	AIProviderAzure       = types.AIProviderAzure
	AIProviderBedrock     = types.AIProviderBedrock
	AIProviderVertex      = types.AIProviderVertex
	AIProviderHuggingFace = types.AIProviderHuggingFace
	AIProviderCerebras    = types.AIProviderCerebras
)

// Provider config structs
type OpenAIProviderConfig = types.OpenAIProviderConfig
type AnthropicProviderConfig = types.AnthropicProviderConfig
type GoogleProviderConfig = types.GoogleProviderConfig
type MistralProviderConfig = types.MistralProviderConfig
type CohereProviderConfig = types.CohereProviderConfig
type GroqProviderConfig = types.GroqProviderConfig
type PerplexityProviderConfig = types.PerplexityProviderConfig
type DeepSeekProviderConfig = types.DeepSeekProviderConfig
type FireworksProviderConfig = types.FireworksProviderConfig
type TogetherAIProviderConfig = types.TogetherAIProviderConfig
type XAIProviderConfig = types.XAIProviderConfig
type CerebrasProviderConfig = types.CerebrasProviderConfig

// ── Common Types ─────────────────────────────────────────────────────────────

type Result = types.Result
type ResourceInfo = types.ResourceInfo
type LogEntry = types.LogEntry
type ErrorContext = types.ErrorContext
type HealthStatus = types.HealthStatus
type HealthCheck = types.HealthCheck
type KernelMetrics = types.KernelMetrics
type RunningPlugin = types.RunningPlugin
type RunningPluginRecord = types.RunningPluginRecord
type InstalledPlugin = types.InstalledPlugin
type DeployOption = types.DeployOption

var (
	WithRole        = types.WithRole
	WithPackageName = types.WithPackageName
	WithRestoring   = types.WithRestoring
	InvokeErrorHandler = types.InvokeErrorHandler
)

// ── Client ───────────────────────────────────────────────────────────────────

type BusClient = types.BusClient
type StreamEvent = types.StreamEvent

func NewClient(baseURL string) *BusClient {
	return types.NewClient(baseURL)
}

// ── Errors ───────────────────────────────────────────────────────────────────

var (
	ErrMCPNotConfigured = types.ErrMCPNotConfigured
	ErrCommandTopic     = types.ErrCommandTopic
)

// ── Scaling ──────────────────────────────────────────────────────────────────

type InstanceManager = engine.InstanceManager
type PoolConfig = engine.PoolConfig
type PoolInfo = types.PoolInfo
type ScalingStrategy = types.ScalingStrategy
type ScalingDecision = types.ScalingDecision
type ThresholdStrategy = engine.ThresholdStrategy

var (
	NewInstanceManager   = engine.NewInstanceManager
	NewStaticStrategy    = engine.NewStaticStrategy
	NewThresholdStrategy = engine.NewThresholdStrategy
)

// ── Constructors ─────────────────────────────────────────────────────────────

func NewKernel(cfg KernelConfig) (*Kernel, error) {
	return engine.NewKernel(cfg)
}

func NewNode(cfg NodeConfig) (*Node, error) {
	return engine.NewNode(cfg)
}

// RegisterTool is a convenience for registering typed Go tools on a Kernel.
func RegisterTool[T any](k *Kernel, name string, tool tools.TypedTool[T]) error {
	return engine.RegisterTool(k, name, tool)
}

// ── Embedded Type Definitions (for CLI scaffolding) ──────────────────────────

var (
	KitDTS      = engine.KitDTS
	AiDTS       = engine.AiDTS
	AgentDTS    = engine.AgentDTS
	BrainkitDTS = engine.BrainkitDTS
	GlobalsDTS  = engine.GlobalsDTS
)
