// Ported from: packages/ai/src/util/merge-abort-signals.test.ts
//
// All 14 TS test cases are ported below. The TS tests use AbortSignal/AbortController;
// the Go equivalents use context.Context with context.WithCancelCause for cause
// propagation. context.Cause(ctx) is the Go equivalent of signal.reason.
//
// Test mapping (TS it() → Go Test function):
//   1.  "should return a signal that is initially not aborted"                          → TestMergeContexts_NotCancelled
//   2.  "should abort when the first signal aborts"                                     → TestMergeContexts_FirstCancels
//   3.  "should abort when the second signal aborts"                                    → TestMergeContexts_SecondCancels
//   4.  "should preserve the abort reason from the triggering signal"                   → TestMergeContexts_PreservesErrorCause
//   5.  "should preserve string abort reason"                                           → TestMergeContexts_PreservesStringCause
//   6.  "should handle already-aborted signals"                                         → TestMergeContexts_AlreadyCancelled
//   7.  "should use the first already-aborted signal reason when multiple are aborted"  → TestMergeContexts_FirstAlreadyCancelledCauseWins
//   8.  "should return undefined when no signals provided"                              → TestMergeContexts_NoContexts
//   9.  "should return undefined when only null/undefined signals provided"             → TestMergeContexts_AllNilContexts
//   10. "should filter out null and undefined signals"                                  → TestMergeContexts_NilContextsFiltered
//   11. "should return the signal directly when only one valid signal provided"         → TestMergeContexts_SingleValidAmongNils
//   12. "should use the first aborting signal reason when multiple abort simultaneously"→ TestMergeContexts_FirstCauseWinsOnSimultaneous
//   13. "should return the original signal when only one signal provided"               → TestMergeContexts_SingleContext
//   14. "should work with many signals"                                                 → TestMergeContexts_ManyContexts
package util

import (
	"context"
	"errors"
	"testing"
	"time"
)

// awaitDone waits for ctx.Done() with a timeout. Returns true if done, false if timed out.
func awaitDone(ctx context.Context, timeout time.Duration) bool {
	select {
	case <-ctx.Done():
		return true
	case <-time.After(timeout):
		return false
	}
}

// TS: it('should return a signal that is initially not aborted')
func TestMergeContexts_NotCancelled(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	merged := MergeContexts(ctx1, ctx2)

	select {
	case <-merged.Done():
		t.Fatal("merged context should not be cancelled")
	default:
		// OK
	}
}

// TS: it('should abort when the first signal aborts')
func TestMergeContexts_FirstCancels(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	merged := MergeContexts(ctx1, ctx2)
	cancel1()

	if !awaitDone(merged, time.Second) {
		t.Fatal("expected merged context to be cancelled")
	}
}

// TS: it('should abort when the second signal aborts')
func TestMergeContexts_SecondCancels(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())

	merged := MergeContexts(ctx1, ctx2)
	cancel2()

	if !awaitDone(merged, time.Second) {
		t.Fatal("expected merged context to be cancelled")
	}
}

