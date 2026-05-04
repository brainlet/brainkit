package schedules

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
)

func TestModuleCloseContextReturnsScheduleHandlerLeaseError(t *testing.T) {
	want := errors.New("detach schedule handler")
	ctx := context.WithValue(context.Background(), scheduleCloseContextKey{}, "close")
	handle := &scheduleCloseHandle{err: want}
	module := &Module{scheduleHandlerLease: handle}
	module.scheduleHandlerLeaseActive.Store(true)

	err := module.CloseContext(ctx)
	if !errors.Is(err, want) {
		t.Fatalf("CloseContext error = %v, want %v", err, want)
	}
	if handle.ctx != ctx {
		t.Fatalf("lease closed with context %v, want %v", handle.ctx, ctx)
	}
	if module.scheduleHandlerLease == nil {
		t.Fatal("schedule handler lease was cleared after close failure")
	}
	if !module.scheduleHandlerLeaseActive.Load() {
		t.Fatal("schedule handler lease no longer marked active after close failure")
	}

	handle.setErr(nil)
	if err := module.CloseContext(ctx); err != nil {
		t.Fatalf("retry close module: %v", err)
	}
	if got := handle.closeCount(); got != 2 {
		t.Fatalf("lease close count = %d, want 2", got)
	}
	if module.scheduleHandlerLease != nil {
		t.Fatal("schedule handler lease was not cleared after successful retry")
	}
	if module.scheduleHandlerLeaseActive.Load() {
		t.Fatal("schedule handler lease still marked active after successful retry")
	}
}

func TestModuleCloseContextClosesOwnedStoreOnce(t *testing.T) {
	store := &scheduleCloseStore{}
	module := &Module{cfg: Config{Store: store, ownsStore: true}}

	if err := module.CloseContext(context.Background()); err != nil {
		t.Fatalf("close module: %v", err)
	}
	if got := store.closeCount(); got != 1 {
		t.Fatalf("owned store close count = %d, want 1", got)
	}
	if module.cfg.Store != nil || module.cfg.ownsStore {
		t.Fatal("owned store was not detached after close")
	}

	if err := module.CloseContext(context.Background()); err != nil {
		t.Fatalf("second close module: %v", err)
	}
	if got := store.closeCount(); got != 1 {
		t.Fatalf("owned store close count after second close = %d, want 1", got)
	}
}

func TestModuleCloseContextDoesNotCloseBorrowedStore(t *testing.T) {
	store := &scheduleCloseStore{}
	module := &Module{cfg: Config{Store: store}}

	if err := module.CloseContext(context.Background()); err != nil {
		t.Fatalf("close module: %v", err)
	}
	if got := store.closeCount(); got != 0 {
		t.Fatalf("borrowed store close count = %d, want 0", got)
	}
	if module.cfg.Store == nil {
		t.Fatal("borrowed store should remain attached to config")
	}
}

func TestModuleCloseContextKeepsOwnedStoreWhenStoreCloseFails(t *testing.T) {
	want := errors.New("close schedule store")
	ctx := context.WithValue(context.Background(), scheduleCloseContextKey{}, "close")
	store := &scheduleCloseStore{closeErr: want}
	module := &Module{cfg: Config{Store: store, ownsStore: true}}

	err := module.CloseContext(ctx)
	if !errors.Is(err, want) {
		t.Fatalf("CloseContext error = %v, want %v", err, want)
	}
	if store.ctx != ctx {
		t.Fatalf("owned store closed with context %v, want %v", store.ctx, ctx)
	}
	if got := store.closeCount(); got != 1 {
		t.Fatalf("owned store close count = %d, want 1", got)
	}
	if module.cfg.Store != store || !module.cfg.ownsStore {
		t.Fatalf("store config = (%#v, owns=%v), want retained owned store", module.cfg.Store, module.cfg.ownsStore)
	}

	store.setCloseErr(nil)
	if err := module.CloseContext(ctx); err != nil {
		t.Fatalf("retry close module: %v", err)
	}
	if got := store.closeCount(); got != 2 {
		t.Fatalf("owned store close count after retry = %d, want 2", got)
	}
	if module.cfg.Store != nil || module.cfg.ownsStore {
		t.Fatal("owned store was not detached after successful retry")
	}
}

