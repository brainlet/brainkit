package health

import (
	"os"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	provreg "github.com/brainlet/brainkit/registry"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testProbeAIProviderRealOpenAI probes the real OpenAI API with a real key.
func testProbeAIProviderRealOpenAI(t *testing.T, _ *suite.TestEnv) {
	testutil.LoadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY required")
	}

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test",
		CallerID:  "test-probe",
		FSRoot:    t.TempDir(),
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

// testProbeAIProviderBadKey verifies probe detects invalid credentials.
func testProbeAIProviderBadKey(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test",
		CallerID:  "test-probe-bad",
		FSRoot:    t.TempDir(),
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

// testProbeAIProviderNotRegistered verifies error for unknown provider.
func testProbeAIProviderNotRegistered(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test",
		CallerID:  "test-probe-notfound",
		FSRoot:    t.TempDir(),
	})
	require.NoError(t, err)
	defer k.Close()

	result := k.ProbeAIProvider("nonexistent")
	assert.False(t, result.Available)
	assert.Contains(t, result.Error, "not registered")
}

// testProbeStorageInMemory verifies in-memory storage is always healthy.
func testProbeStorageInMemory(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test",
		CallerID:  "test-probe-storage",
		FSRoot:    t.TempDir(),
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.InMemoryStorage(),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	jsResult := k.ProbeStorage("default")
	assert.True(t, jsResult.Available, "InMemoryStore should instantiate in JS")
	assert.Empty(t, jsResult.Error)
}

// testProbeVectorStoreRealPgVector tests vector store probing with a real Postgres+pgvector.
func testProbeVectorStoreRealPgVector(t *testing.T, _ *suite.TestEnv) {
	if !testutil.PodmanAvailable() {
		t.Skip("Podman required for pgvector container")
	}

	pgConnStr := testutil.StartPgVectorContainer(t)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test",
		CallerID:  "test-probe-vector",
		FSRoot:    t.TempDir(),
		Vectors: map[string]brainkit.VectorConfig{
			"main": brainkit.PgVectorStore(pgConnStr),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	result := k.ProbeVectorStore("main")
	assert.True(t, result.Available, "PgVector store should be reachable")
	assert.Empty(t, result.Error)
	assert.Greater(t, result.Latency, time.Duration(0))
}

// testProbeAll runs probes for everything registered.
func testProbeAll(t *testing.T, _ *suite.TestEnv) {
	testutil.LoadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY required")
	}

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test",
		CallerID:  "test-probeall",
		FSRoot:    t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai": {
				Type:   provreg.AIProviderOpenAI,
				Config: provreg.OpenAIProviderConfig{APIKey: key},
			},
		},
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.InMemoryStorage(),
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
}

// testProbePeriodicTicker verifies that periodic probing fires.
func testProbePeriodicTicker(t *testing.T, _ *suite.TestEnv) {
	testutil.LoadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY required")
	}

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test",
		CallerID:  "test-periodic",
		FSRoot:    t.TempDir(),
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

	require.Eventually(t, func() bool {
		providers := k.ListAIProviders()
		return len(providers) == 1 && providers[0].Healthy && !providers[0].LastProbed.IsZero()
	}, 5*time.Second, 200*time.Millisecond, "periodic probe should have marked OpenAI healthy")
}
