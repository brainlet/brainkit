// Ported from: packages/core/src/workspace/skills/versioned-skill-source.ts
package skills

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/blobs"
	storagetypes "github.com/brainlet/brainkit/agent-kit/core/storage"
)

// =============================================================================
// Versioned Skill Source
// =============================================================================

// VersionedSkillSource is a SkillSource implementation that reads skill files
// from a versioned content-addressable blob store, using a SkillVersionTree manifest.
//
// This is used by production agents to read from published skill versions
// rather than the live filesystem.
type VersionedSkillSource struct {
	tree             *storagetypes.SkillVersionTree
	blobStore        blobs.BlobStore
	versionCreatedAt time.Time
	directories      map[string]bool
}

// NewVersionedSkillSource creates a new VersionedSkillSource.
func NewVersionedSkillSource(tree *storagetypes.SkillVersionTree, blobStore blobs.BlobStore, versionCreatedAt time.Time) *VersionedSkillSource {
	vs := &VersionedSkillSource{
		tree:             tree,
		blobStore:        blobStore,
		versionCreatedAt: versionCreatedAt,
	}
	vs.directories = vs.computeDirectories()
	return vs
}

// computeDirectories computes all directory paths implied by the file tree.
func (vs *VersionedSkillSource) computeDirectories() map[string]bool {
	dirs := map[string]bool{
		"":  true,
		".": true,
	}
	for filePath := range vs.tree.Entries {
		parts := strings.Split(filePath, "/")
		for i := 1; i < len(parts); i++ {
			dir := strings.Join(parts[:i], "/")
			dirs[dir] = true
		}
	}
	return dirs
}

// normalizePath normalizes a path by stripping leading/trailing slashes and dots.
func (vs *VersionedSkillSource) normalizePath(path string) string {
	normalized := strings.TrimLeft(path, "./\\")
	normalized = strings.TrimRight(normalized, "/\\")
	return normalized
}

// Exists checks if a path exists.
func (vs *VersionedSkillSource) Exists(path string) (bool, error) {
	normalized := vs.normalizePath(path)
	if _, ok := vs.tree.Entries[normalized]; ok {
		return true, nil
	}
	return vs.directories[normalized], nil
}

// Stat gets file/directory stat info.
func (vs *VersionedSkillSource) Stat(path string) (*SkillSourceStat, error) {
	normalized := vs.normalizePath(path)
	name := normalized
	if idx := strings.LastIndex(normalized, "/"); idx >= 0 {
		name = normalized[idx+1:]
	}
	if name == "" {
		name = "."
	}

	// Check if it's a file in the tree
	if entry, ok := vs.tree.Entries[normalized]; ok {
		mimeType := ""
		if entry.MimeType != nil {
			mimeType = *entry.MimeType
		}
		return &SkillSourceStat{
			Name:       name,
			Type:       "file",
			Size:       int64(entry.Size),
			CreatedAt:  vs.versionCreatedAt,
			ModifiedAt: vs.versionCreatedAt,
			MimeType:   mimeType,
		}, nil
	}

	// Check if it's a directory
	if vs.directories[normalized] {
		return &SkillSourceStat{
			Name:       name,
			Type:       "directory",
			Size:       0,
			CreatedAt:  vs.versionCreatedAt,
			ModifiedAt: vs.versionCreatedAt,
		}, nil
	}

	return nil, fmt.Errorf("path not found in skill version tree: %s", path)
}

// ReadFile reads a file's contents.
func (vs *VersionedSkillSource) ReadFile(path string) ([]byte, error) {
	normalized := vs.normalizePath(path)
	entry, ok := vs.tree.Entries[normalized]
	if !ok {
		return nil, fmt.Errorf("file not found in skill version tree: %s", path)
	}

	blob, err := vs.blobStore.Get(context.Background(), entry.BlobHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get blob %s: %w", entry.BlobHash, err)
	}
	if blob == nil {
		return nil, fmt.Errorf("blob not found for hash %s (file: %s)", entry.BlobHash, path)
	}

	// Decode base64-encoded binary content
	if entry.Encoding != nil && *entry.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(string(blob.Content))
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 content for %s: %w", path, err)
		}
		return decoded, nil
	}

	return blob.Content, nil
}

// Readdir lists directory contents.
func (vs *VersionedSkillSource) Readdir(path string) ([]SkillSourceEntry, error) {
	normalized := vs.normalizePath(path)
	if !vs.directories[normalized] {
		return nil, fmt.Errorf("directory not found in skill version tree: %s", path)
	}

	prefix := ""
	if normalized != "" {
		prefix = normalized + "/"
	}

	seen := make(map[string]bool)
	var entries []SkillSourceEntry

	for filePath := range vs.tree.Entries {
		if !strings.HasPrefix(filePath, prefix) {
			continue
		}

		remaining := filePath[len(prefix):]
		parts := strings.SplitN(remaining, "/", 2)
		nextSegment := parts[0]
		if nextSegment == "" || seen[nextSegment] {
			continue
		}
		seen[nextSegment] = true

		isDirectory := len(parts) > 1
		entryType := "file"
		if isDirectory {
			entryType = "directory"
		}
		entries = append(entries, SkillSourceEntry{
			Name: nextSegment,
			Type: entryType,
		})
	}

	return entries, nil
}
