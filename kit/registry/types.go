package registry

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

// VectorStoreType identifies a Mastra vector store backend.
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

// StorageType identifies a Mastra storage adapter.
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
