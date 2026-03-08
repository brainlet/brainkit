// Ported from: packages/core/src/storage/filesystem-db.ts
package fsutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ISO date regex for the dateReviver
var isoDateRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)

// ---------------------------------------------------------------------------
// FilesystemDB
// ---------------------------------------------------------------------------

// FilesystemDB is a thin I/O layer for filesystem-based storage.
// It manages reading/writing JSON files in a directory, similar to how
// InMemoryDB holds Maps for in-memory storage.
//
// Each editor domain gets its own JSON file (e.g., "agents.json", "prompt-blocks.json").
// Skills use a real file tree under "skills/" instead of JSON.
type FilesystemDB struct {
	// Dir is the absolute path to the storage directory.
	Dir string

	mu          sync.RWMutex
	cache       map[string]map[string]any
	initialized bool
}

// NewFilesystemDB creates a new FilesystemDB for the given directory.
func NewFilesystemDB(dir string) *FilesystemDB {
	return &FilesystemDB{
		Dir:   dir,
		cache: make(map[string]map[string]any),
	}
}

// Init initializes the storage directory. Called once; subsequent calls are no-ops.
func (db *FilesystemDB) Init() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.initialized {
		return nil
	}
	db.ensureDirLocked()
	db.initialized = true
	return nil
}

// ensureDirLocked ensures the storage directory and skills subdirectory exist.
// Caller must hold db.mu.
func (db *FilesystemDB) ensureDirLocked() {
	if err := os.MkdirAll(db.Dir, 0o755); err != nil {
		// Best effort
		return
	}
	skillsDir := filepath.Join(db.Dir, "skills")
	os.MkdirAll(skillsDir, 0o755)
}

// ==========================================================================
// Domain-level JSON operations
// ==========================================================================

// ReadDomain reads a domain JSON file and returns its entity map.
// Uses in-memory cache; reads from disk on first access.
func (db *FilesystemDB) ReadDomain(filename string) map[string]any {
	db.mu.RLock()
	if cached, ok := db.cache[filename]; ok {
		db.mu.RUnlock()
		return cached
	}
	db.mu.RUnlock()

	db.mu.Lock()
	defer db.mu.Unlock()

	// Double-check after acquiring write lock
	if cached, ok := db.cache[filename]; ok {
		return cached
	}

	filePath := filepath.Join(db.Dir, filename)
	data := make(map[string]any)

	raw, err := os.ReadFile(filePath)
	if err == nil {
		parsed, parseErr := jsonUnmarshalWithDateReviver(raw)
		if parseErr == nil {
			data = parsed
		}
		// If corrupted, start fresh with empty map
	}

	db.cache[filename] = data
	return data
}

// WriteDomain writes a domain's full entity map to its JSON file.
// Uses atomic write (write to .tmp, then rename) to prevent corruption.
func (db *FilesystemDB) WriteDomain(filename string, data map[string]any) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.cache[filename] = data

	filePath := filepath.Join(db.Dir, filename)
	tmpPath := filePath + ".tmp"

	// Ensure parent directory exists
	parentDir := filepath.Dir(filePath)
	os.MkdirAll(parentDir, 0o755)

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return // Best effort
	}

	if err := os.WriteFile(tmpPath, jsonBytes, 0o644); err != nil {
		return
	}
	os.Rename(tmpPath, filePath)
}

// ClearDomain clears all data from a domain JSON file.
func (db *FilesystemDB) ClearDomain(filename string) {
	db.WriteDomain(filename, make(map[string]any))
}

// InvalidateCache invalidates the in-memory cache for a domain,
// forcing a re-read from disk on next access.
// If filename is empty, all caches are invalidated.
func (db *FilesystemDB) InvalidateCache(filename string) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if filename != "" {
		delete(db.cache, filename)
	} else {
		db.cache = make(map[string]map[string]any)
	}
}

// ==========================================================================
// Entity-level convenience methods
// ==========================================================================

// Get retrieves a single entity by ID from a domain JSON file.
func (db *FilesystemDB) Get(filename string, id string) any {
	data := db.ReadDomain(filename)
	v, ok := data[id]
	if !ok {
		return nil
	}
	return v
}

