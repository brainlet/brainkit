// Ported from: packages/ai/src/test/not-implemented.ts
package testutil

import "fmt"

// ErrNotImplemented is the error returned by NotImplemented.
var ErrNotImplemented = fmt.Errorf("not implemented")

// NotImplemented panics with "not implemented". Used as the default
// for mock function fields that have not been configured.
func NotImplemented() {
	panic("not implemented")
}
