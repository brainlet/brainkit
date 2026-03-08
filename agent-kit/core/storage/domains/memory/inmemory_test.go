// Ported from: stores/_test-utils/src/domains/memory/index.ts
// and: stores/_test-utils/src/domains/memory/threads.ts
// and: stores/_test-utils/src/domains/memory/messages-list.ts
// and: stores/_test-utils/src/domains/memory/messages-paginated.ts
// and: stores/_test-utils/src/domains/memory/messages-bulk-delete.ts
// and: stores/_test-utils/src/domains/memory/messages-update.ts
// and: stores/_test-utils/src/domains/memory/resources.ts
// and: stores/_test-utils/src/domains/memory/data.ts
//
// The upstream mastra project has no dedicated memory storage test file at
// packages/core/src/storage/domains/memory/memory.test.ts — the canonical
// storage-level tests live in stores/_test-utils/src/domains/memory/ and are
// re-used by each storage adapter. This Go file faithfully ports those tests
// against InMemoryMemory.
package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// Test helpers — ported from stores/_test-utils/src/domains/memory/data.ts
// ---------------------------------------------------------------------------

// roleAlternator alternates between "user" and "assistant" roles,
// mirroring the TS getRole()/resetRole() helpers.
type roleAlternator struct {
	current string
}

func newRoleAlternator() *roleAlternator {
	return &roleAlternator{current: "assistant"}
}

func (r *roleAlternator) next() string {
	if r.current == "user" {
		r.current = "assistant"
	} else {
		r.current = "user"
	}
	return r.current
}

func (r *roleAlternator) reset() {
	r.current = "assistant"
}

// createSampleThread mirrors the TS createSampleThread helper.
func createSampleThread(opts ...func(map[string]any)) StorageThreadType {
	now := time.Now()
	t := map[string]any{
		"id":         "thread-" + uuid.New().String(),
		"resourceId": "resource-" + uuid.New().String(),
		"title":      "Test Thread",
		"createdAt":  now,
		"updatedAt":  now,
		"metadata":   map[string]any{"key": "value"},
	}
	for _, fn := range opts {
		fn(t)
	}
	return t
}

// createSampleThreadWithParams mirrors the TS createSampleThreadWithParams helper.
func createSampleThreadWithParams(threadID, resourceID string, createdAt, updatedAt time.Time) StorageThreadType {
	return map[string]any{
		"id":         threadID,
		"resourceId": resourceID,
		"title":      "Test Thread with given ThreadId and ResourceId",
		"createdAt":  createdAt,
		"updatedAt":  updatedAt,
		"metadata":   map[string]any{"key": "value"},
	}
}

// createSampleMessageV2 mirrors the TS createSampleMessageV2 helper.
// It returns a MastraDBMessage (map[string]any) in V2 format.
func createSampleMessageV2(threadID string, opts ...func(map[string]any)) MastraDBMessage {
	contentText := "Sample content " + uuid.New().String()
	msg := map[string]any{
		"id":         uuid.New().String(),
		"threadId":   threadID,
		"thread_id":  threadID, // inmemory.go uses thread_id for storage lookups
		"resourceId": "test-resource",
		"role":       "user",
		"createdAt":  time.Now(),
		"content": map[string]any{
			"format":  2,
			"parts":   []any{map[string]any{"type": "text", "text": contentText}},
			"content": contentText,
		},
	}
	for _, fn := range opts {
		fn(msg)
	}
	// Ensure thread_id stays in sync with threadId.
	if tid, ok := msg["threadId"].(string); ok {
		msg["thread_id"] = tid
	}
	return msg
}

// withContent sets the content.content field on a message.
func withContent(c string) func(map[string]any) {
	return func(m map[string]any) {
		content, _ := m["content"].(map[string]any)
		if content == nil {
			content = map[string]any{"format": 2}
		}
		content["content"] = c
		content["parts"] = []any{map[string]any{"type": "text", "text": c}}
		m["content"] = content
	}
}

// withResourceID sets the resourceId field on a message.
func withResourceID(rid string) func(map[string]any) {
	return func(m map[string]any) {
		m["resourceId"] = rid
	}
}

// withCreatedAt sets the createdAt field on a message.
func withCreatedAt(t time.Time) func(map[string]any) {
	return func(m map[string]any) {
		m["createdAt"] = t
	}
}

// withRole sets the role field on a message.
func withRole(role string) func(map[string]any) {
	return func(m map[string]any) {
		m["role"] = role
	}
}

// withID sets the id field on a message.
func withID(id string) func(map[string]any) {
	return func(m map[string]any) {
		m["id"] = id
	}
}

// createSampleResource mirrors the TS createSampleResource helper.
func createSampleResource(opts ...func(map[string]any)) StorageResourceType {
	now := time.Now()
	r := map[string]any{
		"id":            "resource-" + uuid.New().String(),
		"workingMemory": "Sample working memory content",
		"metadata":      map[string]any{"key": "value", "test": true},
		"createdAt":     now,
		"updatedAt":     now,
	}
	for _, fn := range opts {
		fn(r)
	}
	return r
}

// newStorage creates a fresh InMemoryMemory instance for testing.
func newStorage() *InMemoryMemory {
	return NewInMemoryMemory(domains.NewInMemoryDB())
}

// getContentString extracts the content.content string from a message map.
func getContentString(msg MastraDBMessage) string {
	content, _ := msg["content"].(map[string]any)
	if content == nil {
		return ""
	}
	s, _ := content["content"].(string)
	return s
}

// getTimeField extracts a time.Time from a map field.
func getTimeField(m map[string]any, field string) time.Time {
	if t, ok := m[field].(time.Time); ok {
		return t
	}
	if s, ok := m[field].(string); ok {
		t, _ := time.Parse(time.RFC3339Nano, s)
		return t
	}
	return time.Time{}
}

// ===========================================================================
// Tests — Init & DangerouslyClearAll
// ===========================================================================

func TestInMemoryMemory_Init(t *testing.T) {
	// Init is a no-op for in-memory; just verify it doesn't error.
	ctx := context.Background()
	storage := newStorage()

	if err := storage.Init(ctx); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
}

func TestInMemoryMemory_SupportsObservationalMemory(t *testing.T) {
	storage := newStorage()
	if !storage.SupportsObservationalMemory() {
		t.Error("expected SupportsObservationalMemory() to return true")
	}
}

func TestInMemoryMemory_DangerouslyClearAll(t *testing.T) {
	// TS pattern: verify clear removes all data.
	ctx := context.Background()
	storage := newStorage()

	// Save a thread and a message.
	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	msg := createSampleMessageV2(thread["id"].(string))
	if _, err := storage.SaveMessages(ctx, []MastraDBMessage{msg}); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	// Clear all.
	if err := storage.DangerouslyClearAll(ctx); err != nil {
		t.Fatalf("DangerouslyClearAll returned error: %v", err)
	}

	// Verify thread is gone.
	got, err := storage.GetThreadByID(ctx, thread["id"].(string))
	if err != nil {
		t.Fatalf("GetThreadByID returned error: %v", err)
	}
	if got != nil {
		t.Error("expected thread to be cleared")
	}
}

// ===========================================================================
// Tests — Threads (ported from threads.ts)
// ===========================================================================

func TestInMemoryMemory_CreateAndRetrieveThread(t *testing.T) {
	// TS: "should create and retrieve a thread"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	saved, err := storage.SaveThread(ctx, thread)
	if err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	if saved["id"] != thread["id"] {
		t.Errorf("expected saved id=%v, got %v", thread["id"], saved["id"])
	}

	retrieved, err := storage.GetThreadByID(ctx, thread["id"].(string))
	if err != nil {
		t.Fatalf("GetThreadByID returned error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected non-nil thread")
	}
	if retrieved["title"] != thread["title"] {
		t.Errorf("title mismatch: got %v, want %v", retrieved["title"], thread["title"])
	}
}

func TestInMemoryMemory_CreateAndRetrieveWithGivenIDs(t *testing.T) {
	// TS: "should create and retrieve a thread with the same given threadId and resourceId"
	ctx := context.Background()
	storage := newStorage()

	exampleThreadID := "1346362547862769664"
	exampleResourceID := "532374164040974346"
	createdAt := time.Now()
	updatedAt := time.Now()
	thread := createSampleThreadWithParams(exampleThreadID, exampleResourceID, createdAt, updatedAt)

	saved, err := storage.SaveThread(ctx, thread)
	if err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	if saved["id"] != exampleThreadID {
		t.Errorf("expected id=%s, got %v", exampleThreadID, saved["id"])
	}

	retrieved, err := storage.GetThreadByID(ctx, exampleThreadID)
	if err != nil {
		t.Fatalf("GetThreadByID returned error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected non-nil thread")
	}
	if retrieved["id"] != exampleThreadID {
		t.Errorf("expected id=%s, got %v", exampleThreadID, retrieved["id"])
	}
	if retrieved["resourceId"] != exampleResourceID {
		t.Errorf("expected resourceId=%s, got %v", exampleResourceID, retrieved["resourceId"])
	}
	if retrieved["title"] != thread["title"] {
		t.Errorf("title mismatch: got %v, want %v", retrieved["title"], thread["title"])
	}
}

