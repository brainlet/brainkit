// Ported from: packages/ai/src/telemetry/telemetry-integration.ts
package telemetry

// Listener is a function that handles an event and optionally returns an error.
// In TypeScript this is (event: EVENT) => PromiseLike<void> | void.
// In Go, we use func(event interface{}) error.
// TODO: import from brainlink/experiments/ai-kit/util once it exists
type Listener = func(event interface{}) error

// TelemetryIntegration defines the interface for custom telemetry integrations.
// Methods can be nil if the integration does not implement that lifecycle hook.
//
// In TypeScript these reference typed events from generate-text/callback-events.
// TODO: use proper typed events once those packages exist
type TelemetryIntegration struct {
	OnStart          Listener
	OnStepStart      Listener
	OnToolCallStart  Listener
	OnToolCallFinish Listener
	OnStepFinish     Listener
	OnFinish         Listener
}
