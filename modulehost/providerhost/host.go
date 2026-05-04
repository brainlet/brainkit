// Package providerhost owns provider bootstrap, probing, and runtime refresh
// orchestration for one Kit runtime.
package providerhost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit/internal/types"
	provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
)

// Hooks are the optional runtime callbacks used for probes that need JS-side
// factories and for refreshing provider credentials inside the JS runtime.
type Hooks struct {
	HasJSRuntime    func() bool
	EvalTS          func(context.Context, string, string) (string, error)
	CallJS          func(context.Context, string, any) (json.RawMessage, error)
	ShutdownContext context.Context
}

// Manager owns the provider registry plus runtime-backed provider operations.
type Manager struct {
	mu                 sync.Mutex
	providers          *provreg.ProviderRegistry
	hasJSRuntime       func() bool
	evalTS             func(context.Context, string, string) (string, error)
	callJS             func(context.Context, string, any) (json.RawMessage, error)
	ctx                context.Context
	cancel             context.CancelFunc
	shutdownCtx        context.Context
	operations         sync.WaitGroup
	activeOperations   atomic.Int64
	closeWait          sync.Once
	closeDone          chan struct{}
	providerKeyMapping map[string]string
	explicitKeyMapping bool
	closing            bool
	closed             bool
}

// DebugSnapshot reports provider-host lifecycle counters for inspect/debug
// surfaces.
type DebugSnapshot struct {
	AIProviders      int
	VectorStores     int
	Storages         int
	Closing          bool
	Closed           bool
	ActiveProbes     int64
	ActiveOperations int64
}

// NewManager creates a provider host for one Kit runtime and registers the
// configured AI providers.
func NewManager(cfg types.KernelConfig, hooks Hooks) (*Manager, error) {
	if hooks.HasJSRuntime == nil {
		hooks.HasJSRuntime = func() bool { return false }
	}
	providers := provreg.New(cfg.Probe)
	ctx, cancel := context.WithCancel(context.Background())
	manager := &Manager{
		providers:          providers,
		hasJSRuntime:       hooks.HasJSRuntime,
		evalTS:             hooks.EvalTS,
		callJS:             hooks.CallJS,
		ctx:                ctx,
		cancel:             cancel,
		shutdownCtx:        hooks.ShutdownContext,
		closeDone:          make(chan struct{}),
		providerKeyMapping: providerKeyMapping(cfg.ProviderKeyMapping),
		explicitKeyMapping: cfg.ProviderKeyMapping != nil,
	}
	for name, reg := range cfg.AIProviders {
		if err := providers.RegisterAIProvider(name, reg); err != nil {
			_ = providers.Close()
			return nil, err
		}
	}
	return manager, nil
}

// Close releases provider-host owned background work.
func (m *Manager) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.CloseContext(ctx)
}

// CloseContext releases provider-host owned background work under
// caller-owned lifecycle cancellation.
func (m *Manager) CloseContext(ctx context.Context) error {
	if m == nil || m.providers == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	m.mu.Lock()
	m.closing = true
	if !m.closed {
		m.closed = true
		if m.cancel != nil {
			m.cancel()
		}
	}
	closeDone := m.closeDone
	m.closeWait.Do(func() {
		go func() {
			m.operations.Wait()
			close(closeDone)
		}()
	})
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.closing = false
		m.mu.Unlock()
	}()
	var err error
	err = errors.Join(err, m.providers.CloseContext(ctx))
	select {
	case <-closeDone:
	case <-ctx.Done():
		err = errors.Join(err, fmt.Errorf("provider host operations: %w", ctx.Err()))
	}
	return err
}

// Registry returns the shared provider/storage/vector registry.
func (m *Manager) Registry() *provreg.ProviderRegistry {
	if m == nil {
		return nil
	}
	return m.providers
}

// DebugSnapshot returns a read-only provider-host lifecycle view.
func (m *Manager) DebugSnapshot() DebugSnapshot {
	if m == nil || m.providers == nil {
		return DebugSnapshot{}
	}
	m.mu.Lock()
	closing := m.closing
	closed := m.closed
	m.mu.Unlock()
	return DebugSnapshot{
		AIProviders:      len(m.providers.ListAIProviders()),
		VectorStores:     len(m.providers.ListVectorStores()),
		Storages:         len(m.providers.ListStorages()),
		Closing:          closing,
		Closed:           closed,
		ActiveProbes:     m.providers.ActiveProbes(),
		ActiveOperations: m.activeOperations.Load(),
	}
}

// ProbeAIProvider runs a live HTTP probe against a registered AI provider.
func (m *Manager) ProbeAIProvider(name string) provreg.ProbeResult {
	return m.ProbeAIProviderContext(context.Background(), name)
}

