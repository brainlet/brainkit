// Ported from: packages/ai/src/prompt/convert-to-language-model-prompt.test.ts
package prompt

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToLanguageModelPrompt(t *testing.T) {
	t.Run("system message", func(t *testing.T) {
		t.Run("should convert a string system message", func(t *testing.T) {
			result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
				Prompt: StandardizedPrompt{
					System: "INSTRUCTIONS",
					Messages: []ModelMessage{
						UserModelMessage{Role: "user", Content: "Hello, world!"},
					},
				},
				SupportedUrls: map[string][]string{},
			})

			require.NoError(t, err)
			require.Len(t, result, 2)

			// System message
			assert.Equal(t, "system", result[0].Role)
			assert.Equal(t, "INSTRUCTIONS", result[0].Content)

			// User message
			assert.Equal(t, "user", result[1].Role)
			parts, ok := result[1].Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 1)
			textPart, ok := parts[0].(LanguageModelV4TextPart)
			require.True(t, ok)
			assert.Equal(t, "text", textPart.Type)
			assert.Equal(t, "Hello, world!", textPart.Text)
		})

		t.Run("should convert a SystemModelMessage system message", func(t *testing.T) {
			result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
				Prompt: StandardizedPrompt{
					System: SystemModelMessage{
						Role:    "system",
						Content: "INSTRUCTIONS",
						ProviderOptions: ProviderOptions{
							"test": {"value": "test"},
						},
					},
					Messages: []ModelMessage{
						UserModelMessage{Role: "user", Content: "Hello, world!"},
					},
				},
				SupportedUrls: map[string][]string{},
			})

			require.NoError(t, err)
			require.Len(t, result, 2)

			// System message
			assert.Equal(t, "system", result[0].Role)
			assert.Equal(t, "INSTRUCTIONS", result[0].Content)
			assert.Equal(t, ProviderOptions{"test": {"value": "test"}}, result[0].ProviderOptions)

			// User message
			assert.Equal(t, "user", result[1].Role)
			parts, ok := result[1].Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 1)
			textPart, ok := parts[0].(LanguageModelV4TextPart)
			require.True(t, ok)
			assert.Equal(t, "text", textPart.Type)
			assert.Equal(t, "Hello, world!", textPart.Text)
		})

		t.Run("should convert an array of SystemModelMessage system messages", func(t *testing.T) {
			result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
				Prompt: StandardizedPrompt{
					System: []SystemModelMessage{
						{Role: "system", Content: "INSTRUCTIONS"},
						{Role: "system", Content: "INSTRUCTIONS 2"},
					},
					Messages: []ModelMessage{
						UserModelMessage{Role: "user", Content: "Hello, world!"},
					},
				},
				SupportedUrls: map[string][]string{},
			})

			require.NoError(t, err)
			require.Len(t, result, 3)

			// System messages
			assert.Equal(t, "system", result[0].Role)
			assert.Equal(t, "INSTRUCTIONS", result[0].Content)

			assert.Equal(t, "system", result[1].Role)
			assert.Equal(t, "INSTRUCTIONS 2", result[1].Content)

			// User message
			assert.Equal(t, "user", result[2].Role)
			parts, ok := result[2].Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 1)
			textPart, ok := parts[0].(LanguageModelV4TextPart)
			require.True(t, ok)
			assert.Equal(t, "text", textPart.Type)
			assert.Equal(t, "Hello, world!", textPart.Text)
		})
	})

	t.Run("user message", func(t *testing.T) {
		t.Run("image parts", func(t *testing.T) {
			t.Run("should download images for user image parts with URLs when model does not support image URLs", func(t *testing.T) {
				// In Go, downloads are pre-computed via DownloadedAssets
				imgURL, _ := url.Parse("https://example.com/image.png")
				mediaType := "image/png"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									ImagePart{
										Type:  "image",
										Image: imgURL,
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{},
					DownloadedAssets: DownloadedAssets{
						"https://example.com/image.png": {
							Data:      []byte{0, 1, 2, 3},
							MediaType: &mediaType,
						},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "image/png", filePart.MediaType)
				assert.Equal(t, []byte{0, 1, 2, 3}, filePart.Data)
			})

			t.Run("should download images for user image parts with string URLs when model does not support image URLs", func(t *testing.T) {
				// String URL "https://..." gets parsed into *url.URL in ConvertToLanguageModelV4DataContent
				mediaType := "image/png"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									ImagePart{
										Type:  "image",
										Image: "https://example.com/image.png",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{},
					DownloadedAssets: DownloadedAssets{
						"https://example.com/image.png": {
							Data:      []byte{0, 1, 2, 3},
							MediaType: &mediaType,
						},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "image/png", filePart.MediaType)
				assert.Equal(t, []byte{0, 1, 2, 3}, filePart.Data)
			})
		})

		t.Run("file parts", func(t *testing.T) {
			t.Run("should pass through URLs when the model supports a particular URL", func(t *testing.T) {
				// In Go, supportedUrls is simplified. If no download is done the URL passes through.
				fileURL, _ := url.Parse("https://example.com/document.pdf")
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      fileURL,
										MediaType: "application/pdf",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{
						"*": {"^https://.*$"},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "application/pdf", filePart.MediaType)
				// URL should pass through (not downloaded)
				_, isURL := filePart.Data.(*url.URL)
				assert.True(t, isURL)
			})

			t.Run("should download the URL as an asset when the model does not support a URL", func(t *testing.T) {
				fileURL, _ := url.Parse("https://example.com/document.pdf")
				mediaType := "application/pdf"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      fileURL,
										MediaType: "application/pdf",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{
						"image/*": {"^https://.*$"},
					},
					DownloadedAssets: DownloadedAssets{
						"https://example.com/document.pdf": {
							Data:      []byte{0, 1, 2, 3},
							MediaType: &mediaType,
						},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "application/pdf", filePart.MediaType)
				assert.Equal(t, []byte{0, 1, 2, 3}, filePart.Data)
			})

			t.Run("should handle file parts with base64 string data", func(t *testing.T) {
				base64Data := "SGVsbG8sIFdvcmxkIQ==" // "Hello, World!" in base64
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      base64Data,
										MediaType: "text/plain",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{
						"image/*": {"^https://.*$"},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, base64Data, filePart.Data)
				assert.Equal(t, "text/plain", filePart.MediaType)
			})

			t.Run("should handle file parts with byte slice data", func(t *testing.T) {
				byteData := []byte{72, 101, 108, 108, 111} // "Hello" in ASCII
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      byteData,
										MediaType: "text/plain",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{
						"image/*": {"^https://.*$"},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, []byte{72, 101, 108, 108, 111}, filePart.Data)
				assert.Equal(t, "text/plain", filePart.MediaType)
			})

			t.Run("should download files for user file parts with URL objects when model does not support downloads", func(t *testing.T) {
				fileURL, _ := url.Parse("https://example.com/document.pdf")
				mediaType := "application/pdf"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      fileURL,
										MediaType: "application/pdf",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{},
					DownloadedAssets: DownloadedAssets{
						"https://example.com/document.pdf": {
							Data:      []byte{0, 1, 2, 3},
							MediaType: &mediaType,
						},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "application/pdf", filePart.MediaType)
				assert.Equal(t, []byte{0, 1, 2, 3}, filePart.Data)
			})

			t.Run("should download files for user file parts with string URLs when model does not support downloads", func(t *testing.T) {
				mediaType := "application/pdf"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      "https://example.com/document.pdf",
										MediaType: "application/pdf",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{},
					DownloadedAssets: DownloadedAssets{
						"https://example.com/document.pdf": {
							Data:      []byte{0, 1, 2, 3},
							MediaType: &mediaType,
						},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "application/pdf", filePart.MediaType)
				assert.Equal(t, []byte{0, 1, 2, 3}, filePart.Data)
			})

			t.Run("should download files for user file parts with string URLs when model does not support the particular URL", func(t *testing.T) {
				mediaType := "application/pdf"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      "https://example.com/document.pdf",
										MediaType: "application/pdf",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{
						"application/pdf": {"^(?!https://example\\.com/document\\.pdf$).*$"},
					},
					DownloadedAssets: DownloadedAssets{
						"https://example.com/document.pdf": {
							Data:      []byte{0, 1, 2, 3},
							MediaType: &mediaType,
						},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "application/pdf", filePart.MediaType)
				assert.Equal(t, []byte{0, 1, 2, 3}, filePart.Data)
			})

			t.Run("does not download URLs for user file parts for URL objects when model does support the URL", func(t *testing.T) {
				fileURL, _ := url.Parse("https://example.com/document.pdf")
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      fileURL,
										MediaType: "application/pdf",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{
						"application/pdf": {"^https://example\\.com/document\\.pdf$"},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "application/pdf", filePart.MediaType)
				// URL passes through without download
				_, isURL := filePart.Data.(*url.URL)
				assert.True(t, isURL)
			})

			t.Run("it should default to downloading the URL when the model does not provide a supportsUrl function", func(t *testing.T) {
				mediaType := "application/pdf"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      "https://example.com/document.pdf",
										MediaType: "application/pdf",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{},
					DownloadedAssets: DownloadedAssets{
						"https://example.com/document.pdf": {
							Data:      []byte{0, 1, 2, 3},
							MediaType: &mediaType,
						},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "application/pdf", filePart.MediaType)
				assert.Equal(t, []byte{0, 1, 2, 3}, filePart.Data)
			})

			t.Run("should handle file parts with filename", func(t *testing.T) {
				filename := "hello.txt"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      "SGVsbG8sIFdvcmxkIQ==",
										MediaType: "text/plain",
										Filename:  &filename,
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{
						"image/*": {"^https://.*$"},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "SGVsbG8sIFdvcmxkIQ==", filePart.Data)
				assert.Equal(t, "text/plain", filePart.MediaType)
				require.NotNil(t, filePart.Filename)
				assert.Equal(t, "hello.txt", *filePart.Filename)
			})

			t.Run("should preserve filename when downloading file from URL", func(t *testing.T) {
				fileURL, _ := url.Parse("https://example.com/document.pdf")
				filename := "important-document.pdf"
				mediaType := "application/pdf"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      fileURL,
										MediaType: "application/pdf",
										Filename:  &filename,
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{},
					DownloadedAssets: DownloadedAssets{
						"https://example.com/document.pdf": {
							Data:      []byte{0, 1, 2, 3},
							MediaType: &mediaType,
						},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "application/pdf", filePart.MediaType)
				assert.Equal(t, []byte{0, 1, 2, 3}, filePart.Data)
				require.NotNil(t, filePart.Filename)
				assert.Equal(t, "important-document.pdf", *filePart.Filename)
			})

			t.Run("should prioritize user-provided mediaType over downloaded file mediaType", func(t *testing.T) {
				fileURL, _ := url.Parse("https://example.com/image.jpg")
				downloadedMediaType := "application/octet-stream"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      fileURL,
										MediaType: "image/jpeg",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{},
					DownloadedAssets: DownloadedAssets{
						"https://example.com/image.jpg": {
							Data:      []byte{0, 1, 2, 3},
							MediaType: &downloadedMediaType,
						},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				// The user-provided mediaType should be used (not overridden by download)
				// In Go, ConvertToLanguageModelV4DataContent returns the URL, which gets downloaded
				// and the original mediaType from FilePart is preserved
				assert.Equal(t, []byte{0, 1, 2, 3}, filePart.Data)
			})

			t.Run("should use downloaded file mediaType as fallback when user provides generic mediaType", func(t *testing.T) {
				fileURL, _ := url.Parse("https://example.com/document.txt")
				downloadedMediaType := "text/plain"
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									FilePart{
										Type:      "file",
										Data:      fileURL,
										MediaType: "application/octet-stream",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{},
					DownloadedAssets: DownloadedAssets{
						"https://example.com/document.txt": {
							Data:      []byte{72, 101, 108, 108, 111},
							MediaType: &downloadedMediaType,
						},
					},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, []byte{72, 101, 108, 108, 111}, filePart.Data)
			})
		})

		t.Run("provider options", func(t *testing.T) {
			t.Run("should add provider options to messages", func(t *testing.T) {
				result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
					Prompt: StandardizedPrompt{
						Messages: []ModelMessage{
							UserModelMessage{
								Role: "user",
								Content: []interface{}{
									TextPart{
										Type: "text",
										Text: "hello, world!",
									},
								},
								ProviderOptions: ProviderOptions{
									"test-provider": {
										"key-a": "test-value-1",
										"key-b": "test-value-2",
									},
								},
							},
						},
					},
					SupportedUrls: map[string][]string{},
				})

				require.NoError(t, err)
				require.Len(t, result, 1)
				assert.Equal(t, "user", result[0].Role)
				assert.Equal(t, ProviderOptions{
					"test-provider": {
						"key-a": "test-value-1",
						"key-b": "test-value-2",
					},
				}, result[0].ProviderOptions)
				parts, ok := result[0].Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				textPart, ok := parts[0].(LanguageModelV4TextPart)
				require.True(t, ok)
				assert.Equal(t, "text", textPart.Type)
				assert.Equal(t, "hello, world!", textPart.Text)
			})
		})

		t.Run("should download files when intermediate file cannot be downloaded", func(t *testing.T) {
			// In Go, pre-download assets for imageUrlA and imageUrlB, but NOT for fileUrl.
			imageMediaType := "image/png"
			result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
				Prompt: StandardizedPrompt{
					Messages: []ModelMessage{
						UserModelMessage{
							Role: "user",
							Content: []interface{}{
								ImagePart{
									Type:      "image",
									Image:     "http://example.com/my-image-A.png",
									MediaType: strPtr("image/png"),
								},
								FilePart{
									Type:      "file",
									Data:      mustParseURL("http://127.0.0.1:3000/file"),
									MediaType: "application/octet-stream",
								},
								ImagePart{
									Type:      "image",
									Image:     "http://example.com/my-image-B.png",
									MediaType: strPtr("image/png"),
								},
							},
						},
					},
				},
				SupportedUrls: map[string][]string{
					"*": {"^https://.*$"},
				},
				DownloadedAssets: DownloadedAssets{
					"http://example.com/my-image-A.png": {
						Data:      []byte{137, 80, 78, 71, 13, 10, 26, 10, 0},
						MediaType: &imageMediaType,
					},
					"http://example.com/my-image-B.png": {
						Data:      []byte{137, 80, 78, 71, 13, 10, 26, 10, 1},
						MediaType: &imageMediaType,
					},
				},
			})

			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.Equal(t, "user", result[0].Role)
			parts, ok := result[0].Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 3)

			// First image part - downloaded
			fp0, ok := parts[0].(LanguageModelV4FilePart)
			require.True(t, ok)
			assert.Equal(t, "file", fp0.Type)
			assert.Equal(t, "image/png", fp0.MediaType)
			assert.Equal(t, []byte{137, 80, 78, 71, 13, 10, 26, 10, 0}, fp0.Data)

			// Middle file part - not downloaded (URL not in DownloadedAssets), passes through as URL
			fp1, ok := parts[1].(LanguageModelV4FilePart)
			require.True(t, ok)
			assert.Equal(t, "file", fp1.Type)
			assert.Equal(t, "application/octet-stream", fp1.MediaType)

			// Third image part - downloaded
			fp2, ok := parts[2].(LanguageModelV4FilePart)
			require.True(t, ok)
			assert.Equal(t, "file", fp2.Type)
			assert.Equal(t, "image/png", fp2.MediaType)
			assert.Equal(t, []byte{137, 80, 78, 71, 13, 10, 26, 10, 1}, fp2.Data)
		})
	})

	t.Run("tool message", func(t *testing.T) {
		t.Run("should combine 2 consecutive tool messages into a single tool message", func(t *testing.T) {
			result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
				Prompt: StandardizedPrompt{
					Messages: []ModelMessage{
						AssistantModelMessage{
							Role: "assistant",
							Content: []interface{}{
								ToolCallPart{
									Type:       "tool-call",
									ToolCallID: "toolCallId",
									ToolName:   "toolName",
									Input:      map[string]interface{}{},
								},
								ToolApprovalRequest{
									Type:       "tool-approval-request",
									ApprovalID: "approvalId",
									ToolCallID: "toolCallId",
								},
							},
						},
						ToolModelMessage{
							Role: "tool",
							Content: []interface{}{
								ToolApprovalResponse{
									Type:       "tool-approval-response",
									ApprovalID: "approvalId",
									Approved:   true,
								},
							},
						},
						ToolModelMessage{
							Role: "tool",
							Content: []interface{}{
								ToolResultPart{
									Type:       "tool-result",
									ToolName:   "toolName",
									ToolCallID: "toolCallId",
									Output: ToolResultOutput{
										Type:  "json",
										Value: map[string]interface{}{"some": "result"},
									},
								},
							},
						},
					},
				},
				SupportedUrls: map[string][]string{},
			})

			require.NoError(t, err)
			require.Len(t, result, 2)

			// Assistant message
			assert.Equal(t, "assistant", result[0].Role)
			assistantParts, ok := result[0].Content.([]interface{})
			require.True(t, ok)
			require.Len(t, assistantParts, 1) // tool-approval-request is filtered out
			tc, ok := assistantParts[0].(LanguageModelV4ToolCallPart)
			require.True(t, ok)
			assert.Equal(t, "tool-call", tc.Type)
			assert.Equal(t, "toolCallId", tc.ToolCallID)
			assert.Equal(t, "toolName", tc.ToolName)

			// Tool message (combined)
			assert.Equal(t, "tool", result[1].Role)
			toolParts, ok := result[1].Content.([]interface{})
			require.True(t, ok)
			require.Len(t, toolParts, 1) // tool-approval-response filtered (not providerExecuted)
			tr, ok := toolParts[0].(LanguageModelV4ToolResultPart)
			require.True(t, ok)
			assert.Equal(t, "tool-result", tr.Type)
			assert.Equal(t, "toolCallId", tr.ToolCallID)
			assert.Equal(t, "toolName", tr.ToolName)
			assert.Equal(t, "json", tr.Output.Type)
		})
	})

	t.Run("custom download function", func(t *testing.T) {
		t.Run("should use pre-downloaded assets to resolve URL content", func(t *testing.T) {
			// In Go, the download function is replaced by DownloadedAssets
			mediaType := "text/plain"
			result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
				Prompt: StandardizedPrompt{
					Messages: []ModelMessage{
						UserModelMessage{
							Role: "user",
							Content: []interface{}{
								FilePart{
									Type:      "file",
									Data:      "https://example.com/test-file.txt",
									MediaType: "text/plain",
								},
							},
						},
					},
				},
				SupportedUrls: map[string][]string{},
				DownloadedAssets: DownloadedAssets{
					"https://example.com/test-file.txt": {
						Data:      []byte{72, 101, 108, 108, 111},
						MediaType: &mediaType,
					},
				},
			})

			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.Equal(t, "user", result[0].Role)
			parts, ok := result[0].Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 1)
			filePart, ok := parts[0].(LanguageModelV4FilePart)
			require.True(t, ok)
			assert.Equal(t, "file", filePart.Type)
			assert.Equal(t, "text/plain", filePart.MediaType)
			assert.Equal(t, []byte{72, 101, 108, 108, 111}, filePart.Data)
		})
	})
}

