// Ported from: packages/core/src/workspace/skills/publish.ts
package skills

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/blobs"
	storagetypes "github.com/brainlet/brainkit/agent-kit/core/storage"
	graymatter "github.com/brainlet/brainkit/gray-matter"
)

// =============================================================================
// Publish Types
// =============================================================================

// SkillPublishResult holds the result of collecting a skill's filesystem tree.
type SkillPublishResult struct {
	// Snapshot holds denormalized snapshot fields parsed from SKILL.md frontmatter.
	Snapshot SkillPublishSnapshot
	// Tree is the content-addressable file tree manifest.
	Tree storagetypes.SkillVersionTree
	// Blobs are the blob entries to store (already deduplicated by hash).
	Blobs []storagetypes.StorageBlobEntry
}

// SkillPublishSnapshot holds the denormalized fields from a SKILL.md frontmatter.
type SkillPublishSnapshot struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Instructions  string                 `json:"instructions"`
	License       string                 `json:"license,omitempty"`
	Compatibility interface{}            `json:"compatibility,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	References    []string               `json:"references,omitempty"`
	Scripts       []string               `json:"scripts,omitempty"`
	Assets        []string               `json:"assets,omitempty"`
}

// =============================================================================
// Internal Helpers
// =============================================================================

// hashContent computes SHA-256 hex hash of content.
func hashContent(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h)
}

// walkedFile is a file found during directory walking.
type walkedFile struct {
	path     string
	content  []byte
	isBinary bool
}

// walkSkillDirectory recursively walks a directory in a SkillSource,
// returning all files with their relative paths and content.
func walkSkillDirectory(source SkillSource, basePath, currentPath string) ([]walkedFile, error) {
	entries, err := source.Readdir(currentPath)
	if err != nil {
		return nil, err
	}

	var files []walkedFile
	for _, entry := range entries {
		entryPath := joinSkillPath(currentPath, entry.Name)

		if entry.Type == "directory" {
			subFiles, err := walkSkillDirectory(source, basePath, entryPath)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		} else {
			rawContent, err := source.ReadFile(entryPath)
			if err != nil {
				return nil, err
			}

			relativePath := entryPath
			if len(basePath) > 0 && strings.HasPrefix(entryPath, basePath+"/") {
				relativePath = entryPath[len(basePath)+1:]
			}

			mimeType := DetectMimeType(entry.Name)
			isBinary := IsBinaryMimeType(mimeType)

			files = append(files, walkedFile{
				path:     relativePath,
				content:  rawContent,
				isBinary: isBinary,
			})
		}
	}

	return files, nil
}

// joinSkillPath joins path segments using forward slashes.
func joinSkillPath(segments ...string) string {
	var result []string
	for i, seg := range segments {
		if i == 0 {
			result = append(result, strings.TrimRight(seg, "/"))
		} else {
			trimmed := strings.Trim(seg, "/")
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return strings.Join(result, "/")
}

// collectSubdirPaths collects file paths under a specific subdirectory prefix.
func collectSubdirPaths(allPaths []string, subdir string) []string {
	prefix := subdir + "/"
	var result []string
	for _, p := range allPaths {
		if strings.HasPrefix(p, prefix) {
			result = append(result, p[len(prefix):])
		}
	}
	return result
}

// =============================================================================
// Public API
// =============================================================================

// CollectSkillForPublish collects a skill from a SkillSource for publishing.
// Walks the skill directory, hashes all files, parses SKILL.md frontmatter,
// and returns everything needed to create a new version.
func CollectSkillForPublish(source SkillSource, skillPath string) (*SkillPublishResult, error) {
	// 1. Walk the skill directory recursively
	files, err := walkSkillDirectory(source, skillPath, skillPath)
	if err != nil {
		return nil, fmt.Errorf("failed to walk skill directory: %w", err)
	}

	// 2. Build tree entries and blob entries
	treeEntries := make(map[string]storagetypes.SkillVersionTreeEntry)
	blobMap := make(map[string]storagetypes.StorageBlobEntry)
	now := time.Now()

	for _, file := range files {
		hash := hashContent(file.content)
		mimeType := DetectMimeType(file.path)

		if file.isBinary {
			size := len(file.content)
			base64Content := base64.StdEncoding.EncodeToString(file.content)

			encoding := "base64"
			var mimeTypePtr *string
			if mimeType != "" {
				mimeTypePtr = &mimeType
			}

			treeEntries[file.path] = storagetypes.SkillVersionTreeEntry{
				BlobHash: hash,
				Size:     size,
				MimeType: mimeTypePtr,
				Encoding: &encoding,
			}

			if _, exists := blobMap[hash]; !exists {
				blobMap[hash] = storagetypes.StorageBlobEntry{
					Hash:      hash,
					Content:   base64Content,
					Size:      size,
					MimeType:  mimeTypePtr,
					CreatedAt: now,
				}
			}
		} else {
			content := string(file.content)
			size := len(file.content)

			var mimeTypePtr *string
			if mimeType != "" {
				mimeTypePtr = &mimeType
			}

			treeEntries[file.path] = storagetypes.SkillVersionTreeEntry{
				BlobHash: hash,
				Size:     size,
				MimeType: mimeTypePtr,
			}

			if _, exists := blobMap[hash]; !exists {
				blobMap[hash] = storagetypes.StorageBlobEntry{
					Hash:      hash,
					Content:   content,
					Size:      size,
					MimeType:  mimeTypePtr,
					CreatedAt: now,
				}
			}
		}
	}

	tree := storagetypes.SkillVersionTree{Entries: treeEntries}

	var blobSlice []storagetypes.StorageBlobEntry
	for _, b := range blobMap {
		blobSlice = append(blobSlice, b)
	}

	// 3. Parse SKILL.md for denormalized fields
	var skillMdContent []byte
	for _, f := range files {
		if f.path == "SKILL.md" {
			skillMdContent = f.content
			break
		}
	}
	if skillMdContent == nil {
		return nil, fmt.Errorf("SKILL.md not found in %s", skillPath)
	}

	parsed, err := graymatter.Parse(string(skillMdContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse SKILL.md frontmatter: %w", err)
	}
	frontmatter := parsed.DataMap()
	body := parsed.Content

	// 4. Discover references/, scripts/, assets/ subdirectories
	var allPaths []string
	for _, f := range files {
		allPaths = append(allPaths, f.path)
	}
	references := collectSubdirPaths(allPaths, "references")
	scripts := collectSubdirPaths(allPaths, "scripts")
	assets := collectSubdirPaths(allPaths, "assets")

	// 5. Build snapshot
	snapshot := SkillPublishSnapshot{
		Name:         getString(frontmatter, "name"),
		Description:  getString(frontmatter, "description"),
		Instructions: strings.TrimSpace(body),
		License:      getString(frontmatter, "license"),
	}
	if v, ok := frontmatter["compatibility"]; ok {
		snapshot.Compatibility = v
	}
	if v, ok := frontmatter["metadata"].(map[string]interface{}); ok {
		snapshot.Metadata = v
	}
	if len(references) > 0 {
		snapshot.References = references
	}
	if len(scripts) > 0 {
		snapshot.Scripts = scripts
	}
	if len(assets) > 0 {
		snapshot.Assets = assets
	}

	return &SkillPublishResult{
		Snapshot: snapshot,
		Tree:     tree,
		Blobs:    blobSlice,
	}, nil
}

// PublishSkillFromSource publishes a skill: collect files, store blobs, create version.
func PublishSkillFromSource(source SkillSource, skillPath string, blobStore blobs.BlobStore) (*SkillPublishResult, error) {
	result, err := CollectSkillForPublish(source, skillPath)
	if err != nil {
		return nil, err
	}

	// Convert to blobs.StorageBlobEntry for the blob store
	var blobEntries []blobs.StorageBlobEntry
	for _, b := range result.Blobs {
		blobEntries = append(blobEntries, blobs.StorageBlobEntry{
			Hash:    b.Hash,
			Content: []byte(b.Content),
		})
	}

	if err := blobStore.PutMany(context.Background(), blobEntries); err != nil {
		return nil, fmt.Errorf("failed to store blobs: %w", err)
	}

	return result, nil
}

// getString safely gets a string value from a map.
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
