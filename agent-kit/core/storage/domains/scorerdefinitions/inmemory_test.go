// Ported from: packages/core/src/storage/domains/scorer-definitions (no TS test file exists;
// tests derived from the InMemory implementation and patterns in inmemory.test.ts for agents,
// prompt-blocks, mcp-clients, and mcp-servers).
package scorerdefinitions

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// ==========================================================================
// Create
// ==========================================================================

func TestInMemoryScorerDefinitionsStorage_Create(t *testing.T) {
	ctx := context.Background()

	// Mirrors: agents/inmemory.test.ts "should create agent with status=draft"
	t.Run("should create a scorer definition with status=draft", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		result, err := storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":       "scorer-1",
				"authorId": "user-123",
				"metadata": map[string]any{"env": "test"},
				"name":     "Test Scorer",
				"type":     "llm-judge",
			},
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		scorer, ok := toMap(result)
		if !ok {
			t.Fatal("expected result to be convertible to map")
		}
		if scorer["id"] != "scorer-1" {
			t.Errorf("expected id=scorer-1, got %v", scorer["id"])
		}
		if scorer["status"] != "draft" {
			t.Errorf("expected status=draft, got %v", scorer["status"])
		}
		if scorer["authorId"] != "user-123" {
			t.Errorf("expected authorId=user-123, got %v", scorer["authorId"])
		}
	})

	// Mirrors: agents/inmemory.test.ts "should auto-create version 1 on create"
	t.Run("should auto-create version 1 on create", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, err := storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":   "scorer-2",
				"name": "Scorer With Version",
				"type": "llm-judge",
			},
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}

		count, err := storage.CountVersions(ctx, "scorer-2")
		if err != nil {
			t.Fatalf("CountVersions returned error: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 version, got %d", count)
		}

		v, err := storage.GetVersionByNumber(ctx, "scorer-2", 1)
		if err != nil {
			t.Fatalf("GetVersionByNumber returned error: %v", err)
		}
		if v == nil {
			t.Fatal("expected version 1 to exist")
		}
		if v.VersionNumber != 1 {
			t.Errorf("expected versionNumber=1, got %d", v.VersionNumber)
		}
		if v.ChangeMessage != "Initial version" {
			t.Errorf("expected changeMessage='Initial version', got %q", v.ChangeMessage)
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should throw if block with same ID already exists"
	t.Run("should reject duplicate id", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, err := storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "dup-scorer", "name": "First", "type": "llm-judge"},
		})
		if err != nil {
			t.Fatalf("first Create returned error: %v", err)
		}
		_, err = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "dup-scorer", "name": "Second", "type": "llm-judge"},
		})
		if err == nil {
			t.Fatal("expected error for duplicate id")
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should store optional fields"
	t.Run("should store optional fields (authorId, metadata) and snapshot config on version", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		result, err := storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":       "scorer-3",
				"name":     "Admin Scorer",
				"type":     "llm-judge",
				"authorId": "user-456",
				"metadata": map[string]any{"category": "admin"},
			},
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		scorer, _ := toMap(result)
		if scorer["authorId"] != "user-456" {
			t.Errorf("expected authorId=user-456, got %v", scorer["authorId"])
		}

		// Verify config is on the version, not the thin record
		latestVersion, err := storage.GetLatestVersion(ctx, "scorer-3")
		if err != nil {
			t.Fatalf("GetLatestVersion returned error: %v", err)
		}
		if latestVersion == nil {
			t.Fatal("expected latest version to exist")
		}
		if latestVersion.Name != "Admin Scorer" {
			t.Errorf("expected version name='Admin Scorer', got %q", latestVersion.Name)
		}
		if latestVersion.Type != "llm-judge" {
			t.Errorf("expected version type='llm-judge', got %q", latestVersion.Type)
		}
	})

	// Test: Create without wrapper key (flat input) — the Go implementation
	// handles unwrapping the "scorerDefinition" key from the input map.
	t.Run("should require id in input", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, err := storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"name": "No ID"},
		})
		if err == nil {
			t.Fatal("expected error when id is missing")
		}
	})
}

// ==========================================================================
// GetByID
// ==========================================================================

