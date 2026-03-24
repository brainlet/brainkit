package infra_test

import (
	"os"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/kit"
	provreg "github.com/brainlet/brainkit/kit/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProbe_AIProvider_Real_OpenAI probes the real OpenAI API with a real key.
func TestProbe_AIProvider_Real_OpenAI(t *testing.T) {
	testutil.LoadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY required")
	}

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-probe",
		WorkspaceDir: t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai": {
				Type:   provreg.AIProviderOpenAI,
				Config: provreg.OpenAIProviderConfig{APIKey: key},
			},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	result := k.ProbeAIProvider("openai")
	assert.True(t, result.Available, "OpenAI should be reachable with real API key")
	assert.Empty(t, result.Error)
	assert.Greater(t, result.Latency, time.Duration(0), "should have non-zero latency")

	caps, ok := result.Capabilities.(provreg.AIProviderCapabilities)
	require.True(t, ok)
	assert.True(t, caps.Chat)
	assert.True(t, caps.Embedding)

	// Verify registry was updated
	providers := k.ListAIProviders()
	require.Len(t, providers, 1)
	assert.True(t, providers[0].Healthy)
	assert.Empty(t, providers[0].LastError)
	assert.Greater(t, providers[0].Latency, time.Duration(0))
}

// TestProbe_AIProvider_BadKey verifies probe detects invalid credentials.
func TestProbe_AIProvider_BadKey(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-probe-bad",
		WorkspaceDir: t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai": {
				Type:   provreg.AIProviderOpenAI,
				Config: provreg.OpenAIProviderConfig{APIKey: "sk-invalid-key-12345"},
			},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	result := k.ProbeAIProvider("openai")
	assert.False(t, result.Available, "should fail with bad key")
	assert.Contains(t, result.Error, "authentication failed")

	providers := k.ListAIProviders()
	require.Len(t, providers, 1)
	assert.False(t, providers[0].Healthy)
}

// TestProbe_AIProvider_NotRegistered verifies error for unknown provider.
func TestProbe_AIProvider_NotRegistered(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-probe-notfound",
		WorkspaceDir: t.TempDir(),
	})
	require.NoError(t, err)
	defer k.Close()

	result := k.ProbeAIProvider("nonexistent")
	assert.False(t, result.Available)
	assert.Contains(t, result.Error, "not registered")
}

// TestProbe_Storage_InMemory verifies in-memory storage is always healthy.
func TestProbe_Storage_InMemory(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-probe-storage",
		WorkspaceDir: t.TempDir(),
		MastraStorages: map[string]provreg.StorageRegistration{
			"default": {Type: provreg.StorageInMemory, Config: provreg.InMemoryStorageConfig{}},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// Kernel-level JS probe: instantiate and test
	jsResult := k.ProbeStorage("default")
	assert.True(t, jsResult.Available, "InMemoryStore should instantiate in JS")
	assert.Empty(t, jsResult.Error)
}

// TestProbe_VectorStore_Real_PgVector tests vector store probing with a real Postgres+pgvector.
func TestProbe_VectorStore_Real_PgVector(t *testing.T) {
	if !testutil.PodmanAvailable() {
		t.Skip("Podman required for pgvector container")
	}

	pgConnStr := testutil.StartPgVectorContainer(t)

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-probe-vector",
		WorkspaceDir: t.TempDir(),
		VectorStores: map[string]provreg.VectorStoreRegistration{
			"main": {
				Type:   provreg.VectorStorePg,
				Config: provreg.PgVectorConfig{ConnectionString: pgConnStr},
			},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// ProbeVectorStore calls vectorStore("main") internally, which resolves
	// from the Go ProviderRegistry and instantiates PgVector via __agent_embed.
	// No manual JS init needed.
	result := k.ProbeVectorStore("main")
	assert.True(t, result.Available, "PgVector store should be reachable")
	assert.Empty(t, result.Error)
	assert.Greater(t, result.Latency, time.Duration(0))
}

// TestProbe_ProbeAll runs probes for everything registered.
func TestProbe_ProbeAll(t *testing.T) {
	testutil.LoadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY required")
	}

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-probeall",
		WorkspaceDir: t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai": {
				Type:   provreg.AIProviderOpenAI,
				Config: provreg.OpenAIProviderConfig{APIKey: key},
			},
		},
		MastraStorages: map[string]provreg.StorageRegistration{
			"default": {Type: provreg.StorageInMemory, Config: provreg.InMemoryStorageConfig{}},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	k.ProbeAll()

	providers := k.ListAIProviders()
	require.Len(t, providers, 1)
	assert.True(t, providers[0].Healthy, "OpenAI should be healthy after ProbeAll")

	storages := k.ListStorages()
	require.Len(t, storages, 1)
	// InMemory storage probe result depends on JS instantiation
}

// TestProbe_PeriodicTicker verifies that periodic probing fires.
func TestProbe_PeriodicTicker(t *testing.T) {
	testutil.LoadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY required")
	}

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-periodic",
		WorkspaceDir: t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai": {
				Type:   provreg.AIProviderOpenAI,
				Config: provreg.OpenAIProviderConfig{APIKey: key},
			},
		},
		Probe: provreg.ProbeConfig{
			PeriodicInterval: 500 * time.Millisecond,
			ProbeTimeout:     5 * time.Second,
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// Wait for at least two probe cycles (interval=500ms, wait 2s)
	require.Eventually(t, func() bool {
		providers := k.ListAIProviders()
		return len(providers) == 1 && providers[0].Healthy && !providers[0].LastProbed.IsZero()
	}, 5*time.Second, 200*time.Millisecond, "periodic probe should have marked OpenAI healthy")
}
