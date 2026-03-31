package registry

// Vector store config structs — one per backend, matching Mastra constructor signatures.

// LibSQLVectorConfig configures a LibSQL/Turso vector store.
type LibSQLVectorConfig struct {
	URL       string
	AuthToken string
	SyncURL   string
}

// PgVectorConfig configures a PostgreSQL + pgvector store.
type PgVectorConfig struct {
	ConnectionString string
	Host             string
	Port             int
	Database         string
	User             string
	Password         string
	SchemaName       string
	VectorType       string // "vector" (4 bytes/dim) | "halfvec" (2 bytes/dim)
}

// MongoDBVectorConfig configures a MongoDB Atlas vector store.
type MongoDBVectorConfig struct {
	URI                string
	DBName             string
	EmbeddingFieldPath string // default: "embedding"
}

// PineconeVectorConfig configures a Pinecone vector store.
type PineconeVectorConfig struct {
	APIKey string
	Cloud  string // "aws" | "gcp" | "azure", default: "aws"
	Region string // default: "us-east-1"
}

// QdrantVectorConfig configures a Qdrant vector store.
type QdrantVectorConfig struct {
	URL    string
	APIKey string
}

// ChromaVectorConfig configures a Chroma vector store.
type ChromaVectorConfig struct {
	Host string
	Port int
}

// UpstashVectorConfig configures an Upstash vector store.
type UpstashVectorConfig struct {
	URL   string
	Token string
}

// AstraVectorConfig configures a DataStax Astra vector store.
type AstraVectorConfig struct {
	Token    string
	Endpoint string
	Keyspace string
}

// ElasticsearchVectorConfig configures an Elasticsearch vector store.
type ElasticsearchVectorConfig struct {
	URL      string
	Username string
	Password string
	APIKey   string
}

// OpenSearchVectorConfig configures an OpenSearch vector store.
type OpenSearchVectorConfig struct {
	URL      string
	Username string
	Password string
}

// TurbopufferVectorConfig configures a Turbopuffer vector store.
type TurbopufferVectorConfig struct {
	APIKey string
}

// CloudflareVectorConfig configures a Cloudflare Vectorize store.
type CloudflareVectorConfig struct {
	AccountID string
	APIToken  string
}

// DuckDBVectorConfig configures a DuckDB vector store.
type DuckDBVectorConfig struct {
	Path string
}

// LanceVectorConfig configures a LanceDB vector store.
type LanceVectorConfig struct {
	Path string
}

// ConvexVectorConfig configures a Convex vector store.
type ConvexVectorConfig struct {
	URL      string
	AdminKey string
}

// CouchbaseVectorConfig configures a Couchbase vector store.
type CouchbaseVectorConfig struct {
	ConnectionString string
	Username         string
	Password         string
	BucketName       string
}

// S3VectorsVectorConfig configures an S3 Vectors store.
type S3VectorsVectorConfig struct {
	VectorBucketName string
	Region           string
	AccessKey        string
	SecretKey        string
}
