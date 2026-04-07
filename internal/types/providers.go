package types

import "time"

// ── AI Provider Types ────────────────────────────────────────────────────────

// AIProviderType identifies an AI SDK provider.
type AIProviderType string

const (
	AIProviderOpenAI      AIProviderType = "openai"
	AIProviderAnthropic   AIProviderType = "anthropic"
	AIProviderGoogle      AIProviderType = "google"
	AIProviderMistral     AIProviderType = "mistral"
	AIProviderCohere      AIProviderType = "cohere"
	AIProviderGroq        AIProviderType = "groq"
	AIProviderPerplexity  AIProviderType = "perplexity"
	AIProviderDeepSeek    AIProviderType = "deepseek"
	AIProviderFireworks   AIProviderType = "fireworks"
	AIProviderTogetherAI  AIProviderType = "togetherai"
	AIProviderXAI         AIProviderType = "xai"
	AIProviderAzure       AIProviderType = "azure"
	AIProviderBedrock     AIProviderType = "bedrock"
	AIProviderVertex      AIProviderType = "vertex"
	AIProviderHuggingFace AIProviderType = "huggingface"
	AIProviderCerebras    AIProviderType = "cerebras"
)

// AIProviderRegistration wraps a typed config for registration.
type AIProviderRegistration struct {
	Type   AIProviderType
	Config any
}

// AIProviderCapabilities describes what model types a provider supports.
type AIProviderCapabilities struct {
	Chat          bool `json:"chat"`
	Embedding     bool `json:"embedding"`
	Image         bool `json:"image"`
	Transcription bool `json:"transcription"`
	Speech        bool `json:"speech"`
	Reranking     bool `json:"reranking"`
}

// KnownAICapabilities returns the known capability set for a provider type.
func KnownAICapabilities(typ AIProviderType) AIProviderCapabilities {
	switch typ {
	case AIProviderOpenAI:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true, Transcription: true, Speech: true}
	case AIProviderAnthropic:
		return AIProviderCapabilities{Chat: true}
	case AIProviderGoogle:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true}
	case AIProviderMistral:
		return AIProviderCapabilities{Chat: true, Embedding: true}
	case AIProviderCohere:
		return AIProviderCapabilities{Chat: true, Embedding: true, Reranking: true}
	case AIProviderGroq:
		return AIProviderCapabilities{Chat: true, Embedding: true}
	case AIProviderPerplexity:
		return AIProviderCapabilities{Chat: true, Embedding: true}
	case AIProviderDeepSeek:
		return AIProviderCapabilities{Chat: true}
	case AIProviderFireworks:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true}
	case AIProviderTogetherAI:
		return AIProviderCapabilities{Chat: true, Embedding: true, Reranking: true}
	case AIProviderXAI:
		return AIProviderCapabilities{Chat: true}
	case AIProviderAzure:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true, Transcription: true, Speech: true}
	case AIProviderBedrock:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true}
	case AIProviderVertex:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true}
	case AIProviderHuggingFace:
		return AIProviderCapabilities{Chat: true, Embedding: true, Image: true}
	case AIProviderCerebras:
		return AIProviderCapabilities{Chat: true}
	default:
		return AIProviderCapabilities{Chat: true}
	}
}

// ProviderInfo describes a registered AI provider.
type ProviderInfo struct {
	Name         string                `json:"name"`
	Type         AIProviderType        `json:"type"`
	Capabilities AIProviderCapabilities `json:"capabilities"`
	Healthy      bool                  `json:"healthy"`
	LastProbed   time.Time             `json:"lastProbed"`
	LastError    string                `json:"lastError"`
	Latency      time.Duration         `json:"latency"`
}

// ProbeConfig configures probing behavior.
type ProbeConfig struct {
	CacheTTL         time.Duration
	ProbeOnRegister  bool
	ProbeTimeout     time.Duration
	PeriodicInterval time.Duration
}

// ProbeResult is returned by explicit Probe* calls.
type ProbeResult struct {
	Available    bool          `json:"available"`
	Capabilities any           `json:"capabilities"`
	Latency      time.Duration `json:"latency"`
	Error        string        `json:"error,omitempty"`
}

// ── AI Provider Config Structs ───────────────────────────────────────────────

