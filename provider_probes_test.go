package brainkit

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	probesmod "github.com/brainlet/brainkit/modules/probes"
	"github.com/stretchr/testify/require"
)

func TestProviderProbesAreScheduledByProbesModule(t *testing.T) {
	t.Run("core does not start provider probes", func(t *testing.T) {
		server, hits := newProviderProbeServer()
		defer server.Close()

		k, err := New(Config{
			Transport: Memory(),
			Namespace: "probe-boundary-core",
			CallerID:  "probe-boundary-core",
			Providers: []ProviderConfig{
				OpenAI("sk-test", WithBaseURL(server.URL)),
			},
		})
		require.NoError(t, err)
		defer k.Close()

		require.Never(t, func() bool {
			return hits.Load() > 0
		}, 200*time.Millisecond, 10*time.Millisecond)

		providers := k.kernel.ProviderRegistry().ListAIProviders()
		require.Len(t, providers, 1)
		require.True(t, providers[0].LastProbed.IsZero())
		require.Equal(t, "probe pending", providers[0].LastError)
	})

	t.Run("probes module owns initial provider sweep", func(t *testing.T) {
		server, hits := newProviderProbeServer()
		defer server.Close()

		mod := probesmod.New(probesmod.Config{Interval: 0})
		k, err := New(Config{
			Transport: Memory(),
			Namespace: "probe-boundary-module",
			CallerID:  "probe-boundary-module",
			Providers: []ProviderConfig{
				OpenAI("sk-test", WithBaseURL(server.URL)),
			},
			Modules: []bkmodule.Module{mod},
		})
		require.NoError(t, err)
		defer k.Close()

		require.Eventually(t, func() bool {
			return hits.Load() > 0
		}, time.Second, 10*time.Millisecond)

		require.Eventually(t, func() bool {
			providers := k.kernel.ProviderRegistry().ListAIProviders()
			return len(providers) == 1 && providers[0].Healthy && !providers[0].LastProbed.IsZero()
		}, time.Second, 10*time.Millisecond)

		require.True(t, mod.DebugSnapshot().RunnerAttached)
	})
}

func newProviderProbeServer() (*httptest.Server, *atomic.Int64) {
	var hits atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	return server, &hits
}
