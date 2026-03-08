// Ported from: packages/core/src/loop/network/validation.test.ts
//
// NOTE: These tests compile but will panic at runtime due to a pre-existing
// bug in run_command_tool.go line 34: regexp.MustCompile(`\\(?![ ])`) uses
// a Perl-style negative lookahead (?!...) which Go's regexp package does
// not support. The MustCompile call panics during package init(), before
// any test code runs. Fix the regex in run_command_tool.go to unblock
// these tests (e.g. replace with `\\[^ ]`).
package network

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Mock scorer
// ---------------------------------------------------------------------------

// mockScorer implements MastraScorer for testing.
type mockScorer struct {
	id      string
	name    string
	runFn   func(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error)
	callMu  sync.Mutex
	called  int
}

func (m *mockScorer) ID() string   { return m.id }
func (m *mockScorer) Name() string { return m.name }
func (m *mockScorer) Run(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error) {
	m.callMu.Lock()
	m.called++
	m.callMu.Unlock()
	return m.runFn(ctx, input)
}
func (m *mockScorer) CallCount() int {
	m.callMu.Lock()
	defer m.callMu.Unlock()
	return m.called
}

// createMockScorer creates a mock scorer that returns the given score and reason.
func createMockScorer(id string, score float64, reason string) *mockScorer {
	return &mockScorer{
		id:   id,
		name: fmt.Sprintf("%s Scorer", id),
		runFn: func(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error) {
			return &ScorerRunResult{Score: score, Reason: reason}, nil
		},
	}
}

// createDelayedMockScorer creates a mock scorer with a delay before returning.
func createDelayedMockScorer(id string, score float64, reason string, delay time.Duration) *mockScorer {
	return &mockScorer{
		id:   id,
		name: fmt.Sprintf("%s Scorer", id),
		runFn: func(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error) {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return &ScorerRunResult{Score: score, Reason: reason}, nil
		},
	}
}

// createErrorScorer creates a mock scorer that returns an error.
func createErrorScorer(id string, errMsg string) *mockScorer {
	return &mockScorer{
		id:   id,
		name: fmt.Sprintf("%s Scorer", id),
		runFn: func(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error) {
			return nil, fmt.Errorf("%s", errMsg)
		},
	}
}

// ---------------------------------------------------------------------------
// Helper: create mock context
// ---------------------------------------------------------------------------

func createMockContext(overrides ...func(*CompletionContext)) CompletionContext {
	ctx := CompletionContext{
		Iteration:     1,
		MaxIterations: 10,
		Messages:      []MastraDBMessage{},
		OriginalTask:  "Test task",
		SelectedPrimitive: SelectedPrimitiveInfo{
			ID:   "test-agent",
			Type: "agent",
		},
		PrimitivePrompt: "Do something",
		PrimitiveResult: "Done",
		NetworkName:     "test-network",
		RunID:           "test-run-id",
	}
	for _, fn := range overrides {
		fn(&ctx)
	}
	return ctx
}

func createMockStreamContext(overrides ...func(*StreamCompletionContext)) StreamCompletionContext {
	ctx := StreamCompletionContext{
		Iteration:     1,
		MaxIterations: 10,
		OriginalTask:  "Test task",
		CurrentText:   "Current output text",
		ToolCalls:     []ToolCallInfo{},
		ToolResults:   []ToolResultInfo{},
		RunID:         "test-run-id",
		Messages:      []MastraDBMessage{},
	}
	for _, fn := range overrides {
		fn(&ctx)
	}
	return ctx
}

// ---------------------------------------------------------------------------
// runCompletionScorers tests
// ---------------------------------------------------------------------------

