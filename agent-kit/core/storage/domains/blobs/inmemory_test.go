// Ported from: stores/_test-utils/src/domains/blobs/index.ts (createBlobsTest)
//
// The upstream mastra project has no dedicated blobs storage test file at
// packages/core/src/storage/domains/blobs/blobs.test.ts — the canonical
// storage-level tests live in stores/_test-utils/src/domains/blobs/index.ts
// and are re-used by each storage adapter. This Go file faithfully ports
// those tests against InMemoryBlobStore.
package blobs

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// hashContent returns the SHA-256 hex digest of the given content, matching
// the content-addressable keying scheme used by BlobStore.
func hashContent(content []byte) string {
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h[:])
}

// makeBlobEntry creates a StorageBlobEntry with a deterministic hash.
func makeBlobEntry(content string) StorageBlobEntry {
	data := []byte(content)
	return StorageBlobEntry{
		Hash:    hashContent(data),
		Content: data,
	}
}

// ===========================================================================
// Tests — Init & DangerouslyClearAll
// ===========================================================================

func TestInMemoryBlobStore_Init(t *testing.T) {
	// Init is a no-op for in-memory; verify it does not error.
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	if err := store.Init(ctx); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
}

func TestInMemoryBlobStore_DangerouslyClearAll(t *testing.T) {
	// Verify that DangerouslyClearAll removes all stored blobs.
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	blob1 := makeBlobEntry("hello world")
	blob2 := makeBlobEntry("goodbye world")

	if err := store.Put(ctx, blob1); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}
	if err := store.Put(ctx, blob2); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	// Confirm they exist.
	has1, _ := store.Has(ctx, blob1.Hash)
	has2, _ := store.Has(ctx, blob2.Hash)
	if !has1 || !has2 {
		t.Fatal("expected both blobs to exist before clear")
	}

	// Clear all.
	if err := store.DangerouslyClearAll(ctx); err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	// Confirm they are gone.
	has1, _ = store.Has(ctx, blob1.Hash)
	has2, _ = store.Has(ctx, blob2.Hash)
	if has1 {
		t.Error("expected first blob to be cleared")
	}
	if has2 {
		t.Error("expected second blob to be cleared")
	}
}

// ===========================================================================
// Tests — Put & Get
// ===========================================================================

func TestInMemoryBlobStore_PutAndGet(t *testing.T) {
	// TS: "should store and retrieve a blob by hash"
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	blob := makeBlobEntry("test content")

	if err := store.Put(ctx, blob); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	retrieved, err := store.Get(ctx, blob.Hash)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected retrieved blob to not be nil")
	}
	if retrieved.Hash != blob.Hash {
		t.Errorf("hash mismatch: got %s, want %s", retrieved.Hash, blob.Hash)
	}
	if string(retrieved.Content) != string(blob.Content) {
		t.Errorf("content mismatch: got %q, want %q", string(retrieved.Content), string(blob.Content))
	}
}

func TestInMemoryBlobStore_Put_Idempotent(t *testing.T) {
	// TS: "should be a no-op if the hash already exists"
	// Putting the same hash twice should not overwrite the existing blob.
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	blob := makeBlobEntry("original content")

	if err := store.Put(ctx, blob); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	// Try to put again with same hash but different content (simulating a collision scenario).
	// The implementation should be a no-op — the original content is preserved.
	duplicate := StorageBlobEntry{
		Hash:    blob.Hash,
		Content: []byte("different content"),
	}
	if err := store.Put(ctx, duplicate); err != nil {
		t.Fatalf("Put (duplicate) returned error: %v", err)
	}

	// Retrieve and verify original content is preserved.
	retrieved, err := store.Get(ctx, blob.Hash)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected retrieved blob to not be nil")
	}
	if string(retrieved.Content) != string(blob.Content) {
		t.Errorf("expected original content %q, got %q", string(blob.Content), string(retrieved.Content))
	}
}

func TestInMemoryBlobStore_Get_NotFound(t *testing.T) {
	// TS: "should return nil if not found"
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	result, err := store.Get(ctx, "nonexistent-hash")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for nonexistent blob, got %+v", result)
	}
}

// ===========================================================================
// Tests — Has
// ===========================================================================

