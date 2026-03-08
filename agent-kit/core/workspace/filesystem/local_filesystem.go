// Ported from: packages/core/src/workspace/filesystem/local-filesystem.ts
package filesystem

import (
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	workspace "github.com/brainlet/brainkit/agent-kit/core/workspace"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// =============================================================================
// LocalFilesystem Options
// =============================================================================

// LocalFilesystemOptions configures a LocalFilesystem instance.
type LocalFilesystemOptions struct {
	// ID is the unique identifier for this filesystem instance.
	ID string
	// BasePath is the base directory path on disk.
	BasePath string
	// Contained restricts all file operations to stay within BasePath.
	// Prevents path traversal attacks and symlink escapes.
	// Default: true.
	Contained *bool
	// ReadOnly blocks all write operations when true.
	// Default: false.
	ReadOnly bool
	// AllowedPaths are additional absolute paths allowed beyond BasePath.
	// Useful with Contained=true to grant access to specific directories.
	AllowedPaths []string
	// Instructions is the custom instructions override.
	Instructions workspace.InstructionsOption
	// Logger is an optional logger instance.
	Logger Logger
}

// LocalMountConfig is the mount configuration for local filesystems.
type LocalMountConfig struct {
	FilesystemMountConfig
	// BasePath is the host path for local mounts.
	BasePath string
}

// =============================================================================
// LocalFilesystem
// =============================================================================

// LocalFilesystem stores files in a folder on the local disk.
// This is the default filesystem for development and local agents.
type LocalFilesystem struct {
	MastraFilesystem
	id                    string
	name                  string
	provider              string
	readOnly              bool
	basePath              string
	contained             bool
	allowedPaths          []string
	instructionsOverride  workspace.InstructionsOption
}

// NewLocalFilesystem creates a new LocalFilesystem.
func NewLocalFilesystem(opts LocalFilesystemOptions) *LocalFilesystem {
	contained := true
	if opts.Contained != nil {
		contained = *opts.Contained
	}

	basePath, _ := filepath.Abs(ExpandTilde(opts.BasePath))

	allowedPaths := make([]string, len(opts.AllowedPaths))
	for i, p := range opts.AllowedPaths {
		abs, _ := filepath.Abs(ExpandTilde(p))
		allowedPaths[i] = abs
	}

	id := opts.ID
	if id == "" {
		id = generateLocalFSID()
	}

	lfs := &LocalFilesystem{
		MastraFilesystem: NewMastraFilesystem(MastraFilesystemOptions{
			Name:   "LocalFilesystem",
			Logger: opts.Logger,
		}),
		id:                   id,
		name:                 "LocalFilesystem",
		provider:             "local",
		readOnly:             opts.ReadOnly,
		basePath:             basePath,
		contained:            contained,
		allowedPaths:         allowedPaths,
		instructionsOverride: opts.Instructions,
	}

	// Wire up the init/destroy hooks
	lfs.MastraFilesystem.onInitHook = func() error {
		return lfs.onInit()
	}
	lfs.MastraFilesystem.onDestroyHook = func() error {
		return lfs.onDestroy()
	}

	return lfs
}

func generateLocalFSID() string {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 36)
	rnd := strconv.FormatInt(int64(rand.Intn(1<<30)), 36)
	return "local-fs-" + ts + "-" + rnd
}

// =============================================================================
// Identity Methods
// =============================================================================

// ID returns the unique identifier for this filesystem.
func (lfs *LocalFilesystem) ID() string { return lfs.id }

// Name returns the filesystem name.
func (lfs *LocalFilesystem) Name() string { return lfs.name }

// Provider returns the provider type ("local").
func (lfs *LocalFilesystem) Provider() string { return lfs.provider }

// ReadOnly returns whether the filesystem is in read-only mode.
func (lfs *LocalFilesystem) ReadOnly() bool { return lfs.readOnly }

// IsReadOnly is an alias for ReadOnly (for convenience).
func (lfs *LocalFilesystem) IsReadOnly() bool { return lfs.readOnly }

