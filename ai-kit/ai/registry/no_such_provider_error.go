// Ported from: packages/ai/src/registry/no-such-provider-error.ts
package registry

import (
	"fmt"
	"strings"

	aierrors "github.com/brainlet/brainkit/ai-kit/provider/errors"
)

// NoSuchProviderError indicates that the requested provider does not exist.
// It extends NoSuchModelError, mirroring the TypeScript inheritance chain.
type NoSuchProviderError struct {
	aierrors.NoSuchModelError

	// ProviderID is the ID of the provider that was not found.
	ProviderID string

	// AvailableProviders is the list of available provider IDs.
	AvailableProviders []string
}

// NoSuchProviderErrorOptions are the options for creating a NoSuchProviderError.
type NoSuchProviderErrorOptions struct {
	ModelID            string
	ModelType          aierrors.ModelType
	ProviderID         string
	AvailableProviders []string
	Message            string
}

// NewNoSuchProviderError creates a new NoSuchProviderError.
func NewNoSuchProviderError(opts NoSuchProviderErrorOptions) *NoSuchProviderError {
	message := opts.Message
	if message == "" {
		message = fmt.Sprintf(
			"No such provider: %s (available providers: %s)",
			opts.ProviderID,
			strings.Join(opts.AvailableProviders, ","),
		)
	}

	return &NoSuchProviderError{
		NoSuchModelError: aierrors.NoSuchModelError{
			AISDKError: aierrors.AISDKError{
				Name:    "AI_NoSuchProviderError",
				Message: message,
			},
			ModelID:   opts.ModelID,
			ModelType: opts.ModelType,
		},
		ProviderID:         opts.ProviderID,
		AvailableProviders: opts.AvailableProviders,
	}
}

// IsNoSuchProviderError checks if an error is a NoSuchProviderError.
func IsNoSuchProviderError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*NoSuchProviderError)
	return ok
}
