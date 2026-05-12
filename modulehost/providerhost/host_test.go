package providerhost

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
	"github.com/stretchr/testify/require"
)

func TestAutoDetectProvidersUsesEnvVarsAndHonorsExplicitMap(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "from-os-env")

	cfg := types.KernelConfig{
		EnvVars: map[string]string{"OPENAI_API_KEY": "from-config-env"},
	}
	AutoDetectProviders(&cfg)

	reg, ok := cfg.AIProviders["openai"]
	if !ok {
		t.Fatalf("expected openai provider to be auto-detected")
	}
	creds := ExtractProviderCredentials(reg)
	if creds.APIKey != "from-config-env" {
		t.Fatalf("detected API key = %q, want EnvVars override", creds.APIKey)
	}

	explicit := types.KernelConfig{AIProviders: map[string]types.AIProviderRegistration{}}
	AutoDetectProviders(&explicit)
	if len(explicit.AIProviders) != 0 {
		t.Fatalf("explicit empty provider map must disable auto-detection, got %d providers", len(explicit.AIProviders))
	}
}

func TestManagerRegistersProvidersAndRefreshesMappedSecret(t *testing.T) {
	var called struct {
		fn   string
		args any
	}
	manager, err := NewManager(types.KernelConfig{
		AIProviders: map[string]types.AIProviderRegistration{
			"openai": {
				Type:   types.AIProviderOpenAI,
				Config: types.OpenAIProviderConfig{APIKey: "initial"},
			},
		},
		ProviderKeyMapping: map[string]string{"CUSTOM_KEY": "openai"},
	}, Hooks{
		HasJSRuntime: func() bool { return true },
		CallJS: func(_ context.Context, fn string, args any) (json.RawMessage, error) {
			called.fn = fn
			called.args = args
			return json.RawMessage(`{"refreshed":true}`), nil
		},
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if !manager.Registry().HasAIProvider("openai") {
		t.Fatalf("configured provider was not registered")
	}

	if err := manager.RefreshProviderSecret(context.Background(), "CUSTOM_KEY", "rotated"); err != nil {
		t.Fatalf("RefreshProviderSecret: %v", err)
	}

	reg, ok := manager.Registry().GetAIProvider("openai")
	if !ok {
		t.Fatal("openai provider disappeared")
	}
	cfg, ok := reg.Config.(types.OpenAIProviderConfig)
	if !ok {
		t.Fatalf("provider config type = %T", reg.Config)
	}
	if cfg.APIKey != "rotated" {
		t.Fatalf("provider API key = %q, want rotated", cfg.APIKey)
	}

	if called.fn != "__brainkit.secrets.refreshProvider" {
		t.Fatalf("refresh fn = %q", called.fn)
	}
	payload, ok := called.args.(map[string]string)
	if !ok {
		t.Fatalf("refresh args type = %T", called.args)
	}
	if payload["provider"] != "openai" {
		t.Fatalf("refresh payload = %#v", payload)
	}
	if _, leaked := payload["apiKey"]; leaked {
		t.Fatalf("refresh payload should not carry secret material: %#v", payload)
	}
}

func TestManagerRefreshProviderSecretPropagatesRuntimeError(t *testing.T) {
	want := errors.New("js refresh failed")
	manager, err := NewManager(types.KernelConfig{
		AIProviders: map[string]types.AIProviderRegistration{
			"openai": {
				Type:   types.AIProviderOpenAI,
				Config: types.OpenAIProviderConfig{APIKey: "initial"},
			},
		},
		ProviderKeyMapping: map[string]string{"CUSTOM_KEY": "openai"},
	}, Hooks{
		HasJSRuntime: func() bool { return true },
		CallJS: func(context.Context, string, any) (json.RawMessage, error) {
			return nil, want
		},
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	err = manager.RefreshProviderSecret(context.Background(), "CUSTOM_KEY", "rotated")
	if !errors.Is(err, want) {
		t.Fatalf("RefreshProviderSecret error = %v, want %v", err, want)
	}

	reg, ok := manager.Registry().GetAIProvider("openai")
	if !ok {
		t.Fatal("openai provider disappeared")
	}
	if cfg := reg.Config.(types.OpenAIProviderConfig); cfg.APIKey != "rotated" {
		t.Fatalf("provider API key = %q, want rotated even when cache refresh reports error", cfg.APIKey)
	}
}

func TestManagerRefreshProviderSecretDefaultMappingIgnoresAbsentProvider(t *testing.T) {
	manager, err := NewManager(types.KernelConfig{}, Hooks{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.RefreshProviderSecret(context.Background(), "OPENAI_API_KEY", "rotated"); err != nil {
		t.Fatalf("default mapping for absent provider should be no-op, got %v", err)
	}
}

func TestManagerRefreshProviderSecretExplicitMissingProviderErrors(t *testing.T) {
	manager, err := NewManager(types.KernelConfig{
		ProviderKeyMapping: map[string]string{"CUSTOM_KEY": "missing"},
	}, Hooks{})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.RefreshProviderSecret(context.Background(), "CUSTOM_KEY", "rotated"); err == nil {
		t.Fatal("expected explicit missing provider mapping to fail")
	}
}

func TestManagerImplementsProviderRuntimeCapabilities(t *testing.T) {
	manager, err := NewManager(types.KernelConfig{}, Hooks{})
	require.NoError(t, err)
	defer manager.Close()

	var _ bkmodule.ProviderSecretRefresher = manager
}

func TestManagerInvalidatesRegistryRuntimeCache(t *testing.T) {
	var called struct {
		fn   string
		args any
	}
	manager, err := NewManager(types.KernelConfig{}, Hooks{
		HasJSRuntime: func() bool { return true },
		CallJS: func(_ context.Context, fn string, args any) (json.RawMessage, error) {
			called.fn = fn
			called.args = args
			return json.RawMessage(`{"cleared":true}`), nil
		},
	})
	require.NoError(t, err)
	defer manager.Close()

	require.NoError(t, manager.InvalidateRegistryRuntimeCache(context.Background(), "provider", "openai"))
	require.Equal(t, "__brainkit.registry.clearCache", called.fn)
	require.Equal(t, map[string]string{"category": "provider", "name": "openai"}, called.args)
}

func TestManagerRuntimeCacheInvalidationIsLifecycleScoped(t *testing.T) {
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	var once sync.Once
	manager, err := NewManager(types.KernelConfig{}, Hooks{
		HasJSRuntime:    func() bool { return true },
		ShutdownContext: shutdownCtx,
		CallJS: func(ctx context.Context, _ string, _ any) (json.RawMessage, error) {
			once.Do(func() { close(started) })
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})
	require.NoError(t, err)
	defer manager.Close()

	done := make(chan error, 1)
	go func() {
		done <- manager.InvalidateRegistryRuntimeCache(context.Background(), "provider", "openai")
	}()

	require.Eventually(t, func() bool {
		select {
		case <-started:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return manager.DebugSnapshot().ActiveOperations == 1
	}, time.Second, 10*time.Millisecond)

	shutdownCancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("runtime cache invalidation did not stop after shutdown")
	}
	require.Eventually(t, func() bool {
		return manager.DebugSnapshot().ActiveOperations == 0
	}, time.Second, 10*time.Millisecond)
}

func TestManagerCloseCancelsActiveRuntimeOperation(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	manager, err := NewManager(types.KernelConfig{}, Hooks{
		HasJSRuntime: func() bool { return true },
		CallJS: func(ctx context.Context, _ string, _ any) (json.RawMessage, error) {
			once.Do(func() { close(started) })
			<-ctx.Done()
			return nil, ctx.Err()
		},
	})
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		done <- manager.InvalidateRegistryRuntimeCache(context.Background(), "provider", "openai")
	}()

	require.Eventually(t, func() bool {
		select {
		case <-started:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	require.NoError(t, manager.Close())
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("runtime cache invalidation did not return after manager close")
	}
	require.Equal(t, int64(0), manager.DebugSnapshot().ActiveOperations)
}

func TestManagerCloseContextReturnsDeadlineWhenRuntimeOperationIgnoresContext(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	manager, err := NewManager(types.KernelConfig{}, Hooks{
		HasJSRuntime: func() bool { return true },
		CallJS: func(context.Context, string, any) (json.RawMessage, error) {
			once.Do(func() { close(started) })
			<-release
			return json.RawMessage(`{"cleared":true}`), nil
		},
	})
	require.NoError(t, err)
	defer manager.Close()

	done := make(chan error, 1)
	go func() {
		done <- manager.InvalidateRegistryRuntimeCache(context.Background(), "provider", "openai")
	}()

	require.Eventually(t, func() bool {
		select {
		case <-started:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return manager.DebugSnapshot().ActiveOperations == 1
	}, time.Second, 10*time.Millisecond)

	closeCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err = manager.CloseContext(closeCtx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	snapshot := manager.DebugSnapshot()
	require.True(t, snapshot.Closed)
	require.False(t, snapshot.Closing)
	require.Equal(t, int64(1), snapshot.ActiveOperations)

	close(release)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("runtime cache invalidation did not return after release")
	}

	require.NoError(t, manager.CloseContext(context.Background()))
	snapshot = manager.DebugSnapshot()
	require.True(t, snapshot.Closed)
	require.False(t, snapshot.Closing)
	require.Equal(t, int64(0), snapshot.ActiveOperations)
}

func TestManagerCloseContextRetriesWithSinglePendingRuntimeOperationWait(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	manager, err := NewManager(types.KernelConfig{}, Hooks{
		HasJSRuntime: func() bool { return true },
		CallJS: func(context.Context, string, any) (json.RawMessage, error) {
			once.Do(func() { close(started) })
			<-release
			return json.RawMessage(`{"cleared":true}`), nil
		},
	})
	require.NoError(t, err)
	defer manager.Close()

	done := make(chan error, 1)
	go func() {
		done <- manager.InvalidateRegistryRuntimeCache(context.Background(), "provider", "openai")
	}()

	require.Eventually(t, func() bool {
		select {
		case <-started:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return manager.DebugSnapshot().ActiveOperations == 1
	}, time.Second, 10*time.Millisecond)

	for i := 0; i < 2; i++ {
		closeCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		err := manager.CloseContext(closeCtx)
		cancel()
		require.ErrorIs(t, err, context.DeadlineExceeded)
		snapshot := manager.DebugSnapshot()
		require.True(t, snapshot.Closed)
		require.False(t, snapshot.Closing)
		require.Equal(t, int64(1), snapshot.ActiveOperations)
	}

	close(release)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("runtime cache invalidation did not return after release")
	}
	require.NoError(t, manager.CloseContext(context.Background()))
	require.Equal(t, int64(0), manager.DebugSnapshot().ActiveOperations)
}

func TestManagerDebugSnapshotReportsClosingDuringClose(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	manager, err := NewManager(types.KernelConfig{}, Hooks{
		HasJSRuntime: func() bool { return true },
		CallJS: func(context.Context, string, any) (json.RawMessage, error) {
			once.Do(func() { close(started) })
			<-release
			return json.RawMessage(`{"cleared":true}`), nil
		},
	})
	require.NoError(t, err)
	defer manager.Close()

	opDone := make(chan error, 1)
	go func() {
		opDone <- manager.InvalidateRegistryRuntimeCache(context.Background(), "provider", "openai")
	}()
	require.Eventually(t, func() bool {
		select {
		case <-started:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	closeCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- manager.CloseContext(closeCtx)
	}()
	require.Eventually(t, func() bool {
		snapshot := manager.DebugSnapshot()
		return snapshot.Closing && snapshot.Closed && snapshot.ActiveOperations == 1
	}, time.Second, 10*time.Millisecond)

	close(release)
	require.NoError(t, <-opDone)
	require.NoError(t, <-closeDone)
	snapshot := manager.DebugSnapshot()
	require.True(t, snapshot.Closed)
	require.False(t, snapshot.Closing)
	require.Equal(t, int64(0), snapshot.ActiveOperations)
}

func TestProbeVectorStoreUsesRuntimeHook(t *testing.T) {
	manager, err := NewManager(types.KernelConfig{}, Hooks{
		HasJSRuntime: func() bool { return true },
		EvalJS: func(_ context.Context, filename, code string) (string, error) {
			if filename != "__probe_vectorstore.ts" {
				t.Fatalf("filename = %q", filename)
			}
			return `{"available":true}`, nil
		},
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := manager.Registry().RegisterVectorStore("main", provreg.VectorStoreRegistration{
		Type: provreg.VectorStoreLibSQL,
	}); err != nil {
		t.Fatalf("RegisterVectorStore: %v", err)
	}

	result := manager.ProbeVectorStore("main")
	if !result.Available || result.Error != "" {
		t.Fatalf("probe result = %#v", result)
	}
	stores := manager.Registry().ListVectorStores()
	if len(stores) != 1 || !stores[0].Healthy || stores[0].LastProbed.IsZero() {
		t.Fatalf("registry probe state = %#v", stores)
	}
}
