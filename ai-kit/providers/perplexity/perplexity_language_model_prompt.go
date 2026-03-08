// Ported from: packages/perplexity/src/perplexity-language-model-prompt.ts
package perplexity

// PerplexityPrompt is a list of Perplexity messages.
type PerplexityPrompt = []PerplexityMessage

// PerplexityMessage represents a message in the Perplexity API format.
type PerplexityMessage struct {
	Role    string `json:"role"` // "system", "user", or "assistant"
	Content any    `json:"content"` // string or []PerplexityMessageContent
}

// PerplexityMessageContent is a content part within a message.
// It can be a text part, image_url part, or file_url part.
type PerplexityMessageContent interface {
	perplexityContentType() string
}

// PerplexityTextContent is a text content part.
type PerplexityTextContent struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

func (PerplexityTextContent) perplexityContentType() string { return "text" }

// PerplexityImageURLContent is an image URL content part.
type PerplexityImageURLContent struct {
	Type     string                      `json:"type"` // "image_url"
	ImageURL PerplexityImageURLReference `json:"image_url"`
}

func (PerplexityImageURLContent) perplexityContentType() string { return "image_url" }

// PerplexityImageURLReference holds the URL for an image.
type PerplexityImageURLReference struct {
	URL string `json:"url"`
}

// PerplexityFileURLContent is a file URL content part.
type PerplexityFileURLContent struct {
	Type     string                     `json:"type"` // "file_url"
	FileURL  PerplexityFileURLReference `json:"file_url"`
	FileName *string                    `json:"file_name,omitempty"`
}

func (PerplexityFileURLContent) perplexityContentType() string { return "file_url" }

// PerplexityFileURLReference holds the URL for a file.
type PerplexityFileURLReference struct {
	URL string `json:"url"`
}
