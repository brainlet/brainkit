// Ported from: stores/_test-utils/src/domains/workspaces/index.ts (createWorkspacesTest)
// and: packages/core/src/storage/domains/workspaces/inmemory.test.ts
//
// The upstream mastra project tests workspace storage via the shared test
// harness in stores/_test-utils. This Go file faithfully ports those tests
// against InMemoryWorkspacesStorage.
package workspaces

import (
	"context"
	"testing"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// intPtr returns a pointer to an int.
func intPtr(i int) *int { return &i }

// versionOrderByPtr returns a pointer to a WorkspaceVersionOrderBy.
func versionOrderByPtr(v WorkspaceVersionOrderBy) *WorkspaceVersionOrderBy { return &v }

// versionSortDirPtr returns a pointer to a WorkspaceVersionSortDirection.
func versionSortDirPtr(v WorkspaceVersionSortDirection) *WorkspaceVersionSortDirection { return &v }

// makeWorkspaceInput creates a standard workspace creation input map.
func makeWorkspaceInput(id string, extras ...map[string]any) map[string]any {
	input := map[string]any{
		"id":   id,
		"name": "Workspace " + id,
	}
	for _, extra := range extras {
		for k, v := range extra {
			input[k] = v
		}
	}
	return input
}

// ===========================================================================
// Tests — Init
// ===========================================================================

func TestInMemoryWorkspacesStorage_Init(t *testing.T) {
	// Init is a no-op for in-memory; verify it does not error.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	if err := storage.Init(ctx); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
}

// ===========================================================================
// Tests — Create & GetByID
// ===========================================================================

func TestInMemoryWorkspacesStorage_CreateAndGetByID(t *testing.T) {
	// TS: "should create and retrieve a workspace"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := makeWorkspaceInput("ws-1", map[string]any{
		"description": "Test workspace",
		"authorId":    "author-1",
		"metadata":    map[string]any{"key": "value"},
	})

	created, err := storage.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created == nil {
		t.Fatal("expected created workspace, got nil")
	}

	result, err := storage.GetByID(ctx, "ws-1")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected workspace, got nil")
	}

	// Verify the returned entity is a storageWorkspaceType with correct fields.
	ws, ok := result.(storageWorkspaceType)
	if !ok {
		t.Fatalf("expected storageWorkspaceType, got %T", result)
	}
	if ws.ID != "ws-1" {
		t.Errorf("id mismatch: got %s, want ws-1", ws.ID)
	}
	if ws.Status != "draft" {
		t.Errorf("expected initial status=draft, got %s", ws.Status)
	}
	if ws.AuthorID != "author-1" {
		t.Errorf("authorId mismatch: got %s, want author-1", ws.AuthorID)
	}
	if ws.CreatedAt.IsZero() {
		t.Error("expected createdAt to be set")
	}
	if ws.UpdatedAt.IsZero() {
		t.Error("expected updatedAt to be set")
	}
}

func TestInMemoryWorkspacesStorage_Create_MissingID(t *testing.T) {
	// Creating without an id should return an error.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	_, err := storage.Create(ctx, map[string]any{"name": "no-id"})
	if err == nil {
		t.Fatal("expected error for missing id, got nil")
	}
}

func TestInMemoryWorkspacesStorage_Create_DuplicateID(t *testing.T) {
	// Creating a workspace with an existing ID should return an error.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := makeWorkspaceInput("ws-dup")
	if _, err := storage.Create(ctx, input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	_, err := storage.Create(ctx, input)
	if err == nil {
		t.Fatal("expected error for duplicate id, got nil")
	}
}

func TestInMemoryWorkspacesStorage_Create_AutoCreatesVersion(t *testing.T) {
	// Creating a workspace should auto-create version 1.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := makeWorkspaceInput("ws-version", map[string]any{
		"name":        "My Workspace",
		"description": "Initial description",
	})
	if _, err := storage.Create(ctx, input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	count, err := storage.CountVersions(ctx, "ws-version")
	if err != nil {
		t.Fatalf("CountVersions returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 version after create, got %d", count)
	}

	latest, err := storage.GetLatestVersion(ctx, "ws-version")
	if err != nil {
		t.Fatalf("GetLatestVersion returned error: %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest version, got nil")
	}
	if latest.VersionNumber != 1 {
		t.Errorf("expected version number 1, got %d", latest.VersionNumber)
	}
	if latest.ChangeMessage != "Initial version" {
		t.Errorf("expected changeMessage=Initial version, got %s", latest.ChangeMessage)
	}
}

