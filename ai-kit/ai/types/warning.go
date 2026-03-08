// Ported from: packages/ai/src/types/warning.ts
package aitypes

// Warning from the model provider for this call. The call will proceed, but e.g.
// some settings might not be supported, which can lead to suboptimal results.
//
// Corresponds to SharedV4Warning from @ai-sdk/provider.
type Warning struct {
	// Type of warning: "unsupported", "compatibility", or "other".
	Type string `json:"type"`

	// Feature is the feature that is not supported or used in compatibility mode.
	// Applicable when Type is "unsupported" or "compatibility".
	Feature string `json:"feature,omitempty"`

	// Details provides additional details about the warning.
	// Applicable when Type is "unsupported" or "compatibility".
	Details string `json:"details,omitempty"`

	// Message is the warning message.
	// Applicable when Type is "other".
	Message string `json:"message,omitempty"`
}