func TestRunCompletionScorers(t *testing.T) {
	t.Run("strategy all default", func(t *testing.T) {
		t.Run("returns complete when all scorers pass", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 1, "Passed")
			scorer2 := createMockScorer("scorer-2", 1, "Passed")
			compCtx := createMockContext()

			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				compCtx,
				nil,
			)

			if !result.Complete {
				t.Error("expected complete to be true")
			}
			if len(result.Scorers) != 2 {
				t.Errorf("expected 2 scorers, got %d", len(result.Scorers))
			}
			for _, s := range result.Scorers {
				if !s.Passed {
					t.Errorf("expected scorer %s to pass", s.ScorerID)
				}
			}
			if scorer1.CallCount() != 1 {
				t.Errorf("expected scorer-1 to be called once, got %d", scorer1.CallCount())
			}
			if scorer2.CallCount() != 1 {
				t.Errorf("expected scorer-2 to be called once, got %d", scorer2.CallCount())
			}
		})

		t.Run("returns incomplete when any scorer fails", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 1, "Passed")
			scorer2 := createMockScorer("scorer-2", 0, "Failed")
			compCtx := createMockContext()

			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				compCtx,
				nil,
			)

			if result.Complete {
				t.Error("expected complete to be false")
			}
			hasFailed := false
			for _, s := range result.Scorers {
				if !s.Passed {
					hasFailed = true
					break
				}
			}
			if !hasFailed {
				t.Error("expected at least one scorer to fail")
			}
		})

		t.Run("returns incomplete when all scorers fail", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 0, "Failed")
			scorer2 := createMockScorer("scorer-2", 0, "Failed")
			compCtx := createMockContext()

			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				compCtx,
				nil,
			)

			if result.Complete {
				t.Error("expected complete to be false")
			}
			for _, s := range result.Scorers {
				if s.Passed {
					t.Errorf("expected scorer %s to fail", s.ScorerID)
				}
			}
		})
	})

	t.Run("strategy any", func(t *testing.T) {
		t.Run("returns complete when at least one scorer passes", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 0, "Failed")
			scorer2 := createMockScorer("scorer-2", 1, "Passed")
			compCtx := createMockContext()

			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				compCtx,
				&CompletionScorerOptions{Strategy: "any", Parallel: true},
			)

			if !result.Complete {
				t.Error("expected complete to be true")
			}
		})

		t.Run("returns incomplete when all scorers fail", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 0, "Failed")
			scorer2 := createMockScorer("scorer-2", 0, "Failed")
			compCtx := createMockContext()

			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				compCtx,
				&CompletionScorerOptions{Strategy: "any", Parallel: true},
			)

			if result.Complete {
				t.Error("expected complete to be false")
			}
		})
	})

	t.Run("error handling", func(t *testing.T) {
		t.Run("handles scorer that throws an error", func(t *testing.T) {
			errorScorer := createErrorScorer("error-scorer", "Scorer crashed")
			compCtx := createMockContext()

			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{errorScorer},
				compCtx,
				nil,
			)

			if result.Complete {
				t.Error("expected complete to be false")
			}
			if result.Scorers[0].Passed {
				t.Error("expected scorer to fail")
			}
			if !strings.Contains(result.Scorers[0].Reason, "Scorer threw an error") {
				t.Errorf("expected reason to contain 'Scorer threw an error', got %q", result.Scorers[0].Reason)
			}
			if !strings.Contains(result.Scorers[0].Reason, "Scorer crashed") {
				t.Errorf("expected reason to contain 'Scorer crashed', got %q", result.Scorers[0].Reason)
			}
		})
	})

	t.Run("sequential execution", func(t *testing.T) {
		t.Run("runs scorers sequentially when parallel false", func(t *testing.T) {
			var mu sync.Mutex
			executionOrder := []string{}

			scorer1 := &mockScorer{
				id:   "scorer-1",
				name: "Scorer 1",
				runFn: func(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error) {
					mu.Lock()
					executionOrder = append(executionOrder, "scorer-1-start")
					mu.Unlock()
					time.Sleep(10 * time.Millisecond)
					mu.Lock()
					executionOrder = append(executionOrder, "scorer-1-end")
					mu.Unlock()
					return &ScorerRunResult{Score: 1}, nil
				},
			}
			scorer2 := &mockScorer{
				id:   "scorer-2",
				name: "Scorer 2",
				runFn: func(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error) {
					mu.Lock()
					executionOrder = append(executionOrder, "scorer-2-start")
					mu.Unlock()
					time.Sleep(10 * time.Millisecond)
					mu.Lock()
					executionOrder = append(executionOrder, "scorer-2-end")
					mu.Unlock()
					return &ScorerRunResult{Score: 1}, nil
				},
			}
			compCtx := createMockContext()

			RunCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				compCtx,
				&CompletionScorerOptions{Parallel: false},
			)

			mu.Lock()
			defer mu.Unlock()
			expected := []string{"scorer-1-start", "scorer-1-end", "scorer-2-start", "scorer-2-end"}
			if len(executionOrder) != len(expected) {
				t.Fatalf("expected %d execution events, got %d: %v", len(expected), len(executionOrder), executionOrder)
			}
			for i, e := range expected {
				if executionOrder[i] != e {
					t.Errorf("execution order[%d]: expected %q, got %q (full: %v)", i, e, executionOrder[i], executionOrder)
				}
			}
		})

		t.Run("short-circuits on failure with all strategy", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 0, "Failed")
			scorer2 := createMockScorer("scorer-2", 1, "Passed")
			compCtx := createMockContext()

			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				compCtx,
				&CompletionScorerOptions{Parallel: false, Strategy: "all"},
			)

			if result.Complete {
				t.Error("expected complete to be false")
			}
			if scorer1.CallCount() != 1 {
				t.Errorf("expected scorer-1 to be called once, got %d", scorer1.CallCount())
			}
			if scorer2.CallCount() != 0 {
				t.Errorf("expected scorer-2 to not be called, got %d", scorer2.CallCount())
			}
		})

		t.Run("short-circuits on success with any strategy", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 1, "Passed")
			scorer2 := createMockScorer("scorer-2", 0, "Failed")
			compCtx := createMockContext()

			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				compCtx,
				&CompletionScorerOptions{Parallel: false, Strategy: "any"},
			)

			if !result.Complete {
				t.Error("expected complete to be true")
			}
			if scorer1.CallCount() != 1 {
				t.Errorf("expected scorer-1 to be called once, got %d", scorer1.CallCount())
			}
			if scorer2.CallCount() != 0 {
				t.Errorf("expected scorer-2 to not be called, got %d", scorer2.CallCount())
			}
		})
	})

	t.Run("context passing", func(t *testing.T) {
		t.Run("passes context to scorers correctly", func(t *testing.T) {
			var capturedInput ScorerRunInput
			scorer := &mockScorer{
				id:   "scorer-1",
				name: "scorer-1 Scorer",
				runFn: func(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error) {
					capturedInput = input
					return &ScorerRunResult{Score: 1}, nil
				},
			}

			compCtx := createMockContext(func(c *CompletionContext) {
				c.OriginalTask = "Custom task"
				c.PrimitiveResult = "Custom result"
				c.RunID = "custom-run-id"
			})

			RunCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer},
				compCtx,
				nil,
			)

			if capturedInput.RunID != "custom-run-id" {
				t.Errorf("expected runId 'custom-run-id', got %q", capturedInput.RunID)
			}

			// The input is the CompletionContext struct
			inputCtx, ok := capturedInput.Input.(CompletionContext)
			if !ok {
				t.Fatalf("expected input to be CompletionContext, got %T", capturedInput.Input)
			}
			if inputCtx.OriginalTask != "Custom task" {
				t.Errorf("expected originalTask 'Custom task', got %q", inputCtx.OriginalTask)
			}
			if inputCtx.PrimitiveResult != "Custom result" {
				t.Errorf("expected primitiveResult 'Custom result', got %q", inputCtx.PrimitiveResult)
			}

			// Output is the primitiveResult
			outputStr, ok := capturedInput.Output.(string)
			if !ok {
				t.Fatalf("expected output to be string, got %T", capturedInput.Output)
			}
			if outputStr != "Custom result" {
				t.Errorf("expected output 'Custom result', got %q", outputStr)
			}
		})
	})

	t.Run("result structure", func(t *testing.T) {
		t.Run("returns correct result structure", func(t *testing.T) {
			scorer := createMockScorer("test-scorer", 1, "Test reason")
			compCtx := createMockContext()

			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer},
				compCtx,
				nil,
			)

			if !result.Complete {
				t.Error("expected complete to be true")
			}
			if result.CompletionReason != "Test reason" {
				t.Errorf("expected completionReason 'Test reason', got %q", result.CompletionReason)
			}
			if result.TimedOut {
				t.Error("expected timedOut to be false")
			}
			if result.TotalDuration < 0 {
				t.Errorf("expected non-negative totalDuration, got %d", result.TotalDuration)
			}

			if len(result.Scorers) != 1 {
				t.Fatalf("expected 1 scorer result, got %d", len(result.Scorers))
			}
			s := result.Scorers[0]
			if s.Score != 1 {
				t.Errorf("expected score 1, got %g", s.Score)
			}
			if !s.Passed {
				t.Error("expected passed to be true")
			}
			if s.Reason != "Test reason" {
				t.Errorf("expected reason 'Test reason', got %q", s.Reason)
			}
			if s.ScorerID != "test-scorer" {
				t.Errorf("expected scorerId 'test-scorer', got %q", s.ScorerID)
			}
			if s.ScorerName != "test-scorer Scorer" {
				t.Errorf("expected scorerName 'test-scorer Scorer', got %q", s.ScorerName)
			}
			if s.Duration < 0 {
				t.Errorf("expected non-negative duration, got %d", s.Duration)
			}
		})
	})

	t.Run("empty scorers", func(t *testing.T) {
		t.Run("returns complete with empty scorers array and all strategy", func(t *testing.T) {
			compCtx := createMockContext()
			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{},
				compCtx,
				&CompletionScorerOptions{Strategy: "all", Parallel: true},
			)

			// Empty array with 'all' strategy: vacuously true (all of nothing passed)
			if !result.Complete {
				t.Error("expected complete to be true for empty scorers with 'all' strategy")
			}
			if len(result.Scorers) != 0 {
				t.Errorf("expected 0 scorers, got %d", len(result.Scorers))
			}
		})

		t.Run("returns incomplete with empty scorers array and any strategy", func(t *testing.T) {
			compCtx := createMockContext()
			result := RunCompletionScorers(
				context.Background(),
				[]MastraScorer{},
				compCtx,
				&CompletionScorerOptions{Strategy: "any", Parallel: true},
			)

			// Empty array with 'any' strategy: false (none passed)
			if result.Complete {
				t.Error("expected complete to be false for empty scorers with 'any' strategy")
			}
			if len(result.Scorers) != 0 {
				t.Errorf("expected 0 scorers, got %d", len(result.Scorers))
			}
		})
	})
}

