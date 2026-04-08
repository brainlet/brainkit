package auth_test

import (
	"os"
	"testing"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestUpstash_TokenAuth — Upstash Redis with token auth.
// Ported from test/auth/auth_test.go:TestUpstash_TokenAuth.
func TestUpstash_TokenAuth(t *testing.T) {
	testutil.LoadEnv(t)
	url := os.Getenv("UPSTASH_REDIS_REST_URL")
	token := os.Getenv("UPSTASH_REDIS_REST_TOKEN")
	if url == "" || token == "" {
		t.Skip("needs UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN in .env")
	}

	k := newKit(t, map[string]string{
		"UPSTASH_REDIS_REST_URL":   url,
		"UPSTASH_REDIS_REST_TOKEN": token,
	})

	result := evalStore(t, k, "upstash-token", `
		var store = new embed.UpstashStore({
			id: "upstash-auth-test",
			url: process.env.UPSTASH_REDIS_REST_URL,
			token: process.env.UPSTASH_REDIS_REST_TOKEN,
		});
	`)
	require.Contains(t, result, `"ok":true`)
}

// TestInMemory_NoAuth — InMemory store (no auth, baseline).
// Ported from test/auth/auth_test.go:TestInMemory_NoAuth.
func TestInMemory_NoAuth(t *testing.T) {
	k := newKit(t, map[string]string{})

	result := evalStore(t, k, "inmemory", `
		var store = new embed.InMemoryStore();
	`)
	require.Contains(t, result, `"ok":true`)
}