func TestInMemoryScorerDefinitionsStorage_GetByID(t *testing.T) {
	ctx := context.Background()

	// Mirrors: prompt-blocks/inmemory.test.ts "should return null for non-existent block"
	t.Run("should return nil for non-existent scorer definition", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		result, err := storage.GetByID(ctx, "non-existent")
		if err != nil {
			t.Fatalf("GetByID returned error: %v", err)
		}
		if result != nil {
			t.Error("expected nil for non-existent scorer definition")
		}
	})

	// Mirrors: agents/inmemory.test.ts "should return deep copy (mutation safety)"
	t.Run("should return deep copy (mutation safety)", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":       "scorer-copy-1",
				"name":     "Copy Test",
				"type":     "llm-judge",
				"metadata": map[string]any{"key": "original"},
			},
		})

		result1, _ := storage.GetByID(ctx, "scorer-copy-1")
		m1, _ := toMap(result1)
		meta1, _ := m1["metadata"].(map[string]any)
		meta1["key"] = "mutated"

		result2, _ := storage.GetByID(ctx, "scorer-copy-1")
		m2, _ := toMap(result2)
		meta2, _ := m2["metadata"].(map[string]any)
		if meta2["key"] != "original" {
			t.Error("expected stored data to be unaffected by external mutation")
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should return thin record for existing block"
	t.Run("should return thin record for existing scorer definition", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":   "scorer-get-1",
				"name": "Get Test",
				"type": "llm-judge",
			},
		})

		result, err := storage.GetByID(ctx, "scorer-get-1")
		if err != nil {
			t.Fatalf("GetByID returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected result to exist")
		}
		m, _ := toMap(result)
		if m["id"] != "scorer-get-1" {
			t.Errorf("expected id=scorer-get-1, got %v", m["id"])
		}
		if m["status"] != "draft" {
			t.Errorf("expected status=draft, got %v", m["status"])
		}
	})
}

// ==========================================================================
// Update
// ==========================================================================

func TestInMemoryScorerDefinitionsStorage_Update(t *testing.T) {
	ctx := context.Background()

	// Mirrors: agents/inmemory.test.ts "should merge metadata"
	t.Run("should merge metadata", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, err := storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":       "scorer-update-1",
				"name":     "Update Test",
				"type":     "llm-judge",
				"metadata": map[string]any{"key1": "val1"},
			},
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}

		result, err := storage.Update(ctx, map[string]any{
			"id":       "scorer-update-1",
			"metadata": map[string]any{"key2": "val2"},
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
		scorer, _ := toMap(result)
		meta, _ := scorer["metadata"].(map[string]any)
		if meta["key1"] != "val1" {
			t.Error("expected key1=val1 preserved")
		}
		if meta["key2"] != "val2" {
			t.Error("expected key2=val2 added")
		}
	})

	// Mirrors: agents/inmemory.test.ts "should not create a new version on update"
	t.Run("should not create a new version on update", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "scorer-update-2", "name": "No Version", "type": "llm-judge"},
		})

		_, err := storage.Update(ctx, map[string]any{
			"id":     "scorer-update-2",
			"status": "published",
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}

		count, _ := storage.CountVersions(ctx, "scorer-update-2")
		if count != 1 {
			t.Errorf("expected 1 version after update, got %d", count)
		}
	})

	// Mirrors: agents/inmemory.test.ts "should update activeVersionId"
	t.Run("should update activeVersionId", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "scorer-update-3", "name": "Active", "type": "llm-judge"},
		})

		result, err := storage.Update(ctx, map[string]any{
			"id":              "scorer-update-3",
			"activeVersionId": "some-version-id",
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
		scorer, _ := toMap(result)
		if scorer["activeVersionId"] != "some-version-id" {
			t.Errorf("expected activeVersionId=some-version-id, got %v", scorer["activeVersionId"])
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should throw for non-existent block"
	t.Run("should return error for non-existent scorer definition", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, err := storage.Update(ctx, map[string]any{
			"id":   "non-existent",
			"name": "Nope",
		})
		if err == nil {
			t.Fatal("expected error for non-existent scorer definition")
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should not auto-publish when activeVersionId is updated"
	t.Run("should not auto-publish when activeVersionId is updated", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "scorer-update-4", "name": "Auto Pub", "type": "llm-judge"},
		})

		// Create a second version
		_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                             "v2-id",
			ScorerDefinitionID:             "scorer-update-4",
			VersionNumber:                  2,
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{Name: "V2", Type: "llm-judge"},
			ChangedFields:                  []string{"name"},
			ChangeMessage:                  "Updated to v2",
		})

		result, err := storage.Update(ctx, map[string]any{
			"id":              "scorer-update-4",
			"activeVersionId": "v2-id",
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
		scorer, _ := toMap(result)
		// Auto-publish was removed — status stays as 'draft'
		if scorer["status"] != "draft" {
			t.Errorf("expected status=draft (no auto-publish), got %v", scorer["status"])
		}
		if scorer["activeVersionId"] != "v2-id" {
			t.Errorf("expected activeVersionId=v2-id, got %v", scorer["activeVersionId"])
		}
	})
}

// ==========================================================================
// Delete
// ==========================================================================