func TestInMemoryMemory_NonExistentThread(t *testing.T) {
	// TS: "should return null for non-existent thread"
	ctx := context.Background()
	storage := newStorage()

	result, err := storage.GetThreadByID(ctx, "non-existent")
	if err != nil {
		t.Fatalf("GetThreadByID returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-existent thread, got %+v", result)
	}
}

func TestInMemoryMemory_GetThreadsByResourceID(t *testing.T) {
	// TS: "should get threads by resource ID"
	ctx := context.Background()
	storage := newStorage()

	thread1 := createSampleThread()
	thread2 := createSampleThread()
	thread2["resourceId"] = thread1["resourceId"]

	if _, err := storage.SaveThread(ctx, thread1); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	if _, err := storage.SaveThread(ctx, thread2); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	perPage := 10
	result, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter:  &ThreadsFilter{ResourceID: thread1["resourceId"].(string)},
		Page:    0,
		PerPage: &perPage,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(result.Threads) != 2 {
		t.Fatalf("expected 2 threads, got %d", len(result.Threads))
	}

	ids := map[string]bool{}
	for _, th := range result.Threads {
		ids[th["id"].(string)] = true
	}
	if !ids[thread1["id"].(string)] || !ids[thread2["id"].(string)] {
		t.Error("expected both thread IDs in results")
	}
}

func TestInMemoryMemory_UpdateThreadTitleAndMetadata(t *testing.T) {
	// TS: "should update thread title and metadata"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	newMetadata := map[string]any{"newKey": "newValue"}
	updated, err := storage.UpdateThread(ctx, UpdateThreadInput{
		ID:       thread["id"].(string),
		Title:    "Updated Title",
		Metadata: newMetadata,
	})
	if err != nil {
		t.Fatalf("UpdateThread returned error: %v", err)
	}

	if updated["title"] != "Updated Title" {
		t.Errorf("expected title=Updated Title, got %v", updated["title"])
	}
	meta, _ := updated["metadata"].(map[string]any)
	if meta == nil {
		t.Fatal("expected non-nil metadata")
	}
	// Metadata should be merged: original key + newKey.
	if meta["key"] != "value" {
		t.Errorf("expected original key=value in metadata")
	}
	if meta["newKey"] != "newValue" {
		t.Errorf("expected newKey=newValue in metadata")
	}

	// Verify persistence.
	retrieved, err := storage.GetThreadByID(ctx, thread["id"].(string))
	if err != nil {
		t.Fatalf("GetThreadByID returned error: %v", err)
	}
	if retrieved["title"] != "Updated Title" {
		t.Errorf("expected persisted title=Updated Title, got %v", retrieved["title"])
	}
}

func TestInMemoryMemory_DeleteThread(t *testing.T) {
	// TS: "should delete thread"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	if err := storage.DeleteThread(ctx, thread["id"].(string)); err != nil {
		t.Fatalf("DeleteThread returned error: %v", err)
	}

	retrieved, err := storage.GetThreadByID(ctx, thread["id"].(string))
	if err != nil {
		t.Fatalf("GetThreadByID returned error: %v", err)
	}
	if retrieved != nil {
		t.Error("expected thread to be nil after deletion")
	}
}

func TestInMemoryMemory_DeleteThreadAndMessages(t *testing.T) {
	// TS: "should delete thread and its messages"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	messages := []MastraDBMessage{
		createSampleMessageV2(threadID),
		createSampleMessageV2(threadID),
	}
	if _, err := storage.SaveMessages(ctx, messages); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	if err := storage.DeleteThread(ctx, threadID); err != nil {
		t.Fatalf("DeleteThread returned error: %v", err)
	}

	// Verify thread is gone.
	retrieved, err := storage.GetThreadByID(ctx, threadID)
	if err != nil {
		t.Fatalf("GetThreadByID returned error: %v", err)
	}
	if retrieved != nil {
		t.Error("expected thread to be nil after deletion")
	}

	// Verify messages were also deleted.
	result, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: threadID,
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if len(result.Messages) != 0 {
		t.Errorf("expected 0 messages after thread deletion, got %d", len(result.Messages))
	}
}

func TestInMemoryMemory_PaginatedThreads(t *testing.T) {
	// TS: "should return paginated threads with total count"
	ctx := context.Background()
	storage := newStorage()

	resourceID := "pg-paginated-resource-" + uuid.New().String()
	for i := 0; i < 17; i++ {
		th := createSampleThread()
		th["resourceId"] = resourceID
		if _, err := storage.SaveThread(ctx, th); err != nil {
			t.Fatalf("SaveThread returned error: %v", err)
		}
	}

	perPage7 := 7
	page1, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter:  &ThreadsFilter{ResourceID: resourceID},
		Page:    0,
		PerPage: &perPage7,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(page1.Threads) != 7 {
		t.Fatalf("expected 7 threads on page 1, got %d", len(page1.Threads))
	}
	if page1.Total != 17 {
		t.Errorf("expected total=17, got %d", page1.Total)
	}
	if page1.Page != 0 {
		t.Errorf("expected page=0, got %d", page1.Page)
	}
	if page1.PerPage != 7 {
		t.Errorf("expected perPage=7, got %d", page1.PerPage)
	}
	if !page1.HasMore {
		t.Error("expected hasMore=true")
	}

	page3, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter:  &ThreadsFilter{ResourceID: resourceID},
		Page:    2,
		PerPage: &perPage7,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(page3.Threads) != 3 {
		t.Fatalf("expected 3 threads on page 3, got %d", len(page3.Threads))
	}
	if page3.Total != 17 {
		t.Errorf("expected total=17, got %d", page3.Total)
	}
	if page3.HasMore {
		t.Error("expected hasMore=false on last page")
	}
}

func TestInMemoryMemory_ListThreadsNoFilter(t *testing.T) {
	// TS: "should return paginated results when no pagination params for listThreads"
	ctx := context.Background()
	storage := newStorage()

	resourceID := "pg-non-paginated-resource-" + uuid.New().String()
	th := createSampleThread()
	th["resourceId"] = resourceID
	if _, err := storage.SaveThread(ctx, th); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	perPage100 := 100
	results, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter:  &ThreadsFilter{ResourceID: resourceID},
		Page:    0,
		PerPage: &perPage100,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(results.Threads) != 1 {
		t.Errorf("expected 1 thread, got %d", len(results.Threads))
	}
	if results.Total != 1 {
		t.Errorf("expected total=1, got %d", results.Total)
	}
	if results.HasMore {
		t.Error("expected hasMore=false")
	}
}

// ===========================================================================
// Edge Cases and Error Handling (ported from threads.ts)
// ===========================================================================

func TestInMemoryMemory_LargeMetadata(t *testing.T) {
	// TS: "should handle large metadata objects"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	largeArray := make([]any, 10)
	for i := range largeArray {
		largeArray[i] = map[string]any{"index": i, "data": strings.Repeat("test", 10)}
	}
	meta, _ := thread["metadata"].(map[string]any)
	meta["largeArray"] = largeArray
	thread["metadata"] = meta

	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	retrieved, err := storage.GetThreadByID(ctx, thread["id"].(string))
	if err != nil {
		t.Fatalf("GetThreadByID returned error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected non-nil thread")
	}
	retMeta, _ := retrieved["metadata"].(map[string]any)
	retArray, _ := retMeta["largeArray"].([]any)
	if len(retArray) != 10 {
		t.Errorf("expected 10 items in largeArray, got %d", len(retArray))
	}
}

func TestInMemoryMemory_SpecialCharactersInTitle(t *testing.T) {
	// TS: "should handle special characters in thread titles"
	ctx := context.Background()
	storage := newStorage()

	title := `Special 'quotes' and "double quotes" and emoji`
	thread := createSampleThread()
	thread["title"] = title

	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	retrieved, err := storage.GetThreadByID(ctx, thread["id"].(string))
	if err != nil {
		t.Fatalf("GetThreadByID returned error: %v", err)
	}
	if retrieved["title"] != title {
		t.Errorf("title mismatch: got %v, want %v", retrieved["title"], title)
	}
}

