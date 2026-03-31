package registry

// Storage config structs — one per backend, matching Mastra storage adapter constructors.

// InMemoryStorageConfig configures an in-memory storage (no persistence).
type InMemoryStorageConfig struct{}

// LibSQLStorageConfig configures a LibSQL/Turso storage adapter.
type LibSQLStorageConfig struct {
	URL              string
	AuthToken        string
	MaxRetries       int // default: 5
	InitialBackoffMs int // default: 100
}

// PostgresStorageConfig configures a PostgreSQL storage adapter.
type PostgresStorageConfig struct {
	ConnectionString string
	Host             string
	Port             int
	Database         string
	User             string
	Password         string
	SchemaName       string
}

// MongoDBStorageConfig configures a MongoDB storage adapter.
type MongoDBStorageConfig struct {
	URI    string
	DBName string
}

// UpstashStorageConfig configures an Upstash storage adapter.
type UpstashStorageConfig struct {
	URL   string
	Token string
}

// CloudflareD1StorageConfig configures a Cloudflare D1 storage adapter.
type CloudflareD1StorageConfig struct {
	AccountID  string
	APIToken   string
	DatabaseID string
}

// CloudflareKVStorageConfig configures a Cloudflare Workers KV storage adapter.
type CloudflareKVStorageConfig struct {
	AccountID   string
	APIToken    string
	NamespaceID string
}

// ClickHouseStorageConfig configures a ClickHouse storage adapter.
type ClickHouseStorageConfig struct {
	URL      string
	Database string
	User     string
	Password string
}

// ConvexStorageConfig configures a Convex storage adapter.
type ConvexStorageConfig struct {
	URL      string
	AdminKey string
}

// CouchbaseStorageConfig configures a Couchbase storage adapter.
type CouchbaseStorageConfig struct {
	ConnectionString string
	Username         string
	Password         string
	BucketName       string
}

// DynamoDBStorageConfig configures an AWS DynamoDB storage adapter.
type DynamoDBStorageConfig struct {
	Region    string
	TableName string
	AccessKey string
	SecretKey string
}

// LanceStorageConfig configures a LanceDB storage adapter.
type LanceStorageConfig struct {
	Path string
}

// MSSQLStorageConfig configures a Microsoft SQL Server storage adapter.
type MSSQLStorageConfig struct {
	ConnectionString string
}

// DuckDBStorageConfig configures a DuckDB storage adapter.
type DuckDBStorageConfig struct {
	Path string
}
