package types

// StorageConfig configures a storage backend in the resource pool.
// Deployments access it via storage("name") in .ts code.
type StorageConfig struct {
	Type             string // "sqlite", "postgres", "mongodb", "upstash", "memory"
	Path             string // sqlite only
	ConnectionString string // postgres only
	URI              string // mongodb only
	DBName           string // mongodb only
	URL              string // upstash only
	Token            string // upstash only
}

// SQLiteStorage creates a SQLite storage config.
func SQLiteStorage(path string) StorageConfig {
	return StorageConfig{Type: "sqlite", Path: path}
}

// PostgresStorage creates a PostgreSQL storage config.
func PostgresStorage(connStr string) StorageConfig {
	return StorageConfig{Type: "postgres", ConnectionString: connStr}
}

// MongoDBStorage creates a MongoDB storage config.
func MongoDBStorage(uri, dbName string) StorageConfig {
	return StorageConfig{Type: "mongodb", URI: uri, DBName: dbName}
}

// UpstashStorage creates an Upstash storage config.
func UpstashStorage(url, token string) StorageConfig {
	return StorageConfig{Type: "upstash", URL: url, Token: token}
}

// InMemoryStorage creates an in-memory storage config.
func InMemoryStorage() StorageConfig {
	return StorageConfig{Type: "memory"}
}

// VectorConfig configures a vector store backend in the resource pool.
// Deployments access it via vectorStore("name") in .ts code.
type VectorConfig struct {
	Type             string // "sqlite", "pgvector", "mongodb"
	Path             string // sqlite only
	ConnectionString string // pgvector only
	URI              string // mongodb only
	DBName           string // mongodb only
}

// SQLiteVector creates a SQLite-backed vector store config.
func SQLiteVector(path string) VectorConfig {
	return VectorConfig{Type: "sqlite", Path: path}
}

// PgVectorStore creates a PostgreSQL pgvector config.
func PgVectorStore(connStr string) VectorConfig {
	return VectorConfig{Type: "pgvector", ConnectionString: connStr}
}

// MongoDBVectorStore creates a MongoDB vector store config.
func MongoDBVectorStore(uri, dbName string) VectorConfig {
	return VectorConfig{Type: "mongodb", URI: uri, DBName: dbName}
}
