// Ported from: packages/ai/src/util/merge-abort-signals.ts
//
// Go idiom translation:
//   TS AbortSignal        → context.Context
//   TS AbortController    → context.WithCancelCause / context.CancelCauseFunc
//   TS signal.reason      → context.Cause(ctx)
//   TS signal.aborted     → ctx.Err() != nil
//   TS null/undefined     → nil (filtered out)
//
// The TS version uses AbortController + signal.reason to propagate *why* a
// signal was aborted. The Go equivalent is context.WithCancelCause (Go 1.20+)
// paired with context.Cause(). We use context.AfterFunc (Go 1.21+) to watch
// each input context without spawning goroutines.
//
// Sync note: if the TS API changes to accept/return different types, this
// function's signature should stay as (contexts ...context.Context) context.Context
// since that is the idiomatic Go equivalent. The cause propagation via
// context.WithCancelCause is internal — callers retrieve it with context.Cause().
package util

import "context"

// MergeContexts merges multiple contexts into a single context.
// The returned context is cancelled when ANY input context is cancelled.
// The cancellation cause from the triggering context is propagated and
// can be retrieved with context.Cause(merged).
//
// Behavior by input count:
//   - 0 valid contexts → returns context.Background() (never cancelled)
//   - 1 valid context  → returns that context directly (no wrapper)
//   - 2+ valid contexts → returns a new context cancelled by any input
//
// Nil contexts are silently filtered out.
func MergeContexts(contexts ...context.Context) context.Context {
	var valid []context.Context
	for _, ctx := range contexts {
		if ctx != nil {
			valid = append(valid, ctx)
		}
	}

	switch len(valid) {
	case 0:
		return context.Background()
	case 1:
		return valid[0]
	default:
		return mergeWithCause(valid)
	}
}

// mergeWithCause creates a derived context that cancels when any input cancels,
// propagating the cause from the first context that triggers cancellation.
//
// Implementation notes for future sync:
//   - context.WithCancelCause gives us cancel(cause) so context.Cause(merged)
//     returns the reason, matching TS signal.reason semantics.
//   - context.AfterFunc registers a callback on each input context's Done channel
//     without spawning a goroutine per context (more efficient than select loops).
//   - cancel is idempotent: if multiple inputs cancel simultaneously, the first
//     one's cause wins — matching the TS behavior where the first listener fires.
//   - The _ = cancel line satisfies go vet's "cancel not used on all paths" check.
//     AfterFunc closures capture and call cancel, but vet cannot see into closures.
//
// Ordering guarantee for already-cancelled contexts:
//   In TS (single-threaded), AbortSignal listeners fire synchronously in
//   registration order, so the first already-aborted signal's reason always wins.
//   In Go, AfterFunc callbacks for already-done contexts fire in goroutines with
//   NO ordering guarantee. To match TS semantics, we check for already-cancelled
//   contexts synchronously BEFORE registering AfterFunc watchers. This ensures
//   the first already-cancelled context in slice order determines the cause.
func mergeWithCause(valid []context.Context) context.Context {
	merged, cancel := context.WithCancelCause(context.Background())

	// Synchronous pre-check: if any input is already cancelled, use the first
	// one's cause immediately. This guarantees deterministic ordering for
	// already-cancelled contexts, matching TS single-threaded semantics.
	for _, ctx := range valid {
		if ctx.Err() != nil {
			cancel(context.Cause(ctx))
			return merged
		}
	}

	// No inputs are cancelled yet. Register watchers for future cancellation.
	// When any input cancels, its AfterFunc callback fires and propagates the
	// cause. cancel is idempotent: if multiple inputs cancel concurrently,
	// whichever AfterFunc goroutine runs first determines the cause.
	for _, ctx := range valid {
		parent := ctx // capture loop variable for closure
		context.AfterFunc(parent, func() {
			cancel(context.Cause(parent))
		})
	}

	// Satisfy go vet: cancel is called by AfterFunc closures above, but static
	// analysis cannot see into closure captures. This reference silences the
	// "cancel not used on all paths" diagnostic.
	_ = cancel

	return merged
}
