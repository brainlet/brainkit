package engine

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/modulehost/providerhost"
	"github.com/brainlet/brainkit/modulehost/storagehost"
)

func TestRegistryMutationManagerAddProviderInvalidatesRuntimeCacheAndPreservesMutation(t *testing.T) {
	want := errors.New("runtime cache unavailable")
	var called struct {
		fn   string
		args any
	}
	k := newRegistryMutationKernel(t, func(_ context.Context, fn string, args any) (json.RawMessage, error) {
		called.fn = fn
		called.args = args
		return nil, want
	})

	err := k.AddRegistryProvider(context.Background(), "openai", "openai", json.RawMessage(`{"APIKey":"test-key"}`))
	if !errors.Is(err, want) {
		t.Fatalf("AddRegistryProvider error = %v, want %v", err, want)
	}
	if !k.providerHost.Registry().HasAIProvider("openai") {
		t.Fatal("provider mutation should remain visible when runtime cache invalidation fails")
	}
	if called.fn != "__brainkit.registry."+"clearCache" {
		t.Fatalf("runtime cache function = %q", called.fn)
	}
	if got := called.args; !sameStringMap(got, map[string]string{"category": "provider", "name": "openai"}) {
		t.Fatalf("runtime cache args = %#v", got)
	}
}

func TestRegistryMutationManagerAddStorageInvalidatesRuntimeCacheAndPreservesMutation(t *testing.T) {
	want := errors.New("runtime cache unavailable")
	var called struct {
		fn   string
		args any
	}
	k := newRegistryMutationKernel(t, func(_ context.Context, fn string, args any) (json.RawMessage, error) {
		called.fn = fn
		called.args = args
		return nil, want
	})

	err := k.AddRegistryStorage(context.Background(), "docs", "memory", json.RawMessage(`{"url":"mem://docs"}`))
	if !errors.Is(err, want) {
		t.Fatalf("AddRegistryStorage error = %v, want %v", err, want)
	}
	if !k.providerHost.Registry().HasStorage("docs") {
		t.Fatal("storage mutation should remain visible when runtime cache invalidation fails")
	}
	if called.fn != "__brainkit.registry."+"clearCache" {
		t.Fatalf("runtime cache function = %q", called.fn)
	}
	if got := called.args; !sameStringMap(got, map[string]string{"category": "storage", "name": "docs"}) {
		t.Fatalf("runtime cache args = %#v", got)
	}
}

func TestRegistryMutationManagerAddVectorDecodesConfigThroughStorageHost(t *testing.T) {
	var called struct {
		fn   string
		args any
	}
	k := newRegistryMutationKernel(t, func(_ context.Context, fn string, args any) (json.RawMessage, error) {
		called.fn = fn
		called.args = args
		return json.RawMessage(`{"cleared":true}`), nil
	})

	err := k.AddRegistryVector(context.Background(), "docs", "sqlite", json.RawMessage(`{
		"path": "/tmp/docs.db",
		"connectionString": "postgres://unused",
		"uri": "mongodb://unused",
		"dbName": "vectors"
	}`))
	if err != nil {
		t.Fatalf("AddRegistryVector: %v", err)
	}
	reg, ok := k.providerHost.Registry().GetVectorStore("docs")
	if !ok {
		t.Fatal("vector store was not registered")
	}
	if reg.Type != types.VectorStoreLibSQL {
		t.Fatalf("vector type = %q, want libsql", reg.Type)
	}
	if called.fn != "__brainkit.registry."+"clearCache" {
		t.Fatalf("runtime cache function = %q", called.fn)
	}
	if got := called.args; !sameStringMap(got, map[string]string{"category": "vectorStore", "name": "docs"}) {
		t.Fatalf("runtime cache args = %#v", got)
	}
}

func newRegistryMutationKernel(t *testing.T, callJS func(context.Context, string, any) (json.RawMessage, error)) *Kernel {
	t.Helper()
	providers, err := providerhost.NewManager(types.KernelConfig{}, providerhost.Hooks{
		HasJSRuntime: func() bool { return true },
		CallJS:       callJS,
	})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(func() { _ = providers.Close() })
	return &Kernel{
		providerHost: providers,
		storageHost:  storagehost.NewManager(providers.Registry(), func() bool { return false }),
	}
}

func sameStringMap(v any, want map[string]string) bool {
	got, ok := v.(map[string]string)
	if !ok || len(got) != len(want) {
		return false
	}
	for k, wantValue := range want {
		if got[k] != wantValue {
			return false
		}
	}
	return true
}