func TestInMemoryWorkspacesStorage_GetByID_NotFound(t *testing.T) {
	// Getting a non-existent workspace should return nil.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	result, err := storage.GetByID(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-existent workspace, got %+v", result)
	}
}

func TestInMemoryWorkspacesStorage_Create_InvalidInputType(t *testing.T) {
	// Creating with an invalid input type should return an error.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	_, err := storage.Create(ctx, "not a map")
	if err == nil {
		t.Fatal("expected error for invalid input type, got nil")
	}
}

// ===========================================================================
// Tests — Update
// ===========================================================================

func TestInMemoryWorkspacesStorage_Update(t *testing.T) {
	// TS: "should update workspace metadata fields"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := makeWorkspaceInput("ws-upd", map[string]any{"authorId": "author-1"})
	if _, err := storage.Create(ctx, input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	updated, err := storage.Update(ctx, map[string]any{
		"id":       "ws-upd",
		"authorId": "author-2",
		"status":   "published",
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated == nil {
		t.Fatal("expected updated workspace, got nil")
	}

	result, _ := storage.GetByID(ctx, "ws-upd")
	ws := result.(storageWorkspaceType)
	if ws.AuthorID != "author-2" {
		t.Errorf("expected authorId=author-2, got %s", ws.AuthorID)
	}
	if ws.Status != "published" {
		t.Errorf("expected status=published, got %s", ws.Status)
	}
}

func TestInMemoryWorkspacesStorage_Update_NotFound(t *testing.T) {
	// Updating a non-existent workspace should return an error.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	_, err := storage.Update(ctx, map[string]any{"id": "nonexistent"})
	if err == nil {
		t.Fatal("expected error for update on non-existent workspace, got nil")
	}
}

func TestInMemoryWorkspacesStorage_Update_MissingID(t *testing.T) {
	// Updating without an ID should return an error.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	_, err := storage.Update(ctx, map[string]any{"name": "no-id"})
	if err == nil {
		t.Fatal("expected error for missing id in update, got nil")
	}
}

func TestInMemoryWorkspacesStorage_Update_MetadataMerge(t *testing.T) {
	// TS: "should merge metadata on update"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := makeWorkspaceInput("ws-meta", map[string]any{
		"metadata": map[string]any{"key1": "val1"},
	})
	if _, err := storage.Create(ctx, input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Update with additional metadata.
	if _, err := storage.Update(ctx, map[string]any{
		"id":       "ws-meta",
		"metadata": map[string]any{"key2": "val2"},
	}); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	result, _ := storage.GetByID(ctx, "ws-meta")
	ws := result.(storageWorkspaceType)

	if ws.Metadata["key1"] != "val1" {
		t.Error("expected original metadata key1=val1 to be preserved")
	}
	if ws.Metadata["key2"] != "val2" {
		t.Error("expected new metadata key2=val2 to be added")
	}
}

func TestInMemoryWorkspacesStorage_Update_ActiveVersionIDSetsPublished(t *testing.T) {
	// TS: "should auto-set status to 'published' when activeVersionId is set"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := makeWorkspaceInput("ws-auto-pub")
	if _, err := storage.Create(ctx, input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Verify initial status is draft.
	result, _ := storage.GetByID(ctx, "ws-auto-pub")
	ws := result.(storageWorkspaceType)
	if ws.Status != "draft" {
		t.Errorf("expected initial status=draft, got %s", ws.Status)
	}

	// Get latest version ID.
	latest, _ := storage.GetLatestVersion(ctx, "ws-auto-pub")

	// Setting activeVersionId without explicit status should set status to "published".
	if _, err := storage.Update(ctx, map[string]any{
		"id":              "ws-auto-pub",
		"activeVersionId": latest.ID,
	}); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	result, _ = storage.GetByID(ctx, "ws-auto-pub")
	ws = result.(storageWorkspaceType)
	if ws.Status != "published" {
		t.Errorf("expected status=published after setting activeVersionId, got %s", ws.Status)
	}
}

func TestInMemoryWorkspacesStorage_Update_ConfigFieldCreatesNewVersion(t *testing.T) {
	// TS: "should auto-create a new version when config fields change"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := makeWorkspaceInput("ws-auto-ver", map[string]any{
		"name":        "Original Name",
		"description": "Original Description",
	})
	if _, err := storage.Create(ctx, input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Version 1 should exist after creation.
	count1, _ := storage.CountVersions(ctx, "ws-auto-ver")
	if count1 != 1 {
		t.Fatalf("expected 1 version after create, got %d", count1)
	}

	// Update with a config field change.
	if _, err := storage.Update(ctx, map[string]any{
		"id":   "ws-auto-ver",
		"name": "Updated Name",
	}); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	// Version 2 should now exist.
	count2, _ := storage.CountVersions(ctx, "ws-auto-ver")
	if count2 != 2 {
		t.Fatalf("expected 2 versions after config update, got %d", count2)
	}

	latest, _ := storage.GetLatestVersion(ctx, "ws-auto-ver")
	if latest.VersionNumber != 2 {
		t.Errorf("expected latest version number=2, got %d", latest.VersionNumber)
	}
}

func TestInMemoryWorkspacesStorage_Update_NonConfigFieldNoNewVersion(t *testing.T) {
	// Updating only non-config fields (e.g., authorId) should NOT create a new version.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := makeWorkspaceInput("ws-no-ver")
	if _, err := storage.Create(ctx, input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	count1, _ := storage.CountVersions(ctx, "ws-no-ver")
	if count1 != 1 {
		t.Fatalf("expected 1 version after create, got %d", count1)
	}

	// Update only metadata fields.
	if _, err := storage.Update(ctx, map[string]any{
		"id":       "ws-no-ver",
		"authorId": "new-author",
	}); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	// Still only 1 version.
	count2, _ := storage.CountVersions(ctx, "ws-no-ver")
	if count2 != 1 {
		t.Fatalf("expected still 1 version after non-config update, got %d", count2)
	}
}

// ===========================================================================
// Tests — Delete
// ===========================================================================

func TestInMemoryWorkspacesStorage_Delete(t *testing.T) {
	// TS: "should delete a workspace and its versions"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := makeWorkspaceInput("ws-del")
	if _, err := storage.Create(ctx, input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Verify it exists.
	result, _ := storage.GetByID(ctx, "ws-del")
	if result == nil {
		t.Fatal("expected workspace to exist before delete")
	}

	// Delete.
	if err := storage.Delete(ctx, "ws-del"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	// Verify it is gone.
	result, _ = storage.GetByID(ctx, "ws-del")
	if result != nil {
		t.Error("expected workspace to be deleted")
	}

	// Versions should also be deleted.
	count, _ := storage.CountVersions(ctx, "ws-del")
	if count != 0 {
		t.Errorf("expected 0 versions after delete, got %d", count)
	}
}

func TestInMemoryWorkspacesStorage_Delete_NonExistent(t *testing.T) {
	// Deleting a non-existent workspace should not error.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	err := storage.Delete(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Delete returned error for non-existent workspace: %v", err)
	}
}

// ===========================================================================
// Tests — List
// ===========================================================================

func TestInMemoryWorkspacesStorage_List_Basic(t *testing.T) {
	// TS: "should list workspaces"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	for i := 0; i < 3; i++ {
		id := "ws-list-" + string(rune('A'+i))
		if _, err := storage.Create(ctx, makeWorkspaceInput(id)); err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
	}

	result, err := storage.List(ctx, map[string]any{"page": 0, "perPage": 10})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}

	workspaces, ok := resultMap["workspaces"].([]any)
	if !ok {
		t.Fatalf("expected workspaces slice, got %T", resultMap["workspaces"])
	}
	if len(workspaces) != 3 {
		t.Fatalf("expected 3 workspaces, got %d", len(workspaces))
	}
	if resultMap["total"] != 3 {
		t.Errorf("expected total=3, got %v", resultMap["total"])
	}
}

func TestInMemoryWorkspacesStorage_List_Pagination(t *testing.T) {
	// TS: "should handle pagination"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	for i := 0; i < 5; i++ {
		id := "ws-page-" + string(rune('A'+i))
		if _, err := storage.Create(ctx, makeWorkspaceInput(id)); err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
	}

	// First page.
	page1, err := storage.List(ctx, map[string]any{"page": 0, "perPage": 2})
	if err != nil {
		t.Fatalf("List (page 0) returned error: %v", err)
	}
	p1Map := page1.(map[string]any)
	p1Items := p1Map["workspaces"].([]any)
	if len(p1Items) != 2 {
		t.Fatalf("expected 2 items on page 0, got %d", len(p1Items))
	}
	if p1Map["hasMore"] != true {
		t.Error("expected hasMore=true on page 0")
	}

	// Last page.
	page3, err := storage.List(ctx, map[string]any{"page": 2, "perPage": 2})
	if err != nil {
		t.Fatalf("List (page 2) returned error: %v", err)
	}
	p3Map := page3.(map[string]any)
	p3Items := p3Map["workspaces"].([]any)
	if len(p3Items) != 1 {
		t.Fatalf("expected 1 item on last page, got %d", len(p3Items))
	}
	if p3Map["hasMore"] != false {
		t.Error("expected hasMore=false on last page")
	}
}

func TestInMemoryWorkspacesStorage_List_Empty(t *testing.T) {
	// Listing with no workspaces should return empty.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	result, err := storage.List(ctx, map[string]any{"page": 0, "perPage": 10})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	resultMap := result.(map[string]any)
	workspaces := resultMap["workspaces"].([]any)
	if len(workspaces) != 0 {
		t.Errorf("expected 0 workspaces, got %d", len(workspaces))
	}
	if resultMap["total"] != 0 {
		t.Errorf("expected total=0, got %v", resultMap["total"])
	}
}

func TestInMemoryWorkspacesStorage_List_FilterByAuthorID(t *testing.T) {
	// TS: "should filter workspaces by authorId"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	if _, err := storage.Create(ctx, makeWorkspaceInput("ws-a1", map[string]any{"authorId": "author-1"})); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := storage.Create(ctx, makeWorkspaceInput("ws-a2", map[string]any{"authorId": "author-2"})); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	result, err := storage.List(ctx, map[string]any{
		"page":     0,
		"perPage":  10,
		"authorId": "author-1",
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	resultMap := result.(map[string]any)
	workspaces := resultMap["workspaces"].([]any)
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace with authorId=author-1, got %d", len(workspaces))
	}
}

