package auth_test

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/campaigns"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestLibSQL_EmbeddedNoAuth — libsql embedded with no auth.
// Ported from test/auth/auth_test.go:TestLibSQL_EmbeddedNoAuth.
func TestLibSQL_EmbeddedNoAuth(t *testing.T) {
	k := newKit(t, nil)

	result := evalStore(t, k, "libsql-embedded", `
		var store = storage("default");
	`)
	require.Contains(t, result, `"ok":true`)
}

// TestLibSQL_ContainerNoAuth — libsql container with no auth.
// Ported from test/auth/auth_test.go:TestLibSQL_ContainerNoAuth.
func TestLibSQL_ContainerNoAuth(t *testing.T) {
	campaigns.RequirePodman(t)
	testutil.CleanupOrphanedContainers(t)

	addr := testutil.StartContainer(t,
		"ghcr.io/tursodatabase/libsql-server:latest",
		"8080/tcp",
		[]string{"sqld", "--http-listen-addr", "0.0.0.0:8080"},
		wait.ForHTTP("/health").WithStartupTimeout(30*time.Second))

	k := newKit(t, map[string]string{
		"LIBSQL_URL": "http://" + addr,
	})

	result := evalStore(t, k, "libsql-container", `
		var store = new embed.LibSQLStore({
			id: "libsql-container-test",
			url: process.env.LIBSQL_URL,
		});
	`)
	require.Contains(t, result, `"ok":true`)
}
