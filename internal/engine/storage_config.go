package engine

import (
	provreg "github.com/brainlet/brainkit/internal/providers"
	"github.com/brainlet/brainkit/internal/types"
)

// storageRegistryType maps user-facing types.StorageConfig.Type to the internal
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

// storageToRegistration converts a types.StorageConfig to the internal registry format.
func storageToRegistration(cfg types.StorageConfig, bridgeURL string) provreg.StorageRegistration {
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

// vectorRegistryType maps user-facing types.VectorConfig.Type to the internal type.
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

// vectorToRegistration converts a types.VectorConfig to the internal registry format.
func vectorToRegistration(cfg types.VectorConfig, bridgeURL string) provreg.VectorStoreRegistration {
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
