// Ported from: packages/core/src/storage/domains/agents/inmemory.test.ts
package agents

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestInMemoryAgentsStorage_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("should create an agent with status=draft", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		result, err := storage.Create(ctx, map[string]any{
			"id":       "test-agent-1",
			"authorId": "user-123",
			"metadata": map[string]any{"env": "test"},
			"name":     "Test Agent",
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		agent, ok := toMap(result)
		if !ok {
			t.Fatal("expected result to be convertible to map")
		}
		if agent["id"] != "test-agent-1" {
			t.Errorf("expected id=test-agent-1, got %v", agent["id"])
		}
		if agent["status"] != "draft" {
			t.Errorf("expected status=draft, got %v", agent["status"])
		}
		if agent["authorId"] != "user-123" {
			t.Errorf("expected authorId=user-123, got %v", agent["authorId"])
		}
	})

	t.Run("should auto-create version 1 on create", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, err := storage.Create(ctx, map[string]any{
			"id":   "test-agent-2",
			"name": "Agent With Version",
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}

		count, err := storage.CountVersions(ctx, "test-agent-2")
		if err != nil {
			t.Fatalf("CountVersions returned error: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 version, got %d", count)
		}

		v, err := storage.GetVersionByNumber(ctx, "test-agent-2", 1)
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

	t.Run("should reject duplicate id", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, err := storage.Create(ctx, map[string]any{"id": "dup-agent"})
		if err != nil {
			t.Fatalf("first Create returned error: %v", err)
		}
		_, err = storage.Create(ctx, map[string]any{"id": "dup-agent"})
		if err == nil {
			t.Fatal("expected error for duplicate id")
		}
	})
}

func TestInMemoryAgentsStorage_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("should merge metadata", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, err := storage.Create(ctx, map[string]any{
			"id":       "agent-update-1",
			"metadata": map[string]any{"key1": "val1"},
		})
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}

		result, err := storage.Update(ctx, map[string]any{
			"id":       "agent-update-1",
			"metadata": map[string]any{"key2": "val2"},
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
		agent, _ := toMap(result)
		meta, _ := agent["metadata"].(map[string]any)
		if meta["key1"] != "val1" {
			t.Error("expected key1=val1 preserved")
		}
		if meta["key2"] != "val2" {
			t.Error("expected key2=val2 added")
		}
	})

	t.Run("should not create a new version on update", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "agent-update-2"})

		_, err := storage.Update(ctx, map[string]any{
			"id":     "agent-update-2",
			"status": "published",
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}

		count, _ := storage.CountVersions(ctx, "agent-update-2")
		if count != 1 {
			t.Errorf("expected 1 version after update, got %d", count)
		}
	})

	t.Run("should update activeVersionId", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "agent-update-3"})

		result, err := storage.Update(ctx, map[string]any{
			"id":              "agent-update-3",
			"activeVersionId": "some-version-id",
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
		agent, _ := toMap(result)
		if agent["activeVersionId"] != "some-version-id" {
			t.Errorf("expected activeVersionId=some-version-id, got %v", agent["activeVersionId"])
		}
	})
}

func TestInMemoryAgentsStorage_GetByID(t *testing.T) {
	ctx := context.Background()

	t.Run("should return nil for non-existent agent", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		result, err := storage.GetByID(ctx, "non-existent")
		if err != nil {
			t.Fatalf("GetByID returned error: %v", err)
		}
		if result != nil {
			t.Error("expected nil for non-existent agent")
		}
	})

	t.Run("should return deep copy (mutation safety)", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"id":       "agent-copy-1",
			"metadata": map[string]any{"key": "original"},
		})

		result1, _ := storage.GetByID(ctx, "agent-copy-1")
		m1, _ := toMap(result1)
		meta1, _ := m1["metadata"].(map[string]any)
		meta1["key"] = "mutated"

		result2, _ := storage.GetByID(ctx, "agent-copy-1")
		m2, _ := toMap(result2)
		meta2, _ := m2["metadata"].(map[string]any)
		if meta2["key"] != "original" {
			t.Error("expected stored data to be unaffected by external mutation")
		}
	})
}