// ---------------------------------------------------------------------------
// formatCompletionFeedback tests
// ---------------------------------------------------------------------------

func TestFormatCompletionFeedback(t *testing.T) {
	t.Run("formats complete result", func(t *testing.T) {
		result := CompletionRunResult{
			Complete:         true,
			CompletionReason: "All checks passed",
			Scorers: []ScorerResult{
				{
					Score:      1,
					Passed:     true,
					Reason:     "Test passed",
					ScorerID:   "test-scorer",
					ScorerName: "Test Scorer",
					Duration:   100,
				},
			},
			TotalDuration: 150,
			TimedOut:      false,
		}

		feedback := FormatCompletionFeedback(result, false)

		assertContains(t, feedback, "#### Completion Check Results")
		assertContains(t, feedback, "COMPLETE")
		assertContains(t, feedback, "Duration: 150ms")
		assertContains(t, feedback, "Test Scorer (test-scorer)")
		assertContains(t, feedback, "Score: 1")
		assertContains(t, feedback, "Reason: Test passed")
		assertNotContains(t, feedback, "timed out")
	})

	t.Run("formats incomplete result", func(t *testing.T) {
		result := CompletionRunResult{
			Complete:         false,
			CompletionReason: "Check failed",
			Scorers: []ScorerResult{
				{
					Score:      0,
					Passed:     false,
					Reason:     "Test failed",
					ScorerID:   "test-scorer",
					ScorerName: "Test Scorer",
					Duration:   100,
				},
			},
			TotalDuration: 150,
			TimedOut:      false,
		}

		feedback := FormatCompletionFeedback(result, false)

		assertContains(t, feedback, "NOT COMPLETE")
		assertContains(t, feedback, "Score: 0")
		assertContains(t, feedback, "Reason: Test failed")
		assertContains(t, feedback, "Will continue working on the task.")
	})

	t.Run("formats max iterations reached result", func(t *testing.T) {
		result := CompletionRunResult{
			Complete:         false,
			CompletionReason: "Check failed",
			Scorers: []ScorerResult{
				{
					Score:      0,
					Passed:     false,
					Reason:     "Test failed",
					ScorerID:   "test-scorer",
					ScorerName: "Test Scorer",
					Duration:   100,
				},
			},
			TotalDuration: 150,
			TimedOut:      false,
		}

		feedback := FormatCompletionFeedback(result, true)

		assertContains(t, feedback, "NOT COMPLETE")
		assertContains(t, feedback, "Score: 0")
		assertContains(t, feedback, "Reason: Test failed")
		assertContains(t, feedback, "Max iterations reached.")
	})

	t.Run("formats timeout indication", func(t *testing.T) {
		result := CompletionRunResult{
			Complete:      false,
			Scorers:       []ScorerResult{},
			TotalDuration: 600000,
			TimedOut:      true,
		}

		feedback := FormatCompletionFeedback(result, false)

		assertContains(t, feedback, "Scoring timed out")
	})

	t.Run("formats multiple scorers", func(t *testing.T) {
		result := CompletionRunResult{
			Complete: false,
			Scorers: []ScorerResult{
				{
					Score:      1,
					Passed:     true,
					Reason:     "First passed",
					ScorerID:   "scorer-1",
					ScorerName: "Scorer One",
					Duration:   50,
				},
				{
					Score:      0,
					Passed:     false,
					Reason:     "Second failed",
					ScorerID:   "scorer-2",
					ScorerName: "Scorer Two",
					Duration:   75,
				},
			},
			TotalDuration: 125,
			TimedOut:      false,
		}

		feedback := FormatCompletionFeedback(result, false)

		assertContains(t, feedback, "Scorer One (scorer-1)")
		assertContains(t, feedback, "Scorer Two (scorer-2)")
		assertContains(t, feedback, "First passed")
		assertContains(t, feedback, "Second failed")
	})

	t.Run("handles scorer without reason", func(t *testing.T) {
		result := CompletionRunResult{
			Complete: true,
			Scorers: []ScorerResult{
				{
					Score:      1,
					Passed:     true,
					ScorerID:   "test-scorer",
					ScorerName: "Test Scorer",
					Duration:   100,
				},
			},
			TotalDuration: 100,
			TimedOut:      false,
		}

		feedback := FormatCompletionFeedback(result, false)

		assertContains(t, feedback, "Score: 1")
		// Should not have "Reason:" line since no reason provided
		assertNotContains(t, feedback, "Reason:")
	})
}