func TestInMemoryWorkspacesStorage_List_FilterByMetadata(t *testing.T) {
	// TS: "should filter workspaces by metadata"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	if _, err := storage.Create(ctx, makeWorkspaceInput("ws-m1", map[string]any{
		"metadata": map[string]any{"env": "prod"},
	})); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := storage.Create(ctx, makeWorkspaceInput("ws-m2", map[string]any{
		"metadata": map[string]any{"env": "staging"},
	})); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	result, err := storage.List(ctx, map[string]any{
		"page":     0,
		"perPage":  10,
		"metadata": map[string]any{"env": "prod"},
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	resultMap := result.(map[string]any)
	workspaces := resultMap["workspaces"].([]any)
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace with metadata.env=prod, got %d", len(workspaces))
	}
}

func TestInMemoryWorkspacesStorage_List_NilArgs(t *testing.T) {
	// Listing with nil args should use default pagination.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	if _, err := storage.Create(ctx, makeWorkspaceInput("ws-nil")); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	result, err := storage.List(ctx, nil)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	resultMap := result.(map[string]any)
	workspaces := resultMap["workspaces"].([]any)
	if len(workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(workspaces))
	}
}

// ===========================================================================
// Tests — Version CRUD
// ===========================================================================

func TestInMemoryWorkspacesStorage_CreateVersion(t *testing.T) {
	// TS: "should create a version"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	version, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
		ID:            "ver-1",
		WorkspaceID:   "ws-1",
		VersionNumber: 1,
		ChangedFields: []string{"name"},
		ChangeMessage: "First version",
		Snapshot:      map[string]any{"name": "Test"},
	})
	if err != nil {
		t.Fatalf("CreateVersion returned error: %v", err)
	}
	if version == nil {
		t.Fatal("expected version, got nil")
	}
	if version.ID != "ver-1" {
		t.Errorf("id mismatch: got %s", version.ID)
	}
	if version.VersionNumber != 1 {
		t.Errorf("versionNumber mismatch: got %d", version.VersionNumber)
	}
	if version.CreatedAt.IsZero() {
		t.Error("expected createdAt to be set")
	}
}

