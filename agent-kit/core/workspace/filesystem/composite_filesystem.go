// Ported from: packages/core/src/workspace/filesystem/composite-filesystem.ts
package filesystem

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	workspace "github.com/brainlet/brainkit/agent-kit/core/workspace"
)

// =============================================================================
// CompositeFilesystem Config
// =============================================================================

// CompositeFilesystemConfig holds configuration for CompositeFilesystem.
type CompositeFilesystemConfig struct {
	// Mounts maps mount paths to filesystem instances.
	Mounts map[string]WorkspaceFilesystem
}

// resolvedMount is the result of resolving a path to a mounted filesystem.
type resolvedMount struct {
	fs        WorkspaceFilesystem
	fsPath    string
	mountPath string
}

// =============================================================================
// CompositeFilesystem
// =============================================================================

// CompositeFilesystem routes file operations to mounted filesystems based on path.
// Creates a unified filesystem view by combining multiple filesystems at different
// mount points.
type CompositeFilesystem struct {
	id       string
	name     string
	provider string
	readOnly bool
	status   ProviderStatus
	mounts   map[string]WorkspaceFilesystem
}

// NewCompositeFilesystem creates a new CompositeFilesystem.
func NewCompositeFilesystem(config CompositeFilesystemConfig) (*CompositeFilesystem, error) {
	if len(config.Mounts) == 0 {
		return nil, fmt.Errorf("CompositeFilesystem requires at least one mount")
	}

	cfs := &CompositeFilesystem{
		id:       "cfs-" + strconv.FormatInt(time.Now().UnixMilli(), 36),
		name:     "CompositeFilesystem",
		provider: "composite",
		status:   ProviderStatusReady,
		mounts:   make(map[string]WorkspaceFilesystem),
	}

	for p, fs := range config.Mounts {
		normalized := cfs.normalizePath(p)
		cfs.mounts[normalized] = fs
	}

	// Composite is read-only when every mount is read-only
	allReadOnly := true
	for _, fs := range cfs.mounts {
		if !fs.ReadOnly() {
			allReadOnly = false
			break
		}
	}
	cfs.readOnly = allReadOnly

	// Validate no nested mount paths
	mountPaths := cfs.MountPaths()
	for _, a := range mountPaths {
		for _, b := range mountPaths {
			if a != b && strings.HasPrefix(b, a+"/") {
				return nil, fmt.Errorf("Nested mount paths are not supported: %q is nested under %q", b, a)
			}
		}
	}

	return cfs, nil
}

// =============================================================================
// Identity
// =============================================================================

// ID returns the unique identifier.
func (cfs *CompositeFilesystem) ID() string { return cfs.id }

// Name returns "CompositeFilesystem".
func (cfs *CompositeFilesystem) Name() string { return cfs.name }

// Provider returns "composite".
func (cfs *CompositeFilesystem) Provider() string { return cfs.provider }

// ReadOnly returns true if all mounted filesystems are read-only.
func (cfs *CompositeFilesystem) ReadOnly() bool { return cfs.readOnly }

// Icon returns nil.
func (cfs *CompositeFilesystem) Icon() *FilesystemIcon { return nil }

// DisplayName returns empty string.
func (cfs *CompositeFilesystem) DisplayName() string { return "" }

// Description returns empty string.
func (cfs *CompositeFilesystem) Description() string { return "" }

// BasePath returns empty string (composite has no single base path).
func (cfs *CompositeFilesystem) BasePath() string { return "" }

// Status returns the current provider status.
func (cfs *CompositeFilesystem) Status() ProviderStatus { return cfs.status }

// MountPaths returns all mount paths.
func (cfs *CompositeFilesystem) MountPaths() []string {
	paths := make([]string, 0, len(cfs.mounts))
	for p := range cfs.mounts {
		paths = append(paths, p)
	}
	return paths
}

// GetMount returns the filesystem mounted at the given path, or nil.
func (cfs *CompositeFilesystem) GetMount(mountPath string) WorkspaceFilesystem {
	return cfs.mounts[cfs.normalizePath(mountPath)]
}

// GetFilesystemForPath returns the underlying filesystem for a given path.
func (cfs *CompositeFilesystem) GetFilesystemForPath(p string) WorkspaceFilesystem {
	r := cfs.resolveMount(p)
	if r == nil {
		return nil
	}
	return r.fs
}

// GetMountPathForPath returns the mount path for a given path.
func (cfs *CompositeFilesystem) GetMountPathForPath(p string) string {
	r := cfs.resolveMount(p)
	if r == nil {
		return ""
	}
	return r.mountPath
}