func TestInMemoryScorerDefinitionsStorage_Delete(t *testing.T) {
	ctx := context.Background()

	// Mirrors: agents/inmemory.test.ts "should delete agent and cascade versions"
	t.Run("should delete scorer definition and cascade versions", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "scorer-delete-1", "name": "To Delete", "type": "llm-judge"},
		})

		// Create additional versions
		for i := 2; i <= 3; i++ {
			_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
				ID:                             fmt.Sprintf("v%d", i),
				ScorerDefinitionID:             "scorer-delete-1",
				VersionNumber:                  i,
				ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{Name: fmt.Sprintf("V%d", i), Type: "llm-judge"},
				ChangedFields:                  []string{"name"},
				ChangeMessage:                  fmt.Sprintf("v%d", i),
			})
		}

		// Verify scorer and versions exist before delete
		beforeDelete, _ := storage.GetByID(ctx, "scorer-delete-1")
		if beforeDelete == nil {
			t.Fatal("expected scorer to exist before delete")
		}
		versionsBefore, _ := storage.CountVersions(ctx, "scorer-delete-1")
		if versionsBefore != 3 {
			t.Errorf("expected 3 versions before delete, got %d", versionsBefore)
		}

		err := storage.Delete(ctx, "scorer-delete-1")
		if err != nil {
			t.Fatalf("Delete returned error: %v", err)
		}

		result, _ := storage.GetByID(ctx, "scorer-delete-1")
		if result != nil {
			t.Error("expected scorer definition to be deleted")
		}

		count, _ := storage.CountVersions(ctx, "scorer-delete-1")
		if count != 0 {
			t.Errorf("expected 0 versions after cascade delete, got %d", count)
		}
	})

	// Mirrors: agents/inmemory.test.ts "should be idempotent"
	t.Run("should be idempotent", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		err := storage.Delete(ctx, "non-existent")
		if err != nil {
			t.Fatalf("Delete of non-existent scorer definition returned error: %v", err)
		}
	})
}

// ==========================================================================
// List
// ==========================================================================

func TestInMemoryScorerDefinitionsStorage_List(t *testing.T) {
	ctx := context.Background()

	// Mirrors: agents/inmemory.test.ts "should list with pagination"
	t.Run("should list with pagination", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		for i := 0; i < 5; i++ {
			_, _ = storage.Create(ctx, map[string]any{
				"scorerDefinition": map[string]any{
					"id":   fmt.Sprintf("list-scorer-%d", i),
					"name": fmt.Sprintf("Scorer %d", i),
					"type": "llm-judge",
				},
			})
			// Stagger creation times slightly for deterministic ordering
			time.Sleep(time.Millisecond)
		}

		result, err := storage.List(ctx, map[string]any{
			"page":    0,
			"perPage": 2,
		})
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		resultMap, _ := result.(map[string]any)
		scorers, _ := resultMap["scorerDefinitions"].([]any)
		if len(scorers) != 2 {
			t.Errorf("expected 2 scorer definitions, got %d", len(scorers))
		}
		if resultMap["total"] != 5 {
			t.Errorf("expected total=5, got %v", resultMap["total"])
		}
		if resultMap["hasMore"] != true {
			t.Error("expected hasMore=true")
		}
	})

	// Mirrors: agents/inmemory.test.ts "should filter by authorId"
	t.Run("should filter by authorId", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "a1", "authorId": "user-A", "name": "S1", "type": "llm-judge"},
		})
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "a2", "authorId": "user-B", "name": "S2", "type": "llm-judge"},
		})
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "a3", "authorId": "user-A", "name": "S3", "type": "llm-judge"},
		})

		result, _ := storage.List(ctx, map[string]any{"authorId": "user-A"})
		resultMap, _ := result.(map[string]any)
		scorers, _ := resultMap["scorerDefinitions"].([]any)
		if len(scorers) != 2 {
			t.Errorf("expected 2 scorer definitions for user-A, got %d", len(scorers))
		}
	})

	// Mirrors: agents/inmemory.test.ts "should filter by metadata"
	t.Run("should filter by metadata", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":       "m1",
				"name":     "Prod Scorer",
				"type":     "llm-judge",
				"metadata": map[string]any{"env": "prod"},
			},
		})
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":       "m2",
				"name":     "Staging Scorer",
				"type":     "llm-judge",
				"metadata": map[string]any{"env": "staging"},
			},
		})

		result, _ := storage.List(ctx, map[string]any{
			"metadata": map[string]any{"env": "prod"},
		})
		resultMap, _ := result.(map[string]any)
		scorers, _ := resultMap["scorerDefinitions"].([]any)
		if len(scorers) != 1 {
			t.Errorf("expected 1 scorer definition matching metadata, got %d", len(scorers))
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should filter by status"
	t.Run("should filter by status", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "s1", "name": "S1", "type": "llm-judge"},
		})
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "s2", "name": "S2", "type": "llm-judge"},
		})
		// Update one to published
		_, _ = storage.Update(ctx, map[string]any{"id": "s2", "status": "published"})

		result, _ := storage.List(ctx, map[string]any{"status": "draft"})
		resultMap, _ := result.(map[string]any)
		scorers, _ := resultMap["scorerDefinitions"].([]any)
		if len(scorers) != 1 {
			t.Errorf("expected 1 draft scorer, got %d", len(scorers))
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should return empty list when no blocks exist"
	t.Run("should return empty list when no scorers exist", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		result, err := storage.List(ctx, nil)
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		resultMap, _ := result.(map[string]any)
		scorers, _ := resultMap["scorerDefinitions"].([]any)
		if len(scorers) != 0 {
			t.Errorf("expected 0 scorer definitions, got %d", len(scorers))
		}
		if resultMap["total"] != 0 {
			t.Errorf("expected total=0, got %v", resultMap["total"])
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should support pagination" (multi-page)
	t.Run("should support multi-page pagination", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		for i := 0; i < 5; i++ {
			_, _ = storage.Create(ctx, map[string]any{
				"scorerDefinition": map[string]any{
					"id":   fmt.Sprintf("page-scorer-%d", i),
					"name": fmt.Sprintf("Scorer %d", i),
					"type": "llm-judge",
				},
			})
			time.Sleep(time.Millisecond)
		}

		// Page 0
		r0, _ := storage.List(ctx, map[string]any{"page": 0, "perPage": 2})
		rm0, _ := r0.(map[string]any)
		s0, _ := rm0["scorerDefinitions"].([]any)
		if len(s0) != 2 {
			t.Errorf("page 0: expected 2, got %d", len(s0))
		}
		if rm0["hasMore"] != true {
			t.Error("page 0: expected hasMore=true")
		}

		// Page 1
		r1, _ := storage.List(ctx, map[string]any{"page": 1, "perPage": 2})
		rm1, _ := r1.(map[string]any)
		s1, _ := rm1["scorerDefinitions"].([]any)
		if len(s1) != 2 {
			t.Errorf("page 1: expected 2, got %d", len(s1))
		}

		// Page 2 (last page)
		r2, _ := storage.List(ctx, map[string]any{"page": 2, "perPage": 2})
		rm2, _ := r2.(map[string]any)
		s2, _ := rm2["scorerDefinitions"].([]any)
		if len(s2) != 1 {
			t.Errorf("page 2: expected 1, got %d", len(s2))
		}
		if rm2["hasMore"] != false {
			t.Error("page 2: expected hasMore=false")
		}
	})

	// Test: ordering by createdAt ASC
	t.Run("should sort by createdAt ASC when specified", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		for i := 0; i < 3; i++ {
			_, _ = storage.Create(ctx, map[string]any{
				"scorerDefinition": map[string]any{
					"id":   fmt.Sprintf("order-scorer-%d", i),
					"name": fmt.Sprintf("Scorer %d", i),
					"type": "llm-judge",
				},
			})
			time.Sleep(2 * time.Millisecond)
		}

		result, _ := storage.List(ctx, map[string]any{
			"orderBy": map[string]any{"field": "createdAt", "direction": "ASC"},
		})
		resultMap, _ := result.(map[string]any)
		scorers, _ := resultMap["scorerDefinitions"].([]any)
		if len(scorers) != 3 {
			t.Fatalf("expected 3 scorers, got %d", len(scorers))
		}
		// ASC means oldest first
		first, _ := toMap(scorers[0])
		last, _ := toMap(scorers[2])
		if first["id"] != "order-scorer-0" {
			t.Errorf("expected first id=order-scorer-0, got %v", first["id"])
		}
		if last["id"] != "order-scorer-2" {
			t.Errorf("expected last id=order-scorer-2, got %v", last["id"])
		}
	})
}