func TestInMemoryWorkspacesStorage_CreateVersion_DuplicateID(t *testing.T) {
	// Creating a version with an existing ID should return an error.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := CreateWorkspaceVersionInput{
		ID:            "ver-dup",
		WorkspaceID:   "ws-1",
		VersionNumber: 1,
		ChangeMessage: "First",
	}
	if _, err := storage.CreateVersion(ctx, input); err != nil {
		t.Fatalf("CreateVersion returned error: %v", err)
	}

	_, err := storage.CreateVersion(ctx, input)
	if err == nil {
		t.Fatal("expected error for duplicate version ID, got nil")
	}
}

func TestInMemoryWorkspacesStorage_CreateVersion_DuplicateVersionNumber(t *testing.T) {
	// Creating a version with an existing version number for the same workspace should error.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
		ID:            "ver-a",
		WorkspaceID:   "ws-1",
		VersionNumber: 1,
		ChangeMessage: "First",
	}); err != nil {
		t.Fatalf("CreateVersion returned error: %v", err)
	}

	_, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
		ID:            "ver-b",
		WorkspaceID:   "ws-1",
		VersionNumber: 1, // duplicate for same workspace
		ChangeMessage: "Second attempt at version 1",
	})
	if err == nil {
		t.Fatal("expected error for duplicate version number, got nil")
	}
}

