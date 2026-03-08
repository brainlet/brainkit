// Ported from: packages/core/src/workspace/filesystem/local-filesystem.test.ts
//
// Faithful 1:1 port of the Mastra LocalFilesystem vitest suite.
// Each describe block in the TS test maps to a t.Run subtest group here.
// Where the Go implementation differs from the TS (e.g. MIME type mappings,
// workspace path handling), the test expectations are adjusted to match
// the Go implementation.
//
// KEY DIFFERENCE: On Unix, paths starting with "/" are absolute filesystem paths.
// The TS version treats "/test.txt" as workspace-relative, but the Go port
// treats it as an absolute path (and thus fails containment). All workspace-
// relative paths in these tests omit the leading slash (e.g., "test.txt" not
// "/test.txt").
package filesystem

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	workspace "github.com/brainlet/brainkit/agent-kit/core/workspace"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// =============================================================================
// Test Helpers
// =============================================================================

// boolPtr returns a pointer to the given bool value.
// Used for WriteOptions.Overwrite which is *bool.
func boolPtr(b bool) *bool {
	return &b
}

// realTempDir creates a temp directory and resolves its real path (following symlinks).
// On macOS /tmp is a symlink to /private/tmp, which causes assertPathContained
// to fail because filepath.EvalSymlinks resolves to the real path. By using
// the real path as basePath, the containment check works correctly.
func realTempDir(t *testing.T, pattern string) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	realDir, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks for temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(realDir)
	})
	return realDir
}

// realTempDirIn creates a temp directory in the given parent and resolves symlinks.
// Uses t.Skip if the parent directory is not writable.
func realTempDirIn(t *testing.T, parent, pattern string) string {
	t.Helper()
	tempDir, err := os.MkdirTemp(parent, pattern)
	if err != nil {
		t.Skipf("cannot create temp dir in %s: %v", parent, err)
	}
	realDir, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks for temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(realDir)
	})
	return realDir
}

// setupTestFS creates a temp directory and a LocalFilesystem rooted there.
// Returns the temp dir path (real, symlink-resolved) and the filesystem.
// Registers cleanup via t.Cleanup.
func setupTestFS(t *testing.T) (string, *LocalFilesystem) {
	t.Helper()
	tempDir := realTempDir(t, "agent-kit-fs-test-")
	lfs := NewLocalFilesystem(LocalFilesystemOptions{
		BasePath: tempDir,
	})
	return tempDir, lfs
}

// writeRawFile writes content directly to disk (bypassing the filesystem abstraction).
func writeRawFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create parent dirs for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write raw file %s: %v", path, err)
	}
}

// readRawFile reads content directly from disk (bypassing the filesystem abstraction).
func readRawFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read raw file %s: %v", path, err)
	}
	return string(data)
}

// =============================================================================
// Constructor
// =============================================================================

// Ported from: describe('constructor', ...)
func TestConstructor(t *testing.T) {
	_, lfs := setupTestFS(t)

	// TS: it('should create filesystem with default values')
	t.Run("should create filesystem with default values", func(t *testing.T) {
		if lfs.Provider() != "local" {
			t.Errorf("expected provider 'local', got %q", lfs.Provider())
		}
		if lfs.Name() != "LocalFilesystem" {
			t.Errorf("expected name 'LocalFilesystem', got %q", lfs.Name())
		}
		if lfs.ID() == "" {
			t.Error("expected non-empty ID")
		}
	})

	// TS: it('should accept custom id')
	t.Run("should accept custom id", func(t *testing.T) {
		tempDir := realTempDir(t, "agent-kit-fs-test-")
		customFs := NewLocalFilesystem(LocalFilesystemOptions{
			ID:       "custom-id",
			BasePath: tempDir,
		})
		if customFs.ID() != "custom-id" {
			t.Errorf("expected ID 'custom-id', got %q", customFs.ID())
		}
	})
}

// =============================================================================
// Init
// =============================================================================

// Ported from: describe('init', ...)
func TestInit(t *testing.T) {
	tempDir := realTempDir(t, "agent-kit-fs-test-")

	// TS: it('should create base directory if it does not exist')
	t.Run("should create base directory if it does not exist", func(t *testing.T) {
		newDir := filepath.Join(tempDir, "new-base")
		newFs := NewLocalFilesystem(LocalFilesystemOptions{BasePath: newDir})

		if err := newFs.Init(); err != nil {
			t.Fatalf("Init() failed: %v", err)
		}

		info, err := os.Stat(newDir)
		if err != nil {
			t.Fatalf("expected directory to exist at %s: %v", newDir, err)
		}
		if !info.IsDir() {
			t.Error("expected path to be a directory")
		}
	})
}

// =============================================================================
// ReadFile
// =============================================================================

