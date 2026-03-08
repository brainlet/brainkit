// Ported from: packages/core/src/workspace/workspace.test.ts
package workspace

import (
	"strings"
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/workspace/search"
	"github.com/brainlet/brainkit/agent-kit/core/workspace/skills"
)

// mockVector satisfies search.MastraVector for testing.
type mockVector struct{}

func (m *mockVector) Upsert(_ search.VectorUpsertParams) error            { return nil }
func (m *mockVector) Query(_ search.VectorQueryParams) ([]search.VectorQueryResult, error) { return nil, nil }
func (m *mockVector) DeleteVector(_ search.VectorDeleteParams) error       { return nil }

// mockSkillsResolver satisfies skills.SkillsResolver for testing.
type mockSkillsResolver struct{}

func (m *mockSkillsResolver) ResolvePaths(_ skills.SkillsContext) ([]string, error) {
	return []string{"/skills"}, nil
}

func TestNewWorkspace(t *testing.T) {
	t.Run("creates workspace with filesystem", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, err := NewWorkspace(WorkspaceConfig{
			Filesystem: fs,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ws.Filesystem() == nil {
			t.Error("Filesystem should be set")
		}
		if ws.Status() != WorkspaceStatusPending {
			t.Errorf("Status = %q, want %q", ws.Status(), WorkspaceStatusPending)
		}
	})

	t.Run("generates ID when not provided", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, err := NewWorkspace(WorkspaceConfig{
			Filesystem: fs,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ws.ID == "" {
			t.Error("ID should be auto-generated")
		}
		if !strings.HasPrefix(ws.ID, "ws-") {
			t.Errorf("ID should start with 'ws-', got %q", ws.ID)
		}
	})

	t.Run("uses provided ID", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, err := NewWorkspace(WorkspaceConfig{
			ID:         "custom-id",
			Filesystem: fs,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ws.ID != "custom-id" {
			t.Errorf("ID = %q, want %q", ws.ID, "custom-id")
		}
	})

	t.Run("generates name from ID when not provided", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, err := NewWorkspace(WorkspaceConfig{
			ID:         "my-workspace-id",
			Filesystem: fs,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(ws.WorkspaceName, "workspace-") {
			t.Errorf("WorkspaceName should start with 'workspace-', got %q", ws.WorkspaceName)
		}
	})

	t.Run("uses provided name", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, err := NewWorkspace(WorkspaceConfig{
			Name:       "My Workspace",
			Filesystem: fs,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ws.WorkspaceName != "My Workspace" {
			t.Errorf("WorkspaceName = %q, want %q", ws.WorkspaceName, "My Workspace")
		}
	})

	t.Run("errors when no providers are given", func(t *testing.T) {
		_, err := NewWorkspace(WorkspaceConfig{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		wsErr, ok := err.(*WorkspaceError)
		if !ok {
			t.Fatalf("expected *WorkspaceError, got %T", err)
		}
		if wsErr.Code != "NO_PROVIDERS" {
			t.Errorf("Code = %q, want %q", wsErr.Code, "NO_PROVIDERS")
		}
	})

	t.Run("errors when both filesystem and mounts are provided", func(t *testing.T) {
		fs := &mockFilesystem{}
		_, err := NewWorkspace(WorkspaceConfig{
			Filesystem: fs,
			Mounts: map[string]WorkspaceFilesystem{
				"/data": fs,
			},
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		wsErr, ok := err.(*WorkspaceError)
		if !ok {
			t.Fatalf("expected *WorkspaceError, got %T", err)
		}
		if wsErr.Code != "INVALID_CONFIG" {
			t.Errorf("Code = %q, want %q", wsErr.Code, "INVALID_CONFIG")
		}
	})

	t.Run("errors when vectorStore is provided without embedder", func(t *testing.T) {
		fs := &mockFilesystem{}
		_, err := NewWorkspace(WorkspaceConfig{
			Filesystem:  fs,
			VectorStore: &mockVector{},
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		wsErr, ok := err.(*WorkspaceError)
		if !ok {
			t.Fatalf("expected *WorkspaceError, got %T", err)
		}
		if wsErr.Code != "INVALID_SEARCH_CONFIG" {
			t.Errorf("Code = %q, want %q", wsErr.Code, "INVALID_SEARCH_CONFIG")
		}
	})

	t.Run("creates workspace with skills only", func(t *testing.T) {
		ws, err := NewWorkspace(WorkspaceConfig{
			Skills: &mockSkillsResolver{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ws == nil {
			t.Fatal("workspace should not be nil")
		}
	})
}

func TestWorkspaceAccessors(t *testing.T) {
	fs := &mockFilesystem{}
	ws, err := NewWorkspace(WorkspaceConfig{
		Filesystem: fs,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("Filesystem returns the filesystem provider", func(t *testing.T) {
		if ws.Filesystem() == nil {
			t.Error("Filesystem should not be nil")
		}
	})

	t.Run("Sandbox returns nil when not configured", func(t *testing.T) {
		if ws.Sandbox() != nil {
			t.Error("Sandbox should be nil when not configured")
		}
	})

	t.Run("LSP returns nil when not configured", func(t *testing.T) {
		if ws.LSP() != nil {
			t.Error("LSP should be nil when not configured")
		}
	})

	t.Run("CanBM25 returns false when search not configured", func(t *testing.T) {
		if ws.CanBM25() {
			t.Error("CanBM25 should return false")
		}
	})

	t.Run("CanVector returns false when search not configured", func(t *testing.T) {
		if ws.CanVector() {
			t.Error("CanVector should return false")
		}
	})

	t.Run("CanHybrid returns false when search not configured", func(t *testing.T) {
		if ws.CanHybrid() {
			t.Error("CanHybrid should return false")
		}
	})
}

func TestWorkspaceToolsConfig(t *testing.T) {
	t.Run("GetToolsConfig returns nil when not set", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, _ := NewWorkspace(WorkspaceConfig{Filesystem: fs})
		if ws.GetToolsConfig() != nil {
			t.Error("GetToolsConfig should return nil when not configured")
		}
	})

	t.Run("SetToolsConfig updates the config", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, _ := NewWorkspace(WorkspaceConfig{Filesystem: fs})

		enabled := true
		config := WorkspaceToolsConfig{
			"my_tool": WorkspaceToolConfig{Enabled: &enabled},
		}
		ws.SetToolsConfig(config)

		got := ws.GetToolsConfig()
		if got == nil {
			t.Fatal("GetToolsConfig should not be nil after set")
		}
		if got["my_tool"].Enabled == nil || !*got["my_tool"].Enabled {
			t.Error("tool should be enabled")
		}
	})
}

func TestWorkspaceInit(t *testing.T) {
	t.Run("transitions to ready status on successful init", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, _ := NewWorkspace(WorkspaceConfig{Filesystem: fs})

		err := ws.Init()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ws.Status() != WorkspaceStatusReady {
			t.Errorf("Status = %q, want %q", ws.Status(), WorkspaceStatusReady)
		}
	})
}

func TestWorkspaceDestroy(t *testing.T) {
	t.Run("transitions to destroyed status", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, _ := NewWorkspace(WorkspaceConfig{Filesystem: fs})
		_ = ws.Init()

		err := ws.Destroy()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ws.Status() != WorkspaceStatusDestroyed {
			t.Errorf("Status = %q, want %q", ws.Status(), WorkspaceStatusDestroyed)
		}
	})
}

func TestWorkspaceGetInfo(t *testing.T) {
	t.Run("returns workspace info", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, _ := NewWorkspace(WorkspaceConfig{
			ID:         "test-ws",
			Name:       "Test Workspace",
			Filesystem: fs,
		})

		info, err := ws.GetInfo(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.ID != "test-ws" {
			t.Errorf("ID = %q, want %q", info.ID, "test-ws")
		}
		if info.Name != "Test Workspace" {
			t.Errorf("Name = %q, want %q", info.Name, "Test Workspace")
		}
		if info.Filesystem == nil {
			t.Error("Filesystem info should not be nil")
		}
		if info.Sandbox != nil {
			t.Error("Sandbox info should be nil when no sandbox configured")
		}
	})
}

func TestWorkspaceSearch(t *testing.T) {
	t.Run("returns error when search is not configured", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, _ := NewWorkspace(WorkspaceConfig{Filesystem: fs})

		_, err := ws.Search("query", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		searchErr, ok := err.(*SearchNotAvailableError)
		if !ok {
			t.Fatalf("expected *SearchNotAvailableError, got %T", err)
		}
		if searchErr.Code != "NO_SEARCH" {
			t.Errorf("Code = %q, want %q", searchErr.Code, "NO_SEARCH")
		}
	})
}

func TestWorkspaceIndex(t *testing.T) {
	t.Run("returns error when search is not configured", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, _ := NewWorkspace(WorkspaceConfig{Filesystem: fs})

		err := ws.Index("/path/to/file", "content", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestWorkspaceGetPathContext(t *testing.T) {
	t.Run("returns path context with filesystem details", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, _ := NewWorkspace(WorkspaceConfig{Filesystem: fs})

		ctx := ws.GetPathContext()
		if ctx.Filesystem == nil {
			t.Fatal("Filesystem should not be nil")
		}
		if ctx.Filesystem.Provider != "mock" {
			t.Errorf("Provider = %q, want %q", ctx.Filesystem.Provider, "mock")
		}
	})

	t.Run("returns nil sandbox when not configured", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, _ := NewWorkspace(WorkspaceConfig{Filesystem: fs})

		ctx := ws.GetPathContext()
		if ctx.Sandbox != nil {
			t.Error("Sandbox should be nil")
		}
	})
}

func TestWorkspaceSkills(t *testing.T) {
	t.Run("returns nil when skills not configured", func(t *testing.T) {
		fs := &mockFilesystem{}
		ws, _ := NewWorkspace(WorkspaceConfig{Filesystem: fs})
		if ws.Skills() != nil {
			t.Error("Skills should be nil when not configured")
		}
	})
}

func TestRandomAlphanumeric(t *testing.T) {
	t.Run("generates string of correct length", func(t *testing.T) {
		result := randomAlphanumeric(10)
		if len(result) != 10 {
			t.Errorf("length = %d, want 10", len(result))
		}
	})

	t.Run("generates only lowercase alphanumeric chars", func(t *testing.T) {
		result := randomAlphanumeric(100)
		for _, c := range result {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
				t.Errorf("unexpected character: %c", c)
			}
		}
	})
}
