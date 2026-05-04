package server_test

import (
	"context"
	bkmodule "github.com/brainlet/brainkit/module"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/gateway"
	"github.com/brainlet/brainkit/server"
	"github.com/brainlet/brainkit/server/configfile"
	"github.com/brainlet/brainkit/transports"
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
		Transport: transports.EmbeddedNATS(),
	})
	require.Error(t, err, "FSRoot required")

	_, err = server.New(server.Config{
		Namespace: "x",
		Transport: transports.EmbeddedNATS(),
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
		Transport: transports.EmbeddedNATS(),
		FSRoot:    tmp,
		Modules: []bkmodule.Module{
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

// TestKit exposes the composed runtime.
func TestKit(t *testing.T) {
	tmp := t.TempDir()
	addr := freePort(t)

	srv, err := server.New(server.Config{
		Namespace: "server-accessors",
		Transport: transports.EmbeddedNATS(),
		FSRoot:    tmp,
		Modules: []bkmodule.Module{
			gateway.New(gateway.Config{Listen: addr}),
		},
	})
	require.NoError(t, err)
	defer srv.Close()

	kit := srv.Kit()
	require.NotNil(t, kit)
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

	cfg, err := configfile.Load(yamlPath)
	require.NoError(t, err)
	assert.Equal(t, "loaded", cfg.Namespace)
	assert.Equal(t, "opened-sesame", cfg.SecretKey)
	assert.Equal(t, tmp, cfg.FSRoot)
	assert.NotEqual(t, brainkit.TransportConfig{}, cfg.Transport)
	require.NotNil(t, cfg.Store)
	t.Cleanup(func() { _ = cfg.Store.Close() })

	var haveGateway bool
	for _, m := range cfg.Modules {
		if m != nil && m.ID() == "gateway" {
			haveGateway = true
		}
	}
	assert.True(t, haveGateway, "modules.gateway should produce a gateway module")
}

// TestLoadConfigLegacyTopLevelGateway refuses the pre-registry YAML
// shape loudly. Silently ignoring `gateway:` at the root would make
// a user's previous config shape boot with an empty module list and fail
// validation with a confusing "gateway is required" error even
// though the key is visibly in the file.
func TestLoadConfigPreviousTopLevelGateway(t *testing.T) {
	tmp := t.TempDir()
	yamlPath := filepath.Join(tmp, "config.yaml")
	content := `namespace: previous
fs_root: ` + tmp + `
transport:
  type: embedded
gateway:
  listen: ` + freePort(t) + `
`
	require.NoError(t, os.WriteFile(yamlPath, []byte(content), 0644))

	_, err := configfile.Load(yamlPath)
	require.Error(t, err, "previous top-level gateway must be rejected")
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

	_, err := configfile.Load(yamlPath)
	require.Error(t, err, "expected unknown-module error")
	assert.Contains(t, err.Error(), "billig")
	assert.Contains(t, err.Error(), "registered")
}
