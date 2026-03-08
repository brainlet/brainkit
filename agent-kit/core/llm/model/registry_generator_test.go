// Ported from: packages/core/src/llm/model/registry-generator.test.ts
package model

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerateTypesContent(t *testing.T) {
	t.Run("should not quote valid JS identifiers", func(t *testing.T) {
		models := map[string][]string{
			"openai":      {"gpt-4"},
			"_private":    {"model-1"},
			"$provider":   {"model-2"},
			"provider123": {"model-3"},
		}

		content := GenerateTypesContent(models)

		if !strings.Contains(content, "readonly openai:") {
			t.Error("expected content to contain 'readonly openai:'")
		}
		if !strings.Contains(content, "readonly _private:") {
			t.Error("expected content to contain 'readonly _private:'")
		}
		if !strings.Contains(content, "readonly $provider:") {
			t.Error("expected content to contain 'readonly $provider:'")
		}
		if !strings.Contains(content, "readonly provider123:") {
			t.Error("expected content to contain 'readonly provider123:'")
		}
	})

	t.Run("should quote provider names with special characters", func(t *testing.T) {
		models := map[string][]string{
			"fireworks-ai": {"llama-v3-70b"},
		}

		content := GenerateTypesContent(models)

		if !strings.Contains(content, "readonly 'fireworks-ai':") {
			t.Error("expected content to contain \"readonly 'fireworks-ai':\"")
		}
	})

	t.Run("should quote provider names starting with digits", func(t *testing.T) {
		models := map[string][]string{
			"302ai": {"model-1"},
		}

		content := GenerateTypesContent(models)

		if !strings.Contains(content, "readonly '302ai':") {
			t.Error("expected content to contain \"readonly '302ai':\"")
		}
		// Verify it does NOT have an unquoted digit-starting identifier
		digitStartRegex := regexp.MustCompile(`readonly\s+\d`)
		if digitStartRegex.MatchString(content) {
			t.Error("expected content to NOT have unquoted digit-starting provider name")
		}
	})
}
