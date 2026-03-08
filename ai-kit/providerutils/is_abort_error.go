// Ported from: packages/provider-utils/src/is-abort-error.ts
package providerutils

import "context"

// IsAbortError checks whether the given error represents a cancellation/abort.
// In Go this maps to context.Canceled and context.DeadlineExceeded.
func IsAbortError(err error) bool {
	if err == nil {
		return false
	}
	return err == context.Canceled || err == context.DeadlineExceeded
}
