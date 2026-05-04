package engine

import (
	"context"

	bkmodule "github.com/brainlet/brainkit/module"
	provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
)

// ProbeAIProvider runs a live HTTP probe against a registered AI provider.
func (k *Kernel) ProbeAIProvider(name string) provreg.ProbeResult {
	if k == nil || k.providerHost == nil {
		return provreg.ProbeResult{Error: "provider host not initialized"}
	}
	return k.providerHost.ProbeAIProviderContext(context.Background(), name)
}

// ProbeVectorStore probes a vector store through the provider host.
func (k *Kernel) ProbeVectorStore(name string) provreg.ProbeResult {
	if k == nil || k.providerHost == nil {
		return provreg.ProbeResult{Error: "provider host not initialized"}
	}
	return k.providerHost.ProbeVectorStoreContext(context.Background(), name)
}

// ProbeStorage probes a storage backend through the provider host.
func (k *Kernel) ProbeStorage(name string) provreg.ProbeResult {
	if k == nil || k.providerHost == nil {
		return provreg.ProbeResult{Error: "provider host not initialized"}
	}
	return k.providerHost.ProbeStorageContext(context.Background(), name)
}

// ProbeAll runs probes for all registered providers, vector stores, and storages.
func (k *Kernel) ProbeAll() {
	k.ProbeAllContext(context.Background())
}

// ProbeAllContext runs probes for all registered providers, vector stores, and
// storages under caller-owned cancellation plus the kernel shutdown signal.
func (k *Kernel) ProbeAllContext(ctx context.Context) {
	if k == nil || k.providerHost == nil {
		return
	}
	k.providerHost.ProbeAllContext(ctx)
}

// RefreshProviderSecret routes provider secret refreshes through the
// module-owned provider host under caller-owned cancellation plus kernel
// shutdown cancellation.
func (k *Kernel) RefreshProviderSecret(ctx context.Context, name, newValue string) error {
	if k == nil || k.providerHost == nil {
		return nil
	}
	return k.providerHost.RefreshProviderSecret(ctx, name, newValue)
}

// ProviderSecretRefresher returns the provider-host owned secret refresh
// capability for module mounts.
func (k *Kernel) ProviderSecretRefresher() bkmodule.ProviderSecretRefresher {
	if k == nil || k.providerHost == nil {
		return nil
	}
	return k.providerHost
}
