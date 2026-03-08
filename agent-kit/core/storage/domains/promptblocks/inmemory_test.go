// Ported from: packages/core/src/storage/domains/prompt-blocks/inmemory.test.ts
package promptblocks

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestInMemoryPromptBlocksStorage_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("should create a block with status=draft", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		result, err := storage.Create(ctx, map[string]any{
			"id":       "block-1",
			"authorId": "user-123",
			"metadata": map[string]any{"env": "test"},
			"content":  "Hello {{name}}",
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		block, ok := toMap(result)
		if !ok {
			t.Fatal("expected result to be convertible to map")
		}
		if block["id"] != "block-1" {
			t.Errorf("expected id=block-1, got %v", block["id"])
		}
		if block["status"] != "draft" {
			t.Errorf("expected status=draft, got %v", block["status"])
		}
	})

	t.Run("should auto-create version 1", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "block-2", "content": "Test"})

		count, _ := storage.CountVersions(ctx, "block-2")
		if count != 1 {
			t.Errorf("expected 1 version, got %d", count)
		}

		v, _ := storage.GetVersionByNumber(ctx, "block-2", 1)
		if v == nil {
			t.Fatal("expected version 1 to exist")
		}
		if v.ChangeMessage != "Initial version" {
			t.Errorf("expected changeMessage='Initial version', got %q", v.ChangeMessage)
		}
	})

	t.Run("should reject duplicate id", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "dup-block"})
		_, err := storage.Create(ctx, map[string]any{"id": "dup-block"})
		if err == nil {
			t.Fatal("expected error for duplicate id")
		}
	})
}

func TestInMemoryPromptBlocksStorage_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("should return nil for non-existent", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		r, err := storage.GetByID(ctx, "non-existent")
		if err != nil {
			t.Fatalf("GetByID returned error: %v", err)
		}
		if r != nil {
			t.Error("expected nil")
		}
	})
}

func TestInMemoryPromptBlocksStorage_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("should merge metadata", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"id":       "update-block-1",
			"metadata": map[string]any{"key1": "val1"},
		})

		result, err := storage.Update(ctx, map[string]any{
			"id":       "update-block-1",
			"metadata": map[string]any{"key2": "val2"},
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
		block, _ := toMap(result)
		meta, _ := block["metadata"].(map[string]any)
		if meta["key1"] != "val1" {
			t.Error("expected key1=val1 preserved")
		}
		if meta["key2"] != "val2" {
			t.Error("expected key2=val2 added")
		}
	})

	t.Run("should not create a new version on update", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "update-block-2"})
		_, _ = storage.Update(ctx, map[string]any{
			"id":     "update-block-2",
			"status": "published",
		})
		count, _ := storage.CountVersions(ctx, "update-block-2")
		if count != 1 {
			t.Errorf("expected 1 version after update, got %d", count)
		}
	})
}

func TestInMemoryPromptBlocksStorage_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("should cascade delete versions", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "del-block"})

		err := storage.Delete(ctx, "del-block")
		if err != nil {
			t.Fatalf("Delete returned error: %v", err)
		}

		r, _ := storage.GetByID(ctx, "del-block")
		if r != nil {
			t.Error("expected block to be deleted")
		}
		count, _ := storage.CountVersions(ctx, "del-block")
		if count != 0 {
			t.Errorf("expected 0 versions, got %d", count)
		}
	})
}

func TestInMemoryPromptBlocksStorage_List(t *testing.T) {
	ctx := context.Background()

	t.Run("should paginate", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		for i := 0; i < 5; i++ {
			_, _ = storage.Create(ctx, map[string]any{
				"id": fmt.Sprintf("list-block-%d", i),
			})
			time.Sleep(time.Millisecond)
		}

		result, err := storage.List(ctx, map[string]any{"page": 0, "perPage": 2})
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		rm, _ := result.(map[string]any)
		blocks, _ := rm["promptBlocks"].([]any)
		if len(blocks) != 2 {
			t.Errorf("expected 2 blocks, got %d", len(blocks))
		}
		if rm["total"] != 5 {
			t.Errorf("expected total=5, got %v", rm["total"])
		}
		if rm["hasMore"] != true {
			t.Error("expected hasMore=true")
		}
	})

	t.Run("should filter by authorId", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "a1", "authorId": "user-A"})
		_, _ = storage.Create(ctx, map[string]any{"id": "a2", "authorId": "user-B"})
		_, _ = storage.Create(ctx, map[string]any{"id": "a3", "authorId": "user-A"})

		result, _ := storage.List(ctx, map[string]any{"authorId": "user-A"})
		rm, _ := result.(map[string]any)
		blocks, _ := rm["promptBlocks"].([]any)
		if len(blocks) != 2 {
			t.Errorf("expected 2 blocks for user-A, got %d", len(blocks))
		}
	})
}

