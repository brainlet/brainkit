package storagehost

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
	"github.com/brainlet/brainkit/sdk"
)

// Manager owns runtime storage bridge processes and mirrors configured
// storage/vector entries into the provider registry.
type Manager struct {
	mu              sync.Mutex
	bridges         map[string]Bridge
	closingBridges  map[string]struct{}
	providers       *providerreg.ProviderRegistry
	hasJSRuntime    func() bool
	closing         bool
	activeCloseJobs int
}

// DebugSnapshot reports storage bridge ownership state for inspect/debug
// surfaces.
type DebugSnapshot struct {
	Closing      bool
	BridgeCount  int
	BridgeNames  []string
	ActiveCloses int
}

const defaultBridgeCloseTimeout = 10 * time.Second

type bridgeCloseTarget struct {
	name      string
	bridge    Bridge
	onSuccess func()
}

// NewManager creates a storage host for one Kit runtime.
func NewManager(providers *providerreg.ProviderRegistry, hasJSRuntime func() bool) *Manager {
	if hasJSRuntime == nil {
		hasJSRuntime = func() bool { return false }
	}
	return &Manager{
		bridges:      map[string]Bridge{},
		providers:    providers,
		hasJSRuntime: hasJSRuntime,
	}
}

// AddStorage registers a new named storage at runtime. For sqlite and an
// active JS runtime, it starts a bridge before updating the provider registry.
func (m *Manager) AddStorage(name string, cfg types.StorageConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cfg.Type == "sqlite" && m.hasJSRuntime() {
		if _, exists := m.bridges[name]; exists {
			return &sdk.AlreadyExistsError{Resource: "storage", Name: name}
		}
		srv, err := newBridge("sqlite", cfg.Path)
		if err != nil {
			return err
		}
		m.bridges[name] = srv
		if err := m.providers.RegisterStorage(name, storageToRegistration(cfg, srv.URL())); err != nil {
			if closeErr := m.closeBridgeLocked(name); closeErr != nil {
				return errors.Join(err, closeErr)
			}
			return err
		}
		return nil
	}
	return m.providers.RegisterStorage(name, storageToRegistration(cfg, ""))
}

// AddVector registers a new named vector store at runtime. For sqlite and an
// active JS runtime, it starts a bridge before updating the provider registry.
func (m *Manager) AddVector(name string, cfg types.VectorConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cfg.Type == "sqlite" && m.hasJSRuntime() {
		key := vectorBridgeKey(name)
		if _, exists := m.bridges[key]; exists {
			return &sdk.AlreadyExistsError{Resource: "vector", Name: name}
		}
		srv, err := newBridge("sqlite", cfg.Path)
		if err != nil {
			return err
		}
		m.bridges[key] = srv
		if err := m.providers.RegisterVectorStore(name, vectorToRegistration(cfg, srv.URL())); err != nil {
			if closeErr := m.closeBridgeLocked(key); closeErr != nil {
				return errors.Join(err, closeErr)
			}
			return err
		}
		return nil
	}
	return m.providers.RegisterVectorStore(name, vectorToRegistration(cfg, ""))
}

// RemoveStorage stops and removes a named storage.
func (m *Manager) RemoveStorage(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultBridgeCloseTimeout)
	defer cancel()
	return m.RemoveStorageContext(ctx, name)
}

// RemoveStorageContext stops and removes a named storage under caller-owned
// teardown cancellation.
func (m *Manager) RemoveStorageContext(ctx context.Context, name string) error {
	return m.removeBridgeContext(ctx, name, func() {
		m.providers.UnregisterStorage(name)
	})
}

// RemoveVector stops and removes a named vector store.
func (m *Manager) RemoveVector(name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultBridgeCloseTimeout)
	defer cancel()
	return m.RemoveVectorContext(ctx, name)
}

// RemoveVectorContext stops and removes a named vector store under caller-owned
// teardown cancellation.
func (m *Manager) RemoveVectorContext(ctx context.Context, name string) error {
	key := vectorBridgeKey(name)
	return m.removeBridgeContext(ctx, key, func() {
		m.providers.UnregisterVectorStore(name)
	})
}

// URL returns the HTTP URL for a named storage bridge.
func (m *Manager) URL(name string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if srv, ok := m.bridges[name]; ok {
		return srv.URL()
	}
	return ""
}

// Names snapshots active storage bridge names.
func (m *Manager) Names() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.bridges))
	for name := range m.bridges {
		out = append(out, name)
	}
	return out
}