func TestModuleCloseContextSkipsOwnedStoreWhenSchedulerCloseTimesOut(t *testing.T) {
	store := &scheduleCloseStore{}
	pub := &blockingPublisher{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	scheduler := newScheduler(pub, store, nil, nil, nil, nil)
	module := &Module{cfg: Config{Store: store, ownsStore: true}, scheduler: scheduler}

	if _, err := scheduler.Schedule(context.Background(), ScheduleConfig{
		ID:         "tick",
		Expression: "every 1ms",
		Topic:      "events.tick",
		Payload:    json.RawMessage(`{}`),
	}); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	select {
	case <-pub.started:
	case <-time.After(time.Second):
		t.Fatal("scheduled fire did not start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := module.CloseContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseContext error = %v, want context deadline exceeded", err)
	}
	if got := store.closeCount(); got != 0 {
		t.Fatalf("owned store close count = %d, want 0 while scheduler fire is active", got)
	}
	if module.cfg.Store != store || !module.cfg.ownsStore {
		t.Fatalf("store config = (%#v, owns=%v), want retained owned store", module.cfg.Store, module.cfg.ownsStore)
	}

	close(pub.release)
	if err := module.CloseContext(context.Background()); err != nil {
		t.Fatalf("retry close module: %v", err)
	}
	if got := store.closeCount(); got != 1 {
		t.Fatalf("owned store close count after scheduler joined = %d, want 1", got)
	}
	if module.cfg.Store != nil || module.cfg.ownsStore {
		t.Fatal("owned store was not detached after scheduler joined")
	}
}

func TestModuleDebugSnapshotReportsClosingDuringSlowStoreClose(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	store := &scheduleCloseStore{closeStarted: started, closeRelease: release}
	module := &Module{cfg: Config{Store: store, ownsStore: true}}

	done := make(chan error, 1)
	go func() {
		done <- module.CloseContext(context.Background())
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for schedule store close to start")
	}

	snapshotDone := make(chan DebugSnapshot, 1)
	go func() {
		snapshotDone <- module.DebugSnapshot()
	}()
	select {
	case snapshot := <-snapshotDone:
		if !snapshot.Closing || !snapshot.StoreConfigured {
			t.Fatalf("snapshot during close = %#v, want closing configured store", snapshot)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DebugSnapshot blocked while schedule store Close was in progress")
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("CloseContext: %v", err)
	}
	snapshot := module.DebugSnapshot()
	if snapshot.Closing || snapshot.StoreConfigured {
		t.Fatalf("snapshot after close = %#v, want detached store", snapshot)
	}
}

func TestModuleCloseContextReturnsDeadlineWhenOwnedStoreIgnoresContext(t *testing.T) {
	release := make(chan struct{})
	store := &scheduleCloseStore{closeRelease: release}
	module := &Module{cfg: Config{Store: store, ownsStore: true}}
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
	if !snapshot.StoreClosing || !snapshot.StoreConfigured {
		t.Fatalf("snapshot after deadline = %#v, want retained closing store", snapshot)
	}

	close(release)
	requireEventuallyScheduleSnapshot(t, module, func(snapshot DebugSnapshot) bool {
		return !snapshot.StoreClosing && !snapshot.StoreConfigured
	}, "schedule store close job to finish")
}

func requireEventuallyScheduleSnapshot(t *testing.T, module *Module, fn func(DebugSnapshot) bool, label string) {
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

func TestModuleDebugSnapshotReportsClosingDuringSlowLeaseClose(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	handle := &scheduleCloseHandle{closeStarted: started, closeRelease: release}
	module := &Module{scheduleHandlerLease: handle}
	module.scheduleHandlerLeaseActive.Store(true)

	done := make(chan error, 1)
	go func() {
		done <- module.CloseContext(context.Background())
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for schedule handler lease close to start")
	}

	snapshotDone := make(chan DebugSnapshot, 1)
	go func() {
		snapshotDone <- module.DebugSnapshot()
	}()
	select {
	case snapshot := <-snapshotDone:
		if !snapshot.Closing || !snapshot.ScheduleHandlerLeaseAttached {
			t.Fatalf("snapshot during close = %#v, want closing attached schedule handler lease", snapshot)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DebugSnapshot blocked while schedule handler lease Close was in progress")
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("CloseContext: %v", err)
	}
	snapshot := module.DebugSnapshot()
	if snapshot.Closing || snapshot.ScheduleHandlerLeaseAttached {
		t.Fatalf("snapshot after close = %#v, want detached schedule handler lease", snapshot)
	}
}

type scheduleCloseContextKey struct{}

type scheduleCloseHandle struct {
	mu           sync.Mutex
	ctx          context.Context
	err          error
	closed       int
	closeStarted chan struct{}
	closeRelease chan struct{}
	closeOnce    sync.Once
}

var _ bkmodule.Handle = (*scheduleCloseHandle)(nil)

func (h *scheduleCloseHandle) Close(ctx context.Context) error {
	h.mu.Lock()
	h.ctx = ctx
	h.closed++
	err := h.err
	started := h.closeStarted
	release := h.closeRelease
	h.mu.Unlock()

	if started != nil {
		h.closeOnce.Do(func() { close(started) })
	}
	if release != nil {
		<-release
	}
	return err
}

func (h *scheduleCloseHandle) setErr(err error) {
	h.mu.Lock()
	h.err = err
	h.mu.Unlock()
}

func (h *scheduleCloseHandle) closeCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closed
}

type scheduleCloseStore struct {
	mu           sync.Mutex
	ctx          context.Context
	closed       int
	closeErr     error
	closeStarted chan struct{}
	closeRelease chan struct{}
	closeOnce    sync.Once
}

func (s *scheduleCloseStore) SaveSchedule(types.PersistedSchedule) error { return nil }
func (s *scheduleCloseStore) LoadSchedules() ([]types.PersistedSchedule, error) {
	return nil, nil
}
func (s *scheduleCloseStore) DeleteSchedule(string) error { return nil }
func (s *scheduleCloseStore) ClaimScheduleFire(string, time.Time) (bool, error) {
	return true, nil
}
func (s *scheduleCloseStore) Close() error {
	return s.CloseContext(context.Background())
}

func (s *scheduleCloseStore) CloseContext(ctx context.Context) error {
	s.mu.Lock()
	s.ctx = ctx
	s.closed++
	err := s.closeErr
	started := s.closeStarted
	release := s.closeRelease
	s.mu.Unlock()

	if started != nil {
		s.closeOnce.Do(func() { close(started) })
	}
	if release != nil {
		<-release
	}
	return err
}

func (s *scheduleCloseStore) setCloseErr(err error) {
	s.mu.Lock()
	s.closeErr = err
	s.mu.Unlock()
}

func (s *scheduleCloseStore) closeCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}