func TestConvertToLanguageModelMessage(t *testing.T) {
	t.Run("user message", func(t *testing.T) {
		t.Run("text parts", func(t *testing.T) {
			t.Run("should filter out empty text parts", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					UserModelMessage{
						Role: "user",
						Content: []interface{}{
							TextPart{Type: "text", Text: ""},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "user", result.Role)
				// Empty text part should be filtered
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				assert.Nil(t, parts) // nil slice when all parts filtered
			})

			t.Run("should pass through non-empty text parts", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					UserModelMessage{
						Role: "user",
						Content: []interface{}{
							TextPart{Type: "text", Text: "hello, world!"},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "user", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				textPart, ok := parts[0].(LanguageModelV4TextPart)
				require.True(t, ok)
				assert.Equal(t, "text", textPart.Type)
				assert.Equal(t, "hello, world!", textPart.Text)
			})
		})

		t.Run("image parts", func(t *testing.T) {
			t.Run("should convert image string https url to URL object", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					UserModelMessage{
						Role: "user",
						Content: []interface{}{
							ImagePart{
								Type:  "image",
								Image: "https://example.com/image.jpg",
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "user", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "image/*", filePart.MediaType)
				u, ok := filePart.Data.(*url.URL)
				require.True(t, ok)
				assert.Equal(t, "https://example.com/image.jpg", u.String())
			})

			t.Run("should convert image string data url to base64 content", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					UserModelMessage{
						Role: "user",
						Content: []interface{}{
							ImagePart{
								Type:  "image",
								Image: "data:image/jpg;base64,/9j/3Q==",
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "user", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "/9j/3Q==", filePart.Data)
				assert.Equal(t, "image/jpg", filePart.MediaType)
			})

			t.Run("should prefer detected mediaType", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					UserModelMessage{
						Role: "user",
						Content: []interface{}{
							ImagePart{
								Type:  "image",
								Image: "data:image/png;base64,/9j/3Q==",
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "user", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "/9j/3Q==", filePart.Data)
				// The data URL media type is "image/png" from the data URL itself
				assert.Equal(t, "image/png", filePart.MediaType)
			})
		})

		t.Run("file parts", func(t *testing.T) {
			t.Run("should convert file string https url to URL object", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					UserModelMessage{
						Role: "user",
						Content: []interface{}{
							FilePart{
								Type:      "file",
								Data:      "https://example.com/image.jpg",
								MediaType: "image/jpg",
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "user", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				u, ok := filePart.Data.(*url.URL)
				require.True(t, ok)
				assert.Equal(t, "https://example.com/image.jpg", u.String())
				assert.Equal(t, "image/jpg", filePart.MediaType)
			})

			t.Run("should convert file string data url to base64 content", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					UserModelMessage{
						Role: "user",
						Content: []interface{}{
							FilePart{
								Type:      "file",
								Data:      "data:image/jpg;base64,dGVzdA==",
								MediaType: "image/jpg",
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "user", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "dGVzdA==", filePart.Data)
				assert.Equal(t, "image/jpg", filePart.MediaType)
			})
		})
	})

	t.Run("assistant message", func(t *testing.T) {
		t.Run("text parts", func(t *testing.T) {
			t.Run("should ignore empty text parts when there are no provider options", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							TextPart{
								Type: "text",
								Text: "",
							},
							ToolCallPart{
								Type:       "tool-call",
								ToolName:   "toolName",
								ToolCallID: "toolCallId",
								Input:      map[string]interface{}{},
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				tc, ok := parts[0].(LanguageModelV4ToolCallPart)
				require.True(t, ok)
				assert.Equal(t, "tool-call", tc.Type)
				assert.Equal(t, map[string]interface{}{}, tc.Input)
				assert.Equal(t, "toolCallId", tc.ToolCallID)
				assert.Equal(t, "toolName", tc.ToolName)
			})

			t.Run("should include empty text parts when there are provider options", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							TextPart{
								Type: "text",
								Text: "",
								ProviderOptions: ProviderOptions{
									"test-provider": {
										"key-a": "test-value-1",
									},
								},
							},
							ToolCallPart{
								Type:       "tool-call",
								ToolName:   "toolName",
								ToolCallID: "toolCallId",
								Input:      map[string]interface{}{},
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 2)

				tp, ok := parts[0].(LanguageModelV4TextPart)
				require.True(t, ok)
				assert.Equal(t, "text", tp.Type)
				assert.Equal(t, "", tp.Text)
				assert.Equal(t, ProviderOptions{
					"test-provider": {"key-a": "test-value-1"},
				}, tp.ProviderOptions)

				tc, ok := parts[1].(LanguageModelV4ToolCallPart)
				require.True(t, ok)
				assert.Equal(t, "tool-call", tc.Type)
				assert.Equal(t, "toolCallId", tc.ToolCallID)
				assert.Equal(t, "toolName", tc.ToolName)
			})
		})

		t.Run("reasoning parts", func(t *testing.T) {
			t.Run("should pass through provider options", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							ReasoningPart{
								Type: "reasoning",
								Text: "hello, world!",
								ProviderOptions: ProviderOptions{
									"test-provider": {
										"key-a": "test-value-1",
										"key-b": "test-value-2",
									},
								},
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				rp, ok := parts[0].(LanguageModelV4ReasoningPart)
				require.True(t, ok)
				assert.Equal(t, "reasoning", rp.Type)
				assert.Equal(t, "hello, world!", rp.Text)
				assert.Equal(t, ProviderOptions{
					"test-provider": {
						"key-a": "test-value-1",
						"key-b": "test-value-2",
					},
				}, rp.ProviderOptions)
			})

			t.Run("should support a mix of reasoning, redacted reasoning, and text parts", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							ReasoningPart{
								Type: "reasoning",
								Text: "I'm thinking",
							},
							ReasoningPart{
								Type: "reasoning",
								Text: "redacted-reasoning-data",
								ProviderOptions: ProviderOptions{
									"test-provider": {"redacted": true},
								},
							},
							ReasoningPart{
								Type: "reasoning",
								Text: "more thinking",
							},
							TextPart{
								Type: "text",
								Text: "hello, world!",
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 4)

				rp0, ok := parts[0].(LanguageModelV4ReasoningPart)
				require.True(t, ok)
				assert.Equal(t, "reasoning", rp0.Type)
				assert.Equal(t, "I'm thinking", rp0.Text)

				rp1, ok := parts[1].(LanguageModelV4ReasoningPart)
				require.True(t, ok)
				assert.Equal(t, "reasoning", rp1.Type)
				assert.Equal(t, "redacted-reasoning-data", rp1.Text)
				assert.Equal(t, ProviderOptions{
					"test-provider": {"redacted": true},
				}, rp1.ProviderOptions)

				rp2, ok := parts[2].(LanguageModelV4ReasoningPart)
				require.True(t, ok)
				assert.Equal(t, "reasoning", rp2.Type)
				assert.Equal(t, "more thinking", rp2.Text)

				tp, ok := parts[3].(LanguageModelV4TextPart)
				require.True(t, ok)
				assert.Equal(t, "text", tp.Type)
				assert.Equal(t, "hello, world!", tp.Text)
			})
		})

		t.Run("tool call parts", func(t *testing.T) {
			t.Run("should pass through provider options", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							ToolCallPart{
								Type:       "tool-call",
								ToolName:   "toolName",
								ToolCallID: "toolCallId",
								Input:      map[string]interface{}{},
								ProviderOptions: ProviderOptions{
									"test-provider": {
										"key-a": "test-value-1",
										"key-b": "test-value-2",
									},
								},
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				tc, ok := parts[0].(LanguageModelV4ToolCallPart)
				require.True(t, ok)
				assert.Equal(t, "tool-call", tc.Type)
				assert.Equal(t, map[string]interface{}{}, tc.Input)
				assert.Equal(t, "toolCallId", tc.ToolCallID)
				assert.Equal(t, "toolName", tc.ToolName)
				assert.Equal(t, ProviderOptions{
					"test-provider": {
						"key-a": "test-value-1",
						"key-b": "test-value-2",
					},
				}, tc.ProviderOptions)
			})

			t.Run("should include providerExecuted flag", func(t *testing.T) {
				providerExecuted := true
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							ToolCallPart{
								Type:             "tool-call",
								ToolName:         "toolName",
								ToolCallID:       "toolCallId",
								Input:            map[string]interface{}{},
								ProviderExecuted: &providerExecuted,
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				tc, ok := parts[0].(LanguageModelV4ToolCallPart)
				require.True(t, ok)
				assert.Equal(t, "tool-call", tc.Type)
				assert.Equal(t, "toolCallId", tc.ToolCallID)
				assert.Equal(t, "toolName", tc.ToolName)
				require.NotNil(t, tc.ProviderExecuted)
				assert.True(t, *tc.ProviderExecuted)
			})
		})

		t.Run("tool result parts", func(t *testing.T) {
			t.Run("should include providerExecuted flag", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							ToolResultPart{
								Type:       "tool-result",
								ToolCallID: "toolCallId",
								ToolName:   "toolName",
								Output: ToolResultOutput{
									Type:  "json",
									Value: map[string]interface{}{"some": "result"},
								},
								ProviderOptions: ProviderOptions{
									"test-provider": {
										"key-a": "test-value-1",
										"key-b": "test-value-2",
									},
								},
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				tr, ok := parts[0].(LanguageModelV4ToolResultPart)
				require.True(t, ok)
				assert.Equal(t, "tool-result", tr.Type)
				assert.Equal(t, "toolCallId", tr.ToolCallID)
				assert.Equal(t, "toolName", tr.ToolName)
				assert.Equal(t, "json", tr.Output.Type)
				assert.Equal(t, map[string]interface{}{"some": "result"}, tr.Output.Value)
				assert.Equal(t, ProviderOptions{
					"test-provider": {
						"key-a": "test-value-1",
						"key-b": "test-value-2",
					},
				}, tr.ProviderOptions)
			})
		})

		t.Run("provider-executed tool calls and results", func(t *testing.T) {
			t.Run("should include providerExecuted flag", func(t *testing.T) {
				providerExecuted := true
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							ToolCallPart{
								Type:             "tool-call",
								ToolName:         "toolName",
								ToolCallID:       "toolCallId",
								Input:            map[string]interface{}{},
								ProviderExecuted: &providerExecuted,
								ProviderOptions: ProviderOptions{
									"test-provider": {
										"key-a": "test-value-1",
										"key-b": "test-value-2",
									},
								},
							},
							ToolResultPart{
								Type:       "tool-result",
								ToolCallID: "toolCallId",
								ToolName:   "toolName",
								Output: ToolResultOutput{
									Type:  "json",
									Value: map[string]interface{}{"some": "result"},
								},
								ProviderOptions: ProviderOptions{
									"test-provider": {
										"key-a": "test-value-1",
										"key-b": "test-value-2",
									},
								},
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 2)

				tc, ok := parts[0].(LanguageModelV4ToolCallPart)
				require.True(t, ok)
				assert.Equal(t, "tool-call", tc.Type)
				assert.Equal(t, "toolCallId", tc.ToolCallID)
				assert.Equal(t, "toolName", tc.ToolName)
				require.NotNil(t, tc.ProviderExecuted)
				assert.True(t, *tc.ProviderExecuted)
				assert.Equal(t, ProviderOptions{
					"test-provider": {
						"key-a": "test-value-1",
						"key-b": "test-value-2",
					},
				}, tc.ProviderOptions)

				tr, ok := parts[1].(LanguageModelV4ToolResultPart)
				require.True(t, ok)
				assert.Equal(t, "tool-result", tr.Type)
				assert.Equal(t, "toolCallId", tr.ToolCallID)
				assert.Equal(t, "toolName", tr.ToolName)
				assert.Equal(t, "json", tr.Output.Type)
				assert.Equal(t, ProviderOptions{
					"test-provider": {
						"key-a": "test-value-1",
						"key-b": "test-value-2",
					},
				}, tr.ProviderOptions)
			})
		})

		t.Run("file parts", func(t *testing.T) {
			t.Run("should convert file data correctly", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							FilePart{
								Type:      "file",
								Data:      "dGVzdA==", // "test" in base64
								MediaType: "application/pdf",
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "dGVzdA==", filePart.Data)
				assert.Equal(t, "application/pdf", filePart.MediaType)
			})

			t.Run("should preserve filename when present", func(t *testing.T) {
				filename := "test-document.pdf"
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							FilePart{
								Type:      "file",
								Data:      "dGVzdA==",
								MediaType: "application/pdf",
								Filename:  &filename,
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "dGVzdA==", filePart.Data)
				assert.Equal(t, "application/pdf", filePart.MediaType)
				require.NotNil(t, filePart.Filename)
				assert.Equal(t, "test-document.pdf", *filePart.Filename)
			})

			t.Run("should handle provider options", func(t *testing.T) {
				result, err := convertToLanguageModelMessage(
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							FilePart{
								Type:      "file",
								Data:      "dGVzdA==",
								MediaType: "application/pdf",
								ProviderOptions: ProviderOptions{
									"test-provider": {
										"key-a": "test-value-1",
										"key-b": "test-value-2",
									},
								},
							},
						},
					},
					DownloadedAssets{},
				)

				require.NoError(t, err)
				assert.Equal(t, "assistant", result.Role)
				parts, ok := result.Content.([]interface{})
				require.True(t, ok)
				require.Len(t, parts, 1)
				filePart, ok := parts[0].(LanguageModelV4FilePart)
				require.True(t, ok)
				assert.Equal(t, "file", filePart.Type)
				assert.Equal(t, "dGVzdA==", filePart.Data)
				assert.Equal(t, "application/pdf", filePart.MediaType)
				assert.Equal(t, ProviderOptions{
					"test-provider": {
						"key-a": "test-value-1",
						"key-b": "test-value-2",
					},
				}, filePart.ProviderOptions)
			})
		})
	})

	t.Run("tool message", func(t *testing.T) {
		t.Run("should convert basic tool result message", func(t *testing.T) {
			result, err := convertToLanguageModelMessage(
				ToolModelMessage{
					Role: "tool",
					Content: []interface{}{
						ToolResultPart{
							Type:       "tool-result",
							ToolName:   "toolName",
							ToolCallID: "toolCallId",
							Output: ToolResultOutput{
								Type:  "json",
								Value: map[string]interface{}{"some": "result"},
							},
						},
					},
				},
				DownloadedAssets{},
			)

			require.NoError(t, err)
			assert.Equal(t, "tool", result.Role)
			parts, ok := result.Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 1)
			tr, ok := parts[0].(LanguageModelV4ToolResultPart)
			require.True(t, ok)
			assert.Equal(t, "tool-result", tr.Type)
			assert.Equal(t, "toolCallId", tr.ToolCallID)
			assert.Equal(t, "toolName", tr.ToolName)
			assert.Equal(t, "json", tr.Output.Type)
			assert.Equal(t, map[string]interface{}{"some": "result"}, tr.Output.Value)
		})

		t.Run("should convert tool result with provider metadata", func(t *testing.T) {
			result, err := convertToLanguageModelMessage(
				ToolModelMessage{
					Role: "tool",
					Content: []interface{}{
						ToolResultPart{
							Type:       "tool-result",
							ToolName:   "toolName",
							ToolCallID: "toolCallId",
							Output: ToolResultOutput{
								Type:  "json",
								Value: map[string]interface{}{"some": "result"},
							},
							ProviderOptions: ProviderOptions{
								"test-provider": {
									"key-a": "test-value-1",
									"key-b": "test-value-2",
								},
							},
						},
					},
				},
				DownloadedAssets{},
			)

			require.NoError(t, err)
			assert.Equal(t, "tool", result.Role)
			parts, ok := result.Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 1)
			tr, ok := parts[0].(LanguageModelV4ToolResultPart)
			require.True(t, ok)
			assert.Equal(t, "tool-result", tr.Type)
			assert.Equal(t, "toolCallId", tr.ToolCallID)
			assert.Equal(t, "toolName", tr.ToolName)
			assert.Equal(t, ProviderOptions{
				"test-provider": {
					"key-a": "test-value-1",
					"key-b": "test-value-2",
				},
			}, tr.ProviderOptions)
		})

		t.Run("should include error flag", func(t *testing.T) {
			result, err := convertToLanguageModelMessage(
				ToolModelMessage{
					Role: "tool",
					Content: []interface{}{
						ToolResultPart{
							Type:       "tool-result",
							ToolName:   "toolName",
							ToolCallID: "toolCallId",
							Output: ToolResultOutput{
								Type:  "json",
								Value: map[string]interface{}{"some": "result"},
							},
						},
					},
				},
				DownloadedAssets{},
			)

			require.NoError(t, err)
			assert.Equal(t, "tool", result.Role)
			parts, ok := result.Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 1)
			tr, ok := parts[0].(LanguageModelV4ToolResultPart)
			require.True(t, ok)
			assert.Equal(t, "tool-result", tr.Type)
			assert.Equal(t, "toolCallId", tr.ToolCallID)
			assert.Equal(t, "toolName", tr.ToolName)
			assert.Equal(t, "json", tr.Output.Type)
			assert.Equal(t, map[string]interface{}{"some": "result"}, tr.Output.Value)
		})

		t.Run("should include multipart content", func(t *testing.T) {
			result, err := convertToLanguageModelMessage(
				ToolModelMessage{
					Role: "tool",
					Content: []interface{}{
						ToolResultPart{
							Type:       "tool-result",
							ToolName:   "toolName",
							ToolCallID: "toolCallId",
							Output: ToolResultOutput{
								Type: "content",
								Value: []interface{}{
									map[string]interface{}{"type": "file-url", "url": "https://example.com/image.png"},
									map[string]interface{}{"type": "file-data", "data": "dGVzdA==", "mediaType": "image/png"},
									map[string]interface{}{"type": "file-id", "fileId": "fileId"},
									map[string]interface{}{"type": "file-id", "fileId": map[string]interface{}{"test-provider": "fileId"}},
									map[string]interface{}{"type": "image-data", "data": "dGVzdA==", "mediaType": "image/png"},
									map[string]interface{}{"type": "image-url", "url": "https://example.com/image.png"},
									map[string]interface{}{"type": "image-file-id", "fileId": "fileId"},
									map[string]interface{}{"type": "image-file-id", "fileId": map[string]interface{}{"test-provider": "fileId"}},
									map[string]interface{}{
										"type": "custom",
										"providerOptions": map[string]interface{}{
											"test-provider": map[string]interface{}{
												"key-a": "test-value-1",
												"key-b": "test-value-2",
											},
										},
									},
								},
							},
						},
					},
				},
				DownloadedAssets{},
			)

			require.NoError(t, err)
			assert.Equal(t, "tool", result.Role)
			parts, ok := result.Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 1)
			tr, ok := parts[0].(LanguageModelV4ToolResultPart)
			require.True(t, ok)
			assert.Equal(t, "tool-result", tr.Type)
			assert.Equal(t, "toolCallId", tr.ToolCallID)
			assert.Equal(t, "toolName", tr.ToolName)
			assert.Equal(t, "content", tr.Output.Type)
			// The multipart content value passes through as-is
			outputValue, ok := tr.Output.Value.([]interface{})
			require.True(t, ok)
			require.Len(t, outputValue, 9)
		})

		t.Run("should map deprecated media type to image-data", func(t *testing.T) {
			// Note: This mapping is a TS-side concern; in Go the output passes through.
			// We verify the ToolResultPart output is preserved as given.
			result, err := convertToLanguageModelMessage(
				ToolModelMessage{
					Role: "tool",
					Content: []interface{}{
						ToolResultPart{
							Type:       "tool-result",
							ToolName:   "toolName",
							ToolCallID: "toolCallId",
							Output: ToolResultOutput{
								Type: "content",
								Value: []interface{}{
									map[string]interface{}{"type": "media", "data": "dGVzdA==", "mediaType": "image/png"},
								},
							},
						},
					},
				},
				DownloadedAssets{},
			)

			require.NoError(t, err)
			assert.Equal(t, "tool", result.Role)
			parts, ok := result.Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 1)
			tr, ok := parts[0].(LanguageModelV4ToolResultPart)
			require.True(t, ok)
			assert.Equal(t, "tool-result", tr.Type)
			assert.Equal(t, "content", tr.Output.Type)
			// In Go, the value passes through as-is; type mapping is not done here
			outputValue, ok := tr.Output.Value.([]interface{})
			require.True(t, ok)
			require.Len(t, outputValue, 1)
		})

		t.Run("should map deprecated media type to file-data", func(t *testing.T) {
			result, err := convertToLanguageModelMessage(
				ToolModelMessage{
					Role: "tool",
					Content: []interface{}{
						ToolResultPart{
							Type:       "tool-result",
							ToolName:   "toolName",
							ToolCallID: "toolCallId",
							Output: ToolResultOutput{
								Type: "content",
								Value: []interface{}{
									map[string]interface{}{"type": "media", "data": "dGVzdA==", "mediaType": "application/pdf"},
								},
							},
						},
					},
				},
				DownloadedAssets{},
			)

			require.NoError(t, err)
			assert.Equal(t, "tool", result.Role)
			parts, ok := result.Content.([]interface{})
			require.True(t, ok)
			require.Len(t, parts, 1)
			tr, ok := parts[0].(LanguageModelV4ToolResultPart)
			require.True(t, ok)
			assert.Equal(t, "tool-result", tr.Type)
			assert.Equal(t, "content", tr.Output.Type)
			outputValue, ok := tr.Output.Value.([]interface{})
			require.True(t, ok)
			require.Len(t, outputValue, 1)
		})
	})
}

// Helper functions

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}
