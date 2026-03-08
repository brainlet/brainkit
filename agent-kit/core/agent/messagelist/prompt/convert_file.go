// Ported from: packages/core/src/agent/message-list/prompt/convert-file.ts
package prompt

// DownloadedAsset represents a downloaded file with its media type and data.
type DownloadedAsset struct {
	MediaType string
	Data      []byte
}

// LanguageModelV2FilePart represents a file part in V2 LLM prompt format.
// TODO: In TS this comes from @ai-sdk/provider-v5 LanguageModelV2FilePart.
type LanguageModelV2FilePart struct {
	Type            string           `json:"type"` // "file"
	MediaType       string           `json:"mediaType"`
	Filename        string           `json:"filename,omitempty"`
	Data            any              `json:"data"` // []byte | string | *url.URL
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`
}

// LanguageModelV2TextPart represents a text part in V2 LLM prompt format.
// TODO: In TS this comes from @ai-sdk/provider-v5 LanguageModelV2TextPart.
type LanguageModelV2TextPart struct {
	Type            string                    `json:"type"` // "text"
	Text            string                    `json:"text"`
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`
}

// ConvertImageFilePart converts an image or file part to a V2 LLM prompt part.
// TODO: This depends on convertToDataContent and detectMediaType from ../../stream/aisdk/v5/compat
// which are not yet ported. Creating a simplified stub version.
func ConvertImageFilePart(part map[string]any, downloadedAssets map[string]DownloadedAsset) any {
	partType, _ := part["type"].(string)

	var originalData any
	switch partType {
	case "image":
		originalData = part["image"]
	case "file":
		originalData = part["data"]
	default:
		return part // return as-is for unknown types
	}

	mediaType, _ := part["mediaType"].(string)

	// Check if the data is a URL and was downloaded
	if dataStr, ok := originalData.(string); ok && downloadedAssets != nil {
		if downloaded, found := downloadedAssets[dataStr]; found {
			if mediaType == "" {
				mediaType = downloaded.MediaType
			}
			return LanguageModelV2FilePart{
				Type:      "file",
				MediaType: mediaType,
				Filename:  stringFromMap(part, "filename"),
				Data:      downloaded.Data,
				ProviderOptions: providerOptionsFromMap(part),
			}
		}
	}

	// Default: return a file part with the original data
	if mediaType == "" {
		if partType == "image" {
			mediaType = "image/*"
		} else {
			mediaType = "application/octet-stream"
		}
	}

	return LanguageModelV2FilePart{
		Type:      "file",
		MediaType: mediaType,
		Filename:  stringFromMap(part, "filename"),
		Data:      originalData,
		ProviderOptions: providerOptionsFromMap(part),
	}
}

func stringFromMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func providerOptionsFromMap(m map[string]any) map[string]map[string]any {
	v, ok := m["providerOptions"].(map[string]map[string]any)
	if ok {
		return v
	}
	return nil
}
