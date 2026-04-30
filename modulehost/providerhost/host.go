// Package providerhost owns provider bootstrap, probing, and runtime refresh
// orchestration for one Kit runtime.
package providerhost

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
)

// Hooks are the optional runtime callbacks used for probes that need JS-side
// factories and for refreshing provider credentials inside the JS runtime.
type Hooks struct {
	HasJSRuntime func() bool
	EvalTS       func(context.Context, string, string) (string, error)
	CallJSSync   func(string, any)
}

// Manager owns the provider registry plus runtime-backed provider operations.
type Manager struct {
	providers          *provreg.ProviderRegistry
	hasJSRuntime       func() bool
	evalTS             func(context.Context, string, string) (string, error)
	callJSSync         func(string, any)
	providerKeyMapping map[string]string
}

// NewManager creates a provider host for one Kit runtime and registers the
// configured AI providers.
func NewManager(cfg types.KernelConfig, hooks Hooks) (*Manager, error) {
	if hooks.HasJSRuntime == nil {
		hooks.HasJSRuntime = func() bool { return false }
	}
	providers := provreg.New(cfg.Probe)
	manager := &Manager{
		providers:          providers,
		hasJSRuntime:       hooks.HasJSRuntime,
		evalTS:             hooks.EvalTS,
		callJSSync:         hooks.CallJSSync,
		providerKeyMapping: providerKeyMapping(cfg.ProviderKeyMapping),
	}
	for name, reg := range cfg.AIProviders {
		if err := providers.RegisterAIProvider(name, reg); err != nil {
			return nil, err
		}
	}
	return manager, nil
}

// Registry returns the shared provider/storage/vector registry.
func (m *Manager) Registry() *provreg.ProviderRegistry {
	if m == nil {
		return nil
	}
	return m.providers
}

// ProbeAIProvider runs a live HTTP probe against a registered AI provider.
func (m *Manager) ProbeAIProvider(name string) provreg.ProbeResult {
	return m.providers.ProbeAIProvider(name)
}