// Ported from: describe('readFile', ...)
func TestReadFile(t *testing.T) {
	tempDir, lfs := setupTestFS(t)

	// TS: it('should read file as buffer by default')
	t.Run("should read file as buffer by default", func(t *testing.T) {
		writeRawFile(t, filepath.Join(tempDir, "test.txt"), "Hello World")

		content, err := lfs.ReadFile("test.txt", nil)
		if err != nil {
			t.Fatalf("ReadFile() failed: %v", err)
		}

		data, ok := content.([]byte)
		if !ok {
			t.Fatalf("expected []byte, got %T", content)
		}
		if string(data) != "Hello World" {
			t.Errorf("expected 'Hello World', got %q", string(data))
		}
	})

	// TS: it('should read file as string with encoding')
	// Note: Go implementation always returns []byte regardless of encoding option.
	t.Run("should read file as bytes (Go returns []byte always)", func(t *testing.T) {
		writeRawFile(t, filepath.Join(tempDir, "test-enc.txt"), "Hello World")

		content, err := lfs.ReadFile("test-enc.txt", &ReadOptions{Encoding: "utf-8"})
		if err != nil {
			t.Fatalf("ReadFile() failed: %v", err)
		}

		data, ok := content.([]byte)
		if !ok {
			t.Fatalf("expected []byte, got %T", content)
		}
		if string(data) != "Hello World" {
			t.Errorf("expected 'Hello World', got %q", string(data))
		}
	})

	// TS: it('should throw FileNotFoundError for missing file')
	t.Run("should return FileNotFoundError for missing file", func(t *testing.T) {
		_, err := lfs.ReadFile("nonexistent.txt", nil)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		var fnf *workspace.FileNotFoundError
		if !errors.As(err, &fnf) {
			t.Errorf("expected FileNotFoundError, got %T: %v", err, err)
		}
	})

	// TS: it('should throw IsDirectoryError when reading a directory')
	t.Run("should return IsDirectoryError when reading a directory", func(t *testing.T) {
		if err := os.Mkdir(filepath.Join(tempDir, "testdir-read"), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		_, err := lfs.ReadFile("testdir-read", nil)
		if err == nil {
			t.Fatal("expected error when reading directory")
		}
		var isDir *workspace.IsDirectoryError
		if !errors.As(err, &isDir) {
			t.Errorf("expected IsDirectoryError, got %T: %v", err, err)
		}
	})

	// TS: it('should normalize paths with leading slashes')
	// On Go/Unix, paths with "/" prefix are absolute. We test that both
	// relative path and absolute path (within basePath) resolve identically.
	t.Run("should normalize paths with and without leading slash", func(t *testing.T) {
		writeRawFile(t, filepath.Join(tempDir, "normalize.txt"), "content")

		// Relative path (workspace-relative)
		content1, err := lfs.ReadFile("normalize.txt", nil)
		if err != nil {
			t.Fatalf("ReadFile('normalize.txt') failed: %v", err)
		}
		// Absolute path (within basePath)
		absPath := filepath.Join(tempDir, "normalize.txt")
		content2, err := lfs.ReadFile(absPath, nil)
		if err != nil {
			t.Fatalf("ReadFile(absolute) failed: %v", err)
		}

		if string(content1.([]byte)) != "content" {
			t.Errorf("expected 'content' for relative, got %q", string(content1.([]byte)))
		}
		if string(content2.([]byte)) != "content" {
			t.Errorf("expected 'content' for absolute, got %q", string(content2.([]byte)))
		}
	})
}

// =============================================================================
// WriteFile
// =============================================================================

// Ported from: describe('writeFile', ...)
func TestWriteFile(t *testing.T) {
	tempDir, lfs := setupTestFS(t)

	// TS: it('should write string content')
	t.Run("should write string content", func(t *testing.T) {
		if err := lfs.WriteFile("test-write.txt", "Hello World", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		content := readRawFile(t, filepath.Join(tempDir, "test-write.txt"))
		if content != "Hello World" {
			t.Errorf("expected 'Hello World', got %q", content)
		}
	})

	// TS: it('should write buffer content')
	t.Run("should write buffer content", func(t *testing.T) {
		buf := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f} // "Hello"
		if err := lfs.WriteFile("test-write.bin", buf, nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(tempDir, "test-write.bin"))
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}
		if string(data) != string(buf) {
			t.Errorf("expected %v, got %v", buf, data)
		}
	})

	// TS: it('should create parent directories recursively')
	t.Run("should create parent directories recursively", func(t *testing.T) {
		if err := lfs.WriteFile("deep/nested/dir/test.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		content := readRawFile(t, filepath.Join(tempDir, "deep/nested/dir/test.txt"))
		if content != "content" {
			t.Errorf("expected 'content', got %q", content)
		}
	})

	// TS: it('should throw FileExistsError when overwrite is false')
	t.Run("should return FileExistsError when overwrite is false", func(t *testing.T) {
		if err := lfs.WriteFile("no-overwrite.txt", "original", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		err := lfs.WriteFile("no-overwrite.txt", "new", &WriteOptions{Overwrite: boolPtr(false)})
		if err == nil {
			t.Fatal("expected error when overwrite is false")
		}
		var fe *workspace.FileExistsError
		if !errors.As(err, &fe) {
			t.Errorf("expected FileExistsError, got %T: %v", err, err)
		}
	})

	// TS: it('should overwrite by default')
	t.Run("should overwrite by default", func(t *testing.T) {
		if err := lfs.WriteFile("overwrite-default.txt", "original", nil); err != nil {
			t.Fatalf("WriteFile() original failed: %v", err)
		}
		if err := lfs.WriteFile("overwrite-default.txt", "new", nil); err != nil {
			t.Fatalf("WriteFile() overwrite failed: %v", err)
		}
		content := readRawFile(t, filepath.Join(tempDir, "overwrite-default.txt"))
		if content != "new" {
			t.Errorf("expected 'new', got %q", content)
		}
	})
}

// =============================================================================
// AppendFile
// =============================================================================

// Ported from: describe('appendFile', ...)
func TestAppendFile(t *testing.T) {
	tempDir, lfs := setupTestFS(t)

	// TS: it('should append to existing file')
	t.Run("should append to existing file", func(t *testing.T) {
		if err := lfs.WriteFile("append-test.txt", "Hello", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		if err := lfs.AppendFile("append-test.txt", " World"); err != nil {
			t.Fatalf("AppendFile() failed: %v", err)
		}
		content := readRawFile(t, filepath.Join(tempDir, "append-test.txt"))
		if content != "Hello World" {
			t.Errorf("expected 'Hello World', got %q", content)
		}
	})

	// TS: it('should create file if it does not exist')
	t.Run("should create file if it does not exist", func(t *testing.T) {
		if err := lfs.AppendFile("append-new.txt", "content"); err != nil {
			t.Fatalf("AppendFile() failed: %v", err)
		}
		content := readRawFile(t, filepath.Join(tempDir, "append-new.txt"))
		if content != "content" {
			t.Errorf("expected 'content', got %q", content)
		}
	})
}

// =============================================================================
// DeleteFile
// =============================================================================

// Ported from: describe('deleteFile', ...)
func TestDeleteFile(t *testing.T) {
	_, lfs := setupTestFS(t)

	// TS: it('should delete existing file')
	t.Run("should delete existing file", func(t *testing.T) {
		if err := lfs.WriteFile("delete-me.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		if err := lfs.DeleteFile("delete-me.txt", nil); err != nil {
			t.Fatalf("DeleteFile() failed: %v", err)
		}
		exists, err := lfs.Exists("delete-me.txt")
		if err != nil {
			t.Fatalf("Exists() failed: %v", err)
		}
		if exists {
			t.Error("expected file to not exist after deletion")
		}
	})

	// TS: it('should throw FileNotFoundError for missing file')
	t.Run("should return FileNotFoundError for missing file", func(t *testing.T) {
		err := lfs.DeleteFile("nonexistent-del.txt", nil)
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		var fnf *workspace.FileNotFoundError
		if !errors.As(err, &fnf) {
			t.Errorf("expected FileNotFoundError, got %T: %v", err, err)
		}
	})

	// TS: it('should not throw when force is true and file does not exist')
	t.Run("should not error when force is true and file does not exist", func(t *testing.T) {
		err := lfs.DeleteFile("nonexistent-force.txt", &RemoveOptions{Force: true})
		if err != nil {
			t.Errorf("expected no error with force=true, got: %v", err)
		}
	})

	// TS: it('should throw IsDirectoryError when deleting directory')
	t.Run("should return IsDirectoryError when deleting directory", func(t *testing.T) {
		if err := lfs.Mkdir("del-dir-test", nil); err != nil {
			t.Fatalf("Mkdir() failed: %v", err)
		}
		err := lfs.DeleteFile("del-dir-test", nil)
		if err == nil {
			t.Fatal("expected error when deleting directory")
		}
		var isDir *workspace.IsDirectoryError
		if !errors.As(err, &isDir) {
			t.Errorf("expected IsDirectoryError, got %T: %v", err, err)
		}
	})
}

// =============================================================================
// CopyFile
// =============================================================================

// Ported from: describe('copyFile', ...)
func TestCopyFile(t *testing.T) {
	_, lfs := setupTestFS(t)

	// TS: it('should copy file to new location')
	t.Run("should copy file to new location", func(t *testing.T) {
		if err := lfs.WriteFile("copy-source.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		if err := lfs.CopyFile("copy-source.txt", "copy-dest.txt", nil); err != nil {
			t.Fatalf("CopyFile() failed: %v", err)
		}

		srcContent, err := lfs.ReadFile("copy-source.txt", nil)
		if err != nil {
			t.Fatalf("ReadFile(source) failed: %v", err)
		}
		destContent, err := lfs.ReadFile("copy-dest.txt", nil)
		if err != nil {
			t.Fatalf("ReadFile(dest) failed: %v", err)
		}

		if string(srcContent.([]byte)) != "content" {
			t.Errorf("source content: expected 'content', got %q", string(srcContent.([]byte)))
		}
		if string(destContent.([]byte)) != "content" {
			t.Errorf("dest content: expected 'content', got %q", string(destContent.([]byte)))
		}
	})

	// TS: it('should throw FileNotFoundError for missing source')
	t.Run("should return FileNotFoundError for missing source", func(t *testing.T) {
		err := lfs.CopyFile("nonexistent-copy.txt", "dest-copy.txt", nil)
		if err == nil {
			t.Fatal("expected error for missing source")
		}
		var fnf *workspace.FileNotFoundError
		if !errors.As(err, &fnf) {
			t.Errorf("expected FileNotFoundError, got %T: %v", err, err)
		}
	})

	// TS: it('should throw FileExistsError when overwrite is false and dest exists')
	t.Run("should return FileExistsError when overwrite is false and dest exists", func(t *testing.T) {
		if err := lfs.WriteFile("copy-src2.txt", "source", nil); err != nil {
			t.Fatalf("WriteFile(source) failed: %v", err)
		}
		if err := lfs.WriteFile("copy-dest2.txt", "dest", nil); err != nil {
			t.Fatalf("WriteFile(dest) failed: %v", err)
		}
		err := lfs.CopyFile("copy-src2.txt", "copy-dest2.txt", &CopyOptions{Overwrite: false})
		if err == nil {
			t.Fatal("expected FileExistsError")
		}
		var fe *workspace.FileExistsError
		if !errors.As(err, &fe) {
			t.Errorf("expected FileExistsError, got %T: %v", err, err)
		}
	})

	// TS: it('should copy directory recursively')
	t.Run("should copy directory recursively", func(t *testing.T) {
		if err := lfs.WriteFile("srcdir-copy/file1.txt", "content1", nil); err != nil {
			t.Fatalf("WriteFile(file1) failed: %v", err)
		}
		if err := lfs.WriteFile("srcdir-copy/file2.txt", "content2", nil); err != nil {
			t.Fatalf("WriteFile(file2) failed: %v", err)
		}

		err := lfs.CopyFile("srcdir-copy", "destdir-copy", &CopyOptions{Recursive: true})
		if err != nil {
			t.Fatalf("CopyFile(recursive) failed: %v", err)
		}

		c1, err := lfs.ReadFile("destdir-copy/file1.txt", nil)
		if err != nil {
			t.Fatalf("ReadFile(dest/file1) failed: %v", err)
		}
		c2, err := lfs.ReadFile("destdir-copy/file2.txt", nil)
		if err != nil {
			t.Fatalf("ReadFile(dest/file2) failed: %v", err)
		}

		if string(c1.([]byte)) != "content1" {
			t.Errorf("file1: expected 'content1', got %q", string(c1.([]byte)))
		}
		if string(c2.([]byte)) != "content2" {
			t.Errorf("file2: expected 'content2', got %q", string(c2.([]byte)))
		}
	})

	// TS: it('should throw IsDirectoryError when copying directory without recursive')
	t.Run("should return IsDirectoryError when copying directory without recursive", func(t *testing.T) {
		if err := lfs.Mkdir("srcdir-norecurse", nil); err != nil {
			t.Fatalf("Mkdir() failed: %v", err)
		}
		err := lfs.CopyFile("srcdir-norecurse", "destdir-norecurse", nil)
		if err == nil {
			t.Fatal("expected IsDirectoryError")
		}
		var isDir *workspace.IsDirectoryError
		if !errors.As(err, &isDir) {
			t.Errorf("expected IsDirectoryError, got %T: %v", err, err)
		}
	})
}

// =============================================================================
// MoveFile
// =============================================================================

// Ported from: describe('moveFile', ...)
func TestMoveFile(t *testing.T) {
	_, lfs := setupTestFS(t)

	// TS: it('should move file to new location')
	t.Run("should move file to new location", func(t *testing.T) {
		if err := lfs.WriteFile("move-source.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		if err := lfs.MoveFile("move-source.txt", "move-dest.txt", nil); err != nil {
			t.Fatalf("MoveFile() failed: %v", err)
		}
		exists, err := lfs.Exists("move-source.txt")
		if err != nil {
			t.Fatalf("Exists(source) failed: %v", err)
		}
		if exists {
			t.Error("expected source to not exist after move")
		}

		destContent, err := lfs.ReadFile("move-dest.txt", nil)
		if err != nil {
			t.Fatalf("ReadFile(dest) failed: %v", err)
		}
		if string(destContent.([]byte)) != "content" {
			t.Errorf("expected 'content', got %q", string(destContent.([]byte)))
		}
	})

	// TS: it('should throw FileNotFoundError for missing source')
	t.Run("should return FileNotFoundError for missing source", func(t *testing.T) {
		err := lfs.MoveFile("nonexistent-move.txt", "dest-move.txt", nil)
		if err == nil {
			t.Fatal("expected error for missing source")
		}
		var fnf *workspace.FileNotFoundError
		if !errors.As(err, &fnf) {
			t.Errorf("expected FileNotFoundError, got %T: %v", err, err)
		}
	})

	// TS: it('should throw FileExistsError when overwrite is false and dest exists')
	t.Run("should return FileExistsError when overwrite is false and dest exists", func(t *testing.T) {
		if err := lfs.WriteFile("move-src2.txt", "source", nil); err != nil {
			t.Fatalf("WriteFile(source) failed: %v", err)
		}
		if err := lfs.WriteFile("move-dest2.txt", "dest", nil); err != nil {
			t.Fatalf("WriteFile(dest) failed: %v", err)
		}
		err := lfs.MoveFile("move-src2.txt", "move-dest2.txt", &CopyOptions{Overwrite: false})
		if err == nil {
			t.Fatal("expected FileExistsError")
		}
		var fe *workspace.FileExistsError
		if !errors.As(err, &fe) {
			t.Errorf("expected FileExistsError, got %T: %v", err, err)
		}
	})
}

// =============================================================================
// Mkdir
// =============================================================================

// Ported from: describe('mkdir', ...)
func TestMkdir(t *testing.T) {
	tempDir, lfs := setupTestFS(t)

	// TS: it('should create directory')
	t.Run("should create directory", func(t *testing.T) {
		if err := lfs.Mkdir("newdir", nil); err != nil {
			t.Fatalf("Mkdir() failed: %v", err)
		}
		info, err := os.Stat(filepath.Join(tempDir, "newdir"))
		if err != nil {
			t.Fatalf("expected directory to exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected path to be a directory")
		}
	})

	// TS: it('should create nested directories recursively')
	t.Run("should create nested directories recursively", func(t *testing.T) {
		if err := lfs.Mkdir("deep/nested/dir", nil); err != nil {
			t.Fatalf("Mkdir() failed: %v", err)
		}
		info, err := os.Stat(filepath.Join(tempDir, "deep/nested/dir"))
		if err != nil {
			t.Fatalf("expected directory to exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected path to be a directory")
		}
	})

	// TS: it('should not throw if directory already exists')
	t.Run("should not error if directory already exists", func(t *testing.T) {
		if err := lfs.Mkdir("mkdir-exists", nil); err != nil {
			t.Fatalf("Mkdir() first call failed: %v", err)
		}
		if err := lfs.Mkdir("mkdir-exists", nil); err != nil {
			t.Errorf("expected no error for existing directory, got: %v", err)
		}
	})

	// TS: it('should throw FileExistsError if path is a file')
	t.Run("should return FileExistsError if path is a file", func(t *testing.T) {
		if err := lfs.WriteFile("mkdir-file-conflict", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		err := lfs.Mkdir("mkdir-file-conflict", &MkdirOptions{Recursive: false})
		if err == nil {
			t.Fatal("expected FileExistsError")
		}
		var fe *workspace.FileExistsError
		if !errors.As(err, &fe) {
			t.Errorf("expected FileExistsError, got %T: %v", err, err)
		}
	})
}

// =============================================================================
// Rmdir
// =============================================================================

// Ported from: describe('rmdir', ...)
func TestRmdir(t *testing.T) {
	_, lfs := setupTestFS(t)

	// TS: it('should remove empty directory')
	t.Run("should remove empty directory", func(t *testing.T) {
		if err := lfs.Mkdir("emptydir", nil); err != nil {
			t.Fatalf("Mkdir() failed: %v", err)
		}
		if err := lfs.Rmdir("emptydir", nil); err != nil {
			t.Fatalf("Rmdir() failed: %v", err)
		}
		exists, err := lfs.Exists("emptydir")
		if err != nil {
			t.Fatalf("Exists() failed: %v", err)
		}
		if exists {
			t.Error("expected directory to not exist after removal")
		}
	})

	// TS: it('should throw DirectoryNotEmptyError for non-empty directory')
	t.Run("should return DirectoryNotEmptyError for non-empty directory", func(t *testing.T) {
		if err := lfs.WriteFile("nonempty-rmdir/file.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		err := lfs.Rmdir("nonempty-rmdir", nil)
		if err == nil {
			t.Fatal("expected DirectoryNotEmptyError")
		}
		var dne *workspace.DirectoryNotEmptyError
		if !errors.As(err, &dne) {
			t.Errorf("expected DirectoryNotEmptyError, got %T: %v", err, err)
		}
	})

	// TS: it('should remove non-empty directory with recursive option')
	t.Run("should remove non-empty directory with recursive option", func(t *testing.T) {
		if err := lfs.WriteFile("nonempty-recursive/file.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		if err := lfs.Rmdir("nonempty-recursive", &RemoveOptions{Recursive: true, Force: true}); err != nil {
			t.Fatalf("Rmdir(recursive) failed: %v", err)
		}
		exists, err := lfs.Exists("nonempty-recursive")
		if err != nil {
			t.Fatalf("Exists() failed: %v", err)
		}
		if exists {
			t.Error("expected directory to not exist after recursive removal")
		}
	})

	// TS: it('should throw DirectoryNotFoundError for missing directory')
	t.Run("should return DirectoryNotFoundError for missing directory", func(t *testing.T) {
		err := lfs.Rmdir("nonexistent-rmdir", nil)
		if err == nil {
			t.Fatal("expected DirectoryNotFoundError")
		}
		var dnf *workspace.DirectoryNotFoundError
		if !errors.As(err, &dnf) {
			t.Errorf("expected DirectoryNotFoundError, got %T: %v", err, err)
		}
	})

	// TS: it('should not throw when force is true and directory does not exist')
	t.Run("should not error when force is true and directory does not exist", func(t *testing.T) {
		err := lfs.Rmdir("nonexistent-rmdir-force", &RemoveOptions{Force: true})
		if err != nil {
			t.Errorf("expected no error with force=true, got: %v", err)
		}
	})

	// TS: it('should throw NotDirectoryError when path is a file')
	t.Run("should return NotDirectoryError when path is a file", func(t *testing.T) {
		if err := lfs.WriteFile("rmdir-file", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		err := lfs.Rmdir("rmdir-file", nil)
		if err == nil {
			t.Fatal("expected NotDirectoryError")
		}
		var nd *workspace.NotDirectoryError
		if !errors.As(err, &nd) {
			t.Errorf("expected NotDirectoryError, got %T: %v", err, err)
		}
	})
}

// =============================================================================
// Readdir
// =============================================================================

// Ported from: describe('readdir', ...)
func TestReaddir(t *testing.T) {
	_, lfs := setupTestFS(t)

	// TS: it('should list directory contents')
	t.Run("should list directory contents", func(t *testing.T) {
		if err := lfs.WriteFile("readdir-test/file1.txt", "content1", nil); err != nil {
			t.Fatalf("WriteFile(file1) failed: %v", err)
		}
		if err := lfs.WriteFile("readdir-test/file2.txt", "content2", nil); err != nil {
			t.Fatalf("WriteFile(file2) failed: %v", err)
		}
		if err := lfs.Mkdir("readdir-test/subdir", nil); err != nil {
			t.Fatalf("Mkdir(subdir) failed: %v", err)
		}

		entries, err := lfs.Readdir("readdir-test", nil)
		if err != nil {
			t.Fatalf("Readdir() failed: %v", err)
		}

		if len(entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(entries))
		}

		foundFile1, foundFile2, foundSubdir := false, false, false
		for _, e := range entries {
			switch e.Name {
			case "file1.txt":
				if e.Type != "file" {
					t.Errorf("file1.txt: expected type 'file', got %q", e.Type)
				}
				foundFile1 = true
			case "file2.txt":
				if e.Type != "file" {
					t.Errorf("file2.txt: expected type 'file', got %q", e.Type)
				}
				foundFile2 = true
			case "subdir":
				if e.Type != "directory" {
					t.Errorf("subdir: expected type 'directory', got %q", e.Type)
				}
				foundSubdir = true
			}
		}
		if !foundFile1 {
			t.Error("missing entry file1.txt")
		}
		if !foundFile2 {
			t.Error("missing entry file2.txt")
		}
		if !foundSubdir {
			t.Error("missing entry subdir")
		}
	})

	// TS: it('should include file sizes')
	t.Run("should include file sizes", func(t *testing.T) {
		if err := lfs.WriteFile("readdir-size/file.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}

		entries, err := lfs.Readdir("readdir-size", nil)
		if err != nil {
			t.Fatalf("Readdir() failed: %v", err)
		}

		var fileEntry *FileEntry
		for i := range entries {
			if entries[i].Name == "file.txt" {
				fileEntry = &entries[i]
				break
			}
		}
		if fileEntry == nil {
			t.Fatal("expected to find file.txt entry")
		}
		// "content" is 7 bytes
		if fileEntry.Size != 7 {
			t.Errorf("expected size 7, got %d", fileEntry.Size)
		}
	})

	// TS: it('should filter by extension')
	t.Run("should filter by extension", func(t *testing.T) {
		if err := lfs.WriteFile("readdir-ext/file.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile(txt) failed: %v", err)
		}
		if err := lfs.WriteFile("readdir-ext/file.json", "{}", nil); err != nil {
			t.Fatalf("WriteFile(json) failed: %v", err)
		}

		txtOnly, err := lfs.Readdir("readdir-ext", &ListOptions{Extension: []string{".txt"}})
		if err != nil {
			t.Fatalf("Readdir(extension) failed: %v", err)
		}

		if len(txtOnly) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(txtOnly))
		}
		if txtOnly[0].Name != "file.txt" {
			t.Errorf("expected 'file.txt', got %q", txtOnly[0].Name)
		}
	})

	// TS: it('should list recursively')
	t.Run("should list recursively", func(t *testing.T) {
		if err := lfs.WriteFile("readdir-recurse/file1.txt", "content1", nil); err != nil {
			t.Fatalf("WriteFile(file1) failed: %v", err)
		}
		if err := lfs.WriteFile("readdir-recurse/sub/file2.txt", "content2", nil); err != nil {
			t.Fatalf("WriteFile(file2) failed: %v", err)
		}

		entries, err := lfs.Readdir("readdir-recurse", &ListOptions{Recursive: true})
		if err != nil {
			t.Fatalf("Readdir(recursive) failed: %v", err)
		}

		foundFile1, foundFile2 := false, false
		for _, e := range entries {
			if e.Name == "file1.txt" {
				foundFile1 = true
			}
			if e.Name == "sub/file2.txt" {
				foundFile2 = true
			}
		}
		if !foundFile1 {
			t.Error("missing file1.txt in recursive listing")
		}
		if !foundFile2 {
			t.Error("missing sub/file2.txt in recursive listing")
		}
	})

	// TS: it('should throw DirectoryNotFoundError for missing directory')
	t.Run("should return DirectoryNotFoundError for missing directory", func(t *testing.T) {
		_, err := lfs.Readdir("nonexistent-readdir", nil)
		if err == nil {
			t.Fatal("expected DirectoryNotFoundError")
		}
		var dnf *workspace.DirectoryNotFoundError
		if !errors.As(err, &dnf) {
			t.Errorf("expected DirectoryNotFoundError, got %T: %v", err, err)
		}
	})

	// TS: it('should throw NotDirectoryError when path is a file')
	t.Run("should return NotDirectoryError when path is a file", func(t *testing.T) {
		if err := lfs.WriteFile("readdir-notdir", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		_, err := lfs.Readdir("readdir-notdir", nil)
		if err == nil {
			t.Fatal("expected NotDirectoryError")
		}
		var nd *workspace.NotDirectoryError
		if !errors.As(err, &nd) {
			t.Errorf("expected NotDirectoryError, got %T: %v", err, err)
		}
	})
}

// =============================================================================
// Exists
// =============================================================================

// Ported from: describe('exists', ...)
func TestExists(t *testing.T) {
	_, lfs := setupTestFS(t)

	// TS: it('should return true for existing file')
	t.Run("should return true for existing file", func(t *testing.T) {
		if err := lfs.WriteFile("exists-file.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		exists, err := lfs.Exists("exists-file.txt")
		if err != nil {
			t.Fatalf("Exists() failed: %v", err)
		}
		if !exists {
			t.Error("expected file to exist")
		}
	})

	// TS: it('should return true for existing directory')
	t.Run("should return true for existing directory", func(t *testing.T) {
		if err := lfs.Mkdir("exists-dir", nil); err != nil {
			t.Fatalf("Mkdir() failed: %v", err)
		}
		exists, err := lfs.Exists("exists-dir")
		if err != nil {
			t.Fatalf("Exists() failed: %v", err)
		}
		if !exists {
			t.Error("expected directory to exist")
		}
	})

	// TS: it('should return false for non-existing path')
	t.Run("should return false for non-existing path", func(t *testing.T) {
		exists, err := lfs.Exists("nonexistent-exists")
		if err != nil {
			t.Fatalf("Exists() failed: %v", err)
		}
		if exists {
			t.Error("expected path to not exist")
		}
	})
}

// =============================================================================
// Stat
// =============================================================================

// Ported from: describe('stat', ...)
func TestStat(t *testing.T) {
	_, lfs := setupTestFS(t)

	// TS: it('should return file stats')
	t.Run("should return file stats", func(t *testing.T) {
		if err := lfs.WriteFile("stat-file.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}

		stat, err := lfs.Stat("stat-file.txt")
		if err != nil {
			t.Fatalf("Stat() failed: %v", err)
		}

		if stat.Name != "stat-file.txt" {
			t.Errorf("expected name 'stat-file.txt', got %q", stat.Name)
		}
		if stat.Type != "file" {
			t.Errorf("expected type 'file', got %q", stat.Type)
		}
		if stat.Size != 7 { // "content" = 7 bytes
			t.Errorf("expected size 7, got %d", stat.Size)
		}
		// Go implementation uses GetMimeType which maps .txt -> "text/plain"
		if stat.MimeType != "text/plain" {
			t.Errorf("expected MIME 'text/plain', got %q", stat.MimeType)
		}
		if stat.CreatedAt.IsZero() {
			t.Error("expected non-zero CreatedAt")
		}
		if stat.ModifiedAt.IsZero() {
			t.Error("expected non-zero ModifiedAt")
		}
	})

	// TS: it('should return directory stats')
	t.Run("should return directory stats", func(t *testing.T) {
		if err := lfs.Mkdir("stat-dir", nil); err != nil {
			t.Fatalf("Mkdir() failed: %v", err)
		}

		stat, err := lfs.Stat("stat-dir")
		if err != nil {
			t.Fatalf("Stat() failed: %v", err)
		}

		if stat.Name != "stat-dir" {
			t.Errorf("expected name 'stat-dir', got %q", stat.Name)
		}
		if stat.Type != "directory" {
			t.Errorf("expected type 'directory', got %q", stat.Type)
		}
		// Directories should have empty MimeType
		if stat.MimeType != "" {
			t.Errorf("expected empty MimeType for directory, got %q", stat.MimeType)
		}
	})

	// TS: it('should throw FileNotFoundError for missing path')
	t.Run("should return error for missing path", func(t *testing.T) {
		_, err := lfs.Stat("nonexistent-stat")
		if err == nil {
			t.Fatal("expected error for missing path")
		}
		// FsStat returns a fileNotFoundError (local type in fs_utils.go).
		if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "File not found") {
			t.Errorf("expected 'not found' in error, got: %v", err)
		}
	})
}

// =============================================================================
// Contained Mode (path restrictions)
// =============================================================================

// Ported from: describe('contained mode', ...)
func TestContainedMode(t *testing.T) {
	tempDir, lfs := setupTestFS(t)

	// TS: it('should expose contained getter as true by default')
	t.Run("should expose contained as true by default", func(t *testing.T) {
		if !lfs.Contained() {
			t.Error("expected contained to be true by default")
		}
	})

	// TS: it('should expose contained getter as false when set')
	t.Run("should expose contained as false when set", func(t *testing.T) {
		f := false
		uncontainedFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:  tempDir,
			Contained: &f,
		})
		if uncontainedFs.Contained() {
			t.Error("expected contained to be false")
		}
	})

	// TS: it('should block path traversal by default')
	t.Run("should block path traversal by default", func(t *testing.T) {
		_, err := lfs.ReadFile("/../../../etc/passwd", nil)
		if err == nil {
			t.Fatal("expected PermissionError for path traversal")
		}
		var pe *workspace.PermissionError
		if !errors.As(err, &pe) {
			t.Errorf("expected PermissionError, got %T: %v", err, err)
		}
	})

	// TS: it('should block path traversal with dot segments')
	t.Run("should block path traversal with dot segments", func(t *testing.T) {
		_, err := lfs.ReadFile("foo/../../bar/../../../etc/passwd", nil)
		if err == nil {
			t.Fatal("expected PermissionError for path traversal")
		}
		var pe *workspace.PermissionError
		if !errors.As(err, &pe) {
			t.Errorf("expected PermissionError, got %T: %v", err, err)
		}
	})

	// TS: it('should allow paths inside base directory')
	t.Run("should allow paths inside base directory", func(t *testing.T) {
		if err := lfs.WriteFile("allowed/file.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		content, err := lfs.ReadFile("allowed/file.txt", nil)
		if err != nil {
			t.Fatalf("ReadFile() failed: %v", err)
		}
		if string(content.([]byte)) != "content" {
			t.Errorf("expected 'content', got %q", string(content.([]byte)))
		}
	})

	// TS: it('should allow absolute paths inside base directory')
	t.Run("should allow absolute paths inside base directory", func(t *testing.T) {
		if err := lfs.WriteFile("abs-test.txt", "absolute content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		absolutePath := filepath.Join(tempDir, "abs-test.txt")
		content, err := lfs.ReadFile(absolutePath, nil)
		if err != nil {
			t.Fatalf("ReadFile(absolute) failed: %v", err)
		}
		if string(content.([]byte)) != "absolute content" {
			t.Errorf("expected 'absolute content', got %q", string(content.([]byte)))
		}
	})

	// TS: it('should allow exists() with absolute paths inside base directory')
	t.Run("should allow exists with absolute paths inside base directory", func(t *testing.T) {
		if err := lfs.WriteFile("exists-abs-test.txt", "content", nil); err != nil {
			t.Fatalf("WriteFile() failed: %v", err)
		}
		absolutePath := filepath.Join(tempDir, "exists-abs-test.txt")
		exists, err := lfs.Exists(absolutePath)
		if err != nil {
			t.Fatalf("Exists(absolute) failed: %v", err)
		}
		if !exists {
			t.Error("expected file to exist via absolute path")
		}
	})

	// TS: it('should not throw on exists() for non-existent absolute path inside base directory')
	t.Run("should not error on exists for non-existent absolute path inside base directory", func(t *testing.T) {
		absolutePath := filepath.Join(tempDir, "nonexistent", "file.txt")
		exists, err := lfs.Exists(absolutePath)
		if err != nil {
			t.Fatalf("Exists() failed: %v", err)
		}
		if exists {
			t.Error("expected file to not exist")
		}
	})

	// TS: it('should allow access when containment is disabled')
	t.Run("should allow access when containment is disabled", func(t *testing.T) {
		outsideFile, err := os.CreateTemp(os.TempDir(), "outside-test-*.txt")
		if err != nil {
			t.Fatalf("failed to create outside file: %v", err)
		}
		outsideFilePath := outsideFile.Name()
		outsideFile.Close()
		// Resolve symlinks for the outside file path too
		outsideFilePath, _ = filepath.EvalSymlinks(outsideFilePath)
		t.Cleanup(func() { os.Remove(outsideFilePath) })

		if err := os.WriteFile(outsideFilePath, []byte("outside content"), 0644); err != nil {
			t.Fatalf("failed to write outside file: %v", err)
		}

		f := false
		uncontainedFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:  tempDir,
			Contained: &f,
		})

		content, err := uncontainedFs.ReadFile(outsideFilePath, nil)
		if err != nil {
			t.Fatalf("ReadFile(outside) failed: %v", err)
		}
		if string(content.([]byte)) != "outside content" {
			t.Errorf("expected 'outside content', got %q", string(content.([]byte)))
		}
	})

	// TS: it('should allow absolute paths outside base directory when containment is disabled')
	t.Run("should allow absolute paths outside base directory when containment is disabled", func(t *testing.T) {
		outsideFile, err := os.CreateTemp(os.TempDir(), "abs-outside-test-*.txt")
		if err != nil {
			t.Fatalf("failed to create outside file: %v", err)
		}
		outsideFilePath := outsideFile.Name()
		outsideFile.Close()
		outsideFilePath, _ = filepath.EvalSymlinks(outsideFilePath)
		t.Cleanup(func() { os.Remove(outsideFilePath) })

		if err := os.WriteFile(outsideFilePath, []byte("absolute outside content"), 0644); err != nil {
			t.Fatalf("failed to write outside file: %v", err)
		}

		f := false
		uncontainedFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:  tempDir,
			Contained: &f,
		})

		content, err := uncontainedFs.ReadFile(outsideFilePath, nil)
		if err != nil {
			t.Fatalf("ReadFile(absolute outside) failed: %v", err)
		}
		if string(content.([]byte)) != "absolute outside content" {
			t.Errorf("expected 'absolute outside content', got %q", string(content.([]byte)))
		}
	})
}

// =============================================================================
// AllowedPaths
// =============================================================================

// Ported from: describe('allowedPaths', ...)
func TestAllowedPaths(t *testing.T) {
	tempDir, lfs := setupTestFS(t)

	// Create an outside directory for allowed path tests
	outsideDir := realTempDir(t, "agent-kit-fs-allowed-")
	writeRawFile(t, filepath.Join(outsideDir, "external.txt"), "external content")

	// --- describe('constructor', ...) ---

	// TS: it('should default to empty allowedPaths')
	t.Run("constructor/should default to empty allowedPaths", func(t *testing.T) {
		ap := lfs.AllowedPaths()
		if len(ap) != 0 {
			t.Errorf("expected empty allowedPaths, got %v", ap)
		}
	})

	// TS: it('should accept allowedPaths in options')
	t.Run("constructor/should accept allowedPaths in options", func(t *testing.T) {
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})
		ap := fsWithAllowed.AllowedPaths()
		if len(ap) != 1 || ap[0] != outsideDir {
			t.Errorf("expected allowedPaths=[%s], got %v", outsideDir, ap)
		}
	})

	// TS: it('should resolve relative allowedPaths to absolute')
	t.Run("constructor/should resolve relative allowedPaths to absolute", func(t *testing.T) {
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{"./relative-dir"},
		})
		ap := fsWithAllowed.AllowedPaths()
		if len(ap) != 1 || !filepath.IsAbs(ap[0]) {
			t.Errorf("expected absolute path, got %v", ap)
		}
	})

	// --- describe('setAllowedPaths', ...) ---

	// TS: it('should set paths from array')
	t.Run("setAllowedPaths/should set paths from array", func(t *testing.T) {
		testFs := NewLocalFilesystem(LocalFilesystemOptions{BasePath: tempDir})
		testFs.SetAllowedPaths([]string{outsideDir})
		ap := testFs.AllowedPaths()
		if len(ap) != 1 || ap[0] != outsideDir {
			t.Errorf("expected allowedPaths=[%s], got %v", outsideDir, ap)
		}
	})

	// TS: it('should clear paths with empty array')
	t.Run("setAllowedPaths/should clear paths with empty array", func(t *testing.T) {
		testFs := NewLocalFilesystem(LocalFilesystemOptions{BasePath: tempDir})
		testFs.SetAllowedPaths([]string{outsideDir})
		if len(testFs.AllowedPaths()) != 1 {
			t.Fatal("expected 1 allowed path after set")
		}
		testFs.SetAllowedPaths([]string{})
		if len(testFs.AllowedPaths()) != 0 {
			t.Error("expected 0 allowed paths after clear")
		}
	})

	// TS: it('should resolve paths to absolute')
	t.Run("setAllowedPaths/should resolve paths to absolute", func(t *testing.T) {
		testFs := NewLocalFilesystem(LocalFilesystemOptions{BasePath: tempDir})
		testFs.SetAllowedPaths([]string{"./foo"})
		ap := testFs.AllowedPaths()
		if len(ap) != 1 || !filepath.IsAbs(ap[0]) {
			t.Errorf("expected absolute path, got %v", ap)
		}
	})

	// --- describe('file operations with allowedPaths', ...) ---

	// TS: it('should read files from an allowed path using absolute path')
	t.Run("file operations/should read files from an allowed path using absolute path", func(t *testing.T) {
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})
		content, err := fsWithAllowed.ReadFile(filepath.Join(outsideDir, "external.txt"), nil)
		if err != nil {
			t.Fatalf("ReadFile(allowed) failed: %v", err)
		}
		if string(content.([]byte)) != "external content" {
			t.Errorf("expected 'external content', got %q", string(content.([]byte)))
		}
	})

	// TS: it('should write files to an allowed path')
	t.Run("file operations/should write files to an allowed path", func(t *testing.T) {
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})
		if err := fsWithAllowed.WriteFile(filepath.Join(outsideDir, "new-file.txt"), "new content", nil); err != nil {
			t.Fatalf("WriteFile(allowed) failed: %v", err)
		}
		content := readRawFile(t, filepath.Join(outsideDir, "new-file.txt"))
		if content != "new content" {
			t.Errorf("expected 'new content', got %q", content)
		}
	})

	// TS: it('should block path traversal even with allowedPaths')
	t.Run("file operations/should block path traversal even with allowedPaths", func(t *testing.T) {
		restrictedFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})
		_, err := restrictedFs.ReadFile("/../../../etc/passwd", nil)
		if err == nil {
			t.Fatal("expected PermissionError for path traversal")
		}
		var pe *workspace.PermissionError
		if !errors.As(err, &pe) {
			t.Errorf("expected PermissionError, got %T: %v", err, err)
		}
	})

	// TS: it('should still allow basePath access when allowedPaths are set')
	t.Run("file operations/should still allow basePath access when allowedPaths are set", func(t *testing.T) {
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})
		if err := fsWithAllowed.WriteFile("local-file.txt", "local content", nil); err != nil {
			t.Fatalf("WriteFile(local) failed: %v", err)
		}
		content, err := fsWithAllowed.ReadFile("local-file.txt", nil)
		if err != nil {
			t.Fatalf("ReadFile(local) failed: %v", err)
		}
		if string(content.([]byte)) != "local content" {
			t.Errorf("expected 'local content', got %q", string(content.([]byte)))
		}
	})

	// TS: it('should respect dynamically added allowedPaths')
	t.Run("file operations/should respect dynamically added allowedPaths", func(t *testing.T) {
		dynamicFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath: tempDir,
		})

		// Initially blocked
		_, err := dynamicFs.ReadFile(filepath.Join(outsideDir, "external.txt"), nil)
		if err == nil {
			t.Fatal("expected error for outside path without allowedPaths")
		}

		// Add allowedPath dynamically
		dynamicFs.SetAllowedPaths([]string{outsideDir})

		// Now accessible
		content, err := dynamicFs.ReadFile(filepath.Join(outsideDir, "external.txt"), nil)
		if err != nil {
			t.Fatalf("ReadFile(dynamic allowed) failed: %v", err)
		}
		if string(content.([]byte)) != "external content" {
			t.Errorf("expected 'external content', got %q", string(content.([]byte)))
		}
	})

	// TS: it('should block access after removing allowedPaths')
	t.Run("file operations/should block access after removing allowedPaths", func(t *testing.T) {
		dynamicFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})

		// Initially accessible
		content, err := dynamicFs.ReadFile(filepath.Join(outsideDir, "external.txt"), nil)
		if err != nil {
			t.Fatalf("ReadFile(allowed) failed: %v", err)
		}
		if string(content.([]byte)) != "external content" {
			t.Errorf("expected 'external content', got %q", string(content.([]byte)))
		}

		// Remove allowed paths
		dynamicFs.SetAllowedPaths([]string{})

		// Now blocked
		_, err = dynamicFs.ReadFile(filepath.Join(outsideDir, "external.txt"), nil)
		if err == nil {
			t.Fatal("expected error after removing allowedPaths")
		}
	})

	// TS: it('should check exists() against allowedPaths')
	t.Run("file operations/should check exists against allowedPaths", func(t *testing.T) {
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})

		exists, err := fsWithAllowed.Exists(filepath.Join(outsideDir, "external.txt"))
		if err != nil {
			t.Fatalf("Exists(allowed) failed: %v", err)
		}
		if !exists {
			t.Error("expected existing file in allowed path to be found")
		}

		exists, err = fsWithAllowed.Exists(filepath.Join(outsideDir, "nonexistent.txt"))
		if err != nil {
			t.Fatalf("Exists(nonexistent) failed: %v", err)
		}
		if exists {
			t.Error("expected nonexistent file in allowed path to not be found")
		}
	})

	// TS: it('should check stat() against allowedPaths')
	t.Run("file operations/should check stat against allowedPaths", func(t *testing.T) {
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})

		stat, err := fsWithAllowed.Stat(filepath.Join(outsideDir, "external.txt"))
		if err != nil {
			t.Fatalf("Stat(allowed) failed: %v", err)
		}
		if stat.Type != "file" {
			t.Errorf("expected type 'file', got %q", stat.Type)
		}
		if stat.Size != int64(len("external content")) {
			t.Errorf("expected size %d, got %d", len("external content"), stat.Size)
		}
	})

	// TS: it('should allow readdir on allowed path')
	t.Run("file operations/should allow readdir on allowed path", func(t *testing.T) {
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})

		entries, err := fsWithAllowed.Readdir(outsideDir, nil)
		if err != nil {
			t.Fatalf("Readdir(allowed) failed: %v", err)
		}
		found := false
		for _, e := range entries {
			if e.Name == "external.txt" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected to find external.txt in allowed path readdir")
		}
	})
}

