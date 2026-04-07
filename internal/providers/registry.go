package providers

import (
	"time"

	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
)

// Type aliases from internal/types
type AIProviderRegistration = types.AIProviderRegistration
type AIProviderType = types.AIProviderType
type AIProviderCapabilities = types.AIProviderCapabilities
type VectorStoreRegistration = types.VectorStoreRegistration
type VectorStoreType = types.VectorStoreType
type VectorStoreCapabilities = types.VectorStoreCapabilities
type VectorStoreInfo = types.VectorStoreInfo
type StorageRegistration = types.StorageRegistration
type StorageType = types.StorageType
type StorageCapabilities = types.StorageCapabilities
type StorageInfo = types.StorageInfo
type ProviderInfo = types.ProviderInfo
type ProbeConfig = types.ProbeConfig
type ProbeResult = types.ProbeResult

// Re-export constants and functions from types
var (
	KnownAICapabilities        = types.KnownAICapabilities
	DefaultVectorCapabilities   = types.DefaultVectorCapabilities
	DefaultStorageCapabilities  = types.DefaultStorageCapabilities
)

// Re-export all storage type constants
const (
	StorageInMemory = types.StorageInMemory
	StorageLibSQL   = types.StorageLibSQL
	StoragePostgres = types.StoragePostgres
	StorageMongoDB  = types.StorageMongoDB
	StorageUpstash  = types.StorageUpstash
)

// Re-export vector store type constants
const (
	VectorStoreLibSQL  = types.VectorStoreLibSQL
	VectorStorePg      = types.VectorStorePg
	VectorStoreMongoDB = types.VectorStoreMongoDB
)

// Re-export storage config types
type LibSQLStorageConfig = types.LibSQLStorageConfig
type PostgresStorageConfig = types.PostgresStorageConfig
type MongoDBStorageConfig = types.MongoDBStorageConfig
type UpstashStorageConfig = types.UpstashStorageConfig

// Re-export vector config types
type LibSQLVectorConfig = types.LibSQLVectorConfig
type PgVectorConfig = types.PgVectorConfig
type MongoDBVectorConfig = types.MongoDBVectorConfig

// Re-export provider config types
type OpenAIProviderConfig = types.OpenAIProviderConfig
type AnthropicProviderConfig = types.AnthropicProviderConfig
type GoogleProviderConfig = types.GoogleProviderConfig
type MistralProviderConfig = types.MistralProviderConfig
type CohereProviderConfig = types.CohereProviderConfig
type GroqProviderConfig = types.GroqProviderConfig
type PerplexityProviderConfig = types.PerplexityProviderConfig
type DeepSeekProviderConfig = types.DeepSeekProviderConfig
type FireworksProviderConfig = types.FireworksProviderConfig
type TogetherAIProviderConfig = types.TogetherAIProviderConfig
type XAIProviderConfig = types.XAIProviderConfig
type AzureProviderConfig = types.AzureProviderConfig
type BedrockProviderConfig = types.BedrockProviderConfig
type VertexProviderConfig = types.VertexProviderConfig
type HuggingFaceProviderConfig = types.HuggingFaceProviderConfig
type CerebrasProviderConfig = types.CerebrasProviderConfig

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
	mu           syncx.RWMutex
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
	if cfg.PeriodicInterval == 0 {
		cfg.PeriodicInterval = 60 * time.Second
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
	r.aiProviders[name] = &entry{
		registration: reg,
		lastErr:      "probe pending",
	}
	r.mu.Unlock()
	if r.probeConfig.ProbeOnRegister {
		go r.ProbeAIProvider(name)
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
