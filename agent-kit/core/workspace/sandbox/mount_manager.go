// Ported from: packages/core/src/workspace/sandbox/mount-manager.ts
package sandbox

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/brainlet/brainkit/agent-kit/core/workspace/sandbox/mounts"
)

// =============================================================================
// Mount Manager Types
// =============================================================================

// MountFn is the mount function signature used by the sandbox implementation.
type MountFn func(filesystem WorkspaceFilesystemRef, mountPath string) (*MountResult, error)

// OnMountResult represents the result from an onMount hook.
//   - nil: use default mount
//   - Skip == true: skip mount entirely
//   - Success field: hook handled it
type OnMountResult struct {
	// Skip indicates the mount should be skipped entirely.
	Skip bool
	// Success indicates whether the hook handled the mount successfully.
	Success bool
	// Error is an optional error message.
	Error string
}

// OnMountArgs holds arguments passed to the onMount hook.
type OnMountArgs struct {
	// Filesystem is the filesystem being mounted.
	Filesystem WorkspaceFilesystemRef
	// MountPath is the mount path in the sandbox.
	MountPath string
	// Config is the mount configuration from filesystem.GetMountConfig().
	Config *FilesystemMountConfig
	// Sandbox is the sandbox instance.
	Sandbox WorkspaceSandbox
}

// OnMountHook is called for each filesystem before mounting into sandbox.
type OnMountHook func(args OnMountArgs) *OnMountResult

// MountManagerConfig configures a MountManager.
type MountManagerConfig struct {
	// Mount is the mount implementation from the sandbox.
	Mount MountFn
	// Logger is the logger instance.
	Logger MountManagerLogger
}

// MountManagerLogger is the logger interface used by MountManager.
type MountManagerLogger interface {
	Debug(message string, args ...interface{})
	Info(message string, args ...interface{})
	Warn(message string, args ...interface{})
	Error(message string, args ...interface{})
}

// =============================================================================
// Mount Manager
// =============================================================================

// MountManager manages filesystem mounts for a sandbox.
//
// Provides methods for tracking mount state, updating entries,
// and processing pending mounts.
type MountManager struct {
	entries   map[string]*MountEntry
	mountFn   MountFn
	onMount   OnMountHook
	sandbox   WorkspaceSandbox
	logger    MountManagerLogger
	mu        sync.RWMutex
}

// NewMountManager creates a new MountManager.
func NewMountManager(config MountManagerConfig) *MountManager {
	return &MountManager{
		entries: make(map[string]*MountEntry),
		mountFn: config.Mount,
		logger:  config.Logger,
	}
}

// SetContext sets the sandbox reference for onMount hook args.
// Called by Workspace during construction.
func (m *MountManager) SetContext(sandbox WorkspaceSandbox) {
	m.sandbox = sandbox
}

// SetOnMount sets the onMount hook for custom mount handling.
func (m *MountManager) SetOnMount(hook OnMountHook) {
	m.onMount = hook
}

// SetLogger updates the logger instance.
func (m *MountManager) SetLogger(logger MountManagerLogger) {
	m.logger = logger
}

// =============================================================================
// Entry Access
// =============================================================================

// Entries returns all mount entries.
func (m *MountManager) Entries() map[string]*MountEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]*MountEntry, len(m.entries))
	for k, v := range m.entries {
		result[k] = v
	}
	return result
}

// Get returns a mount entry by path.
func (m *MountManager) Get(path string) *MountEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.entries[path]
}

// Has checks if a mount exists at the given path.
func (m *MountManager) Has(path string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.entries[path]
	return ok
}

// =============================================================================
// Entry Modification
// =============================================================================

// Add adds pending mounts from workspace config.
// These will be processed when ProcessPending() is called.
func (m *MountManager) Add(mountMap map[string]WorkspaceFilesystemRef) {
	m.mu.Lock()
	defer m.mu.Unlock()

	paths := make([]string, 0, len(mountMap))
	for p := range mountMap {
		paths = append(paths, p)
	}
	m.logger.Debug(fmt.Sprintf("Adding %d pending mount(s)", len(paths)))

	for path, filesystem := range mountMap {
		m.entries[path] = &MountEntry{
			Filesystem: filesystem,
			State:      MountStatePending,
		}
	}
}

// Set updates a mount entry's state.
// Creates the entry if it doesn't exist and filesystem is provided.
func (m *MountManager) Set(path string, updates MountEntryUpdate) {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.entries[path]
	if ok {
		existing.State = updates.State
		if updates.Config != nil {
			existing.Config = updates.Config
			existing.ConfigHash = m.hashConfig(updates.Config)
		}
		if updates.Error != "" || updates.ClearError {
			existing.Error = updates.Error
		}
	} else if updates.Filesystem != nil {
		m.entries[path] = &MountEntry{
			Filesystem: updates.Filesystem,
			State:      updates.State,
			Config:     updates.Config,
			ConfigHash: func() string {
				if updates.Config != nil {
					return m.hashConfig(updates.Config)
				}
				return ""
			}(),
			Error: updates.Error,
		}
	} else {
		m.logger.Debug(fmt.Sprintf("set() called for unknown path %q without filesystem — no entry created", path))
	}
}