// ===========================================================================
// listThreads with filtering (ported from threads.ts)
// ===========================================================================

func TestInMemoryMemory_ListThreadsFilterByResourceID(t *testing.T) {
	// TS: "should list threads filtered by resourceId only"
	ctx := context.Background()
	storage := newStorage()

	resourceID1 := uuid.New().String()
	resourceID2 := uuid.New().String()

	thread1 := createSampleThreadWithParams(uuid.New().String(), resourceID1, time.Now(), time.Now())
	thread1["metadata"] = map[string]any{"category": "support-filter"}
	thread2 := createSampleThreadWithParams(uuid.New().String(), resourceID1, time.Now(), time.Now())
	thread2["metadata"] = map[string]any{"category": "support-filter"}
	thread3 := createSampleThreadWithParams(uuid.New().String(), resourceID2, time.Now(), time.Now())
	thread3["metadata"] = map[string]any{"category": "sales-filter"}

	for _, th := range []StorageThreadType{thread1, thread2, thread3} {
		if _, err := storage.SaveThread(ctx, th); err != nil {
			t.Fatalf("SaveThread returned error: %v", err)
		}
	}

	perPage := 10
	result, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter:  &ThreadsFilter{ResourceID: resourceID1},
		Page:    0,
		PerPage: &perPage,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(result.Threads) != 2 {
		t.Fatalf("expected 2 threads, got %d", len(result.Threads))
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
}

func TestInMemoryMemory_ListThreadsFilterByMetadata(t *testing.T) {
	// TS: "should list threads filtered by metadata only"
	ctx := context.Background()
	storage := newStorage()

	thread1 := createSampleThread()
	thread1["metadata"] = map[string]any{"category": "support-filter", "priority": "high"}
	thread2 := createSampleThread()
	thread2["metadata"] = map[string]any{"category": "support-filter", "priority": "low"}
	thread3 := createSampleThread()
	thread3["metadata"] = map[string]any{"category": "sales-filter"}

	for _, th := range []StorageThreadType{thread1, thread2, thread3} {
		if _, err := storage.SaveThread(ctx, th); err != nil {
			t.Fatalf("SaveThread returned error: %v", err)
		}
	}

	perPage := 10
	result, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter:  &ThreadsFilter{Metadata: map[string]any{"category": "support-filter"}},
		Page:    0,
		PerPage: &perPage,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(result.Threads) != 2 {
		t.Fatalf("expected 2 threads, got %d", len(result.Threads))
	}
	if result.Total != 2 {
		t.Errorf("expected total=2, got %d", result.Total)
	}
}

func TestInMemoryMemory_ListThreadsFilterByMultipleMetadata(t *testing.T) {
	// TS: "should list threads filtered by multiple metadata fields (AND logic)"
	ctx := context.Background()
	storage := newStorage()

	thread1 := createSampleThread()
	thread1["metadata"] = map[string]any{"category": "support-filter", "priority": "high"}
	thread2 := createSampleThread()
	thread2["metadata"] = map[string]any{"category": "support-filter", "priority": "low"}

	for _, th := range []StorageThreadType{thread1, thread2} {
		if _, err := storage.SaveThread(ctx, th); err != nil {
			t.Fatalf("SaveThread returned error: %v", err)
		}
	}

	perPage := 10
	result, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter: &ThreadsFilter{
			Metadata: map[string]any{"category": "support-filter", "priority": "high"},
		},
		Page:    0,
		PerPage: &perPage,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(result.Threads) != 1 {
		t.Fatalf("expected 1 thread, got %d", len(result.Threads))
	}
	if result.Threads[0]["id"] != thread1["id"] {
		t.Errorf("expected thread1 id, got %v", result.Threads[0]["id"])
	}
}

func TestInMemoryMemory_ListThreadsNoMatchingFilter(t *testing.T) {
	// TS: "should return empty array when no threads match the filter"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	thread["metadata"] = map[string]any{"category": "test"}
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	perPage := 10
	result, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter:  &ThreadsFilter{Metadata: map[string]any{"category": "nonexistent"}},
		Page:    0,
		PerPage: &perPage,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(result.Threads) != 0 {
		t.Errorf("expected 0 threads, got %d", len(result.Threads))
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

func TestInMemoryMemory_ListThreadsNonExistentResource(t *testing.T) {
	// TS: "should return empty array when resourceId does not exist"
	ctx := context.Background()
	storage := newStorage()

	perPage := 10
	result, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter:  &ThreadsFilter{ResourceID: "nonexistent-resource"},
		Page:    0,
		PerPage: &perPage,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(result.Threads) != 0 {
		t.Errorf("expected 0 threads, got %d", len(result.Threads))
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

func TestInMemoryMemory_PaginateFilteredThreads(t *testing.T) {
	// TS: "should paginate filtered results correctly"
	ctx := context.Background()
	storage := newStorage()

	resourceID := uuid.New().String()
	th1 := createSampleThread()
	th1["resourceId"] = resourceID
	th2 := createSampleThread()
	th2["resourceId"] = resourceID

	if _, err := storage.SaveThread(ctx, th1); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	if _, err := storage.SaveThread(ctx, th2); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	perPage1 := 1
	page1, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter:  &ThreadsFilter{ResourceID: resourceID},
		Page:    0,
		PerPage: &perPage1,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(page1.Threads) != 1 {
		t.Fatalf("expected 1 thread on page 1, got %d", len(page1.Threads))
	}
	if page1.Total != 2 {
		t.Errorf("expected total=2, got %d", page1.Total)
	}
	if !page1.HasMore {
		t.Error("expected hasMore=true")
	}

	page2, err := storage.ListThreads(ctx, StorageListThreadsInput{
		Filter:  &ThreadsFilter{ResourceID: resourceID},
		Page:    1,
		PerPage: &perPage1,
	})
	if err != nil {
		t.Fatalf("ListThreads returned error: %v", err)
	}
	if len(page2.Threads) != 1 {
		t.Fatalf("expected 1 thread on page 2, got %d", len(page2.Threads))
	}
	if page2.HasMore {
		t.Error("expected hasMore=false")
	}

	// Ensure different threads.
	if page1.Threads[0]["id"] == page2.Threads[0]["id"] {
		t.Error("expected different threads on different pages")
	}
}

// ===========================================================================
// Tests — Messages (ported from messages-list.ts and messages-paginated.ts)
// ===========================================================================

func TestInMemoryMemory_ListAllMessages(t *testing.T) {
	// TS: "should list all messages for a thread without pagination"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	now := time.Now()
	messages := make([]MastraDBMessage, 5)
	for i := 0; i < 5; i++ {
		messages[i] = createSampleMessageV2(threadID,
			withContent(fmt.Sprintf("Message %d", i+1)),
			withCreatedAt(now.Add(time.Duration(i+1)*time.Second)),
			withResourceID(thread["resourceId"].(string)),
		)
	}
	if _, err := storage.SaveMessages(ctx, messages); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	result, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: threadID,
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if len(result.Messages) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(result.Messages))
	}
	if result.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Total)
	}
}

func TestInMemoryMemory_ListMessagesWithPagination(t *testing.T) {
	// TS: "should list messages with pagination"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	now := time.Now()
	messages := make([]MastraDBMessage, 5)
	for i := 0; i < 5; i++ {
		messages[i] = createSampleMessageV2(threadID,
			withContent(fmt.Sprintf("Message %d", i+1)),
			withCreatedAt(now.Add(time.Duration(i+1)*time.Second)),
		)
	}
	if _, err := storage.SaveMessages(ctx, messages); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	perPage := 2
	page1, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: threadID,
		PerPage:  &perPage,
		Page:     0,
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if len(page1.Messages) != 2 {
		t.Fatalf("expected 2 messages on page 1, got %d", len(page1.Messages))
	}
	if page1.Total != 5 {
		t.Errorf("expected total=5, got %d", page1.Total)
	}
	if page1.Page != 0 {
		t.Errorf("expected page=0, got %d", page1.Page)
	}
	if page1.PerPage != 2 {
		t.Errorf("expected perPage=2, got %d", page1.PerPage)
	}
	if !page1.HasMore {
		t.Error("expected hasMore=true")
	}

	page2, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: threadID,
		PerPage:  &perPage,
		Page:     1,
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if len(page2.Messages) != 2 {
		t.Errorf("expected 2 messages on page 2, got %d", len(page2.Messages))
	}
	if !page2.HasMore {
		t.Error("expected hasMore=true on page 2")
	}

	page3, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: threadID,
		PerPage:  &perPage,
		Page:     2,
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if len(page3.Messages) != 1 {
		t.Errorf("expected 1 message on page 3, got %d", len(page3.Messages))
	}
	if page3.HasMore {
		t.Error("expected hasMore=false on page 3")
	}
}

func TestInMemoryMemory_ListMessagesSortedASC(t *testing.T) {
	// TS: "should sort messages by createdAt ASC by default"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	now := time.Now()
	messages := make([]MastraDBMessage, 5)
	for i := 0; i < 5; i++ {
		messages[i] = createSampleMessageV2(threadID,
			withContent(fmt.Sprintf("Message %d", i+1)),
			withCreatedAt(now.Add(time.Duration(i+1)*time.Second)),
		)
	}
	if _, err := storage.SaveMessages(ctx, messages); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	result, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: threadID,
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}

	// Verify ascending order.
	for i := 0; i < len(result.Messages)-1; i++ {
		t1 := getTimeField(result.Messages[i], "createdAt")
		t2 := getTimeField(result.Messages[i+1], "createdAt")
		if t1.After(t2) {
			t.Errorf("messages not in ascending order at index %d", i)
		}
	}
}

