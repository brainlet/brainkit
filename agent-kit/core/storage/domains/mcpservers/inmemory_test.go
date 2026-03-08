// Ported from: packages/core/src/storage/domains/mcp-servers/inmemory.test.ts
package mcpservers

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestInMemoryMCPServersStorage_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("should create a server with status=draft", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		result, err := storage.Create(ctx, map[string]any{
			"id":       "server-1",
			"authorId": "user-123",
			"metadata": map[string]any{"env": "test"},
			"version":  "1.0.0",
			"tools":    map[string]any{"tool1": true},
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		server, ok := toMap(result)
		if !ok {
			t.Fatal("expected result to be convertible to map")
		}
		if server["id"] != "server-1" {
			t.Errorf("expected id=server-1, got %v", server["id"])
		}
		if server["status"] != "draft" {
			t.Errorf("expected status=draft, got %v", server["status"])
		}
	})

	t.Run("should auto-create version 1", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "server-2", "version": "1.0.0"})

		count, _ := storage.CountVersions(ctx, "server-2")
		if count != 1 {
			t.Errorf("expected 1 version, got %d", count)
		}

		v, _ := storage.GetVersionByNumber(ctx, "server-2", 1)
		if v == nil {
			t.Fatal("expected version 1 to exist")
		}
		if v.ChangeMessage != "Initial version" {
			t.Errorf("expected changeMessage='Initial version', got %q", v.ChangeMessage)
		}
	})

	t.Run("should reject duplicate id", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "dup-server"})
		_, err := storage.Create(ctx, map[string]any{"id": "dup-server"})
		if err == nil {
			t.Fatal("expected error for duplicate id")
		}
	})
}

func TestInMemoryMCPServersStorage_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("should return deep copy (mutation safety)", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"id":       "copy-server",
			"metadata": map[string]any{"key": "original"},
		})

		r1, _ := storage.GetByID(ctx, "copy-server")
		m1, _ := toMap(r1)
		meta1, _ := m1["metadata"].(map[string]any)
		meta1["key"] = "mutated"

		r2, _ := storage.GetByID(ctx, "copy-server")
		m2, _ := toMap(r2)
		meta2, _ := m2["metadata"].(map[string]any)
		if meta2["key"] != "original" {
			t.Error("expected stored data to be unaffected by external mutation")
		}
	})
}

func TestInMemoryMCPServersStorage_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("should merge metadata", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"id":       "update-server-1",
			"metadata": map[string]any{"key1": "val1"},
		})

		result, err := storage.Update(ctx, map[string]any{
			"id":       "update-server-1",
			"metadata": map[string]any{"key2": "val2"},
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
		server, _ := toMap(result)
		meta, _ := server["metadata"].(map[string]any)
		if meta["key1"] != "val1" {
			t.Error("expected key1=val1 preserved")
		}
		if meta["key2"] != "val2" {
			t.Error("expected key2=val2 added")
		}
	})

	t.Run("should not create a new version on update", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "update-server-2"})
		_, _ = storage.Update(ctx, map[string]any{
			"id":     "update-server-2",
			"status": "published",
		})
		count, _ := storage.CountVersions(ctx, "update-server-2")
		if count != 1 {
			t.Errorf("expected 1 version after update, got %d", count)
		}
	})
}

func TestInMemoryMCPServersStorage_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("should cascade delete versions", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "del-server"})

		err := storage.Delete(ctx, "del-server")
		if err != nil {
			t.Fatalf("Delete returned error: %v", err)
		}

		r, _ := storage.GetByID(ctx, "del-server")
		if r != nil {
			t.Error("expected server to be deleted")
		}
		count, _ := storage.CountVersions(ctx, "del-server")
		if count != 0 {
			t.Errorf("expected 0 versions, got %d", count)
		}
	})
}

