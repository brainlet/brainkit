// api.go — Type aliases from internal/types for use within engine.
// Engine uses these aliases so that types.KernelConfig and engine.KernelConfig
// are the same type, enabling the root facade to pass types through.
package engine

import "github.com/brainlet/brainkit/internal/types"

// Config types
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
type DiscoveryConfig = types.DiscoveryConfig
type PeerConfig = types.PeerConfig

// Store types
type KitStore = types.KitStore
type SQLiteStore = types.SQLiteStore
type PersistedDeployment = types.PersistedDeployment
type PersistedSchedule = types.PersistedSchedule
type InstalledPlugin = types.InstalledPlugin
type RunningPluginRecord = types.RunningPluginRecord

// RBAC types
type Role = types.Role
type BusPermissions = types.BusPermissions
type TopicFilter = types.TopicFilter
type CommandPermissions = types.CommandPermissions
type RegistrationPermissions = types.RegistrationPermissions
type RoleAssignment = types.RoleAssignment

// Secrets
type SecretStore = types.SecretStore
type SecretMeta = types.SecretMeta

// Tracing
type TraceStore = types.TraceStore
type Span = types.Span
type TraceContext = types.TraceContext
type TraceQuery = types.TraceQuery
type TraceSummary = types.TraceSummary

// Provider types
type AIProviderRegistration = types.AIProviderRegistration
type AIProviderType = types.AIProviderType
type StorageRegistration = types.StorageRegistration
type StorageType = types.StorageType
type VectorStoreRegistration = types.VectorStoreRegistration
type VectorStoreType = types.VectorStoreType
type ProbeConfig = types.ProbeConfig

// Common types
type Result = types.Result
type ResourceInfo = types.ResourceInfo
type LogEntry = types.LogEntry
type ErrorContext = types.ErrorContext
type HealthStatus = types.HealthStatus
type HealthCheck = types.HealthCheck
type KernelMetrics = types.KernelMetrics
type RunningPlugin = types.RunningPlugin
type DeployOption = types.DeployOption
type DeployConfig = types.DeployConfig
type BusClient = types.BusClient
type StreamEvent = types.StreamEvent

// Scaling
type ScalingStrategy = types.ScalingStrategy
type ScalingDecision = types.ScalingDecision
type MetricsSnapshot = types.MetricsSnapshot
type PoolInfo = types.PoolInfo

// Functions
var (
	InvokeErrorHandler = types.InvokeErrorHandler
	WithRole           = types.WithRole
	WithPackageName    = types.WithPackageName
	WithRestoring      = types.WithRestoring
	NewSQLiteStore     = types.NewSQLiteStore
	NewClient          = types.NewClient

	ErrMCPNotConfigured = types.ErrMCPNotConfigured
	ErrCommandTopic     = types.ErrCommandTopic

	DefaultRegistry = types.DefaultRegistry

	// Storage constructors
	SQLiteStorage      = types.SQLiteStorage
	PostgresStorage    = types.PostgresStorage
	MongoDBStorage     = types.MongoDBStorage
	UpstashStorage     = types.UpstashStorage
	InMemoryStorage    = types.InMemoryStorage
	SQLiteVector       = types.SQLiteVector
	PgVectorStore      = types.PgVectorStore
	MongoDBVectorStore = types.MongoDBVectorStore

	// RBAC presets
	RoleAdmin    = types.RoleAdmin
	RoleService  = types.RoleService
	RoleGateway  = types.RoleGateway
	RoleObserver = types.RoleObserver
)