func TestInMemoryMemory_ListMessagesSortedDESC(t *testing.T) {
	// TS: "should sort messages by createdAt DESC when specified"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	now := time.Now()
	messages := make([]MastraDBMessage, 5)
	for i := 0; i < 5; i++ {
		messages[i] = createSampleMessageV2(threadID,
			withContent(fmt.Sprintf("Message %d", i+1)),
			withCreatedAt(now.Add(time.Duration(i+1)*time.Second)),
		)
	}
	if _, err := storage.SaveMessages(ctx, messages); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	result, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: threadID,
		OrderBy:  &StorageOrderBy{Field: "createdAt", Direction: "DESC"},
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}

	// Verify descending order.
	for i := 0; i < len(result.Messages)-1; i++ {
		t1 := getTimeField(result.Messages[i], "createdAt")
		t2 := getTimeField(result.Messages[i+1], "createdAt")
		if t1.Before(t2) {
			t.Errorf("messages not in descending order at index %d", i)
		}
	}
}

func TestInMemoryMemory_EmptyThread(t *testing.T) {
	// TS: "should handle empty thread"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	result, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: thread["id"].(string),
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if len(result.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result.Messages))
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
	if result.HasMore {
		t.Error("expected hasMore=false")
	}
}

func TestInMemoryMemory_FilterByResourceID(t *testing.T) {
	// TS: "should filter by resourceId"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	threadResourceID := thread["resourceId"].(string)
	now := time.Now()

	// Add 5 messages with the thread's resource ID.
	messages := make([]MastraDBMessage, 5)
	for i := 0; i < 5; i++ {
		messages[i] = createSampleMessageV2(threadID,
			withContent(fmt.Sprintf("Message %d", i+1)),
			withCreatedAt(now.Add(time.Duration(i+1)*time.Second)),
			withResourceID(threadResourceID),
		)
	}
	// Add 1 message with a different resource ID.
	differentMsg := createSampleMessageV2(threadID,
		withContent("Different Resource"),
		withResourceID("different-resource"),
	)
	allMsgs := append(messages, differentMsg)
	if _, err := storage.SaveMessages(ctx, allMsgs); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	result, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID:   threadID,
		ResourceID: threadResourceID,
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if result.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Total)
	}
	for _, msg := range result.Messages {
		if msg["resourceId"] != threadResourceID {
			t.Errorf("expected resourceId=%s, got %v", threadResourceID, msg["resourceId"])
		}
	}
}

func TestInMemoryMemory_FilterByDateRange(t *testing.T) {
	// TS: "should filter by date range"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	now := time.Now()
	twoDaysAgo := now.Add(-48 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)

	dateMessages := []MastraDBMessage{
		createSampleMessageV2(threadID, withContent("Old Message"), withCreatedAt(twoDaysAgo)),
		createSampleMessageV2(threadID, withContent("Yesterday Message"), withCreatedAt(yesterday)),
		createSampleMessageV2(threadID, withContent("Recent Message"), withCreatedAt(now)),
	}
	if _, err := storage.SaveMessages(ctx, dateMessages); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	result, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: threadID,
		Filter: &MessagesFilter{
			DateRange: &DateRangeFilter{Start: &yesterday},
		},
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected total=2 (yesterday + recent), got %d", result.Total)
	}
}

func TestInMemoryMemory_SaveAndRetrieveMessages(t *testing.T) {
	// TS: "should save and retrieve messages"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	messages := []MastraDBMessage{
		createSampleMessageV2(threadID, withContent("Message 1")),
		createSampleMessageV2(threadID, withContent("Message 2")),
	}
	saved, err := storage.SaveMessages(ctx, messages)
	if err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}
	if len(saved) != 2 {
		t.Fatalf("expected 2 saved messages, got %d", len(saved))
	}

	result, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: threadID,
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if len(result.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result.Messages))
	}
}

func TestInMemoryMemory_SaveEmptyMessages(t *testing.T) {
	// TS: "should handle empty message array"
	ctx := context.Background()
	storage := newStorage()

	result, err := storage.SaveMessages(ctx, []MastraDBMessage{})
	if err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestInMemoryMemory_MessageOrder(t *testing.T) {
	// TS: "should maintain message order"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	now := time.Now()
	messages := []MastraDBMessage{
		createSampleMessageV2(threadID, withContent("First"), withCreatedAt(now.Add(1*time.Millisecond))),
		createSampleMessageV2(threadID, withContent("Second"), withCreatedAt(now.Add(2*time.Millisecond))),
		createSampleMessageV2(threadID, withContent("Third"), withCreatedAt(now.Add(3*time.Millisecond))),
	}
	if _, err := storage.SaveMessages(ctx, messages); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	result, err := storage.ListMessages(ctx, StorageListMessagesInput{
		ThreadID: threadID,
	})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if len(result.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result.Messages))
	}

	expected := []string{"First", "Second", "Third"}
	for i, msg := range result.Messages {
		got := getContentString(msg)
		if got != expected[i] {
			t.Errorf("message %d: expected content=%s, got %s", i, expected[i], got)
		}
	}
}

func TestInMemoryMemory_UpsertMessageDifferentThread(t *testing.T) {
	// TS: "should upsert messages: duplicate id and different threadid"
	ctx := context.Background()
	storage := newStorage()

	thread1 := createSampleThread()
	thread2 := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread1); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	if _, err := storage.SaveThread(ctx, thread2); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	t1ID := thread1["id"].(string)
	t2ID := thread2["id"].(string)

	msg := createSampleMessageV2(t1ID, withContent("Thread1 Content"))
	msgID := msg["id"].(string)
	if _, err := storage.SaveMessages(ctx, []MastraDBMessage{msg}); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	// Save same ID to different thread.
	conflicting := createSampleMessageV2(t2ID, withContent("Thread2 Content"), withID(msgID))
	if _, err := storage.SaveMessages(ctx, []MastraDBMessage{conflicting}); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	// Thread 1 should NOT have the message.
	r1, _ := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: t1ID})
	found := false
	for _, m := range r1.Messages {
		if m["id"] == msgID {
			found = true
		}
	}
	if found {
		t.Error("expected message to be moved away from thread1")
	}

	// Thread 2 should have the message with updated content.
	r2, _ := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: t2ID})
	foundInT2 := false
	for _, m := range r2.Messages {
		if m["id"] == msgID {
			foundInT2 = true
			if getContentString(m) != "Thread2 Content" {
				t.Errorf("expected content=Thread2 Content, got %s", getContentString(m))
			}
		}
	}
	if !foundInT2 {
		t.Error("expected message to be in thread2")
	}
}