func TestInMemoryAgentsStorage_GetByIDResolved(t *testing.T) {
	ctx := context.Background()

	t.Run("should fall back to latest version when no activeVersionId", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"id":   "agent-resolve-1",
			"name": "Resolved Agent",
		})

		result, err := storage.GetByIDResolved(ctx, "agent-resolve-1", "")
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

	t.Run("should use activeVersionId when set", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"id": "agent-resolve-2",
		})
		v1, _ := storage.GetVersionByNumber(ctx, "agent-resolve-2", 1)

		_, _ = storage.Update(ctx, map[string]any{
			"id":              "agent-resolve-2",
			"activeVersionId": v1.ID,
		})

		result, err := storage.GetByIDResolved(ctx, "agent-resolve-2", "")
		if err != nil {
			t.Fatalf("GetByIDResolved returned error: %v", err)
		}
		resolved, _ := toMap(result)
		if resolved["resolvedVersionId"] != v1.ID {
			t.Errorf("expected resolvedVersionId=%s, got %v", v1.ID, resolved["resolvedVersionId"])
		}
	})

	t.Run("should return nil for non-existent agent", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		result, err := storage.GetByIDResolved(ctx, "non-existent", "")
		if err != nil {
			t.Fatalf("GetByIDResolved returned error: %v", err)
		}
		if result != nil {
			t.Error("expected nil for non-existent agent")
		}
	})
}

func TestInMemoryAgentsStorage_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("should delete agent and cascade versions", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "agent-delete-1"})

		err := storage.Delete(ctx, "agent-delete-1")
		if err != nil {
			t.Fatalf("Delete returned error: %v", err)
		}

		result, _ := storage.GetByID(ctx, "agent-delete-1")
		if result != nil {
			t.Error("expected agent to be deleted")
		}

		count, _ := storage.CountVersions(ctx, "agent-delete-1")
		if count != 0 {
			t.Errorf("expected 0 versions after cascade delete, got %d", count)
		}
	})

	t.Run("should be idempotent", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		err := storage.Delete(ctx, "non-existent")
		if err != nil {
			t.Fatalf("Delete of non-existent agent returned error: %v", err)
		}
	})
}

func TestInMemoryAgentsStorage_List(t *testing.T) {
	ctx := context.Background()

	t.Run("should list with pagination", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		for i := 0; i < 5; i++ {
			_, _ = storage.Create(ctx, map[string]any{
				"id": fmt.Sprintf("list-agent-%d", i),
			})
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
		agents, _ := resultMap["agents"].([]any)
		if len(agents) != 2 {
			t.Errorf("expected 2 agents, got %d", len(agents))
		}
		if resultMap["total"] != 5 {
			t.Errorf("expected total=5, got %v", resultMap["total"])
		}
		if resultMap["hasMore"] != true {
			t.Error("expected hasMore=true")
		}
	})

	t.Run("should filter by authorId", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "a1", "authorId": "user-A"})
		_, _ = storage.Create(ctx, map[string]any{"id": "a2", "authorId": "user-B"})
		_, _ = storage.Create(ctx, map[string]any{"id": "a3", "authorId": "user-A"})

		result, _ := storage.List(ctx, map[string]any{"authorId": "user-A"})
		resultMap, _ := result.(map[string]any)
		agents, _ := resultMap["agents"].([]any)
		if len(agents) != 2 {
			t.Errorf("expected 2 agents for user-A, got %d", len(agents))
		}
	})

	t.Run("should filter by metadata", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
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
		resultMap, _ := result.(map[string]any)
		agents, _ := resultMap["agents"].([]any)
		if len(agents) != 1 {
			t.Errorf("expected 1 agent matching metadata, got %d", len(agents))
		}
	})
}