// ResolveAbsolutePath resolves a workspace-relative path to an absolute disk path.
func (cfs *CompositeFilesystem) ResolveAbsolutePath(p string) string {
	r := cfs.resolveMount(p)
	if r == nil {
		return ""
	}
	return r.fs.ResolveAbsolutePath(r.fsPath)
}

// =============================================================================
// Path Resolution (Internal)
// =============================================================================

func (cfs *CompositeFilesystem) normalizePath(p string) string {
	if p == "" || p == "/" {
		return "/"
	}
	n := path.Clean(p)
	if !strings.HasPrefix(n, "/") {
		n = "/" + n
	}
	if len(n) > 1 && strings.HasSuffix(n, "/") {
		n = n[:len(n)-1]
	}
	return n
}

func (cfs *CompositeFilesystem) resolveMount(p string) *resolvedMount {
	normalized := cfs.normalizePath(p)
	var best *struct {
		mountPath string
		fs        WorkspaceFilesystem
	}

	for mountPath, fs := range cfs.mounts {
		if normalized == mountPath || strings.HasPrefix(normalized, mountPath+"/") {
			if best == nil || len(mountPath) > len(best.mountPath) {
				best = &struct {
					mountPath string
					fs        WorkspaceFilesystem
				}{mountPath, fs}
			}
		}
	}

	if best == nil {
		return nil
	}

	fsPath := normalized[len(best.mountPath):]
	if fsPath == "" {
		fsPath = "/"
	}
	if !strings.HasPrefix(fsPath, "/") {
		fsPath = "/" + fsPath
	}

	return &resolvedMount{
		fs:        best.fs,
		fsPath:    fsPath,
		mountPath: best.mountPath,
	}
}

func (cfs *CompositeFilesystem) getVirtualEntries(p string) []FileEntry {
	normalized := cfs.normalizePath(p)
	if cfs.resolveMount(normalized) != nil {
		return nil
	}

	entriesMap := make(map[string]FileEntry)
	for mountPath, fs := range cfs.mounts {
		isUnder := false
		if normalized == "/" {
			isUnder = strings.HasPrefix(mountPath, "/")
		} else {
			isUnder = strings.HasPrefix(mountPath, normalized+"/")
		}

		if isUnder {
			var remaining string
			if normalized == "/" {
				remaining = mountPath[1:]
			} else {
				remaining = mountPath[len(normalized)+1:]
			}
			parts := strings.SplitN(remaining, "/", 2)
			next := parts[0]
			if next != "" {
				if _, exists := entriesMap[next]; !exists {
					entry := FileEntry{Name: next, Type: "directory"}

					// If it's a direct mount point, include filesystem metadata
					isDirectMount := remaining == next
					if isDirectMount {
						status := fs.Status()
						entry.Mount = &FileEntryMount{
							Provider: fs.Provider(),
							Status:   &status,
						}
					}

					entriesMap[next] = entry
				}
			}
		}
	}

	if len(entriesMap) == 0 {
		return nil
	}

	result := make([]FileEntry, 0, len(entriesMap))
	for _, entry := range entriesMap {
		result = append(result, entry)
	}
	return result
}

func (cfs *CompositeFilesystem) isVirtualPath(p string) bool {
	normalized := cfs.normalizePath(p)
	if normalized == "/" {
		if _, has := cfs.mounts["/"]; !has {
			return true
		}
	}
	for mountPath := range cfs.mounts {
		if strings.HasPrefix(mountPath, normalized+"/") {
			return true
		}
	}
	return false
}

func (cfs *CompositeFilesystem) assertWritable(fs WorkspaceFilesystem, p, operation string) error {
	if fs.ReadOnly() {
		return workspace.NewPermissionError(p, operation+" (filesystem is read-only)")
	}
	return nil
}

// =============================================================================
// WorkspaceFilesystem Implementation
// =============================================================================

// Init initializes all mounted filesystems.
func (cfs *CompositeFilesystem) Init() error {
	cfs.status = ProviderStatusInitializing
	for mountPath, fs := range cfs.mounts {
		if err := CallLifecycle(fs, "init"); err != nil {
			// Individual mount failed — log but continue
			_ = fmt.Errorf("[CompositeFilesystem] Mount %q failed to initialize: %v", mountPath, err)
		}
	}
	cfs.status = ProviderStatusReady
	return nil
}