// Icon returns nil (no icon for local filesystem).
func (lfs *LocalFilesystem) Icon() *FilesystemIcon { return nil }

// DisplayName returns empty string (uses name).
func (lfs *LocalFilesystem) DisplayName() string { return "" }

// Description returns empty string.
func (lfs *LocalFilesystem) Description() string { return "" }

// Status returns the current provider status.
func (lfs *LocalFilesystem) Status() ProviderStatus { return lfs.GetStatus() }

// BasePath returns the absolute base path on disk where files are stored.
func (lfs *LocalFilesystem) BasePath() string { return lfs.basePath }

// Contained returns whether file operations are restricted to basePath.
func (lfs *LocalFilesystem) Contained() bool { return lfs.contained }

// AllowedPaths returns the current set of additional allowed paths.
func (lfs *LocalFilesystem) AllowedPaths() []string {
	out := make([]string, len(lfs.allowedPaths))
	copy(out, lfs.allowedPaths)
	return out
}

// SetAllowedPaths updates the allowed paths list.
func (lfs *LocalFilesystem) SetAllowedPaths(paths []string) {
	resolved := make([]string, len(paths))
	for i, p := range paths {
		abs, _ := filepath.Abs(ExpandTilde(p))
		resolved[i] = abs
	}
	lfs.allowedPaths = resolved
}

// GetMountConfig returns the mount configuration for sandbox integration.
func (lfs *LocalFilesystem) GetMountConfig() *FilesystemMountConfig {
	return &FilesystemMountConfig{
		Type:      "local",
		LocalPath: lfs.basePath,
	}
}

// GetLocalMountConfig returns the local-specific mount configuration.
func (lfs *LocalFilesystem) GetLocalMountConfig() *LocalMountConfig {
	return &LocalMountConfig{
		FilesystemMountConfig: FilesystemMountConfig{Type: "local", LocalPath: lfs.basePath},
		BasePath:              lfs.basePath,
	}
}

// toBytes converts interface{} content to []byte.
func toBytes(content interface{}) []byte {
	switch v := content.(type) {
	case []byte:
		return v
	case string:
		return []byte(v)
	default:
		return []byte(fmt.Sprintf("%v", v))
	}
}

// =============================================================================
// Path Resolution
// =============================================================================

