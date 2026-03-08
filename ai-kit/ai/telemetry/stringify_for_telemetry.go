// Ported from: packages/ai/src/telemetry/stringify-for-telemetry.ts
package telemetry

import (
	"encoding/base64"
	"encoding/json"
)

// LanguageModelV4Message represents a message in the language model prompt.
// TODO: import from brainlink/experiments/ai-kit/provider once it exists
type LanguageModelV4Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ContentPart
}

// LanguageModelV4Prompt is a slice of LanguageModelV4Message.
type LanguageModelV4Prompt = []LanguageModelV4Message

// ContentPart represents a content part within a message.
type ContentPart struct {
	Type            string                            `json:"type"`
	Text            string                            `json:"text,omitempty"`
	Data            interface{}                       `json:"data,omitempty"`    // []byte, string, or URL string
	MediaType       string                            `json:"mediaType,omitempty"`
	Filename        string                            `json:"filename,omitempty"`
	ProviderOptions map[string]map[string]interface{} `json:"providerOptions,omitempty"`
}

// StringifyForTelemetry serializes prompt content for OpenTelemetry tracing.
// It converts []byte data to base64 strings to avoid JSON.stringify producing
// objects with stringified indices as keys.
func StringifyForTelemetry(prompt LanguageModelV4Prompt) (string, error) {
	serializable := make([]map[string]interface{}, len(prompt))

	for i, message := range prompt {
		m := map[string]interface{}{
			"role": message.Role,
		}

		switch content := message.Content.(type) {
		case string:
			m["content"] = content
		case []ContentPart:
			parts := make([]map[string]interface{}, len(content))
			for j, part := range content {
				p := make(map[string]interface{})
				p["type"] = part.Type

				if part.Text != "" {
					p["text"] = part.Text
				}

				if part.Type == "file" && part.Data != nil {
					switch data := part.Data.(type) {
					case []byte:
						p["data"] = base64.StdEncoding.EncodeToString(data)
					default:
						p["data"] = data
					}
				} else if part.Data != nil {
					p["data"] = part.Data
				}

				if part.MediaType != "" {
					p["mediaType"] = part.MediaType
				}
				if part.Filename != "" {
					p["filename"] = part.Filename
				}
				if part.ProviderOptions != nil {
					p["providerOptions"] = part.ProviderOptions
				}

				parts[j] = p
			}
			m["content"] = parts
		case []interface{}:
			// Handle generic interface slices
			parts := make([]interface{}, len(content))
			for j, raw := range content {
				if partMap, ok := raw.(map[string]interface{}); ok {
					// Check if it's a file part with byte data
					if partMap["type"] == "file" {
						if data, ok := partMap["data"].([]byte); ok {
							partMap["data"] = base64.StdEncoding.EncodeToString(data)
						}
					}
					parts[j] = partMap
				} else {
					parts[j] = raw
				}
			}
			m["content"] = parts
		default:
			m["content"] = content
		}

		serializable[i] = m
	}

	data, err := json.Marshal(serializable)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
