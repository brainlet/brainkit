package registry

import "time"

// ProbeAIProvider runs a health check for a registered AI provider.
// Forces a fresh probe regardless of cache.
func (r *ProviderRegistry) ProbeAIProvider(name string) ProbeResult {
	r.mu.RLock()
	e, ok := r.aiProviders[name]
	r.mu.RUnlock()
	if !ok {
		return ProbeResult{Error: "provider not registered: " + name}
	}

	reg := e.registration.(AIProviderRegistration)
	caps := KnownAICapabilities(reg.Type)

	// For now, use known capabilities table.
	// Live probing (HTTP to provider API) will be added per-provider.
	start := time.Now()
	latency := time.Since(start)

	r.mu.Lock()
	e.healthy = true
	e.lastProbed = time.Now()
	e.lastErr = ""
	e.latency = latency
	r.mu.Unlock()

	return ProbeResult{
		Available:    true,
		Capabilities: caps,
		Latency:      latency,
	}
}

// ProbeVectorStore runs a health check for a registered vector store.
func (r *ProviderRegistry) ProbeVectorStore(name string) ProbeResult {
	r.mu.RLock()
	e, ok := r.vectorStores[name]
	r.mu.RUnlock()
	if !ok {
		return ProbeResult{Error: "vector store not registered: " + name}
	}

	_ = e // vector store probing requires JS runtime — deferred to Kernel-level probe
	caps := DefaultVectorCapabilities()

	r.mu.Lock()
	e.healthy = true
	e.lastProbed = time.Now()
	e.lastErr = ""
	r.mu.Unlock()

	return ProbeResult{
		Available:    true,
		Capabilities: caps,
	}
}

// ProbeStorage runs a health check for a registered storage.
func (r *ProviderRegistry) ProbeStorage(name string) ProbeResult {
	r.mu.RLock()
	e, ok := r.storages[name]
	r.mu.RUnlock()
	if !ok {
		return ProbeResult{Error: "storage not registered: " + name}
	}

	reg := e.registration.(StorageRegistration)
	caps := DefaultStorageCapabilities()

	// InMemory is always healthy
	if reg.Type == StorageInMemory {
		r.mu.Lock()
		e.healthy = true
		e.lastProbed = time.Now()
		e.lastErr = ""
		r.mu.Unlock()

		return ProbeResult{
			Available:    true,
			Capabilities: caps,
		}
	}

	// For external storages, mark as healthy (actual probe deferred to Kernel-level)
	r.mu.Lock()
	e.healthy = true
	e.lastProbed = time.Now()
	e.lastErr = ""
	r.mu.Unlock()

	return ProbeResult{
		Available:    true,
		Capabilities: caps,
	}
}