// DebugSnapshot returns a read-only storage-host lifecycle view.
func (m *Manager) DebugSnapshot() DebugSnapshot {
	m.mu.Lock()
	names := make([]string, 0, len(m.bridges))
	for name := range m.bridges {
		names = append(names, name)
	}
	closing := m.closing
	activeCloses := m.activeCloseJobs
	m.mu.Unlock()
	sort.Strings(names)
	return DebugSnapshot{Closing: closing, BridgeCount: len(names), BridgeNames: names, ActiveCloses: activeCloses}
}

// ExistingNames snapshots active storage bridge names as a set.
func (m *Manager) ExistingNames() map[string]bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]bool, len(m.bridges))
	for name := range m.bridges {
		out[name] = true
	}
	return out
}

// CloseExcept closes storage bridges not present in keep.
func (m *Manager) CloseExcept(keep map[string]bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultBridgeCloseTimeout)
	defer cancel()
	return m.CloseExceptContext(ctx, keep)
}

// CloseExceptContext closes storage bridges not present in keep under
// caller-owned teardown cancellation.
func (m *Manager) CloseExceptContext(ctx context.Context, keep map[string]bool) error {
	m.mu.Lock()
	targets := make([]bridgeCloseTarget, 0, len(m.bridges))
	names := make([]string, 0, len(m.bridges))
	for name := range m.bridges {
		if keep[name] {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if target, ok := m.markBridgeClosingLocked(name, nil); ok {
			targets = append(targets, target)
		}
	}
	m.mu.Unlock()
	return m.closeBridgeTargets(ctx, targets)
}

// CloseAll closes every active storage bridge.
func (m *Manager) CloseAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultBridgeCloseTimeout)
	defer cancel()
	return m.CloseAllContext(ctx)
}

// CloseAllContext closes every active storage bridge under caller-owned
// teardown cancellation.
func (m *Manager) CloseAllContext(ctx context.Context) error {
	m.mu.Lock()
	targets := make([]bridgeCloseTarget, 0, len(m.bridges))
	names := make([]string, 0, len(m.bridges))
	for name := range m.bridges {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if target, ok := m.markBridgeClosingLocked(name, nil); ok {
			targets = append(targets, target)
		}
	}
	m.mu.Unlock()
	return m.closeBridgeTargets(ctx, targets)
}

// InitBridges starts sqlite bridges for all sqlite storage entries. It returns
// a path-to-URL map so sqlite vector stores can share matching storage bridges.
func (m *Manager) InitBridges(cfg types.KernelConfig) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	bridgeURLs := make(map[string]string)
	started := []string{}
	for name, scfg := range cfg.Storages {
		if scfg.Type != "sqlite" {
			continue
		}
		if _, exists := m.bridges[name]; exists {
			return nil, fmt.Errorf("storage %q bridge already exists", name)
		}
		srv, err := newBridge("sqlite", scfg.Path)
		if err != nil {
			var cleanupErr error
			for _, startedName := range started {
				if closeErr := m.closeBridgeLocked(startedName); closeErr != nil {
					cleanupErr = errors.Join(cleanupErr, closeErr)
				}
			}
			return nil, errors.Join(fmt.Errorf("storage %q: %w", name, err), cleanupErr)
		}
		m.bridges[name] = srv
		started = append(started, name)
		bridgeURLs[scfg.Path] = srv.URL()
	}
	return bridgeURLs, nil
}

// RegisterStorages mirrors configured storages into the provider registry.
func (m *Manager) RegisterStorages(cfg types.KernelConfig, bridgeURLs map[string]string) error {
	for name, scfg := range cfg.Storages {
		bridgeURL := ""
		if scfg.Type == "sqlite" {
			bridgeURL = bridgeURLs[scfg.Path]
		}
		if err := m.providers.RegisterStorage(name, storageToRegistration(scfg, bridgeURL)); err != nil {
			if scfg.Type == "sqlite" && bridgeURL != "" {
				return errors.Join(fmt.Errorf("storage %q: %w", name, err), m.closeBridge(name))
			}
			return fmt.Errorf("storage %q: %w", name, err)
		}
	}
	return nil
}

