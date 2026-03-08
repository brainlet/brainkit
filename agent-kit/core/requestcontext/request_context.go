// Ported from: packages/core/src/request-context/index.ts
package requestcontext

import (
	"encoding/json"
	"sort"
	"sync"
)

// MastraResourceIDKey is a reserved key for setting resourceId from middleware.
// When set in RequestContext, this takes precedence over client-provided values
// for security (prevents attackers from hijacking another user's memory).
const MastraResourceIDKey = "mastra__resourceId"

// MastraThreadIDKey is a reserved key for setting threadId from middleware.
// When set in RequestContext, this takes precedence over client-provided values
// for security (prevents attackers from hijacking another user's memory).
const MastraThreadIDKey = "mastra__threadId"

// RequestContext is a thread-safe key-value container for request-scoped data.
// It mirrors the TypeScript RequestContext class from @mastra/core.
type RequestContext struct {
	mu       sync.RWMutex
	registry map[string]any
}

// NewRequestContext creates an empty RequestContext.
func NewRequestContext() *RequestContext {
	return &RequestContext{
		registry: make(map[string]any),
	}
}

// NewRequestContextFromMap creates a RequestContext pre-populated from a map.
// This is the Go equivalent of constructing from a plain object (e.g. deserialized from JSON).
func NewRequestContextFromMap(m map[string]any) *RequestContext {
	reg := make(map[string]any, len(m))
	for k, v := range m {
		reg[k] = v
	}
	return &RequestContext{
		registry: reg,
	}
}

// NewRequestContextFromEntries creates a RequestContext from key-value pairs.
// This is the Go equivalent of constructing from an array of tuples.
func NewRequestContextFromEntries(entries []Entry) *RequestContext {
	reg := make(map[string]any, len(entries))
	for _, e := range entries {
		reg[e.Key] = e.Value
	}
	return &RequestContext{
		registry: reg,
	}
}

// Entry represents a key-value pair, equivalent to a [string, unknown] tuple in TypeScript.
type Entry struct {
	Key   string
	Value any
}

// Set stores a value under the given key.
func (rc *RequestContext) Set(key string, value any) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.registry[key] = value
}

// Get retrieves the value stored under the given key.
// Returns nil if the key does not exist.
func (rc *RequestContext) Get(key string) any {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.registry[key]
}

// Has returns true if the key exists in the container.
func (rc *RequestContext) Has(key string) bool {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	_, ok := rc.registry[key]
	return ok
}

// Delete removes a key from the container. Returns true if the key existed.
func (rc *RequestContext) Delete(key string) bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	_, ok := rc.registry[key]
	if ok {
		delete(rc.registry, key)
	}
	return ok
}

// Clear removes all entries from the container.
func (rc *RequestContext) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.registry = make(map[string]any)
}

// Keys returns all keys in the container. The order is sorted for determinism.
func (rc *RequestContext) Keys() []string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	keys := make([]string, 0, len(rc.registry))
	for k := range rc.registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Values returns all values in the container. The order matches Keys().
func (rc *RequestContext) Values() []any {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	keys := make([]string, 0, len(rc.registry))
	for k := range rc.registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	vals := make([]any, len(keys))
	for i, k := range keys {
		vals[i] = rc.registry[k]
	}
	return vals
}

// Entries returns a copy of all entries as a map.
func (rc *RequestContext) Entries() map[string]any {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	m := make(map[string]any, len(rc.registry))
	for k, v := range rc.registry {
		m[k] = v
	}
	return m
}

// Size returns the number of entries in the container.
func (rc *RequestContext) Size() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return len(rc.registry)
}

// ForEach executes the given function for each entry in the container.
// The iteration order is sorted by key for determinism.
func (rc *RequestContext) ForEach(fn func(key string, value any)) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	keys := make([]string, 0, len(rc.registry))
	for k := range rc.registry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fn(k, rc.registry[k])
	}
}

// MarshalJSON implements json.Marshaler. It serializes only JSON-serializable
// values, skipping entries that cannot be marshaled (matching toJSON() behavior
// in the TypeScript original).
func (rc *RequestContext) MarshalJSON() ([]byte, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	result := make(map[string]any, len(rc.registry))
	for k, v := range rc.registry {
		if isSerializable(v) {
			result[k] = v
		}
	}
	return json.Marshal(result)
}

// All returns a copy of all entries as a map, equivalent to the `all` getter
// in the TypeScript original. Useful for destructuring-style access.
func (rc *RequestContext) All() map[string]any {
	return rc.Entries()
}

// isSerializable checks if a value can be safely serialized to JSON.
// This mirrors the private isSerializable method in the TypeScript original.
func isSerializable(value any) bool {
	if value == nil {
		return true
	}
	_, err := json.Marshal(value)
	return err == nil
}
