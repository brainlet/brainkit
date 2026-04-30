package engine

import provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"

// ProbeAIProvider runs a live HTTP probe against a registered AI provider.
func (k *Kernel) ProbeAIProvider(name string) provreg.ProbeResult {
	return k.providerHost.ProbeAIProvider(name)
}

// ProbeVectorStore probes a vector store through the provider host.
func (k *Kernel) ProbeVectorStore(name string) provreg.ProbeResult {
	return k.providerHost.ProbeVectorStore(name)
}

// ProbeStorage probes a storage backend through the provider host.
func (k *Kernel) ProbeStorage(name string) provreg.ProbeResult {
	return k.providerHost.ProbeStorage(name)
}

// ProbeAll runs probes for all registered providers, vector stores, and storages.
func (k *Kernel) ProbeAll() {
	k.providerHost.ProbeAll()
}

// RefreshProviderIfSecret routes provider secret refreshes through the
// module-owned provider host.
func (k *Kernel) RefreshProviderIfSecret(name, newValue string) {
	k.providerHost.RefreshProviderIfSecret(name, newValue)
}
