package tracing

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	_ "modernc.org/sqlite"
)

func TestTracingDebugSnapshotReportsOwnedStoreState(t *testing.T) {
	store := &debugTraceStore{}
	module := &Module{
		cfg:   Config{Store: store},
		store: store,
	}
	module.traceStoreLeaseActive.Store(true)

	got := module.DebugSnapshot()
	if got.Closing {
		t.Fatalf("closing = true, want false")
	}
	if !got.StoreConfigured || !got.StoreAttached {
		t.Fatalf("store flags = configured %v attached %v, want both true", got.StoreConfigured, got.StoreAttached)
	}
	if !got.TraceStoreLeaseAttached {
		t.Fatalf("trace store lease attached = false, want true")
	}
	if !got.StoreCloseable {
		t.Fatalf("store closeable = false, want true")
	}
	if !strings.Contains(got.StoreType, "debugTraceStore") {
		t.Fatalf("store type = %q, want debugTraceStore", got.StoreType)
	}
	if got.SQLite != nil {
		t.Fatalf("sqlite snapshot = %#v, want nil for custom store", got.SQLite)
	}

	if err := module.Close(); err != nil {
		t.Fatalf("close module: %v", err)
	}
	if !store.closed {
		t.Fatalf("store was not closed")
	}
	got = module.DebugSnapshot()
	if !got.StoreConfigured || got.StoreAttached {
		t.Fatalf("closed store flags = configured %v attached %v, want true and false", got.StoreConfigured, got.StoreAttached)
	}
	if got.TraceStoreLeaseAttached {
		t.Fatalf("closed trace store lease attached = true, want false")
	}
}

