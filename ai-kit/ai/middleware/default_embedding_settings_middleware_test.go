// Ported from: packages/ai/src/middleware/default-embedding-settings-middleware.test.ts
package middleware

import (
	"testing"

	em "github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

var embeddingBaseParams = em.CallOptions{
	Values: []string{"hello world"},
}

var mockEM = &mockEmbeddingModel{}

func TestDefaultEmbeddingSettings_MergeHeaders(t *testing.T) {
	m := DefaultEmbeddingSettingsMiddleware(DefaultEmbeddingSettings{
		Headers: map[string]string{
			"X-Custom-Header":  "test",
			"X-Another-Header": "test2",
		},
	})

	params := embeddingBaseParams
	params.Headers = map[string]string{"X-Custom-Header": "test2"}

	result, err := m.TransformParams(mw.EmbeddingTransformParamsOptions{
		Params: params,
		Model:  mockEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Headers["X-Custom-Header"] != "test2" {
		t.Errorf("expected X-Custom-Header=test2, got %v", result.Headers["X-Custom-Header"])
	}
	if result.Headers["X-Another-Header"] != "test2" {
		t.Errorf("expected X-Another-Header=test2, got %v", result.Headers["X-Another-Header"])
	}
}

func TestDefaultEmbeddingSettings_EmptyDefaultHeaders(t *testing.T) {
	m := DefaultEmbeddingSettingsMiddleware(DefaultEmbeddingSettings{
		Headers: map[string]string{},
	})

	params := embeddingBaseParams
	params.Headers = map[string]string{"X-Param-Header": "param"}

	result, err := m.TransformParams(mw.EmbeddingTransformParamsOptions{
		Params: params,
		Model:  mockEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Headers["X-Param-Header"] != "param" {
		t.Errorf("expected X-Param-Header=param, got %v", result.Headers["X-Param-Header"])
	}
}

func TestDefaultEmbeddingSettings_EmptyParamHeaders(t *testing.T) {
	m := DefaultEmbeddingSettingsMiddleware(DefaultEmbeddingSettings{
		Headers: map[string]string{"X-Default-Header": "default"},
	})

	params := embeddingBaseParams
	params.Headers = map[string]string{}

	result, err := m.TransformParams(mw.EmbeddingTransformParamsOptions{
		Params: params,
		Model:  mockEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Headers["X-Default-Header"] != "default" {
		t.Errorf("expected X-Default-Header=default, got %v", result.Headers["X-Default-Header"])
	}
}

func TestDefaultEmbeddingSettings_BothHeadersUndefined(t *testing.T) {
	m := DefaultEmbeddingSettingsMiddleware(DefaultEmbeddingSettings{})

	result, err := m.TransformParams(mw.EmbeddingTransformParamsOptions{
		Params: embeddingBaseParams,
		Model:  mockEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Headers != nil {
		t.Errorf("expected nil headers, got %v", result.Headers)
	}
}

func TestDefaultEmbeddingSettings_EmptyDefaultProviderOptions(t *testing.T) {
	m := DefaultEmbeddingSettingsMiddleware(DefaultEmbeddingSettings{
		ProviderOptions: map[string]map[string]any{},
	})

	params := embeddingBaseParams
	params.ProviderOptions = map[string]map[string]any{
		"google": {
			"outputDimensionality": 512,
			"taskType":             "SEMANTIC_SIMILARITY",
		},
	}

	result, err := m.TransformParams(mw.EmbeddingTransformParamsOptions{
		Params: params,
		Model:  mockEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	google, ok := result.ProviderOptions["google"]
	if !ok {
		t.Fatal("expected google provider options")
	}
	if google["outputDimensionality"] != 512 {
		t.Errorf("expected outputDimensionality=512, got %v", google["outputDimensionality"])
	}
	if google["taskType"] != "SEMANTIC_SIMILARITY" {
		t.Errorf("expected taskType=SEMANTIC_SIMILARITY, got %v", google["taskType"])
	}
}

func TestDefaultEmbeddingSettings_EmptyParamProviderOptions(t *testing.T) {
	m := DefaultEmbeddingSettingsMiddleware(DefaultEmbeddingSettings{
		ProviderOptions: map[string]map[string]any{
			"google": {
				"outputDimensionality": 512,
				"taskType":             "SEMANTIC_SIMILARITY",
			},
		},
	})

	params := embeddingBaseParams
	params.ProviderOptions = map[string]map[string]any{}

	result, err := m.TransformParams(mw.EmbeddingTransformParamsOptions{
		Params: params,
		Model:  mockEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	google, ok := result.ProviderOptions["google"]
	if !ok {
		t.Fatal("expected google provider options")
	}
	if google["outputDimensionality"] != 512 {
		t.Errorf("expected outputDimensionality=512, got %v", google["outputDimensionality"])
	}
}

func TestDefaultEmbeddingSettings_BothProviderOptionsUndefined(t *testing.T) {
	m := DefaultEmbeddingSettingsMiddleware(DefaultEmbeddingSettings{})

	result, err := m.TransformParams(mw.EmbeddingTransformParamsOptions{
		Params: embeddingBaseParams,
		Model:  mockEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ProviderOptions != nil {
		t.Errorf("expected nil providerOptions, got %v", result.ProviderOptions)
	}
}
