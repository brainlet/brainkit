package storagehost

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
)

type fakeBridge struct {
	url       string
	closeErr  error
	started   chan struct{}
	release   chan struct{}
	ignoreCtx bool
	mu        sync.Mutex
	closeHits int
	startOnce sync.Once
}

func (b *fakeBridge) URL() string { return b.url }

func (b *fakeBridge) CloseContext(ctx context.Context) error {
	b.mu.Lock()
	b.closeHits++
	b.mu.Unlock()
	if b.started != nil {
		b.startOnce.Do(func() { close(b.started) })
	}
	if b.release != nil {
		if b.ignoreCtx {
			<-b.release
			return b.closeErr
		}
		select {
		case <-b.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return b.closeErr
}

func (b *fakeBridge) Close() error {
	return b.CloseContext(context.Background())
}

func (b *fakeBridge) CloseHits() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closeHits
}

func TestAddVectorSQLiteStartsBridgeAndRemoveClosesIt(t *testing.T) {
	var startedPath string
	var started *fakeBridge
	RegisterBridgeBuilder("sqlite", func(path string) (Bridge, error) {
		startedPath = path
		started = &fakeBridge{url: "http://127.0.0.1:4321"}
		return started, nil
	})
	t.Cleanup(func() { RegisterBridgeBuilder("sqlite", nil) })

	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })

	if err := manager.AddVector("docs", types.SQLiteVector("/tmp/docs.db")); err != nil {
		t.Fatalf("add vector: %v", err)
	}
	if startedPath != "/tmp/docs.db" {
		t.Fatalf("bridge path = %q, want /tmp/docs.db", startedPath)
	}
	if !slices.Contains(manager.Names(), "vec_docs") {
		t.Fatalf("bridge names = %v, want vec_docs", manager.Names())
	}

	reg, ok := providers.GetVectorStore("docs")
	if !ok {
		t.Fatal("vector store not registered")
	}
	if reg.Type != providerreg.VectorStoreLibSQL {
		t.Fatalf("vector type = %q, want %q", reg.Type, providerreg.VectorStoreLibSQL)
	}
	config, ok := reg.Config.(providerreg.LibSQLVectorConfig)
	if !ok {
		t.Fatalf("vector config type = %T, want LibSQLVectorConfig", reg.Config)
	}
	if config.URL != "http://127.0.0.1:4321" {
		t.Fatalf("vector bridge URL = %q", config.URL)
	}

	if err := manager.RemoveVector("docs"); err != nil {
		t.Fatalf("remove vector: %v", err)
	}
	if providers.HasVectorStore("docs") {
		t.Fatal("vector store still registered after remove")
	}
	if started.CloseHits() != 1 {
		t.Fatalf("bridge close hits = %d, want 1", started.CloseHits())
	}
}

func TestRemoveStorageKeepsBridgeAndRegistryWhenCloseFails(t *testing.T) {
	want := errors.New("close failed")
	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })
	bridge := &fakeBridge{url: "http://127.0.0.1:4321", closeErr: want}
	manager.bridges["default"] = bridge
	if err := providers.RegisterStorage("default", providerreg.StorageRegistration{Type: providerreg.StorageLibSQL}); err != nil {
		t.Fatalf("register storage: %v", err)
	}

	err := manager.RemoveStorage("default")
	if !errors.Is(err, want) {
		t.Fatalf("RemoveStorage error = %v, want %v", err, want)
	}
	if bridge.CloseHits() != 1 {
		t.Fatalf("bridge close hits = %d, want 1", bridge.CloseHits())
	}
	if !slices.Contains(manager.Names(), "default") {
		t.Fatalf("bridge names = %v, want default preserved", manager.Names())
	}
	if !providers.HasStorage("default") {
		t.Fatal("storage registry entry should remain when bridge close fails")
	}
}

func TestRemoveVectorKeepsBridgeAndRegistryWhenCloseFails(t *testing.T) {
	want := errors.New("close failed")
	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })
	bridge := &fakeBridge{url: "http://127.0.0.1:4321", closeErr: want}
	manager.bridges["vec_docs"] = bridge
	if err := providers.RegisterVectorStore("docs", providerreg.VectorStoreRegistration{Type: providerreg.VectorStoreLibSQL}); err != nil {
		t.Fatalf("register vector: %v", err)
	}

	err := manager.RemoveVector("docs")
	if !errors.Is(err, want) {
		t.Fatalf("RemoveVector error = %v, want %v", err, want)
	}
	if bridge.CloseHits() != 1 {
		t.Fatalf("bridge close hits = %d, want 1", bridge.CloseHits())
	}
	if !slices.Contains(manager.Names(), "vec_docs") {
		t.Fatalf("bridge names = %v, want vec_docs preserved", manager.Names())
	}
	if !providers.HasVectorStore("docs") {
		t.Fatal("vector registry entry should remain when bridge close fails")
	}
}

