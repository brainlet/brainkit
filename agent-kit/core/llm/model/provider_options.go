// Ported from: packages/core/src/llm/model/provider-options.ts
package model

// ---------------------------------------------------------------------------
// Provider-specific option types (stubs)
// ---------------------------------------------------------------------------

// AnthropicProviderOptions is a stub for @ai-sdk/anthropic-v5 options.
// ai-kit only ported the @ai-sdk/provider layer; vendor-specific options remain local stubs.
type AnthropicProviderOptions = map[string]any

// DeepSeekChatOptions is a stub for @ai-sdk/deepseek-v5 options.
// ai-kit only ported the @ai-sdk/provider layer; vendor-specific options remain local stubs.
type DeepSeekChatOptions = map[string]any

// GoogleGenerativeAIProviderOptions is a stub for @ai-sdk/google-v5 options.
// ai-kit only ported the @ai-sdk/provider layer; vendor-specific options remain local stubs.
type GoogleGenerativeAIProviderOptions = map[string]any

// OpenAIResponsesProviderOptions is a stub for @ai-sdk/openai-v5 options.
// ai-kit only ported the @ai-sdk/provider layer; vendor-specific options remain local stubs.
type OpenAIResponsesProviderOptions = map[string]any

// XaiProviderOptions is a stub for @ai-sdk/xai-v5 options.
// ai-kit only ported the @ai-sdk/provider layer; vendor-specific options remain local stubs.
type XaiProviderOptions = map[string]any

// ---------------------------------------------------------------------------
// Aliases
// ---------------------------------------------------------------------------

// GoogleProviderOptions is an alias for GoogleGenerativeAIProviderOptions.
type GoogleProviderOptions = GoogleGenerativeAIProviderOptions

// DeepSeekProviderOptions is an alias for DeepSeekChatOptions.
type DeepSeekProviderOptions = DeepSeekChatOptions

// ---------------------------------------------------------------------------
// OpenAI transport types
// ---------------------------------------------------------------------------

// OpenAITransport selects the transport used for streaming responses.
//   - "fetch" uses HTTP streaming.
//   - "websocket" uses the OpenAI Responses WebSocket API when supported.
//   - "auto" chooses WebSocket when supported, otherwise falls back to fetch.
type OpenAITransport string

const (
	OpenAITransportAuto      OpenAITransport = "auto"
	OpenAITransportWebSocket OpenAITransport = "websocket"
	OpenAITransportFetch     OpenAITransport = "fetch"
)

// OpenAIWebSocketOptions holds WebSocket-specific configuration for OpenAI streaming.
type OpenAIWebSocketOptions struct {
	// URL is the WebSocket endpoint URL.
	// Default: "wss://api.openai.com/v1/responses"
	URL string `json:"url,omitempty"`
	// Headers contains additional headers sent when establishing the WebSocket connection.
	// Authorization and OpenAI-Beta are managed internally.
	Headers map[string]string `json:"headers,omitempty"`
	// CloseOnFinish controls whether to close the WebSocket connection when the stream finishes.
	// Default: true
	CloseOnFinish *bool `json:"closeOnFinish,omitempty"`
}

// OpenAIProviderOptions extends OpenAIResponsesProviderOptions with transport options.
type OpenAIProviderOptions struct {
	// Transport selects the transport used for streaming responses.
	Transport OpenAITransport `json:"transport,omitempty"`
	// WebSocket holds WebSocket-specific configuration.
	WebSocket *OpenAIWebSocketOptions `json:"websocket,omitempty"`
	// Extra holds any additional provider-specific options.
	Extra map[string]any `json:"-"`
}

// ---------------------------------------------------------------------------
// ProviderOptions
// ---------------------------------------------------------------------------

// ProviderOptions holds provider-specific options for AI SDK models.
// Provider options are keyed by provider ID and contain provider-specific configuration.
type ProviderOptions struct {
	Anthropic map[string]any `json:"anthropic,omitempty"`
	DeepSeek  map[string]any `json:"deepseek,omitempty"`
	Google    map[string]any `json:"google,omitempty"`
	OpenAI    map[string]any `json:"openai,omitempty"`
	XAI       map[string]any `json:"xai,omitempty"`
	// Extra holds options for any other providers not explicitly listed.
	Extra map[string]map[string]any `json:"-"`
}