// Destroy destroys all mounted filesystems.
func (cfs *CompositeFilesystem) Destroy() error {
	cfs.status = ProviderStatusDestroying
	var errs []error
	for _, fs := range cfs.mounts {
		if err := CallLifecycle(fs, "destroy"); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		cfs.status = ProviderStatusError
		return errors.Join(errs...)
	}
	cfs.status = ProviderStatusDestroyed
	return nil
}

// ReadFile reads a file from the appropriate mounted filesystem.
func (cfs *CompositeFilesystem) ReadFile(p string, options *ReadOptions) (interface{}, error) {
	r := cfs.resolveMount(p)
	if r == nil {
		return nil, fmt.Errorf("No mount for path: %s", p)
	}
	return r.fs.ReadFile(r.fsPath, options)
}

// WriteFile writes a file to the appropriate mounted filesystem.
func (cfs *CompositeFilesystem) WriteFile(p string, content interface{}, options *WriteOptions) error {
	r := cfs.resolveMount(p)
	if r == nil {
		return fmt.Errorf("No mount for path: %s", p)
	}
	if err := cfs.assertWritable(r.fs, p, "writeFile"); err != nil {
		return err
	}
	return r.fs.WriteFile(r.fsPath, content, options)
}

// AppendFile appends content to a file.
func (cfs *CompositeFilesystem) AppendFile(p string, content interface{}) error {
	r := cfs.resolveMount(p)
	if r == nil {
		return fmt.Errorf("No mount for path: %s", p)
	}
	if err := cfs.assertWritable(r.fs, p, "appendFile"); err != nil {
		return err
	}
	return r.fs.AppendFile(r.fsPath, content)
}

// DeleteFile deletes a file.
func (cfs *CompositeFilesystem) DeleteFile(p string, options *RemoveOptions) error {
	r := cfs.resolveMount(p)
	if r == nil {
		return fmt.Errorf("No mount for path: %s", p)
	}
	if err := cfs.assertWritable(r.fs, p, "deleteFile"); err != nil {
		return err
	}
	return r.fs.DeleteFile(r.fsPath, options)
}

// CopyFile copies a file, supporting cross-mount copies.
func (cfs *CompositeFilesystem) CopyFile(src, dest string, options *CopyOptions) error {
	srcR := cfs.resolveMount(src)
	destR := cfs.resolveMount(dest)
	if srcR == nil {
		return fmt.Errorf("No mount for source: %s", src)
	}
	if destR == nil {
		return fmt.Errorf("No mount for dest: %s", dest)
	}
	if err := cfs.assertWritable(destR.fs, dest, "copyFile"); err != nil {
		return err
	}

	// Same mount — delegate
	if srcR.mountPath == destR.mountPath {
		return srcR.fs.CopyFile(srcR.fsPath, destR.fsPath, options)
	}

	// Cross-mount copy — read then write
	content, err := srcR.fs.ReadFile(srcR.fsPath, nil)
	if err != nil {
		return err
	}
	overwrite := true
	if options != nil {
		overwrite = options.Overwrite
	}
	return destR.fs.WriteFile(destR.fsPath, content, &WriteOptions{Overwrite: &overwrite})
}

// MoveFile moves a file, supporting cross-mount moves.
func (cfs *CompositeFilesystem) MoveFile(src, dest string, options *CopyOptions) error {
	srcR := cfs.resolveMount(src)
	destR := cfs.resolveMount(dest)
	if srcR == nil {
		return fmt.Errorf("No mount for source: %s", src)
	}
	if destR == nil {
		return fmt.Errorf("No mount for dest: %s", dest)
	}
	if err := cfs.assertWritable(destR.fs, dest, "moveFile"); err != nil {
		return err
	}
	if err := cfs.assertWritable(srcR.fs, src, "moveFile"); err != nil {
		return err
	}

	// Same mount — delegate
	if srcR.mountPath == destR.mountPath {
		return srcR.fs.MoveFile(srcR.fsPath, destR.fsPath, options)
	}

	// Cross-mount move — copy then delete
	if err := cfs.CopyFile(src, dest, options); err != nil {
		return err
	}
	return srcR.fs.DeleteFile(srcR.fsPath, nil)
}

// Readdir reads a directory, returning virtual entries at mount boundaries.
func (cfs *CompositeFilesystem) Readdir(p string, options *ListOptions) ([]FileEntry, error) {
	virtual := cfs.getVirtualEntries(p)
	if virtual != nil {
		return virtual, nil
	}

	r := cfs.resolveMount(p)
	if r == nil {
		return nil, fmt.Errorf("No mount for path: %s", p)
	}
	return r.fs.Readdir(r.fsPath, options)
}