func TestInMemoryBlobStore_Has_Exists(t *testing.T) {
	// TS: "should return true if blob exists"
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	blob := makeBlobEntry("check existence")
	if err := store.Put(ctx, blob); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	exists, err := store.Has(ctx, blob.Hash)
	if err != nil {
		t.Fatalf("Has returned error: %v", err)
	}
	if !exists {
		t.Error("expected Has to return true for existing blob")
	}
}

func TestInMemoryBlobStore_Has_NotExists(t *testing.T) {
	// TS: "should return false if blob does not exist"
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	exists, err := store.Has(ctx, "nonexistent-hash")
	if err != nil {
		t.Fatalf("Has returned error: %v", err)
	}
	if exists {
		t.Error("expected Has to return false for nonexistent blob")
	}
}

// ===========================================================================
// Tests — Delete
// ===========================================================================

func TestInMemoryBlobStore_Delete_Exists(t *testing.T) {
	// TS: "should delete an existing blob and return true"
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	blob := makeBlobEntry("to be deleted")
	if err := store.Put(ctx, blob); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	deleted, err := store.Delete(ctx, blob.Hash)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if !deleted {
		t.Error("expected Delete to return true for existing blob")
	}

	// Verify it is actually gone.
	result, err := store.Get(ctx, blob.Hash)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result != nil {
		t.Error("expected blob to be deleted")
	}
}

func TestInMemoryBlobStore_Delete_NotExists(t *testing.T) {
	// TS: "should return false if the blob didn't exist"
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	deleted, err := store.Delete(ctx, "nonexistent-hash")
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if deleted {
		t.Error("expected Delete to return false for nonexistent blob")
	}
}

func TestInMemoryBlobStore_Delete_ThenHas(t *testing.T) {
	// After deleting a blob, Has should return false.
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	blob := makeBlobEntry("delete then check")
	if err := store.Put(ctx, blob); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	if _, err := store.Delete(ctx, blob.Hash); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	exists, err := store.Has(ctx, blob.Hash)
	if err != nil {
		t.Fatalf("Has returned error: %v", err)
	}
	if exists {
		t.Error("expected Has to return false after delete")
	}
}

// ===========================================================================
// Tests — PutMany
// ===========================================================================

func TestInMemoryBlobStore_PutMany(t *testing.T) {
	// TS: "should store multiple blobs in a batch"
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	blobs := []StorageBlobEntry{
		makeBlobEntry("batch content 1"),
		makeBlobEntry("batch content 2"),
		makeBlobEntry("batch content 3"),
	}

	if err := store.PutMany(ctx, blobs); err != nil {
		t.Fatalf("PutMany returned error: %v", err)
	}

	// Verify all were stored.
	for _, blob := range blobs {
		exists, err := store.Has(ctx, blob.Hash)
		if err != nil {
			t.Fatalf("Has returned error for hash %s: %v", blob.Hash, err)
		}
		if !exists {
			t.Errorf("expected blob with hash %s to exist after PutMany", blob.Hash)
		}
	}
}

func TestInMemoryBlobStore_PutMany_SkipsExisting(t *testing.T) {
	// TS: "should skip blobs that already exist"
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	existing := makeBlobEntry("already here")
	if err := store.Put(ctx, existing); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	newBlob := makeBlobEntry("new content")

	// Put both — existing should be skipped, newBlob should be stored.
	if err := store.PutMany(ctx, []StorageBlobEntry{existing, newBlob}); err != nil {
		t.Fatalf("PutMany returned error: %v", err)
	}

	// Verify existing content is preserved (not overwritten).
	retrieved, err := store.Get(ctx, existing.Hash)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(retrieved.Content) != string(existing.Content) {
		t.Errorf("expected original content to be preserved, got %q", string(retrieved.Content))
	}

	// Verify new blob was stored.
	has, err := store.Has(ctx, newBlob.Hash)
	if err != nil {
		t.Fatalf("Has returned error: %v", err)
	}
	if !has {
		t.Error("expected new blob to be stored by PutMany")
	}
}

func TestInMemoryBlobStore_PutMany_EmptySlice(t *testing.T) {
	// Edge case: PutMany with an empty slice should be a no-op.
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	if err := store.PutMany(ctx, []StorageBlobEntry{}); err != nil {
		t.Fatalf("PutMany with empty slice returned error: %v", err)
	}

	if err := store.PutMany(ctx, nil); err != nil {
		t.Fatalf("PutMany with nil slice returned error: %v", err)
	}
}

