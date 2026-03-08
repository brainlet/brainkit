// Ported from: packages/provider/src/errors/ai-sdk-error.ts
package errors

import "fmt"

// AISDKError is the base custom error type for AI SDK related errors.
type AISDKError struct {
	// Name is the error type name (e.g., "AI_APICallError").
	Name string

	// Message is the human-readable error message.
	Message string

	// Cause is the underlying cause of the error, if any.
	Cause error
}

// Error implements the error interface.
func (e *AISDKError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Name, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

// Unwrap returns the underlying cause for errors.Is/As support.
func (e *AISDKError) Unwrap() error {
	return e.Cause
}

// IsAISDKError checks if the given error is an AISDKError.
func IsAISDKError(err error) bool {
	var target *AISDKError
	return As(err, &target)
}

// As is a convenience wrapper around errors.As from the standard library,
// re-exported here to avoid import cycles.
func As(err error, target any) bool {
	// We use a manual type-assertion walk because importing "errors"
	// would shadow this package name.
	type unwrapper interface {
		Unwrap() error
	}
	for err != nil {
		// Check if err matches the target type.
		if tryAs(err, target) {
			return true
		}
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

// tryAs attempts a single type assertion of err into target.
func tryAs(err error, target any) bool {
	switch t := target.(type) {
	case **AISDKError:
		if e, ok := err.(*AISDKError); ok {
			*t = e
			return true
		}
	case **APICallError:
		if e, ok := err.(*APICallError); ok {
			*t = e
			return true
		}
	case **EmptyResponseBodyError:
		if e, ok := err.(*EmptyResponseBodyError); ok {
			*t = e
			return true
		}
	case **InvalidArgumentError:
		if e, ok := err.(*InvalidArgumentError); ok {
			*t = e
			return true
		}
	case **InvalidPromptError:
		if e, ok := err.(*InvalidPromptError); ok {
			*t = e
			return true
		}
	case **InvalidResponseDataError:
		if e, ok := err.(*InvalidResponseDataError); ok {
			*t = e
			return true
		}
	case **JSONParseError:
		if e, ok := err.(*JSONParseError); ok {
			*t = e
			return true
		}
	case **LoadAPIKeyError:
		if e, ok := err.(*LoadAPIKeyError); ok {
			*t = e
			return true
		}
	case **LoadSettingError:
		if e, ok := err.(*LoadSettingError); ok {
			*t = e
			return true
		}
	case **NoContentGeneratedError:
		if e, ok := err.(*NoContentGeneratedError); ok {
			*t = e
			return true
		}
	case **NoSuchModelError:
		if e, ok := err.(*NoSuchModelError); ok {
			*t = e
			return true
		}
	case **TooManyEmbeddingValuesForCallError:
		if e, ok := err.(*TooManyEmbeddingValuesForCallError); ok {
			*t = e
			return true
		}
	case **TypeValidationError:
		if e, ok := err.(*TypeValidationError); ok {
			*t = e
			return true
		}
	case **UnsupportedFunctionalityError:
		if e, ok := err.(*UnsupportedFunctionalityError); ok {
			*t = e
			return true
		}
	}
	return false
}
