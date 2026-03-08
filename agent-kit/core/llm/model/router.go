// Ported from: packages/core/src/llm/model/router.ts
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

var openAIWSAllowlist = map[string]bool{"openai": true}

const openAIAPIHost = "api.openai.com"

// ProvidersWithInstalledPackages lists providers that use their
// corresponding AI SDK package instead of openai-compat endpoints.
//
// Ported from: packages/core/src/llm/model/gateways/constants.ts
var ProvidersWithInstalledPackages = []string{
	"anthropic",
	"cerebras",
	"deepinfra",
	"deepseek",
	"google",
	"groq",
	"mistral",
	"openai",
	"openrouter",
	"perplexity",
	"togetherai",
	"vercel",
	"xai",
}

// ExcludedProviders lists providers that don't show up in the model router.
var ExcludedProviders = []string{"github-copilot"}

// ---------------------------------------------------------------------------
// StreamTransport stub
// ---------------------------------------------------------------------------

// StreamTransport is a stub for stream.StreamTransport.
// STUB REASON: The real stream.StreamTransport has fields (Type string, CloseFunc func(),
// CloseOnFinish bool) with a Close() method. This stub omits CloseFunc. Also, stream
// has its own stub of StreamTransport in stream/base/output.go with yet different fields
// (CloseOnFinish bool, Close func()). Structural mismatch across packages.
type StreamTransport struct {
	Type          string `json:"type"`
	CloseOnFinish bool   `json:"closeOnFinish"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isLanguageModelV3 checks if a model is a LanguageModelV3 (AI SDK v6).
func isLanguageModelV3(model GatewayLanguageModel) bool {
	return model.SpecificationVersion() == "v3"
}

func getOpenAITransport(providerOptions map[string]any) (OpenAITransport, *OpenAIWebSocketOptions) {
	if providerOptions == nil {
		return OpenAITransportFetch, nil
	}

	openaiOpts, ok := providerOptions["openai"].(map[string]any)
	if !ok {
		return OpenAITransportFetch, nil
	}

	transport := OpenAITransportFetch
	if t, ok := openaiOpts["transport"].(string); ok {
		transport = OpenAITransport(t)
	}

	var wsOpts *OpenAIWebSocketOptions
	if ws, ok := openaiOpts["websocket"].(map[string]any); ok {
		wsOpts = &OpenAIWebSocketOptions{}
		if u, ok := ws["url"].(string); ok {
			wsOpts.URL = u
		}
		if h, ok := ws["headers"].(map[string]string); ok {
			wsOpts.Headers = h
		}
		if c, ok := ws["closeOnFinish"].(bool); ok {
			wsOpts.CloseOnFinish = &c
		}
	}

	return transport, wsOpts
}

func isOpenAIBaseURL(baseURL string) bool {
	if baseURL == "" {
		return true
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	return u.Hostname() == openAIAPIHost
}

func stableHeaderKey(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	entries := make([][2]string, 0, len(headers))
	for k, v := range headers {
		entries = append(entries, [2]string{k, v})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i][0] < entries[j][0]
	})
	var sb strings.Builder
	sb.WriteString("[")
	for i, e := range entries {
		if i > 0 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, "[%q,%q]", e[0], e[1])
	}
	sb.WriteString("]")
	return sb.String()
}

// ---------------------------------------------------------------------------
// ModelRouterLanguageModel
// ---------------------------------------------------------------------------

// ModelRouterLanguageModel implements MastraLanguageModelV2 by routing model
// requests through the gateway system. It normalises OpenAI-compatible
// config strings (e.g., "openai/gpt-4o") or config objects into a resolved
// model via the gateway.
type ModelRouterLanguageModel struct {
	mu sync.RWMutex

	// Public, read-only fields (matching the TS class).
	specificationVersion         string
	defaultObjectGenerationMode  string
	supportsStructuredOutputs    bool
	supportsImageURLs            bool

	modelID  string
	provider string

	config  modelRouterConfig
	gateway MastraModelGateway

	lastStreamTransport *StreamTransport
}

type modelRouterConfig struct {
	ID       string            `json:"id"`
	RouterID string            `json:"routerId"`
	URL      string            `json:"url,omitempty"`
	APIKey   string            `json:"apiKey,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// Static caches shared across all ModelRouterLanguageModel instances.
var (
	modelInstancesMu sync.RWMutex
	modelInstances   = make(map[string]GatewayLanguageModel)
)

// NewModelRouterLanguageModel creates a new ModelRouterLanguageModel.
// config can be a ModelRouterModelID (string) or OpenAICompatibleConfig.
func NewModelRouterLanguageModel(
	config any,
	customGateways []MastraModelGateway,
) (*ModelRouterLanguageModel, error) {
	// Normalize config to always have an 'id' field for routing
	var normalizedID string
	var cfgURL, cfgAPIKey string
	var cfgHeaders map[string]string

	switch c := config.(type) {
	case string:
		normalizedID = c
	case OpenAICompatibleConfig:
		if c.HasProviderModel() {
			normalizedID = c.ProviderID + "/" + c.ModelID
		} else {
			normalizedID = c.ID
		}
		cfgURL = c.URL
		cfgAPIKey = c.APIKey
		cfgHeaders = c.Headers
	default:
		return nil, fmt.Errorf("invalid model config type: %T", config)
	}

	parsedConfig := modelRouterConfig{
		ID:       normalizedID,
		RouterID: normalizedID,
		URL:      cfgURL,
		APIKey:   cfgAPIKey,
		Headers:  cfgHeaders,
	}

	// Resolve gateway using the normalized ID
	allGateways := append(customGateways, defaultGateways()...)
	gateway, err := FindGatewayForModel(normalizedID, allGateways)
	if err != nil {
		return nil, err
	}

	// Extract provider from ID
	gatewayPrefix := ""
	if gateway.ID() != "models.dev" {
		gatewayPrefix = gateway.ID()
	}

	parsed, err := ParseModelRouterID(normalizedID, gatewayPrefix)
	if err != nil {
		return nil, err
	}

	providerName := parsed.ProviderID
	if providerName == "" {
		providerName = "openai-compatible"
	}

	if parsed.ProviderID != "" && parsed.ModelID != normalizedID {
		parsedConfig.ID = parsed.ModelID
	}

	return &ModelRouterLanguageModel{
		specificationVersion:        "v2",
		defaultObjectGenerationMode: "json",
		supportsStructuredOutputs:   true,
		supportsImageURLs:           true,
		modelID:                     parsedConfig.ID,
		provider:                    providerName,
		config:                      parsedConfig,
		gateway:                     gateway,
	}, nil
}

// defaultGateways returns the default set of model gateways.
// In the TS source this is [NetlifyGateway, ModelsDevGateway].
// Until those are ported, this returns an empty slice.
// TODO: populate with real gateway implementations once ported.
func defaultGateways() []MastraModelGateway {
	return nil
}

// SpecificationVersion implements MastraLanguageModel.
func (m *ModelRouterLanguageModel) SpecificationVersion() string { return m.specificationVersion }

// Provider implements MastraLanguageModel.
func (m *ModelRouterLanguageModel) Provider() string { return m.provider }

// ModelID implements MastraLanguageModel.
func (m *ModelRouterLanguageModel) ModelID() string { return m.modelID }

// SupportsStructuredOutputs returns whether the model supports structured outputs.
func (m *ModelRouterLanguageModel) SupportsStructuredOutputs() bool { return m.supportsStructuredOutputs }

// SupportsImageURLs returns whether the model supports image URLs.
func (m *ModelRouterLanguageModel) SupportsImageURLs() bool { return m.supportsImageURLs }

// GetStreamTransport returns the last stream transport used (if any).
func (m *ModelRouterLanguageModel) GetStreamTransport() *StreamTransport {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastStreamTransport
}

// DoGenerate performs a non-streaming generation via the resolved gateway model.
//
// TS:
//
//	async doGenerate(options: LanguageModelV2CallOptions): Promise<StreamResult> {
//	  ...
//	  if (isLanguageModelV3(model)) {
//	    const aiSDKV6Model = new AISDKV6LanguageModel(model);
//	    return aiSDKV6Model.doGenerate(options as any) as unknown as Promise<StreamResult>;
//	  }
//	  const aiSDKV5Model = new AISDKV5LanguageModel(model);
//	  return aiSDKV5Model.doGenerate(options);
//	}
func (m *ModelRouterLanguageModel) DoGenerate(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error) {
	apiKey, err := m.resolveAPIKey()
	if err != nil {
		// TS returns an error stream; in Go we return the error directly.
		return LanguageModelV2StreamResult{}, err
	}

	model, err := m.resolveLanguageModel(resolveModelArgs{
		APIKey:  apiKey,
		Headers: m.config.Headers,
	})
	if err != nil {
		return LanguageModelV2StreamResult{}, err
	}

	// Handle both V2 and V3 models
	if isLanguageModelV3(model) {
		// TS: const aiSDKV6Model = new AISDKV6LanguageModel(model);
		//     return aiSDKV6Model.doGenerate(options as any) as unknown as Promise<StreamResult>;
		v3Model, ok := model.(LanguageModelV3)
		if !ok {
			return LanguageModelV2StreamResult{}, fmt.Errorf("resolved model claims v3 but does not implement LanguageModelV3")
		}
		wrapper := NewAISDKV6LanguageModelStub(v3Model)
		result, err := wrapper.DoGenerate(LanguageModelV3CallOptions{
			ProviderOptions: options.ProviderOptions,
		})
		if err != nil {
			return LanguageModelV2StreamResult{}, err
		}
		// Cast V3 stream result to V2 format - the stream contents are compatible at runtime
		return LanguageModelV2StreamResult{Stream: result.Stream}, nil
	}

	// TS: const aiSDKV5Model = new AISDKV5LanguageModel(model);
	//     return aiSDKV5Model.doGenerate(options);
	v2Model, ok := model.(LanguageModelV2)
	if !ok {
		return LanguageModelV2StreamResult{}, fmt.Errorf("resolved model does not implement LanguageModelV2")
	}
	wrapper := NewAISDKV5LanguageModel(v2Model)
	return wrapper.DoGenerate(options)
}

// DoStream performs a streaming generation via the resolved gateway model.
//
// TS:
//
//	async doStream(options: LanguageModelV2CallOptions): Promise<StreamResult> {
//	  ...
//	  const { transport, websocket } = getOpenAITransport(options.providerOptions);
//	  const requestedTransport = transport === 'auto' ? 'websocket' : transport;
//	  const allowWebSocket = requestedTransport === 'websocket' && OPENAI_WS_ALLOWLIST.has(this.provider) && !this.config.url && this.gateway.id === 'models.dev';
//	  const resolvedTransport = allowWebSocket ? 'websocket' : 'fetch';
//	  ...
//	  if (isLanguageModelV3(model)) {
//	    const aiSDKV6Model = new AISDKV6LanguageModel(model);
//	    return aiSDKV6Model.doStream(options as any) as unknown as Promise<StreamResult>;
//	  }
//	  const aiSDKV5Model = new AISDKV5LanguageModel(model);
//	  return aiSDKV5Model.doStream(options);
//	}
func (m *ModelRouterLanguageModel) DoStream(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error) {
	apiKey, err := m.resolveAPIKey()
	if err != nil {
		return LanguageModelV2StreamResult{}, err
	}

	// Resolve transport from provider options
	// TS: const { transport, websocket } = getOpenAITransport(options.providerOptions);
	transport, wsOpts := getOpenAITransport(options.ProviderOptions)

	// TS: const requestedTransport = transport === 'auto' ? 'websocket' : transport;
	requestedTransport := transport
	if requestedTransport == OpenAITransportAuto {
		requestedTransport = OpenAITransportWebSocket
	}

	// TS: const allowWebSocket = requestedTransport === 'websocket' && OPENAI_WS_ALLOWLIST.has(this.provider) && !this.config.url && this.gateway.id === 'models.dev';
	allowWebSocket := requestedTransport == OpenAITransportWebSocket &&
		openAIWSAllowlist[m.provider] &&
		m.config.URL == "" &&
		m.gateway.ID() == "models.dev"

	// TS: const resolvedTransport = allowWebSocket ? 'websocket' : 'fetch';
	resolvedTransport := OpenAITransportFetch
	if allowWebSocket {
		resolvedTransport = OpenAITransportWebSocket
	}

	model, err := m.resolveLanguageModel(resolveModelArgs{
		APIKey:          apiKey,
		Headers:         m.config.Headers,
		Transport:       resolvedTransport,
		OpenAIWebSocket: wsOpts,
	})
	if err != nil {
		return LanguageModelV2StreamResult{}, err
	}

	// Handle both V2 and V3 models
	if isLanguageModelV3(model) {
		v3Model, ok := model.(LanguageModelV3)
		if !ok {
			return LanguageModelV2StreamResult{}, fmt.Errorf("resolved model claims v3 but does not implement LanguageModelV3")
		}
		wrapper := NewAISDKV6LanguageModelStub(v3Model)
		result, err := wrapper.DoStream(LanguageModelV3CallOptions{
			ProviderOptions: options.ProviderOptions,
		})
		if err != nil {
			return LanguageModelV2StreamResult{}, err
		}
		// Cast V3 stream result to V2 format - the stream contents are compatible at runtime
		return LanguageModelV2StreamResult{Stream: result.Stream}, nil
	}

	v2Model, ok := model.(LanguageModelV2)
	if !ok {
		return LanguageModelV2StreamResult{}, fmt.Errorf("resolved model does not implement LanguageModelV2")
	}
	wrapper := NewAISDKV5LanguageModel(v2Model)
	return wrapper.DoStream(options)
}

// resolveAPIKey resolves the API key from config or the gateway.
func (m *ModelRouterLanguageModel) resolveAPIKey() (string, error) {
	if m.config.URL != "" {
		if m.config.APIKey != "" {
			return m.config.APIKey, nil
		}
		return "", nil
	}
	if m.config.APIKey != "" {
		return m.config.APIKey, nil
	}
	return m.gateway.GetAPIKey(m.config.RouterID)
}

type resolveModelArgs struct {
	APIKey          string
	Headers         map[string]string
	Transport       OpenAITransport
	OpenAIWebSocket *OpenAIWebSocketOptions
}

// resolveLanguageModel resolves the underlying language model with caching.
func (m *ModelRouterLanguageModel) resolveLanguageModel(args resolveModelArgs) (GatewayLanguageModel, error) {
	gatewayPrefix := ""
	if m.gateway.ID() != "models.dev" {
		gatewayPrefix = m.gateway.ID()
	}

	parsed, err := ParseModelRouterID(m.config.RouterID, gatewayPrefix)
	if err != nil {
		return nil, err
	}

	resolvedTransport := args.Transport
	if resolvedTransport == "" {
		resolvedTransport = OpenAITransportFetch
	}

	wsKey := ""
	if resolvedTransport == OpenAITransportWebSocket && args.OpenAIWebSocket != nil {
		wsKey = fmt.Sprintf("%s:%s", args.OpenAIWebSocket.URL, stableHeaderKey(args.OpenAIWebSocket.Headers))
	}

	// Build cache key
	h := sha256.New()
	h.Write([]byte(m.gateway.ID()))
	h.Write([]byte(parsed.ModelID))
	h.Write([]byte(parsed.ProviderID))
	h.Write([]byte(args.APIKey))
	h.Write([]byte(m.config.URL))
	h.Write([]byte(stableHeaderKey(args.Headers)))
	h.Write([]byte(string(resolvedTransport)))
	h.Write([]byte(wsKey))
	key := hex.EncodeToString(h.Sum(nil))

	// Check cache
	modelInstancesMu.RLock()
	if cached, ok := modelInstances[key]; ok {
		modelInstancesMu.RUnlock()
		return cached, nil
	}
	modelInstancesMu.RUnlock()

	// Resolve from gateway
	model, err := m.gateway.ResolveLanguageModel(ResolveLanguageModelArgs{
		ModelID:    parsed.ModelID,
		ProviderID: parsed.ProviderID,
		APIKey:     args.APIKey,
		Headers:    args.Headers,
	})
	if err != nil {
		return nil, err
	}

	// Cache
	modelInstancesMu.Lock()
	modelInstances[key] = model
	modelInstancesMu.Unlock()

	return model, nil
}