func TestInMemoryMemory_UpsertMessageSameThread(t *testing.T) {
	// TS: "should upsert messages: duplicate id+threadId results in update, not duplicate row"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	msg := createSampleMessageV2(threadID, withContent("Original"))
	msgID := msg["id"].(string)
	if _, err := storage.SaveMessages(ctx, []MastraDBMessage{msg}); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	// Insert same ID with different content.
	updated := createSampleMessageV2(threadID, withContent("Updated"), withID(msgID))
	if _, err := storage.SaveMessages(ctx, []MastraDBMessage{updated}); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	result, _ := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: threadID})

	// Only one message should exist.
	count := 0
	for _, m := range result.Messages {
		if m["id"] == msgID {
			count++
			if getContentString(m) != "Updated" {
				t.Errorf("expected content=Updated, got %s", getContentString(m))
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 message with id=%s, got %d", msgID, count)
	}
}

// ===========================================================================
// Tests — ListMessagesByID (ported from messages-paginated.ts listMessagesById)
// ===========================================================================

func TestInMemoryMemory_ListMessagesByIDEmpty(t *testing.T) {
	// TS: "should return an empty array if no message IDs are provided"
	ctx := context.Background()
	storage := newStorage()

	result, err := storage.ListMessagesByID(ctx, []string{})
	if err != nil {
		t.Fatalf("ListMessagesByID returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result))
	}
}

func TestInMemoryMemory_ListMessagesByIDMultipleThreads(t *testing.T) {
	// TS: "should return messages from multiple threads"
	ctx := context.Background()
	storage := newStorage()

	thread1 := createSampleThread()
	thread2 := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread1); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	if _, err := storage.SaveThread(ctx, thread2); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	t1ID := thread1["id"].(string)
	t2ID := thread2["id"].(string)

	msg1 := createSampleMessageV2(t1ID, withContent("Message 1"))
	msg2 := createSampleMessageV2(t2ID, withContent("Message A"))

	if _, err := storage.SaveMessages(ctx, []MastraDBMessage{msg1}); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}
	if _, err := storage.SaveMessages(ctx, []MastraDBMessage{msg2}); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	result, err := storage.ListMessagesByID(ctx, []string{msg1["id"].(string), msg2["id"].(string)})
	if err != nil {
		t.Fatalf("ListMessagesByID returned error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	// Verify both threads represented.
	hasT1 := false
	hasT2 := false
	for _, m := range result {
		if m["thread_id"] == t1ID || m["threadId"] == t1ID {
			hasT1 = true
		}
		if m["thread_id"] == t2ID || m["threadId"] == t2ID {
			hasT2 = true
		}
	}
	if !hasT1 || !hasT2 {
		t.Error("expected messages from both threads")
	}
}

// ===========================================================================
// Tests — Messages Bulk Delete (ported from messages-bulk-delete.ts)
// ===========================================================================

func TestInMemoryMemory_DeleteMultipleMessages(t *testing.T) {
	// TS: "should delete multiple messages successfully"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	messages := make([]MastraDBMessage, 5)
	for i := 0; i < 5; i++ {
		messages[i] = createSampleMessageV2(threadID,
			withContent(fmt.Sprintf("Message %d", i)),
			withID(fmt.Sprintf("msg-%d", i)),
		)
	}
	if _, err := storage.SaveMessages(ctx, messages); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	// Delete messages 1, 2, and 4.
	if err := storage.DeleteMessages(ctx, []string{"msg-1", "msg-2", "msg-4"}); err != nil {
		t.Fatalf("DeleteMessages returned error: %v", err)
	}

	// Verify only messages 0 and 3 remain.
	result, err := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: threadID})
	if err != nil {
		t.Fatalf("ListMessages returned error: %v", err)
	}
	if len(result.Messages) != 2 {
		t.Fatalf("expected 2 remaining messages, got %d", len(result.Messages))
	}
	ids := map[string]bool{}
	for _, m := range result.Messages {
		ids[m["id"].(string)] = true
	}
	if !ids["msg-0"] || !ids["msg-3"] {
		t.Errorf("expected msg-0 and msg-3 to remain, got %v", ids)
	}
}

func TestInMemoryMemory_DeleteEmptyArray(t *testing.T) {
	// TS: "should handle empty array gracefully"
	ctx := context.Background()
	storage := newStorage()

	if err := storage.DeleteMessages(ctx, []string{}); err != nil {
		t.Fatalf("DeleteMessages with empty array returned error: %v", err)
	}
}

func TestInMemoryMemory_DeleteNonExistentMessages(t *testing.T) {
	// TS: "should handle deleting non-existent messages"
	ctx := context.Background()
	storage := newStorage()

	if err := storage.DeleteMessages(ctx, []string{"non-existent-1", "non-existent-2"}); err != nil {
		t.Fatalf("DeleteMessages of non-existent messages returned error: %v", err)
	}
}

func TestInMemoryMemory_DeleteMessagesDifferentThreads(t *testing.T) {
	// TS: "should handle messages from different threads"
	ctx := context.Background()
	storage := newStorage()

	thread1 := createSampleThread()
	thread2 := createSampleThread()
	thread1["id"] = "bulk-thread-1"
	thread2["id"] = "bulk-thread-2"
	if _, err := storage.SaveThread(ctx, thread1); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	if _, err := storage.SaveThread(ctx, thread2); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	msgs1 := make([]MastraDBMessage, 2)
	for i := 0; i < 2; i++ {
		msgs1[i] = createSampleMessageV2("bulk-thread-1",
			withContent(fmt.Sprintf("Thread 1 Message %d", i)),
			withID(fmt.Sprintf("bulk-thread1-msg-%d", i)),
		)
	}
	msgs2 := make([]MastraDBMessage, 2)
	for i := 0; i < 2; i++ {
		msgs2[i] = createSampleMessageV2("bulk-thread-2",
			withContent(fmt.Sprintf("Thread 2 Message %d", i)),
			withID(fmt.Sprintf("bulk-thread2-msg-%d", i)),
		)
	}
	if _, err := storage.SaveMessages(ctx, msgs1); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}
	if _, err := storage.SaveMessages(ctx, msgs2); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	// Delete one message from each thread.
	if err := storage.DeleteMessages(ctx, []string{"bulk-thread1-msg-0", "bulk-thread2-msg-1"}); err != nil {
		t.Fatalf("DeleteMessages returned error: %v", err)
	}

	r1, _ := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: "bulk-thread-1"})
	if len(r1.Messages) != 1 {
		t.Fatalf("expected 1 message in thread1, got %d", len(r1.Messages))
	}
	if r1.Messages[0]["id"] != "bulk-thread1-msg-1" {
		t.Errorf("expected remaining msg id=bulk-thread1-msg-1, got %v", r1.Messages[0]["id"])
	}

	r2, _ := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: "bulk-thread-2"})
	if len(r2.Messages) != 1 {
		t.Fatalf("expected 1 message in thread2, got %d", len(r2.Messages))
	}
	if r2.Messages[0]["id"] != "bulk-thread2-msg-0" {
		t.Errorf("expected remaining msg id=bulk-thread2-msg-0, got %v", r2.Messages[0]["id"])
	}
}

func TestInMemoryMemory_DeleteMixedValidInvalidIDs(t *testing.T) {
	// TS: "should handle mixed valid and invalid message IDs"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	threadID := thread["id"].(string)

	messages := make([]MastraDBMessage, 3)
	for i := 0; i < 3; i++ {
		messages[i] = createSampleMessageV2(threadID,
			withContent(fmt.Sprintf("Message %d", i)),
			withID(fmt.Sprintf("mixed-msg-%d", i)),
		)
	}
	if _, err := storage.SaveMessages(ctx, messages); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	// Delete mix of valid and invalid IDs.
	if err := storage.DeleteMessages(ctx, []string{"mixed-msg-0", "invalid-id-1", "mixed-msg-2", "invalid-id-2"}); err != nil {
		t.Fatalf("DeleteMessages returned error: %v", err)
	}

	result, _ := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: threadID})
	if len(result.Messages) != 1 {
		t.Fatalf("expected 1 remaining message, got %d", len(result.Messages))
	}
	if result.Messages[0]["id"] != "mixed-msg-1" {
		t.Errorf("expected remaining msg id=mixed-msg-1, got %v", result.Messages[0]["id"])
	}
}

// ===========================================================================
// Tests — Messages Update (ported from messages-update.ts)
// ===========================================================================

func TestInMemoryMemory_UpdateMessageRole(t *testing.T) {
	// TS: "should update a single field of a message (e.g., role)"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	msg := createSampleMessageV2(threadID, withRole("user"))
	if _, err := storage.SaveMessages(ctx, []MastraDBMessage{msg}); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	msgID := msg["id"].(string)
	updated, err := storage.UpdateMessages(ctx, UpdateMessagesInput{
		Messages: []any{map[string]any{"id": msgID, "role": "assistant"}},
	})
	if err != nil {
		t.Fatalf("UpdateMessages returned error: %v", err)
	}
	if len(updated) != 1 {
		t.Fatalf("expected 1 updated message, got %d", len(updated))
	}
	if updated[0]["role"] != "assistant" {
		t.Errorf("expected role=assistant, got %v", updated[0]["role"])
	}

	// Verify persistence.
	result, _ := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: threadID})
	found := false
	for _, m := range result.Messages {
		if m["id"] == msgID {
			found = true
			if m["role"] != "assistant" {
				t.Errorf("persisted role: expected assistant, got %v", m["role"])
			}
		}
	}
	if !found {
		t.Error("updated message not found")
	}
}

