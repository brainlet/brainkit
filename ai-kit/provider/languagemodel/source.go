// Ported from: packages/provider/src/language-model/v3/language-model-v3-source.ts
package languagemodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// Source represents a source that has been used as input to generate the response.
// This is a sealed interface; implementations are SourceURL and SourceDocument.
type Source interface {
	isContent()
	isStreamPart()
	sourceType() string
}

// SourceURL references web content.
type SourceURL struct {
	// ID is the unique identifier of the source.
	ID string

	// URL is the URL of the source.
	URL string

	// Title is the title of the source.
	Title *string

	// ProviderMetadata is additional provider metadata for the source.
	ProviderMetadata shared.ProviderMetadata
}

func (SourceURL) isContent()          {}
func (SourceURL) isStreamPart()       {}
func (SourceURL) sourceType() string  { return "url" }

// SourceDocument references files/documents.
type SourceDocument struct {
	// ID is the unique identifier of the source.
	ID string

	// MediaType is the IANA media type of the document (e.g., "application/pdf").
	MediaType string

	// Title is the title of the document.
	Title string

	// Filename is an optional filename of the document.
	Filename *string

	// ProviderMetadata is additional provider metadata for the source.
	ProviderMetadata shared.ProviderMetadata
}

func (SourceDocument) isContent()          {}
func (SourceDocument) isStreamPart()       {}
func (SourceDocument) sourceType() string  { return "document" }
