// Ported from: packages/core/src/storage/domains/operations/inmemory.ts
package operations

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Compile-time interface check.
var _ StoreOperations = (*InMemoryStoreOperations)(nil)

// tableWorkflowSnapshot is the table name constant used for workflow-specific key logic.
const tableWorkflowSnapshot = "mastra_workflow_snapshot"

// InMemoryStoreOperations is an in-memory implementation of StoreOperations.
// It maintains a map of table names to maps of record IDs to records.
type InMemoryStoreOperations struct {
	mu   sync.RWMutex
	data map[TableName]map[string]map[string]any
}

// NewInMemoryStoreOperations creates a new InMemoryStoreOperations with all
// standard Mastra tables pre-initialized.
func NewInMemoryStoreOperations() *InMemoryStoreOperations {
	s := &InMemoryStoreOperations{
		data: make(map[TableName]map[string]map[string]any),
	}

	// Initialize all standard tables.
	tables := []string{
		"mastra_workflow_snapshot",
		"mastra_messages",
		"mastra_threads",
		"mastra_traces",
		"mastra_resources",
		"mastra_scorers",
		"mastra_ai_spans",
		"mastra_agents",
		"mastra_agent_versions",
		"mastra_observational_memory",
		"mastra_prompt_blocks",
		"mastra_prompt_block_versions",
		"mastra_scorer_definitions",
		"mastra_scorer_definition_versions",
		"mastra_mcp_clients",
		"mastra_mcp_client_versions",
		"mastra_mcp_servers",
		"mastra_mcp_server_versions",
		"mastra_workspaces",
		"mastra_workspace_versions",
		"mastra_skills",
		"mastra_skill_versions",
		"mastra_skill_blobs",
		"mastra_datasets",
		"mastra_dataset_items",
		"mastra_dataset_versions",
		"mastra_experiments",
		"mastra_experiment_results",
	}
	for _, t := range tables {
		s.data[t] = make(map[string]map[string]any)
	}

	return s
}

// GetDatabase returns the underlying data map.
func (s *InMemoryStoreOperations) GetDatabase() any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data
}

// Insert inserts a single record into a table.
func (s *InMemoryStoreOperations) Insert(_ context.Context, tableName TableName, record map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	table := s.getOrCreateTable(tableName)

	key, _ := record["id"].(string)

	// Special key logic for workflow snapshots.
	if tableName == tableWorkflowSnapshot && key == "" {
		runID, _ := record["run_id"].(string)
		if runID != "" {
			wfName, _ := record["workflow_name"].(string)
			if wfName != "" {
				key = wfName + "-" + runID
			} else {
				key = runID
			}
			record["id"] = key
		}
	}

	if key == "" {
		key = fmt.Sprintf("auto-%d-%f", time.Now().UnixMilli(), rand.Float64())
		record["id"] = key
	}

	table[key] = record
	return nil
}

// BatchInsert inserts multiple records into a table.
func (s *InMemoryStoreOperations) BatchInsert(_ context.Context, tableName TableName, records []map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	table := s.getOrCreateTable(tableName)

	for _, record := range records {
		key, _ := record["id"].(string)

		if tableName == tableWorkflowSnapshot && key == "" {
			runID, _ := record["run_id"].(string)
			if runID != "" {
				key = runID
				record["id"] = key
			}
		}

		if key == "" {
			key = fmt.Sprintf("auto-%d-%f", time.Now().UnixMilli(), rand.Float64())
			record["id"] = key
		}

		table[key] = record
	}
	return nil
}

// Load retrieves a record by matching all key-value pairs.
// Returns nil if no record matches.
func (s *InMemoryStoreOperations) Load(_ context.Context, tableName TableName, keys map[string]any) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table, ok := s.data[tableName]
	if !ok {
		return nil, nil
	}

	for _, record := range table {
		match := true
		for k, v := range keys {
			if record[k] != v {
				match = false
				break
			}
		}
		if match {
			return record, nil
		}
	}
	return nil, nil
}

// CreateTable creates a new table (replaces any existing data).
func (s *InMemoryStoreOperations) CreateTable(_ context.Context, tableName TableName, _ map[string]StorageColumn) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[tableName] = make(map[string]map[string]any)
	return nil
}

// ClearTable removes all records from a table.
func (s *InMemoryStoreOperations) ClearTable(_ context.Context, tableName TableName) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if table, ok := s.data[tableName]; ok {
		for k := range table {
			delete(table, k)
		}
	}
	return nil
}

// DropTable removes a table entirely (clears it in-memory).
func (s *InMemoryStoreOperations) DropTable(_ context.Context, tableName TableName) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if table, ok := s.data[tableName]; ok {
		for k := range table {
			delete(table, k)
		}
	}
	return nil
}

// AlterTable is a no-op for in-memory storage (schema is implicit).
func (s *InMemoryStoreOperations) AlterTable(_ context.Context, _ TableName, _ map[string]StorageColumn, _ []string) error {
	return nil
}

// HasColumn always returns true for in-memory storage (schema is implicit).
func (s *InMemoryStoreOperations) HasColumn(_ context.Context, _ string, _ string) (bool, error) {
	return true, nil
}

// CreateIndex is not supported by in-memory storage — returns an error.
func (s *InMemoryStoreOperations) CreateIndex(_ context.Context, _ CreateIndexOptions) error {
	return fmt.Errorf("index management is not supported by in-memory storage")
}

// DropIndex is not supported by in-memory storage — returns an error.
func (s *InMemoryStoreOperations) DropIndex(_ context.Context, _ string) error {
	return fmt.Errorf("index management is not supported by in-memory storage")
}

// ListIndexes is not supported by in-memory storage — returns an error.
func (s *InMemoryStoreOperations) ListIndexes(_ context.Context, _ string) ([]IndexInfo, error) {
	return nil, fmt.Errorf("index management is not supported by in-memory storage")
}

// DescribeIndex is not supported by in-memory storage — returns an error.
func (s *InMemoryStoreOperations) DescribeIndex(_ context.Context, _ string) (StorageIndexStats, error) {
	return StorageIndexStats{}, fmt.Errorf("index management is not supported by in-memory storage")
}

// getOrCreateTable returns the table map, creating it if it doesn't exist.
// Must be called with the write lock held.
func (s *InMemoryStoreOperations) getOrCreateTable(tableName TableName) map[string]map[string]any {
	table, ok := s.data[tableName]
	if !ok {
		table = make(map[string]map[string]any)
		s.data[tableName] = table
	}
	return table
}