func TestInMemoryMemory_UpdateMessageContent(t *testing.T) {
	// TS: "should update only the content string within the content field, preserving metadata"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	msg := createSampleMessageV2(threadID, withContent("old content"))
	// Add metadata to the content.
	content, _ := msg["content"].(map[string]any)
	content["metadata"] = map[string]any{"initial": true}
	msg["content"] = content
	if _, err := storage.SaveMessages(ctx, []MastraDBMessage{msg}); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	msgID := msg["id"].(string)
	_, err := storage.UpdateMessages(ctx, UpdateMessagesInput{
		Messages: []any{map[string]any{
			"id":      msgID,
			"content": map[string]any{"content": "new content"},
		}},
	})
	if err != nil {
		t.Fatalf("UpdateMessages returned error: %v", err)
	}

	result, _ := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: threadID})
	for _, m := range result.Messages {
		if m["id"] == msgID {
			c, _ := m["content"].(map[string]any)
			if c["content"] != "new content" {
				t.Errorf("expected content=new content, got %v", c["content"])
			}
			meta, _ := c["metadata"].(map[string]any)
			if meta == nil || meta["initial"] != true {
				t.Errorf("expected metadata.initial=true to be preserved, got %v", meta)
			}
		}
	}
}

func TestInMemoryMemory_UpdateMultipleMessages(t *testing.T) {
	// TS: "should update multiple messages at once"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	msg1 := createSampleMessageV2(threadID, withRole("user"), withContent("content1"))
	msg2 := createSampleMessageV2(threadID, withContent("original"))
	if _, err := storage.SaveMessages(ctx, []MastraDBMessage{msg1, msg2}); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	_, err := storage.UpdateMessages(ctx, UpdateMessagesInput{
		Messages: []any{
			map[string]any{"id": msg1["id"], "role": "assistant"},
			map[string]any{"id": msg2["id"], "content": map[string]any{"content": "updated"}},
		},
	})
	if err != nil {
		t.Fatalf("UpdateMessages returned error: %v", err)
	}

	result, _ := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: threadID})
	for _, m := range result.Messages {
		if m["id"] == msg1["id"] {
			if m["role"] != "assistant" {
				t.Errorf("msg1: expected role=assistant, got %v", m["role"])
			}
		}
		if m["id"] == msg2["id"] {
			if getContentString(m) != "updated" {
				t.Errorf("msg2: expected content=updated, got %s", getContentString(m))
			}
		}
	}
}

func TestInMemoryMemory_UpdateNonExistentMessage(t *testing.T) {
	// TS: "should not fail when trying to update a non-existent message"
	ctx := context.Background()
	storage := newStorage()

	_, err := storage.UpdateMessages(ctx, UpdateMessagesInput{
		Messages: []any{map[string]any{"id": uuid.New().String(), "role": "assistant"}},
	})
	if err != nil {
		t.Fatalf("UpdateMessages on non-existent message returned error: %v", err)
	}
}

// ===========================================================================
// Tests — ListMessagesByResourceID (ported from messages-list.ts)
// ===========================================================================

func TestInMemoryMemory_ListMessagesByResourceID_AcrossThreads(t *testing.T) {
	// TS: "should list all messages for a resource across multiple threads"
	ctx := context.Background()
	storage := newStorage()

	thread1 := createSampleThread()
	thread2 := createSampleThread()
	// Thread2 has same resource as thread1.
	thread2["resourceId"] = thread1["resourceId"]

	if _, err := storage.SaveThread(ctx, thread1); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}
	if _, err := storage.SaveThread(ctx, thread2); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	t1ID := thread1["id"].(string)
	t2ID := thread2["id"].(string)
	resID := thread1["resourceId"].(string)

	now := time.Now()
	msgs := []MastraDBMessage{
		createSampleMessageV2(t1ID, withContent("T1 Msg 1"), withResourceID(resID), withCreatedAt(now.Add(1*time.Second))),
		createSampleMessageV2(t1ID, withContent("T1 Msg 2"), withResourceID(resID), withCreatedAt(now.Add(2*time.Second))),
		createSampleMessageV2(t2ID, withContent("T2 Msg 1"), withResourceID(resID), withCreatedAt(now.Add(3*time.Second))),
	}
	if _, err := storage.SaveMessages(ctx, msgs); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	result, err := storage.ListMessagesByResourceID(ctx, StorageListMessagesByResourceIDInput{
		ResourceID: resID,
	})
	if err != nil {
		t.Fatalf("ListMessagesByResourceID returned error: %v", err)
	}
	if len(result.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result.Messages))
	}
	for _, m := range result.Messages {
		if m["resourceId"] != resID {
			t.Errorf("expected resourceId=%s, got %v", resID, m["resourceId"])
		}
	}
}

func TestInMemoryMemory_ListMessagesByResourceID_Empty(t *testing.T) {
	// TS: "should return empty array when no messages match resourceId"
	ctx := context.Background()
	storage := newStorage()

	result, err := storage.ListMessagesByResourceID(ctx, StorageListMessagesByResourceIDInput{
		ResourceID: "non-existent-resource",
	})
	if err != nil {
		t.Fatalf("ListMessagesByResourceID returned error: %v", err)
	}
	if len(result.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(result.Messages))
	}
	if result.Total != 0 {
		t.Errorf("expected total=0, got %d", result.Total)
	}
}

func TestInMemoryMemory_ListMessagesByResourceID_Pagination(t *testing.T) {
	// TS: "should support pagination when querying by resourceId"
	ctx := context.Background()
	storage := newStorage()

	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	resID := thread["resourceId"].(string)
	now := time.Now()
	msgs := make([]MastraDBMessage, 5)
	for i := 0; i < 5; i++ {
		msgs[i] = createSampleMessageV2(threadID,
			withContent(fmt.Sprintf("Msg %d", i+1)),
			withResourceID(resID),
			withCreatedAt(now.Add(time.Duration(i+1)*time.Second)),
		)
	}
	if _, err := storage.SaveMessages(ctx, msgs); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	perPage := 2
	result, err := storage.ListMessagesByResourceID(ctx, StorageListMessagesByResourceIDInput{
		ResourceID: resID,
		PerPage:    &perPage,
		Page:       0,
	})
	if err != nil {
		t.Fatalf("ListMessagesByResourceID returned error: %v", err)
	}
	if len(result.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result.Messages))
	}
	if result.Total != 5 {
		t.Errorf("expected total=5, got %d", result.Total)
	}
	if !result.HasMore {
		t.Error("expected hasMore=true")
	}
	if result.Page != 0 {
		t.Errorf("expected page=0, got %d", result.Page)
	}
	if result.PerPage != 2 {
		t.Errorf("expected perPage=2, got %d", result.PerPage)
	}
}

// ===========================================================================
// Tests — Resources (ported from resources.ts)
// ===========================================================================

func TestInMemoryMemory_CreateAndRetrieveResource(t *testing.T) {
	// TS: "should create and retrieve a resource"
	ctx := context.Background()
	storage := newStorage()

	resource := createSampleResource()
	saved, err := storage.SaveResource(ctx, resource)
	if err != nil {
		t.Fatalf("SaveResource returned error: %v", err)
	}
	if saved["id"] != resource["id"] {
		t.Errorf("expected saved id=%v, got %v", resource["id"], saved["id"])
	}
	if saved["workingMemory"] != resource["workingMemory"] {
		t.Errorf("expected workingMemory=%v, got %v", resource["workingMemory"], saved["workingMemory"])
	}

	retrieved, err := storage.GetResourceByID(ctx, resource["id"].(string))
	if err != nil {
		t.Fatalf("GetResourceByID returned error: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected non-nil resource")
	}
	if retrieved["id"] != resource["id"] {
		t.Errorf("expected id=%v, got %v", resource["id"], retrieved["id"])
	}
	if retrieved["workingMemory"] != resource["workingMemory"] {
		t.Errorf("expected workingMemory=%v, got %v", resource["workingMemory"], retrieved["workingMemory"])
	}
}

func TestInMemoryMemory_NonExistentResource(t *testing.T) {
	// TS: "should return null for non-existent resource"
	ctx := context.Background()
	storage := newStorage()

	result, err := storage.GetResourceByID(ctx, "non-existent")
	if err != nil {
		t.Fatalf("GetResourceByID returned error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for non-existent resource, got %+v", result)
	}
}

