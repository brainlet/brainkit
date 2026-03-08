// Ported from: packages/ai/src/test/mock-language-model-v4.ts
package testutil

import (
	"regexp"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// MockLanguageModelV4 is a test double for a LanguageModel with specificationVersion "v4".
type MockLanguageModelV4 struct {
	ProviderID string
	ModelIDVal string

	supportedUrls func() (map[string][]*regexp.Regexp, error)

	DoGenerateFunc func(options lm.CallOptions) (lm.GenerateResult, error)
	DoStreamFunc   func(options lm.CallOptions) (lm.StreamResult, error)

	DoGenerateCalls []lm.CallOptions
	DoStreamCalls   []lm.CallOptions
}

// MockLanguageModelV4Options configures a MockLanguageModelV4.
type MockLanguageModelV4Options struct {
	Provider      string
	ModelID       string
	SupportedUrls interface{} // map[string][]*regexp.Regexp or func() (map[string][]*regexp.Regexp, error)
	DoGenerate    interface{} // func(lm.CallOptions)(lm.GenerateResult,error) | lm.GenerateResult | []lm.GenerateResult
	DoStream      interface{} // func(lm.CallOptions)(lm.StreamResult,error) | lm.StreamResult | []lm.StreamResult
}

// NewMockLanguageModelV4 creates a new MockLanguageModelV4 with the given options.
func NewMockLanguageModelV4(opts ...MockLanguageModelV4Options) *MockLanguageModelV4 {
	var o MockLanguageModelV4Options
	if len(opts) > 0 {
		o = opts[0]
	}

	m := &MockLanguageModelV4{
		ProviderID:      "mock-provider",
		ModelIDVal:      "mock-model-id",
		DoGenerateCalls: []lm.CallOptions{},
		DoStreamCalls:   []lm.CallOptions{},
	}

	if o.Provider != "" {
		m.ProviderID = o.Provider
	}
	if o.ModelID != "" {
		m.ModelIDVal = o.ModelID
	}

	// supportedUrls
	switch v := o.SupportedUrls.(type) {
	case func() (map[string][]*regexp.Regexp, error):
		m.supportedUrls = v
	case map[string][]*regexp.Regexp:
		m.supportedUrls = func() (map[string][]*regexp.Regexp, error) { return v, nil }
	default:
		m.supportedUrls = func() (map[string][]*regexp.Regexp, error) {
			return map[string][]*regexp.Regexp{}, nil
		}
	}

	// doGenerate
	switch v := o.DoGenerate.(type) {
	case func(lm.CallOptions) (lm.GenerateResult, error):
		m.DoGenerateFunc = v
	case lm.GenerateResult:
		m.DoGenerateFunc = func(_ lm.CallOptions) (lm.GenerateResult, error) { return v, nil }
	case []lm.GenerateResult:
		m.DoGenerateFunc = func(_ lm.CallOptions) (lm.GenerateResult, error) {
			idx := len(m.DoGenerateCalls)
			if idx < len(v) {
				return v[idx], nil
			}
			return v[len(v)-1], nil
		}
	default:
		m.DoGenerateFunc = func(_ lm.CallOptions) (lm.GenerateResult, error) {
			panic("not implemented")
		}
	}

	// doStream
	switch v := o.DoStream.(type) {
	case func(lm.CallOptions) (lm.StreamResult, error):
		m.DoStreamFunc = v
	case lm.StreamResult:
		m.DoStreamFunc = func(_ lm.CallOptions) (lm.StreamResult, error) { return v, nil }
	case []lm.StreamResult:
		m.DoStreamFunc = func(_ lm.CallOptions) (lm.StreamResult, error) {
			idx := len(m.DoStreamCalls)
			if idx < len(v) {
				return v[idx], nil
			}
			return v[len(v)-1], nil
		}
	default:
		m.DoStreamFunc = func(_ lm.CallOptions) (lm.StreamResult, error) {
			panic("not implemented")
		}
	}

	return m
}

func (m *MockLanguageModelV4) SpecificationVersion() string { return "v4" }
func (m *MockLanguageModelV4) Provider() string             { return m.ProviderID }
func (m *MockLanguageModelV4) ModelID() string              { return m.ModelIDVal }

func (m *MockLanguageModelV4) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return m.supportedUrls()
}

func (m *MockLanguageModelV4) DoGenerate(options lm.CallOptions) (lm.GenerateResult, error) {
	m.DoGenerateCalls = append(m.DoGenerateCalls, options)
	return m.DoGenerateFunc(options)
}

func (m *MockLanguageModelV4) DoStream(options lm.CallOptions) (lm.StreamResult, error) {
	m.DoStreamCalls = append(m.DoStreamCalls, options)
	return m.DoStreamFunc(options)
}
