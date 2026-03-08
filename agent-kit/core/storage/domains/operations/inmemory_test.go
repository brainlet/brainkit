// Ported from: packages/core/src/storage/domains/operations/inmemory.ts
// Tests for InMemoryStoreOperations — the in-memory implementation of StoreOperations.
//
// The TypeScript source does not have a dedicated operations test file; these tests
// are derived from the InMemoryStore implementation behavior and the StoreOperations
// interface contract, following the same patterns as the datasets domain tests.
package operations

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Constructor / Initialization
// ---------------------------------------------------------------------------

func TestNewInMemoryStoreOperations_InitializesTables(t *testing.T) {
	// The constructor pre-initializes all standard Mastra tables as empty maps.
	// This mirrors the TypeScript constructor which creates a Map() for each table.
	s := NewInMemoryStoreOperations()
	db := s.GetDatabase().(map[TableName]map[string]map[string]any)

	expectedTables := []string{
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

	for _, name := range expectedTables {
		if _, ok := db[name]; !ok {
			t.Errorf("expected table %q to be initialized, but it was not found", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Insert
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_Insert(t *testing.T) {
	ctx := context.Background()

	t.Run("should insert a record with explicit id", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		err := s.Insert(ctx, "mastra_agents", map[string]any{
			"id":   "agent-1",
			"name": "Test Agent",
		})
		if err != nil {
			t.Fatalf("Insert returned error: %v", err)
		}

		result, err := s.Load(ctx, "mastra_agents", map[string]any{"id": "agent-1"})
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected record to exist after insert")
		}
		rec := result.(map[string]any)
		if rec["name"] != "Test Agent" {
			t.Errorf("expected name='Test Agent', got %v", rec["name"])
		}
	})

	t.Run("should auto-generate id when not provided", func(t *testing.T) {
		// Mirrors TypeScript: key = `auto-${Date.now()}-${Math.random()}`
		s := NewInMemoryStoreOperations()
		record := map[string]any{"name": "No ID"}
		err := s.Insert(ctx, "mastra_agents", record)
		if err != nil {
			t.Fatalf("Insert returned error: %v", err)
		}

		generatedID, ok := record["id"].(string)
		if !ok || generatedID == "" {
			t.Fatal("expected auto-generated id to be set on record")
		}
		if !strings.HasPrefix(generatedID, "auto-") {
			t.Errorf("expected auto-generated id to start with 'auto-', got %q", generatedID)
		}
	})

	t.Run("should use workflow_name-run_id as key for workflow snapshots without id", func(t *testing.T) {
		// Mirrors TypeScript: if TABLE_WORKFLOW_SNAPSHOT && !record.id && record.run_id,
		// key = record.workflow_name ? `${record.workflow_name}-${record.run_id}` : record.run_id
		s := NewInMemoryStoreOperations()
		record := map[string]any{
			"run_id":        "run-123",
			"workflow_name": "my-workflow",
			"status":        "running",
		}
		err := s.Insert(ctx, "mastra_workflow_snapshot", record)
		if err != nil {
			t.Fatalf("Insert returned error: %v", err)
		}

		if record["id"] != "my-workflow-run-123" {
			t.Errorf("expected id='my-workflow-run-123', got %v", record["id"])
		}

		result, err := s.Load(ctx, "mastra_workflow_snapshot", map[string]any{"id": "my-workflow-run-123"})
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected record to be loadable by composite key")
		}
	})

	t.Run("should use run_id alone as key for workflow snapshots without workflow_name", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		record := map[string]any{
			"run_id": "run-456",
			"status": "completed",
		}
		err := s.Insert(ctx, "mastra_workflow_snapshot", record)
		if err != nil {
			t.Fatalf("Insert returned error: %v", err)
		}

		if record["id"] != "run-456" {
			t.Errorf("expected id='run-456', got %v", record["id"])
		}
	})

	t.Run("should overwrite existing record with same id", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "dup-1", "name": "Original"})
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "dup-1", "name": "Updated"})

		result, _ := s.Load(ctx, "mastra_agents", map[string]any{"id": "dup-1"})
		rec := result.(map[string]any)
		if rec["name"] != "Updated" {
			t.Errorf("expected name='Updated', got %v", rec["name"])
		}
	})

	t.Run("should create table on-the-fly if it does not exist", func(t *testing.T) {
		// getOrCreateTable creates the table map if missing.
		s := NewInMemoryStoreOperations()
		err := s.Insert(ctx, "custom_table", map[string]any{
			"id":    "custom-1",
			"value": "hello",
		})
		if err != nil {
			t.Fatalf("Insert returned error: %v", err)
		}

		result, err := s.Load(ctx, "custom_table", map[string]any{"id": "custom-1"})
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected record in dynamically-created table")
		}
	})
}