func TestInMemoryMCPServersStorage_List(t *testing.T) {
	ctx := context.Background()

	t.Run("should paginate", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		// Note: list default-filters to status=published, so we must publish these.
		for i := 0; i < 5; i++ {
			_, _ = storage.Create(ctx, map[string]any{
				"id": fmt.Sprintf("list-server-%d", i),
			})
			_, _ = storage.Update(ctx, map[string]any{
				"id":     fmt.Sprintf("list-server-%d", i),
				"status": "published",
			})
			time.Sleep(time.Millisecond)
		}

		result, err := storage.List(ctx, map[string]any{"page": 0, "perPage": 2})
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		rm, _ := result.(map[string]any)
		servers, _ := rm["mcpServers"].([]any)
		if len(servers) != 2 {
			t.Errorf("expected 2 servers, got %d", len(servers))
		}
		if rm["total"] != 5 {
			t.Errorf("expected total=5, got %v", rm["total"])
		}
		if rm["hasMore"] != true {
			t.Error("expected hasMore=true")
		}
	})

	t.Run("should filter by status=draft", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "s1"}) // draft
		_, _ = storage.Create(ctx, map[string]any{"id": "s2"}) // draft
		_, _ = storage.Create(ctx, map[string]any{"id": "s3"})
		_, _ = storage.Update(ctx, map[string]any{"id": "s3", "status": "published"})

		result, _ := storage.List(ctx, map[string]any{"status": "draft"})
		rm, _ := result.(map[string]any)
		servers, _ := rm["mcpServers"].([]any)
		if len(servers) != 2 {
			t.Errorf("expected 2 draft servers, got %d", len(servers))
		}
	})
}

func TestInMemoryMCPServersStorage_Versions(t *testing.T) {
	ctx := context.Background()

	t.Run("should create and retrieve version", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-server-1"})

		v, err := storage.CreateVersion(ctx, CreateMCPServerVersionInput{
			ID:            "v2-id",
			MCPServerID:   "ver-server-1",
			VersionNumber: 2,
			ChangeMessage: "Second version",
			Snapshot:      map[string]any{"version": "2.0.0"},
		})
		if err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
		if v.VersionNumber != 2 {
			t.Errorf("expected versionNumber=2, got %d", v.VersionNumber)
		}
	})

	t.Run("GetLatestVersion should return highest", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-server-latest"})
		_, _ = storage.CreateVersion(ctx, CreateMCPServerVersionInput{
			ID: "v2", MCPServerID: "ver-server-latest", VersionNumber: 2,
		})
		_, _ = storage.CreateVersion(ctx, CreateMCPServerVersionInput{
			ID: "v3", MCPServerID: "ver-server-latest", VersionNumber: 3,
		})

		latest, _ := storage.GetLatestVersion(ctx, "ver-server-latest")
		if latest == nil || latest.VersionNumber != 3 {
			t.Errorf("expected versionNumber=3")
		}
	})

	t.Run("ListVersions should paginate", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-server-list"})
		for i := 2; i <= 5; i++ {
			_, _ = storage.CreateVersion(ctx, CreateMCPServerVersionInput{
				ID: fmt.Sprintf("lv-%d", i), MCPServerID: "ver-server-list", VersionNumber: i,
			})
		}

		perPage := 2
		result, _ := storage.ListVersions(ctx, ListMCPServerVersionsInput{
			MCPServerID: "ver-server-list",
			PerPage:     &perPage,
		})
		if len(result.Versions) != 2 {
			t.Errorf("expected 2 versions, got %d", len(result.Versions))
		}
		if result.Total != 5 {
			t.Errorf("expected total=5, got %d", result.Total)
		}
	})
}

func TestInMemoryMCPServersStorage_GetByIDResolved(t *testing.T) {
	ctx := context.Background()

	t.Run("should resolve with latest version", func(t *testing.T) {
		storage := NewInMemoryMCPServersStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "resolve-1", "version": "1.0.0"})

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

func TestInMemoryMCPServersStorage_DangerouslyClearAll(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryMCPServersStorage()
	_, _ = storage.Create(ctx, map[string]any{"id": "c1"})
	_, _ = storage.Create(ctx, map[string]any{"id": "c2"})

	err := storage.DangerouslyClearAll(ctx)
	if err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	// List with status=draft since defaults are draft
	result, _ := storage.List(ctx, map[string]any{"status": "draft"})
	rm, _ := result.(map[string]any)
	if rm["total"] != 0 {
		t.Errorf("expected total=0, got %v", rm["total"])
	}
}
