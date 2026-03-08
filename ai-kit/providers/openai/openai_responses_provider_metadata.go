// Ported from: packages/openai/src/responses/openai-responses-provider-metadata.ts
package openai

// ResponsesProviderMetadata contains provider-level metadata for a Responses API call.
type ResponsesProviderMetadata struct {
	ResponseID string                   `json:"responseId"`
	Logprobs   []OpenAIResponsesLogprob `json:"logprobs,omitempty"`
	ServiceTier string                  `json:"serviceTier,omitempty"`
}

// ResponsesReasoningProviderMetadata contains provider metadata for reasoning content.
type ResponsesReasoningProviderMetadata struct {
	ItemID                    string  `json:"itemId"`
	ReasoningEncryptedContent *string `json:"reasoningEncryptedContent,omitempty"`
}

// ResponsesTextProviderMetadata contains provider metadata for text content.
type ResponsesTextProviderMetadata struct {
	ItemID      string `json:"itemId"`
	Phase       string `json:"phase,omitempty"`
	Annotations []any  `json:"annotations,omitempty"`
}

// ResponsesSourceDocumentProviderMetadata represents source document metadata.
// It can be one of: FileCitationMetadata, ContainerFileCitationMetadata, FilePathMetadata.
type ResponsesSourceDocumentProviderMetadata interface {
	sourceDocumentMetadataType() string
}

// FileCitationMetadata is source document metadata for a file citation.
type FileCitationMetadata struct {
	Type   string `json:"type"` // "file_citation"
	FileID string `json:"fileId"`
	Index  int    `json:"index"`
}

func (FileCitationMetadata) sourceDocumentMetadataType() string { return "file_citation" }

// ContainerFileCitationMetadata is source document metadata for a container file citation.
type ContainerFileCitationMetadata struct {
	Type        string `json:"type"` // "container_file_citation"
	FileID      string `json:"fileId"`
	ContainerID string `json:"containerId"`
}

func (ContainerFileCitationMetadata) sourceDocumentMetadataType() string {
	return "container_file_citation"
}

// FilePathMetadata is source document metadata for a file path.
type FilePathMetadata struct {
	Type   string `json:"type"` // "file_path"
	FileID string `json:"fileId"`
	Index  int    `json:"index"`
}

func (FilePathMetadata) sourceDocumentMetadataType() string { return "file_path" }
