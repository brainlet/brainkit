package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ProbeAIProvider runs a live HTTP health check against a registered AI provider.
// Hits the provider's models endpoint to verify API key validity and connectivity.
func (r *ProviderRegistry) ProbeAIProvider(name string) ProbeResult {
	r.mu.RLock()
	e, ok := r.aiProviders[name]
	r.mu.RUnlock()
	if !ok {
		return ProbeResult{Error: "provider not registered: " + name}
	}

	reg := e.registration.(AIProviderRegistration)
	caps := KnownAICapabilities(reg.Type)

	endpoint, headers := probeEndpoint(reg)
	if endpoint == "" {
		// Provider type doesn't support HTTP probing (Bedrock, Vertex — need SDK auth)
		r.mu.Lock()
		e.healthy = true
		e.lastProbed = time.Now()
		e.lastErr = "probe not available for " + string(reg.Type)
		r.mu.Unlock()
		return ProbeResult{Available: true, Capabilities: caps}
	}

	timeout := r.probeConfig.ProbeTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	start := time.Now()
	err := httpProbe(endpoint, headers, timeout)
	latency := time.Since(start)

	r.mu.Lock()
	e.lastProbed = time.Now()
	e.latency = latency
	if err != nil {
		e.healthy = false
		e.lastErr = err.Error()
		r.mu.Unlock()
		return ProbeResult{Error: err.Error(), Latency: latency}
	}
	e.healthy = true
	e.lastErr = ""
	r.mu.Unlock()

	return ProbeResult{Available: true, Capabilities: caps, Latency: latency}
}

// probeEndpoint returns the HTTP endpoint + headers to probe for a given provider.
// Returns ("", nil) if the provider type doesn't support HTTP probing.
func probeEndpoint(reg AIProviderRegistration) (string, map[string]string) {
	var apiKey, baseURL string

	switch cfg := reg.Config.(type) {
	case OpenAIProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case AnthropicProviderConfig:
		apiKey = cfg.APIKey
		if cfg.AuthToken != "" {
			apiKey = cfg.AuthToken
		}
		baseURL = cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.anthropic.com/v1"
		}
		h := mergeHeaders(cfg.Headers, map[string]string{
			"x-api-key":         apiKey,
			"anthropic-version": "2023-06-01",
		})
		return baseURL + "/models", h

	case GoogleProviderConfig:
		baseURL = cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://generativelanguage.googleapis.com/v1beta"
		}
		return baseURL + "/models?key=" + cfg.APIKey, cfg.Headers

	case MistralProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.mistral.ai/v1"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case CohereProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.cohere.com/v2"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case GroqProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.groq.com/openai/v1"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case PerplexityProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.perplexity.ai"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case DeepSeekProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.deepseek.com/v1"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case FireworksProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.fireworks.ai/inference/v1"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case TogetherAIProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.together.xyz/v1"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case XAIProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.x.ai/v1"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case CerebrasProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.cerebras.ai/v1"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case HuggingFaceProviderConfig:
		apiKey, baseURL = cfg.APIKey, cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api-inference.huggingface.co"
		}
		return baseURL + "/models", bearerAuth(apiKey, cfg.Headers)

	case AzureProviderConfig:
		baseURL = cfg.BaseURL
		if baseURL == "" && cfg.ResourceName != "" {
			baseURL = "https://" + cfg.ResourceName + ".openai.azure.com/openai"
		}
		if baseURL == "" {
			return "", nil
		}
		h := mergeHeaders(cfg.Headers, map[string]string{"api-key": cfg.APIKey})
		return baseURL + "/models?api-version=2024-02-01", h

	case BedrockProviderConfig:
		return "", nil // AWS SDK auth — not HTTP-probable
	case VertexProviderConfig:
		return "", nil // GCP auth — not HTTP-probable
	}

	return "", nil
}

// httpProbe sends a GET request to the endpoint and checks for a 2xx response.
func httpProbe(endpoint string, headers map[string]string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("probe: build request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("probe: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("probe: authentication failed (HTTP %d)", resp.StatusCode)
	}
	return fmt.Errorf("probe: HTTP %d", resp.StatusCode)
}

func bearerAuth(apiKey string, extra map[string]string) map[string]string {
	h := make(map[string]string, len(extra)+1)
	for k, v := range extra {
		h[k] = v
	}
	if apiKey != "" {
		h["Authorization"] = "Bearer " + apiKey
	}
	return h
}

func mergeHeaders(base, overlay map[string]string) map[string]string {
	h := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		h[k] = v
	}
	for k, v := range overlay {
		h[k] = v
	}
	return h
}

// ProbeVectorStore runs a registry-level health check for a vector store.
// For real connectivity testing, use Kernel.ProbeVectorStore which has JS runtime access.
func (r *ProviderRegistry) ProbeVectorStore(name string) ProbeResult {
	r.mu.RLock()
	e, ok := r.vectorStores[name]
	r.mu.RUnlock()
	if !ok {
		return ProbeResult{Error: "vector store not registered: " + name}
	}

	// Registry-level: mark as unknown until Kernel-level probe runs
	r.mu.Lock()
	e.lastProbed = time.Now()
	e.lastErr = "awaiting JS runtime probe"
	r.mu.Unlock()

	return ProbeResult{Capabilities: DefaultVectorCapabilities()}
}

// ProbeStorage runs a registry-level health check for a storage backend.
// For real connectivity testing, use Kernel.ProbeStorage which has JS runtime access.
func (r *ProviderRegistry) ProbeStorage(name string) ProbeResult {
	r.mu.RLock()
	e, ok := r.storages[name]
	r.mu.RUnlock()
	if !ok {
		return ProbeResult{Error: "storage not registered: " + name}
	}

	reg := e.registration.(StorageRegistration)

	// InMemory is always healthy
	if reg.Type == StorageInMemory {
		r.mu.Lock()
		e.healthy = true
		e.lastProbed = time.Now()
		e.lastErr = ""
		r.mu.Unlock()
		return ProbeResult{Available: true, Capabilities: DefaultStorageCapabilities()}
	}

	// External storages need Kernel-level JS probe
	r.mu.Lock()
	e.lastProbed = time.Now()
	e.lastErr = "awaiting JS runtime probe"
	r.mu.Unlock()

	return ProbeResult{Capabilities: DefaultStorageCapabilities()}
}

// UpdateProbeResult allows the Kernel to push JS-runtime probe results back into the registry.
func (r *ProviderRegistry) UpdateProbeResult(category, name string, healthy bool, latency time.Duration, errMsg string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var e *entry
	switch category {
	case "vectorStore":
		e = r.vectorStores[name]
	case "storage":
		e = r.storages[name]
	case "provider":
		e = r.aiProviders[name]
	}
	if e == nil {
		return
	}
	e.healthy = healthy
	e.lastProbed = time.Now()
	e.latency = latency
	e.lastErr = errMsg
}
