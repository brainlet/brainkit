// Ported from: packages/core/src/harness/workspace-resolution.test.ts
package harness

import (
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/workspace"
)

// mockHarnessFilesystem is a minimal WorkspaceFilesystem for harness tests.
type mockHarnessFilesystem struct{}

func (m *mockHarnessFilesystem) ID() string                                                        { return "mock-fs" }
func (m *mockHarnessFilesystem) Name() string                                                      { return "mock" }
func (m *mockHarnessFilesystem) Provider() string                                                  { return "mock" }
func (m *mockHarnessFilesystem) ReadOnly() bool                                                    { return false }
func (m *mockHarnessFilesystem) BasePath() string                                                  { return "/" }
func (m *mockHarnessFilesystem) Icon() *workspace.FilesystemIcon                                   { return nil }
func (m *mockHarnessFilesystem) DisplayName() string                                               { return "Mock" }
func (m *mockHarnessFilesystem) Description() string                                               { return "" }
func (m *mockHarnessFilesystem) GetInstructions(_ *workspace.InstructionsOpts) string               { return "" }
func (m *mockHarnessFilesystem) GetMountConfig() *workspace.FilesystemMountConfig                  { return nil }
func (m *mockHarnessFilesystem) ReadFile(_ string, _ *workspace.ReadOptions) (interface{}, error)   { return "", nil }
func (m *mockHarnessFilesystem) WriteFile(_ string, _ interface{}, _ *workspace.WriteOptions) error { return nil }
func (m *mockHarnessFilesystem) AppendFile(_ string, _ interface{}) error                          { return nil }
func (m *mockHarnessFilesystem) DeleteFile(_ string, _ *workspace.RemoveOptions) error             { return nil }
func (m *mockHarnessFilesystem) CopyFile(_, _ string, _ *workspace.CopyOptions) error              { return nil }
func (m *mockHarnessFilesystem) MoveFile(_, _ string, _ *workspace.CopyOptions) error              { return nil }
func (m *mockHarnessFilesystem) Mkdir(_ string, _ *workspace.MkdirOptions) error                   { return nil }
func (m *mockHarnessFilesystem) Rmdir(_ string, _ *workspace.RemoveOptions) error                  { return nil }
func (m *mockHarnessFilesystem) Readdir(_ string, _ *workspace.ListOptions) ([]workspace.FileEntry, error) {
	return nil, nil
}
func (m *mockHarnessFilesystem) ResolveAbsolutePath(p string) string              { return p }
func (m *mockHarnessFilesystem) Exists(_ string) (bool, error)                    { return false, nil }
func (m *mockHarnessFilesystem) Stat(_ string) (*workspace.FileStat, error)       { return nil, nil }
func (m *mockHarnessFilesystem) Init() error                                      { return nil }
func (m *mockHarnessFilesystem) Destroy() error                                   { return nil }
func (m *mockHarnessFilesystem) GetInfo() (*workspace.FilesystemInfo, error) {
	return &workspace.FilesystemInfo{Provider: "mock"}, nil
}
func (m *mockHarnessFilesystem) Status() workspace.ProviderStatus { return "ready" }

func TestHarnessWorkspaceStaticInstance(t *testing.T) {
	t.Run("workspace is accessible when configured", func(t *testing.T) {
		ws, err := workspace.NewWorkspace(workspace.WorkspaceConfig{
			ID:         "ws-1",
			Name:       "test-workspace",
			Filesystem: &mockHarnessFilesystem{},
		})
		if err != nil {
			t.Fatalf("failed to create workspace: %v", err)
		}
		h, err := New(HarnessConfig{
			ID: "test",
			Modes: []HarnessMode{
				{ID: "default", Name: "Default", Default: true},
			},
			Workspace: ws,
		})
		if err != nil {
			t.Fatalf("failed to create harness: %v", err)
		}

		if h.config.Workspace == nil {
			t.Fatal("expected Workspace to be set in config")
		}
		if h.config.Workspace.WorkspaceName != "test-workspace" {
			t.Errorf("expected workspace name = %q, got %q", "test-workspace", h.config.Workspace.WorkspaceName)
		}
	})
}

func TestHarnessWorkspaceDynamicFactory(t *testing.T) {
	t.Skip("not yet implemented - requires resolveWorkspace, getWorkspace, hasWorkspace methods and workspace factory support")
}