// ---------------------------------------------------------------------------
// runStreamCompletionScorers tests
// ---------------------------------------------------------------------------

func TestRunStreamCompletionScorers(t *testing.T) {
	t.Run("strategy all default", func(t *testing.T) {
		t.Run("returns complete when all scorers pass", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 1, "Passed")
			scorer2 := createMockScorer("scorer-2", 1, "Passed")
			sCtx := createMockStreamContext()

			result := RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				sCtx,
				nil,
			)

			if !result.Complete {
				t.Error("expected complete to be true")
			}
			if len(result.Scorers) != 2 {
				t.Errorf("expected 2 scorers, got %d", len(result.Scorers))
			}
			for _, s := range result.Scorers {
				if !s.Passed {
					t.Errorf("expected scorer %s to pass", s.ScorerID)
				}
			}
		})

		t.Run("returns incomplete when any scorer fails", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 1, "Passed")
			scorer2 := createMockScorer("scorer-2", 0, "Failed")
			sCtx := createMockStreamContext()

			result := RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				sCtx,
				nil,
			)

			if result.Complete {
				t.Error("expected complete to be false")
			}
			hasFailed := false
			for _, s := range result.Scorers {
				if !s.Passed {
					hasFailed = true
					break
				}
			}
			if !hasFailed {
				t.Error("expected at least one scorer to fail")
			}
		})

		t.Run("returns incomplete when all scorers fail", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 0, "Failed")
			scorer2 := createMockScorer("scorer-2", 0, "Failed")
			sCtx := createMockStreamContext()

			result := RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				sCtx,
				nil,
			)

			if result.Complete {
				t.Error("expected complete to be false")
			}
			for _, s := range result.Scorers {
				if s.Passed {
					t.Errorf("expected scorer %s to fail", s.ScorerID)
				}
			}
		})
	})

	t.Run("strategy any", func(t *testing.T) {
		t.Run("returns complete when at least one scorer passes", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 0, "Failed")
			scorer2 := createMockScorer("scorer-2", 1, "Passed")
			sCtx := createMockStreamContext()

			result := RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				sCtx,
				&CompletionScorerOptions{Strategy: "any", Parallel: true},
			)

			if !result.Complete {
				t.Error("expected complete to be true")
			}
		})

		t.Run("returns incomplete when all scorers fail", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 0, "Failed")
			scorer2 := createMockScorer("scorer-2", 0, "Failed")
			sCtx := createMockStreamContext()

			result := RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				sCtx,
				&CompletionScorerOptions{Strategy: "any", Parallel: true},
			)

			if result.Complete {
				t.Error("expected complete to be false")
			}
		})
	})

	t.Run("context adaptation", func(t *testing.T) {
		t.Run("adapts stream context to completion context for scorers", func(t *testing.T) {
			var capturedInput ScorerRunInput
			scorer := &mockScorer{
				id:   "scorer-1",
				name: "scorer-1 Scorer",
				runFn: func(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error) {
					capturedInput = input
					return &ScorerRunResult{Score: 1}, nil
				},
			}

			sCtx := createMockStreamContext(func(c *StreamCompletionContext) {
				c.OriginalTask = "Custom stream task"
				c.CurrentText = "Stream output text"
				c.RunID = "stream-run-id"
				c.AgentID = "my-agent"
				c.AgentName = "My Agent"
				c.ToolCalls = []ToolCallInfo{{Name: "fetchData", Args: map[string]any{"url": "https://example.com"}}}
				c.ToolResults = []ToolResultInfo{{Name: "fetchData", Result: map[string]any{"data": "test"}}}
			})

			RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer},
				sCtx,
				nil,
			)

			if capturedInput.RunID != "stream-run-id" {
				t.Errorf("expected runId 'stream-run-id', got %q", capturedInput.RunID)
			}

			inputCtx, ok := capturedInput.Input.(CompletionContext)
			if !ok {
				t.Fatalf("expected input to be CompletionContext, got %T", capturedInput.Input)
			}
			if inputCtx.OriginalTask != "Custom stream task" {
				t.Errorf("expected originalTask 'Custom stream task', got %q", inputCtx.OriginalTask)
			}
			if inputCtx.PrimitiveResult != "Stream output text" {
				t.Errorf("expected primitiveResult 'Stream output text', got %q", inputCtx.PrimitiveResult)
			}
			if inputCtx.SelectedPrimitive.ID != "stream" {
				t.Errorf("expected selectedPrimitive.ID 'stream', got %q", inputCtx.SelectedPrimitive.ID)
			}
			if inputCtx.SelectedPrimitive.Type != "agent" {
				t.Errorf("expected selectedPrimitive.Type 'agent', got %q", inputCtx.SelectedPrimitive.Type)
			}
			if inputCtx.NetworkName != "My Agent" {
				t.Errorf("expected networkName 'My Agent', got %q", inputCtx.NetworkName)
			}

			outputStr, ok := capturedInput.Output.(string)
			if !ok {
				t.Fatalf("expected output to be string, got %T", capturedInput.Output)
			}
			if outputStr != "Stream output text" {
				t.Errorf("expected output 'Stream output text', got %q", outputStr)
			}

			// Check custom context has tool data
			if inputCtx.CustomContext == nil {
				t.Fatal("expected customContext to be non-nil")
			}
			toolCalls, ok := inputCtx.CustomContext["toolCalls"]
			if !ok {
				t.Fatal("expected customContext to have 'toolCalls'")
			}
			tc, ok := toolCalls.([]ToolCallInfo)
			if !ok {
				t.Fatalf("expected toolCalls to be []ToolCallInfo, got %T", toolCalls)
			}
			if len(tc) != 1 || tc[0].Name != "fetchData" {
				t.Errorf("unexpected toolCalls: %+v", tc)
			}

			agentID, ok := inputCtx.CustomContext["agentId"]
			if !ok || agentID != "my-agent" {
				t.Errorf("expected customContext.agentId 'my-agent', got %v", agentID)
			}
			agentName, ok := inputCtx.CustomContext["agentName"]
			if !ok || agentName != "My Agent" {
				t.Errorf("expected customContext.agentName 'My Agent', got %v", agentName)
			}
		})

		t.Run("uses agentId as networkName when agentName is not provided", func(t *testing.T) {
			var capturedInput ScorerRunInput
			scorer := &mockScorer{
				id:   "scorer-1",
				name: "scorer-1 Scorer",
				runFn: func(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error) {
					capturedInput = input
					return &ScorerRunResult{Score: 1}, nil
				},
			}

			sCtx := createMockStreamContext(func(c *StreamCompletionContext) {
				c.AgentID = "my-agent-id"
				c.AgentName = ""
			})

			RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer},
				sCtx,
				nil,
			)

			inputCtx, ok := capturedInput.Input.(CompletionContext)
			if !ok {
				t.Fatalf("expected input to be CompletionContext, got %T", capturedInput.Input)
			}
			if inputCtx.NetworkName != "my-agent-id" {
				t.Errorf("expected networkName 'my-agent-id', got %q", inputCtx.NetworkName)
			}
		})

		t.Run("uses stream as default networkName when neither agentId nor agentName provided", func(t *testing.T) {
			var capturedInput ScorerRunInput
			scorer := &mockScorer{
				id:   "scorer-1",
				name: "scorer-1 Scorer",
				runFn: func(ctx context.Context, input ScorerRunInput) (*ScorerRunResult, error) {
					capturedInput = input
					return &ScorerRunResult{Score: 1}, nil
				},
			}

			sCtx := createMockStreamContext(func(c *StreamCompletionContext) {
				c.AgentID = ""
				c.AgentName = ""
			})

			RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer},
				sCtx,
				nil,
			)

			inputCtx, ok := capturedInput.Input.(CompletionContext)
			if !ok {
				t.Fatalf("expected input to be CompletionContext, got %T", capturedInput.Input)
			}
			if inputCtx.NetworkName != "stream" {
				t.Errorf("expected networkName 'stream', got %q", inputCtx.NetworkName)
			}
		})
	})

	t.Run("error handling", func(t *testing.T) {
		t.Run("handles scorer that throws an error", func(t *testing.T) {
			errorScorer := createErrorScorer("error-scorer", "Scorer crashed")
			sCtx := createMockStreamContext()

			result := RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{errorScorer},
				sCtx,
				nil,
			)

			if result.Complete {
				t.Error("expected complete to be false")
			}
			if result.Scorers[0].Passed {
				t.Error("expected scorer to fail")
			}
			if !strings.Contains(result.Scorers[0].Reason, "Scorer threw an error") {
				t.Errorf("expected reason to contain 'Scorer threw an error', got %q", result.Scorers[0].Reason)
			}
			if !strings.Contains(result.Scorers[0].Reason, "Scorer crashed") {
				t.Errorf("expected reason to contain 'Scorer crashed', got %q", result.Scorers[0].Reason)
			}
		})
	})

	t.Run("sequential execution", func(t *testing.T) {
		t.Run("short-circuits on failure with all strategy", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 0, "Failed")
			scorer2 := createMockScorer("scorer-2", 1, "Passed")
			sCtx := createMockStreamContext()

			result := RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				sCtx,
				&CompletionScorerOptions{Parallel: false, Strategy: "all"},
			)

			if result.Complete {
				t.Error("expected complete to be false")
			}
			if scorer1.CallCount() != 1 {
				t.Errorf("expected scorer-1 to be called once, got %d", scorer1.CallCount())
			}
			if scorer2.CallCount() != 0 {
				t.Errorf("expected scorer-2 to not be called, got %d", scorer2.CallCount())
			}
		})

		t.Run("short-circuits on success with any strategy", func(t *testing.T) {
			scorer1 := createMockScorer("scorer-1", 1, "Passed")
			scorer2 := createMockScorer("scorer-2", 0, "Failed")
			sCtx := createMockStreamContext()

			result := RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{scorer1, scorer2},
				sCtx,
				&CompletionScorerOptions{Parallel: false, Strategy: "any"},
			)

			if !result.Complete {
				t.Error("expected complete to be true")
			}
			if scorer1.CallCount() != 1 {
				t.Errorf("expected scorer-1 to be called once, got %d", scorer1.CallCount())
			}
			if scorer2.CallCount() != 0 {
				t.Errorf("expected scorer-2 to not be called, got %d", scorer2.CallCount())
			}
		})
	})

	t.Run("empty scorers", func(t *testing.T) {
		t.Run("returns complete with empty scorers array and all strategy", func(t *testing.T) {
			sCtx := createMockStreamContext()
			result := RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{},
				sCtx,
				&CompletionScorerOptions{Strategy: "all", Parallel: true},
			)

			if !result.Complete {
				t.Error("expected complete to be true for empty scorers with 'all' strategy")
			}
			if len(result.Scorers) != 0 {
				t.Errorf("expected 0 scorers, got %d", len(result.Scorers))
			}
		})

		t.Run("returns incomplete with empty scorers array and any strategy", func(t *testing.T) {
			sCtx := createMockStreamContext()
			result := RunStreamCompletionScorers(
				context.Background(),
				[]MastraScorer{},
				sCtx,
				&CompletionScorerOptions{Strategy: "any", Parallel: true},
			)

			if result.Complete {
				t.Error("expected complete to be false for empty scorers with 'any' strategy")
			}
			if len(result.Scorers) != 0 {
				t.Errorf("expected 0 scorers, got %d", len(result.Scorers))
			}
		})
	})
}

