// Ported from: packages/core/src/storage/domains/mcp-clients/inmemory.test.ts
package mcpclients

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestInMemoryMCPClientsStorage_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("should create a client with status=draft", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		result, err := storage.Create(ctx, map[string]any{
			"id":       "client-1",
			"authorId": "user-123",
			"metadata": map[string]any{"env": "test"},
			"name":     "Test Client",
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		client, ok := toMap(result)
		if !ok {
			t.Fatal("expected result to be convertible to map")
		}
		if client["id"] != "client-1" {
			t.Errorf("expected id=client-1, got %v", client["id"])
		}
		if client["status"] != "draft" {
			t.Errorf("expected status=draft, got %v", client["status"])
		}
	})

	t.Run("should auto-create version 1", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "client-2", "name": "Client 2"})

		count, _ := storage.CountVersions(ctx, "client-2")
		if count != 1 {
			t.Errorf("expected 1 version, got %d", count)
		}

		v, _ := storage.GetVersionByNumber(ctx, "client-2", 1)
		if v == nil {
			t.Fatal("expected version 1 to exist")
		}
		if v.ChangeMessage != "Initial version" {
			t.Errorf("expected changeMessage='Initial version', got %q", v.ChangeMessage)
		}
	})

	t.Run("should reject duplicate id", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "dup-client"})
		_, err := storage.Create(ctx, map[string]any{"id": "dup-client"})
		if err == nil {
			t.Fatal("expected error for duplicate id")
		}
	})
}

func TestInMemoryMCPClientsStorage_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("should return deep copy (mutation safety)", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"id":       "copy-client",
			"metadata": map[string]any{"key": "original"},
		})

		r1, _ := storage.GetByID(ctx, "copy-client")
		m1, _ := toMap(r1)
		meta1, _ := m1["metadata"].(map[string]any)
		meta1["key"] = "mutated"

		r2, _ := storage.GetByID(ctx, "copy-client")
		m2, _ := toMap(r2)
		meta2, _ := m2["metadata"].(map[string]any)
		if meta2["key"] != "original" {
			t.Error("expected stored data to be unaffected by external mutation")
		}
	})
}

func TestInMemoryMCPClientsStorage_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("should merge metadata", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"id":       "update-client-1",
			"metadata": map[string]any{"key1": "val1"},
		})

		result, err := storage.Update(ctx, map[string]any{
			"id":       "update-client-1",
			"metadata": map[string]any{"key2": "val2"},
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
		client, _ := toMap(result)
		meta, _ := client["metadata"].(map[string]any)
		if meta["key1"] != "val1" {
			t.Error("expected key1=val1 preserved")
		}
		if meta["key2"] != "val2" {
			t.Error("expected key2=val2 added")
		}
	})

	t.Run("should not create a new version on update", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "update-client-2"})
		_, _ = storage.Update(ctx, map[string]any{
			"id":     "update-client-2",
			"status": "published",
		})

		count, _ := storage.CountVersions(ctx, "update-client-2")
		if count != 1 {
			t.Errorf("expected 1 version after update, got %d", count)
		}
	})
}

func TestInMemoryMCPClientsStorage_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("should cascade delete versions", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "del-client"})

		err := storage.Delete(ctx, "del-client")
		if err != nil {
			t.Fatalf("Delete returned error: %v", err)
		}

		r, _ := storage.GetByID(ctx, "del-client")
		if r != nil {
			t.Error("expected client to be deleted")
		}
		count, _ := storage.CountVersions(ctx, "del-client")
		if count != 0 {
			t.Errorf("expected 0 versions, got %d", count)
		}
	})

	t.Run("should be idempotent", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		err := storage.Delete(ctx, "non-existent")
		if err != nil {
			t.Fatalf("Delete of non-existent returned error: %v", err)
		}
	})
}

