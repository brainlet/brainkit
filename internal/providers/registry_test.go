package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIProvider_RegisterListUnregister(t *testing.T) {
	r := New(ProbeConfig{})

	err := r.RegisterAIProvider("openai", AIProviderRegistration{
		Type:   AIProviderOpenAI,
		Config: OpenAIProviderConfig{APIKey: "sk-test", Organization: "org-123"},
	})
	require.NoError(t, err)

	// List
	providers := r.ListAIProviders()
	require.Len(t, providers, 1)
	assert.Equal(t, "openai", providers[0].Name)
	assert.Equal(t, AIProviderOpenAI, providers[0].Type)
	assert.True(t, providers[0].Capabilities.Chat)
	assert.True(t, providers[0].Capabilities.Embedding)
	assert.False(t, providers[0].Healthy, "should not be healthy before probe")
	assert.Equal(t, "probe pending", providers[0].LastError)
	assert.True(t, providers[0].LastProbed.IsZero(), "should not have been probed")

	// Has
	assert.True(t, r.HasAIProvider("openai"))
	assert.False(t, r.HasAIProvider("anthropic"))

	// Get
	reg, ok := r.GetAIProvider("openai")
	require.True(t, ok)
	assert.Equal(t, AIProviderOpenAI, reg.Type)
	cfg, ok := reg.Config.(OpenAIProviderConfig)
	require.True(t, ok)
	assert.Equal(t, "sk-test", cfg.APIKey)
	assert.Equal(t, "org-123", cfg.Organization)

	// Unregister
	r.UnregisterAIProvider("openai")
	assert.False(t, r.HasAIProvider("openai"))
	assert.Empty(t, r.ListAIProviders())
}

func TestVectorStore_RegisterListUnregister(t *testing.T) {
	r := New(ProbeConfig{})

	err := r.RegisterVectorStore("main", VectorStoreRegistration{
		Type:   VectorStorePg,
		Config: PgVectorConfig{ConnectionString: "postgres://localhost/test"},
	})
	require.NoError(t, err)

	stores := r.ListVectorStores()
	require.Len(t, stores, 1)
	assert.Equal(t, "main", stores[0].Name)
	assert.Equal(t, VectorStorePg, stores[0].Type)
	assert.True(t, stores[0].Capabilities.CreateIndex)
	assert.True(t, stores[0].Capabilities.Query)

	reg, ok := r.GetVectorStore("main")
	require.True(t, ok)
	cfg := reg.Config.(PgVectorConfig)
	assert.Equal(t, "postgres://localhost/test", cfg.ConnectionString)

	r.UnregisterVectorStore("main")
	assert.False(t, r.HasVectorStore("main"))
}

func TestStorage_RegisterListUnregister(t *testing.T) {
	r := New(ProbeConfig{})

	err := r.RegisterStorage("default", StorageRegistration{
		Type:   StorageInMemory,
		Config: InMemoryStorageConfig{},
	})
	require.NoError(t, err)

	storages := r.ListStorages()
	require.Len(t, storages, 1)
	assert.Equal(t, "default", storages[0].Name)
	assert.Equal(t, StorageInMemory, storages[0].Type)
	assert.True(t, storages[0].Capabilities.Memory)

	r.UnregisterStorage("default")
	assert.Empty(t, r.ListStorages())
}

func TestRegister_EmptyName(t *testing.T) {
	r := New(ProbeConfig{})

	err := r.RegisterAIProvider("", AIProviderRegistration{Type: AIProviderOpenAI})
	assert.Error(t, err)

	err = r.RegisterVectorStore("", VectorStoreRegistration{Type: VectorStorePg})
	assert.Error(t, err)

	err = r.RegisterStorage("", StorageRegistration{Type: StorageInMemory})
	assert.Error(t, err)
}

func TestMultipleRegistrations(t *testing.T) {
	r := New(ProbeConfig{})

	r.RegisterAIProvider("openai", AIProviderRegistration{Type: AIProviderOpenAI, Config: OpenAIProviderConfig{APIKey: "sk-1"}})
	r.RegisterAIProvider("anthropic", AIProviderRegistration{Type: AIProviderAnthropic, Config: AnthropicProviderConfig{APIKey: "sk-2"}})
	r.RegisterVectorStore("vectors", VectorStoreRegistration{Type: VectorStoreLibSQL, Config: LibSQLVectorConfig{URL: "libsql://test"}})
	r.RegisterStorage("store", StorageRegistration{Type: StoragePostgres, Config: PostgresStorageConfig{ConnectionString: "pg://test"}})

	assert.Len(t, r.ListAIProviders(), 2)
	assert.Len(t, r.ListVectorStores(), 1)
	assert.Len(t, r.ListStorages(), 1)

	// Anthropic should not have embedding
	providers := r.ListAIProviders()
	for _, p := range providers {
		if p.Type == AIProviderAnthropic {
			assert.False(t, p.Capabilities.Embedding)
			assert.True(t, p.Capabilities.Chat)
		}
	}
}

func TestKnownCapabilities(t *testing.T) {
	// OpenAI has everything except reranking
	openai := KnownAICapabilities(AIProviderOpenAI)
	assert.True(t, openai.Chat)
	assert.True(t, openai.Embedding)
	assert.True(t, openai.Image)
	assert.True(t, openai.Transcription)
	assert.True(t, openai.Speech)
	assert.False(t, openai.Reranking)

	// Cohere has chat + embedding + reranking
	cohere := KnownAICapabilities(AIProviderCohere)
	assert.True(t, cohere.Chat)
	assert.True(t, cohere.Embedding)
	assert.True(t, cohere.Reranking)
	assert.False(t, cohere.Image)

	// Unknown defaults to chat only
	unknown := KnownAICapabilities("unknown-provider")
	assert.True(t, unknown.Chat)
	assert.False(t, unknown.Embedding)
}
