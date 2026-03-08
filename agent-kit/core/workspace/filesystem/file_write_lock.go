// Ported from: packages/core/src/workspace/filesystem/file-write-lock.ts
package filesystem

import (
	"path"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// File Write Lock Interface
// =============================================================================

// FileWriteLock serializes write operations to individual file paths.
// Prevents concurrent writes to the same file from corrupting content.
type FileWriteLock interface {
	// WithLock executes fn while holding the lock for the given file path.
	// Concurrent calls to the same path are serialized.
	// Default timeout is 30 seconds.
	WithLock(filePath string, fn func() (interface{}, error)) (interface{}, error)
}

// =============================================================================
// In-Memory Implementation
// =============================================================================

// DefaultWriteLockTimeout is the default timeout for write lock acquisition.
const DefaultWriteLockTimeout = 30 * time.Second

// pathMutex holds a per-path mutex with a reference counter.
type pathMutex struct {
	mu       sync.Mutex
	refCount int
}

// InMemoryFileWriteLock is a simple in-memory implementation of FileWriteLock.
// Uses a per-path mutex to serialize writes to each file independently.
type InMemoryFileWriteLock struct {
	mu    sync.Mutex
	locks map[string]*pathMutex
}

// NewInMemoryFileWriteLock creates a new InMemoryFileWriteLock.
func NewInMemoryFileWriteLock() *InMemoryFileWriteLock {
	return &InMemoryFileWriteLock{
		locks: make(map[string]*pathMutex),
	}
}

// normalizeWritePath normalizes a file path for consistent key lookup.
func normalizeWritePath(filePath string) string {
	p := strings.TrimPrefix(filePath, "./")
	return path.Clean(p)
}

// WithLock executes fn while holding the lock for the given file path.
func (l *InMemoryFileWriteLock) WithLock(filePath string, fn func() (interface{}, error)) (interface{}, error) {
	key := normalizeWritePath(filePath)

	// Get or create the per-path mutex
	l.mu.Lock()
	pm, exists := l.locks[key]
	if !exists {
		pm = &pathMutex{}
		l.locks[key] = pm
	}
	pm.refCount++
	l.mu.Unlock()

	// Acquire the per-path lock
	pm.mu.Lock()
	defer func() {
		pm.mu.Unlock()

		// Cleanup if no more waiters
		l.mu.Lock()
		pm.refCount--
		if pm.refCount == 0 {
			delete(l.locks, key)
		}
		l.mu.Unlock()
	}()

	return fn()
}