// ---------------------------------------------------------------------------
// formatStreamCompletionFeedback tests
// ---------------------------------------------------------------------------

func TestFormatStreamCompletionFeedback(t *testing.T) {
	t.Run("formats complete result with stream-specific messaging", func(t *testing.T) {
		result := CompletionRunResult{
			Complete:         true,
			CompletionReason: "All checks passed",
			Scorers: []ScorerResult{
				{
					Score:      1,
					Passed:     true,
					Reason:     "Test passed",
					ScorerID:   "test-scorer",
					ScorerName: "Test Scorer",
					Duration:   100,
				},
			},
			TotalDuration: 150,
			TimedOut:      false,
		}

		feedback := FormatStreamCompletionFeedback(result, false)

		assertContains(t, feedback, "#### Completion Check Results")
		assertContains(t, feedback, "COMPLETE")
		assertContains(t, feedback, "Duration: 150ms")
		assertContains(t, feedback, "**Test Scorer** (test-scorer)")
		assertContains(t, feedback, "Score: 1")
		assertContains(t, feedback, "Reason: Test passed")
		assertContains(t, feedback, "The task is complete")
	})

	t.Run("formats incomplete result with continuation message", func(t *testing.T) {
		result := CompletionRunResult{
			Complete:         false,
			CompletionReason: "Check failed",
			Scorers: []ScorerResult{
				{
					Score:      0,
					Passed:     false,
					Reason:     "Validation failed",
					ScorerID:   "test-scorer",
					ScorerName: "Test Scorer",
					Duration:   100,
				},
			},
			TotalDuration: 150,
			TimedOut:      false,
		}

		feedback := FormatStreamCompletionFeedback(result, false)

		assertContains(t, feedback, "NOT COMPLETE")
		assertContains(t, feedback, "Score: 0")
		assertContains(t, feedback, "Reason: Validation failed")
		assertContains(t, feedback, "The task is not yet complete")
		assertContains(t, feedback, "continue working")
	})

	t.Run("formats max iterations reached message", func(t *testing.T) {
		result := CompletionRunResult{
			Complete:         false,
			CompletionReason: "Check failed",
			Scorers: []ScorerResult{
				{
					Score:      0,
					Passed:     false,
					Reason:     "Still in progress",
					ScorerID:   "test-scorer",
					ScorerName: "Test Scorer",
					Duration:   100,
				},
			},
			TotalDuration: 150,
			TimedOut:      false,
		}

		feedback := FormatStreamCompletionFeedback(result, true)

		assertContains(t, feedback, "NOT COMPLETE")
		assertContains(t, feedback, "Max iterations reached")
	})

	t.Run("formats timeout indication", func(t *testing.T) {
		result := CompletionRunResult{
			Complete:      false,
			Scorers:       []ScorerResult{},
			TotalDuration: 600000,
			TimedOut:      true,
		}

		feedback := FormatStreamCompletionFeedback(result, false)

		assertContains(t, feedback, "Scoring timed out")
	})

	t.Run("formats multiple scorers with bold names", func(t *testing.T) {
		result := CompletionRunResult{
			Complete: false,
			Scorers: []ScorerResult{
				{
					Score:      1,
					Passed:     true,
					Reason:     "First passed",
					ScorerID:   "scorer-1",
					ScorerName: "Scorer One",
					Duration:   50,
				},
				{
					Score:      0,
					Passed:     false,
					Reason:     "Second failed",
					ScorerID:   "scorer-2",
					ScorerName: "Scorer Two",
					Duration:   75,
				},
			},
			TotalDuration: 125,
			TimedOut:      false,
		}

		feedback := FormatStreamCompletionFeedback(result, false)

		assertContains(t, feedback, "**Scorer One** (scorer-1)")
		assertContains(t, feedback, "**Scorer Two** (scorer-2)")
		assertContains(t, feedback, "First passed")
		assertContains(t, feedback, "Second failed")
	})

	t.Run("handles scorer without reason", func(t *testing.T) {
		result := CompletionRunResult{
			Complete: true,
			Scorers: []ScorerResult{
				{
					Score:      1,
					Passed:     true,
					ScorerID:   "test-scorer",
					ScorerName: "Test Scorer",
					Duration:   100,
				},
			},
			TotalDuration: 100,
			TimedOut:      false,
		}

		feedback := FormatStreamCompletionFeedback(result, false)

		assertContains(t, feedback, "Score: 1")
		assertNotContains(t, feedback, "Reason:")
	})
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Errorf("expected string to contain %q, got:\n%s", substr, s)
	}
}

func assertNotContains(t *testing.T, s, substr string) {
	t.Helper()
	if strings.Contains(s, substr) {
		t.Errorf("expected string NOT to contain %q, got:\n%s", substr, s)
	}
}
