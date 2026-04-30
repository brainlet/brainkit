// Package resourcehost owns JS-runtime deployment resource tracking.
package resourcehost

import (
	"sync"
	"time"
)

// Entry is a tracked deployment resource.
type Entry struct {
	Type      string
	ID        string
	Name      string
	Source    string
	CreatedAt time.Time
}

// Key returns the stable type/id registry key.
func (e Entry) Key() string {
	return e.Type + ":" + e.ID
}

// Registry is a concurrent-safe resource registry. It is a pure data store:
// callers own resource-specific cleanup after removal.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]Entry
}

// NewRegistry creates an empty resource registry.
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]Entry),
	}
}

// Register adds or replaces a resource entry.
func (r *Registry) Register(entry Entry) {
	key := entry.Key()
	r.mu.Lock()
	r.entries[key] = entry
	r.mu.Unlock()
}

// Unregister removes a single entry and returns it.
func (r *Registry) Unregister(typ, id string) (Entry, bool) {
	key := typ + ":" + id
	r.mu.Lock()
	entry, ok := r.entries[key]
	if ok {
		delete(r.entries, key)
	}
	r.mu.Unlock()
	return entry, ok
}

// Get returns a single entry.
func (r *Registry) Get(typ, id string) (Entry, bool) {
	key := typ + ":" + id
	r.mu.RLock()
	entry, ok := r.entries[key]
	r.mu.RUnlock()
	return entry, ok
}

// List returns all entries, optionally filtered by type.
func (r *Registry) List(typ string) []Entry {
	r.mu.RLock()
	result := make([]Entry, 0, len(r.entries))
	for _, entry := range r.entries {
		if typ == "" || entry.Type == typ {
			result = append(result, entry)
		}
	}
	r.mu.RUnlock()
	return result
}

// ListBySource returns all entries created by a specific source file.
func (r *Registry) ListBySource(source string) []Entry {
	r.mu.RLock()
	var result []Entry
	for _, entry := range r.entries {
		if entry.Source == source {
			result = append(result, entry)
		}
	}
	r.mu.RUnlock()
	return result
}

// RemoveBySource atomically removes all entries for a source and returns them.
func (r *Registry) RemoveBySource(source string) []Entry {
	r.mu.Lock()
	var removed []Entry
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
func (r *Registry) Len() int {
	r.mu.RLock()
	n := len(r.entries)
	r.mu.RUnlock()
	return n
}