// TS: it('should preserve the abort reason from the triggering signal')
// In TS: controller1.abort(new Error('custom abort reason')) → merged.reason === that error.
// In Go: context.WithCancelCause + context.Cause() is the equivalent.
func TestMergeContexts_PreservesErrorCause(t *testing.T) {
	reason := errors.New("custom abort reason")
	ctx1, cancel1 := context.WithCancelCause(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	merged := MergeContexts(ctx1, ctx2)
	cancel1(reason)

	if !awaitDone(merged, time.Second) {
		t.Fatal("expected merged context to be cancelled")
	}

	got := context.Cause(merged)
	if !errors.Is(got, reason) {
		t.Fatalf("expected cause %v, got %v", reason, got)
	}
}

// TS: it('should preserve string abort reason')
// Go doesn't have string causes directly — we use errors.New() which is
// the idiomatic equivalent. The test verifies cause propagation works
// regardless of the error type.
func TestMergeContexts_PreservesStringCause(t *testing.T) {
	reason := errors.New("string reason")
	ctx1, cancel1 := context.WithCancelCause(context.Background())

	merged := MergeContexts(ctx1)

	// Single context → returns it directly, so cause is on the original.
	cancel1(reason)

	if !awaitDone(merged, time.Second) {
		t.Fatal("expected merged context to be cancelled")
	}

	got := context.Cause(merged)
	if !errors.Is(got, reason) {
		t.Fatalf("expected cause %v, got %v", reason, got)
	}
}

// TS: it('should handle already-aborted signals')
func TestMergeContexts_AlreadyCancelled(t *testing.T) {
	reason := errors.New("already aborted")
	ctx1, cancel1 := context.WithCancelCause(context.Background())
	cancel1(reason) // already cancelled before merge

	merged := MergeContexts(ctx1)

	// Single context path returns it directly.
	if !awaitDone(merged, time.Second) {
		t.Fatal("expected merged context to be cancelled immediately")
	}

	got := context.Cause(merged)
	if !errors.Is(got, reason) {
		t.Fatalf("expected cause %v, got %v", reason, got)
	}
}

// TS: it('should use the first already-aborted signal reason when multiple are aborted')
// When multiple inputs are already cancelled, the first one's cause should win.
func TestMergeContexts_FirstAlreadyCancelledCauseWins(t *testing.T) {
	reason1 := errors.New("first reason")
	reason2 := errors.New("second reason")

	ctx1, cancel1 := context.WithCancelCause(context.Background())
	ctx2, cancel2 := context.WithCancelCause(context.Background())

	cancel1(reason1)
	cancel2(reason2)

	merged := MergeContexts(ctx1, ctx2)

	if !awaitDone(merged, time.Second) {
		t.Fatal("expected merged context to be cancelled immediately")
	}

	got := context.Cause(merged)
	// The first context's AfterFunc callback fires first since it was registered
	// first, so its cause should win. This matches the TS behavior.
	if !errors.Is(got, reason1) {
		t.Fatalf("expected cause %v, got %v", reason1, got)
	}
}

// TS: it('should return undefined when no signals provided')
// Go equivalent: returns context.Background() (never cancelled).
func TestMergeContexts_NoContexts(t *testing.T) {
	merged := MergeContexts()
	if merged == nil {
		t.Fatal("expected non-nil context")
	}
	select {
	case <-merged.Done():
		t.Fatal("background context should not be done")
	default:
		// OK
	}
}

// TS: it('should return undefined when only null/undefined signals provided')
// Go equivalent: all nil inputs → returns context.Background().
func TestMergeContexts_AllNilContexts(t *testing.T) {
	merged := MergeContexts(nil, nil, nil)
	if merged == nil {
		t.Fatal("expected non-nil context")
	}
	select {
	case <-merged.Done():
		t.Fatal("background context should not be done")
	default:
		// OK
	}
}

// TS: it('should filter out null and undefined signals')
func TestMergeContexts_NilContextsFiltered(t *testing.T) {
	reason := errors.New("abort reason")
	ctx1, cancel1 := context.WithCancelCause(context.Background())

	merged := MergeContexts(nil, ctx1, nil)

	// Single valid context → returned directly.
	select {
	case <-merged.Done():
		t.Fatal("should not be cancelled")
	default:
		// OK
	}

	cancel1(reason)

	if !awaitDone(merged, time.Second) {
		t.Fatal("expected merged context to be cancelled")
	}

	got := context.Cause(merged)
	if !errors.Is(got, reason) {
		t.Fatalf("expected cause %v, got %v", reason, got)
	}
}

// TS: it('should return the signal directly when only one valid signal provided')
func TestMergeContexts_SingleValidAmongNils(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	merged := MergeContexts(nil, ctx, nil)

	// Should return the same context (no wrapper).
	if merged != ctx {
		t.Fatal("expected same context for single valid input among nils")
	}
}

// TS: it('should use the first aborting signal reason when multiple abort simultaneously')
//
// Divergence note: In TS (single-threaded), the first registered listener's
// reason always wins. In Go, AfterFunc callbacks for concurrent cancellations
// run in separate goroutines with no ordering guarantee. This test verifies
// that SOME cause is propagated, not which specific one wins.
func TestMergeContexts_FirstCauseWinsOnSimultaneous(t *testing.T) {
	reason1 := errors.New("first reason")
	reason2 := errors.New("second reason")

	ctx1, cancel1 := context.WithCancelCause(context.Background())
	ctx2, cancel2 := context.WithCancelCause(context.Background())

	merged := MergeContexts(ctx1, ctx2)

	// Both cancel concurrently. In Go, either cause may win.
	cancel1(reason1)
	cancel2(reason2)

	if !awaitDone(merged, time.Second) {
		t.Fatal("expected merged context to be cancelled")
	}

	got := context.Cause(merged)
	if !errors.Is(got, reason1) && !errors.Is(got, reason2) {
		t.Fatalf("expected cause to be one of [%v, %v], got %v", reason1, reason2, got)
	}
}

// TS: it('should return the original signal when only one signal provided')
func TestMergeContexts_SingleContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	merged := MergeContexts(ctx)
	if merged != ctx {
		t.Fatal("expected same context for single input")
	}
}

// TS: it('should work with many signals')
func TestMergeContexts_ManyContexts(t *testing.T) {
	const n = 10
	contexts := make([]context.Context, n)
	cancels := make([]context.CancelCauseFunc, n)
	for i := range contexts {
		contexts[i], cancels[i] = context.WithCancelCause(context.Background())
		defer cancels[i](nil)
	}

	merged := MergeContexts(contexts...)

	select {
	case <-merged.Done():
		t.Fatal("should not be cancelled")
	default:
		// OK
	}

	reason := errors.New("signal 5 reason")
	cancels[5](reason)

	if !awaitDone(merged, time.Second) {
		t.Fatal("expected merged context to be cancelled")
	}

	got := context.Cause(merged)
	if !errors.Is(got, reason) {
		t.Fatalf("expected cause %v, got %v", reason, got)
	}
}