func TestInMemoryMCPClientsStorage_List(t *testing.T) {
	ctx := context.Background()

	t.Run("should paginate", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		for i := 0; i < 5; i++ {
			_, _ = storage.Create(ctx, map[string]any{
				"id": fmt.Sprintf("list-client-%d", i),
			})
			time.Sleep(time.Millisecond)
		}

		result, err := storage.List(ctx, map[string]any{"page": 0, "perPage": 2})
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		rm, _ := result.(map[string]any)
		clients, _ := rm["mcpClients"].([]any)
		if len(clients) != 2 {
			t.Errorf("expected 2 clients, got %d", len(clients))
		}
		if rm["total"] != 5 {
			t.Errorf("expected total=5, got %v", rm["total"])
		}
		if rm["hasMore"] != true {
			t.Error("expected hasMore=true")
		}
	})

	t.Run("should filter by authorId", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "a1", "authorId": "user-A"})
		_, _ = storage.Create(ctx, map[string]any{"id": "a2", "authorId": "user-B"})
		_, _ = storage.Create(ctx, map[string]any{"id": "a3", "authorId": "user-A"})

		result, _ := storage.List(ctx, map[string]any{"authorId": "user-A"})
		rm, _ := result.(map[string]any)
		clients, _ := rm["mcpClients"].([]any)
		if len(clients) != 2 {
			t.Errorf("expected 2 clients for user-A, got %d", len(clients))
		}
	})

	t.Run("should filter by metadata", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"id":       "m1",
			"metadata": map[string]any{"env": "prod"},
		})
		_, _ = storage.Create(ctx, map[string]any{
			"id":       "m2",
			"metadata": map[string]any{"env": "staging"},
		})

		result, _ := storage.List(ctx, map[string]any{
			"metadata": map[string]any{"env": "prod"},
		})
		rm, _ := result.(map[string]any)
		clients, _ := rm["mcpClients"].([]any)
		if len(clients) != 1 {
			t.Errorf("expected 1 client matching metadata, got %d", len(clients))
		}
	})
}

func TestInMemoryMCPClientsStorage_Versions(t *testing.T) {
	ctx := context.Background()

	t.Run("should create and retrieve version", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-client-1"})

		v, err := storage.CreateVersion(ctx, CreateMCPClientVersionInput{
			ID:            "v2-id",
			MCPClientID:   "ver-client-1",
			VersionNumber: 2,
			ChangeMessage: "Second version",
			Snapshot:      map[string]any{"name": "Updated"},
		})
		if err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
		if v.VersionNumber != 2 {
			t.Errorf("expected versionNumber=2, got %d", v.VersionNumber)
		}

		retrieved, _ := storage.GetVersion(ctx, "v2-id")
		if retrieved == nil {
			t.Fatal("expected version to exist")
		}
	})

	t.Run("should reject duplicate version number", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-client-dup"})
		_, err := storage.CreateVersion(ctx, CreateMCPClientVersionInput{
			ID:            "dup-v",
			MCPClientID:   "ver-client-dup",
			VersionNumber: 1,
		})
		if err == nil {
			t.Fatal("expected error for duplicate version number")
		}
	})

	t.Run("GetLatestVersion should return highest", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-client-latest"})
		_, _ = storage.CreateVersion(ctx, CreateMCPClientVersionInput{
			ID: "v2", MCPClientID: "ver-client-latest", VersionNumber: 2,
		})
		_, _ = storage.CreateVersion(ctx, CreateMCPClientVersionInput{
			ID: "v3", MCPClientID: "ver-client-latest", VersionNumber: 3,
		})

		latest, _ := storage.GetLatestVersion(ctx, "ver-client-latest")
		if latest == nil || latest.VersionNumber != 3 {
			t.Errorf("expected versionNumber=3, got %v", latest)
		}
	})

	t.Run("ListVersions should paginate", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-client-list"})
		for i := 2; i <= 5; i++ {
			_, _ = storage.CreateVersion(ctx, CreateMCPClientVersionInput{
				ID: fmt.Sprintf("lv-%d", i), MCPClientID: "ver-client-list", VersionNumber: i,
			})
		}

		perPage := 2
		result, _ := storage.ListVersions(ctx, ListMCPClientVersionsInput{
			MCPClientID: "ver-client-list",
			PerPage:     &perPage,
		})
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

	t.Run("CountVersions", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-client-count"})
		_, _ = storage.CreateVersion(ctx, CreateMCPClientVersionInput{
			ID: "v2c", MCPClientID: "ver-client-count", VersionNumber: 2,
		})
		count, _ := storage.CountVersions(ctx, "ver-client-count")
		if count != 2 {
			t.Errorf("expected 2 versions, got %d", count)
		}
	})
}

func TestInMemoryMCPClientsStorage_GetByIDResolved(t *testing.T) {
	ctx := context.Background()

	t.Run("should resolve with latest version", func(t *testing.T) {
		storage := NewInMemoryMCPClientsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "resolve-1", "name": "Test"})

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

func TestInMemoryMCPClientsStorage_DangerouslyClearAll(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryMCPClientsStorage()
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
