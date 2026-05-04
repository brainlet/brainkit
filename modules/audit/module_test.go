package audit

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	bkmodule "github.com/brainlet/brainkit/module"
)

func TestAuditCloseContextDoesNotCloseOwnedStoreWhenStoreLeaseFails(t *testing.T) {
	leaseErr := errors.New("detach audit store")
	ctx := context.WithValue(context.Background(), auditCloseContextKey{}, "close")
	store := &debugAuditStore{}
	handle := &auditCloseHandle{err: leaseErr}
	module := &Module{
		cfg:        Config{Store: store, OwnStore: true},
		storeLease: handle,
	}
	module.storeAttached.Store(true)

	err := module.CloseContext(ctx)
	if !errors.Is(err, leaseErr) {
		t.Fatalf("CloseContext error = %v, want lease error %v", err, leaseErr)
	}
	if handle.ctx != ctx {
		t.Fatalf("store lease closed with context %v, want %v", handle.ctx, ctx)
	}
	if module.storeLease == nil {
		t.Fatal("store lease was cleared after close failure")
	}
	if !module.storeAttached.Load() {
		t.Fatal("store lease no longer marked active after close failure")
	}
	if module.cfg.Store != store || !module.cfg.OwnStore {
		t.Fatalf("store config = (%#v, owns=%v), want retained owned store", module.cfg.Store, module.cfg.OwnStore)
	}
	if store.closed {
		t.Fatal("owned store was closed even though store lease close failed")
	}
}

func TestAuditCloseContextKeepsOwnedStoreWhenStoreCloseFails(t *testing.T) {
	storeErr := errors.New("close audit store")
	ctx := context.WithValue(context.Background(), auditCloseContextKey{}, "close")
	store := &debugAuditStore{closeErr: storeErr}
	handle := &auditCloseHandle{}
	module := &Module{
		cfg:        Config{Store: store, OwnStore: true},
		storeLease: handle,
	}
	module.storeAttached.Store(true)

	err := module.CloseContext(ctx)
	if !errors.Is(err, storeErr) {
		t.Fatalf("CloseContext error = %v, want store error %v", err, storeErr)
	}
	if store.ctx != ctx {
		t.Fatalf("owned store closed with context %v, want %v", store.ctx, ctx)
	}
	if handle.ctx != ctx {
		t.Fatalf("store lease closed with context %v, want %v", handle.ctx, ctx)
	}
	if module.storeLease != nil {
		t.Fatal("store lease was not cleared after successful lease close")
	}
	if module.storeAttached.Load() {
		t.Fatal("store lease still marked active")
	}
	if module.cfg.Store != store || !module.cfg.OwnStore {
		t.Fatalf("store config = (%#v, owns=%v), want retained failed owned store", module.cfg.Store, module.cfg.OwnStore)
	}
	if !store.closed {
		t.Fatal("owned store close was not attempted")
	}
}

func TestAuditDebugSnapshotReportsClosingDuringSlowStoreClose(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	store := &debugAuditStore{closeStarted: started, closeRelease: release}
	module := &Module{
		cfg: Config{Store: store, OwnStore: true},
	}

	done := make(chan error, 1)
	go func() {
		done <- module.CloseContext(context.Background())
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for audit store close to start")
	}

	snapshotDone := make(chan DebugSnapshot, 1)
	go func() {
		snapshotDone <- module.DebugSnapshot()
	}()
	select {
	case snapshot := <-snapshotDone:
		if !snapshot.Closing || !snapshot.StoreConfigured || !snapshot.OwnStore {
			t.Fatalf("snapshot during close = %#v, want closing owned store", snapshot)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DebugSnapshot blocked while audit store Close was in progress")
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("CloseContext: %v", err)
	}
	snapshot := module.DebugSnapshot()
	if snapshot.Closing || snapshot.StoreConfigured || snapshot.OwnStore {
		t.Fatalf("snapshot after close = %#v, want detached owned store", snapshot)
	}
}

func TestAuditCloseContextReturnsDeadlineWhenOwnedStoreIgnoresContext(t *testing.T) {
	release := make(chan struct{})
	store := &debugAuditStore{closeRelease: release}
	module := &Module{cfg: Config{Store: store, OwnStore: true}}
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
	if !snapshot.StoreClosing || !snapshot.StoreConfigured || !snapshot.OwnStore {
		t.Fatalf("snapshot after deadline = %#v, want retained closing owned store", snapshot)
	}

	close(release)
	requireEventuallyAuditSnapshot(t, module, func(snapshot DebugSnapshot) bool {
		return !snapshot.StoreClosing && !snapshot.StoreConfigured && !snapshot.OwnStore
	}, "owned store close job to finish")
}

func requireEventuallyAuditSnapshot(t *testing.T, module *Module, fn func(DebugSnapshot) bool, label string) {
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

type debugAuditStore struct {
	closed       bool
	ctx          context.Context
	closeErr     error
	closeStarted chan struct{}
	closeRelease chan struct{}
	closeOnce    sync.Once
}

func (s *debugAuditStore) Record(auditpkg.Event) {}

func (s *debugAuditStore) Query(auditpkg.Query) ([]auditpkg.Event, error) {
	return nil, nil
}

func (s *debugAuditStore) Prune(time.Duration) error { return nil }

func (s *debugAuditStore) Count() (int64, error) { return 0, nil }

func (s *debugAuditStore) CountByCategory() (map[string]int64, error) {
	return nil, nil
}

func (s *debugAuditStore) Close() error {
	return s.CloseContext(context.Background())
}

func (s *debugAuditStore) CloseContext(ctx context.Context) error {
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

type auditCloseContextKey struct{}

type auditCloseHandle struct {
	ctx context.Context
	err error
}

var _ bkmodule.Handle = (*auditCloseHandle)(nil)

func (h *auditCloseHandle) Close(ctx context.Context) error {
	h.ctx = ctx
	return h.err
}
