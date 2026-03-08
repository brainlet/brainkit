// Ported from: packages/core/src/loop/test-utils/MastraLanguageModelV3Mock.ts
package testutils

// ---------------------------------------------------------------------------
// Simplified test types (V3-specific)
//
// The real V3 types are fully ported in ai-kit:
//   - languagemodel.CallOptions  (brainlink/experiments/ai-kit/provider/languagemodel)
//   - languagemodel.StreamPart   (brainlink/experiments/ai-kit/provider/languagemodel)
//   - shared.Warning             (brainlink/experiments/ai-kit/provider/shared)
//
// These mocks use map[string]any for testing flexibility. Production code
// should use the real ai-kit types via the aisdk/v6 adapter layer.
// ---------------------------------------------------------------------------

// LanguageModelV3CallOptions is a simplified test type for V3 call options.
// See ai-kit languagemodel.CallOptions for the real type.
type LanguageModelV3CallOptions = map[string]any

// LanguageModelV3StreamPart is a simplified test type for V3 stream parts.
// See ai-kit languagemodel.StreamPart for the real type.
type LanguageModelV3StreamPart = map[string]any

// SharedV3Warning is a simplified test type for V3 warnings.
// See ai-kit shared.Warning for the real type.
type SharedV3Warning = map[string]any

// ---------------------------------------------------------------------------
// V3 Results
// ---------------------------------------------------------------------------

// FinishReasonV3 holds a V3 finish reason with unified and raw values.
type FinishReasonV3 struct {
	Unified string `json:"unified"`
	Raw     string `json:"raw"`
}

// DoGenerateResultV3 holds the result from a V3 doGenerate call.
type DoGenerateResultV3 struct {
	Content      []map[string]any `json:"content"`
	FinishReason FinishReasonV3   `json:"finishReason"`
	Usage        UsageV3          `json:"usage"`
	Warnings     []SharedV3Warning `json:"warnings,omitempty"`
	Request      *RequestBody     `json:"request,omitempty"`
	Response     *DoGenerateResponseMeta `json:"response,omitempty"`
}

// DoStreamResultV3 holds the result from a V3 doStream call.
type DoStreamResultV3 struct {
	Stream   <-chan LanguageModelV3StreamPart
	Request  *RequestBody
	Response *ResponseHeaders
	Warnings []SharedV3Warning
}

// ---------------------------------------------------------------------------
// MastraLanguageModelV3MockConfig
// ---------------------------------------------------------------------------

// MastraLanguageModelV3MockConfig holds optional configuration for the V3 mock.
type MastraLanguageModelV3MockConfig struct {
	Provider      string
	ModelID       string
	SupportedURLs map[string][]string
	DoGenerate    func(options map[string]any) (*DoGenerateResultV3, error)
	DoStream      func(options map[string]any) (*DoStreamResultV3, error)
}

// ---------------------------------------------------------------------------
// MastraLanguageModelV3Mock
// ---------------------------------------------------------------------------

// MastraLanguageModelV3Mock implements a mock of MastraLanguageModelV3 for
// testing. It wraps a MockLanguageModelV3 and delegates through
// AISDKV6LanguageModel, matching the TS class structure.
//
// In Go we simplify by directly implementing the mock methods since the
// real AISDKV6LanguageModel and MockLanguageModelV3 are not yet ported.
type MastraLanguageModelV3Mock struct {
	SpecificationVersion string
	Provider             string
	ModelID              string
	SupportedURLs        map[string][]string

	doGenerate func(options map[string]any) (*DoGenerateResultV3, error)
	doStream   func(options map[string]any) (*DoStreamResultV3, error)

	// DoGenerateCalls records all calls to DoGenerate for test assertions.
	DoGenerateCalls []LanguageModelV3CallOptions
	// DoStreamCalls records all calls to DoStream for test assertions.
	DoStreamCalls []LanguageModelV3CallOptions
}

// NewMastraLanguageModelV3Mock creates a new V3 language model mock.
func NewMastraLanguageModelV3Mock(config MastraLanguageModelV3MockConfig) *MastraLanguageModelV3Mock {
	provider := config.Provider
	if provider == "" {
		provider = "mock-provider"
	}
	modelID := config.ModelID
	if modelID == "" {
		modelID = "mock-model-id"
	}

	return &MastraLanguageModelV3Mock{
		SpecificationVersion: "v3",
		Provider:             provider,
		ModelID:              modelID,
		SupportedURLs:        config.SupportedURLs,
		doGenerate:           config.DoGenerate,
		doStream:             config.DoStream,
	}
}

// DoGenerate invokes the mock's doGenerate function, recording the call.
func (m *MastraLanguageModelV3Mock) DoGenerate(options LanguageModelV3CallOptions) (*DoGenerateResultV3, error) {
	m.DoGenerateCalls = append(m.DoGenerateCalls, options)
	if m.doGenerate == nil {
		return &DoGenerateResultV3{}, nil
	}
	return m.doGenerate(options)
}

// DoStream invokes the mock's doStream function, recording the call.
func (m *MastraLanguageModelV3Mock) DoStream(options LanguageModelV3CallOptions) (*DoStreamResultV3, error) {
	m.DoStreamCalls = append(m.DoStreamCalls, options)
	if m.doStream == nil {
		return &DoStreamResultV3{}, nil
	}
	return m.doStream(options)
}

// GetModelID returns the mock model's ID.
func (m *MastraLanguageModelV3Mock) GetModelID() string {
	return m.ModelID
}

// GetSpecificationVersion returns the specification version.
func (m *MastraLanguageModelV3Mock) GetSpecificationVersion() string {
	return m.SpecificationVersion
}

// GetProvider returns the provider name.
func (m *MastraLanguageModelV3Mock) GetProvider() string {
	return m.Provider
}
