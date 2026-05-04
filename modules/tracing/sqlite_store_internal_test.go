package tracing

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSQLiteTraceStoreCloseContextRetriesWithSinglePendingCleanupWait(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewSQLiteTraceStore(db)
	if err != nil {
		t.Fatal(err)
	}
	store.cleanWG.Add(1)
	store.cleanupRunning.Store(true)

	for i := 0; i < 2; i++ {
		closeCtx, cancelClose := context.WithTimeout(context.Background(), 10*time.Millisecond)
		err := store.CloseContext(closeCtx)
		cancelClose()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("CloseContext attempt %d error = %v, want context deadline exceeded", i+1, err)
		}
		snapshot := store.debugSnapshot()
		if snapshot.Closed || !snapshot.CleanupRunning {
			t.Fatalf("snapshot after close attempt %d = %#v, want open with cleanup running", i+1, snapshot)
		}
	}

	store.cleanupRunning.Store(false)
	store.cleanWG.Done()
	if err := store.CloseContext(context.Background()); err != nil {
		t.Fatalf("retry CloseContext: %v", err)
	}
	snapshot := store.debugSnapshot()
	if !snapshot.Closed || snapshot.CleanupRunning {
		t.Fatalf("snapshot after retry = %#v, want closed with cleanup stopped", snapshot)
	}
}
