package brainkit

import "github.com/brainlet/brainkit/internal/types"

// StorageConfig configures a storage backend in the resource pool.
type StorageConfig = types.StorageConfig

// VectorConfig configures a vector store backend in the resource pool.
type VectorConfig = types.VectorConfig

// Storage convenience constructors. SQLiteStorage and SQLiteVector require
// importing github.com/brainlet/brainkit/storagebridges/sqlite or the aggregate
// github.com/brainlet/brainkit/storagebridges in the binary that creates the
// Kit.
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
