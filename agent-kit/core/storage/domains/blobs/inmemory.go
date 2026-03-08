// Ported from: packages/core/src/storage/domains/blobs/inmemory.ts
package blobs

import (
	"context"
	"sync"
)

// Compile-time interface check.
var _ BlobStore = (*InMemoryBlobStore)(nil)

// InMemoryBlobStore is an in-memory implementation of BlobStore for testing.
type InMemoryBlobStore struct {
	mu    sync.RWMutex
	blobs map[string]StorageBlobEntry
}

// NewInMemoryBlobStore creates a new InMemoryBlobStore.
func NewInMemoryBlobStore() *InMemoryBlobStore {
	return &InMemoryBlobStore{
		blobs: make(map[string]StorageBlobEntry),
	}
}

// Init is a no-op for the in-memory store.
func (s *InMemoryBlobStore) Init(_ context.Context) error {
	return nil
}

// Put stores a blob. If the hash already exists, this is a no-op.
func (s *InMemoryBlobStore) Put(_ context.Context, entry StorageBlobEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.blobs[entry.Hash]; !exists {
		s.blobs[entry.Hash] = entry
	}
	return nil
}

// Get retrieves a blob by its hash. Returns nil if not found.
func (s *InMemoryBlobStore) Get(_ context.Context, hash string) (*StorageBlobEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.blobs[hash]
	if !ok {
		return nil, nil
	}
	return &entry, nil
}

// Has checks if a blob exists by hash.
func (s *InMemoryBlobStore) Has(_ context.Context, hash string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.blobs[hash]
	return ok, nil
}

// Delete removes a blob by hash. Returns true if deleted, false if not found.
func (s *InMemoryBlobStore) Delete(_ context.Context, hash string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.blobs[hash]; !ok {
		return false, nil
	}
	delete(s.blobs, hash)
	return true, nil
}

// PutMany stores multiple blobs in a batch. Skips any that already exist.
func (s *InMemoryBlobStore) PutMany(_ context.Context, entries []StorageBlobEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entry := range entries {
		if _, exists := s.blobs[entry.Hash]; !exists {
			s.blobs[entry.Hash] = entry
		}
	}
	return nil
}

// GetMany retrieves multiple blobs by their hashes.
// Returns a map of hash -> entry. Missing hashes are omitted.
func (s *InMemoryBlobStore) GetMany(_ context.Context, hashes []string) (map[string]StorageBlobEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]StorageBlobEntry, len(hashes))
	for _, hash := range hashes {
		if entry, ok := s.blobs[hash]; ok {
			result[hash] = entry
		}
	}
	return result, nil
}

// DangerouslyClearAll deletes all blobs. Used for testing.
func (s *InMemoryBlobStore) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.blobs = make(map[string]StorageBlobEntry)
	return nil
}
