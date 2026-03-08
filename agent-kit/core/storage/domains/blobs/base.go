// Ported from: packages/core/src/storage/domains/blobs/base.ts
package blobs

import "context"

// StorageBlobEntry represents a content-addressable blob entry.
// This is the canonical type; storage/types.go re-exports it.
type StorageBlobEntry struct {
	// SHA-256 hash of the blob content, used as the key.
	Hash string `json:"hash"`
	// The raw blob content.
	Content []byte `json:"content"`
}

// ---------------------------------------------------------------------------
// BlobStore Interface
// ---------------------------------------------------------------------------

// BlobStore is the interface for content-addressable blob storage.
// Used to store file contents for skill versioning.
//
// Blobs are keyed by their SHA-256 hash, providing natural deduplication.
type BlobStore interface {
	// Init initializes the blob store (create tables, etc).
	Init(ctx context.Context) error

	// Put stores a blob. If the hash already exists, this is a no-op.
	Put(ctx context.Context, entry StorageBlobEntry) error

	// Get retrieves a blob by its hash.
	// Returns nil if not found.
	Get(ctx context.Context, hash string) (*StorageBlobEntry, error)

	// Has checks if a blob exists by hash.
	Has(ctx context.Context, hash string) (bool, error)

	// Delete removes a blob by hash.
	// Returns true if the blob was deleted, false if it didn't exist.
	Delete(ctx context.Context, hash string) (bool, error)

	// PutMany stores multiple blobs in a batch. Skips any that already exist.
	PutMany(ctx context.Context, entries []StorageBlobEntry) error

	// GetMany retrieves multiple blobs by their hashes.
	// Returns a map of hash -> entry. Missing hashes are omitted.
	GetMany(ctx context.Context, hashes []string) (map[string]StorageBlobEntry, error)

	// DangerouslyClearAll deletes all blobs. Used for testing.
	DangerouslyClearAll(ctx context.Context) error
}
