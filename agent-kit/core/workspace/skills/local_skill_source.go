// Ported from: packages/core/src/workspace/skills/local-skill-source.ts
package skills

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// =============================================================================
// Local Skill Source
// =============================================================================

// LocalSkillSourceOptions holds configuration for LocalSkillSource.
type LocalSkillSourceOptions struct {
	// BasePath for resolving relative skill paths. Defaults to cwd.
	BasePath string
}

// LocalSkillSource is a read-only skill source backed by the local filesystem.
// Uses os package to read skills directly from disk.
type LocalSkillSource struct {
	basePath string
}

// NewLocalSkillSource creates a new LocalSkillSource.
func NewLocalSkillSource(opts *LocalSkillSourceOptions) *LocalSkillSource {
	basePath := ""
	if opts != nil && opts.BasePath != "" {
		basePath = opts.BasePath
	} else {
		basePath, _ = os.Getwd()
	}
	return &LocalSkillSource{basePath: basePath}
}

// resolvePath resolves a path relative to the base path.
func (s *LocalSkillSource) resolvePath(skillPath string) string {
	if filepath.IsAbs(skillPath) {
		return skillPath
	}
	return filepath.Join(s.basePath, skillPath)
}

// Exists checks if a path exists.
func (s *LocalSkillSource) Exists(path string) (bool, error) {
	resolved := s.resolvePath(path)
	_, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Stat gets file/directory stat info.
func (s *LocalSkillSource) Stat(path string) (*SkillSourceStat, error) {
	resolved := s.resolvePath(path)
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}

	entryType := "file"
	if info.IsDir() {
		entryType = "directory"
	}

	return &SkillSourceStat{
		Name:       info.Name(),
		Type:       entryType,
		Size:       info.Size(),
		CreatedAt:  info.ModTime(), // Go doesn't expose creation time on all platforms
		ModifiedAt: info.ModTime(),
	}, nil
}

// ReadFile reads a file's contents.
func (s *LocalSkillSource) ReadFile(path string) ([]byte, error) {
	resolved := s.resolvePath(path)
	return os.ReadFile(resolved)
}

// Readdir lists directory contents.
func (s *LocalSkillSource) Readdir(path string) ([]SkillSourceEntry, error) {
	resolved := s.resolvePath(path)
	dirEntries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}

	var entries []SkillSourceEntry
	for _, de := range dirEntries {
		entryType := "file"
		if de.IsDir() {
			entryType = "directory"
		}

		isSymlink := false
		if de.Type()&os.ModeSymlink != 0 {
			isSymlink = true
		}

		entries = append(entries, SkillSourceEntry{
			Name:      de.Name(),
			Type:      entryType,
			IsSymlink: isSymlink,
		})
	}

	return entries, nil
}

// =============================================================================
// Text File Detection
// =============================================================================

// textExtensions are file extensions considered text files.
var textExtensions = map[string]bool{
	".md": true, ".txt": true, ".json": true, ".yaml": true, ".yml": true,
	".sh": true, ".py": true, ".js": true, ".ts": true, ".html": true,
	".css": true, ".xml": true, ".toml": true, ".ini": true, ".cfg": true,
	".csv": true, ".svg": true, ".go": true, ".rs": true, ".java": true,
	".rb": true, ".php": true, ".c": true, ".cpp": true, ".h": true,
	".hpp": true, ".tsx": true, ".jsx": true, ".mjs": true, ".cjs": true,
}

// isTextFile checks if a file path is a text file based on extension.
func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return textExtensions[ext]
}

// =============================================================================
// MIME Type Detection
// =============================================================================

// mimeTypes maps file extensions to MIME types.
var mimeTypes = map[string]string{
	".md":   "text/markdown",
	".txt":  "text/plain",
	".json": "application/json",
	".yaml": "text/yaml",
	".yml":  "text/yaml",
	".sh":   "text/x-shellscript",
	".py":   "text/x-python",
	".js":   "text/javascript",
	".ts":   "text/typescript",
	".html": "text/html",
	".css":  "text/css",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".svg":  "image/svg+xml",
}

// DetectMimeType returns the MIME type for a filename based on extension.
func DetectMimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return ""
}

// IsBinaryMimeType checks if a MIME type represents binary content.
func IsBinaryMimeType(mimeType string) bool {
	if mimeType == "" {
		return false
	}
	if strings.HasPrefix(mimeType, "text/") {
		return false
	}
	if mimeType == "application/json" {
		return false
	}
	if mimeType == "image/svg+xml" {
		return false
	}
	return true
}

// SkillSourceStatFromFileInfo creates a SkillSourceStat from os.FileInfo.
func SkillSourceStatFromFileInfo(info os.FileInfo) *SkillSourceStat {
	entryType := "file"
	if info.IsDir() {
		entryType = "directory"
	}
	return &SkillSourceStat{
		Name:       info.Name(),
		Type:       entryType,
		Size:       info.Size(),
		CreatedAt:  info.ModTime(),
		ModifiedAt: info.ModTime(),
	}
}

// NowTime returns the current time for use in versioned sources.
func NowTime() time.Time {
	return time.Now()
}