// isWithinAnyRoot checks if an absolute path falls within basePath or any allowed path.
func (lfs *LocalFilesystem) isWithinAnyRoot(absolutePath string) bool {
	roots := append([]string{lfs.basePath}, lfs.allowedPaths...)
	for _, root := range roots {
		rel, err := filepath.Rel(root, absolutePath)
		if err != nil {
			continue
		}
		if !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}

// resolvePath resolves an input path to an absolute filesystem path,
// applying containment checks.
func (lfs *LocalFilesystem) resolvePath(inputPath string) (string, error) {
	wasTilde := strings.HasPrefix(inputPath, "~")
	inputPath = ExpandTilde(inputPath)

	var absolutePath string

	if !lfs.contained && filepath.IsAbs(inputPath) {
		// Containment disabled — absolute paths are real filesystem paths
		absolutePath = filepath.Clean(inputPath)
	} else if lfs.contained && filepath.IsAbs(inputPath) {
		// Containment enabled — check if this is a real path within basePath
		normalized := filepath.Clean(inputPath)
		if lfs.isWithinAnyRoot(normalized) {
			absolutePath = normalized
		} else if wasTilde {
			// Path started with ~ — user meant a real filesystem path
			absolutePath = normalized
		} else {
			// In contained mode, treat absolute paths as workspace-relative.
			// Strip the leading "/" and join with basePath so that "/file.txt"
			// resolves to basePath/file.txt, not the literal root /file.txt.
			// This matches the TS behavior where paths like "/file.txt" are
			// always relative to the workspace root when containment is on.
			relPath := strings.TrimPrefix(inputPath, "/")
			absolutePath = filepath.Join(lfs.basePath, relPath)
		}
	} else {
		absolutePath = ResolveWorkspacePath(lfs.basePath, inputPath)
	}

	if lfs.contained {
		if !lfs.isWithinAnyRoot(absolutePath) {
			return "", workspace.NewPermissionError(inputPath, "access")
		}
	}

	return absolutePath, nil
}

// ResolveAbsolutePath resolves a workspace-relative path to an absolute disk path.
// Returns empty string if the path violates containment.
func (lfs *LocalFilesystem) ResolveAbsolutePath(inputPath string) string {
	p, err := lfs.resolvePath(inputPath)
	if err != nil {
		return ""
	}
	return p
}

// toRelativePath converts an absolute path back to a workspace-relative path.
func (lfs *LocalFilesystem) toRelativePath(absolutePath string) string {
	rel, err := filepath.Rel(lfs.basePath, absolutePath)
	if err != nil {
		return absolutePath
	}
	return "/" + filepath.ToSlash(rel)
}

// assertWritable checks if the filesystem allows write operations.
func (lfs *LocalFilesystem) assertWritable(operation string) error {
	if lfs.readOnly {
		return workspace.NewWorkspaceReadOnlyError(operation)
	}
	return nil
}

// assertPathContained verifies that the resolved path doesn't escape basePath
// via symlinks. Uses filepath.EvalSymlinks to resolve symlinks and check the
// actual target.
func (lfs *LocalFilesystem) assertPathContained(absolutePath string) error {
	if !lfs.contained {
		return nil
	}

	// Resolve real paths for all roots
	rootReals := make([]string, 0, len(lfs.allowedPaths)+1)
	for _, root := range append([]string{lfs.basePath}, lfs.allowedPaths...) {
		real, err := filepath.EvalSymlinks(root)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return err
		}
		rootReals = append(rootReals, real)
	}

	if len(rootReals) == 0 {
		return workspace.NewDirectoryNotFoundError(lfs.basePath)
	}

	targetReal, err := filepath.EvalSymlinks(absolutePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Walk up to find an existing parent
			parentPath := absolutePath
			for {
				nextParent := filepath.Dir(parentPath)
				if nextParent == parentPath {
					return workspace.NewDirectoryNotFoundError(absolutePath)
				}
				parentPath = nextParent
				real, err2 := filepath.EvalSymlinks(parentPath)
				if err2 != nil {
					if errors.Is(err2, fs.ErrNotExist) {
						continue
					}
					return err2
				}
				targetReal = real
				break
			}
		} else {
			return err
		}
	}

	for _, rootReal := range rootReals {
		if targetReal == rootReal || strings.HasPrefix(targetReal, rootReal+string(filepath.Separator)) {
			return nil
		}
	}

	return workspace.NewPermissionError(absolutePath, "access")
}

// =============================================================================
// File Operations
// =============================================================================

// ReadFile reads a file from the filesystem.
func (lfs *LocalFilesystem) ReadFile(inputPath string, options *ReadOptions) (interface{}, error) {
	lfs.GetLogger().Debug("Reading file", "path", inputPath)
	if err := lfs.EnsureReady(); err != nil {
		return nil, err
	}
	absolutePath, err := lfs.resolvePath(inputPath)
	if err != nil {
		return nil, err
	}
	if err := lfs.assertPathContained(absolutePath); err != nil {
		return nil, err
	}

	info, err := os.Stat(absolutePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, workspace.NewFileNotFoundError(inputPath)
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, workspace.NewIsDirectoryError(inputPath)
	}

	data, err := os.ReadFile(absolutePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, workspace.NewFileNotFoundError(inputPath)
		}
		return nil, err
	}
	return data, nil
}

