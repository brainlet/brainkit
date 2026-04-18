package server_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/gateway"
	"github.com/brainlet/brainkit/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	return ln.Addr().String()
}

// TestNewRejectsMissingFields checks the validator.
func TestNewRejectsMissingFields(t *testing.T) {
	_, err := server.New(server.Config{})
	require.Error(t, err)

	_, err = server.New(server.Config{Namespace: "x"})
	require.Error(t, err)

	_, err = server.New(server.Config{
		Namespace: "x",
		Transport: brainkit.EmbeddedNATS(),
	})
	require.Error(t, err, "FSRoot required")

	_, err = server.New(server.Config{
		Namespace: "x",
		Transport: brainkit.EmbeddedNATS(),
		FSRoot:    t.TempDir(),
	})
	require.Error(t, err, "Gateway.Listen required")
}

// TestStartStopLifecycle boots a server and tears it down. No
// goroutine leak checks at this level — just that the lifecycle
// completes cleanly.
func TestStartStopLifecycle(t *testing.T) {
	tmp := t.TempDir()
	addr := freePort(t)

	srv, err := server.New(server.Config{
		Namespace: "server-lifecycle",
		Transport: brainkit.EmbeddedNATS(),
		FSRoot:    tmp,
		Gateway:   gateway.Config{Listen: addr},
	})
	require.NoError(t, err)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()

	// Give the gateway time to bind.
	time.Sleep(500 * time.Millisecond)

	cancel()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	require.NoError(t, srv.Stop(stopCtx))

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Start did not return after cancel")
	}
}

// TestKitAccessors verifies Kit() exposes the composed runtime and
// the accessor surface added in session 10.
func TestKitAccessors(t *testing.T) {
	tmp := t.TempDir()
	addr := freePort(t)

	srv, err := server.New(server.Config{
		Namespace: "server-accessors",
		Transport: brainkit.EmbeddedNATS(),
		FSRoot:    tmp,
		Gateway:   gateway.Config{Listen: addr},
	})
	require.NoError(t, err)
	defer srv.Close()

	kit := srv.Kit()
	require.NotNil(t, kit)
	require.NotNil(t, kit.Providers())
	require.NotNil(t, kit.Secrets())
	assert.Equal(t, "server-accessors", kit.Namespace())
}

// TestLoadConfig round-trips the example YAML + verifies env
// substitution pulls a value out of the process environment.
func TestLoadConfig(t *testing.T) {
	tmp := t.TempDir()
	yamlPath := filepath.Join(tmp, "config.yaml")
	content := `namespace: loaded
fs_root: ` + tmp + `
transport:
  type: embedded
gateway:
  listen: ` + freePort(t) + `
secret_key: $TEST_SECRET
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(content), 0644))

	t.Setenv("TEST_SECRET", "opened-sesame")

	cfg, err := server.LoadConfig(yamlPath)
	require.NoError(t, err)
	assert.Equal(t, "loaded", cfg.Namespace)
	assert.Equal(t, "opened-sesame", cfg.SecretKey)
	assert.Equal(t, tmp, cfg.FSRoot)
	assert.NotEqual(t, brainkit.TransportConfig{}, cfg.Transport)
}