func TestInMemoryMemory_UpdateResourceWorkingMemoryAndMetadata(t *testing.T) {
	// TS: "should update resource workingMemory and metadata"
	ctx := context.Background()
	storage := newStorage()

	resource := createSampleResource()
	if _, err := storage.SaveResource(ctx, resource); err != nil {
		t.Fatalf("SaveResource returned error: %v", err)
	}

	newWM := "Updated working memory content"
	newMeta := map[string]any{"newKey": "newValue", "updated": true}
	updated, err := storage.UpdateResource(ctx, UpdateResourceInput{
		ResourceID:    resource["id"].(string),
		WorkingMemory: &newWM,
		Metadata:      newMeta,
	})
	if err != nil {
		t.Fatalf("UpdateResource returned error: %v", err)
	}
	if updated["workingMemory"] != newWM {
		t.Errorf("expected workingMemory=%s, got %v", newWM, updated["workingMemory"])
	}

	// Metadata should be merged.
	meta, _ := updated["metadata"].(map[string]any)
	if meta["key"] != "value" {
		t.Error("expected original key=value in metadata")
	}
	if meta["newKey"] != "newValue" {
		t.Error("expected newKey=newValue in metadata")
	}

	// Verify persistence.
	retrieved, _ := storage.GetResourceByID(ctx, resource["id"].(string))
	if retrieved["workingMemory"] != newWM {
		t.Errorf("persisted workingMemory mismatch: got %v", retrieved["workingMemory"])
	}
}

func TestInMemoryMemory_UpdateOnlyWorkingMemory(t *testing.T) {
	// TS: "should update only workingMemory when metadata is not provided"
	ctx := context.Background()
	storage := newStorage()

	resource := createSampleResource()
	if _, err := storage.SaveResource(ctx, resource); err != nil {
		t.Fatalf("SaveResource returned error: %v", err)
	}

	newWM := "Updated working memory only"
	updated, err := storage.UpdateResource(ctx, UpdateResourceInput{
		ResourceID:    resource["id"].(string),
		WorkingMemory: &newWM,
	})
	if err != nil {
		t.Fatalf("UpdateResource returned error: %v", err)
	}
	if updated["workingMemory"] != newWM {
		t.Errorf("expected workingMemory=%s, got %v", newWM, updated["workingMemory"])
	}

	// Metadata should be unchanged.
	meta, _ := updated["metadata"].(map[string]any)
	if meta["key"] != "value" {
		t.Error("expected original metadata to be preserved")
	}
}

func TestInMemoryMemory_UpdateOnlyMetadata(t *testing.T) {
	// TS: "should update only metadata when workingMemory is not provided"
	ctx := context.Background()
	storage := newStorage()

	resource := createSampleResource()
	if _, err := storage.SaveResource(ctx, resource); err != nil {
		t.Fatalf("SaveResource returned error: %v", err)
	}

	newMeta := map[string]any{"onlyMetadata": "updated"}
	updated, err := storage.UpdateResource(ctx, UpdateResourceInput{
		ResourceID: resource["id"].(string),
		Metadata:   newMeta,
	})
	if err != nil {
		t.Fatalf("UpdateResource returned error: %v", err)
	}

	if updated["workingMemory"] != resource["workingMemory"] {
		t.Errorf("expected workingMemory to be unchanged")
	}
	meta, _ := updated["metadata"].(map[string]any)
	if meta["onlyMetadata"] != "updated" {
		t.Error("expected onlyMetadata=updated in metadata")
	}
	// Original key should be preserved (merged).
	if meta["key"] != "value" {
		t.Error("expected original key=value to be preserved")
	}
}

func TestInMemoryMemory_UpdateNonExistentResource(t *testing.T) {
	// TS: "should create new resource when updating non-existent resource"
	ctx := context.Background()
	storage := newStorage()

	nonExistentID := "resource-" + uuid.New().String()
	newWM := "New working memory"
	newMeta := map[string]any{"created": true, "source": "update"}
	created, err := storage.UpdateResource(ctx, UpdateResourceInput{
		ResourceID:    nonExistentID,
		WorkingMemory: &newWM,
		Metadata:      newMeta,
	})
	if err != nil {
		t.Fatalf("UpdateResource returned error: %v", err)
	}
	if created["id"] != nonExistentID {
		t.Errorf("expected id=%s, got %v", nonExistentID, created["id"])
	}
	if created["workingMemory"] != newWM {
		t.Errorf("expected workingMemory=%s, got %v", newWM, created["workingMemory"])
	}

	// Verify it was actually created.
	retrieved, _ := storage.GetResourceByID(ctx, nonExistentID)
	if retrieved == nil {
		t.Fatal("expected resource to be created")
	}
	if retrieved["workingMemory"] != newWM {
		t.Errorf("expected persisted workingMemory=%s, got %v", newWM, retrieved["workingMemory"])
	}
}

func TestInMemoryMemory_EmptyWorkingMemory(t *testing.T) {
	// TS: "should handle empty workingMemory"
	ctx := context.Background()
	storage := newStorage()

	resource := createSampleResource(func(r map[string]any) {
		r["workingMemory"] = ""
	})
	saved, err := storage.SaveResource(ctx, resource)
	if err != nil {
		t.Fatalf("SaveResource returned error: %v", err)
	}
	if saved["workingMemory"] != "" {
		t.Errorf("expected empty workingMemory, got %v", saved["workingMemory"])
	}

	retrieved, _ := storage.GetResourceByID(ctx, resource["id"].(string))
	if retrieved["workingMemory"] != "" {
		t.Errorf("expected empty workingMemory in retrieval, got %v", retrieved["workingMemory"])
	}
}

func TestInMemoryMemory_ComplexMetadata(t *testing.T) {
	// TS: "should handle complex metadata structures"
	ctx := context.Background()
	storage := newStorage()

	complexMeta := map[string]any{
		"nested": map[string]any{
			"object": map[string]any{
				"with": map[string]any{
					"arrays": []any{1, 2, 3},
				},
			},
		},
		"mixed": map[string]any{
			"string":  "test",
			"number":  123,
			"boolean": false,
			"null":    nil,
		},
	}

	resource := createSampleResource(func(r map[string]any) {
		r["metadata"] = complexMeta
	})
	if _, err := storage.SaveResource(ctx, resource); err != nil {
		t.Fatalf("SaveResource returned error: %v", err)
	}

	retrieved, _ := storage.GetResourceByID(ctx, resource["id"].(string))
	if retrieved == nil {
		t.Fatal("expected non-nil resource")
	}

	// Verify nested structure is preserved by marshaling to JSON.
	savedJSON, _ := json.Marshal(retrieved["metadata"])
	expectedJSON, _ := json.Marshal(complexMeta)
	if string(savedJSON) != string(expectedJSON) {
		t.Errorf("metadata mismatch:\n  got:  %s\n  want: %s", savedJSON, expectedJSON)
	}
}

func TestInMemoryMemory_LargeWorkingMemory(t *testing.T) {
	// TS: "should handle large workingMemory content"
	ctx := context.Background()
	storage := newStorage()

	largeWM := strings.Repeat("A", 10000)
	resource := createSampleResource(func(r map[string]any) {
		r["workingMemory"] = largeWM
	})
	if _, err := storage.SaveResource(ctx, resource); err != nil {
		t.Fatalf("SaveResource returned error: %v", err)
	}

	retrieved, _ := storage.GetResourceByID(ctx, resource["id"].(string))
	if retrieved["workingMemory"] != largeWM {
		t.Error("expected large workingMemory to be preserved")
	}
}

// ===========================================================================
// Tests — Observational Memory (basic coverage)
// ===========================================================================

func TestInMemoryMemory_InitializeAndGetObservationalMemory(t *testing.T) {
	ctx := context.Background()
	storage := newStorage()

	record, err := storage.InitializeObservationalMemory(ctx, CreateObservationalMemoryInput{
		ThreadID:   "thread-1",
		ResourceID: "resource-1",
		Scope:      ObservationalMemoryScopeThread,
		Config:     map[string]any{"maxTokens": 1000},
	})
	if err != nil {
		t.Fatalf("InitializeObservationalMemory returned error: %v", err)
	}
	if record == nil {
		t.Fatal("expected non-nil record")
	}
	if record.ThreadID != "thread-1" {
		t.Errorf("expected threadId=thread-1, got %s", record.ThreadID)
	}
	if record.ResourceID != "resource-1" {
		t.Errorf("expected resourceId=resource-1, got %s", record.ResourceID)
	}
	if record.Scope != ObservationalMemoryScopeThread {
		t.Errorf("expected scope=thread, got %s", record.Scope)
	}
	if record.OriginType != ObservationalMemoryOriginInitial {
		t.Errorf("expected originType=initial, got %s", record.OriginType)
	}

	// Retrieve it.
	got, err := storage.GetObservationalMemory(ctx, "thread-1", "resource-1")
	if err != nil {
		t.Fatalf("GetObservationalMemory returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil record from GetObservationalMemory")
	}
	if got.ID != record.ID {
		t.Errorf("expected id=%s, got %s", record.ID, got.ID)
	}
}