func TestInMemoryWorkspacesStorage_GetVersion(t *testing.T) {
	// TS: "should retrieve a version by ID"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
		ID:            "ver-get",
		WorkspaceID:   "ws-1",
		VersionNumber: 1,
		ChangeMessage: "Test version",
	}); err != nil {
		t.Fatalf("CreateVersion returned error: %v", err)
	}

	version, err := storage.GetVersion(ctx, "ver-get")
	if err != nil {
		t.Fatalf("GetVersion returned error: %v", err)
	}
	if version == nil {
		t.Fatal("expected version, got nil")
	}
	if version.ID != "ver-get" {
		t.Errorf("id mismatch: got %s", version.ID)
	}
}

func TestInMemoryWorkspacesStorage_GetVersion_NotFound(t *testing.T) {
	// Getting a non-existent version should return nil.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	result, err := storage.GetVersion(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetVersion returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-existent version, got %+v", result)
	}
}

func TestInMemoryWorkspacesStorage_GetVersionByNumber(t *testing.T) {
	// TS: "should retrieve a version by workspace ID and version number"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	for i := 1; i <= 3; i++ {
		if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
			ID:            "ver-num-" + string(rune('0'+i)),
			WorkspaceID:   "ws-nums",
			VersionNumber: i,
			ChangeMessage: "Version " + string(rune('0'+i)),
		}); err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
	}

	version, err := storage.GetVersionByNumber(ctx, "ws-nums", 2)
	if err != nil {
		t.Fatalf("GetVersionByNumber returned error: %v", err)
	}
	if version == nil {
		t.Fatal("expected version, got nil")
	}
	if version.VersionNumber != 2 {
		t.Errorf("expected version number 2, got %d", version.VersionNumber)
	}
}

func TestInMemoryWorkspacesStorage_GetVersionByNumber_NotFound(t *testing.T) {
	// Getting a non-existent version number should return nil.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	result, err := storage.GetVersionByNumber(ctx, "ws-1", 999)
	if err != nil {
		t.Fatalf("GetVersionByNumber returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-existent version number, got %+v", result)
	}
}

func TestInMemoryWorkspacesStorage_GetLatestVersion(t *testing.T) {
	// TS: "should retrieve the latest version"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	for i := 1; i <= 3; i++ {
		if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
			ID:            "ver-latest-" + string(rune('0'+i)),
			WorkspaceID:   "ws-latest",
			VersionNumber: i,
			ChangeMessage: "Version " + string(rune('0'+i)),
		}); err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
	}

	latest, err := storage.GetLatestVersion(ctx, "ws-latest")
	if err != nil {
		t.Fatalf("GetLatestVersion returned error: %v", err)
	}
	if latest == nil {
		t.Fatal("expected latest version, got nil")
	}
	if latest.VersionNumber != 3 {
		t.Errorf("expected latest version number 3, got %d", latest.VersionNumber)
	}
}

