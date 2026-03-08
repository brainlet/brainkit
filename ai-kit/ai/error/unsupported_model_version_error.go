// Ported from: packages/ai/src/error/unsupported-model-version-error.ts
package aierror

import "fmt"

const unsupportedModelVersionErrorName = "AI_UnsupportedModelVersionError"

// UnsupportedModelVersionError is returned when a model with an unsupported version is used.
type UnsupportedModelVersionError struct {
	AISDKError

	// Version is the unsupported model version string.
	Version string

	// Provider is the provider name.
	Provider string

	// ModelID is the model identifier.
	ModelID string
}

// NewUnsupportedModelVersionError creates a new UnsupportedModelVersionError.
func NewUnsupportedModelVersionError(version, provider, modelID string) *UnsupportedModelVersionError {
	return &UnsupportedModelVersionError{
		AISDKError: AISDKError{
			Name: unsupportedModelVersionErrorName,
			Message: fmt.Sprintf(
				`Unsupported model version %s for provider "%s" and model "%s". `+
					`AI SDK 5 only supports models that implement specification version "v2".`,
				version, provider, modelID,
			),
		},
		Version:  version,
		Provider: provider,
		ModelID:  modelID,
	}
}

// IsUnsupportedModelVersionError checks whether the given error is an UnsupportedModelVersionError.
func IsUnsupportedModelVersionError(err error) bool {
	_, ok := err.(*UnsupportedModelVersionError)
	return ok
}
