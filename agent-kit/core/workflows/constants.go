// Ported from: packages/core/src/workflows/constants.ts
package workflows

// In the TypeScript source, symbols are used to inject pubsub and stream format
// into the step context without exposing them in the public API.
// In Go, we use unexported context keys instead of JS Symbols.

// pubsubContextKey is an unexported context key used to inject pubsub into step context
// without exposing it in the public API.
// Steps can access pubsub via this key for internal event publishing.
//
// TS equivalent: export const PUBSUB_SYMBOL = Symbol('pubsub');
type pubsubContextKey struct{}

// streamFormatContextKey is an unexported context key used to pass stream format
// preferences through step context.
//
// TS equivalent: export const STREAM_FORMAT_SYMBOL = Symbol('stream_format');
type streamFormatContextKey struct{}

// nestedWorkflowResultKey is an unexported context key used to identify results
// from nested workflow execution.
//
// When a workflow contains another workflow as a step, the inner workflow's Execute()
// returns a result wrapped with this key. The step handler checks for this key to
// detect nested workflow results and handle them specially - extracting the actual
// result and nested runId for proper state management.
//
// This key is safe to use (unlike PendingMarker) because it stays in-memory within
// a single execution context - it's never serialized to storage or passed between
// distributed engine instances.
//
// TS equivalent: export const NESTED_WORKFLOW_RESULT_SYMBOL = Symbol('nested_workflow_result');
type nestedWorkflowResultKey struct{}

// NestedWorkflowResult wraps a result from nested workflow execution.
// Used internally to detect and unwrap nested workflow results.
type NestedWorkflowResult struct {
	Result    any
	NestedRun string
}
