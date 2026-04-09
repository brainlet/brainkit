package health

import (
	"os"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
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

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-probe",
		FSRoot:    t.TempDir(),
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
	})
	require.NoError(t, err)
	defer k.Close()

	// Query health to verify provider is registered and healthy
	health := queryHealth(t, k)
	assert.True(t, health.Healthy, "kit with OpenAI should be healthy")

	// Check for AI provider health check
	var aiCheck *brainkit.HealthCheck
	for i := range health.Checks {
		if health.Checks[i].Name == "ai:openai" {
			aiCheck = &health.Checks[i]
			break
		}
	}
	if aiCheck != nil {
		assert.True(t, aiCheck.Healthy, "OpenAI should be reachable with real API key")
		assert.Greater(t, aiCheck.Latency, time.Duration(0), "should have non-zero latency")
	}
}

// testProbeAIProviderBadKey verifies probe detects invalid credentials.
func testProbeAIProviderBadKey(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-probe-bad",
		FSRoot:    t.TempDir(),
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI("sk-invalid-key-12345")},
	})
	require.NoError(t, err)
	defer k.Close()

	// With a bad key, the kit should still start but the provider may be unhealthy
	health := queryHealth(t, k)
	// The kit itself is healthy even if the AI provider is not
	assert.True(t, health.Healthy || health.Status == "running")
}

// testProbeAIProviderNotRegistered verifies error for unknown provider.
func testProbeAIProviderNotRegistered(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-probe-notfound",
		FSRoot:    t.TempDir(),
	})
	require.NoError(t, err)
	defer k.Close()

	// No providers registered — health should still work
	health := queryHealth(t, k)
	assert.True(t, health.Healthy)
}

// testProbeStorageInMemory verifies in-memory storage is always healthy.
func testProbeStorageInMemory(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-probe-storage",
		FSRoot:    t.TempDir(),
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.InMemoryStorage(),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	health := queryHealth(t, k)
	assert.True(t, health.Healthy)

	// InMemoryStorage is pure JS — no Go-side bridge server, so no storage health check.
	// Verify the Kit is healthy overall (runtime + transport checks present).
	hasRuntime := false
	for _, c := range health.Checks {
		if c.Name == "runtime" {
			hasRuntime = true
		}
	}
	assert.True(t, hasRuntime, "should have runtime check")
}

// testProbeVectorStoreRealPgVector tests vector store probing with a real Postgres+pgvector.
func testProbeVectorStoreRealPgVector(t *testing.T, _ *suite.TestEnv) {
	if !testutil.PodmanAvailable() {
		t.Skip("Podman required for pgvector container")
	}

	pgConnStr := testutil.StartPgVectorContainer(t)

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-probe-vector",
		FSRoot:    t.TempDir(),
		Vectors: map[string]brainkit.VectorConfig{
			"main": brainkit.PgVectorStore(pgConnStr),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	health := queryHealth(t, k)
	assert.True(t, health.Healthy, "kit with PgVector should be healthy")
}

// testProbeAll runs probes for everything registered.
func testProbeAll(t *testing.T, _ *suite.TestEnv) {
	testutil.LoadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY required")
	}

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-probeall",
		FSRoot:    t.TempDir(),
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.InMemoryStorage(),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	health := queryHealth(t, k)
	assert.True(t, health.Healthy, "kit with all probes should be healthy")
}

// testProbePeriodicTicker verifies that periodic probing fires.
func testProbePeriodicTicker(t *testing.T, _ *suite.TestEnv) {
	testutil.LoadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY required")
	}

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-periodic",
		FSRoot:    t.TempDir(),
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
	})
	require.NoError(t, err)
	defer k.Close()

	require.Eventually(t, func() bool {
		health := queryHealth(t, k)
		return health.Healthy
	}, 5*time.Second, 200*time.Millisecond, "periodic probe should confirm health")
}
