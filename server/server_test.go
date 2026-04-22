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
	require.Error(t, err, "gateway module required")
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
		Modules: []brainkit.Module{
			gateway.New(gateway.Config{Listen: addr}),
		},
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
		Modules: []brainkit.Module{
			gateway.New(gateway.Config{Listen: addr}),
		},
	})
	require.NoError(t, err)
	defer srv.Close()

	kit := srv.Kit()
	require.NotNil(t, kit)
	require.NotNil(t, kit.Providers())
	require.NotNil(t, kit.Secrets())
	assert.Equal(t, "server-accessors", kit.Namespace())
}

// TestLoadConfig round-trips a minimal YAML + verifies env
// substitution pulls a value out of the process environment and that
// the registry-driven module path produces a gateway module from
// `modules.gateway:`.
func TestLoadConfig(t *testing.T) {
	tmp := t.TempDir()
	yamlPath := filepath.Join(tmp, "config.yaml")
	content := `namespace: loaded
fs_root: ` + tmp + `
transport:
  type: embedded
secret_key: $TEST_SECRET
modules:
  gateway:
    listen: ` + freePort(t) + `
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(content), 0644))

	t.Setenv("TEST_SECRET", "opened-sesame")

	cfg, err := server.LoadConfig(yamlPath)
	require.NoError(t, err)
	assert.Equal(t, "loaded", cfg.Namespace)
	assert.Equal(t, "opened-sesame", cfg.SecretKey)
	assert.Equal(t, tmp, cfg.FSRoot)
	assert.NotEqual(t, brainkit.TransportConfig{}, cfg.Transport)

	var haveGateway bool
	for _, m := range cfg.Modules {
		if m != nil && m.Name() == "gateway" {
			haveGateway = true
		}
	}
	assert.True(t, haveGateway, "modules.gateway should produce a gateway module")
}

// TestLoadConfigLegacyTopLevelGateway refuses the pre-registry YAML
// shape loudly. Silently ignoring `gateway:` at the root would make
// a user's old config boot with an empty module list and fail
// validation with a confusing "gateway is required" error even
// though the key is visibly in the file.
func TestLoadConfigLegacyTopLevelGateway(t *testing.T) {
	tmp := t.TempDir()
	yamlPath := filepath.Join(tmp, "config.yaml")
	content := `namespace: legacy
fs_root: ` + tmp + `
transport:
  type: embedded
gateway:
  listen: ` + freePort(t) + `
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(content), 0644))

	_, err := server.LoadConfig(yamlPath)
	require.Error(t, err, "legacy top-level gateway must be rejected")
	assert.Contains(t, err.Error(), "modules.gateway")
}

// TestLoadConfigUnknownModule surfaces typos loudly — the whole
// point of the registry is that `modules.billig:` (for billing)
// fails at load rather than silently skipping the module.
func TestLoadConfigUnknownModule(t *testing.T) {
	tmp := t.TempDir()
	yamlPath := filepath.Join(tmp, "config.yaml")
	content := `namespace: oops
fs_root: ` + tmp + `
transport:
  type: embedded
modules:
  gateway:
    listen: ` + freePort(t) + `
  billig: {}
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(content), 0644))

	_, err := server.LoadConfig(yamlPath)
	require.Error(t, err, "expected unknown-module error")
	assert.Contains(t, err.Error(), "billig")
	assert.Contains(t, err.Error(), "registered")
}
