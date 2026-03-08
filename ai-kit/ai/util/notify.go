// Ported from: packages/ai/src/util/notify.ts
package util

// Listener is a callback function that can be used to notify listeners.
type Listener[EVENT any] func(event EVENT) error

// Notify calls all provided callbacks with the given event.
// Errors in callbacks do not break the flow.
func Notify[EVENT any](event EVENT, callbacks ...Listener[EVENT]) {
	for _, cb := range callbacks {
		if cb == nil {
			continue
		}
		_ = cb(event) // errors are silently ignored, matching TS behavior
	}
}