func TestInMemoryMemory_ObservationalMemoryFlags(t *testing.T) {
	ctx := context.Background()
	storage := newStorage()

	record, err := storage.InitializeObservationalMemory(ctx, CreateObservationalMemoryInput{
		ThreadID:   "thread-flags",
		ResourceID: "resource-flags",
		Scope:      ObservationalMemoryScopeThread,
		Config:     map[string]any{},
	})
	if err != nil {
		t.Fatalf("InitializeObservationalMemory returned error: %v", err)
	}

	// Test SetObservingFlag.
	if err := storage.SetObservingFlag(ctx, record.ID, true); err != nil {
		t.Fatalf("SetObservingFlag returned error: %v", err)
	}
	got, _ := storage.GetObservationalMemory(ctx, "thread-flags", "resource-flags")
	if !got.IsObserving {
		t.Error("expected isObserving=true")
	}

	// Test SetReflectingFlag.
	if err := storage.SetReflectingFlag(ctx, record.ID, true); err != nil {
		t.Fatalf("SetReflectingFlag returned error: %v", err)
	}
	got, _ = storage.GetObservationalMemory(ctx, "thread-flags", "resource-flags")
	if !got.IsReflecting {
		t.Error("expected isReflecting=true")
	}

	// Test SetBufferingObservationFlag.
	tokens := 500
	if err := storage.SetBufferingObservationFlag(ctx, record.ID, true, &tokens); err != nil {
		t.Fatalf("SetBufferingObservationFlag returned error: %v", err)
	}
	got, _ = storage.GetObservationalMemory(ctx, "thread-flags", "resource-flags")
	if !got.IsBufferingObservation {
		t.Error("expected isBufferingObservation=true")
	}
	if got.LastBufferedAtTokens != 500 {
		t.Errorf("expected lastBufferedAtTokens=500, got %d", got.LastBufferedAtTokens)
	}

	// Test SetBufferingReflectionFlag.
	if err := storage.SetBufferingReflectionFlag(ctx, record.ID, true); err != nil {
		t.Fatalf("SetBufferingReflectionFlag returned error: %v", err)
	}
	got, _ = storage.GetObservationalMemory(ctx, "thread-flags", "resource-flags")
	if !got.IsBufferingReflection {
		t.Error("expected isBufferingReflection=true")
	}
}

func TestInMemoryMemory_UpdateActiveObservations(t *testing.T) {
	ctx := context.Background()
	storage := newStorage()

	record, err := storage.InitializeObservationalMemory(ctx, CreateObservationalMemoryInput{
		ThreadID:   "thread-obs",
		ResourceID: "resource-obs",
		Scope:      ObservationalMemoryScopeThread,
		Config:     map[string]any{},
	})
	if err != nil {
		t.Fatalf("InitializeObservationalMemory returned error: %v", err)
	}

	now := time.Now()
	if err := storage.UpdateActiveObservations(ctx, UpdateActiveObservationsInput{
		ID:                 record.ID,
		Observations:       "User prefers concise responses",
		TokenCount:         50,
		LastObservedAt:     now,
		ObservedMessageIDs: []string{"msg-1", "msg-2"},
	}); err != nil {
		t.Fatalf("UpdateActiveObservations returned error: %v", err)
	}

	got, _ := storage.GetObservationalMemory(ctx, "thread-obs", "resource-obs")
	if got.ActiveObservations != "User prefers concise responses" {
		t.Errorf("expected activeObservations to be updated, got %s", got.ActiveObservations)
	}
	if got.ObservationTokenCount != 50 {
		t.Errorf("expected observationTokenCount=50, got %d", got.ObservationTokenCount)
	}
	if len(got.ObservedMessageIDs) != 2 {
		t.Errorf("expected 2 observedMessageIds, got %d", len(got.ObservedMessageIDs))
	}
}

func TestInMemoryMemory_ClearObservationalMemory(t *testing.T) {
	ctx := context.Background()
	storage := newStorage()

	_, err := storage.InitializeObservationalMemory(ctx, CreateObservationalMemoryInput{
		ThreadID:   "thread-clear",
		ResourceID: "resource-clear",
		Scope:      ObservationalMemoryScopeThread,
		Config:     map[string]any{},
	})
	if err != nil {
		t.Fatalf("InitializeObservationalMemory returned error: %v", err)
	}

	// Verify it exists.
	got, _ := storage.GetObservationalMemory(ctx, "thread-clear", "resource-clear")
	if got == nil {
		t.Fatal("expected observational memory to exist before clearing")
	}

	// Clear it.
	if err := storage.ClearObservationalMemory(ctx, "thread-clear", "resource-clear"); err != nil {
		t.Fatalf("ClearObservationalMemory returned error: %v", err)
	}

	// Verify it's gone.
	got, _ = storage.GetObservationalMemory(ctx, "thread-clear", "resource-clear")
	if got != nil {
		t.Error("expected observational memory to be nil after clearing")
	}
}

func TestInMemoryMemory_SetPendingMessageTokens(t *testing.T) {
	ctx := context.Background()
	storage := newStorage()

	record, err := storage.InitializeObservationalMemory(ctx, CreateObservationalMemoryInput{
		ThreadID:   "thread-pending",
		ResourceID: "resource-pending",
		Scope:      ObservationalMemoryScopeThread,
		Config:     map[string]any{},
	})
	if err != nil {
		t.Fatalf("InitializeObservationalMemory returned error: %v", err)
	}

	if err := storage.SetPendingMessageTokens(ctx, record.ID, 1234); err != nil {
		t.Fatalf("SetPendingMessageTokens returned error: %v", err)
	}

	got, _ := storage.GetObservationalMemory(ctx, "thread-pending", "resource-pending")
	if got.PendingMessageTokens != 1234 {
		t.Errorf("expected pendingMessageTokens=1234, got %d", got.PendingMessageTokens)
	}
}

// ===========================================================================
// Tests — Thread Cloning (ported from threads.ts conceptual)
// ===========================================================================

func TestInMemoryMemory_CloneThread(t *testing.T) {
	ctx := context.Background()
	storage := newStorage()

	// Create source thread with messages.
	thread := createSampleThread()
	if _, err := storage.SaveThread(ctx, thread); err != nil {
		t.Fatalf("SaveThread returned error: %v", err)
	}

	threadID := thread["id"].(string)
	now := time.Now()
	msgs := []MastraDBMessage{
		createSampleMessageV2(threadID, withContent("Msg 1"), withCreatedAt(now.Add(1*time.Second))),
		createSampleMessageV2(threadID, withContent("Msg 2"), withCreatedAt(now.Add(2*time.Second))),
		createSampleMessageV2(threadID, withContent("Msg 3"), withCreatedAt(now.Add(3*time.Second))),
	}
	if _, err := storage.SaveMessages(ctx, msgs); err != nil {
		t.Fatalf("SaveMessages returned error: %v", err)
	}

	// Clone it.
	cloneResult, err := storage.CloneThread(ctx, StorageCloneThreadInput{
		SourceThreadID: threadID,
		Title:          "Cloned Thread",
	})
	if err != nil {
		t.Fatalf("CloneThread returned error: %v", err)
	}

	// Verify cloned thread.
	if cloneResult.Thread == nil {
		t.Fatal("expected non-nil cloned thread")
	}
	clonedID, _ := cloneResult.Thread["id"].(string)
	if clonedID == "" {
		t.Fatal("expected non-empty cloned thread ID")
	}
	if clonedID == threadID {
		t.Error("expected cloned thread to have a different ID")
	}
	if cloneResult.Thread["title"] != "Cloned Thread" {
		t.Errorf("expected cloned title=Cloned Thread, got %v", cloneResult.Thread["title"])
	}

	// Verify cloned messages.
	if len(cloneResult.ClonedMessages) != 3 {
		t.Fatalf("expected 3 cloned messages, got %d", len(cloneResult.ClonedMessages))
	}

	// Verify cloned messages belong to the new thread.
	for _, cm := range cloneResult.ClonedMessages {
		cmThreadID, _ := cm["thread_id"].(string)
		if cmThreadID == "" {
			cmThreadID, _ = cm["threadId"].(string)
		}
		if cmThreadID != clonedID {
			t.Errorf("expected cloned message thread_id=%s, got %s", clonedID, cmThreadID)
		}
	}

	// Verify original thread is untouched.
	origResult, _ := storage.ListMessages(ctx, StorageListMessagesInput{ThreadID: threadID})
	if len(origResult.Messages) != 3 {
		t.Errorf("expected original thread to still have 3 messages, got %d", len(origResult.Messages))
	}
}

// Ensure unused imports are consumed.
var _ = json.Marshal
var _ = strings.Repeat
