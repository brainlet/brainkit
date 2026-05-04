package providerreg

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
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

// DecodeAIProviderConfig decodes the typed provider config payload used by both
// Go registry commands and JS registry control bridges.
func DecodeAIProviderConfig(typ string, raw json.RawMessage) (any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	switch typ {
	case "openai":
		var c types.OpenAIProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "anthropic":
		var c types.AnthropicProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "google":
		var c types.GoogleProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "mistral":
		var c types.MistralProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "cohere":
		var c types.CohereProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "groq":
		var c types.GroqProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "perplexity":
		var c types.PerplexityProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "deepseek":
		var c types.DeepSeekProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "fireworks":
		var c types.FireworksProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "togetherai":
		var c types.TogetherAIProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "xai":
		var c types.XAIProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "azure":
		var c types.AzureProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "bedrock":
		var c types.BedrockProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "vertex":
		var c types.VertexProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "huggingface":
		var c types.HuggingFaceProviderConfig
		return c, json.Unmarshal(raw, &c)
	case "cerebras":
		var c types.CerebrasProviderConfig
		return c, json.Unmarshal(raw, &c)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", typ)
	}
}

// Re-export constants and functions from types
var (
	KnownAICapabilities        = types.KnownAICapabilities
	DefaultVectorCapabilities  = types.DefaultVectorCapabilities
	DefaultStorageCapabilities = types.DefaultStorageCapabilities
)

// Re-export AI provider type constants
const (
	AIProviderOpenAI      = types.AIProviderOpenAI
	AIProviderAnthropic   = types.AIProviderAnthropic
	AIProviderGoogle      = types.AIProviderGoogle
	AIProviderMistral     = types.AIProviderMistral
	AIProviderCohere      = types.AIProviderCohere
	AIProviderGroq        = types.AIProviderGroq
	AIProviderPerplexity  = types.AIProviderPerplexity
	AIProviderDeepSeek    = types.AIProviderDeepSeek
	AIProviderFireworks   = types.AIProviderFireworks
	AIProviderTogetherAI  = types.AIProviderTogetherAI
	AIProviderXAI         = types.AIProviderXAI
	AIProviderAzure       = types.AIProviderAzure
	AIProviderBedrock     = types.AIProviderBedrock
	AIProviderVertex      = types.AIProviderVertex
	AIProviderHuggingFace = types.AIProviderHuggingFace
	AIProviderCerebras    = types.AIProviderCerebras
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
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	activeProbes atomic.Int64
	closed       bool
	closeWait    sync.Once
	closeDone    chan struct{}
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
	ctx, cancel := context.WithCancel(context.Background())
	return &ProviderRegistry{
		aiProviders:  make(map[string]*entry),
		vectorStores: make(map[string]*entry),
		storages:     make(map[string]*entry),
		probeConfig:  cfg,
		ctx:          ctx,
		cancel:       cancel,
		closeDone:    make(chan struct{}),
	}
}

// Close cancels registry-owned asynchronous probes and waits for them to exit.
func (r *ProviderRegistry) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return r.CloseContext(ctx)
}

// CloseContext cancels registry-owned asynchronous probes and waits for them
// under caller-owned lifecycle cancellation.
func (r *ProviderRegistry) CloseContext(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		if r.cancel != nil {
			r.cancel()
		}
	}
	closeDone := r.closeDone
	r.closeWait.Do(func() {
		go func() {
			r.wg.Wait()
			close(closeDone)
		}()
	})
	r.mu.Unlock()
	select {
	case <-closeDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ActiveProbes reports registry-owned async probe goroutines. It is intended
// for lifecycle tests and diagnostics, not for application control flow.
func (r *ProviderRegistry) ActiveProbes() int64 {
	if r == nil {
		return 0
	}
	return r.activeProbes.Load()
}

// --- AI Providers ---

func (r *ProviderRegistry) RegisterAIProvider(name string, reg AIProviderRegistration) error {
	if name == "" {
		return &sdk.ValidationError{Field: "name", Message: "provider name is required"}
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return fmt.Errorf("provider registry is closed")
	}
	r.aiProviders[name] = &entry{
		registration: reg,
		lastErr:      "probe pending",
	}
	if r.probeConfig.ProbeOnRegister {
		r.startAIProbeLocked(name)
	}
	r.mu.Unlock()
	return nil
}

func (r *ProviderRegistry) startAIProbeLocked(name string) {
	r.wg.Add(1)
	r.activeProbes.Add(1)
	go func() {
		defer r.wg.Done()
		defer r.activeProbes.Add(-1)
		r.ProbeAIProviderContext(r.ctx, name)
	}()
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

// UpdateAIProvider applies an in-place registration update while preserving the
// registry entry owner. It returns false when the provider is not registered.
func (r *ProviderRegistry) UpdateAIProvider(name string, update func(AIProviderRegistration) (AIProviderRegistration, error)) (bool, error) {
	if name == "" {
		return false, &sdk.ValidationError{Field: "name", Message: "provider name is required"}
	}
	if update == nil {
		return false, fmt.Errorf("provider update function is required")
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return false, fmt.Errorf("provider registry is closed")
	}
	e, ok := r.aiProviders[name]
	if !ok {
		r.mu.Unlock()
		return false, nil
	}
	reg := e.registration.(AIProviderRegistration)
	next, err := update(reg)
	if err != nil {
		r.mu.Unlock()
		return true, err
	}
	e.registration = next
	e.healthy = false
	e.lastProbed = time.Time{}
	e.latency = 0
	e.lastErr = "probe pending"
	if r.probeConfig.ProbeOnRegister {
		r.startAIProbeLocked(name)
	}
	r.mu.Unlock()
	return true, nil
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
	if r.closed {
		return fmt.Errorf("provider registry is closed")
	}
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
	if r.closed {
		return fmt.Errorf("provider registry is closed")
	}
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

// Additional re-exports
type QdrantVectorConfig = types.QdrantVectorConfig
type InMemoryStorageConfig = types.InMemoryStorageConfig

const VectorStoreQdrant = types.VectorStoreQdrant
