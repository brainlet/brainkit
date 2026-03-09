// Ported from: packages/core/src/workspace/gitignore.test.ts
package workspace

import (
	"testing"
)

func TestLoadGitignore(t *testing.T) {
	t.Run("returns nil when filesystem returns error", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileErr: &FileNotFoundError{},
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter != nil {
			t.Error("filter should be nil when no .gitignore exists")
		}
	})

	t.Run("returns nil for empty .gitignore", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "   \n\n  ",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter != nil {
			t.Error("filter should be nil for empty .gitignore")
		}
	})

	t.Run("creates working filter from .gitignore content", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "node_modules\n*.log\n",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("node_modules") {
			t.Error("should ignore node_modules")
		}
		if !filter("debug.log") {
			t.Error("should ignore .log files")
		}
		if filter("src/index.ts") {
			t.Error("should not ignore .ts files")
		}
	})

	t.Run("normalizes paths with leading ./ and /", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "dist\n",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("./dist") {
			t.Error("should handle leading ./")
		}
		if !filter("/dist") {
			t.Error("should handle leading /")
		}
	})

	t.Run("returns false for empty path", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "dist\n",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter("") {
			t.Error("empty path should not be ignored")
		}
	})

	t.Run("matches simple file name with glob", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "*.log",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("debug.log") {
			t.Error("should match *.log against debug.log")
		}
	})

	t.Run("matches directory name anywhere", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "node_modules",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("node_modules") {
			t.Error("should match at root")
		}
		if !filter("src/node_modules") {
			t.Error("should match in subdirectory")
		}
	})

	t.Run("matches deeply nested file", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "*.log",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("deep/nested/file.log") {
			t.Error("should match *.log in nested path")
		}
	})

	t.Run("negation pattern un-ignores files", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "*.log\n!important.log",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if filter("important.log") {
			t.Error("important.log should be un-ignored by negation pattern")
		}
		if !filter("debug.log") {
			t.Error("debug.log should still be ignored")
		}
	})

	t.Run("does not match unrelated files", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "*.log",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if filter("src/index.ts") {
			t.Error("should not match unrelated file")
		}
	})

	t.Run("** matches everything", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "**",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("anything/at/all") {
			t.Error("** should match any path")
		}
	})

	t.Run("**/name matches at any level", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "**/test.log",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("foo/bar/test.log") {
			t.Error("should match nested test.log")
		}
		if !filter("test.log") {
			t.Error("should match root-level test.log")
		}
	})

	t.Run("name/** matches everything under name", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "build/**",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("build/output.js") {
			t.Error("should match files under build/")
		}
		if filter("other/file.js") {
			t.Error("should not match files outside build/")
		}
	})

	t.Run("pattern without slashes matches basename anywhere", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "file.txt",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("src/file.txt") {
			t.Error("should match file.txt in any directory")
		}
		if !filter("file.txt") {
			t.Error("should match file.txt at root")
		}
	})

	t.Run("pattern with slash matches from root", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "src/file.txt",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("src/file.txt") {
			t.Error("should match exact rooted path")
		}
	})

	t.Run("skips comments and empty lines", func(t *testing.T) {
		fs := &mockFilesystem{
			readFileResult: "# comment\n\nnode_modules\n# another\n*.log",
		}
		filter, err := LoadGitignore(fs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter == nil {
			t.Fatal("filter should not be nil")
		}

		if !filter("node_modules") {
			t.Error("should match node_modules")
		}
		if !filter("test.log") {
			t.Error("should match *.log")
		}
		if filter("src/index.ts") {
			t.Error("should not match unrelated file")
		}
	})
}

// mockFilesystem is a minimal mock implementing WorkspaceFilesystem for testing.
type mockFilesystem struct {
	readFileResult interface{}
	readFileErr    error
}

func (m *mockFilesystem) ID() string                                       { return "mock" }
func (m *mockFilesystem) Name() string                                     { return "mock" }
func (m *mockFilesystem) Provider() string                                 { return "mock" }
func (m *mockFilesystem) ReadOnly() bool                                   { return false }
func (m *mockFilesystem) BasePath() string                                 { return "/" }
func (m *mockFilesystem) Icon() *FilesystemIcon                            { return nil }
func (m *mockFilesystem) DisplayName() string                              { return "Mock" }
func (m *mockFilesystem) Description() string                              { return "" }
func (m *mockFilesystem) GetInstructions(_ *InstructionsOpts) string       { return "" }
func (m *mockFilesystem) GetMountConfig() *FilesystemMountConfig           { return nil }
func (m *mockFilesystem) ReadFile(_ string, _ *ReadOptions) (interface{}, error) {
	if m.readFileErr != nil {
		return nil, m.readFileErr
	}
	return m.readFileResult, nil
}
func (m *mockFilesystem) WriteFile(_ string, _ interface{}, _ *WriteOptions) error { return nil }
func (m *mockFilesystem) AppendFile(_ string, _ interface{}) error                 { return nil }
func (m *mockFilesystem) DeleteFile(_ string, _ *RemoveOptions) error              { return nil }
func (m *mockFilesystem) CopyFile(_, _ string, _ *CopyOptions) error               { return nil }
func (m *mockFilesystem) MoveFile(_, _ string, _ *CopyOptions) error               { return nil }
func (m *mockFilesystem) Mkdir(_ string, _ *MkdirOptions) error                    { return nil }
func (m *mockFilesystem) Rmdir(_ string, _ *RemoveOptions) error                   { return nil }
func (m *mockFilesystem) Readdir(_ string, _ *ListOptions) ([]FileEntry, error)    { return nil, nil }
func (m *mockFilesystem) ResolveAbsolutePath(p string) string                      { return p }
func (m *mockFilesystem) Exists(_ string) (bool, error)                            { return false, nil }
func (m *mockFilesystem) Stat(_ string) (*FileStat, error)                         { return nil, nil }
func (m *mockFilesystem) Init() error                                              { return nil }
func (m *mockFilesystem) Destroy() error                                           { return nil }
func (m *mockFilesystem) GetInfo() (*FilesystemInfo, error)                        { return nil, nil }
func (m *mockFilesystem) Status() ProviderStatus                                   { return ProviderStatusReady }