// ProbeVectorStore probes a vector store by instantiating it in the JS runtime
// and calling listIndexes().
func (m *Manager) ProbeVectorStore(name string) provreg.ProbeResult {
	start := time.Now()
	if !m.runtimeReady() {
		err := "js runtime not configured"
		m.providers.UpdateProbeResult("vectorStore", name, false, time.Since(start), err)
		return provreg.ProbeResult{Error: err, Latency: time.Since(start)}
	}
	result, err := m.evalTS(context.Background(), "__probe_vectorstore.ts", fmt.Sprintf(`
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
		m.providers.UpdateProbeResult("vectorStore", name, false, latency, err.Error())
		return provreg.ProbeResult{Error: err.Error(), Latency: latency}
	}

	var parsed struct {
		Available bool   `json:"available"`
		Error     string `json:"error"`
	}
	_ = json.Unmarshal([]byte(result), &parsed)

	m.providers.UpdateProbeResult("vectorStore", name, parsed.Available, latency, parsed.Error)
	return provreg.ProbeResult{
		Available:    parsed.Available,
		Capabilities: provreg.DefaultVectorCapabilities(),
		Latency:      latency,
		Error:        parsed.Error,
	}
}

// ProbeStorage probes a storage backend by instantiating it in the JS runtime
// and calling a simple operation.
func (m *Manager) ProbeStorage(name string) provreg.ProbeResult {
	start := time.Now()
	if !m.runtimeReady() {
		err := "js runtime not configured"
		m.providers.UpdateProbeResult("storage", name, false, time.Since(start), err)
		return provreg.ProbeResult{Error: err, Latency: time.Since(start)}
	}
	result, err := m.evalTS(context.Background(), "__probe_storage.ts", fmt.Sprintf(`
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
		m.providers.UpdateProbeResult("storage", name, false, latency, err.Error())
		return provreg.ProbeResult{Error: err.Error(), Latency: latency}
	}

	var parsed struct {
		Available bool   `json:"available"`
		Error     string `json:"error"`
	}
	_ = json.Unmarshal([]byte(result), &parsed)

	m.providers.UpdateProbeResult("storage", name, parsed.Available, latency, parsed.Error)
	return provreg.ProbeResult{
		Available:    parsed.Available,
		Capabilities: provreg.DefaultStorageCapabilities(),
		Latency:      latency,
		Error:        parsed.Error,
	}
}

// ProbeAll runs probes for all registered providers, vector stores, and storages.
func (m *Manager) ProbeAll() {
	for _, p := range m.providers.ListAIProviders() {
		m.ProbeAIProvider(p.Name)
	}
	if !m.runtimeReady() {
		return
	}
	for _, v := range m.providers.ListVectorStores() {
		m.ProbeVectorStore(v.Name)
	}
	for _, s := range m.providers.ListStorages() {
		m.ProbeStorage(s.Name)
	}
}

// RefreshProviderIfSecret checks whether a secret name maps to a provider key
// and refreshes the JS-side provider cache when the runtime is present.
func (m *Manager) RefreshProviderIfSecret(name, newValue string) {
	if m == nil || m.callJSSync == nil {
		return
	}
	provName, ok := m.providerKeyMapping[name]
	if !ok {
		return
	}
	m.callJSSync("__brainkit.secrets.refreshProvider", map[string]string{
		"provider": provName,
		"apiKey":   newValue,
	})
}

func (m *Manager) runtimeReady() bool {
	return m != nil && m.hasJSRuntime != nil && m.hasJSRuntime() && m.evalTS != nil
}

// AutoDetectProviders scans os.Getenv and cfg.EnvVars for known API key
// patterns only when the caller did not explicitly provide an AIProviders map.
// A non-nil map, including an empty one, means provider configuration is fully
// explicit and environment auto-detection is disabled.
func AutoDetectProviders(cfg *types.KernelConfig) {
	if cfg.AIProviders != nil {
		return
	}
	cfg.AIProviders = make(map[string]provreg.AIProviderRegistration)

	type providerMapping struct {
		name string
		typ  provreg.AIProviderType
		make func(apiKey string) any
	}

	mappings := map[string]providerMapping{
		"OPENAI_API_KEY":     {"openai", provreg.AIProviderOpenAI, func(k string) any { return provreg.OpenAIProviderConfig{APIKey: k} }},
		"ANTHROPIC_API_KEY":  {"anthropic", provreg.AIProviderAnthropic, func(k string) any { return provreg.AnthropicProviderConfig{APIKey: k} }},
		"GOOGLE_API_KEY":     {"google", provreg.AIProviderGoogle, func(k string) any { return provreg.GoogleProviderConfig{APIKey: k} }},
		"MISTRAL_API_KEY":    {"mistral", provreg.AIProviderMistral, func(k string) any { return provreg.MistralProviderConfig{APIKey: k} }},
		"GROQ_API_KEY":       {"groq", provreg.AIProviderGroq, func(k string) any { return provreg.GroqProviderConfig{APIKey: k} }},
		"DEEPSEEK_API_KEY":   {"deepseek", provreg.AIProviderDeepSeek, func(k string) any { return provreg.DeepSeekProviderConfig{APIKey: k} }},
		"XAI_API_KEY":        {"xai", provreg.AIProviderXAI, func(k string) any { return provreg.XAIProviderConfig{APIKey: k} }},
		"COHERE_API_KEY":     {"cohere", provreg.AIProviderCohere, func(k string) any { return provreg.CohereProviderConfig{APIKey: k} }},
		"PERPLEXITY_API_KEY": {"perplexity", provreg.AIProviderPerplexity, func(k string) any { return provreg.PerplexityProviderConfig{APIKey: k} }},
		"TOGETHER_API_KEY":   {"togetherai", provreg.AIProviderTogetherAI, func(k string) any { return provreg.TogetherAIProviderConfig{APIKey: k} }},
		"FIREWORKS_API_KEY":  {"fireworks", provreg.AIProviderFireworks, func(k string) any { return provreg.FireworksProviderConfig{APIKey: k} }},
		"CEREBRAS_API_KEY":   {"cerebras", provreg.AIProviderCerebras, func(k string) any { return provreg.CerebrasProviderConfig{APIKey: k} }},
	}

	for envKey, mapping := range mappings {
		if _, explicit := cfg.AIProviders[mapping.name]; explicit {
			continue
		}
		apiKey := ""
		if v, ok := cfg.EnvVars[envKey]; ok && v != "" {
			apiKey = v
		} else {
			apiKey = os.Getenv(envKey)
		}
		if apiKey == "" {
			continue
		}
		cfg.AIProviders[mapping.name] = provreg.AIProviderRegistration{
			Type:   mapping.typ,
			Config: mapping.make(apiKey),
		}
	}
}

// ExtractProviderCredentials extracts APIKey and BaseURL from a typed provider
// registration.
func ExtractProviderCredentials(reg provreg.AIProviderRegistration) struct{ APIKey, BaseURL string } {
	if cp, ok := reg.Config.(types.CredentialProvider); ok {
		apiKey, baseURL := cp.ProviderCredentials()
		return struct{ APIKey, BaseURL string }{apiKey, baseURL}
	}
	return struct{ APIKey, BaseURL string }{}
}

func providerKeyMapping(configured map[string]string) map[string]string {
	if configured == nil {
		configured = defaultProviderKeyMapping
	}
	out := make(map[string]string, len(configured))
	for k, v := range configured {
		out[k] = v
	}
	return out
}

var defaultProviderKeyMapping = map[string]string{
	"OPENAI_API_KEY":    "openai",
	"ANTHROPIC_API_KEY": "anthropic",
	"GOOGLE_API_KEY":    "google",
	"MISTRAL_API_KEY":   "mistral",
	"GROQ_API_KEY":      "groq",
	"DEEPSEEK_API_KEY":  "deepseek",
	"XAI_API_KEY":       "xai",
	"COHERE_API_KEY":    "cohere",
}
