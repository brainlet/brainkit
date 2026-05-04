package engine

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
)

// RegistryMutationManager returns the kernel-owned live registry mutation
// capability. Bus command modules use this so provider/storage/vector mutation,
// bridge lifecycle, and JS runtime cache invalidation stay owned by the runtime
// hosts rather than by command handlers.
func (k *Kernel) RegistryMutationManager() bkmodule.RegistryMutationManager {
	if k == nil || k.providerHost == nil {
		return nil
	}
	return k
}

// AddRegistryProvider registers an AI provider and synchronously invalidates
// any active runtime-side provider cache. The provider mutation remains visible
// when cache invalidation fails; callers receive the invalidation error so they
// can retry or surface the stale-runtime state.
func (k *Kernel) AddRegistryProvider(ctx context.Context, name, providerType string, raw json.RawMessage) error {
	if k == nil || k.providerHost == nil {
		return fmt.Errorf("brainkit: provider host not initialized")
	}
	config, err := provreg.DecodeAIProviderConfig(providerType, raw)
	if err != nil {
		return err
	}
	if err := k.providerHost.Registry().RegisterAIProvider(name, provreg.AIProviderRegistration{
		Type:   provreg.AIProviderType(providerType),
		Config: config,
	}); err != nil {
		return err
	}
	return k.providerHost.InvalidateRegistryRuntimeCache(ctx, "provider", name)
}

// RemoveRegistryProvider unregisters an AI provider and synchronously clears
// the active runtime-side provider cache.
func (k *Kernel) RemoveRegistryProvider(ctx context.Context, name string) error {
	if k == nil || k.providerHost == nil {
		return fmt.Errorf("brainkit: provider host not initialized")
	}
	k.providerHost.Registry().UnregisterAIProvider(name)
	return k.providerHost.InvalidateRegistryRuntimeCache(ctx, "provider", name)
}

// AddRegistryStorage registers a storage backend through the storage host so
// any required bridge resource is owned by the storage lifecycle manager.
func (k *Kernel) AddRegistryStorage(ctx context.Context, name, storageType string, raw json.RawMessage) error {
	if k == nil || k.storageHost == nil || k.providerHost == nil {
		return fmt.Errorf("brainkit: storage host not initialized")
	}
	cfg, err := decodeRegistryStorageConfig(storageType, raw)
	if err != nil {
		return err
	}
	if err := k.storageHost.AddStorage(name, cfg); err != nil {
		return err
	}
	return k.providerHost.InvalidateRegistryRuntimeCache(ctx, "storage", name)
}

// RemoveRegistryStorage removes a storage backend through the storage host so
// bridge teardown remains close-context aware.
func (k *Kernel) RemoveRegistryStorage(ctx context.Context, name string) error {
	if k == nil || k.storageHost == nil || k.providerHost == nil {
		return fmt.Errorf("brainkit: storage host not initialized")
	}
	if err := k.storageHost.RemoveStorageContext(ctx, name); err != nil {
		return err
	}
	return k.providerHost.InvalidateRegistryRuntimeCache(ctx, "storage", name)
}

// AddRegistryVector registers a vector store through the storage host so any
// required bridge resource is owned by the storage lifecycle manager.
func (k *Kernel) AddRegistryVector(ctx context.Context, name, vectorType string, raw json.RawMessage) error {
	if k == nil || k.storageHost == nil || k.providerHost == nil {
		return fmt.Errorf("brainkit: storage host not initialized")
	}
	cfg, err := decodeRegistryVectorConfig(vectorType, raw)
	if err != nil {
		return err
	}
	if err := k.storageHost.AddVector(name, cfg); err != nil {
		return err
	}
	return k.providerHost.InvalidateRegistryRuntimeCache(ctx, "vectorStore", name)
}

// RemoveRegistryVector removes a vector store through the storage host so
// bridge teardown remains close-context aware.
func (k *Kernel) RemoveRegistryVector(ctx context.Context, name string) error {
	if k == nil || k.storageHost == nil || k.providerHost == nil {
		return fmt.Errorf("brainkit: storage host not initialized")
	}
	if err := k.storageHost.RemoveVectorContext(ctx, name); err != nil {
		return err
	}
	return k.providerHost.InvalidateRegistryRuntimeCache(ctx, "vectorStore", name)
}

func decodeRegistryStorageConfig(typ string, raw json.RawMessage) (types.StorageConfig, error) {
	var base struct {
		Path             string `json:"path"`
		ConnectionString string `json:"connectionString"`
		URI              string `json:"uri"`
		DBName           string `json:"dbName"`
		URL              string `json:"url"`
		Token            string `json:"token"`
	}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &base); err != nil {
			return types.StorageConfig{}, fmt.Errorf("invalid storage config: %w", err)
		}
	}
	return types.StorageConfig{
		Type:             typ,
		Path:             base.Path,
		ConnectionString: base.ConnectionString,
		URI:              base.URI,
		DBName:           base.DBName,
		URL:              base.URL,
		Token:            base.Token,
	}, nil
}

func decodeRegistryVectorConfig(typ string, raw json.RawMessage) (types.VectorConfig, error) {
	var base struct {
		Path             string `json:"path"`
		ConnectionString string `json:"connectionString"`
		URI              string `json:"uri"`
		DBName           string `json:"dbName"`
	}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &base); err != nil {
			return types.VectorConfig{}, fmt.Errorf("invalid vector config: %w", err)
		}
	}
	return types.VectorConfig{
		Type:             typ,
		Path:             base.Path,
		ConnectionString: base.ConnectionString,
		URI:              base.URI,
		DBName:           base.DBName,
	}, nil
}

var _ bkmodule.RegistryMutationManager = (*Kernel)(nil)