func TestInMemoryAgentsStorage_Versions(t *testing.T) {
	ctx := context.Background()

	t.Run("should create and retrieve a version", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-agent-1"})

		v, err := storage.CreateVersion(ctx, CreateAgentVersionInput{
			ID:            "v2-id",
			AgentID:       "ver-agent-1",
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
	})

	t.Run("should reject duplicate version number", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-agent-dup"})

		_, err := storage.CreateVersion(ctx, CreateAgentVersionInput{
			ID:            "dup-v",
			AgentID:       "ver-agent-dup",
			VersionNumber: 1, // already exists from Create
		})
		if err == nil {
			t.Fatal("expected error for duplicate version number")
		}
	})

	t.Run("GetLatestVersion should return highest version number", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-agent-latest"})
		_, _ = storage.CreateVersion(ctx, CreateAgentVersionInput{
			ID:            "v2",
			AgentID:       "ver-agent-latest",
			VersionNumber: 2,
		})
		_, _ = storage.CreateVersion(ctx, CreateAgentVersionInput{
			ID:            "v3",
			AgentID:       "ver-agent-latest",
			VersionNumber: 3,
		})

		latest, err := storage.GetLatestVersion(ctx, "ver-agent-latest")
		if err != nil {
			t.Fatalf("GetLatestVersion returned error: %v", err)
		}
		if latest == nil {
			t.Fatal("expected latest version to exist")
		}
		if latest.VersionNumber != 3 {
			t.Errorf("expected versionNumber=3, got %d", latest.VersionNumber)
		}
	})

	t.Run("ListVersions should paginate", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-agent-list"})
		for i := 2; i <= 5; i++ {
			_, _ = storage.CreateVersion(ctx, CreateAgentVersionInput{
				ID:            fmt.Sprintf("lv-%d", i),
				AgentID:       "ver-agent-list",
				VersionNumber: i,
			})
		}

		perPage := 2
		result, err := storage.ListVersions(ctx, ListVersionsInput{
			AgentID: "ver-agent-list",
			PerPage: &perPage,
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

	t.Run("DeleteVersion should remove version", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-agent-del"})
		v1, _ := storage.GetVersionByNumber(ctx, "ver-agent-del", 1)

		err := storage.DeleteVersion(ctx, v1.ID)
		if err != nil {
			t.Fatalf("DeleteVersion returned error: %v", err)
		}

		count, _ := storage.CountVersions(ctx, "ver-agent-del")
		if count != 0 {
			t.Errorf("expected 0 versions, got %d", count)
		}
	})

	t.Run("DeleteVersionsByParentID should remove all versions", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{"id": "ver-agent-delall"})
		_, _ = storage.CreateVersion(ctx, CreateAgentVersionInput{
			ID:            "extra-v",
			AgentID:       "ver-agent-delall",
			VersionNumber: 2,
		})

		err := storage.DeleteVersionsByParentID(ctx, "ver-agent-delall")
		if err != nil {
			t.Fatalf("DeleteVersionsByParentID returned error: %v", err)
		}

		count, _ := storage.CountVersions(ctx, "ver-agent-delall")
		if count != 0 {
			t.Errorf("expected 0 versions, got %d", count)
		}
	})
}

func TestInMemoryAgentsStorage_DangerouslyClearAll(t *testing.T) {
	ctx := context.Background()
	storage := NewInMemoryAgentsStorage()
	_, _ = storage.Create(ctx, map[string]any{"id": "clear-1"})
	_, _ = storage.Create(ctx, map[string]any{"id": "clear-2"})

	err := storage.DangerouslyClearAll(ctx)
	if err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	result, _ := storage.List(ctx, nil)
	resultMap, _ := result.(map[string]any)
	if resultMap["total"] != 0 {
		t.Errorf("expected total=0 after clear, got %v", resultMap["total"])
	}
}

func TestInMemoryAgentsStorage_ListResolved(t *testing.T) {
	ctx := context.Background()

	t.Run("should resolve entities in list", func(t *testing.T) {
		storage := NewInMemoryAgentsStorage()
		_, _ = storage.Create(ctx, map[string]any{
			"id":   "lr-agent-1",
			"name": "Resolved 1",
		})
		_, _ = storage.Create(ctx, map[string]any{
			"id":   "lr-agent-2",
			"name": "Resolved 2",
		})

		result, err := storage.ListResolved(ctx, nil)
		if err != nil {
			t.Fatalf("ListResolved returned error: %v", err)
		}
		resultMap, _ := result.(map[string]any)
		agents, _ := resultMap["agents"].([]any)
		if len(agents) != 2 {
			t.Errorf("expected 2 resolved agents, got %d", len(agents))
		}
		for _, a := range agents {
			am, _ := toMap(a)
			if am["resolvedVersionId"] == nil || am["resolvedVersionId"] == "" {
				t.Error("expected each resolved agent to have resolvedVersionId")
			}
		}
	})
}
