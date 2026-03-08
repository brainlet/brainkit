// Ported from: packages/provider/src/language-model/v3/language-model-v3.ts
package languagemodel

import "regexp"

// LanguageModel is the specification for a language model that implements
// the language model interface version 3.
type LanguageModel interface {
	// SpecificationVersion returns the language model interface version.
	// Must return "v3".
	SpecificationVersion() string

	// Provider returns the provider ID.
	Provider() string

	// ModelID returns the provider-specific model ID.
	ModelID() string

	// SupportedUrls returns the supported URL patterns by media type for the provider.
	//
	// The keys are media type patterns or full media types (e.g. "*/*" for everything,
	// "audio/*", "video/*", or "application/pdf") and the values are arrays of regular
	// expressions that match the URL paths.
	//
	// The matching should be against lower-case URLs.
	// Matched URLs are supported natively by the model and are not downloaded.
	SupportedUrls() (map[string][]*regexp.Regexp, error)

	// DoGenerate generates a language model output (non-streaming).
	//
	// Naming: "Do" prefix to prevent accidental direct usage of the method
	// by the user.
	DoGenerate(options CallOptions) (GenerateResult, error)

	// DoStream generates a language model output (streaming).
	//
	// Naming: "Do" prefix to prevent accidental direct usage of the method
	// by the user.
	DoStream(options CallOptions) (StreamResult, error)
}