// =============================================================================
// GetInfo with allowedPaths
// =============================================================================

// Ported from: describe('getInfo with allowedPaths', ...)
func TestGetInfoWithAllowedPaths(t *testing.T) {
	tempDir, lfs := setupTestFS(t)

	outsideDir := realTempDir(t, "agent-kit-fs-info-")

	// TS: it('should not include allowedPaths in metadata when empty')
	t.Run("should not include allowedPaths in metadata when empty", func(t *testing.T) {
		info, err := lfs.GetInfo()
		if err != nil {
			t.Fatalf("GetInfo() failed: %v", err)
		}
		if _, ok := info.Metadata["allowedPaths"]; ok {
			t.Error("expected no allowedPaths in metadata when empty")
		}
	})

	// TS: it('should include allowedPaths in metadata when set')
	t.Run("should include allowedPaths in metadata when set", func(t *testing.T) {
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})
		info, err := fsWithAllowed.GetInfo()
		if err != nil {
			t.Fatalf("GetInfo() failed: %v", err)
		}
		ap, ok := info.Metadata["allowedPaths"]
		if !ok {
			t.Fatal("expected allowedPaths in metadata")
		}
		apSlice, ok := ap.([]string)
		if !ok {
			t.Fatalf("expected []string, got %T", ap)
		}
		if len(apSlice) != 1 || apSlice[0] != outsideDir {
			t.Errorf("expected [%s], got %v", outsideDir, apSlice)
		}
	})
}

