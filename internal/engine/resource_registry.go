package engine

import (
	"sync"
	"time"
)

// ResourceEntry is a tracked resource in the registry.
type ResourceEntry struct {
	Type      string
	ID        string
	Name      string
	Source    string
	CreatedAt time.Time
}

func (e ResourceEntry) Key() string {
	return e.Type + ":" + e.ID
}

// ResourceRegistry is a concurrent-safe registry for tracking deployed resources.
// Pure data store — no cleanup logic, no JS dependencies.
// The caller (DeploymentManager) dispatches cleanup by resource type after removal.
type ResourceRegistry struct {
	mu      sync.RWMutex
	entries map[string]ResourceEntry // key = "type:id"
}

func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		entries: make(map[string]ResourceEntry),
	}
}

// Register adds or replaces a resource entry.
func (r *ResourceRegistry) Register(entry ResourceEntry) {
	key := entry.Key()
	r.mu.Lock()
	r.entries[key] = entry
	r.mu.Unlock()
}

// Unregister removes a single entry and returns it. Returns zero value + false if not found.
func (r *ResourceRegistry) Unregister(typ, id string) (ResourceEntry, bool) {
	key := typ + ":" + id
	r.mu.Lock()
	entry, ok := r.entries[key]
	if ok {
		delete(r.entries, key)
	}
	r.mu.Unlock()
	return entry, ok
}

// Get returns a single entry. Returns zero value + false if not found.
func (r *ResourceRegistry) Get(typ, id string) (ResourceEntry, bool) {
	key := typ + ":" + id
	r.mu.RLock()
	entry, ok := r.entries[key]
	r.mu.RUnlock()
	return entry, ok
}

// List returns all entries, optionally filtered by type (empty = all).
func (r *ResourceRegistry) List(typ string) []ResourceEntry {
	r.mu.RLock()
	result := make([]ResourceEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		if typ == "" || entry.Type == typ {
			result = append(result, entry)
		}
	}
	r.mu.RUnlock()
	return result
}

// ListBySource returns all entries created by a specific source file.
func (r *ResourceRegistry) ListBySource(source string) []ResourceEntry {
	r.mu.RLock()
	var result []ResourceEntry
	for _, entry := range r.entries {
		if entry.Source == source {
			result = append(result, entry)
		}
	}
	r.mu.RUnlock()
	return result
}

// RemoveBySource atomically removes all entries for a source and returns them.
// The caller uses the returned entries to dispatch type-specific cleanup.
func (r *ResourceRegistry) RemoveBySource(source string) []ResourceEntry {
	r.mu.Lock()
	var removed []ResourceEntry
	for key, entry := range r.entries {
		if entry.Source == source {
			removed = append(removed, entry)
			delete(r.entries, key)
		}
	}
	r.mu.Unlock()
	return removed
}

// Len returns the number of entries.
func (r *ResourceRegistry) Len() int {
	r.mu.RLock()
	n := len(r.entries)
	r.mu.RUnlock()
	return n
}