// ===========================================================================
// Tests — GetMany
// ===========================================================================

func TestInMemoryBlobStore_GetMany(t *testing.T) {
	// TS: "should retrieve multiple blobs by their hashes"
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	blob1 := makeBlobEntry("multi get 1")
	blob2 := makeBlobEntry("multi get 2")
	blob3 := makeBlobEntry("multi get 3")

	if err := store.PutMany(ctx, []StorageBlobEntry{blob1, blob2, blob3}); err != nil {
		t.Fatalf("PutMany returned error: %v", err)
	}

	result, err := store.GetMany(ctx, []string{blob1.Hash, blob2.Hash, blob3.Hash})
	if err != nil {
		t.Fatalf("GetMany returned error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}

	for _, blob := range []StorageBlobEntry{blob1, blob2, blob3} {
		entry, ok := result[blob.Hash]
		if !ok {
			t.Errorf("expected hash %s to be in results", blob.Hash)
			continue
		}
		if string(entry.Content) != string(blob.Content) {
			t.Errorf("content mismatch for hash %s: got %q, want %q",
				blob.Hash, string(entry.Content), string(blob.Content))
		}
	}
}

func TestInMemoryBlobStore_GetMany_OmitsMissing(t *testing.T) {
	// TS: "should omit missing hashes from the result map"
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	existing := makeBlobEntry("exists")
	if err := store.Put(ctx, existing); err != nil {
		t.Fatalf("Put returned error: %v", err)
	}

	result, err := store.GetMany(ctx, []string{existing.Hash, "nonexistent-hash-1", "nonexistent-hash-2"})
	if err != nil {
		t.Fatalf("GetMany returned error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result (missing hashes omitted), got %d", len(result))
	}
	if _, ok := result[existing.Hash]; !ok {
		t.Error("expected existing hash to be in results")
	}
	if _, ok := result["nonexistent-hash-1"]; ok {
		t.Error("expected nonexistent-hash-1 to be omitted")
	}
	if _, ok := result["nonexistent-hash-2"]; ok {
		t.Error("expected nonexistent-hash-2 to be omitted")
	}
}

func TestInMemoryBlobStore_GetMany_EmptyInput(t *testing.T) {
	// Edge case: GetMany with empty/nil input returns empty map.
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	result, err := store.GetMany(ctx, []string{})
	if err != nil {
		t.Fatalf("GetMany with empty slice returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}

	result, err = store.GetMany(ctx, nil)
	if err != nil {
		t.Fatalf("GetMany with nil slice returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results for nil input, got %d", len(result))
	}
}

func TestInMemoryBlobStore_GetMany_AllMissing(t *testing.T) {
	// Edge case: GetMany when none of the hashes exist returns empty map.
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	result, err := store.GetMany(ctx, []string{"missing-1", "missing-2", "missing-3"})
	if err != nil {
		t.Fatalf("GetMany returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 results when all hashes are missing, got %d", len(result))
	}
}

// ===========================================================================
// Tests — Concurrency Safety
// ===========================================================================

func TestInMemoryBlobStore_ConcurrentPutAndGet(t *testing.T) {
	// Verify that concurrent Put and Get operations do not race.
	ctx := context.Background()
	store := NewInMemoryBlobStore()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Concurrent puts.
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			blob := makeBlobEntry(fmt.Sprintf("concurrent content %d", n))
			if err := store.Put(ctx, blob); err != nil {
				t.Errorf("concurrent Put failed: %v", err)
			}
		}(i)
	}

	// Concurrent gets (may return nil for not-yet-stored blobs, which is fine).
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			blob := makeBlobEntry(fmt.Sprintf("concurrent content %d", n))
			_, err := store.Get(ctx, blob.Hash)
			if err != nil {
				t.Errorf("concurrent Get failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

// ===========================================================================
// Tests — Interface Compliance
// ===========================================================================

func TestInMemoryBlobStore_ImplementsBlobStore(t *testing.T) {
	// Compile-time check is in the production code (var _ BlobStore = ...),
	// but this test documents and validates the interface compliance at runtime.
	var store BlobStore = NewInMemoryBlobStore()
	if store == nil {
		t.Fatal("expected NewInMemoryBlobStore to return a non-nil BlobStore")
	}
}