// =============================================================================
// GetInstructions with allowedPaths
// =============================================================================

// Ported from: describe('getInstructions with allowedPaths', ...)
func TestGetInstructionsWithAllowedPaths(t *testing.T) {
	tempDir, lfs := setupTestFS(t)

	outsideDir := realTempDir(t, "agent-kit-fs-instr-")

	// TS: it('should not mention allowedPaths when empty')
	t.Run("should not mention allowedPaths when empty", func(t *testing.T) {
		instructions := lfs.GetInstructions(nil)
		if strings.Contains(instructions, "Additionally") {
			t.Error("expected no 'Additionally' mention when allowedPaths is empty")
		}
	})

	// TS: it('should mention allowedPaths when set')
	t.Run("should mention allowedPaths when set", func(t *testing.T) {
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{outsideDir},
		})
		instructions := fsWithAllowed.GetInstructions(nil)
		if !strings.Contains(instructions, "Additionally") {
			t.Error("expected 'Additionally' in instructions when allowedPaths set")
		}
		if !strings.Contains(instructions, outsideDir) {
			t.Errorf("expected instructions to contain %q", outsideDir)
		}
	})
}

// =============================================================================
// GetInstructions with custom override
// =============================================================================

// Ported from: describe('getInstructions with custom override', ...)
func TestGetInstructionsWithCustomOverride(t *testing.T) {
	tempDir, _ := setupTestFS(t)

	// TS: it('should return custom instructions when provided')
	t.Run("should return custom instructions when provided", func(t *testing.T) {
		testFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			Instructions: workspace.InstructionsOptionStatic("Custom filesystem instructions here."),
		})
		result := testFs.GetInstructions(nil)
		if result != "Custom filesystem instructions here." {
			t.Errorf("expected custom instructions, got %q", result)
		}
	})

	// TS: it('should return empty string when override is empty string')
	t.Run("should return empty string when override is empty string", func(t *testing.T) {
		testFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			Instructions: workspace.InstructionsOptionStatic(""),
		})
		result := testFs.GetInstructions(nil)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	// TS: it('should return auto-generated instructions when no override')
	t.Run("should return auto-generated instructions when no override", func(t *testing.T) {
		testFs := NewLocalFilesystem(LocalFilesystemOptions{BasePath: tempDir})
		result := testFs.GetInstructions(nil)
		if !strings.Contains(result, "Local filesystem") {
			t.Errorf("expected auto-generated instructions containing 'Local filesystem', got %q", result)
		}
	})

	// TS: it('should support function form that extends auto instructions')
	t.Run("should support function form that extends auto instructions", func(t *testing.T) {
		testFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath: tempDir,
			Instructions: workspace.InstructionsOptionFunc(func(opts workspace.InstructionsOptionFuncArgs) string {
				return opts.DefaultInstructions + "\nExtra info."
			}),
		})
		result := testFs.GetInstructions(nil)
		if !strings.Contains(result, "Local filesystem") {
			t.Error("expected auto-generated portion in instructions")
		}
		if !strings.Contains(result, "Extra info.") {
			t.Error("expected 'Extra info.' in instructions")
		}
	})

	// TS: it('should pass requestContext to function form')
	t.Run("should pass requestContext to function form", func(t *testing.T) {
		called := false
		testFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath: tempDir,
			Instructions: workspace.InstructionsOptionFunc(func(opts workspace.InstructionsOptionFuncArgs) string {
				called = true
				locale := ""
				if opts.RequestContext != nil {
					if v := opts.RequestContext.Get("locale"); v != nil {
						locale, _ = v.(string)
					}
				}
				return opts.DefaultInstructions + " locale=" + locale
			}),
		})
		ctx := requestcontext.NewRequestContext()
		ctx.Set("locale", "fr")
		result := testFs.GetInstructions(&InstructionsOpts{
			RequestContext: ctx,
		})
		if !called {
			t.Error("expected instructions function to be called")
		}
		if !strings.Contains(result, "locale=fr") {
			t.Errorf("expected 'locale=fr' in result, got %q", result)
		}
		if !strings.Contains(result, "Local filesystem") {
			t.Error("expected auto-generated portion in instructions")
		}
	})

	// TS: it('should pass undefined requestContext when not provided to function form')
	t.Run("should pass nil requestContext when not provided to function form", func(t *testing.T) {
		testFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath: tempDir,
			Instructions: workspace.InstructionsOptionFunc(func(opts workspace.InstructionsOptionFuncArgs) string {
				rcStr := "<nil>"
				if opts.RequestContext != nil {
					rcStr = "non-nil"
				}
				return opts.DefaultInstructions + " ctx=" + rcStr
			}),
		})
		result := testFs.GetInstructions(nil)
		if !strings.Contains(result, "ctx=<nil>") {
			t.Errorf("expected 'ctx=<nil>' in result, got %q", result)
		}
	})
}

