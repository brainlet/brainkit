// Ported from: packages/core/src/workspace/glob.test.ts
package workspace

import (
	"testing"
)

func TestIsGlobPattern(t *testing.T) {
	t.Run("returns false for plain paths", func(t *testing.T) {
		cases := []string{"/docs", "/docs/readme.md", "src/index.ts", ".env"}
		for _, c := range cases {
			if IsGlobPattern(c) {
				t.Errorf("IsGlobPattern(%q) = true, want false", c)
			}
		}
	})

	t.Run("returns true for paths with glob chars", func(t *testing.T) {
		cases := []string{
			"*.ts",
			"/docs/**/*.md",
			"/src/{a,b}",
			"test?.js",
			"[abc].txt",
		}
		for _, c := range cases {
			if !IsGlobPattern(c) {
				t.Errorf("IsGlobPattern(%q) = false, want true", c)
			}
		}
	})
}

func TestExtractGlobBase(t *testing.T) {
	t.Run("returns full path for non-glob patterns", func(t *testing.T) {
		if got := ExtractGlobBase("/exact/path"); got != "/exact/path" {
			t.Errorf("got %q, want %q", got, "/exact/path")
		}
	})

	t.Run("extracts base from /docs/**/*.md", func(t *testing.T) {
		if got := ExtractGlobBase("/docs/**/*.md"); got != "/docs" {
			t.Errorf("got %q, want %q", got, "/docs")
		}
	})

	t.Run("returns root for **/*.md", func(t *testing.T) {
		if got := ExtractGlobBase("**/*.md"); got != "/" {
			t.Errorf("got %q, want %q", got, "/")
		}
	})

	t.Run("extracts base from /src/*.ts", func(t *testing.T) {
		if got := ExtractGlobBase("/src/*.ts"); got != "/src" {
			t.Errorf("got %q, want %q", got, "/src")
		}
	})

	t.Run("returns root for *.ts", func(t *testing.T) {
		if got := ExtractGlobBase("*.ts"); got != "/" {
			t.Errorf("got %q, want %q", got, "/")
		}
	})
}

func TestCreateGlobMatcher(t *testing.T) {
	t.Run("matches files with **/*.ts pattern", func(t *testing.T) {
		match := CreateGlobMatcher([]string{"**/*.ts"}, nil)
		if !match("src/index.ts") {
			t.Error("should match src/index.ts")
		}
		if !match("src/deep/nested/file.ts") {
			t.Error("should match src/deep/nested/file.ts")
		}
		if match("src/style.css") {
			t.Error("should not match src/style.css")
		}
	})

	t.Run("matches files at root with *.ts pattern", func(t *testing.T) {
		match := CreateGlobMatcher([]string{"*.ts"}, nil)
		if !match("index.ts") {
			t.Error("should match index.ts")
		}
	})

	t.Run("matches with multiple patterns", func(t *testing.T) {
		match := CreateGlobMatcher([]string{"**/*.ts", "**/*.js"}, nil)
		if !match("src/index.ts") {
			t.Error("should match .ts files")
		}
		if !match("lib/util.js") {
			t.Error("should match .js files")
		}
		if match("style.css") {
			t.Error("should not match .css files")
		}
	})

	t.Run("skips dotfiles by default", func(t *testing.T) {
		match := CreateGlobMatcher([]string{"**/*"}, nil)
		if match(".hidden") {
			t.Error("should skip dotfiles by default")
		}
		if match("dir/.env") {
			t.Error("should skip dotfiles in subdirectories")
		}
	})

	t.Run("matches dotfiles when dot option is set", func(t *testing.T) {
		match := CreateGlobMatcher([]string{"**/*"}, &GlobMatcherOptions{Dot: true})
		if !match(".hidden") {
			t.Error("should match dotfiles when dot=true")
		}
		if !match("dir/.env") {
			t.Error("should match dotfiles in subdirectories when dot=true")
		}
	})

	t.Run("normalizes leading ./ from paths", func(t *testing.T) {
		match := CreateGlobMatcher([]string{"**/*.ts"}, nil)
		if !match("./src/index.ts") {
			t.Error("should match paths with leading ./")
		}
	})

	t.Run("normalizes leading / from paths", func(t *testing.T) {
		match := CreateGlobMatcher([]string{"**/*.ts"}, nil)
		if !match("/src/index.ts") {
			t.Error("should match paths with leading /")
		}
	})
}

func TestCreateGlobMatcherSingle(t *testing.T) {
	t.Run("works as a single pattern shorthand", func(t *testing.T) {
		match := CreateGlobMatcherSingle("**/*.md", nil)
		if !match("docs/readme.md") {
			t.Error("should match .md files")
		}
		if match("docs/readme.txt") {
			t.Error("should not match .txt files")
		}
	})
}

