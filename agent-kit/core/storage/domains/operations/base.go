// Ported from: packages/core/src/storage/domains/operations/base.ts
package operations

import (
	"context"

	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// Operations Types
// ---------------------------------------------------------------------------

// TableName represents a Mastra storage table name.
type TableName = string

// StorageColumnType defines the column types supported by storage operations.
type StorageColumnType string

const (
	ColumnTypeText      StorageColumnType = "text"
	ColumnTypeTimestamp StorageColumnType = "timestamp"
	ColumnTypeFloat    StorageColumnType = "float"
	ColumnTypeInteger  StorageColumnType = "integer"
	ColumnTypeBigint   StorageColumnType = "bigint"
	ColumnTypeJSONB    StorageColumnType = "jsonb"
	ColumnTypeUUID     StorageColumnType = "uuid"
)

// StorageColumn describes a column in a storage table.
type StorageColumn struct {
	Type       string `json:"type"`
	PrimaryKey bool   `json:"primaryKey,omitempty"`
	Nullable   bool   `json:"nullable,omitempty"`
}

// CreateIndexOptions is the options for creating a database index.
type CreateIndexOptions = domains.CreateIndexOptions

// IndexInfo describes an existing database index.
type IndexInfo = domains.IndexInfo

// StorageIndexStats extends IndexInfo with usage statistics.
type StorageIndexStats = domains.StorageIndexStats

// ---------------------------------------------------------------------------
// StoreOperations Interface
// ---------------------------------------------------------------------------

// StoreOperations is the storage interface for low-level table operations
// (insert, load, create/alter/drop tables, index management).
// This is the raw storage layer.
type StoreOperations interface {
	// HasColumn checks if a table has a specific column.
	HasColumn(ctx context.Context, table string, column string) (bool, error)

	// CreateTable creates a new table with the given schema.
	CreateTable(ctx context.Context, tableName TableName, schema map[string]StorageColumn) error

	// ClearTable removes all records from a table.
	ClearTable(ctx context.Context, tableName TableName) error

	// DropTable removes a table entirely.
	DropTable(ctx context.Context, tableName TableName) error

	// AlterTable alters a table's schema (adds columns if not exists).
	AlterTable(ctx context.Context, tableName TableName, schema map[string]StorageColumn, ifNotExists []string) error

	// Insert inserts a single record into a table.
	Insert(ctx context.Context, tableName TableName, record map[string]any) error

	// BatchInsert inserts multiple records into a table.
	BatchInsert(ctx context.Context, tableName TableName, records []map[string]any) error

	// Load retrieves a record by matching key-value pairs.
	// Returns nil if not found.
	Load(ctx context.Context, tableName TableName, keys map[string]any) (any, error)

	// GetDatabase returns the underlying database reference (for testing/introspection).
	GetDatabase() any

	// --- Optional Index Management ---
	// Storage adapters can implement these to provide index management capabilities.
	// Adapters that do not support indexes should return an appropriate error.

	// CreateIndex creates a database index on specified columns.
	CreateIndex(ctx context.Context, options CreateIndexOptions) error

	// DropIndex drops a database index by name.
	DropIndex(ctx context.Context, indexName string) error

	// ListIndexes lists database indexes for a table (or all tables if tableName is empty).
	ListIndexes(ctx context.Context, tableName string) ([]IndexInfo, error)

	// DescribeIndex gets detailed statistics for a specific index.
	DescribeIndex(ctx context.Context, indexName string) (StorageIndexStats, error)
}