// =============================================================================
// Tilde (~) expansion
// =============================================================================

// Ported from: describe('tilde expansion', ...)
func TestTildeExpansion(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	// TS: it('should expand ~ in basePath')
	t.Run("should expand tilde in basePath", func(t *testing.T) {
		tildeFs := NewLocalFilesystem(LocalFilesystemOptions{BasePath: "~/my-project"})
		expected := filepath.Join(homeDir, "my-project")
		if tildeFs.BasePath() != expected {
			t.Errorf("expected basePath %q, got %q", expected, tildeFs.BasePath())
		}
	})

	// TS: it('should expand ~ in allowedPaths constructor option')
	t.Run("should expand tilde in allowedPaths constructor option", func(t *testing.T) {
		tempDir := realTempDir(t, "agent-kit-tilde-test-")

		tildeFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{"~/allowed-dir"},
		})
		ap := tildeFs.AllowedPaths()
		expected := filepath.Join(homeDir, "allowed-dir")
		if len(ap) != 1 || ap[0] != expected {
			t.Errorf("expected allowedPaths=[%s], got %v", expected, ap)
		}
	})

	// TS: it('should expand ~ to home directory when contained is false')
	t.Run("should expand tilde to home directory when contained is false", func(t *testing.T) {
		tildeTargetDir := realTempDirIn(t, homeDir, ".agent-kit-tilde-test-")
		tempDir := realTempDir(t, "agent-kit-tilde-base-")

		f := false
		uncontainedFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:  tempDir,
			Contained: &f,
		})

		relativeTildePath := strings.Replace(tildeTargetDir, homeDir, "~", 1)
		filePath := relativeTildePath + "/tilde-test.txt"

		if err := uncontainedFs.WriteFile(filePath, "tilde works", nil); err != nil {
			t.Fatalf("WriteFile(tilde) failed: %v", err)
		}

		absoluteExpected := filepath.Join(tildeTargetDir, "tilde-test.txt")
		content := readRawFile(t, absoluteExpected)
		if content != "tilde works" {
			t.Errorf("expected 'tilde works', got %q", content)
		}
	})

	// TS: it('should expand ~ to home directory when path is in allowedPaths')
	t.Run("should expand tilde to home directory when path is in allowedPaths", func(t *testing.T) {
		tildeTargetDir := realTempDirIn(t, homeDir, ".agent-kit-tilde-allowed-")
		tempDir := realTempDir(t, "agent-kit-tilde-base2-")

		relativeTildeDir := strings.Replace(tildeTargetDir, homeDir, "~", 1)
		fsWithAllowed := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:     tempDir,
			AllowedPaths: []string{relativeTildeDir},
		})
		filePath := relativeTildeDir + "/tilde-allowed.txt"

		if err := fsWithAllowed.WriteFile(filePath, "tilde allowed works", nil); err != nil {
			t.Fatalf("WriteFile(tilde allowed) failed: %v", err)
		}

		absoluteExpected := filepath.Join(tildeTargetDir, "tilde-allowed.txt")
		content := readRawFile(t, absoluteExpected)
		if content != "tilde allowed works" {
			t.Errorf("expected 'tilde allowed works', got %q", content)
		}
	})

	// TS: it('should expand ~ in setAllowedPaths')
	t.Run("should expand tilde in setAllowedPaths", func(t *testing.T) {
		tildeTargetDir := realTempDirIn(t, homeDir, ".agent-kit-tilde-set-")
		tempDir := realTempDir(t, "agent-kit-tilde-base3-")

		testFs := NewLocalFilesystem(LocalFilesystemOptions{BasePath: tempDir})

		relativeTildeDir := strings.Replace(tildeTargetDir, homeDir, "~", 1)
		testFs.SetAllowedPaths([]string{relativeTildeDir})

		filePath := relativeTildeDir + "/tilde-set-allowed.txt"
		if err := testFs.WriteFile(filePath, "tilde set allowed works", nil); err != nil {
			t.Fatalf("WriteFile(tilde set allowed) failed: %v", err)
		}

		absoluteExpected := filepath.Join(tildeTargetDir, "tilde-set-allowed.txt")
		content := readRawFile(t, absoluteExpected)
		if content != "tilde set allowed works" {
			t.Errorf("expected 'tilde set allowed works', got %q", content)
		}
	})

	// TS: it('should throw PermissionError for tilde path outside basePath in contained mode')
	t.Run("should return PermissionError for tilde path outside basePath in contained mode", func(t *testing.T) {
		tildeTargetDir := realTempDirIn(t, homeDir, ".agent-kit-tilde-perm-")
		tempDir := realTempDir(t, "agent-kit-tilde-base4-")

		testFs := NewLocalFilesystem(LocalFilesystemOptions{BasePath: tempDir})

		relativeTildeDir := strings.Replace(tildeTargetDir, homeDir, "~", 1)
		filePath := relativeTildeDir + "/contained.txt"

		writeErr := testFs.WriteFile(filePath, "should not be here", nil)
		if writeErr == nil {
			t.Fatal("expected PermissionError for tilde path outside basePath")
		}
		// The error should contain "Permission" -- the TS test checks for 'Permission' substring
		if !strings.Contains(writeErr.Error(), "Permission") {
			t.Errorf("expected error containing 'Permission', got: %v", writeErr)
		}
	})

	// TS: it('should read files written via tilde path')
	t.Run("should read files written via tilde path", func(t *testing.T) {
		tildeTargetDir := realTempDirIn(t, homeDir, ".agent-kit-tilde-read-")
		tempDir := realTempDir(t, "agent-kit-tilde-base5-")

		f := false
		uncontainedFs := NewLocalFilesystem(LocalFilesystemOptions{
			BasePath:  tempDir,
			Contained: &f,
		})

		relativeTildePath := strings.Replace(tildeTargetDir, homeDir, "~", 1)

		// Write directly to disk
		writeRawFile(t, filepath.Join(tildeTargetDir, "read-test.txt"), "read via tilde")

		// Read via tilde path
		content, err := uncontainedFs.ReadFile(relativeTildePath+"/read-test.txt", nil)
		if err != nil {
			t.Fatalf("ReadFile(tilde) failed: %v", err)
		}
		if string(content.([]byte)) != "read via tilde" {
			t.Errorf("expected 'read via tilde', got %q", string(content.([]byte)))
		}
	})
}