func TestInMemoryPromptBlocksStorage_Versions(t *testing.T) {
	ctx := context.Background()

	t.Run("should create and retrieve version", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-block-1"})

		v, err := storage.CreateVersion(ctx, CreatePromptBlockVersionInput{
			ID:            "v2-id",
			BlockID:       "ver-block-1",
			VersionNumber: 2,
			ChangeMessage: "Second version",
			Snapshot:      map[string]any{"content": "Updated content"},
		})
		if err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
		if v.VersionNumber != 2 {
			t.Errorf("expected versionNumber=2, got %d", v.VersionNumber)
		}
	})

	t.Run("should reject duplicate version number", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-block-dup"})
		_, err := storage.CreateVersion(ctx, CreatePromptBlockVersionInput{
			ID:            "dup-v",
			BlockID:       "ver-block-dup",
			VersionNumber: 1,
		})
		if err == nil {
			t.Fatal("expected error for duplicate version number")
		}
	})

	t.Run("GetLatestVersion should return highest", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-block-latest"})
		_, _ = storage.CreateVersion(ctx, CreatePromptBlockVersionInput{
			ID: "v2", BlockID: "ver-block-latest", VersionNumber: 2,
		})
		_, _ = storage.CreateVersion(ctx, CreatePromptBlockVersionInput{
			ID: "v3", BlockID: "ver-block-latest", VersionNumber: 3,
		})

		latest, _ := storage.GetLatestVersion(ctx, "ver-block-latest")
		if latest == nil || latest.VersionNumber != 3 {
			t.Errorf("expected versionNumber=3")
		}
	})

	t.Run("ListVersions should paginate", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-block-list"})
		for i := 2; i <= 5; i++ {
			_, _ = storage.CreateVersion(ctx, CreatePromptBlockVersionInput{
				ID: fmt.Sprintf("lv-%d", i), BlockID: "ver-block-list", VersionNumber: i,
			})
		}

		perPage := 2
		result, _ := storage.ListVersions(ctx, ListPromptBlockVersionsInput{
			BlockID: "ver-block-list",
			PerPage: &perPage,
		})
		if len(result.Versions) != 2 {
			t.Errorf("expected 2 versions, got %d", len(result.Versions))
		}
		if result.Total != 5 {
			t.Errorf("expected total=5, got %d", result.Total)
		}
	})
}

func TestInMemoryPromptBlocksStorage_GetByIDResolved(t *testing.T) {
	ctx := context.Background()

	t.Run("should resolve with latest version", func(t *testing.T) {
		storage := NewInMemoryPromptBlocksStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "resolve-1", "content": "Test"})

		result, err := storage.GetByIDResolved(ctx, "resolve-1", "")
		if err != nil {
			t.Fatalf("GetByIDResolved returned error: %v", err)
		}
		rm, _ := toMap(result)
		if rm["resolvedVersionId"] == nil || rm["resolvedVersionId"] == "" {
			t.Error("expected resolvedVersionId to be set")
		}
	})
}

func TestInMemoryPromptBlocksStorage_DangerouslyClearAll(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryPromptBlocksStorage()
	_, _ = storage.Create(ctx, map[string]any{"id": "c1"})
	_, _ = storage.Create(ctx, map[string]any{"id": "c2"})

	err := storage.DangerouslyClearAll(ctx)
	if err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	result, _ := storage.List(ctx, nil)
	rm, _ := result.(map[string]any)
	if rm["total"] != 0 {
		t.Errorf("expected total=0, got %v", rm["total"])
	}
}