// MountEntryUpdate holds update fields for a mount entry.
type MountEntryUpdate struct {
	Filesystem WorkspaceFilesystemRef
	State      MountState
	Config     *FilesystemMountConfig
	Error      string
	ClearError bool
}

// Delete deletes a mount entry.
func (m *MountManager) Delete(path string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.entries[path]
	if ok {
		delete(m.entries, path)
	}
	return ok
}

// Clear clears all mount entries.
func (m *MountManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make(map[string]*MountEntry)
}

// =============================================================================
// Mount Processing
// =============================================================================

// ProcessPending processes all pending mounts.
// Call this after sandbox is ready (in start()).
func (m *MountManager) ProcessPending() error {
	m.mu.RLock()
	pendingCount := 0
	for _, entry := range m.entries {
		if entry.State == MountStatePending {
			pendingCount++
		}
	}
	m.mu.RUnlock()

	if pendingCount == 0 {
		return nil
	}

	m.logger.Debug(fmt.Sprintf("Processing %d pending mount(s)", pendingCount))

	m.mu.RLock()
	paths := make([]string, 0, len(m.entries))
	for path := range m.entries {
		paths = append(paths, path)
	}
	m.mu.RUnlock()

	for _, path := range paths {
		m.mu.RLock()
		entry, ok := m.entries[path]
		if !ok || entry.State != MountStatePending {
			m.mu.RUnlock()
			continue
		}
		filesystem := entry.Filesystem
		m.mu.RUnlock()

		fsProvider := filesystem.Provider()

		// Get config if available
		config := filesystem.GetMountConfig()

		// Call onMount hook if configured
		if m.onMount != nil {
			hookResult := m.onMount(OnMountArgs{
				Filesystem: filesystem,
				MountPath:  path,
				Config:     config,
				Sandbox:    m.sandbox,
			})

			if hookResult != nil {
				if hookResult.Skip {
					m.mu.Lock()
					if e, ok := m.entries[path]; ok {
						e.State = MountStateUnsupported
						e.Error = "Skipped by onMount hook"
					}
					m.mu.Unlock()
					m.logger.Debug(fmt.Sprintf("Mount skipped by onMount hook: path=%s provider=%s", path, fsProvider))
					continue
				}

				if hookResult.Success {
					m.mu.Lock()
					if e, ok := m.entries[path]; ok {
						e.State = MountStateMounted
						e.Config = config
						if config != nil {
							e.ConfigHash = m.hashConfig(config)
						}
					}
					m.mu.Unlock()
					m.logger.Info(fmt.Sprintf("Mount handled by onMount hook: path=%s provider=%s", path, fsProvider))
				} else {
					errMsg := hookResult.Error
					if errMsg == "" {
						errMsg = "Mount hook failed"
					}
					m.mu.Lock()
					if e, ok := m.entries[path]; ok {
						e.State = MountStateError
						e.Error = errMsg
					}
					m.mu.Unlock()
					m.logger.Error(fmt.Sprintf("Mount hook failed: path=%s provider=%s error=%s", path, fsProvider, errMsg))
				}
				continue
			}
			// nil result = continue with default mount
		}

		// Check if filesystem supports mounting
		if config == nil {
			m.mu.Lock()
			if e, ok := m.entries[path]; ok {
				e.State = MountStateUnsupported
				e.Error = "Filesystem does not support mounting"
			}
			m.mu.Unlock()
			m.logger.Debug(fmt.Sprintf("Filesystem does not support mounting: path=%s provider=%s", path, fsProvider))
			continue
		}

		// Store config and mark as mounting
		m.mu.Lock()
		if e, ok := m.entries[path]; ok {
			e.Config = config
			e.ConfigHash = m.hashConfig(config)
			e.State = MountStateMounting
		}
		m.mu.Unlock()

		m.logger.Debug(fmt.Sprintf("Mounting filesystem: path=%s provider=%s type=%s", path, fsProvider, config.Type))

		// Call the sandbox's mount implementation
		result, err := m.mountFn(filesystem, path)
		if err != nil {
			if _, ok := err.(*mounts.MountToolNotFoundError); ok {
				m.mu.Lock()
				if e, eOK := m.entries[path]; eOK {
					e.State = MountStateUnavailable
					e.Error = err.Error()
				}
				m.mu.Unlock()
				m.logger.Warn(fmt.Sprintf("FUSE mount unavailable: path=%s provider=%s error=%s", path, fsProvider, err.Error()))
			} else {
				m.mu.Lock()
				if e, eOK := m.entries[path]; eOK {
					e.State = MountStateError
					e.Error = err.Error()
				}
				m.mu.Unlock()
				m.logger.Error(fmt.Sprintf("Mount threw error: path=%s provider=%s error=%s", path, fsProvider, err.Error()))
			}
			continue
		}

		if result.Success {
			m.mu.Lock()
			if e, ok := m.entries[path]; ok {
				e.State = MountStateMounted
			}
			m.mu.Unlock()
			m.logger.Info(fmt.Sprintf("Mount successful: path=%s provider=%s", path, fsProvider))
		} else if result.Unavailable {
			m.mu.Lock()
			if e, ok := m.entries[path]; ok {
				e.State = MountStateUnavailable
				e.Error = result.Error
				if e.Error == "" {
					e.Error = "FUSE tool not installed"
				}
			}
			m.mu.Unlock()
			m.logger.Warn(fmt.Sprintf("FUSE mount unavailable: path=%s provider=%s", path, fsProvider))
		} else {
			errMsg := result.Error
			if errMsg == "" {
				errMsg = "Mount failed"
			}
			m.mu.Lock()
			if e, ok := m.entries[path]; ok {
				e.State = MountStateError
				e.Error = errMsg
			}
			m.mu.Unlock()
			m.logger.Error(fmt.Sprintf("Mount failed: path=%s provider=%s error=%s", path, fsProvider, errMsg))
		}
	}

	return nil
}

