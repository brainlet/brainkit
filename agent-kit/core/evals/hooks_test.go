// Ported from: packages/core/src/evals/hooks.test.ts
package evals

import (
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/hooks"
)

func TestHookConstants(t *testing.T) {
	t.Run("HookOnScorerRun has correct value", func(t *testing.T) {
		if HookOnScorerRun != "onScorerRun" {
			t.Errorf("HookOnScorerRun = %q, want %q", HookOnScorerRun, "onScorerRun")
		}
	})
}

func TestRunScorer(t *testing.T) {
	t.Run("emits hook when sampling is nil (always execute)", func(t *testing.T) {
		emitter := hooks.New()
		var mu sync.Mutex
		var received []any

		emitter.On(HookOnScorerRun, func(event any) {
			mu.Lock()
			received = append(received, event)
			mu.Unlock()
		})

		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "Test",
			Description: "desc",
		})

		RunScorer(emitter, RunScorerParams{
			RunID: "run-1",
			ScorerID: "test-scorer",
			ScorerObject: MastraScorerEntry{
				Scorer:   scorer,
				Sampling: nil,
			},
			Input:      "input",
			Output:     "output",
			Source:     ScoringSourceTest,
			Entity:     map[string]any{"id": "e1"},
			EntityType: ScoringEntityTypeAgent,
		})

		// RunScorer fires asynchronously via goroutine; wait briefly.
		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if len(received) != 1 {
			t.Fatalf("expected 1 event, got %d", len(received))
		}

		payload, ok := received[0].(ScoringHookInput)
		if !ok {
			t.Fatalf("expected ScoringHookInput, got %T", received[0])
		}
		if payload.RunID != "run-1" {
			t.Errorf("RunID = %q, want %q", payload.RunID, "run-1")
		}
		if payload.Source != ScoringSourceTest {
			t.Errorf("Source = %q, want %q", payload.Source, ScoringSourceTest)
		}
	})

	t.Run("emits hook when sampling type is none", func(t *testing.T) {
		emitter := hooks.New()
		var mu sync.Mutex
		var received []any

		emitter.On(HookOnScorerRun, func(event any) {
			mu.Lock()
			received = append(received, event)
			mu.Unlock()
		})

		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "Test",
			Description: "desc",
		})

		RunScorer(emitter, RunScorerParams{
			RunID:    "run-2",
			ScorerID: "test-scorer",
			ScorerObject: MastraScorerEntry{
				Scorer: scorer,
				Sampling: &ScoringSamplingConfig{
					Type: ScoringSamplingNone,
				},
			},
			Input:      "input",
			Output:     "output",
			Source:     ScoringSourceLive,
			Entity:     map[string]any{},
			EntityType: ScoringEntityTypeAgent,
		})

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if len(received) != 1 {
			t.Fatalf("expected 1 event, got %d", len(received))
		}
	})

	t.Run("does not emit when sampling rate is 0", func(t *testing.T) {
		emitter := hooks.New()
		var mu sync.Mutex
		emitted := false

		emitter.On(HookOnScorerRun, func(event any) {
			mu.Lock()
			emitted = true
			mu.Unlock()
		})

		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "Test",
			Description: "desc",
		})

		// With rate 0, rand.Float64() is always >= 0, so should never execute.
		RunScorer(emitter, RunScorerParams{
			RunID:    "run-3",
			ScorerID: "test-scorer",
			ScorerObject: MastraScorerEntry{
				Scorer: scorer,
				Sampling: &ScoringSamplingConfig{
					Type: ScoringSamplingRatio,
					Rate: 0,
				},
			},
			Input:      "input",
			Output:     "output",
			Source:     ScoringSourceTest,
			Entity:     map[string]any{},
			EntityType: ScoringEntityTypeAgent,
		})

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if emitted {
			t.Error("should not emit when sampling rate is 0")
		}
	})

	t.Run("always emits when sampling rate is 1", func(t *testing.T) {
		emitter := hooks.New()
		var mu sync.Mutex
		count := 0

		emitter.On(HookOnScorerRun, func(event any) {
			mu.Lock()
			count++
			mu.Unlock()
		})

		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "Test",
			Description: "desc",
		})

		// With rate 1.0, rand.Float64() is always < 1.0, so should always execute.
		for i := 0; i < 5; i++ {
			RunScorer(emitter, RunScorerParams{
				RunID:    "run",
				ScorerID: "test-scorer",
				ScorerObject: MastraScorerEntry{
					Scorer: scorer,
					Sampling: &ScoringSamplingConfig{
						Type: ScoringSamplingRatio,
						Rate: 1.0,
					},
				},
				Input:      "input",
				Output:     "output",
				Source:     ScoringSourceTest,
				Entity:     map[string]any{},
				EntityType: ScoringEntityTypeAgent,
			})
		}

		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if count != 5 {
			t.Errorf("expected 5 emissions with rate 1.0, got %d", count)
		}
	})

	t.Run("builds scorer map from scorer object", func(t *testing.T) {
		emitter := hooks.New()
		var mu sync.Mutex
		var payload ScoringHookInput

		emitter.On(HookOnScorerRun, func(event any) {
			mu.Lock()
			payload = event.(ScoringHookInput)
			mu.Unlock()
		})

		scorer := NewMastraScorer(ScorerConfig{
			ID:          "my-scorer",
			Name:        "My Scorer",
			Description: "A scorer",
		})

		RunScorer(emitter, RunScorerParams{
			RunID:    "run-1",
			ScorerID: "fallback-id",
			ScorerObject: MastraScorerEntry{
				Scorer: scorer,
			},
			Input:      "in",
			Output:     "out",
			Source:     ScoringSourceTest,
			Entity:     map[string]any{},
			EntityType: ScoringEntityTypeAgent,
		})

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if payload.Scorer == nil {
			t.Fatal("Scorer map should not be nil")
		}
		// Scorer ID from scorer object takes precedence over params.ScorerID.
		if payload.Scorer["id"] != "my-scorer" {
			t.Errorf("Scorer id = %v, want %q", payload.Scorer["id"], "my-scorer")
		}
		if payload.Scorer["name"] != "My Scorer" {
			t.Errorf("Scorer name = %v, want %q", payload.Scorer["name"], "My Scorer")
		}
		if payload.Scorer["description"] != "A scorer" {
			t.Errorf("Scorer description = %v, want %q", payload.Scorer["description"], "A scorer")
		}
	})

	t.Run("sets structured output in payload", func(t *testing.T) {
		emitter := hooks.New()
		var mu sync.Mutex
		var payload ScoringHookInput

		emitter.On(HookOnScorerRun, func(event any) {
			mu.Lock()
			payload = event.(ScoringHookInput)
			mu.Unlock()
		})

		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		})

		RunScorer(emitter, RunScorerParams{
			ScorerID: "test",
			ScorerObject: MastraScorerEntry{
				Scorer: scorer,
			},
			Input:            "in",
			Output:           "out",
			StructuredOutput: true,
			Source:           ScoringSourceTest,
			Entity:           map[string]any{},
			EntityType:       ScoringEntityTypeAgent,
		})

		time.Sleep(50 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()
		if payload.StructuredOutput == nil {
			t.Fatal("StructuredOutput should not be nil")
		}
		if !*payload.StructuredOutput {
			t.Error("StructuredOutput should be true")
		}
	})
}
