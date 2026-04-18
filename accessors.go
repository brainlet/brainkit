package brainkit

import (
	"context"
	"fmt"

	provreg "github.com/brainlet/brainkit/internal/providers"
	"github.com/brainlet/brainkit/internal/types"
)

// ProviderInfo describes a registered AI provider. Alias over the
// internal types so consumers can name the shape without reaching
// into internal/.
type ProviderInfo = types.ProviderInfo

// VectorStoreInfo describes a registered vector store.
type VectorStoreInfo = types.VectorStoreInfo

// StorageInfo describes a registered storage backend.
type StorageInfo = types.StorageInfo

// AIProviderType identifies an AI provider's backing implementation.
type AIProviderType = types.AIProviderType

// VectorStoreType identifies a vector store's backing implementation.
type VectorStoreType = types.VectorStoreType

// StorageType identifies a storage backend's backing implementation.
type StorageType = types.StorageType

// AIProviderRegistration is the stored record for a provider.
type AIProviderRegistration = types.AIProviderRegistration

// Providers is the consolidated accessor for AI provider management.
// Construction is cached per-Kit through (*Kit).Providers().
type Providers struct {
	k *Kit
}

// Register adds or replaces a provider registration.
func (p *Providers) Register(name string, typ AIProviderType, config any) error {
	return p.k.kernel.RegisterAIProvider(name, typ, config)
}

// Unregister removes a provider. No-op when the name is unknown.
func (p *Providers) Unregister(name string) {
	p.k.kernel.UnregisterAIProvider(name)
}

// List returns a snapshot of every registered provider.
func (p *Providers) List() []ProviderInfo {
	return p.k.kernel.ListAIProviders()
}

// Get returns the registration record for name.
func (p *Providers) Get(name string) (AIProviderRegistration, bool) {
	return p.k.kernel.ProviderRegistry().GetAIProvider(name)
}

// Has reports whether a provider is registered under name.
func (p *Providers) Has(name string) bool {
	return p.k.kernel.ProviderRegistry().HasAIProvider(name)
}

// Storages is the consolidated accessor for storage backend management.
type Storages struct {
	k *Kit
}

// Register adds or replaces a storage registration.
func (s *Storages) Register(name string, typ StorageType, config any) error {
	return s.k.kernel.RegisterStorage(name, typ, config)
}

// Unregister removes a storage registration.
func (s *Storages) Unregister(name string) {
	s.k.kernel.UnregisterStorage(name)
}

// List returns a snapshot of every registered storage backend.
func (s *Storages) List() []StorageInfo {
	return s.k.kernel.ListStorages()
}

// Get returns the registration record for name.
func (s *Storages) Get(name string) (types.StorageRegistration, bool) {
	return s.k.kernel.ProviderRegistry().GetStorage(name)
}

// Has reports whether a storage backend is registered under name.
func (s *Storages) Has(name string) bool {
	return s.k.kernel.ProviderRegistry().HasStorage(name)
}

// Vectors is the consolidated accessor for vector store management.
type Vectors struct {
	k *Kit
}

// Register adds or replaces a vector store registration.
func (v *Vectors) Register(name string, typ VectorStoreType, config any) error {
	return v.k.kernel.RegisterVectorStore(name, typ, config)
}

// Unregister removes a vector store registration.
func (v *Vectors) Unregister(name string) {
	v.k.kernel.UnregisterVectorStore(name)
}

// List returns a snapshot of every registered vector store.
func (v *Vectors) List() []VectorStoreInfo {
	return v.k.kernel.ListVectorStores()
}

// Get returns the registration record for name.
func (v *Vectors) Get(name string) (types.VectorStoreRegistration, bool) {
	return v.k.kernel.ProviderRegistry().GetVectorStore(name)
}

// Has reports whether a vector store is registered under name.
func (v *Vectors) Has(name string) bool {
	return v.k.kernel.ProviderRegistry().HasVectorStore(name)
}

// Secrets is the consolidated accessor for encrypted secret management.
// Returns errors wrapping the underlying store; when the Kit is built
// without a secret key, Set/Get/Delete return a NotConfiguredError-
// shaped error from the env-only store fallback.
type Secrets struct {
	k *Kit
}

// Set stores value under name, encrypted with the Kit's secret key.
func (s *Secrets) Set(ctx context.Context, name, value string) error {
	store := s.k.kernel.SecretStore()
	if store == nil {
		return fmt.Errorf("secrets: no secret store configured (set Config.SecretKey)")
	}
	return store.Set(ctx, name, value)
}

// Get returns the decrypted value for name.
func (s *Secrets) Get(ctx context.Context, name string) (string, error) {
	store := s.k.kernel.SecretStore()
	if store == nil {
		return "", fmt.Errorf("secrets: no secret store configured (set Config.SecretKey)")
	}
	return store.Get(ctx, name)
}

// Delete removes the secret named name.
func (s *Secrets) Delete(ctx context.Context, name string) error {
	store := s.k.kernel.SecretStore()
	if store == nil {
		return fmt.Errorf("secrets: no secret store configured (set Config.SecretKey)")
	}
	return store.Delete(ctx, name)
}

// List returns the metadata for every stored secret (no values).
func (s *Secrets) List(ctx context.Context) ([]SecretMeta, error) {
	store := s.k.kernel.SecretStore()
	if store == nil {
		return nil, fmt.Errorf("secrets: no secret store configured (set Config.SecretKey)")
	}
	return store.List(ctx)
}

// Rotate replaces a secret's value. When a plugin restarter module is
// wired (see modules/plugins), plugins whose env.value references this
// secret are restarted to pick up the new value.
func (s *Secrets) Rotate(ctx context.Context, name, newValue string) error {
	if err := s.Set(ctx, name, newValue); err != nil {
		return err
	}
	// SecretsDomain's rotation hook does the restart wiring. Nothing
	// extra here — the Set write IS the rotation event from the
	// accessor's perspective.
	return nil
}

// Providers returns the AI provider accessor.
func (k *Kit) Providers() *Providers {
	if k.providers == nil {
		k.providers = &Providers{k: k}
	}
	return k.providers
}

// Storages returns the storage backend accessor.
func (k *Kit) Storages() *Storages {
	if k.storages == nil {
		k.storages = &Storages{k: k}
	}
	return k.storages
}

// Vectors returns the vector store accessor.
func (k *Kit) Vectors() *Vectors {
	if k.vectors == nil {
		k.vectors = &Vectors{k: k}
	}
	return k.vectors
}

// Secrets returns the secret store accessor.
func (k *Kit) Secrets() *Secrets {
	if k.secrets == nil {
		k.secrets = &Secrets{k: k}
	}
	return k.secrets
}

// Guard against unused imports when the internal provreg package is
// only referenced via type aliases above.
var _ = provreg.AIProviderType("")
