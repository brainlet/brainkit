// Ported from: packages/ai/src/prompt/wrap-gateway-error.ts
package prompt

import (
	"fmt"
	"os"
)

// GatewayAuthenticationError represents an authentication error from the AI Gateway.
// TODO: import from brainlink/experiments/ai-kit/gateway once it exists
type GatewayAuthenticationError struct {
	Name    string
	Message string
}

func (e *GatewayAuthenticationError) Error() string {
	return e.Message
}

// IsGatewayAuthenticationError checks whether the given error is a GatewayAuthenticationError.
func IsGatewayAuthenticationError(err error) bool {
	_, ok := err.(*GatewayAuthenticationError)
	return ok
}

// WrapGatewayError wraps a GatewayAuthenticationError with a more helpful message.
// If the error is not a GatewayAuthenticationError, it is returned as-is.
func WrapGatewayError(err error) error {
	if !IsGatewayAuthenticationError(err) {
		return err
	}

	moreInfoURL := "https://ai-sdk.dev/unauthenticated-ai-gateway"
	isProductionEnv := os.Getenv("NODE_ENV") == "production"

	if isProductionEnv {
		return fmt.Errorf(
			"Unauthenticated. Configure AI_GATEWAY_API_KEY or use a provider module. Learn more: %s",
			moreInfoURL,
		)
	}

	return fmt.Errorf(
		"Unauthenticated request to AI Gateway.\n\n"+
			"To authenticate, set the AI_GATEWAY_API_KEY environment variable with your API key.\n\n"+
			"Alternatively, you can use a provider module instead of the AI Gateway.\n\n"+
			"Learn more: %s\n",
		moreInfoURL,
	)
}
