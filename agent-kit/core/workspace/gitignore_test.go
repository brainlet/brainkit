// Ported from: packages/core/src/workspace/gitignore.test.ts
package workspace

import (
	"testing"
)

func TestParseGitignorePatterns(t *testing.T) {
	t.Run("parses basic patterns", func(t *testing.T) {
		patterns := parseGitignorePatterns("node_modules\n*.log\n")
		if len(patterns) != 2 {
			t.Fatalf("expected 2 patterns, got %d", len(patterns))
		}
		if patterns[0].pattern != "node_modules" {
			t.Errorf("pattern[0] = %q, want %q", patterns[0].pattern, "node_modules")
		}
		if patterns[1].pattern != "*.log" {
			t.Errorf("pattern[1] = %q, want %q", patterns[1].pattern, "*.log")
		}
	})

	t.Run("skips empty lines and comments", func(t *testing.T) {
		patterns := parseGitignorePatterns("# comment\n\nnode_modules\n# another\n*.log")
		if len(patterns) != 2 {
			t.Fatalf("expected 2 patterns, got %d", len(patterns))
		}
	})

	t.Run("detects negation patterns", func(t *testing.T) {
		patterns := parseGitignorePatterns("*.log\n!important.log")
		if len(patterns) != 2 {
			t.Fatalf("expected 2 patterns, got %d", len(patterns))
		}
		if patterns[0].isNegation {
			t.Error("first pattern should not be negation")
		}
		if !patterns[1].isNegation {
			t.Error("second pattern should be negation")
		}
		if patterns[1].pattern != "important.log" {
			t.Errorf("negated pattern = %q, want %q", patterns[1].pattern, "important.log")
		}
	})

	t.Run("detects directory patterns", func(t *testing.T) {
		patterns := parseGitignorePatterns("build/\nnode_modules/")
		if len(patterns) != 2 {
			t.Fatalf("expected 2 patterns, got %d", len(patterns))
		}
		if !patterns[0].isDir {
			t.Error("build/ should be directory pattern")
		}
		if patterns[0].pattern != "build" {
			t.Errorf("pattern = %q, want %q", patterns[0].pattern, "build")
		}
	})

	t.Run("returns nil for empty content", func(t *testing.T) {
		patterns := parseGitignorePatterns("")
		if len(patterns) != 0 {
			t.Errorf("expected 0 patterns, got %d", len(patterns))
		}
	})
}

func TestMatchesGitignore(t *testing.T) {
	t.Run("matches simple file name", func(t *testing.T) {
		patterns := parseGitignorePatterns("*.log")
		if !matchesGitignore("debug.log", patterns) {
			t.Error("should match *.log against debug.log")
		}
	})

	t.Run("matches directory name anywhere", func(t *testing.T) {
		patterns := parseGitignorePatterns("node_modules")
		if !matchesGitignore("node_modules", patterns) {
			t.Error("should match at root")
		}
		if !matchesGitignore("src/node_modules", patterns) {
			t.Error("should match in subdirectory")
		}
	})

	t.Run("matches deeply nested file", func(t *testing.T) {
		patterns := parseGitignorePatterns("*.log")
		if !matchesGitignore("deep/nested/file.log", patterns) {
			t.Error("should match *.log in nested path")
		}
	})

	t.Run("negation pattern un-ignores files", func(t *testing.T) {
		patterns := parseGitignorePatterns("*.log\n!important.log")
		if matchesGitignore("important.log", patterns) {
			t.Error("important.log should be un-ignored by negation pattern")
		}
		if !matchesGitignore("debug.log", patterns) {
			t.Error("debug.log should still be ignored")
		}
	})

	t.Run("does not match unrelated files", func(t *testing.T) {
		patterns := parseGitignorePatterns("*.log")
		if matchesGitignore("src/index.ts", patterns) {
			t.Error("should not match unrelated file")
		}
	})
}

func TestMatchesPattern(t *testing.T) {
	t.Run("** matches everything", func(t *testing.T) {
		if !matchesPattern("anything/at/all", "**") {
			t.Error("** should match any path")
		}
	})

	t.Run("**/name matches at any level", func(t *testing.T) {
		if !matchesPattern("foo/bar/test.log", "**/test.log") {
			t.Error("should match nested test.log")
		}
		if !matchesPattern("test.log", "**/test.log") {
			t.Error("should match root-level test.log")
		}
	})

	t.Run("name/** matches everything under name", func(t *testing.T) {
		if !matchesPattern("build/output.js", "build/**") {
			t.Error("should match files under build/")
		}
		if !matchesPattern("build", "build/**") {
			t.Error("should match the directory itself")
		}
		if matchesPattern("other/file.js", "build/**") {
			t.Error("should not match files outside build/")
		}
	})

	t.Run("pattern without slashes matches basename anywhere", func(t *testing.T) {
		if !matchesPattern("src/file.txt", "file.txt") {
			t.Error("should match file.txt in any directory")
		}
		if !matchesPattern("file.txt", "file.txt") {
			t.Error("should match file.txt at root")
		}
	})

	t.Run("pattern with slash matches from root", func(t *testing.T) {
		if !matchesPattern("src/file.txt", "src/file.txt") {
			t.Error("should match exact rooted path")
		}
	})
}

func TestMatchGlobSimple(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		if !matchGlobSimple("hello", "hello") {
			t.Error("should match exact string")
		}
	})

	t.Run("? matches single character", func(t *testing.T) {
		if !matchGlobSimple("hello", "hell?") {
			t.Error("? should match one char")
		}
		if matchGlobSimple("hel", "hell?") {
			t.Error("? requires exactly one char")
		}
	})

	t.Run("* matches any sequence", func(t *testing.T) {
		if !matchGlobSimple("hello.ts", "*.ts") {
			t.Error("* should match any prefix")
		}
		if !matchGlobSimple(".ts", "*.ts") {
			t.Error("* should match empty prefix")
		}
		if matchGlobSimple("hello.js", "*.ts") {
			t.Error("should not match wrong suffix")
		}
	})

	t.Run("* does not match empty string at end by default", func(t *testing.T) {
		if !matchGlobSimple("file", "file*") {
			t.Error("* at end should match empty")
		}
		if !matchGlobSimple("file.txt", "file*") {
			t.Error("* at end should match any suffix")
		}
	})

	t.Run("complex patterns", func(t *testing.T) {
		if !matchGlobSimple("test_file.go", "test_*.go") {
			t.Error("should match test_*.go pattern")
		}
		if !matchGlobSimple("abc", "a*c") {
			t.Error("should match a*c")
		}
		if !matchGlobSimple("abbc", "a*c") {
			t.Error("should match a*c with multiple middle chars")
		}
	})
}

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
