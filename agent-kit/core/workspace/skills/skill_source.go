// Ported from: packages/core/src/workspace/skills/skill-source.ts
package skills

import "time"

// =============================================================================
// Skill Source Types
// =============================================================================

// SkillSourceStat holds file stat info for skill sources.
// Aligned with FileStat from WorkspaceFilesystem.
type SkillSourceStat struct {
	// Name is the file or directory name.
	Name string
	// Type is "file" or "directory".
	Type string
	// Size is the size in bytes (0 for directories).
	Size int64
	// CreatedAt is the creation time.
	CreatedAt time.Time
	// ModifiedAt is the last modification time.
	ModifiedAt time.Time
	// MimeType is the MIME type (for files).
	MimeType string
}

// SkillSourceEntry is a directory entry from readdir.
type SkillSourceEntry struct {
	// Name is the entry name (file or directory name).
	Name string
	// Type is the entry type: "file" or "directory".
	Type string
	// IsSymlink indicates whether this entry is a symbolic link.
	IsSymlink bool
}

// SkillSource is a minimal read-only interface for loading skills.
// This is the subset of WorkspaceFilesystem methods needed for skill discovery.
// Implementations can be backed by workspace filesystem, local disk, or other sources.
type SkillSource interface {
	// Exists checks if a path exists.
	Exists(path string) (bool, error)
	// Stat gets file/directory stat info.
	Stat(path string) (*SkillSourceStat, error)
	// ReadFile reads a file's contents.
	ReadFile(path string) ([]byte, error)
	// Readdir lists directory contents.
	Readdir(path string) ([]SkillSourceEntry, error)
}
