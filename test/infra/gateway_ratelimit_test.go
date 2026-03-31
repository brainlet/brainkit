package infra_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/kit/gateway"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateway_RateLimiting(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	gw := gateway.New(k, gateway.Config{
		Listen:   ":0",
		NoHealth: true,
		RateLimit: &gateway.RateLimitConfig{
			RequestsPerSecond: 2,
			Burst:             2,
		},
	})
	gw.HandleWebhook("POST", "/test", "gateway.ratelimit.test")
	require.NoError(t, gw.Start())
	defer gw.Stop()

	url := "http://" + gw.Addr() + "/test"

	// First 2 requests should succeed (burst capacity)
	for i := 0; i < 2; i++ {
		resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode, "request %d should succeed", i)
		resp.Body.Close()
	}

	// Third request should be rate limited (burst exhausted, no time to refill)
	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	require.NoError(t, err)
	assert.Equal(t, 429, resp.StatusCode, "third request should be rate limited")
	resp.Body.Close()
}