// ProbeAIProviderContext runs a live HTTP probe under provider-host lifecycle
// cancellation.
func (m *Manager) ProbeAIProviderContext(ctx context.Context, name string) provreg.ProbeResult {
	if m == nil || m.providers == nil {
		return provreg.ProbeResult{Error: "provider host not initialized"}
	}
	ctx, done, err := m.beginOperation(ctx)
	if err != nil {
		return provreg.ProbeResult{Error: err.Error()}
	}
	defer done()
	return m.providers.ProbeAIProviderContext(ctx, name)
}

// ProbeVectorStore probes a vector store by instantiating it in the JS runtime
// and calling listIndexes().
func (m *Manager) ProbeVectorStore(name string) provreg.ProbeResult {
	return m.ProbeVectorStoreContext(context.Background(), name)
}

// ProbeVectorStoreContext probes a vector store under caller-owned lifecycle
// cancellation.
func (m *Manager) ProbeVectorStoreContext(ctx context.Context, name string) provreg.ProbeResult {
	if m == nil || m.providers == nil {
		return provreg.ProbeResult{Error: "provider host not initialized"}
	}
	start := time.Now()
	ctx, done, err := m.beginOperation(ctx)
	if err != nil {
		m.providers.UpdateProbeResult("vectorStore", name, false, time.Since(start), err.Error())
		return provreg.ProbeResult{Error: err.Error(), Latency: time.Since(start)}
	}
	defer done()
	if !m.runtimeReady() {
		err := "js runtime not configured"
		m.providers.UpdateProbeResult("vectorStore", name, false, time.Since(start), err)
		return provreg.ProbeResult{Error: err, Latency: time.Since(start)}
	}
	result, err := m.evalTS(ctx, "__probe_vectorstore.ts", fmt.Sprintf(`
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
	return m.ProbeStorageContext(context.Background(), name)
}

// ProbeStorageContext probes a storage backend under caller-owned lifecycle
// cancellation.
func (m *Manager) ProbeStorageContext(ctx context.Context, name string) provreg.ProbeResult {
	if m == nil || m.providers == nil {
		return provreg.ProbeResult{Error: "provider host not initialized"}
	}
	start := time.Now()
	ctx, done, err := m.beginOperation(ctx)
	if err != nil {
		m.providers.UpdateProbeResult("storage", name, false, time.Since(start), err.Error())
		return provreg.ProbeResult{Error: err.Error(), Latency: time.Since(start)}
	}
	defer done()
	if !m.runtimeReady() {
		err := "js runtime not configured"
		m.providers.UpdateProbeResult("storage", name, false, time.Since(start), err)
		return provreg.ProbeResult{Error: err, Latency: time.Since(start)}
	}
	result, err := m.evalTS(ctx, "__probe_storage.ts", fmt.Sprintf(`
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
	m.ProbeAllContext(context.Background())
}

// ProbeAllContext runs probes for all registered providers, vector stores, and
// storages under caller-owned lifecycle cancellation.
func (m *Manager) ProbeAllContext(ctx context.Context) {
	if m == nil || m.providers == nil {
		return
	}
	ctx, done, err := m.beginOperation(ctx)
	if err != nil {
		return
	}
	defer done()
	for _, p := range m.providers.ListAIProviders() {
		if ctx.Err() != nil {
			return
		}
		m.providers.ProbeAIProviderContext(ctx, p.Name)
	}
	if !m.runtimeReady() {
		return
	}
	for _, v := range m.providers.ListVectorStores() {
		if ctx.Err() != nil {
			return
		}
		m.ProbeVectorStoreContext(ctx, v.Name)
	}
	for _, s := range m.providers.ListStorages() {
		if ctx.Err() != nil {
			return
		}
		m.ProbeStorageContext(ctx, s.Name)
	}
}

// RefreshProviderSecret checks whether a secret name maps to a provider
// credential and refreshes registry/runtime provider state under caller-owned
// cancellation.
func (m *Manager) RefreshProviderSecret(ctx context.Context, name, newValue string) error {
	if m == nil {
		return nil
	}
	ctx, done, err := m.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer done()
	provName, ok := m.providerKeyMapping[name]
	if !ok || provName == "" {
		return nil
	}
	updated, err := m.refreshProviderCredential(name, provName, newValue)
	if err != nil {
		return err
	}
	if !updated {
		if m.explicitKeyMapping {
			return fmt.Errorf("provider %q mapped from secret %q is not registered", provName, name)
		}
		return nil
	}
	if !m.jsCallReady() {
		return nil
	}
	if _, err := m.callJS(ctx, "__brainkit.secrets.refreshProvider", map[string]string{"provider": provName}); err != nil {
		return fmt.Errorf("refresh provider %q runtime cache: %w", provName, err)
	}
	return nil
}

// InvalidateRegistryRuntimeCache clears JS runtime-side provider/storage/vector
// caches after the backing registry entry changes. It is a no-op when no JS
// runtime is attached.
func (m *Manager) InvalidateRegistryRuntimeCache(ctx context.Context, category, name string) error {
	if m == nil {
		return nil
	}
	if !m.jsCallReady() {
		return nil
	}
	ctx, done, err := m.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer done()
	if _, err := m.callJS(ctx, "__brainkit.registry.clearCache", map[string]string{
		"category": category,
		"name":     name,
	}); err != nil {
		return fmt.Errorf("clear %s runtime cache %q: %w", category, name, err)
	}
	return nil
}

func (m *Manager) beginOperation(ctx context.Context) (context.Context, func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil {
		return ctx, func() {}, nil
	}
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ctx, func() {}, context.Canceled
	}
	m.operations.Add(1)
	m.activeOperations.Add(1)
	m.mu.Unlock()

	opCtx, cleanup := m.operationContext(ctx)
	done := func() {
		cleanup()
		m.activeOperations.Add(-1)
		m.operations.Done()
	}
	if err := opCtx.Err(); err != nil {
		done()
		return opCtx, func() {}, err
	}
	return opCtx, done, nil
}

func (m *Manager) operationContext(ctx context.Context) (context.Context, func()) {
	if ctx == nil {
		ctx = context.Background()
	}
	if m == nil {
		return ctx, func() {}
	}
	opCtx, cancel := context.WithCancel(ctx)
	var stopManager func() bool
	if m.ctx != nil {
		if m.ctx.Err() != nil {
			cancel()
		}
		stopManager = context.AfterFunc(m.ctx, cancel)
	}
	var stopShutdown func() bool
	if m.shutdownCtx != nil {
		if m.shutdownCtx.Err() != nil {
			cancel()
		}
		stopShutdown = context.AfterFunc(m.shutdownCtx, cancel)
	}
	return opCtx, func() {
		if stopManager != nil {
			stopManager()
		}
		if stopShutdown != nil {
			stopShutdown()
		}
		cancel()
	}
}

func (m *Manager) runtimeReady() bool {
	return m != nil && m.hasJSRuntime != nil && m.hasJSRuntime() && m.evalTS != nil
}

func (m *Manager) jsCallReady() bool {
	return m != nil && m.hasJSRuntime != nil && m.hasJSRuntime() && m.callJS != nil
}

func (m *Manager) refreshProviderCredential(secretName, providerName, newValue string) (bool, error) {
	if m.providers == nil {
		return false, nil
	}
	return m.providers.UpdateAIProvider(providerName, func(reg provreg.AIProviderRegistration) (provreg.AIProviderRegistration, error) {
		config, err := updateCredentialField(reg.Config, secretName, newValue)
		if err != nil {
			return reg, fmt.Errorf("refresh provider %q from secret %q: %w", providerName, secretName, err)
		}
		reg.Config = config
		return reg, nil
	})
}

func updateCredentialField(config any, secretName, newValue string) (any, error) {
	if config == nil {
		return nil, fmt.Errorf("provider config is nil")
	}
	value := reflect.ValueOf(config)
	wasPointer := false
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, fmt.Errorf("provider config is nil")
		}
		wasPointer = true
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("provider config type %T is not a struct", config)
	}

	next := reflect.New(value.Type()).Elem()
	next.Set(value)
	fieldName := credentialFieldName(next, secretName)
	if fieldName == "" {
		return nil, fmt.Errorf("provider config type %T has no refreshable credential field", config)
	}
	field := next.FieldByName(fieldName)
	if !field.CanSet() || field.Kind() != reflect.String {
		return nil, fmt.Errorf("provider config field %s is not settable", fieldName)
	}
	field.SetString(newValue)

	if wasPointer {
		ptr := reflect.New(next.Type())
		ptr.Elem().Set(next)
		return ptr.Interface(), nil
	}
	return next.Interface(), nil
}

func credentialFieldName(value reflect.Value, secretName string) string {
	upper := strings.ToUpper(secretName)
	candidates := make([]string, 0, 8)
	if strings.Contains(upper, "SECRET") {
		candidates = append(candidates, "SecretKey")
	}
	if strings.Contains(upper, "ACCESS") {
		candidates = append(candidates, "AccessKey")
	}
	if strings.Contains(upper, "AUTH") || strings.Contains(upper, "TOKEN") {
		candidates = append(candidates, "AuthToken", "Token")
	}
	if strings.Contains(upper, "API") || strings.Contains(upper, "KEY") {
		candidates = append(candidates, "APIKey")
	}
	candidates = append(candidates, "APIKey", "AuthToken", "Token", "AccessKey", "SecretKey")

	seen := map[string]bool{}
	for _, name := range candidates {
		if seen[name] {
			continue
		}
		seen[name] = true
		field := value.FieldByName(name)
		if field.IsValid() && field.Kind() == reflect.String {
			return name
		}
	}
	return ""
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