func TestInMemoryWorkspacesStorage_GetLatestVersion_NoVersions(t *testing.T) {
	// Getting latest version for a workspace with no versions should return nil.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	result, err := storage.GetLatestVersion(ctx, "no-versions")
	if err != nil {
		t.Fatalf("GetLatestVersion returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

// ===========================================================================
// Tests — ListVersions
// ===========================================================================

func TestInMemoryWorkspacesStorage_ListVersions(t *testing.T) {
	// TS: "should list versions with pagination"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	for i := 1; i <= 5; i++ {
		if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
			ID:            "ver-list-" + string(rune('0'+i)),
			WorkspaceID:   "ws-list-ver",
			VersionNumber: i,
			ChangeMessage: "Version " + string(rune('0'+i)),
		}); err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
	}

	result, err := storage.ListVersions(ctx, ListWorkspaceVersionsInput{
		WorkspaceID: "ws-list-ver",
		Page:        intPtr(0),
		PerPage:     intPtr(3),
	})
	if err != nil {
		t.Fatalf("ListVersions returned error: %v", err)
	}
	if len(result.Versions) != 3 {
		t.Fatalf("expected 3 versions on page 0, got %d", len(result.Versions))
	}
	if result.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Total)
	}
	if !result.HasMore {
		t.Error("expected hasMore=true")
	}
}

func TestInMemoryWorkspacesStorage_ListVersions_DefaultSortDESC(t *testing.T) {
	// Default sort should be DESC by versionNumber.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	for i := 1; i <= 3; i++ {
		if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
			ID:            "ver-sort-" + string(rune('0'+i)),
			WorkspaceID:   "ws-sort",
			VersionNumber: i,
		}); err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
	}

	result, err := storage.ListVersions(ctx, ListWorkspaceVersionsInput{
		WorkspaceID: "ws-sort",
	})
	if err != nil {
		t.Fatalf("ListVersions returned error: %v", err)
	}
	if len(result.Versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(result.Versions))
	}
	// DESC by versionNumber: 3, 2, 1
	if result.Versions[0].VersionNumber != 3 {
		t.Errorf("expected first version number=3 (DESC), got %d", result.Versions[0].VersionNumber)
	}
	if result.Versions[2].VersionNumber != 1 {
		t.Errorf("expected last version number=1 (DESC), got %d", result.Versions[2].VersionNumber)
	}
}

func TestInMemoryWorkspacesStorage_ListVersions_SortASC(t *testing.T) {
	// Explicitly sorting ASC by versionNumber.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	for i := 1; i <= 3; i++ {
		if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
			ID:            "ver-asc-" + string(rune('0'+i)),
			WorkspaceID:   "ws-asc",
			VersionNumber: i,
		}); err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
	}

	result, err := storage.ListVersions(ctx, ListWorkspaceVersionsInput{
		WorkspaceID:      "ws-asc",
		OrderByField:     versionOrderByPtr(WorkspaceVersionOrderByVersionNumber),
		OrderByDirection: versionSortDirPtr(WorkspaceVersionSortASC),
	})
	if err != nil {
		t.Fatalf("ListVersions returned error: %v", err)
	}
	// ASC by versionNumber: 1, 2, 3
	if result.Versions[0].VersionNumber != 1 {
		t.Errorf("expected first version number=1 (ASC), got %d", result.Versions[0].VersionNumber)
	}
	if result.Versions[2].VersionNumber != 3 {
		t.Errorf("expected last version number=3 (ASC), got %d", result.Versions[2].VersionNumber)
	}
}

func TestInMemoryWorkspacesStorage_ListVersions_FiltersToWorkspace(t *testing.T) {
	// Versions from other workspaces should not be returned.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	// Create versions for two different workspaces.
	if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
		ID: "ver-ws1", WorkspaceID: "ws-1", VersionNumber: 1,
	}); err != nil {
		t.Fatalf("CreateVersion returned error: %v", err)
	}
	if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
		ID: "ver-ws2", WorkspaceID: "ws-2", VersionNumber: 1,
	}); err != nil {
		t.Fatalf("CreateVersion returned error: %v", err)
	}

	result, err := storage.ListVersions(ctx, ListWorkspaceVersionsInput{
		WorkspaceID: "ws-1",
	})
	if err != nil {
		t.Fatalf("ListVersions returned error: %v", err)
	}
	if len(result.Versions) != 1 {
		t.Fatalf("expected 1 version for ws-1, got %d", len(result.Versions))
	}
	if result.Versions[0].WorkspaceID != "ws-1" {
		t.Errorf("expected workspaceId=ws-1, got %s", result.Versions[0].WorkspaceID)
	}
}