// WriteFile writes content to a file.
func (lfs *LocalFilesystem) WriteFile(inputPath string, content interface{}, options *WriteOptions) error {
	data := toBytes(content)
	lfs.GetLogger().Debug("Writing file", "path", inputPath, "size", len(data))
	if err := lfs.EnsureReady(); err != nil {
		return err
	}
	if err := lfs.assertWritable("writeFile"); err != nil {
		return err
	}
	absolutePath, err := lfs.resolvePath(inputPath)
	if err != nil {
		return err
	}
	if err := lfs.assertPathContained(absolutePath); err != nil {
		return err
	}

	recursive := true
	if options != nil && !options.Recursive {
		recursive = false
	}

	// When recursive is explicitly false, verify parent directory exists
	if !recursive {
		dir := filepath.Dir(absolutePath)
		parentPath := filepath.Dir(inputPath)
		info, err := os.Stat(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return workspace.NewDirectoryNotFoundError(parentPath)
			}
			return err
		}
		if !info.IsDir() {
			return workspace.NewNotDirectoryError(parentPath)
		}
	}

	if recursive {
		dir := filepath.Dir(absolutePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Check overwrite
	if options != nil && options.Overwrite != nil && !*options.Overwrite {
		if _, err := os.Stat(absolutePath); err == nil {
			return workspace.NewFileExistsError(inputPath)
		}
	}

	return os.WriteFile(absolutePath, data, 0644)
}

// AppendFile appends content to a file, creating it if needed.
func (lfs *LocalFilesystem) AppendFile(inputPath string, content interface{}) error {
	data := toBytes(content)
	lfs.GetLogger().Debug("Appending to file", "path", inputPath, "size", len(data))
	if err := lfs.EnsureReady(); err != nil {
		return err
	}
	if err := lfs.assertWritable("appendFile"); err != nil {
		return err
	}
	absolutePath, err := lfs.resolvePath(inputPath)
	if err != nil {
		return err
	}
	if err := lfs.assertPathContained(absolutePath); err != nil {
		return err
	}

	dir := filepath.Dir(absolutePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(absolutePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// DeleteFile removes a file from the filesystem.
func (lfs *LocalFilesystem) DeleteFile(inputPath string, options *RemoveOptions) error {
	lfs.GetLogger().Debug("Deleting file", "path", inputPath)
	if err := lfs.EnsureReady(); err != nil {
		return err
	}
	if err := lfs.assertWritable("deleteFile"); err != nil {
		return err
	}
	absolutePath, err := lfs.resolvePath(inputPath)
	if err != nil {
		return err
	}
	if err := lfs.assertPathContained(absolutePath); err != nil {
		return err
	}

	info, err := os.Stat(absolutePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if options != nil && options.Force {
				return nil
			}
			return workspace.NewFileNotFoundError(inputPath)
		}
		return err
	}
	if info.IsDir() {
		return workspace.NewIsDirectoryError(inputPath)
	}

	err = os.Remove(absolutePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if options != nil && options.Force {
				return nil
			}
			return workspace.NewFileNotFoundError(inputPath)
		}
		return err
	}
	return nil
}

// CopyFile copies a file or directory.
func (lfs *LocalFilesystem) CopyFile(src, dest string, options *CopyOptions) error {
	lfs.GetLogger().Debug("Copying file", "src", src, "dest", dest)
	if err := lfs.EnsureReady(); err != nil {
		return err
	}
	if err := lfs.assertWritable("copyFile"); err != nil {
		return err
	}
	srcPath, err := lfs.resolvePath(src)
	if err != nil {
		return err
	}
	destPath, err := lfs.resolvePath(dest)
	if err != nil {
		return err
	}
	if err := lfs.assertPathContained(srcPath); err != nil {
		return err
	}
	if err := lfs.assertPathContained(destPath); err != nil {
		return err
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return workspace.NewFileNotFoundError(src)
		}
		return err
	}

	if info.IsDir() {
		if options == nil || !options.Recursive {
			return workspace.NewIsDirectoryError(src)
		}
		return lfs.copyDirectory(srcPath, destPath, options)
	}

	// File copy
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Check overwrite
	if options != nil && !options.Overwrite {
		if _, err := os.Stat(destPath); err == nil {
			return workspace.NewFileExistsError(dest)
		}
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	return os.WriteFile(destPath, data, 0644)
}

func (lfs *LocalFilesystem) copyDirectory(src, dest string, options *CopyOptions) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcEntry := filepath.Join(src, entry.Name())
		destEntry := filepath.Join(dest, entry.Name())

		// Verify entries don't escape sandbox via symlink
		if err := lfs.assertPathContained(srcEntry); err != nil {
			return err
		}
		if err := lfs.assertPathContained(destEntry); err != nil {
			return err
		}

		if entry.IsDir() {
			if err := lfs.copyDirectory(srcEntry, destEntry, options); err != nil {
				return err
			}
		} else {
			// Check overwrite
			if options != nil && !options.Overwrite {
				if _, err := os.Stat(destEntry); err == nil {
					// Skip existing files when overwrite is false
					continue
				}
			}
			data, err := os.ReadFile(srcEntry)
			if err != nil {
				return err
			}
			if err := os.WriteFile(destEntry, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// MoveFile moves a file or directory.
func (lfs *LocalFilesystem) MoveFile(src, dest string, options *CopyOptions) error {
	lfs.GetLogger().Debug("Moving file", "src", src, "dest", dest)
	if err := lfs.EnsureReady(); err != nil {
		return err
	}
	if err := lfs.assertWritable("moveFile"); err != nil {
		return err
	}
	srcPath, err := lfs.resolvePath(src)
	if err != nil {
		return err
	}
	destPath, err := lfs.resolvePath(dest)
	if err != nil {
		return err
	}
	if err := lfs.assertPathContained(srcPath); err != nil {
		return err
	}
	if err := lfs.assertPathContained(destPath); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// When overwrite: false, use copy+delete to avoid TOCTOU race
	if options != nil && !options.Overwrite {
		if err := lfs.CopyFile(src, dest, &CopyOptions{Overwrite: false}); err != nil {
			return err
		}
		return os.RemoveAll(srcPath)
	}

	// Try rename first (fastest for same-device moves)
	err = os.Rename(srcPath, destPath)
	if err != nil {
		// Fall back to copy+delete for cross-device moves
		if linkErr, ok := err.(*os.LinkError); ok && linkErr.Err.Error() == "cross-device link" {
			if err := lfs.CopyFile(src, dest, options); err != nil {
				return err
			}
			return os.RemoveAll(srcPath)
		}
		if errors.Is(err, fs.ErrNotExist) {
			return workspace.NewFileNotFoundError(src)
		}
		return err
	}
	return nil
}

// =============================================================================
// Directory Operations
// =============================================================================

// Mkdir creates a directory.
func (lfs *LocalFilesystem) Mkdir(inputPath string, options *MkdirOptions) error {
	lfs.GetLogger().Debug("Creating directory", "path", inputPath)
	if err := lfs.EnsureReady(); err != nil {
		return err
	}
	if err := lfs.assertWritable("mkdir"); err != nil {
		return err
	}
	absolutePath, err := lfs.resolvePath(inputPath)
	if err != nil {
		return err
	}
	if err := lfs.assertPathContained(absolutePath); err != nil {
		return err
	}

	recursive := true
	if options != nil && !options.Recursive {
		recursive = false
	}

	if recursive {
		return os.MkdirAll(absolutePath, 0755)
	}

	err = os.Mkdir(absolutePath, 0755)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			info, statErr := os.Stat(absolutePath)
			if statErr == nil && !info.IsDir() {
				return workspace.NewFileExistsError(inputPath)
			}
			return nil // Directory already exists
		}
		if errors.Is(err, fs.ErrNotExist) {
			parentPath := filepath.Dir(inputPath)
			return workspace.NewDirectoryNotFoundError(parentPath)
		}
		return err
	}
	return nil
}

// Rmdir removes a directory.
func (lfs *LocalFilesystem) Rmdir(inputPath string, options *RemoveOptions) error {
	lfs.GetLogger().Debug("Removing directory", "path", inputPath)
	if err := lfs.EnsureReady(); err != nil {
		return err
	}
	if err := lfs.assertWritable("rmdir"); err != nil {
		return err
	}
	absolutePath, err := lfs.resolvePath(inputPath)
	if err != nil {
		return err
	}
	if err := lfs.assertPathContained(absolutePath); err != nil {
		return err
	}

	info, err := os.Stat(absolutePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if options != nil && options.Force {
				return nil
			}
			return workspace.NewDirectoryNotFoundError(inputPath)
		}
		return err
	}
	if !info.IsDir() {
		return workspace.NewNotDirectoryError(inputPath)
	}

	if options != nil && options.Recursive {
		return os.RemoveAll(absolutePath)
	}

	// Non-recursive — check if empty first
	entries, err := os.ReadDir(absolutePath)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return workspace.NewDirectoryNotEmptyError(inputPath)
	}
	return os.Remove(absolutePath)
}