func TestTracingDebugSnapshotReportsSQLiteStoreState(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store, err := NewSQLiteTraceStore(db, WithRetention(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	module := &Module{
		cfg:   Config{Store: store},
		store: store,
	}
	got := module.DebugSnapshot()
	if got.SQLite == nil {
		t.Fatalf("sqlite snapshot = nil, want populated")
	}
	if !got.SQLite.RetentionEnabled || !got.SQLite.CleanupEnabled {
		t.Fatalf("sqlite retention flags = retention %v cleanup %v, want both true", got.SQLite.RetentionEnabled, got.SQLite.CleanupEnabled)
	}
	if !got.SQLite.CleanupRunning || got.SQLite.Closing || got.SQLite.Closed {
		t.Fatalf("sqlite cleanup state = %#v, want running and open", got.SQLite)
	}
	if got.SQLite.RetentionSeconds != int64((2 * time.Hour).Seconds()) {
		t.Fatalf("retention seconds = %d, want %d", got.SQLite.RetentionSeconds, int64((2 * time.Hour).Seconds()))
	}
	if err := store.CloseContext(context.Background()); err != nil {
		t.Fatalf("close sqlite trace store: %v", err)
	}
	got = module.DebugSnapshot()
	if got.SQLite == nil || got.SQLite.CleanupRunning || !got.SQLite.Closed {
		t.Fatalf("sqlite cleanup state after close = %#v, want stopped and closed", got.SQLite)
	}
}

func TestTracingCloseContextKeepsStoreAttachedWhenLeaseCloseFails(t *testing.T) {
	leaseErr := errors.New("detach trace store")
	ctx := context.WithValue(context.Background(), traceCloseContextKey{}, "close")
	store := &debugTraceStore{}
	handle := &traceCloseHandle{err: leaseErr}
	module := &Module{
		cfg:             Config{Store: store},
		store:           store,
		traceStoreLease: handle,
	}
	module.traceStoreLeaseActive.Store(true)

	err := module.CloseContext(ctx)
	if !errors.Is(err, leaseErr) {
		t.Fatalf("CloseContext error = %v, want lease error %v", err, leaseErr)
	}
	if handle.ctx != ctx {
		t.Fatalf("lease closed with context %v, want %v", handle.ctx, ctx)
	}
	if module.traceStoreLease == nil {
		t.Fatal("trace store lease was cleared after close failure")
	}
	if !module.traceStoreLeaseActive.Load() {
		t.Fatal("trace store lease no longer marked active after close failure")
	}
	if module.store != nil {
		if module.store != store {
			t.Fatal("trace store changed after lease close failure")
		}
	} else {
		t.Fatal("trace store was cleared after lease close failure")
	}
	if store.closed {
		t.Fatal("trace store was closed even though lease close failed")
	}
}

func TestTracingCloseContextKeepsStoreWhenStoreCloseFails(t *testing.T) {
	storeErr := errors.New("close trace store")
	ctx := context.WithValue(context.Background(), traceCloseContextKey{}, "close")
	store := &debugTraceStore{closeErr: storeErr}
	handle := &traceCloseHandle{}
	module := &Module{
		cfg:             Config{Store: store},
		store:           store,
		traceStoreLease: handle,
	}
	module.traceStoreLeaseActive.Store(true)

	err := module.CloseContext(ctx)
	if !errors.Is(err, storeErr) {
		t.Fatalf("CloseContext error = %v, want store error %v", err, storeErr)
	}
	if store.ctx != ctx {
		t.Fatalf("trace store closed with context %v, want %v", store.ctx, ctx)
	}
	if handle.ctx != ctx {
		t.Fatalf("lease closed with context %v, want %v", handle.ctx, ctx)
	}
	if module.traceStoreLease != nil {
		t.Fatal("trace store lease was not cleared after successful lease close")
	}
	if module.traceStoreLeaseActive.Load() {
		t.Fatal("trace store lease still marked active")
	}
	if module.store != store {
		t.Fatalf("trace store = %#v, want retained failed store", module.store)
	}
	if !store.closed {
		t.Fatal("trace store close was not attempted")
	}
}

func TestTracingDebugSnapshotReportsClosingDuringSlowStoreClose(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	store := &debugTraceStore{closeStarted: started, closeRelease: release}
	module := &Module{
		cfg:   Config{Store: store},
		store: store,
	}

	done := make(chan error, 1)
	go func() {
		done <- module.CloseContext(context.Background())
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for trace store close to start")
	}

	snapshotDone := make(chan DebugSnapshot, 1)
	go func() {
		snapshotDone <- module.DebugSnapshot()
	}()
	select {
	case snapshot := <-snapshotDone:
		if !snapshot.Closing || !snapshot.StoreAttached || !snapshot.StoreConfigured {
			t.Fatalf("snapshot during close = %#v, want closing attached configured store", snapshot)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DebugSnapshot blocked while trace store Close was in progress")
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("CloseContext: %v", err)
	}
	snapshot := module.DebugSnapshot()
	if snapshot.Closing || snapshot.StoreAttached {
		t.Fatalf("snapshot after close = %#v, want detached store", snapshot)
	}
}

func TestTracingCloseContextReturnsDeadlineWhenStoreIgnoresContext(t *testing.T) {
	release := make(chan struct{})
	store := &debugTraceStore{closeRelease: release}
	module := &Module{
		cfg:   Config{Store: store},
		store: store,
	}
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	err := module.CloseContext(ctx)
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v, want context deadline exceeded", err)
	}
	snapshot := module.DebugSnapshot()
	if !snapshot.StoreClosing || !snapshot.StoreAttached || !snapshot.StoreConfigured {
		t.Fatalf("snapshot after deadline = %#v, want retained closing trace store", snapshot)
	}

	close(release)
	requireEventuallyTracingSnapshot(t, module, func(snapshot DebugSnapshot) bool {
		return !snapshot.StoreClosing && !snapshot.StoreAttached
	}, "trace store close job to finish")
}

func requireEventuallyTracingSnapshot(t *testing.T, module *Module, fn func(DebugSnapshot) bool, label string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if fn(module.DebugSnapshot()) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s; snapshot=%#v", label, module.DebugSnapshot())
}

type debugTraceStore struct {
	closed       bool
	ctx          context.Context
	closeErr     error
	closeStarted chan struct{}
	closeRelease chan struct{}
	closeOnce    sync.Once
}

func (s *debugTraceStore) RecordSpan(Span) error { return nil }

func (s *debugTraceStore) GetTrace(string) ([]Span, error) { return nil, nil }

func (s *debugTraceStore) ListTraces(TraceQuery) ([]TraceSummary, error) { return nil, nil }

func (s *debugTraceStore) Close() error {
	return s.CloseContext(context.Background())
}

func (s *debugTraceStore) CloseContext(ctx context.Context) error {
	s.ctx = ctx
	if s.closeStarted != nil {
		s.closeOnce.Do(func() { close(s.closeStarted) })
	}
	if s.closeRelease != nil {
		<-s.closeRelease
	}
	s.closed = true
	return s.closeErr
}

type traceCloseContextKey struct{}

type traceCloseHandle struct {
	ctx context.Context
	err error
}

var _ bkmodule.Handle = (*traceCloseHandle)(nil)

func (h *traceCloseHandle) Close(ctx context.Context) error {
	h.ctx = ctx
	return h.err
}