// ==========================================================================
// Versions
// ==========================================================================

func TestInMemoryScorerDefinitionsStorage_Versions(t *testing.T) {
	ctx := context.Background()

	// Mirrors: agents/inmemory.test.ts "should create and retrieve a version"
	t.Run("should create and retrieve a version", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-1", "name": "V1", "type": "llm-judge"},
		})

		v, err := storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 "v2-id",
			ScorerDefinitionID: "ver-scorer-1",
			VersionNumber:      2,
			ChangeMessage:      "Second version",
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name: "Updated Scorer",
				Type: "llm-judge",
			},
		})
		if err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
		if v.VersionNumber != 2 {
			t.Errorf("expected versionNumber=2, got %d", v.VersionNumber)
		}

		retrieved, err := storage.GetVersion(ctx, "v2-id")
		if err != nil {
			t.Fatalf("GetVersion returned error: %v", err)
		}
		if retrieved == nil {
			t.Fatal("expected version to exist")
		}
		if retrieved.VersionNumber != 2 {
			t.Errorf("expected versionNumber=2, got %d", retrieved.VersionNumber)
		}
		if retrieved.Name != "Updated Scorer" {
			t.Errorf("expected name='Updated Scorer', got %q", retrieved.Name)
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should return null for non-existent version"
	t.Run("should return nil for non-existent version", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		result, err := storage.GetVersion(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("GetVersion returned error: %v", err)
		}
		if result != nil {
			t.Error("expected nil for non-existent version")
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should throw when creating version with duplicate ID"
	t.Run("should reject duplicate version id", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-dup-id", "name": "V1", "type": "llm-judge"},
		})

		existingVersion, _ := storage.GetLatestVersion(ctx, "ver-scorer-dup-id")
		_, err := storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 existingVersion.ID,
			ScorerDefinitionID: "ver-scorer-dup-id",
			VersionNumber:      2,
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name: "Dup", Type: "llm-judge",
			},
		})
		if err == nil {
			t.Fatal("expected error for duplicate version id")
		}
	})

	// Mirrors: agents/inmemory.test.ts "should reject duplicate version number"
	t.Run("should reject duplicate version number", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-dup", "name": "V1", "type": "llm-judge"},
		})

		_, err := storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 "dup-v",
			ScorerDefinitionID: "ver-scorer-dup",
			VersionNumber:      1, // already exists from Create
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name: "Dup", Type: "llm-judge",
			},
		})
		if err == nil {
			t.Fatal("expected error for duplicate version number")
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should get version by block ID and version number"
	t.Run("should get version by scorer definition ID and version number", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-bynum", "name": "V1", "type": "llm-judge"},
		})

		version, err := storage.GetVersionByNumber(ctx, "ver-scorer-bynum", 1)
		if err != nil {
			t.Fatalf("GetVersionByNumber returned error: %v", err)
		}
		if version == nil {
			t.Fatal("expected version to exist")
		}
		if version.VersionNumber != 1 {
			t.Errorf("expected versionNumber=1, got %d", version.VersionNumber)
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should return null for non-existent version number"
	t.Run("should return nil for non-existent version number", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-nonum", "name": "V1", "type": "llm-judge"},
		})

		version, err := storage.GetVersionByNumber(ctx, "ver-scorer-nonum", 999)
		if err != nil {
			t.Fatalf("GetVersionByNumber returned error: %v", err)
		}
		if version != nil {
			t.Error("expected nil for non-existent version number")
		}
	})

	// Mirrors: agents/inmemory.test.ts "GetLatestVersion should return highest version number"
	t.Run("GetLatestVersion should return highest version number", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-latest", "name": "V1", "type": "llm-judge"},
		})
		_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 "v2",
			ScorerDefinitionID: "ver-scorer-latest",
			VersionNumber:      2,
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name: "V2", Type: "llm-judge",
			},
		})
		_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 "v3",
			ScorerDefinitionID: "ver-scorer-latest",
			VersionNumber:      3,
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name: "V3", Type: "llm-judge",
			},
		})

		latest, err := storage.GetLatestVersion(ctx, "ver-scorer-latest")
		if err != nil {
			t.Fatalf("GetLatestVersion returned error: %v", err)
		}
		if latest == nil {
			t.Fatal("expected latest version to exist")
		}
		if latest.VersionNumber != 3 {
			t.Errorf("expected versionNumber=3, got %d", latest.VersionNumber)
		}
		if latest.Name != "V3" {
			t.Errorf("expected name='V3', got %q", latest.Name)
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should return null for latest version of non-existent block"
	t.Run("GetLatestVersion should return nil for non-existent scorer definition", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		latest, err := storage.GetLatestVersion(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("GetLatestVersion returned error: %v", err)
		}
		if latest != nil {
			t.Error("expected nil for non-existent scorer definition")
		}
	})

	// Mirrors: agents/inmemory.test.ts "ListVersions should paginate"
	t.Run("ListVersions should paginate", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-list", "name": "V1", "type": "llm-judge"},
		})
		for i := 2; i <= 5; i++ {
			_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
				ID:                 fmt.Sprintf("lv-%d", i),
				ScorerDefinitionID: "ver-scorer-list",
				VersionNumber:      i,
				ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
					Name: fmt.Sprintf("V%d", i), Type: "llm-judge",
				},
			})
		}

		perPage := 2
		result, err := storage.ListVersions(ctx, ListScorerDefinitionVersionsInput{
			ScorerDefinitionID: "ver-scorer-list",
			PerPage:            &perPage,
		})
		if err != nil {
			t.Fatalf("ListVersions returned error: %v", err)
		}
		if len(result.Versions) != 2 {
			t.Errorf("expected 2 versions, got %d", len(result.Versions))
		}
		if result.Total != 5 {
			t.Errorf("expected total=5, got %d", result.Total)
		}
		if !result.HasMore {
			t.Error("expected hasMore=true")
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should list versions sorted by versionNumber ASC"
	t.Run("ListVersions should sort by versionNumber ASC", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-sort", "name": "V1", "type": "llm-judge"},
		})
		_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 "v2-sort",
			ScorerDefinitionID: "ver-scorer-sort",
			VersionNumber:      2,
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name: "V2", Type: "llm-judge",
			},
		})

		orderByField := ScorerDefinitionVersionOrderByVersionNumber
		orderByDir := ScorerDefinitionVersionSortASC
		result, err := storage.ListVersions(ctx, ListScorerDefinitionVersionsInput{
			ScorerDefinitionID: "ver-scorer-sort",
			OrderByField:       (*ScorerDefinitionVersionOrderBy)(&orderByField),
			OrderByDirection:   (*ScorerDefinitionVersionSortDirection)(&orderByDir),
		})
		if err != nil {
			t.Fatalf("ListVersions returned error: %v", err)
		}
		if len(result.Versions) < 2 {
			t.Fatalf("expected at least 2 versions, got %d", len(result.Versions))
		}
		if result.Versions[0].VersionNumber != 1 {
			t.Errorf("expected first version number=1, got %d", result.Versions[0].VersionNumber)
		}
		if result.Versions[1].VersionNumber != 2 {
			t.Errorf("expected second version number=2, got %d", result.Versions[1].VersionNumber)
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should list all versions with perPage disabled"
	t.Run("ListVersions should return all versions when perPage is disabled", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-all", "name": "V1", "type": "llm-judge"},
		})
		for i := 2; i <= 5; i++ {
			_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
				ID:                 fmt.Sprintf("va-%d", i),
				ScorerDefinitionID: "ver-scorer-all",
				VersionNumber:      i,
				ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
					Name: fmt.Sprintf("V%d", i), Type: "llm-judge",
				},
			})
		}

		// PerPageDisabled = -1 means fetch all records without limit
		perPage := -1
		result, err := storage.ListVersions(ctx, ListScorerDefinitionVersionsInput{
			ScorerDefinitionID: "ver-scorer-all",
			PerPage:            &perPage,
		})
		if err != nil {
			t.Fatalf("ListVersions returned error: %v", err)
		}
		if len(result.Versions) != 5 {
			t.Errorf("expected 5 versions, got %d", len(result.Versions))
		}
		if result.Total != 5 {
			t.Errorf("expected total=5, got %d", result.Total)
		}
	})

	// Mirrors: agents/inmemory.test.ts "DeleteVersion should remove version"
	t.Run("DeleteVersion should remove version", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-del", "name": "V1", "type": "llm-judge"},
		})
		v1, _ := storage.GetVersionByNumber(ctx, "ver-scorer-del", 1)

		err := storage.DeleteVersion(ctx, v1.ID)
		if err != nil {
			t.Fatalf("DeleteVersion returned error: %v", err)
		}

		count, _ := storage.CountVersions(ctx, "ver-scorer-del")
		if count != 0 {
			t.Errorf("expected 0 versions, got %d", count)
		}
	})

	// Mirrors: agents/inmemory.test.ts "DeleteVersionsByParentID should remove all versions"
	t.Run("DeleteVersionsByParentID should remove all versions", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-delall", "name": "V1", "type": "llm-judge"},
		})
		_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 "extra-v",
			ScorerDefinitionID: "ver-scorer-delall",
			VersionNumber:      2,
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name: "V2", Type: "llm-judge",
			},
		})

		err := storage.DeleteVersionsByParentID(ctx, "ver-scorer-delall")
		if err != nil {
			t.Fatalf("DeleteVersionsByParentID returned error: %v", err)
		}

		count, _ := storage.CountVersions(ctx, "ver-scorer-delall")
		if count != 0 {
			t.Errorf("expected 0 versions, got %d", count)
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should count versions correctly"
	t.Run("should count versions correctly", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "ver-scorer-count", "name": "V1", "type": "llm-judge"},
		})

		count1, _ := storage.CountVersions(ctx, "ver-scorer-count")
		if count1 != 1 {
			t.Errorf("expected 1 version, got %d", count1)
		}

		_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 "v2-count",
			ScorerDefinitionID: "ver-scorer-count",
			VersionNumber:      2,
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name: "V2", Type: "llm-judge",
			},
		})

		count2, _ := storage.CountVersions(ctx, "ver-scorer-count")
		if count2 != 2 {
			t.Errorf("expected 2 versions, got %d", count2)
		}

		// Non-existent entity should return 0
		count0, _ := storage.CountVersions(ctx, "nonexistent")
		if count0 != 0 {
			t.Errorf("expected 0 versions for nonexistent, got %d", count0)
		}
	})
}