func TestMatchGlob(t *testing.T) {
	t.Run("one-off convenience matches correctly", func(t *testing.T) {
		if !MatchGlob("src/index.ts", []string{"**/*.ts"}, nil) {
			t.Error("should match .ts files")
		}
		if MatchGlob("src/index.ts", []string{"**/*.js"}, nil) {
			t.Error("should not match wrong extension")
		}
	})
}

func TestMatchDoublestarEdgeCases(t *testing.T) {
	t.Run("matches prefix/** pattern", func(t *testing.T) {
		match := CreateGlobMatcher([]string{"src/**"}, nil)
		if !match("src/file.ts") {
			t.Error("should match files under src/")
		}
		if !match("src/deep/file.ts") {
			t.Error("should match deeply nested files under src/")
		}
	})

	t.Run("matches left/**/right pattern", func(t *testing.T) {
		match := CreateGlobMatcher([]string{"src/**/test/*.ts"}, nil)
		if !match("src/test/file.ts") {
			t.Error("should match direct test dir")
		}
		if !match("src/deep/test/file.ts") {
			t.Error("should match nested test dir")
		}
	})

	t.Run("** alone matches everything", func(t *testing.T) {
		match := CreateGlobMatcher([]string{"**"}, nil)
		if !match("anything/at/all") {
			t.Error("** should match any path")
		}
	})
}

func TestResolvePathPattern(t *testing.T) {
	// Mock readdir that simulates a directory structure.
	mockReaddir := func(dir string) ([]ReaddirEntry, error) {
		switch dir {
		case "/docs":
			return []ReaddirEntry{
				{Name: "readme.md", Type: "file"},
				{Name: "guide.md", Type: "file"},
				{Name: "images", Type: "directory"},
			}, nil
		case "/docs/images":
			return []ReaddirEntry{
				{Name: "logo.png", Type: "file"},
			}, nil
		default:
			return nil, &FileNotFoundError{}
		}
	}

	t.Run("resolves a plain directory path", func(t *testing.T) {
		entries, err := ResolvePathPattern("/docs", mockReaddir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Path != "/docs" {
			t.Errorf("Path = %q, want %q", entries[0].Path, "/docs")
		}
		if entries[0].Type != "directory" {
			t.Errorf("Type = %q, want %q", entries[0].Type, "directory")
		}
	})

	t.Run("resolves a plain file path", func(t *testing.T) {
		entries, err := ResolvePathPattern("/docs/readme.md", mockReaddir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Type != "file" {
			t.Errorf("Type = %q, want %q", entries[0].Type, "file")
		}
	})

	t.Run("resolves glob pattern matching files", func(t *testing.T) {
		entries, err := ResolvePathPattern("/docs/**/*.md", mockReaddir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) == 0 {
			t.Fatal("expected at least one matching entry")
		}
		for _, e := range entries {
			if e.Type != "file" {
				continue
			}
			if len(e.Path) < 3 || e.Path[len(e.Path)-3:] != ".md" {
				t.Errorf("expected .md file, got %q", e.Path)
			}
		}
	})

	t.Run("strips trailing slash from pattern", func(t *testing.T) {
		entries, err := ResolvePathPattern("/docs/", mockReaddir, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(entries))
		}
		if entries[0].Path != "/docs" {
			t.Errorf("Path = %q, want %q", entries[0].Path, "/docs")
		}
	})
}

func TestWalkAll(t *testing.T) {
	mockReaddir := func(dir string) ([]ReaddirEntry, error) {
		switch dir {
		case "/root":
			return []ReaddirEntry{
				{Name: "file.txt", Type: "file"},
				{Name: "sub", Type: "directory"},
				{Name: "link", Type: "directory", IsSymlink: true},
			}, nil
		case "/root/sub":
			return []ReaddirEntry{
				{Name: "nested.txt", Type: "file"},
			}, nil
		default:
			return nil, &FileNotFoundError{}
		}
	}

	t.Run("walks directory tree", func(t *testing.T) {
		entries := walkAll(mockReaddir, "/root", 0, 10)
		if len(entries) < 3 {
			t.Fatalf("expected at least 3 entries (file, dir, nested file), got %d", len(entries))
		}
	})

	t.Run("skips symlinked directories", func(t *testing.T) {
		entries := walkAll(mockReaddir, "/root", 0, 10)
		for _, e := range entries {
			if e.Path == "/root/link" {
				t.Error("should skip symlinked directories")
			}
		}
	})

	t.Run("respects max depth", func(t *testing.T) {
		entries := walkAll(mockReaddir, "/root", 0, 1)
		for _, e := range entries {
			if e.Path == "/root/sub/nested.txt" {
				t.Error("should not recurse beyond maxDepth")
			}
		}
	})
}