// =============================================================================
// Marker File Helpers
// =============================================================================

// MarkerFilename generates a marker filename for a mount path.
// Used by sandboxes to store mount metadata for reconnection detection.
func (m *MountManager) MarkerFilename(mountPath string) string {
	hash := int32(0)
	for _, ch := range mountPath {
		hash = (hash << 5) - hash + int32(ch)
	}
	absHash := int64(math.Abs(float64(hash)))
	return fmt.Sprintf("mount-%s", base36(absHash))
}

// GetMarkerContent generates marker file content for a mount path.
// Format: "path|configHash" - used for detecting config changes on reconnect.
func (m *MountManager) GetMarkerContent(mountPath string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.entries[mountPath]
	if !ok || entry.ConfigHash == "" {
		return ""
	}
	return fmt.Sprintf("%s|%s", mountPath, entry.ConfigHash)
}

// ParseMarkerContent parses marker file content.
// Returns the parsed path and configHash, or empty strings if invalid format.
func (m *MountManager) ParseMarkerContent(content string) (path, configHash string, ok bool) {
	separatorIndex := strings.LastIndex(content, "|")
	if separatorIndex <= 0 {
		return "", "", false
	}
	path = content[:separatorIndex]
	configHash = content[separatorIndex+1:]
	if path == "" || configHash == "" {
		return "", "", false
	}
	return path, configHash, true
}

// IsConfigMatching checks if a config hash matches the expected hash for a mount path.
func (m *MountManager) IsConfigMatching(mountPath, storedHash string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.entries[mountPath]
	if !ok {
		return false
	}
	return entry.ConfigHash == storedHash
}

// ComputeConfigHash computes a hash for a mount config.
func (m *MountManager) ComputeConfigHash(config *FilesystemMountConfig) string {
	return m.hashConfig(config)
}

// =============================================================================
// Internal
// =============================================================================

// hashConfig hashes a mount config for comparison.
func (m *MountManager) hashConfig(config *FilesystemMountConfig) string {
	normalized := sortKeysJSON(config)
	h := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", h[:8]) // 16 hex chars
}

// sortKeysJSON marshals a value with sorted keys for deterministic hashing.
func sortKeysJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	// json.Marshal in Go already produces sorted keys for structs,
	// but for maps we need to sort manually. Since FilesystemMountConfig
	// is a struct, json.Marshal produces consistent output.
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return string(data)
	}
	return sortMapJSON(raw)
}

// sortMapJSON recursively sorts map keys for deterministic JSON output.
func sortMapJSON(m map[string]interface{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		v := m[k]
		kJSON, _ := json.Marshal(k)
		var vJSON []byte
		switch val := v.(type) {
		case map[string]interface{}:
			vJSON = []byte(sortMapJSON(val))
		default:
			vJSON, _ = json.Marshal(val)
		}
		parts = append(parts, string(kJSON)+":"+string(vJSON))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// base36 converts an int64 to base36 string.
func base36(n int64) string {
	if n == 0 {
		return "0"
	}
	const chars = "0123456789abcdefghijklmnopqrstuvwxyz"
	var result []byte
	for n > 0 {
		result = append([]byte{chars[n%36]}, result...)
		n /= 36
	}
	return string(result)
}