func TestAddVectorSQLiteWithoutJSRuntimeDoesNotStartBridge(t *testing.T) {
	RegisterBridgeBuilder("sqlite", func(path string) (Bridge, error) {
		t.Fatalf("bridge builder should not be called, got path %q", path)
		return nil, nil
	})
	t.Cleanup(func() { RegisterBridgeBuilder("sqlite", nil) })

	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return false })

	if err := manager.AddVector("docs", types.SQLiteVector("/tmp/docs.db")); err != nil {
		t.Fatalf("add vector: %v", err)
	}
	if len(manager.Names()) != 0 {
		t.Fatalf("bridge names = %v, want none", manager.Names())
	}
	reg, ok := providers.GetVectorStore("docs")
	if !ok {
		t.Fatal("vector store not registered")
	}
	config, ok := reg.Config.(providerreg.LibSQLVectorConfig)
	if !ok {
		t.Fatalf("vector config type = %T, want LibSQLVectorConfig", reg.Config)
	}
	if config.URL != "" {
		t.Fatalf("vector bridge URL = %q, want empty", config.URL)
	}
}

func TestRegisterVectorsClosesStartedBridgeOnRegistrationError(t *testing.T) {
	var started *fakeBridge
	RegisterBridgeBuilder("sqlite", func(path string) (Bridge, error) {
		started = &fakeBridge{url: "http://127.0.0.1:4321"}
		return started, nil
	})
	t.Cleanup(func() { RegisterBridgeBuilder("sqlite", nil) })

	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })

	err := manager.RegisterVectors(types.KernelConfig{
		Vectors: map[string]types.VectorConfig{
			"": types.SQLiteVector("/tmp/bad.db"),
		},
	}, map[string]string{})
	if err == nil {
		t.Fatal("RegisterVectors: expected empty-name error")
	}
	if started == nil {
		t.Fatal("bridge was not started before registration failed")
	}
	if started.CloseHits() != 1 {
		t.Fatalf("bridge close hits = %d, want 1", started.CloseHits())
	}
	if len(manager.Names()) != 0 {
		t.Fatalf("bridge names after error = %v, want none", manager.Names())
	}
}

func TestInitBridgesClosesStartedBridgeOnLaterError(t *testing.T) {
	var started *fakeBridge
	RegisterBridgeBuilder("sqlite", func(path string) (Bridge, error) {
		if path == "/tmp/fail.db" {
			return nil, fmt.Errorf("boom")
		}
		started = &fakeBridge{url: "http://127.0.0.1:4321"}
		return started, nil
	})
	t.Cleanup(func() { RegisterBridgeBuilder("sqlite", nil) })

	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })

	_, err := manager.InitBridges(types.KernelConfig{
		Storages: map[string]types.StorageConfig{
			"ok":   types.SQLiteStorage("/tmp/ok.db"),
			"fail": types.SQLiteStorage("/tmp/fail.db"),
		},
	})
	if err == nil {
		t.Fatal("InitBridges: expected bridge error")
	}
	if started != nil && started.CloseHits() != 1 {
		t.Fatalf("started bridge close hits = %d, want 1", started.CloseHits())
	}
	if len(manager.Names()) != 0 {
		t.Fatalf("bridge names after error = %v, want none", manager.Names())
	}
}

func TestCloseExceptReturnsBridgeCloseError(t *testing.T) {
	want := errors.New("close failed")
	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })
	keep := &fakeBridge{url: "http://127.0.0.1:4321"}
	drop := &fakeBridge{url: "http://127.0.0.1:4322", closeErr: want}
	manager.bridges["keep"] = keep
	manager.bridges["drop"] = drop

	err := manager.CloseExcept(map[string]bool{"keep": true})
	if !errors.Is(err, want) {
		t.Fatalf("CloseExcept error = %v, want %v", err, want)
	}
	if drop.CloseHits() != 1 {
		t.Fatalf("dropped bridge close hits = %d, want 1", drop.CloseHits())
	}
	if keep.CloseHits() != 0 {
		t.Fatalf("kept bridge close hits = %d, want 0", keep.CloseHits())
	}
	if !slices.Contains(manager.Names(), "keep") {
		t.Fatalf("bridge names = %v, want keep", manager.Names())
	}
	if !slices.Contains(manager.Names(), "drop") {
		t.Fatalf("bridge names = %v, want failed drop preserved", manager.Names())
	}
}

func TestCloseAllKeepsFailedBridgeForRetry(t *testing.T) {
	want := errors.New("close failed")
	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })
	ok := &fakeBridge{url: "http://127.0.0.1:4321"}
	fail := &fakeBridge{url: "http://127.0.0.1:4322", closeErr: want}
	manager.bridges["ok"] = ok
	manager.bridges["fail"] = fail

	err := manager.CloseAll()
	if !errors.Is(err, want) {
		t.Fatalf("CloseAll error = %v, want %v", err, want)
	}
	if ok.CloseHits() != 1 || fail.CloseHits() != 1 {
		t.Fatalf("close hits ok=%d fail=%d, want both 1", ok.CloseHits(), fail.CloseHits())
	}
	names := manager.Names()
	if slices.Contains(names, "ok") {
		t.Fatalf("bridge names = %v, want ok removed", names)
	}
	if !slices.Contains(names, "fail") {
		t.Fatalf("bridge names = %v, want failed bridge preserved", names)
	}
}