// Readdir reads a directory and returns its entries.
func (lfs *LocalFilesystem) Readdir(inputPath string, options *ListOptions) ([]FileEntry, error) {
	lfs.GetLogger().Debug("Reading directory", "path", inputPath)
	if err := lfs.EnsureReady(); err != nil {
		return nil, err
	}
	absolutePath, err := lfs.resolvePath(inputPath)
	if err != nil {
		return nil, err
	}
	if err := lfs.assertPathContained(absolutePath); err != nil {
		return nil, err
	}

	info, err := os.Stat(absolutePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, workspace.NewDirectoryNotFoundError(inputPath)
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, workspace.NewNotDirectoryError(inputPath)
	}

	dirEntries, err := os.ReadDir(absolutePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, workspace.NewDirectoryNotFoundError(inputPath)
		}
		return nil, err
	}

	var result []FileEntry
	for _, entry := range dirEntries {
		entryPath := filepath.Join(absolutePath, entry.Name())

		// Extension filter
		if options != nil && len(options.Extension) > 0 {
			if !entry.IsDir() {
				ext := filepath.Ext(entry.Name())
				matched := false
				for _, e := range options.Extension {
					if e == ext || e == strings.TrimPrefix(ext, ".") {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}
		}

		// Check if entry is a symlink
		linfo, lerr := os.Lstat(entryPath)
		isSymlink := lerr == nil && linfo.Mode()&os.ModeSymlink != 0
		symlinkTarget := ""
		resolvedType := "file"

		if isSymlink {
			target, err := os.Readlink(entryPath)
			if err == nil {
				symlinkTarget = target
			}
			targetInfo, err := os.Stat(entryPath) // follows symlink
			if err == nil {
				if targetInfo.IsDir() {
					resolvedType = "directory"
				} else {
					resolvedType = "file"
				}
			}
		} else {
			if entry.IsDir() {
				resolvedType = "directory"
			} else {
				resolvedType = "file"
			}
		}

		fe := FileEntry{
			Name:      entry.Name(),
			Type:      resolvedType,
			IsSymlink: isSymlink,
		}
		if isSymlink {
			fe.SymlinkTarget = symlinkTarget
		}

		if resolvedType == "file" && !isSymlink {
			if finfo, err := os.Stat(entryPath); err == nil {
				fe.Size = finfo.Size()
			}
		}

		result = append(result, fe)

		// Recurse into directories
		if options != nil && options.Recursive && resolvedType == "directory" {
			maxDepth := options.MaxDepth
			if maxDepth == 0 {
				maxDepth = 100 // Default to prevent stack overflow
			}
			if maxDepth > 0 {
				subOpts := &ListOptions{
					Recursive: true,
					Extension: options.Extension,
					MaxDepth:  maxDepth - 1,
				}
				subEntries, err := lfs.Readdir(lfs.toRelativePath(entryPath), subOpts)
				if err != nil {
					return nil, err
				}
				for _, se := range subEntries {
					se.Name = entry.Name() + "/" + se.Name
					result = append(result, se)
				}
			}
		}
	}

	return result, nil
}

// =============================================================================
// Stat / Exists
// =============================================================================

// Exists checks if a path exists.
func (lfs *LocalFilesystem) Exists(inputPath string) (bool, error) {
	if err := lfs.EnsureReady(); err != nil {
		return false, err
	}
	absolutePath, err := lfs.resolvePath(inputPath)
	if err != nil {
		return false, err
	}
	if err := lfs.assertPathContained(absolutePath); err != nil {
		return false, err
	}
	return FsExists(absolutePath), nil
}

// Stat returns file metadata.
func (lfs *LocalFilesystem) Stat(inputPath string) (*FileStat, error) {
	if err := lfs.EnsureReady(); err != nil {
		return nil, err
	}
	absolutePath, err := lfs.resolvePath(inputPath)
	if err != nil {
		return nil, err
	}
	if err := lfs.assertPathContained(absolutePath); err != nil {
		return nil, err
	}
	result, err := FsStat(absolutePath, inputPath)
	if err != nil {
		return nil, err
	}
	result.Path = lfs.toRelativePath(absolutePath)
	return result, nil
}

// =============================================================================
// Lifecycle
// =============================================================================

// onInit creates the base directory.
func (lfs *LocalFilesystem) onInit() error {
	lfs.GetLogger().Debug("Initializing filesystem", "basePath", lfs.basePath)
	if err := os.MkdirAll(lfs.basePath, 0755); err != nil {
		return err
	}
	lfs.GetLogger().Debug("Filesystem initialized", "basePath", lfs.basePath)
	return nil
}

// onDestroy is a no-op for LocalFilesystem.
func (lfs *LocalFilesystem) onDestroy() error {
	return nil
}

// =============================================================================
// Info & Instructions
// =============================================================================

// GetInfo returns status and metadata for this filesystem.
func (lfs *LocalFilesystem) GetInfo() (*FilesystemInfo, error) {
	status := lfs.GetStatus()
	info := &FilesystemInfo{
		ID:       lfs.id,
		Name:     lfs.name,
		Provider: lfs.provider,
		ReadOnly: lfs.readOnly,
		Status:   &status,
		Metadata: map[string]interface{}{
			"basePath":  lfs.basePath,
			"contained": lfs.contained,
		},
	}
	if len(lfs.allowedPaths) > 0 {
		ap := make([]string, len(lfs.allowedPaths))
		copy(ap, lfs.allowedPaths)
		info.Metadata["allowedPaths"] = ap
	}
	return info, nil
}

// GetInstructions returns usage instructions for this filesystem.
func (lfs *LocalFilesystem) GetInstructions(opts *InstructionsOpts) string {
	var rc *requestcontext.RequestContext
	if opts != nil {
		if typed, ok := opts.RequestContext.(*requestcontext.RequestContext); ok {
			rc = typed
		}
	}
	return workspace.ResolveInstructions(lfs.instructionsOverride, func() string {
		return lfs.getDefaultInstructions()
	}, rc)
}

func (lfs *LocalFilesystem) getDefaultInstructions() string {
	allowedNote := ""
	if len(lfs.allowedPaths) > 0 {
		allowedNote = fmt.Sprintf(" Additionally, the following paths outside basePath are accessible: %s.",
			strings.Join(lfs.allowedPaths, ", "))
	}

	if lfs.contained {
		return fmt.Sprintf(`Local filesystem at %q. Files at workspace path "/foo" are stored at "%s/foo" on disk.%s`,
			lfs.basePath, lfs.basePath, allowedNote)
	}
	return fmt.Sprintf(`Local filesystem rooted at %q. Containment is disabled so absolute paths access the real filesystem. Use paths relative to %q (e.g. "foo/bar.txt") for workspace files. Avoid unnecessary listing "/" as it would traverse the entire host filesystem.%s`,
		lfs.basePath, lfs.basePath, allowedNote)
}