type OpenAIProviderConfig struct {
	APIKey       string
	BaseURL      string
	Organization string
	Project      string
	Headers      map[string]string
}

type AnthropicProviderConfig struct {
	APIKey    string
	AuthToken string
	BaseURL   string
	Headers   map[string]string
}

type GoogleProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

type MistralProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

type CohereProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

type GroqProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

type PerplexityProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

type DeepSeekProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

type FireworksProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

type TogetherAIProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

type XAIProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

type AzureProviderConfig struct {
	APIKey       string
	ResourceName string
	BaseURL      string
	Headers      map[string]string
}

type BedrockProviderConfig struct {
	Region    string
	AccessKey string
	SecretKey string
	Headers   map[string]string
}

type VertexProviderConfig struct {
	Project  string
	Location string
	APIKey   string
	Headers  map[string]string
}

type HuggingFaceProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

type CerebrasProviderConfig struct {
	APIKey  string
	BaseURL string
	Headers map[string]string
}

// ── Vector Store Types ───────────────────────────────────────────────────────

type VectorStoreType string

const (
	VectorStoreLibSQL        VectorStoreType = "libsql"
	VectorStorePg            VectorStoreType = "pgvector"
	VectorStoreMongoDB       VectorStoreType = "mongodb"
	VectorStorePinecone      VectorStoreType = "pinecone"
	VectorStoreQdrant        VectorStoreType = "qdrant"
	VectorStoreChroma        VectorStoreType = "chroma"
	VectorStoreUpstash       VectorStoreType = "upstash"
	VectorStoreAstra         VectorStoreType = "astra"
	VectorStoreElasticsearch VectorStoreType = "elasticsearch"
	VectorStoreOpenSearch    VectorStoreType = "opensearch"
	VectorStoreTurbopuffer   VectorStoreType = "turbopuffer"
	VectorStoreCloudflare    VectorStoreType = "cloudflare"
	VectorStoreDuckDB        VectorStoreType = "duckdb"
	VectorStoreLance         VectorStoreType = "lance"
	VectorStoreConvex        VectorStoreType = "convex"
	VectorStoreCouchbase     VectorStoreType = "couchbase"
	VectorStoreS3Vectors     VectorStoreType = "s3vectors"
)

type VectorStoreRegistration struct {
	Type   VectorStoreType
	Config any
}

type VectorStoreCapabilities struct {
	CreateIndex   bool `json:"createIndex"`
	DeleteIndex   bool `json:"deleteIndex"`
	ListIndexes   bool `json:"listIndexes"`
	Upsert        bool `json:"upsert"`
	Query         bool `json:"query"`
	UpdateVector  bool `json:"updateVector"`
	DeleteVector  bool `json:"deleteVector"`
	NamedVectors  bool `json:"namedVectors"`
	SparseVectors bool `json:"sparseVectors"`
}

func DefaultVectorCapabilities() VectorStoreCapabilities {
	return VectorStoreCapabilities{
		CreateIndex: true, DeleteIndex: true, ListIndexes: true,
		Upsert: true, Query: true, UpdateVector: true, DeleteVector: true,
	}
}

type VectorStoreInfo struct {
	Name         string                  `json:"name"`
	Type         VectorStoreType         `json:"type"`
	Capabilities VectorStoreCapabilities `json:"capabilities"`
	Healthy      bool                    `json:"healthy"`
	LastProbed   time.Time               `json:"lastProbed"`
	LastError    string                  `json:"lastError"`
	Latency      time.Duration           `json:"latency"`
}

// ── Vector Store Config Structs ──────────────────────────────────────────────

type LibSQLVectorConfig struct {
	URL       string
	AuthToken string
	SyncURL   string
}

type PgVectorConfig struct {
	ConnectionString string
	Host             string
	Port             int
	Database         string
	User             string
	Password         string
	SchemaName       string
	VectorType       string
}

type MongoDBVectorConfig struct {
	URI                string
	DBName             string
	EmbeddingFieldPath string
}

type PineconeVectorConfig struct {
	APIKey string
	Cloud  string
	Region string
}

type QdrantVectorConfig struct {
	URL    string
	APIKey string
}

type ChromaVectorConfig struct {
	Host string
	Port int
}

