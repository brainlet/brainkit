// Ported from: packages/core/src/workspace/skills/composite-versioned-skill-source.ts
package skills

import (
	"fmt"
	"strings"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/blobs"
	storagetypes "github.com/brainlet/brainkit/agent-kit/core/storage"
)

// =============================================================================
// Composite Versioned Skill Source
// =============================================================================

// VersionedSkillEntry is a skill entry for the composite source.
// Each entry represents one skill's versioned tree, mounted under a directory name.
type VersionedSkillEntry struct {
	// DirName is the directory name for this skill.
	DirName string
	// Tree is the skill version's file tree manifest.
	Tree *storagetypes.SkillVersionTree
	// VersionCreatedAt is when this version was created.
	VersionCreatedAt time.Time
}

// CompositeVersionedSkillSourceOptions holds configuration for CompositeVersionedSkillSource.
type CompositeVersionedSkillSourceOptions struct {
	// Fallback source for "live" skills that read from the filesystem.
	Fallback SkillSource
	// FallbackSkills are skill directory names that should be served from the fallback source.
	FallbackSkills []string
}

// CompositeVersionedSkillSource composes multiple versioned skill trees into a virtual directory.
//
// Each skill is mounted under a directory name, so the composite source looks like:
//
//	/                           (root - virtual)
//	/brand-guidelines/          (skill 1 root)
//	/brand-guidelines/SKILL.md  (skill 1 files from blob store)
//	/tone-of-voice/             (skill 2 root)
//	/tone-of-voice/SKILL.md     (skill 2 files from blob store)
type CompositeVersionedSkillSource struct {
	sources        map[string]*VersionedSkillSource
	fallback       SkillSource
	fallbackSkills map[string]bool
}

// NewCompositeVersionedSkillSource creates a new CompositeVersionedSkillSource.
func NewCompositeVersionedSkillSource(entries []VersionedSkillEntry, blobStore blobs.BlobStore, opts *CompositeVersionedSkillSourceOptions) *CompositeVersionedSkillSource {
	cs := &CompositeVersionedSkillSource{
		sources:        make(map[string]*VersionedSkillSource),
		fallbackSkills: make(map[string]bool),
	}

	for _, entry := range entries {
		cs.sources[entry.DirName] = NewVersionedSkillSource(entry.Tree, blobStore, entry.VersionCreatedAt)
	}

	if opts != nil {
		cs.fallback = opts.Fallback
		for _, s := range opts.FallbackSkills {
			cs.fallbackSkills[s] = true
		}
	}

	return cs
}

// normalizePath normalizes a path by stripping leading/trailing slashes and dots.
func (cs *CompositeVersionedSkillSource) normalizePath(path string) string {
	normalized := strings.TrimLeft(path, "./\\")
	normalized = strings.TrimRight(normalized, "/\\")
	return normalized
}

// routeResult holds the routing result for a path.
type routeResult struct {
	source  SkillSource
	subPath string
}

// routePath routes a path to the correct source.
func (cs *CompositeVersionedSkillSource) routePath(path string) *routeResult {
	normalized := cs.normalizePath(path)
	if normalized == "" {
		return nil
	}

	segments := strings.SplitN(normalized, "/", 2)
	skillDir := segments[0]
	subPath := ""
	if len(segments) > 1 {
		subPath = segments[1]
	}

	// Check if this skill should use the fallback source
	if cs.fallbackSkills[skillDir] && cs.fallback != nil {
		return &routeResult{source: cs.fallback, subPath: normalized}
	}

	// Check if this skill has a versioned source
	if versionedSource, ok := cs.sources[skillDir]; ok {
		return &routeResult{source: versionedSource, subPath: subPath}
	}

	// Try the fallback for unknown paths
	if cs.fallback != nil {
		return &routeResult{source: cs.fallback, subPath: normalized}
	}

	return nil
}

// Exists checks if a path exists.
func (cs *CompositeVersionedSkillSource) Exists(path string) (bool, error) {
	normalized := cs.normalizePath(path)
	if normalized == "" {
		return true, nil
	}

	route := cs.routePath(path)
	if route == nil {
		return false, nil
	}

	return route.source.Exists(route.subPath)
}

// Stat gets file/directory stat info.
func (cs *CompositeVersionedSkillSource) Stat(path string) (*SkillSourceStat, error) {
	normalized := cs.normalizePath(path)
	if normalized == "" {
		now := time.Now()
		return &SkillSourceStat{
			Name:       ".",
			Type:       "directory",
			Size:       0,
			CreatedAt:  now,
			ModifiedAt: now,
		}, nil
	}

	route := cs.routePath(path)
	if route == nil {
		return nil, fmt.Errorf("path not found in composite skill source: %s", path)
	}

	return route.source.Stat(route.subPath)
}

// ReadFile reads a file's contents.
func (cs *CompositeVersionedSkillSource) ReadFile(path string) ([]byte, error) {
	route := cs.routePath(path)
	if route == nil {
		return nil, fmt.Errorf("file not found in composite skill source: %s", path)
	}

	return route.source.ReadFile(route.subPath)
}

// Readdir lists directory contents.
func (cs *CompositeVersionedSkillSource) Readdir(path string) ([]SkillSourceEntry, error) {
	normalized := cs.normalizePath(path)

	// Root: list all mounted skill directories
	if normalized == "" {
		var entries []SkillSourceEntry
		seen := make(map[string]bool)

		for dirName := range cs.sources {
			entries = append(entries, SkillSourceEntry{Name: dirName, Type: "directory"})
			seen[dirName] = true
		}

		for dirName := range cs.fallbackSkills {
			if !seen[dirName] {
				entries = append(entries, SkillSourceEntry{Name: dirName, Type: "directory"})
			}
		}

		return entries, nil
	}

	route := cs.routePath(path)
	if route == nil {
		return nil, fmt.Errorf("directory not found in composite skill source: %s", path)
	}

	return route.source.Readdir(route.subPath)
}