// ---------------------------------------------------------------------------
// BatchInsert
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_BatchInsert(t *testing.T) {
	ctx := context.Background()

	t.Run("should insert multiple records", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		records := []map[string]any{
			{"id": "b-1", "name": "Batch 1"},
			{"id": "b-2", "name": "Batch 2"},
			{"id": "b-3", "name": "Batch 3"},
		}
		err := s.BatchInsert(ctx, "mastra_agents", records)
		if err != nil {
			t.Fatalf("BatchInsert returned error: %v", err)
		}

		for _, rec := range records {
			result, _ := s.Load(ctx, "mastra_agents", map[string]any{"id": rec["id"]})
			if result == nil {
				t.Errorf("expected record with id=%v to exist", rec["id"])
			}
		}
	})

	t.Run("should auto-generate ids for records without id", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		records := []map[string]any{
			{"name": "Auto 1"},
			{"name": "Auto 2"},
		}
		err := s.BatchInsert(ctx, "mastra_agents", records)
		if err != nil {
			t.Fatalf("BatchInsert returned error: %v", err)
		}

		for i, rec := range records {
			id, ok := rec["id"].(string)
			if !ok || id == "" {
				t.Errorf("record[%d]: expected auto-generated id", i)
			}
		}
	})

	t.Run("should handle workflow snapshot key logic for batch insert", func(t *testing.T) {
		// In BatchInsert, workflow snapshot key uses run_id only (no workflow_name),
		// matching the TypeScript implementation.
		s := NewInMemoryStoreOperations()
		records := []map[string]any{
			{"run_id": "batch-run-1", "status": "running"},
			{"run_id": "batch-run-2", "status": "completed"},
		}
		err := s.BatchInsert(ctx, "mastra_workflow_snapshot", records)
		if err != nil {
			t.Fatalf("BatchInsert returned error: %v", err)
		}

		if records[0]["id"] != "batch-run-1" {
			t.Errorf("expected id='batch-run-1', got %v", records[0]["id"])
		}
		if records[1]["id"] != "batch-run-2" {
			t.Errorf("expected id='batch-run-2', got %v", records[1]["id"])
		}
	})

	t.Run("should handle empty records slice", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		err := s.BatchInsert(ctx, "mastra_agents", []map[string]any{})
		if err != nil {
			t.Fatalf("BatchInsert with empty slice returned error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_Load(t *testing.T) {
	ctx := context.Background()

	t.Run("should load record by id", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "load-1", "name": "Loadable"})

		result, err := s.Load(ctx, "mastra_agents", map[string]any{"id": "load-1"})
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected record to be found")
		}
		rec := result.(map[string]any)
		if rec["name"] != "Loadable" {
			t.Errorf("expected name='Loadable', got %v", rec["name"])
		}
	})

	t.Run("should load record by multiple key-value pairs", func(t *testing.T) {
		// Mirrors TypeScript: records.filter(record => Object.keys(keys).every(key => record[key] === keys[key]))
		s := NewInMemoryStoreOperations()
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "multi-1", "name": "Alpha", "env": "prod"})
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "multi-2", "name": "Beta", "env": "dev"})

		result, err := s.Load(ctx, "mastra_agents", map[string]any{"name": "Beta", "env": "dev"})
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected record to be found by multiple keys")
		}
		rec := result.(map[string]any)
		if rec["id"] != "multi-2" {
			t.Errorf("expected id='multi-2', got %v", rec["id"])
		}
	})

	t.Run("should return nil when no record matches", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		result, err := s.Load(ctx, "mastra_agents", map[string]any{"id": "nonexistent"})
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for nonexistent record, got %v", result)
		}
	})

	t.Run("should return nil when table does not exist", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		result, err := s.Load(ctx, "nonexistent_table", map[string]any{"id": "x"})
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for nonexistent table, got %v", result)
		}
	})

	t.Run("should return nil when keys partially match", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "partial-1", "name": "Foo", "env": "prod"})

		// name matches but env does not
		result, err := s.Load(ctx, "mastra_agents", map[string]any{"name": "Foo", "env": "dev"})
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for partial key match, got %v", result)
		}
	})
}

