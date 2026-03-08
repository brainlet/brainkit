// Ported from: packages/core/src/storage/storageWithInit.ts
package storage

import (
	"context"
	"os"
	"sync"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// AugmentedStore wraps a MastraCompositeStore to ensure Init is called
// automatically before any domain store is accessed.
//
// In TypeScript this is implemented via a Proxy that intercepts every method
// call and awaits init() first. In Go we use an explicit wrapper with an
// EnsureInit method and a GetStoreWithInit accessor.
//
// The wrapper is idempotent — calling AugmentWithInit on an already-augmented
// store returns the same wrapper.
type AugmentedStore struct {
	*MastraCompositeStore

	mu             sync.Mutex
	hasInitialized bool
	initErr        error
}

// AugmentWithInit wraps a MastraCompositeStore so that Init is called
// automatically before first use.
//
// Equivalent to the TypeScript augmentWithInit() function that returns a Proxy.
//
// If the store's DisableInit flag is true, or the MASTRA_DISABLE_STORAGE_INIT
// environment variable is set to "true", auto-initialization is skipped.
func AugmentWithInit(store *MastraCompositeStore) *AugmentedStore {
	return &AugmentedStore{
		MastraCompositeStore: store,
	}
}

// EnsureInit guarantees that Init has been called on the underlying store
// before returning. It is safe to call concurrently; only one Init runs.
//
// When DisableInit is true or the MASTRA_DISABLE_STORAGE_INIT env var is
// "true", this is a no-op.
func (a *AugmentedStore) EnsureInit(ctx context.Context) error {
	// Skip auto-initialization if disableInit is true.
	if a.MastraCompositeStore.DisableInit {
		return nil
	}

	// Environment variable equivalent of disableInit — used by migration CLI.
	if os.Getenv("MASTRA_DISABLE_STORAGE_INIT") == "true" {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.hasInitialized {
		return a.initErr
	}

	a.initErr = a.MastraCompositeStore.Init(ctx)
	a.hasInitialized = true
	return a.initErr
}

// Init calls Init on the underlying store, tracking the result so that
// subsequent calls to EnsureInit are no-ops. This matches the TypeScript
// behavior where calling init() directly on the proxy also sets the
// hasInitialized flag.
func (a *AugmentedStore) Init(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.hasInitialized {
		return a.initErr
	}

	a.initErr = a.MastraCompositeStore.Init(ctx)
	a.hasInitialized = true
	return a.initErr
}

// GetStoreWithInit calls EnsureInit and then returns the domain store.
// This is the recommended accessor when using an AugmentedStore — it
// guarantees initialization before domain access, mirroring the TypeScript
// Proxy behavior where every method call awaits init() first.
func (a *AugmentedStore) GetStoreWithInit(ctx context.Context, name DomainName) (domains.StorageDomain, error) {
	if err := a.EnsureInit(ctx); err != nil {
		return nil, err
	}
	return a.MastraCompositeStore.GetStore(name), nil
}
