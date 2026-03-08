// Ported from: packages/ai/src/middleware/default-settings-middleware.test.ts
package middleware

import (
	"testing"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

func baseParams() lm.CallOptions {
	return lm.CallOptions{
		Prompt: lm.Prompt{
			lm.UserMessage{Content: []lm.UserMessagePart{lm.TextPart{Text: "Hello, world!"}}},
		},
	}
}

var mockLM = &mockLanguageModel{providerVal: "mock-provider", modelIDVal: "mock-model"}

func TestDefaultSettingsMiddleware_ApplyDefaults(t *testing.T) {
	temp := 0.7
	m := DefaultSettingsMiddleware(DefaultSettings{Temperature: &temp})

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: baseParams(),
		Model:  mockLM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Temperature == nil || *result.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", result.Temperature)
	}
}

func TestDefaultSettingsMiddleware_UserPrecedence(t *testing.T) {
	temp := 0.7
	m := DefaultSettingsMiddleware(DefaultSettings{Temperature: &temp})

	userTemp := 0.5
	params := baseParams()
	params.Temperature = &userTemp

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  mockLM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Temperature == nil || *result.Temperature != 0.5 {
		t.Errorf("expected temperature 0.5, got %v", result.Temperature)
	}
}

func TestDefaultSettingsMiddleware_MergeProviderOptions(t *testing.T) {
	temp := 0.7
	m := DefaultSettingsMiddleware(DefaultSettings{
		Temperature: &temp,
		ProviderOptions: map[string]map[string]any{
			"anthropic": {
				"cacheControl": map[string]any{"type": "ephemeral"},
			},
		},
	})

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: baseParams(),
		Model:  mockLM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Temperature == nil || *result.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", result.Temperature)
	}
	if result.ProviderOptions == nil {
		t.Fatal("expected providerOptions to be non-nil")
	}
	anthropic, ok := result.ProviderOptions["anthropic"]
	if !ok {
		t.Fatal("expected anthropic provider options")
	}
	cc, ok := anthropic["cacheControl"]
	if !ok {
		t.Fatal("expected cacheControl key")
	}
	ccMap, ok := cc.(map[string]interface{})
	if !ok {
		t.Fatalf("expected cacheControl to be a map, got %T", cc)
	}
	if ccMap["type"] != "ephemeral" {
		t.Errorf("expected cacheControl.type=ephemeral, got %v", ccMap["type"])
	}
}

func TestDefaultSettingsMiddleware_KeepZeroTemp(t *testing.T) {
	m := DefaultSettingsMiddleware(DefaultSettings{})

	temp := 0.0
	params := baseParams()
	params.Temperature = &temp

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  mockLM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Temperature == nil || *result.Temperature != 0.0 {
		t.Errorf("expected temperature 0, got %v", result.Temperature)
	}
}

func TestDefaultSettingsMiddleware_UseDefaultTempWhenUndefined(t *testing.T) {
	temp := 0.7
	m := DefaultSettingsMiddleware(DefaultSettings{Temperature: &temp})

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: baseParams(),
		Model:  mockLM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Temperature == nil || *result.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %v", result.Temperature)
	}
}

func TestDefaultSettingsMiddleware_MaxOutputTokens(t *testing.T) {
	maxTokens := 100
	m := DefaultSettingsMiddleware(DefaultSettings{MaxOutputTokens: &maxTokens})

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: baseParams(),
		Model:  mockLM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.MaxOutputTokens == nil || *result.MaxOutputTokens != 100 {
		t.Errorf("expected maxOutputTokens 100, got %v", result.MaxOutputTokens)
	}
}

func TestDefaultSettingsMiddleware_PrioritizeParamMaxOutputTokens(t *testing.T) {
	maxTokens := 100
	m := DefaultSettingsMiddleware(DefaultSettings{MaxOutputTokens: &maxTokens})

	userMax := 50
	params := baseParams()
	params.MaxOutputTokens = &userMax

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  mockLM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.MaxOutputTokens == nil || *result.MaxOutputTokens != 50 {
		t.Errorf("expected maxOutputTokens 50, got %v", result.MaxOutputTokens)
	}
}

func TestDefaultSettingsMiddleware_StopSequences(t *testing.T) {
	m := DefaultSettingsMiddleware(DefaultSettings{StopSequences: []string{"stop"}})

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: baseParams(),
		Model:  mockLM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.StopSequences) != 1 || result.StopSequences[0] != "stop" {
		t.Errorf("expected stopSequences=[stop], got %v", result.StopSequences)
	}
}

func TestDefaultSettingsMiddleware_MergeHeaders(t *testing.T) {
	s1 := "test"
	s2 := "test2"
	m := DefaultSettingsMiddleware(DefaultSettings{
		Headers: map[string]*string{
			"X-Custom-Header":  &s1,
			"X-Another-Header": &s2,
		},
	})

	override := "test2"
	params := baseParams()
	params.Headers = map[string]*string{
		"X-Custom-Header": &override,
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  mockLM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Headers == nil {
		t.Fatal("expected headers to be non-nil")
	}
	if result.Headers["X-Custom-Header"] == nil || *result.Headers["X-Custom-Header"] != "test2" {
		t.Errorf("expected X-Custom-Header=test2")
	}
	if result.Headers["X-Another-Header"] == nil || *result.Headers["X-Another-Header"] != "test2" {
		t.Errorf("expected X-Another-Header=test2")
	}
}

func TestDefaultSettingsMiddleware_MergeComplexProviderOptions(t *testing.T) {
	m := DefaultSettingsMiddleware(DefaultSettings{
		ProviderOptions: map[string]map[string]any{
			"anthropic": {
				"cacheControl": map[string]any{"type": "ephemeral"},
				"feature":      map[string]any{"enabled": true},
			},
			"openai": {
				"logit_bias": map[string]any{"50256": -100},
			},
		},
	})

	params := baseParams()
	params.ProviderOptions = map[string]map[string]any{
		"anthropic": {
			"feature":      map[string]any{"enabled": false},
			"otherSetting": "value",
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  mockLM,
	})
	if err != nil {
		t.Fatal(err)
	}

	anthropic := result.ProviderOptions["anthropic"]
	cc := anthropic["cacheControl"].(map[string]interface{})
	if cc["type"] != "ephemeral" {
		t.Errorf("expected cacheControl.type=ephemeral")
	}
	feat := anthropic["feature"].(map[string]interface{})
	if feat["enabled"] != false {
		t.Errorf("expected feature.enabled=false (user override)")
	}
	if anthropic["otherSetting"] != "value" {
		t.Errorf("expected otherSetting=value")
	}
	openai := result.ProviderOptions["openai"]
	lb := openai["logit_bias"].(map[string]interface{})
	if lb["50256"] != -100 {
		t.Errorf("expected logit_bias.50256=-100")
	}
}