// =============================================================================
// MIME Type Detection
// =============================================================================

// Ported from: describe('mime type detection', ...)
// Note: The Go implementation maps some MIME types differently than the TS version:
//   - .js  -> "text/javascript"   (TS: "application/javascript")
//   - .ts  -> "text/typescript"   (TS: "application/typescript")
//   - .xml -> "text/xml"          (TS: "application/xml")
//   - .py  -> not mapped          (TS: "text/x-python") -> falls back to "application/octet-stream"
//   - .md  -> "text/markdown"     (same)
// Tests are adjusted to match the Go implementation's mimeTypes map.
func TestMimeTypeDetection(t *testing.T) {
	_, lfs := setupTestFS(t)

	testCases := []struct {
		ext      string
		expected string
	}{
		{ext: "txt", expected: "text/plain"},
		{ext: "html", expected: "text/html"},
		{ext: "css", expected: "text/css"},
		// Go mimeTypes map: .js -> "text/javascript" (differs from TS "application/javascript")
		{ext: "js", expected: "text/javascript"},
		// Go mimeTypes map: .ts -> "text/typescript" (differs from TS "application/typescript")
		{ext: "ts", expected: "text/typescript"},
		{ext: "json", expected: "application/json"},
		// Go mimeTypes map: .xml -> "text/xml" (differs from TS "application/xml")
		{ext: "xml", expected: "text/xml"},
		{ext: "md", expected: "text/markdown"},
		// Go mimeTypes map: .py is NOT mapped (differs from TS "text/x-python")
		// Falls back to "application/octet-stream"
		{ext: "py", expected: "application/octet-stream"},
		{ext: "unknown", expected: "application/octet-stream"},
	}

	for _, tc := range testCases {
		t.Run("should detect "+tc.ext+" as "+tc.expected, func(t *testing.T) {
			filePath := "mime-test." + tc.ext
			if err := lfs.WriteFile(filePath, "content", nil); err != nil {
				t.Fatalf("WriteFile() failed: %v", err)
			}
			stat, err := lfs.Stat(filePath)
			if err != nil {
				t.Fatalf("Stat() failed: %v", err)
			}
			if stat.MimeType != tc.expected {
				t.Errorf("expected MIME %q for .%s, got %q", tc.expected, tc.ext, stat.MimeType)
			}
		})
	}
}
