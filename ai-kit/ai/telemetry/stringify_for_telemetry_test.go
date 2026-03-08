// Ported from: packages/ai/src/telemetry/stringify-for-telemetry.test.ts
package telemetry

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringifyForTelemetry(t *testing.T) {
	t.Run("should stringify a prompt with text parts", func(t *testing.T) {
		prompt := LanguageModelV4Prompt{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: []ContentPart{
				{Type: "text", Text: "Hello!"},
			}},
		}

		result, err := StringifyForTelemetry(prompt)
		assert.NoError(t, err)

		// Parse and re-marshal for comparison
		var parsed interface{}
		err = json.Unmarshal([]byte(result), &parsed)
		assert.NoError(t, err)

		expected := `[{"role":"system","content":"You are a helpful assistant."},{"role":"user","content":[{"type":"text","text":"Hello!"}]}]`
		var expectedParsed interface{}
		_ = json.Unmarshal([]byte(expected), &expectedParsed)

		assert.Equal(t, expectedParsed, parsed)
	})

	t.Run("should convert byte slice data to base64 strings", func(t *testing.T) {
		prompt := LanguageModelV4Prompt{
			{Role: "user", Content: []ContentPart{
				{
					Type:      "file",
					Data:      []byte{0x89, 0x50, 0x4e, 0x47, 0xff, 0xff},
					MediaType: "image/png",
				},
			}},
		}

		result, err := StringifyForTelemetry(prompt)
		assert.NoError(t, err)

		var parsed []map[string]interface{}
		err = json.Unmarshal([]byte(result), &parsed)
		assert.NoError(t, err)

		content := parsed[0]["content"].([]interface{})
		filePart := content[0].(map[string]interface{})
		assert.Equal(t, "iVBOR///", filePart["data"])
	})

	t.Run("should preserve the file name and provider options", func(t *testing.T) {
		prompt := LanguageModelV4Prompt{
			{Role: "user", Content: []ContentPart{
				{
					Type:      "file",
					Filename:  "image.png",
					Data:      []byte{0x89, 0x50, 0x4e, 0x47, 0xff, 0xff},
					MediaType: "image/png",
					ProviderOptions: map[string]map[string]interface{}{
						"anthropic": {"key": "value"},
					},
				},
			}},
		}

		result, err := StringifyForTelemetry(prompt)
		assert.NoError(t, err)

		var parsed []map[string]interface{}
		err = json.Unmarshal([]byte(result), &parsed)
		assert.NoError(t, err)

		content := parsed[0]["content"].([]interface{})
		filePart := content[0].(map[string]interface{})
		assert.Equal(t, "image.png", filePart["filename"])
		assert.Equal(t, "iVBOR///", filePart["data"])

		opts := filePart["providerOptions"].(map[string]interface{})
		anthropic := opts["anthropic"].(map[string]interface{})
		assert.Equal(t, "value", anthropic["key"])
	})

	t.Run("should keep URL strings as is", func(t *testing.T) {
		prompt := LanguageModelV4Prompt{
			{Role: "user", Content: []ContentPart{
				{Type: "text", Text: "Check this image:"},
				{
					Type:      "file",
					Data:      "https://example.com/image.jpg",
					MediaType: "image/jpeg",
				},
			}},
		}

		result, err := StringifyForTelemetry(prompt)
		assert.NoError(t, err)

		var parsed []map[string]interface{}
		err = json.Unmarshal([]byte(result), &parsed)
		assert.NoError(t, err)

		content := parsed[0]["content"].([]interface{})
		filePart := content[1].(map[string]interface{})
		assert.Equal(t, "https://example.com/image.jpg", filePart["data"])
	})

	t.Run("should handle a mixed prompt with various content types", func(t *testing.T) {
		prompt := LanguageModelV4Prompt{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: []ContentPart{
				{
					Type:      "file",
					Data:      []byte{0x89, 0x50, 0x4e, 0x47, 0xff, 0xff},
					MediaType: "image/png",
				},
				{
					Type:      "file",
					Data:      "https://example.com/image.jpg",
					MediaType: "image/jpeg",
				},
			}},
			{Role: "assistant", Content: []ContentPart{
				{Type: "text", Text: "I see the images!"},
			}},
		}

		result, err := StringifyForTelemetry(prompt)
		assert.NoError(t, err)

		var parsed []map[string]interface{}
		err = json.Unmarshal([]byte(result), &parsed)
		assert.NoError(t, err)

		assert.Equal(t, 3, len(parsed))
		assert.Equal(t, "system", parsed[0]["role"])
		assert.Equal(t, "You are a helpful assistant.", parsed[0]["content"])

		userContent := parsed[1]["content"].([]interface{})
		assert.Equal(t, 2, len(userContent))

		assistantContent := parsed[2]["content"].([]interface{})
		assert.Equal(t, 1, len(assistantContent))
		textPart := assistantContent[0].(map[string]interface{})
		assert.Equal(t, "I see the images!", textPart["text"])
	})
}