func TestInMemoryWorkspacesStorage_ListVersions_Empty(t *testing.T) {
	// Listing versions for a workspace with none should return empty.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	result, err := storage.ListVersions(ctx, ListWorkspaceVersionsInput{
		WorkspaceID: "no-versions",
	})
	if err != nil {
		t.Fatalf("ListVersions returned error: %v", err)
	}
	if len(result.Versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(result.Versions))
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

// ===========================================================================
// Tests — DeleteVersion & DeleteVersionsByParentID & CountVersions
// ===========================================================================

func TestInMemoryWorkspacesStorage_DeleteVersion(t *testing.T) {
	// TS: "should delete a version by ID"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
		ID: "ver-del", WorkspaceID: "ws-del", VersionNumber: 1,
	}); err != nil {
		t.Fatalf("CreateVersion returned error: %v", err)
	}

	// Verify it exists.
	v, _ := storage.GetVersion(ctx, "ver-del")
	if v == nil {
		t.Fatal("expected version to exist before delete")
	}

	if err := storage.DeleteVersion(ctx, "ver-del"); err != nil {
		t.Fatalf("DeleteVersion returned error: %v", err)
	}

	// Verify it is gone.
	v, _ = storage.GetVersion(ctx, "ver-del")
	if v != nil {
		t.Error("expected version to be deleted")
	}
}

func TestInMemoryWorkspacesStorage_DeleteVersionsByParentID(t *testing.T) {
	// TS: "should delete all versions for a workspace"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	for i := 1; i <= 3; i++ {
		if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
			ID:            "ver-del-p-" + string(rune('0'+i)),
			WorkspaceID:   "ws-del-parent",
			VersionNumber: i,
		}); err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
	}

	// Also create a version for a different workspace.
	if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
		ID: "ver-other", WorkspaceID: "ws-other", VersionNumber: 1,
	}); err != nil {
		t.Fatalf("CreateVersion returned error: %v", err)
	}

	count, _ := storage.CountVersions(ctx, "ws-del-parent")
	if count != 3 {
		t.Fatalf("expected 3 versions before delete, got %d", count)
	}

	if err := storage.DeleteVersionsByParentID(ctx, "ws-del-parent"); err != nil {
		t.Fatalf("DeleteVersionsByParentID returned error: %v", err)
	}

	// All versions for ws-del-parent should be deleted.
	count, _ = storage.CountVersions(ctx, "ws-del-parent")
	if count != 0 {
		t.Errorf("expected 0 versions after delete, got %d", count)
	}

	// Versions for other workspaces should remain.
	countOther, _ := storage.CountVersions(ctx, "ws-other")
	if countOther != 1 {
		t.Errorf("expected 1 version for ws-other to remain, got %d", countOther)
	}
}

func TestInMemoryWorkspacesStorage_CountVersions(t *testing.T) {
	// TS: "should count versions for a workspace"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	for i := 1; i <= 4; i++ {
		if _, err := storage.CreateVersion(ctx, CreateWorkspaceVersionInput{
			ID:            "ver-count-" + string(rune('0'+i)),
			WorkspaceID:   "ws-count",
			VersionNumber: i,
		}); err != nil {
			t.Fatalf("CreateVersion returned error: %v", err)
		}
	}

	count, err := storage.CountVersions(ctx, "ws-count")
	if err != nil {
		t.Fatalf("CountVersions returned error: %v", err)
	}
	if count != 4 {
		t.Errorf("expected 4 versions, got %d", count)
	}
}

func TestInMemoryWorkspacesStorage_CountVersions_Empty(t *testing.T) {
	// Counting versions for a workspace with none should return 0.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	count, err := storage.CountVersions(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("CountVersions returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 versions, got %d", count)
	}
}

// ===========================================================================
// Tests — Resolution Methods
// ===========================================================================

func TestInMemoryWorkspacesStorage_GetByIDResolved(t *testing.T) {
	// TS: "should resolve entity with version config"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	input := makeWorkspaceInput("ws-resolve", map[string]any{
		"name":        "Resolved Workspace",
		"description": "A test workspace",
	})
	if _, err := storage.Create(ctx, input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	resolved, err := storage.GetByIDResolved(ctx, "ws-resolve", "draft")
	if err != nil {
		t.Fatalf("GetByIDResolved returned error: %v", err)
	}
	if resolved == nil {
		t.Fatal("expected resolved entity, got nil")
	}

	resolvedMap, ok := resolved.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", resolved)
	}
	if resolvedMap["id"] != "ws-resolve" {
		t.Errorf("expected id=ws-resolve, got %v", resolvedMap["id"])
	}
	// Should have resolvedVersionId.
	if resolvedMap["resolvedVersionId"] == nil || resolvedMap["resolvedVersionId"] == "" {
		t.Error("expected resolvedVersionId to be set")
	}
}

