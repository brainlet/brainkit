// Ported from: packages/core/src/workspace/filesystem/file-read-tracker.ts
package filesystem

import (
	"path"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// File Read Tracker Interface
// =============================================================================

// ReadCheckResult holds the result of a NeedsReRead check.
type ReadCheckResult struct {
	// NeedsReRead is true if the file needs to be re-read before writing.
	NeedsReRead bool
	// Reason explains why a re-read is needed (empty if NeedsReRead is false).
	Reason string
}

// FileReadTracker tracks which files have been read, enabling
// requireReadBeforeWrite enforcement for write tools.
type FileReadTracker interface {
	// RecordRead records that a file was read at the given modification time.
	RecordRead(filePath string, modifiedAt time.Time)

	// NeedsReRead checks if a file needs to be re-read before writing.
	// Returns NeedsReRead=true with a reason if the file hasn't been read
	// or has been modified since the last read.
	NeedsReRead(filePath string, currentModifiedAt time.Time) ReadCheckResult

	// ClearReadRecord removes the read record for a file.
	// Called after a write to force a re-read before the next write.
	ClearReadRecord(filePath string)
}

// =============================================================================
// In-Memory Implementation
// =============================================================================

// readRecord holds the state of a single file read.
type readRecord struct {
	modifiedAt time.Time
}

// InMemoryFileReadTracker is a simple in-memory implementation of FileReadTracker.
// Thread-safe via sync.RWMutex.
type InMemoryFileReadTracker struct {
	mu      sync.RWMutex
	records map[string]*readRecord
}

// NewInMemoryFileReadTracker creates a new InMemoryFileReadTracker.
func NewInMemoryFileReadTracker() *InMemoryFileReadTracker {
	return &InMemoryFileReadTracker{
		records: make(map[string]*readRecord),
	}
}

// normalizePath normalizes a file path for consistent key lookup.
// Removes leading "./" and cleans the path.
func normalizePath(filePath string) string {
	p := strings.TrimPrefix(filePath, "./")
	return path.Clean(p)
}

// RecordRead records that a file was read at the given modification time.
func (t *InMemoryFileReadTracker) RecordRead(filePath string, modifiedAt time.Time) {
	key := normalizePath(filePath)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records[key] = &readRecord{modifiedAt: modifiedAt}
}

// NeedsReRead checks if a file needs to be re-read before writing.
func (t *InMemoryFileReadTracker) NeedsReRead(filePath string, currentModifiedAt time.Time) ReadCheckResult {
	key := normalizePath(filePath)
	t.mu.RLock()
	defer t.mu.RUnlock()

	record, exists := t.records[key]
	if !exists {
		return ReadCheckResult{
			NeedsReRead: true,
			Reason:      "You must read a file before writing to it. Read " + filePath + " first, then retry your edit.",
		}
	}

	if !record.modifiedAt.Equal(currentModifiedAt) {
		return ReadCheckResult{
			NeedsReRead: true,
			Reason:      filePath + " has been modified since you last read it. Read it again to get the latest content before editing.",
		}
	}

	return ReadCheckResult{NeedsReRead: false}
}

// ClearReadRecord removes the read record for a file.
func (t *InMemoryFileReadTracker) ClearReadRecord(filePath string) {
	key := normalizePath(filePath)
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.records, key)
}
