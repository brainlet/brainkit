// Ported from: packages/ai/src/util/now.ts
package util

import "time"

// Now returns the current time as milliseconds since the Unix epoch.
// This is the Go equivalent of performance.now() / Date.now().
func Now() float64 {
	return float64(time.Now().UnixNano()) / float64(time.Millisecond)
}