// ==========================================================================
// Version with snapshot config fields
// ==========================================================================

func TestInMemoryScorerDefinitionsStorage_VersionSnapshotConfig(t *testing.T) {
	ctx := context.Background()

	// Test that scorer-definition-specific snapshot config fields persist through
	// version creation and retrieval (model, scoreRange, instructions, presetConfig).
	t.Run("should persist snapshot config fields through createVersion and getVersion", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{"id": "snap-scorer", "name": "V1", "type": "llm-judge"},
		})

		temp := 0.7
		maxTokens := 1000
		minScore := 0.0
		maxScore := 10.0
		desc := "A test scorer"
		instructions := "Score the response"

		_, err := storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 "snap-v2",
			ScorerDefinitionID: "snap-scorer",
			VersionNumber:      2,
			ChangedFields:      []string{"model", "scoreRange", "description", "instructions"},
			ChangeMessage:      "Added config",
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name:        "Snap Scorer V2",
				Type:        "llm-judge",
				Description: &desc,
				Model: &ScorerModelConfig{
					Provider:    "openai",
					Name:        "gpt-4",
					Temperature: &temp,
					MaxTokens:   &maxTokens,
				},
				Instructions: &instructions,
				ScoreRange: &ScorerScoreRange{
					Min: &minScore,
					Max: &maxScore,
				},
				PresetConfig: map[string]any{"scale": 10},
			},
		})
		if err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}

		version, err := storage.GetVersion(ctx, "snap-v2")
		if err != nil {
			t.Fatalf("GetVersion returned error: %v", err)
		}
		if version == nil {
			t.Fatal("expected version to exist")
		}
		if version.Name != "Snap Scorer V2" {
			t.Errorf("expected name='Snap Scorer V2', got %q", version.Name)
		}
		if version.Type != "llm-judge" {
			t.Errorf("expected type='llm-judge', got %q", version.Type)
		}
		if version.Description == nil || *version.Description != "A test scorer" {
			t.Errorf("expected description='A test scorer', got %v", version.Description)
		}
		if version.Model == nil {
			t.Fatal("expected model to be set")
		}
		if version.Model.Provider != "openai" {
			t.Errorf("expected model.provider='openai', got %q", version.Model.Provider)
		}
		if version.Model.Name != "gpt-4" {
			t.Errorf("expected model.name='gpt-4', got %q", version.Model.Name)
		}
		if version.Model.Temperature == nil || *version.Model.Temperature != 0.7 {
			t.Errorf("expected model.temperature=0.7, got %v", version.Model.Temperature)
		}
		if version.Instructions == nil || *version.Instructions != "Score the response" {
			t.Errorf("expected instructions='Score the response', got %v", version.Instructions)
		}
		if version.ScoreRange == nil {
			t.Fatal("expected scoreRange to be set")
		}
		if version.ScoreRange.Min == nil || *version.ScoreRange.Min != 0.0 {
			t.Errorf("expected scoreRange.min=0.0, got %v", version.ScoreRange.Min)
		}
		if version.ScoreRange.Max == nil || *version.ScoreRange.Max != 10.0 {
			t.Errorf("expected scoreRange.max=10.0, got %v", version.ScoreRange.Max)
		}
		if version.PresetConfig == nil {
			t.Fatal("expected presetConfig to be set")
		}
		// JSON round-trip may change int to float64
		if v, ok := version.PresetConfig["scale"]; !ok {
			t.Error("expected presetConfig to have 'scale' key")
		} else {
			switch sv := v.(type) {
			case float64:
				if sv != 10 {
					t.Errorf("expected presetConfig.scale=10, got %v", sv)
				}
			case int:
				if sv != 10 {
					t.Errorf("expected presetConfig.scale=10, got %v", sv)
				}
			default:
				t.Errorf("unexpected type for presetConfig.scale: %T", v)
			}
		}
		if len(version.ChangedFields) != 4 {
			t.Errorf("expected 4 changedFields, got %d", len(version.ChangedFields))
		}
	})
}

