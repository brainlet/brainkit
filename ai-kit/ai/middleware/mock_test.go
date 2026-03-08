// Test helpers and mock types for middleware tests
package middleware

import (
	"regexp"

	em "github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	im "github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	provider "github.com/brainlet/brainkit/ai-kit/provider"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// --- Mock Language Model ---

type mockLanguageModel struct {
	providerVal      string
	modelIDVal       string
	supportedUrlsVal map[string][]*regexp.Regexp
	doGenerateFn     func(lm.CallOptions) (lm.GenerateResult, error)
	doStreamFn       func(lm.CallOptions) (lm.StreamResult, error)

	// Captures for inspection
	DoGenerateCalls []lm.CallOptions
	DoStreamCalls   []lm.CallOptions
}

func (m *mockLanguageModel) SpecificationVersion() string { return "v3" }
func (m *mockLanguageModel) Provider() string {
	if m.providerVal != "" {
		return m.providerVal
	}
	return "mock-provider"
}
func (m *mockLanguageModel) ModelID() string {
	if m.modelIDVal != "" {
		return m.modelIDVal
	}
	return "mock-model-id"
}
func (m *mockLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return m.supportedUrlsVal, nil
}
func (m *mockLanguageModel) DoGenerate(opts lm.CallOptions) (lm.GenerateResult, error) {
	m.DoGenerateCalls = append(m.DoGenerateCalls, opts)
	if m.doGenerateFn != nil {
		return m.doGenerateFn(opts)
	}
	return lm.GenerateResult{}, nil
}
func (m *mockLanguageModel) DoStream(opts lm.CallOptions) (lm.StreamResult, error) {
	m.DoStreamCalls = append(m.DoStreamCalls, opts)
	if m.doStreamFn != nil {
		return m.doStreamFn(opts)
	}
	return lm.StreamResult{Stream: make(<-chan lm.StreamPart)}, nil
}

// --- Mock Embedding Model ---

type mockEmbeddingModel struct {
	providerVal              string
	modelIDVal               string
	maxEmbeddingsPerCallVal  *int
	supportsParallelCallsVal bool
	doEmbedFn                func(em.CallOptions) (em.Result, error)

	DoEmbedCalls []em.CallOptions
}

func (m *mockEmbeddingModel) SpecificationVersion() string { return "v3" }
func (m *mockEmbeddingModel) Provider() string {
	if m.providerVal != "" {
		return m.providerVal
	}
	return "mock-provider"
}
func (m *mockEmbeddingModel) ModelID() string {
	if m.modelIDVal != "" {
		return m.modelIDVal
	}
	return "mock-model-id"
}
func (m *mockEmbeddingModel) MaxEmbeddingsPerCall() (*int, error) {
	return m.maxEmbeddingsPerCallVal, nil
}
func (m *mockEmbeddingModel) SupportsParallelCalls() (bool, error) {
	return m.supportsParallelCallsVal, nil
}
func (m *mockEmbeddingModel) DoEmbed(opts em.CallOptions) (em.Result, error) {
	m.DoEmbedCalls = append(m.DoEmbedCalls, opts)
	if m.doEmbedFn != nil {
		return m.doEmbedFn(opts)
	}
	return em.Result{}, nil
}

// --- Mock Image Model ---

type mockImageModel struct {
	providerVal         string
	modelIDVal          string
	maxImagesPerCallVal *int
	doGenerateFn        func(im.CallOptions) (im.GenerateResult, error)

	DoGenerateCalls []im.CallOptions
}

func (m *mockImageModel) SpecificationVersion() string { return "v3" }
func (m *mockImageModel) Provider() string {
	if m.providerVal != "" {
		return m.providerVal
	}
	return "mock-provider"
}
func (m *mockImageModel) ModelID() string {
	if m.modelIDVal != "" {
		return m.modelIDVal
	}
	return "mock-model-id"
}
func (m *mockImageModel) MaxImagesPerCall() (*int, error) {
	return m.maxImagesPerCallVal, nil
}
func (m *mockImageModel) DoGenerate(opts im.CallOptions) (im.GenerateResult, error) {
	m.DoGenerateCalls = append(m.DoGenerateCalls, opts)
	if m.doGenerateFn != nil {
		return m.doGenerateFn(opts)
	}
	return im.GenerateResult{}, nil
}

// --- Mock Provider ---

type mockProvider struct {
	languageModels map[string]lm.LanguageModel
	imageModels    map[string]im.ImageModel
}

func (p *mockProvider) SpecificationVersion() string { return "v3" }
func (p *mockProvider) LanguageModel(id string) (lm.LanguageModel, error) {
	return p.languageModels[id], nil
}
func (p *mockProvider) EmbeddingModel(id string) (em.EmbeddingModel, error) {
	return nil, nil
}
func (p *mockProvider) ImageModel(id string) (im.ImageModel, error) {
	return p.imageModels[id], nil
}
func (p *mockProvider) TranscriptionModel(id string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, nil
}
func (p *mockProvider) SpeechModel(id string) (speechmodel.SpeechModel, error) {
	return nil, nil
}
func (p *mockProvider) RerankingModel(id string) (rerankingmodel.RerankingModel, error) {
	return nil, nil
}

// Ensure mockProvider satisfies provider.Provider at compile time
var _ provider.Provider = (*mockProvider)(nil)

// Helper to collect all stream parts from a channel
func collectStreamParts(ch <-chan lm.StreamPart) []lm.StreamPart {
	var parts []lm.StreamPart
	for p := range ch {
		parts = append(parts, p)
	}
	return parts
}

// Helper to create a stream channel from a slice of parts
func streamFromParts(parts []lm.StreamPart) <-chan lm.StreamPart {
	ch := make(chan lm.StreamPart, len(parts))
	for _, p := range parts {
		ch <- p
	}
	close(ch)
	return ch
}

// ptrStr returns a pointer to a string
func ptrStr(s string) *string { return &s }

// ptrFloat64 returns a pointer to a float64
func ptrFloat64(f float64) *float64 { return &f }

// ptrInt returns a pointer to an int
func ptrInt(i int) *int { return &i }

// ptrBool returns a pointer to a bool
func ptrBool(b bool) *bool { return &b }