func TestInMemoryWorkspacesStorage_GetByIDResolved_NotFound(t *testing.T) {
	// Resolving a non-existent workspace should return nil.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	result, err := storage.GetByIDResolved(ctx, "nonexistent", "")
	if err != nil {
		t.Fatalf("GetByIDResolved returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-existent workspace, got %+v", result)
	}
}

func TestInMemoryWorkspacesStorage_ListResolved(t *testing.T) {
	// TS: "should list resolved entities"
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	for i := 0; i < 2; i++ {
		id := "ws-lr-" + string(rune('A'+i))
		input := makeWorkspaceInput(id, map[string]any{
			"name": "Workspace " + string(rune('A'+i)),
		})
		if _, err := storage.Create(ctx, input); err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
	}

	resolved, err := storage.ListResolved(ctx, map[string]any{
		"page":    0,
		"perPage": 10,
		"status":  "draft",
	})
	if err != nil {
		t.Fatalf("ListResolved returned error: %v", err)
	}

	resolvedMap, ok := resolved.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", resolved)
	}
	workspaces, ok := resolvedMap["workspaces"].([]any)
	if !ok {
		t.Fatalf("expected workspaces slice, got %T", resolvedMap["workspaces"])
	}
	if len(workspaces) != 2 {
		t.Fatalf("expected 2 resolved workspaces, got %d", len(workspaces))
	}

	// Each resolved entity should have resolvedVersionId.
	for i, ws := range workspaces {
		wsMap, ok := ws.(map[string]any)
		if !ok {
			t.Fatalf("workspace[%d]: expected map, got %T", i, ws)
		}
		if wsMap["resolvedVersionId"] == nil || wsMap["resolvedVersionId"] == "" {
			t.Errorf("workspace[%d]: expected resolvedVersionId to be set", i)
		}
	}
}

// ===========================================================================
// Tests — DangerouslyClearAll
// ===========================================================================

func TestInMemoryWorkspacesStorage_DangerouslyClearAll(t *testing.T) {
	// TS pattern from other domains — verify clear removes all data.
	ctx := context.Background()
	storage := NewInMemoryWorkspacesStorage()

	// Create workspaces (which also creates versions).
	if _, err := storage.Create(ctx, makeWorkspaceInput("ws-clear-1")); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := storage.Create(ctx, makeWorkspaceInput("ws-clear-2")); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	// Verify they exist.
	r1, _ := storage.GetByID(ctx, "ws-clear-1")
	r2, _ := storage.GetByID(ctx, "ws-clear-2")
	if r1 == nil || r2 == nil {
		t.Fatal("expected both workspaces to exist before clear")
	}

	// Clear all.
	if err := storage.DangerouslyClearAll(ctx); err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	// Verify workspaces are gone.
	r1, _ = storage.GetByID(ctx, "ws-clear-1")
	r2, _ = storage.GetByID(ctx, "ws-clear-2")
	if r1 != nil {
		t.Error("expected first workspace to be cleared")
	}
	if r2 != nil {
		t.Error("expected second workspace to be cleared")
	}

	// Verify versions are also cleared.
	count1, _ := storage.CountVersions(ctx, "ws-clear-1")
	count2, _ := storage.CountVersions(ctx, "ws-clear-2")
	if count1 != 0 {
		t.Errorf("expected 0 versions for ws-clear-1 after clear, got %d", count1)
	}
	if count2 != 0 {
		t.Errorf("expected 0 versions for ws-clear-2 after clear, got %d", count2)
	}

	// List should return empty.
	listResult, _ := storage.List(ctx, map[string]any{"page": 0, "perPage": 10})
	resultMap := listResult.(map[string]any)
	workspaces := resultMap["workspaces"].([]any)
	if len(workspaces) != 0 {
		t.Errorf("expected 0 workspaces after clear, got %d", len(workspaces))
	}
}
