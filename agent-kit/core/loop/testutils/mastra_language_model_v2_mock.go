// Ported from: packages/core/src/loop/test-utils/MastraLanguageModelV2Mock.ts
package testutils

// ---------------------------------------------------------------------------
// Stub types for unported packages
// ---------------------------------------------------------------------------

// LanguageModelV2CallOptions is a stub for @ai-sdk/provider-v5.LanguageModelV2CallOptions.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V2/V5 types remain local stubs.
type LanguageModelV2CallOptions = map[string]any

// ---------------------------------------------------------------------------
// MastraLanguageModelV2MockConfig
// ---------------------------------------------------------------------------

// MastraLanguageModelV2MockConfig holds optional configuration for the V2 mock.
type MastraLanguageModelV2MockConfig struct {
	Provider      string
	ModelID       string
	SupportedURLs map[string][]string
	DoGenerate    func(options map[string]any) (*DoGenerateResult, error)
	DoStream      func(options map[string]any) (*DoStreamResult, error)
}

// DoGenerateResult holds the result from doGenerate.
type DoGenerateResult struct {
	Content      []map[string]any `json:"content"`
	FinishReason string           `json:"finishReason"`
	Usage        UsageV2          `json:"usage"`
	Warnings     []LanguageModelV2CallWarning `json:"warnings,omitempty"`
	Request      *RequestBody     `json:"request,omitempty"`
	Response     *DoGenerateResponseMeta `json:"response,omitempty"`
}

// DoGenerateResponseMeta holds metadata from a generate response.
type DoGenerateResponseMeta struct {
	ID        string `json:"id"`
	ModelID   string `json:"modelId"`
	Timestamp any    `json:"timestamp"`
}

// DoStreamResult holds the result from doStream.
type DoStreamResult struct {
	Stream   <-chan LanguageModelV2StreamPart
	Request  *RequestBody
	Response *ResponseHeaders
	Warnings []LanguageModelV2CallWarning
}

// ---------------------------------------------------------------------------
// MastraLanguageModelV2Mock
// ---------------------------------------------------------------------------

// MastraLanguageModelV2Mock implements a mock of MastraLanguageModelV2 for
// testing. It wraps a MockLanguageModelV2 and delegates through
// AISDKV5LanguageModel, matching the TS class structure.
//
// In Go we simplify this by directly implementing the mock methods since
// the real AISDKV5LanguageModel and MockLanguageModelV2 are not yet ported.
type MastraLanguageModelV2Mock struct {
	SpecificationVersion string
	Provider             string
	ModelID              string
	SupportedURLs        map[string][]string

	doGenerate func(options map[string]any) (*DoGenerateResult, error)
	doStream   func(options map[string]any) (*DoStreamResult, error)

	// DoGenerateCalls records all calls to DoGenerate for test assertions.
	DoGenerateCalls []LanguageModelV2CallOptions
	// DoStreamCalls records all calls to DoStream for test assertions.
	DoStreamCalls []LanguageModelV2CallOptions
}

// NewMastraLanguageModelV2Mock creates a new V2 language model mock.
func NewMastraLanguageModelV2Mock(config MastraLanguageModelV2MockConfig) *MastraLanguageModelV2Mock {
	provider := config.Provider
	if provider == "" {
		provider = "mock-provider"
	}
	modelID := config.ModelID
	if modelID == "" {
		modelID = "mock-model-id"
	}

	return &MastraLanguageModelV2Mock{
		SpecificationVersion: "v2",
		Provider:             provider,
		ModelID:              modelID,
		SupportedURLs:        config.SupportedURLs,
		doGenerate:           config.DoGenerate,
		doStream:             config.DoStream,
	}
}

// DoGenerate invokes the mock's doGenerate function, recording the call.
func (m *MastraLanguageModelV2Mock) DoGenerate(options LanguageModelV2CallOptions) (*DoGenerateResult, error) {
	m.DoGenerateCalls = append(m.DoGenerateCalls, options)
	if m.doGenerate == nil {
		return &DoGenerateResult{}, nil
	}
	return m.doGenerate(options)
}

// DoStream invokes the mock's doStream function, recording the call.
func (m *MastraLanguageModelV2Mock) DoStream(options LanguageModelV2CallOptions) (*DoStreamResult, error) {
	m.DoStreamCalls = append(m.DoStreamCalls, options)
	if m.doStream == nil {
		return &DoStreamResult{}, nil
	}
	return m.doStream(options)
}

// GetModelID returns the mock model's ID.
func (m *MastraLanguageModelV2Mock) GetModelID() string {
	return m.ModelID
}

// GetSpecificationVersion returns the specification version.
func (m *MastraLanguageModelV2Mock) GetSpecificationVersion() string {
	return m.SpecificationVersion
}

// GetProvider returns the provider name.
func (m *MastraLanguageModelV2Mock) GetProvider() string {
	return m.Provider
}