// GetAll returns all entities from a domain JSON file as a slice.
func (db *FilesystemDB) GetAll(filename string) []any {
	data := db.ReadDomain(filename)
	result := make([]any, 0, len(data))
	for _, v := range data {
		result = append(result, v)
	}
	return result
}

// Set creates or updates an entity in a domain JSON file.
func (db *FilesystemDB) Set(filename string, id string, entity any) {
	data := db.ReadDomain(filename)
	dataCopy := make(map[string]any, len(data)+1)
	for k, v := range data {
		dataCopy[k] = v
	}
	dataCopy[id] = entity
	db.WriteDomain(filename, dataCopy)
}

// Remove removes an entity by ID from a domain JSON file. No-op if not found.
func (db *FilesystemDB) Remove(filename string, id string) {
	data := db.ReadDomain(filename)
	if _, ok := data[id]; !ok {
		return
	}
	dataCopy := make(map[string]any, len(data))
	for k, v := range data {
		if k != id {
			dataCopy[k] = v
		}
	}
	db.WriteDomain(filename, dataCopy)
}

// ==========================================================================
// Skills directory operations (real file tree, not JSON)
// ==========================================================================

// SkillDir returns the path to a skill's directory.
func (db *FilesystemDB) SkillDir(skillName string) (string, error) {
	skillsBase := filepath.Join(db.Dir, "skills")
	dir := filepath.Join(skillsBase, skillName)
	resolved, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	absBase, _ := filepath.Abs(skillsBase)
	if !strings.HasPrefix(resolved, absBase+string(filepath.Separator)) && resolved != absBase {
		return "", fmt.Errorf("path traversal detected: skill name %q escapes skills directory", skillName)
	}
	return resolved, nil
}

// safeSkillPath resolves a file path within a skill directory, returning error if it escapes.
func (db *FilesystemDB) safeSkillPath(skillName, relativePath string) (string, error) {
	base, err := db.SkillDir(skillName)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.Abs(filepath.Join(base, relativePath))
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(resolved, base+string(filepath.Separator)) && resolved != base {
		return "", fmt.Errorf("path traversal detected: %q escapes skill directory", relativePath)
	}
	return resolved, nil
}

// ListSkillFiles lists all files in a skill's directory, returning relative paths.
func (db *FilesystemDB) ListSkillFiles(skillName string) ([]string, error) {
	dir, err := db.SkillDir(skillName)
	if err != nil {
		return nil, err
	}
	if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
		return []string{}, nil
	}

	var results []string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, relErr := filepath.Rel(dir, path)
			if relErr == nil {
				results = append(results, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// ReadSkillFile reads a file from a skill's directory.
func (db *FilesystemDB) ReadSkillFile(skillName, relativePath string) ([]byte, error) {
	filePath, err := db.safeSkillPath(skillName, relativePath)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// WriteSkillFile writes a file to a skill's directory.
func (db *FilesystemDB) WriteSkillFile(skillName, relativePath string, content []byte) error {
	filePath, err := db.safeSkillPath(skillName, relativePath)
	if err != nil {
		return err
	}
	parentDir := filepath.Dir(filePath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filePath, content, 0o644)
}

// DeleteSkillDir deletes a skill's entire directory.
func (db *FilesystemDB) DeleteSkillDir(skillName string) error {
	dir, err := db.SkillDir(skillName)
	if err != nil {
		return err
	}
	if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
		return nil
	}
	return os.RemoveAll(dir)
}

// ---------------------------------------------------------------------------
// JSON date reviver
// ---------------------------------------------------------------------------

// jsonUnmarshalWithDateReviver unmarshals JSON with ISO date string conversion
// to time.Time values embedded in map[string]any structures.
func jsonUnmarshalWithDateReviver(data []byte) (map[string]any, error) {
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	reviveDates(raw)
	return raw, nil
}

// reviveDates recursively converts ISO date strings to time.Time values.
func reviveDates(m map[string]any) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			if isoDateRegex.MatchString(val) {
				if t, err := time.Parse(time.RFC3339Nano, val); err == nil {
					m[k] = t
				}
			}
		case map[string]any:
			reviveDates(val)
		case []any:
			for _, item := range val {
				if nested, ok := item.(map[string]any); ok {
					reviveDates(nested)
				}
			}
		}
	}
}
