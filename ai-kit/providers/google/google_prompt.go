// Ported from: packages/google/src/google-generative-ai-prompt.ts
package google

// GooglePrompt is the Google Generative AI prompt structure.
type GooglePrompt struct {
	SystemInstruction *GoogleSystemInstruction `json:"systemInstruction,omitempty"`
	Contents          []GoogleContent          `json:"contents"`
}

// GoogleSystemInstruction represents a system instruction.
type GoogleSystemInstruction struct {
	Parts []GoogleTextPart `json:"parts"`
}

// GoogleTextPart is a simple text part.
type GoogleTextPart struct {
	Text string `json:"text"`
}

// GoogleContent represents a content message in the Google API format.
type GoogleContent struct {
	Role  string              `json:"role"`
	Parts []GoogleContentPart `json:"parts"`
}

// GoogleContentPart is a union type representing the different parts that
// can appear in a Google content message.
type GoogleContentPart struct {
	// Text content.
	Text *string `json:"text,omitempty"`

	// Whether this is a thought/reasoning part.
	Thought *bool `json:"thought,omitempty"`

	// ThoughtSignature for linking thoughts to responses.
	ThoughtSignature *string `json:"thoughtSignature,omitempty"`

	// InlineData for base64 encoded media.
	InlineData *GoogleInlineData `json:"inlineData,omitempty"`

	// FunctionCall for tool calls.
	FunctionCall *GoogleFunctionCall `json:"functionCall,omitempty"`

	// FunctionResponse for tool results.
	FunctionResponse *GoogleFunctionResponse `json:"functionResponse,omitempty"`

	// FileData for file references by URI.
	FileData *GoogleFileData `json:"fileData,omitempty"`
}

// GoogleInlineData represents inline base64 encoded data.
type GoogleInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// GoogleFunctionCall represents a function call from the model.
type GoogleFunctionCall struct {
	Name string `json:"name"`
	Args any    `json:"args"`
}

// GoogleFunctionResponse represents a function response to the model.
type GoogleFunctionResponse struct {
	Name     string `json:"name"`
	Response any    `json:"response"`
}

// GoogleFileData represents file data referenced by URI.
type GoogleFileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

// GoogleGroundingMetadata is the grounding metadata from the API.
type GoogleGroundingMetadata struct {
	WebSearchQueries  []string                `json:"webSearchQueries,omitempty"`
	ImageSearchQueries []string               `json:"imageSearchQueries,omitempty"`
	RetrievalQueries  []string                `json:"retrievalQueries,omitempty"`
	SearchEntryPoint  *SearchEntryPoint       `json:"searchEntryPoint,omitempty"`
	GroundingChunks   []GroundingChunk        `json:"groundingChunks,omitempty"`
	GroundingSupports []GroundingSupport      `json:"groundingSupports,omitempty"`
	RetrievalMetadata map[string]any          `json:"retrievalMetadata,omitempty"`
}

// SearchEntryPoint contains rendered content for a search entry point.
type SearchEntryPoint struct {
	RenderedContent string `json:"renderedContent"`
}

// GroundingChunk is a grounding chunk from the API response.
type GroundingChunk struct {
	Web              *GroundingChunkWeb              `json:"web,omitempty"`
	Image            *GroundingChunkImage            `json:"image,omitempty"`
	RetrievedContext *GroundingChunkRetrievedContext  `json:"retrievedContext,omitempty"`
	Maps             *GroundingChunkMaps             `json:"maps,omitempty"`
}

// GroundingChunkWeb represents a web grounding chunk.
type GroundingChunkWeb struct {
	URI   string  `json:"uri"`
	Title *string `json:"title,omitempty"`
}

// GroundingChunkImage represents an image grounding chunk.
type GroundingChunkImage struct {
	SourceURI string  `json:"sourceUri"`
	ImageURI  string  `json:"imageUri"`
	Title     *string `json:"title,omitempty"`
	Domain    *string `json:"domain,omitempty"`
}

// GroundingChunkRetrievedContext represents a retrieved context grounding chunk.
type GroundingChunkRetrievedContext struct {
	URI             *string `json:"uri,omitempty"`
	Title           *string `json:"title,omitempty"`
	Text            *string `json:"text,omitempty"`
	FileSearchStore *string `json:"fileSearchStore,omitempty"`
}

// GroundingChunkMaps represents a maps grounding chunk.
type GroundingChunkMaps struct {
	URI     *string `json:"uri,omitempty"`
	Title   *string `json:"title,omitempty"`
	Text    *string `json:"text,omitempty"`
	PlaceID *string `json:"placeId,omitempty"`
}

// GroundingSupport contains support information for grounding.
type GroundingSupport struct {
	Segment              *GroundingSegment `json:"segment,omitempty"`
	SegmentText          *string           `json:"segment_text,omitempty"`
	GroundingChunkIndices []int            `json:"groundingChunkIndices,omitempty"`
	SupportChunkIndices  []int            `json:"supportChunkIndices,omitempty"`
	ConfidenceScores     []float64        `json:"confidenceScores,omitempty"`
	ConfidenceScore      []float64        `json:"confidenceScore,omitempty"`
}

// GroundingSegment represents a segment in grounding support.
type GroundingSegment struct {
	StartIndex *int    `json:"startIndex,omitempty"`
	EndIndex   *int    `json:"endIndex,omitempty"`
	Text       *string `json:"text,omitempty"`
}

// GoogleURLContextMetadata contains URL context metadata from the API.
type GoogleURLContextMetadata struct {
	URLMetadata []URLMetadataEntry `json:"urlMetadata,omitempty"`
}

// URLMetadataEntry is a single URL metadata entry.
type URLMetadataEntry struct {
	RetrievedURL       string `json:"retrievedUrl"`
	URLRetrievalStatus string `json:"urlRetrievalStatus"`
}

// GoogleSafetyRating represents a safety rating from the API.
type GoogleSafetyRating struct {
	Category         *string  `json:"category,omitempty"`
	Probability      *string  `json:"probability,omitempty"`
	ProbabilityScore *float64 `json:"probabilityScore,omitempty"`
	Severity         *string  `json:"severity,omitempty"`
	SeverityScore    *float64 `json:"severityScore,omitempty"`
	Blocked          *bool    `json:"blocked,omitempty"`
}

// GoogleProviderMetadata contains provider-specific metadata for Google
// Generative AI responses.
type GoogleProviderMetadata struct {
	GroundingMetadata  *GoogleGroundingMetadata  `json:"groundingMetadata,omitempty"`
	URLContextMetadata *GoogleURLContextMetadata `json:"urlContextMetadata,omitempty"`
	SafetyRatings      []GoogleSafetyRating      `json:"safetyRatings,omitempty"`
}
