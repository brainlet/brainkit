package kit

import provreg "github.com/brainlet/brainkit/kit/registry"

// StorageConfig configures a storage backend in the resource pool.
// Deployments access it via storage("name") in .ts code.
type StorageConfig struct {
	Type             string // "sqlite", "postgres", "mongodb", "upstash", "memory"
	Path             string // sqlite only — path to SQLite database file
	ConnectionString string // postgres only
	URI              string // mongodb only
	DBName           string // mongodb only
	URL              string // upstash only
	Token            string // upstash only
}

// SQLiteStorage creates a SQLite storage config.
// Behind the scenes, brainkit starts a libsql HTTP bridge server.
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

// storageRegistryType maps user-facing StorageConfig.Type to the internal
// registry type that kit_runtime.js resolveStorage expects.
func storageRegistryType(userType string) provreg.StorageType {
	switch userType {
	case "sqlite":
		return provreg.StorageLibSQL
	case "postgres":
		return provreg.StoragePostgres
	case "mongodb":
		return provreg.StorageMongoDB
	case "upstash":
		return provreg.StorageUpstash
	case "memory":
		return provreg.StorageInMemory
	default:
		return provreg.StorageType(userType)
	}
}

// storageToRegistration converts a StorageConfig to the internal registry format.
// For sqlite, bridgeURL is the running libsql HTTP bridge URL.
func storageToRegistration(cfg StorageConfig, bridgeURL string) provreg.StorageRegistration {
	regType := storageRegistryType(cfg.Type)
	var config any
	switch cfg.Type {
	case "sqlite":
		config = provreg.LibSQLStorageConfig{URL: bridgeURL}
	case "postgres":
		config = provreg.PostgresStorageConfig{ConnectionString: cfg.ConnectionString}
	case "mongodb":
		config = provreg.MongoDBStorageConfig{URI: cfg.URI, DBName: cfg.DBName}
	case "upstash":
		config = provreg.UpstashStorageConfig{URL: cfg.URL, Token: cfg.Token}
	case "memory":
		config = nil
	}
	return provreg.StorageRegistration{Type: regType, Config: config}
}

// vectorRegistryType maps user-facing VectorConfig.Type to the internal type.
func vectorRegistryType(userType string) provreg.VectorStoreType {
	switch userType {
	case "sqlite":
		return provreg.VectorStoreLibSQL
	case "pgvector":
		return provreg.VectorStorePg
	case "mongodb":
		return provreg.VectorStoreMongoDB
	default:
		return provreg.VectorStoreType(userType)
	}
}

// vectorToRegistration converts a VectorConfig to the internal registry format.
func vectorToRegistration(cfg VectorConfig, bridgeURL string) provreg.VectorStoreRegistration {
	regType := vectorRegistryType(cfg.Type)
	var config any
	switch cfg.Type {
	case "sqlite":
		config = provreg.LibSQLVectorConfig{URL: bridgeURL}
	case "pgvector":
		config = provreg.PgVectorConfig{ConnectionString: cfg.ConnectionString}
	case "mongodb":
		config = provreg.MongoDBVectorConfig{URI: cfg.URI, DBName: cfg.DBName}
	}
	return provreg.VectorStoreRegistration{Type: regType, Config: config}
}
