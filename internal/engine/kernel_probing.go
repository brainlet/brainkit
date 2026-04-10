package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	provreg "github.com/brainlet/brainkit/internal/providers"
)

// --- Kernel-level probing (uses JS runtime for vector/storage) ---

// ProbeAIProvider runs a live HTTP probe against a registered AI provider.
func (k *Kernel) ProbeAIProvider(name string) provreg.ProbeResult {
	return k.providers.ProbeAIProvider(name)
}

// ProbeVectorStore probes a vector store by instantiating it in the JS runtime
// and calling listIndexes(). This tests real connectivity, not just config validity.
func (k *Kernel) ProbeVectorStore(name string) provreg.ProbeResult {
	start := time.Now()
	result, err := k.EvalTS(context.Background(), "__probe_vectorstore.ts", fmt.Sprintf(`
		try {
			var vs = vectorStore(%q);
			await vs.listIndexes();
			return JSON.stringify({ available: true });
		} catch(e) {
			return JSON.stringify({ available: false, error: e.message || String(e) });
		}
	`, name))
	latency := time.Since(start)

	if err != nil {
		k.providers.UpdateProbeResult("vectorStore", name, false, latency, err.Error())
		return provreg.ProbeResult{Error: err.Error(), Latency: latency}
	}

	var parsed struct {
		Available bool   `json:"available"`
		Error     string `json:"error"`
	}
	json.Unmarshal([]byte(result), &parsed)

	k.providers.UpdateProbeResult("vectorStore", name, parsed.Available, latency, parsed.Error)
	return provreg.ProbeResult{
		Available:    parsed.Available,
		Capabilities: provreg.DefaultVectorCapabilities(),
		Latency:      latency,
		Error:        parsed.Error,
	}
}

// ProbeStorage probes a storage backend by instantiating it in the JS runtime
// and calling a simple operation. Tests real connectivity.
func (k *Kernel) ProbeStorage(name string) provreg.ProbeResult {
	start := time.Now()
	result, err := k.EvalTS(context.Background(), "__probe_storage.ts", fmt.Sprintf(`
		try {
			var s = storage(%q);
			if (s && typeof s.listThreads === "function") {
				await s.listThreads({});
			}
			return JSON.stringify({ available: true });
		} catch(e) {
			return JSON.stringify({ available: false, error: e.message || String(e) });
		}
	`, name))
	latency := time.Since(start)

	if err != nil {
		k.providers.UpdateProbeResult("storage", name, false, latency, err.Error())
		return provreg.ProbeResult{Error: err.Error(), Latency: latency}
	}

	var parsed struct {
		Available bool   `json:"available"`
		Error     string `json:"error"`
	}
	json.Unmarshal([]byte(result), &parsed)

	k.providers.UpdateProbeResult("storage", name, parsed.Available, latency, parsed.Error)
	return provreg.ProbeResult{
		Available:    parsed.Available,
		Capabilities: provreg.DefaultStorageCapabilities(),
		Latency:      latency,
		Error:        parsed.Error,
	}
}

// ProbeAll runs probes for all registered providers, vector stores, and storages.
func (k *Kernel) ProbeAll() {
	for _, p := range k.providers.ListAIProviders() {
		k.ProbeAIProvider(p.Name)
	}
	for _, v := range k.providers.ListVectorStores() {
		k.ProbeVectorStore(v.Name)
	}
	for _, s := range k.providers.ListStorages() {
		k.ProbeStorage(s.Name)
	}
}

// startPeriodicProbing starts a background goroutine that probes all registered
// resources at the configured interval. Stops when the Kernel is closed.
func (k *Kernel) startPeriodicProbing() {
	interval := k.config.Probe.PeriodicInterval
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			k.mu.Lock()
			closed := k.closed
			k.mu.Unlock()
			if closed {
				ticker.Stop()
				return
			}
			k.ProbeAll()
		}
	}()
}
