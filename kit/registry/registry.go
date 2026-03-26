package registry

import (
	"sync"
	"time"

	"github.com/brainlet/brainkit/sdk"
)

// AIProviderRegistration wraps a typed config for registration.
type AIProviderRegistration struct {
	Type   AIProviderType
	Config any // one of the *ProviderConfig structs
}

// VectorStoreRegistration wraps a typed config for registration.
type VectorStoreRegistration struct {
	Type   VectorStoreType
	Config any // one of the *VectorConfig structs
}

// StorageRegistration wraps a typed config for registration.
type StorageRegistration struct {
	Type   StorageType
	Config any // one of the *StorageConfig structs
}

// ProviderInfo describes a registered AI provider.
type ProviderInfo struct {
	Name         string                `json:"name"`
	Type         AIProviderType        `json:"type"`
	Capabilities AIProviderCapabilities `json:"capabilities"`
	Healthy      bool                  `json:"healthy"`
	LastProbed   time.Time             `json:"lastProbed"`
	LastError    string                `json:"lastError"`
	Latency      time.Duration         `json:"latency"`
}

// VectorStoreInfo describes a registered vector store.
type VectorStoreInfo struct {
	Name         string                  `json:"name"`
	Type         VectorStoreType         `json:"type"`
	Capabilities VectorStoreCapabilities `json:"capabilities"`
	Healthy      bool                    `json:"healthy"`
	LastProbed   time.Time               `json:"lastProbed"`
	LastError    string                  `json:"lastError"`
	Latency      time.Duration           `json:"latency"`
}

// StorageInfo describes a registered storage backend.
type StorageInfo struct {
	Name         string              `json:"name"`
	Type         StorageType         `json:"type"`
	Capabilities StorageCapabilities `json:"capabilities"`
	Healthy      bool                `json:"healthy"`
	LastProbed   time.Time           `json:"lastProbed"`
	LastError    string              `json:"lastError"`
	Latency      time.Duration       `json:"latency"`
}

// ProbeResult is returned by explicit Probe* calls.
type ProbeResult struct {
	Available    bool          `json:"available"`
	Capabilities any           `json:"capabilities"`
	Latency      time.Duration `json:"latency"`
	Error        string        `json:"error,omitempty"`
}

// ProbeConfig configures probing behavior.
type ProbeConfig struct {
	CacheTTL         time.Duration // default: 60s
	ProbeOnRegister  bool          // default: true
	ProbeTimeout     time.Duration // default: 5s
	PeriodicInterval time.Duration // 0 = disabled
}

// entry is the internal state for a registered resource.
type entry struct {
	registration any // AIProviderRegistration, VectorStoreRegistration, StorageRegistration
	healthy      bool
	lastProbed   time.Time
	lastErr      string
	latency      time.Duration
}

// ProviderRegistry manages all registered providers, vector stores, and storages.
type ProviderRegistry struct {
	mu           sync.RWMutex
	aiProviders  map[string]*entry
	vectorStores map[string]*entry
	storages     map[string]*entry
	probeConfig  ProbeConfig
}

// New creates a new ProviderRegistry.
func New(cfg ProbeConfig) *ProviderRegistry {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 60 * time.Second
	}
	if cfg.ProbeTimeout == 0 {
		cfg.ProbeTimeout = 5 * time.Second
	}
	return &ProviderRegistry{
		aiProviders:  make(map[string]*entry),
		vectorStores: make(map[string]*entry),
		storages:     make(map[string]*entry),
		probeConfig:  cfg,
	}
}

// --- AI Providers ---

func (r *ProviderRegistry) RegisterAIProvider(name string, reg AIProviderRegistration) error {
	if name == "" {
		return &sdk.ValidationError{Field: "name", Message: "provider name is required"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.aiProviders[name] = &entry{
		registration: reg,
		lastErr:      "probe pending",
	}
	return nil
}

func (r *ProviderRegistry) UnregisterAIProvider(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.aiProviders, name)
}

func (r *ProviderRegistry) GetAIProvider(name string) (AIProviderRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.aiProviders[name]
	if !ok {
		return AIProviderRegistration{}, false
	}
	return e.registration.(AIProviderRegistration), true
}

func (r *ProviderRegistry) ListAIProviders() []ProviderInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ProviderInfo, 0, len(r.aiProviders))
	for name, e := range r.aiProviders {
		reg := e.registration.(AIProviderRegistration)
		result = append(result, ProviderInfo{
			Name:         name,
			Type:         reg.Type,
			Capabilities: KnownAICapabilities(reg.Type),
			Healthy:      e.healthy,
			LastProbed:   e.lastProbed,
			LastError:    e.lastErr,
			Latency:      e.latency,
		})
	}
	return result
}

func (r *ProviderRegistry) HasAIProvider(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.aiProviders[name]
	return ok
}

// --- Vector Stores ---

func (r *ProviderRegistry) RegisterVectorStore(name string, reg VectorStoreRegistration) error {
	if name == "" {
		return &sdk.ValidationError{Field: "name", Message: "vector store name is required"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.vectorStores[name] = &entry{
		registration: reg,
		lastErr:      "probe pending",
	}
	return nil
}

func (r *ProviderRegistry) UnregisterVectorStore(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.vectorStores, name)
}

func (r *ProviderRegistry) GetVectorStore(name string) (VectorStoreRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.vectorStores[name]
	if !ok {
		return VectorStoreRegistration{}, false
	}
	return e.registration.(VectorStoreRegistration), true
}

func (r *ProviderRegistry) ListVectorStores() []VectorStoreInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]VectorStoreInfo, 0, len(r.vectorStores))
	for name, e := range r.vectorStores {
		reg := e.registration.(VectorStoreRegistration)
		result = append(result, VectorStoreInfo{
			Name:         name,
			Type:         reg.Type,
			Capabilities: DefaultVectorCapabilities(),
			Healthy:      e.healthy,
			LastProbed:   e.lastProbed,
			LastError:    e.lastErr,
			Latency:      e.latency,
		})
	}
	return result
}

func (r *ProviderRegistry) HasVectorStore(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.vectorStores[name]
	return ok
}

// --- Storages ---

func (r *ProviderRegistry) RegisterStorage(name string, reg StorageRegistration) error {
	if name == "" {
		return &sdk.ValidationError{Field: "name", Message: "storage name is required"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.storages[name] = &entry{
		registration: reg,
		lastErr:      "probe pending",
	}
	return nil
}

func (r *ProviderRegistry) UnregisterStorage(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.storages, name)
}

func (r *ProviderRegistry) GetStorage(name string) (StorageRegistration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.storages[name]
	if !ok {
		return StorageRegistration{}, false
	}
	return e.registration.(StorageRegistration), true
}

func (r *ProviderRegistry) ListStorages() []StorageInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]StorageInfo, 0, len(r.storages))
	for name, e := range r.storages {
		reg := e.registration.(StorageRegistration)
		result = append(result, StorageInfo{
			Name:         name,
			Type:         reg.Type,
			Capabilities: DefaultStorageCapabilities(),
			Healthy:      e.healthy,
			LastProbed:   e.lastProbed,
			LastError:    e.lastErr,
			Latency:      e.latency,
		})
	}
	return result
}

func (r *ProviderRegistry) HasStorage(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.storages[name]
	return ok
}