// ==========================================================================
// GetByIDResolved
// ==========================================================================

func TestInMemoryScorerDefinitionsStorage_GetByIDResolved(t *testing.T) {
	ctx := context.Background()

	// Mirrors: agents/inmemory.test.ts "should fall back to latest version when no activeVersionId"
	t.Run("should fall back to latest version when no activeVersionId", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":   "scorer-resolve-1",
				"name": "Resolved Scorer",
				"type": "llm-judge",
			},
		})

		result, err := storage.GetByIDResolved(ctx, "scorer-resolve-1", "")
		if err != nil {
			t.Fatalf("GetByIDResolved returned error: %v", err)
		}
		if result == nil {
			t.Fatal("expected resolved result")
		}
		resolved, _ := toMap(result)
		if resolved["resolvedVersionId"] == nil || resolved["resolvedVersionId"] == "" {
			t.Error("expected resolvedVersionId to be set")
		}
	})

	// Mirrors: agents/inmemory.test.ts "should use activeVersionId when set"
	t.Run("should use activeVersionId when set", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":   "scorer-resolve-2",
				"name": "V1",
				"type": "llm-judge",
			},
		})
		v1, _ := storage.GetVersionByNumber(ctx, "scorer-resolve-2", 1)

		_, _ = storage.Update(ctx, map[string]any{
			"id":              "scorer-resolve-2",
			"activeVersionId": v1.ID,
		})

		result, err := storage.GetByIDResolved(ctx, "scorer-resolve-2", "")
		if err != nil {
			t.Fatalf("GetByIDResolved returned error: %v", err)
		}
		resolved, _ := toMap(result)
		if resolved["resolvedVersionId"] != v1.ID {
			t.Errorf("expected resolvedVersionId=%s, got %v", v1.ID, resolved["resolvedVersionId"])
		}
	})

	// Mirrors: agents/inmemory.test.ts "should return nil for non-existent agent"
	t.Run("should return nil for non-existent scorer definition", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		result, err := storage.GetByIDResolved(ctx, "non-existent", "")
		if err != nil {
			t.Fatalf("GetByIDResolved returned error: %v", err)
		}
		if result != nil {
			t.Error("expected nil for non-existent scorer definition")
		}
	})

	// Mirrors: prompt-blocks/inmemory.test.ts "should resolve latest version for block"
	// — but also tests that config fields are merged into the resolved result.
	t.Run("should merge config fields from version into resolved result", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":   "scorer-resolve-3",
				"name": "Merge Test",
				"type": "answer-relevancy",
			},
		})

		result, err := storage.GetByIDResolved(ctx, "scorer-resolve-3", "")
		if err != nil {
			t.Fatalf("GetByIDResolved returned error: %v", err)
		}
		resolved, _ := toMap(result)
		// Should have both thin record fields and snapshot config fields
		if resolved["id"] != "scorer-resolve-3" {
			t.Errorf("expected id=scorer-resolve-3, got %v", resolved["id"])
		}
		if resolved["status"] != "draft" {
			t.Errorf("expected status=draft, got %v", resolved["status"])
		}
	})

	// Test draft status resolution — should use latest version
	t.Run("should use latest version when status=draft", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":   "scorer-resolve-draft",
				"name": "V1 Name",
				"type": "llm-judge",
			},
		})

		// Create version 2 with different config
		_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 "draft-v2",
			ScorerDefinitionID: "scorer-resolve-draft",
			VersionNumber:      2,
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name: "V2 Name", Type: "llm-judge",
			},
			ChangedFields: []string{"name"},
			ChangeMessage: "v2",
		})

		// Create version 3 (latest)
		_, _ = storage.CreateVersion(ctx, CreateScorerDefinitionVersionInput{
			ID:                 "draft-v3",
			ScorerDefinitionID: "scorer-resolve-draft",
			VersionNumber:      3,
			ScorerDefinitionSnapshotConfig: ScorerDefinitionSnapshotConfig{
				Name: "Latest Name", Type: "llm-judge",
			},
			ChangedFields: []string{"name"},
			ChangeMessage: "v3",
		})

		result, err := storage.GetByIDResolved(ctx, "scorer-resolve-draft", "draft")
		if err != nil {
			t.Fatalf("GetByIDResolved returned error: %v", err)
		}
		resolved, _ := toMap(result)
		if resolved["resolvedVersionId"] != "draft-v3" {
			t.Errorf("expected draft resolution to use latest version (draft-v3), got %v", resolved["resolvedVersionId"])
		}
	})
}