// Mkdir creates a directory.
func (cfs *CompositeFilesystem) Mkdir(p string, options *MkdirOptions) error {
	r := cfs.resolveMount(p)
	if r == nil {
		return fmt.Errorf("No mount for path: %s", p)
	}
	if err := cfs.assertWritable(r.fs, p, "mkdir"); err != nil {
		return err
	}
	return r.fs.Mkdir(r.fsPath, options)
}

// Rmdir removes a directory.
func (cfs *CompositeFilesystem) Rmdir(p string, options *RemoveOptions) error {
	r := cfs.resolveMount(p)
	if r == nil {
		return fmt.Errorf("No mount for path: %s", p)
	}
	if err := cfs.assertWritable(r.fs, p, "rmdir"); err != nil {
		return err
	}
	return r.fs.Rmdir(r.fsPath, options)
}

// Exists checks if a path exists.
func (cfs *CompositeFilesystem) Exists(p string) (bool, error) {
	if cfs.isVirtualPath(p) {
		return true, nil
	}
	r := cfs.resolveMount(p)
	if r == nil {
		return false, nil
	}
	// Mount point root always exists
	if r.fsPath == "/" {
		return true, nil
	}
	return r.fs.Exists(r.fsPath)
}

// Stat returns file metadata.
func (cfs *CompositeFilesystem) Stat(p string) (*FileStat, error) {
	normalized := cfs.normalizePath(p)

	if cfs.isVirtualPath(p) {
		parts := strings.Split(normalized, "/")
		var filtered []string
		for _, part := range parts {
			if part != "" {
				filtered = append(filtered, part)
			}
		}
		name := ""
		if len(filtered) > 0 {
			name = filtered[len(filtered)-1]
		}
		now := time.Now()
		return &FileStat{
			Name:       name,
			Path:       normalized,
			Type:       "directory",
			Size:       0,
			CreatedAt:  now,
			ModifiedAt: now,
		}, nil
	}

	r := cfs.resolveMount(p)
	if r == nil {
		return nil, fmt.Errorf("No mount for path: %s", p)
	}

	// Mount point root always returns directory stat
	if r.fsPath == "/" {
		parts := strings.Split(normalized, "/")
		var filtered []string
		for _, part := range parts {
			if part != "" {
				filtered = append(filtered, part)
			}
		}
		name := ""
		if len(filtered) > 0 {
			name = filtered[len(filtered)-1]
		}
		now := time.Now()
		return &FileStat{
			Name:       name,
			Path:       normalized,
			Type:       "directory",
			Size:       0,
			CreatedAt:  now,
			ModifiedAt: now,
		}, nil
	}

	return r.fs.Stat(r.fsPath)
}

// GetInfo returns status and metadata for this composite filesystem.
func (cfs *CompositeFilesystem) GetInfo() (*FilesystemInfo, error) {
	mountInfos := make(map[string]interface{})
	for mountPath, fs := range cfs.mounts {
		info, _ := fs.GetInfo()
		if info != nil {
			mountInfos[mountPath] = info
		}
	}

	return &FilesystemInfo{
		ID:       cfs.id,
		Name:     cfs.name,
		Provider: cfs.provider,
		Status:   &cfs.status,
		ReadOnly: cfs.readOnly,
		Metadata: map[string]interface{}{
			"mounts": mountInfos,
		},
	}, nil
}

// GetInstructions returns usage instructions describing the mounted filesystems.
func (cfs *CompositeFilesystem) GetInstructions(_ *InstructionsOpts) string {
	var lines []string
	for mountPath, fs := range cfs.mounts {
		name := fs.Provider()
		access := "(read-write)"
		if fs.ReadOnly() {
			access = "(read-only)"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s %s", mountPath, name, access))
	}
	return "Filesystem mount points:\n" + strings.Join(lines, "\n")
}

// GetMountConfig returns nil (composite doesn't have a single mount config).
func (cfs *CompositeFilesystem) GetMountConfig() *FilesystemMountConfig { return nil }

// =============================================================================
// CallLifecycle helper (filesystem-local)
// =============================================================================

// CallLifecycle dispatches a lifecycle method on a WorkspaceFilesystem.
func CallLifecycle(fs WorkspaceFilesystem, method string) error {
	switch method {
	case "init":
		return fs.Init()
	case "destroy":
		return fs.Destroy()
	}
	return nil
}