// ---------------------------------------------------------------------------
// CreateTable
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_CreateTable(t *testing.T) {
	ctx := context.Background()

	t.Run("should create a new table", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		schema := map[string]StorageColumn{
			"id":   {Type: "uuid", PrimaryKey: true},
			"name": {Type: "text"},
		}
		err := s.CreateTable(ctx, "new_table", schema)
		if err != nil {
			t.Fatalf("CreateTable returned error: %v", err)
		}

		// Should be able to insert into the new table.
		err = s.Insert(ctx, "new_table", map[string]any{"id": "ct-1", "name": "Created"})
		if err != nil {
			t.Fatalf("Insert into new table returned error: %v", err)
		}
	})

	t.Run("should replace existing table data on create", func(t *testing.T) {
		// Mirrors TypeScript: this.data[tableName] = new Map()
		s := NewInMemoryStoreOperations()
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "old-1", "name": "Old"})

		err := s.CreateTable(ctx, "mastra_agents", map[string]StorageColumn{})
		if err != nil {
			t.Fatalf("CreateTable returned error: %v", err)
		}

		result, _ := s.Load(ctx, "mastra_agents", map[string]any{"id": "old-1"})
		if result != nil {
			t.Error("expected existing data to be cleared after CreateTable")
		}
	})
}