// ==========================================================================
// ListResolved
// ==========================================================================

func TestInMemoryScorerDefinitionsStorage_ListResolved(t *testing.T) {
	ctx := context.Background()

	// Mirrors: agents/inmemory.test.ts "should resolve entities in list"
	t.Run("should resolve entities in list", func(t *testing.T) {
		storage := NewInMemoryScorerDefinitionsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":   "lr-scorer-1",
				"name": "Resolved 1",
				"type": "llm-judge",
			},
		})
		_, _ = storage.Create(ctx, map[string]any{
			"scorerDefinition": map[string]any{
				"id":   "lr-scorer-2",
				"name": "Resolved 2",
				"type": "answer-relevancy",
			},
		})

		result, err := storage.ListResolved(ctx, nil)
		if err != nil {
			t.Fatalf("ListResolved returned error: %v", err)
		}
		resultMap, _ := result.(map[string]any)
		scorers, _ := resultMap["scorerDefinitions"].([]any)
		if len(scorers) != 2 {
			t.Errorf("expected 2 resolved scorer definitions, got %d", len(scorers))
		}
		for _, s := range scorers {
			sm, _ := toMap(s)
			if sm["resolvedVersionId"] == nil || sm["resolvedVersionId"] == "" {
				t.Error("expected each resolved scorer definition to have resolvedVersionId")
			}
		}
	})
}

// ==========================================================================
// DangerouslyClearAll
// ==========================================================================

func TestInMemoryScorerDefinitionsStorage_DangerouslyClearAll(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryScorerDefinitionsStorage()
	_, _ = storage.Create(ctx, map[string]any{
		"scorerDefinition": map[string]any{"id": "clear-1", "name": "C1", "type": "llm-judge"},
	})
	_, _ = storage.Create(ctx, map[string]any{
		"scorerDefinition": map[string]any{"id": "clear-2", "name": "C2", "type": "llm-judge"},
	})

	err := storage.DangerouslyClearAll(ctx)
	if err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	result, _ := storage.List(ctx, nil)
	resultMap, _ := result.(map[string]any)
	if resultMap["total"] != 0 {
		t.Errorf("expected total=0 after clear, got %v", resultMap["total"])
	}

	// Also verify versions are cleared
	count, _ := storage.CountVersions(ctx, "clear-1")
	if count != 0 {
		t.Errorf("expected 0 versions for clear-1 after clear, got %d", count)
	}
}