// RegisterVectors mirrors configured vector stores into the provider registry.
// For sqlite vectors, it reuses the bridge URL from a matching storage path.
func (m *Manager) RegisterVectors(cfg types.KernelConfig, bridgeURLs map[string]string) error {
	for name, vcfg := range cfg.Vectors {
		bridgeURL := ""
		if vcfg.Type == "sqlite" {
			bridgeURL = bridgeURLs[vcfg.Path]
			if bridgeURL == "" && bridgeURLs != nil {
				srv, err := newBridge("sqlite", vcfg.Path)
				if err != nil {
					return fmt.Errorf("vector %q: %w", name, err)
				}
				m.mu.Lock()
				if _, ok := m.bridges[vectorBridgeKey(name)]; ok {
					m.mu.Unlock()
					err := fmt.Errorf("vector %q bridge already exists", name)
					ctx, cancel := context.WithTimeout(context.Background(), defaultBridgeCloseTimeout)
					if closeErr := srv.CloseContext(ctx); closeErr != nil {
						cancel()
						return errors.Join(err, fmt.Errorf("vector %q cleanup: %w", name, closeErr))
					}
					cancel()
					return err
				}
				m.bridges[vectorBridgeKey(name)] = srv
				m.mu.Unlock()
				bridgeURL = srv.URL()
				bridgeURLs[vcfg.Path] = bridgeURL
			}
		}
		if err := m.providers.RegisterVectorStore(name, vectorToRegistration(vcfg, bridgeURL)); err != nil {
			if vcfg.Type == "sqlite" && bridgeURL != "" && bridgeURLs != nil {
				return errors.Join(fmt.Errorf("vector %q: %w", name, err), m.closeBridge(vectorBridgeKey(name)))
			}
			return fmt.Errorf("vector %q: %w", name, err)
		}
	}
	return nil
}

// RestoreConfiguredRegistry mirrors configured storage and vector entries into
// the provider registry for a live kernel after runtime-specific bridge
// resources have been detached.
func (m *Manager) RestoreConfiguredRegistry(cfg types.KernelConfig, bridgeURLs map[string]string) error {
	return errors.Join(
		m.RegisterStorages(cfg, bridgeURLs),
		m.RegisterVectors(cfg, bridgeURLs),
	)
}

func (m *Manager) closeBridge(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeBridgeLocked(name)
}

func (m *Manager) closeBridgeLocked(name string) error {
	if srv, ok := m.bridges[name]; ok {
		ctx, cancel := context.WithTimeout(context.Background(), defaultBridgeCloseTimeout)
		defer cancel()
		if err := srv.CloseContext(ctx); err != nil {
			return fmt.Errorf("storage %q: %w", name, err)
		}
		delete(m.bridges, name)
	}
	return nil
}

func (m *Manager) removeBridgeContext(ctx context.Context, name string, onSuccess func()) error {
	m.mu.Lock()
	target, ok := m.markBridgeClosingLocked(name, onSuccess)
	if !ok {
		_, exists := m.bridges[name]
		m.mu.Unlock()
		if exists {
			return fmt.Errorf("storage %q: close already in progress", name)
		}
		if onSuccess != nil {
			onSuccess()
		}
		return nil
	}
	m.mu.Unlock()
	return m.closeBridgeTarget(ctx, target)
}

func (m *Manager) markBridgeClosingLocked(name string, onSuccess func()) (bridgeCloseTarget, bool) {
	srv, ok := m.bridges[name]
	if !ok {
		return bridgeCloseTarget{}, false
	}
	if m.closingBridges == nil {
		m.closingBridges = map[string]struct{}{}
	}
	if _, closing := m.closingBridges[name]; closing {
		return bridgeCloseTarget{}, false
	}
	m.closingBridges[name] = struct{}{}
	m.activeCloseJobs++
	m.closing = true
	return bridgeCloseTarget{name: name, bridge: srv, onSuccess: onSuccess}, true
}

func (m *Manager) closeBridgeTargets(ctx context.Context, targets []bridgeCloseTarget) error {
	var err error
	for _, target := range targets {
		err = errors.Join(err, m.closeBridgeTarget(ctx, target))
	}
	return err
}

func (m *Manager) closeBridgeTarget(ctx context.Context, target bridgeCloseTarget) error {
	if ctx == nil {
		ctx = context.Background()
	}
	done := make(chan error, 1)
	go func() {
		done <- m.finishBridgeClose(target, target.bridge.CloseContext(ctx))
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("storage %q: %w", target.name, ctx.Err())
	}
}

func (m *Manager) finishBridgeClose(target bridgeCloseTarget, closeErr error) error {
	var onSuccess func()
	m.mu.Lock()
	if closeErr == nil {
		if current, ok := m.bridges[target.name]; ok && current == target.bridge {
			delete(m.bridges, target.name)
			onSuccess = target.onSuccess
		}
	}
	delete(m.closingBridges, target.name)
	if m.activeCloseJobs > 0 {
		m.activeCloseJobs--
	}
	if m.activeCloseJobs == 0 {
		m.closing = false
	}
	m.mu.Unlock()
	if onSuccess != nil {
		onSuccess()
	}
	if closeErr != nil {
		return fmt.Errorf("storage %q: %w", target.name, closeErr)
	}
	return nil
}

func vectorBridgeKey(name string) string {
	return "vec_" + name
}