// ---------------------------------------------------------------------------
// ClearTable
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_ClearTable(t *testing.T) {
	ctx := context.Background()

	t.Run("should remove all records from a table", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "c-1", "name": "Agent1"})
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "c-2", "name": "Agent2"})

		err := s.ClearTable(ctx, "mastra_agents")
		if err != nil {
			t.Fatalf("ClearTable returned error: %v", err)
		}

		r1, _ := s.Load(ctx, "mastra_agents", map[string]any{"id": "c-1"})
		r2, _ := s.Load(ctx, "mastra_agents", map[string]any{"id": "c-2"})
		if r1 != nil || r2 != nil {
			t.Error("expected all records to be cleared")
		}
	})

	t.Run("should be a no-op for nonexistent table", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		err := s.ClearTable(ctx, "nonexistent_table")
		if err != nil {
			t.Fatalf("ClearTable on nonexistent table returned error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// DropTable
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_DropTable(t *testing.T) {
	ctx := context.Background()

	t.Run("should clear all records from a table", func(t *testing.T) {
		// In-memory DropTable behaves the same as ClearTable — deletes all entries.
		// Mirrors TypeScript: this.data[tableName].clear()
		s := NewInMemoryStoreOperations()
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "d-1", "name": "Drop1"})

		err := s.DropTable(ctx, "mastra_agents")
		if err != nil {
			t.Fatalf("DropTable returned error: %v", err)
		}

		result, _ := s.Load(ctx, "mastra_agents", map[string]any{"id": "d-1"})
		if result != nil {
			t.Error("expected record to be gone after DropTable")
		}
	})

	t.Run("should be a no-op for nonexistent table", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		err := s.DropTable(ctx, "nonexistent_table")
		if err != nil {
			t.Fatalf("DropTable on nonexistent table returned error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// AlterTable
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_AlterTable(t *testing.T) {
	ctx := context.Background()

	t.Run("should be a no-op for in-memory storage", func(t *testing.T) {
		// Mirrors TypeScript: alterTable is logged but does nothing.
		s := NewInMemoryStoreOperations()
		err := s.AlterTable(ctx, "mastra_agents", map[string]StorageColumn{
			"new_col": {Type: "text"},
		}, []string{"new_col"})
		if err != nil {
			t.Fatalf("AlterTable returned error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// HasColumn
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_HasColumn(t *testing.T) {
	ctx := context.Background()

	t.Run("should always return true for in-memory storage", func(t *testing.T) {
		// Mirrors TypeScript: hasColumn always returns true for mock store.
		s := NewInMemoryStoreOperations()
		result, err := s.HasColumn(ctx, "mastra_agents", "nonexistent_column")
		if err != nil {
			t.Fatalf("HasColumn returned error: %v", err)
		}
		if !result {
			t.Error("expected HasColumn to return true for in-memory storage")
		}
	})

	t.Run("should return true for nonexistent table too", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		result, err := s.HasColumn(ctx, "fake_table", "fake_column")
		if err != nil {
			t.Fatalf("HasColumn returned error: %v", err)
		}
		if !result {
			t.Error("expected HasColumn to return true even for nonexistent table")
		}
	})
}

// ---------------------------------------------------------------------------
// GetDatabase
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_GetDatabase(t *testing.T) {
	t.Run("should return the underlying data map", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		db := s.GetDatabase()
		if db == nil {
			t.Fatal("expected GetDatabase to return non-nil")
		}
		dataMap, ok := db.(map[TableName]map[string]map[string]any)
		if !ok {
			t.Fatalf("expected GetDatabase to return map type, got %T", db)
		}
		// Should contain the pre-initialized tables.
		if _, exists := dataMap["mastra_agents"]; !exists {
			t.Error("expected mastra_agents table in database")
		}
	})
}

// ---------------------------------------------------------------------------
// Index Management (unsupported in-memory — should return errors)
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_IndexManagement(t *testing.T) {
	ctx := context.Background()
	s := NewInMemoryStoreOperations()

	t.Run("CreateIndex should return error", func(t *testing.T) {
		err := s.CreateIndex(ctx, CreateIndexOptions{
			Name:    "idx_test",
			Table:   "mastra_agents",
			Columns: []string{"name"},
		})
		if err == nil {
			t.Fatal("expected CreateIndex to return error for in-memory storage")
		}
		if !strings.Contains(err.Error(), "not supported") {
			t.Errorf("expected error message to contain 'not supported', got %q", err.Error())
		}
	})

	t.Run("DropIndex should return error", func(t *testing.T) {
		err := s.DropIndex(ctx, "idx_test")
		if err == nil {
			t.Fatal("expected DropIndex to return error for in-memory storage")
		}
		if !strings.Contains(err.Error(), "not supported") {
			t.Errorf("expected error message to contain 'not supported', got %q", err.Error())
		}
	})

	t.Run("ListIndexes should return error", func(t *testing.T) {
		indexes, err := s.ListIndexes(ctx, "mastra_agents")
		if err == nil {
			t.Fatal("expected ListIndexes to return error for in-memory storage")
		}
		if indexes != nil {
			t.Errorf("expected nil indexes, got %v", indexes)
		}
		if !strings.Contains(err.Error(), "not supported") {
			t.Errorf("expected error message to contain 'not supported', got %q", err.Error())
		}
	})

	t.Run("DescribeIndex should return error", func(t *testing.T) {
		stats, err := s.DescribeIndex(ctx, "idx_test")
		if err == nil {
			t.Fatal("expected DescribeIndex to return error for in-memory storage")
		}
		// Should return zero-value StorageIndexStats.
		if stats.Name != "" {
			t.Errorf("expected empty stats name, got %q", stats.Name)
		}
		if !strings.Contains(err.Error(), "not supported") {
			t.Errorf("expected error message to contain 'not supported', got %q", err.Error())
		}
	})
}

// ---------------------------------------------------------------------------
// Interface Compliance
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_ImplementsInterface(t *testing.T) {
	// Compile-time check is already in inmemory.go, but we verify at test time too.
	var _ StoreOperations = (*InMemoryStoreOperations)(nil)
}

// ---------------------------------------------------------------------------
// Concurrency Safety
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	s := NewInMemoryStoreOperations()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Concurrent writers.
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			_ = s.Insert(ctx, "mastra_agents", map[string]any{
				"id":   strings.Replace("conc-XXXX", "XXXX", strings.Repeat("x", idx%10+1), 1),
				"name": "concurrent",
			})
		}(i)
	}

	// Concurrent readers.
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = s.Load(ctx, "mastra_agents", map[string]any{"name": "concurrent"})
		}()
	}

	wg.Wait()
	// If we reach here without a race condition panic, the test passes.
}

// ---------------------------------------------------------------------------
// Cross-Table Isolation
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_CrossTableIsolation(t *testing.T) {
	ctx := context.Background()
	s := NewInMemoryStoreOperations()

	// Insert into one table.
	_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "iso-1", "name": "Agent"})
	// Insert into another table with the same id.
	_ = s.Insert(ctx, "mastra_threads", map[string]any{"id": "iso-1", "title": "Thread"})

	// Load from agents should return agent record.
	agentResult, _ := s.Load(ctx, "mastra_agents", map[string]any{"id": "iso-1"})
	if agentResult == nil {
		t.Fatal("expected agent record to exist")
	}
	agentRec := agentResult.(map[string]any)
	if agentRec["name"] != "Agent" {
		t.Errorf("expected name='Agent', got %v", agentRec["name"])
	}

	// Load from threads should return thread record.
	threadResult, _ := s.Load(ctx, "mastra_threads", map[string]any{"id": "iso-1"})
	if threadResult == nil {
		t.Fatal("expected thread record to exist")
	}
	threadRec := threadResult.(map[string]any)
	if threadRec["title"] != "Thread" {
		t.Errorf("expected title='Thread', got %v", threadRec["title"])
	}

	// Clearing one table should not affect the other.
	_ = s.ClearTable(ctx, "mastra_agents")
	agentResult, _ = s.Load(ctx, "mastra_agents", map[string]any{"id": "iso-1"})
	if agentResult != nil {
		t.Error("expected agent record to be cleared")
	}

	threadResult, _ = s.Load(ctx, "mastra_threads", map[string]any{"id": "iso-1"})
	if threadResult == nil {
		t.Error("expected thread record to still exist after clearing agents table")
	}
}

// ---------------------------------------------------------------------------
// Workflow Snapshot Edge Cases
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_WorkflowSnapshotEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("should use explicit id even for workflow snapshots if provided", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		record := map[string]any{
			"id":     "explicit-id",
			"run_id": "run-999",
			"status": "running",
		}
		_ = s.Insert(ctx, "mastra_workflow_snapshot", record)

		// When id is already set, it should be used as-is (not overwritten by run_id).
		if record["id"] != "explicit-id" {
			t.Errorf("expected id to remain 'explicit-id', got %v", record["id"])
		}

		result, _ := s.Load(ctx, "mastra_workflow_snapshot", map[string]any{"id": "explicit-id"})
		if result == nil {
			t.Fatal("expected record to be loadable by explicit id")
		}
	})

	t.Run("should auto-generate id for workflow snapshot without run_id or id", func(t *testing.T) {
		s := NewInMemoryStoreOperations()
		record := map[string]any{
			"status": "pending",
		}
		_ = s.Insert(ctx, "mastra_workflow_snapshot", record)

		id, ok := record["id"].(string)
		if !ok || !strings.HasPrefix(id, "auto-") {
			t.Errorf("expected auto-generated id, got %v", record["id"])
		}
	})
}

// ---------------------------------------------------------------------------
// Multiple Loads and Filtering
// ---------------------------------------------------------------------------

func TestInMemoryStoreOperations_LoadFirstMatch(t *testing.T) {
	ctx := context.Background()

	t.Run("should return first matching record when multiple match", func(t *testing.T) {
		// Load returns the first record that matches all keys.
		// This is consistent with the TS filter()[0] behavior.
		s := NewInMemoryStoreOperations()
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "fm-1", "env": "prod"})
		_ = s.Insert(ctx, "mastra_agents", map[string]any{"id": "fm-2", "env": "prod"})

		result, err := s.Load(ctx, "mastra_agents", map[string]any{"env": "prod"})
		if err != nil {
			t.Fatalf("Load returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected at least one record to match")
		}
		rec := result.(map[string]any)
		id := rec["id"].(string)
		if id != "fm-1" && id != "fm-2" {
			t.Errorf("expected id to be 'fm-1' or 'fm-2', got %q", id)
		}
	})
}