type UpstashVectorConfig struct {
	URL   string
	Token string
}

type AstraVectorConfig struct {
	Token    string
	Endpoint string
	Keyspace string
}

type ElasticsearchVectorConfig struct {
	URL      string
	Username string
	Password string
	APIKey   string
}

type OpenSearchVectorConfig struct {
	URL      string
	Username string
	Password string
}

type TurbopufferVectorConfig struct {
	APIKey string
}

type CloudflareVectorConfig struct {
	AccountID string
	APIToken  string
}

type DuckDBVectorConfig struct {
	Path string
}

type LanceVectorConfig struct {
	Path string
}

type ConvexVectorConfig struct {
	URL      string
	AdminKey string
}

type CouchbaseVectorConfig struct {
	ConnectionString string
	Username         string
	Password         string
	BucketName       string
}

type S3VectorsVectorConfig struct {
	VectorBucketName string
	Region           string
	AccessKey        string
	SecretKey        string
}

// ── Storage Types ────────────────────────────────────────────────────────────

type StorageType string

const (
	StorageInMemory     StorageType = "memory"
	StorageLibSQL       StorageType = "libsql"
	StoragePostgres     StorageType = "postgres"
	StorageMongoDB      StorageType = "mongodb"
	StorageUpstash      StorageType = "upstash"
	StorageCloudflareD1 StorageType = "cloudflare-d1"
	StorageCloudflareKV StorageType = "cloudflare-kv"
	StorageClickHouse   StorageType = "clickhouse"
	StorageConvex       StorageType = "convex"
	StorageCouchbase    StorageType = "couchbase"
	StorageDynamoDB     StorageType = "dynamodb"
	StorageLance        StorageType = "lance"
	StorageMSSQL        StorageType = "mssql"
	StorageDuckDB       StorageType = "duckdb"
)

type StorageRegistration struct {
	Type   StorageType
	Config any
}

type StorageCapabilities struct {
	Memory        bool `json:"memory"`
	Workflows     bool `json:"workflows"`
	Scores        bool `json:"scores"`
	Observability bool `json:"observability"`
	Agents        bool `json:"agents"`
	Datasets      bool `json:"datasets"`
	Blobs         bool `json:"blobs"`
}

func DefaultStorageCapabilities() StorageCapabilities {
	return StorageCapabilities{
		Memory: true, Workflows: true, Scores: true,
		Observability: true, Agents: true, Datasets: true,
	}
}

type StorageInfo struct {
	Name         string              `json:"name"`
	Type         StorageType         `json:"type"`
	Capabilities StorageCapabilities `json:"capabilities"`
	Healthy      bool                `json:"healthy"`
	LastProbed   time.Time           `json:"lastProbed"`
	LastError    string              `json:"lastError"`
	Latency      time.Duration       `json:"latency"`
}

// ── Storage Config Structs ───────────────────────────────────────────────────

type InMemoryStorageConfig struct{}

type LibSQLStorageConfig struct {
	URL              string
	AuthToken        string
	MaxRetries       int
	InitialBackoffMs int
}

type PostgresStorageConfig struct {
	ConnectionString string
	Host             string
	Port             int
	Database         string
	User             string
	Password         string
	SchemaName       string
}

type MongoDBStorageConfig struct {
	URI    string
	DBName string
}

type UpstashStorageConfig struct {
	URL   string
	Token string
}

type CloudflareD1StorageConfig struct {
	AccountID  string
	APIToken   string
	DatabaseID string
}

type CloudflareKVStorageConfig struct {
	AccountID   string
	APIToken    string
	NamespaceID string
}

type ClickHouseStorageConfig struct {
	URL      string
	Database string
	User     string
	Password string
}

type ConvexStorageConfig struct {
	URL      string
	AdminKey string
}

type CouchbaseStorageConfig struct {
	ConnectionString string
	Username         string
	Password         string
	BucketName       string
}

type DynamoDBStorageConfig struct {
	Region    string
	TableName string
	AccessKey string
	SecretKey string
}

type LanceStorageConfig struct {
	Path string
}

type MSSQLStorageConfig struct {
	ConnectionString string
}

type DuckDBStorageConfig struct {
	Path string
}
