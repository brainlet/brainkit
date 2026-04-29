package storagehost

import (
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modules/registry/providerreg"
	"github.com/brainlet/brainkit/sdk"
)

// Manager owns runtime storage bridge processes and mirrors configured
// storage/vector entries into the provider registry.
type Manager struct {
	mu           sync.Mutex
	bridges      map[string]Bridge
	providers    *providerreg.ProviderRegistry
	hasJSRuntime func() bool
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
		m.providers.RegisterStorage(name, storageToRegistration(cfg, srv.URL()))
		return nil
	}
	m.providers.RegisterStorage(name, storageToRegistration(cfg, ""))
	return nil
}

// RemoveStorage stops and removes a named storage.
func (m *Manager) RemoveStorage(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if srv, ok := m.bridges[name]; ok {
		_ = srv.Close()
		delete(m.bridges, name)
	}
	m.providers.UnregisterStorage(name)
	return nil
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
func (m *Manager) CloseExcept(keep map[string]bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, srv := range m.bridges {
		if keep[name] {
			continue
		}
		_ = srv.Close()
		delete(m.bridges, name)
	}
}

// CloseAll closes every active storage bridge.
func (m *Manager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var firstErr error
	for name, srv := range m.bridges {
		if err := srv.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("storage %q: %w", name, err)
		}
		delete(m.bridges, name)
	}
	return firstErr
}

// InitBridges starts sqlite bridges for all sqlite storage entries. It returns
// a path-to-URL map so sqlite vector stores can share matching storage bridges.
func (m *Manager) InitBridges(cfg types.KernelConfig) (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	bridgeURLs := make(map[string]string)
	for name, scfg := range cfg.Storages {
		if scfg.Type != "sqlite" {
			continue
		}
		srv, err := newBridge("sqlite", scfg.Path)
		if err != nil {
			return nil, fmt.Errorf("storage %q: %w", name, err)
		}
		m.bridges[name] = srv
		bridgeURLs[scfg.Path] = srv.URL()
	}
	return bridgeURLs, nil
}

// RegisterStorages mirrors configured storages into the provider registry.
func (m *Manager) RegisterStorages(cfg types.KernelConfig, bridgeURLs map[string]string) {
	for name, scfg := range cfg.Storages {
		bridgeURL := ""
		if scfg.Type == "sqlite" {
			bridgeURL = bridgeURLs[scfg.Path]
		}
		m.providers.RegisterStorage(name, storageToRegistration(scfg, bridgeURL))
	}
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
				m.bridges["vec_"+name] = srv
				m.mu.Unlock()
				bridgeURL = srv.URL()
				bridgeURLs[vcfg.Path] = bridgeURL
			}
		}
		m.providers.RegisterVectorStore(name, vectorToRegistration(vcfg, bridgeURL))
	}
	return nil
}