func TestCloseAllDebugSnapshotReportsClosing(t *testing.T) {
	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })
	bridge := &fakeBridge{
		url:     "http://127.0.0.1:4321",
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	manager.bridges["slow"] = bridge

	done := make(chan error, 1)
	go func() {
		done <- manager.CloseAll()
	}()
	select {
	case <-bridge.started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bridge close to start")
	}

	snapshot := manager.DebugSnapshot()
	if !snapshot.Closing {
		t.Fatal("Closing = false, want true")
	}
	if snapshot.ActiveCloses != 1 {
		t.Fatalf("ActiveCloses = %d, want 1", snapshot.ActiveCloses)
	}
	if snapshot.BridgeCount != 1 || !slices.Contains(snapshot.BridgeNames, "slow") {
		t.Fatalf("snapshot during close = %#v, want slow bridge still tracked", snapshot)
	}

	close(bridge.release)
	if err := <-done; err != nil {
		t.Fatalf("CloseAll: %v", err)
	}
	snapshot = manager.DebugSnapshot()
	if snapshot.Closing || snapshot.BridgeCount != 0 {
		t.Fatalf("snapshot after close = %#v, want closing=false and no bridges", snapshot)
	}
}

func TestCloseAllContextKeepsBridgeOnDeadline(t *testing.T) {
	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })
	bridge := &fakeBridge{
		url:     "http://127.0.0.1:4321",
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	manager.bridges["slow"] = bridge

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := manager.CloseAllContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseAllContext error = %v, want deadline exceeded", err)
	}
	if bridge.CloseHits() != 1 {
		t.Fatalf("bridge close hits = %d, want 1", bridge.CloseHits())
	}
	names := manager.Names()
	if !slices.Contains(names, "slow") {
		t.Fatalf("bridge names = %v, want slow preserved after deadline", names)
	}
	requireEventuallyStorageSnapshot(t, manager, func(snapshot DebugSnapshot) bool {
		return !snapshot.Closing && snapshot.ActiveCloses == 0
	}, "no active close after deadline")
	close(bridge.release)
}

func TestCloseAllContextReturnsDeadlineWhenBridgeIgnoresContext(t *testing.T) {
	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })
	bridge := &fakeBridge{
		url:       "http://127.0.0.1:4321",
		started:   make(chan struct{}),
		release:   make(chan struct{}),
		ignoreCtx: true,
	}
	manager.bridges["stuck"] = bridge

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := manager.CloseAllContext(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("CloseAllContext error = %v, want deadline exceeded", err)
	}
	if bridge.CloseHits() != 1 {
		t.Fatalf("bridge close hits = %d, want 1", bridge.CloseHits())
	}
	snapshot := manager.DebugSnapshot()
	if !snapshot.Closing || snapshot.ActiveCloses != 1 || snapshot.BridgeCount != 1 {
		t.Fatalf("snapshot after ignored deadline = %#v, want retained active close", snapshot)
	}

	close(bridge.release)
	requireEventuallyStorageSnapshot(t, manager, func(snapshot DebugSnapshot) bool {
		return !snapshot.Closing && snapshot.ActiveCloses == 0 && snapshot.BridgeCount == 0
	}, "ignored-context bridge close to finish")
}

func TestRemoveStorageContextKeepsRegistryOnDeadline(t *testing.T) {
	providers := providerreg.New(types.ProbeConfig{})
	manager := NewManager(providers, func() bool { return true })
	bridge := &fakeBridge{
		url:     "http://127.0.0.1:4321",
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	manager.bridges["default"] = bridge
	if err := providers.RegisterStorage("default", providerreg.StorageRegistration{Type: providerreg.StorageLibSQL}); err != nil {
		t.Fatalf("register storage: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := manager.RemoveStorageContext(ctx, "default")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RemoveStorageContext error = %v, want deadline exceeded", err)
	}
	if bridge.CloseHits() != 1 {
		t.Fatalf("bridge close hits = %d, want 1", bridge.CloseHits())
	}
	if !slices.Contains(manager.Names(), "default") {
		t.Fatalf("bridge names = %v, want default preserved after deadline", manager.Names())
	}
	if !providers.HasStorage("default") {
		t.Fatal("storage registry entry should remain when bridge close times out")
	}
	requireEventuallyStorageSnapshot(t, manager, func(snapshot DebugSnapshot) bool {
		return !snapshot.Closing && snapshot.ActiveCloses == 0
	}, "remove storage close bookkeeping to clear")
	close(bridge.release)
}

func requireEventuallyStorageSnapshot(t *testing.T, manager *Manager, fn func(DebugSnapshot) bool, label string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if fn(manager.DebugSnapshot()) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s; snapshot=%#v", label, manager.DebugSnapshot())
}
