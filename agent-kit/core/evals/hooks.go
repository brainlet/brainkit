// Ported from: packages/core/src/evals/hooks.ts
package evals

import (
	"math/rand"

	"github.com/brainlet/brainkit/agent-kit/core/hooks"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// AvailableHooks enumerates the hook event types used by the scoring system.
// Corresponds to TS: AvailableHooks enum (only the ON_SCORER_RUN variant is used here).
const (
	HookOnScorerRun = "onScorerRun"
)

// RunScorerParams holds the parameters for the RunScorer function.
// Corresponds to the destructured parameter object of TS: runScorer().
type RunScorerParams struct {
	RunID                string
	ScorerID             string
	ScorerObject         MastraScorerEntry
	Input                any
	Output               any
	RequestContext       map[string]any
	Entity               map[string]any
	StructuredOutput     bool
	Source               ScoringSource
	EntityType           ScoringEntityType
	ThreadID             string
	ResourceID           string
	ObservabilityContext *obstypes.ObservabilityContext
}

// RunScorer determines whether a scorer should execute based on its sampling
// configuration, builds the ScoringHookInput payload, and fires the
// ON_SCORER_RUN hook asynchronously.
//
// Corresponds to TS: export function runScorer({ ... })
func RunScorer(emitter *hooks.Emitter, params RunScorerParams) {
	shouldExecute := false

	// If no sampling config or sampling type is "none", always execute.
	if params.ScorerObject.Sampling == nil || params.ScorerObject.Sampling.Type == ScoringSamplingNone {
		shouldExecute = true
	}

	// Apply sampling strategy.
	if params.ScorerObject.Sampling != nil && params.ScorerObject.Sampling.Type != "" {
		switch params.ScorerObject.Sampling.Type {
		case ScoringSamplingRatio:
			shouldExecute = rand.Float64() < params.ScorerObject.Sampling.Rate
		default:
			shouldExecute = true
		}
	}

	if !shouldExecute {
		return
	}

	// Build the scorer info map.
	scorerID := params.ScorerID
	if params.ScorerObject.Scorer != nil && params.ScorerObject.Scorer.ID() != "" {
		scorerID = params.ScorerObject.Scorer.ID()
	}

	scorerMap := map[string]any{
		"id":          scorerID,
		"name":        params.ScorerObject.Scorer.Name(),
		"description": params.ScorerObject.Scorer.Description(),
	}

	structuredOutput := params.StructuredOutput
	payload := ScoringHookInput{
		Scorer:               scorerMap,
		Input:                params.Input,
		Output:               params.Output,
		RequestContext:       params.RequestContext,
		RunID:                params.RunID,
		Source:               params.Source,
		Entity:               params.Entity,
		StructuredOutput:     &structuredOutput,
		EntityType:           params.EntityType,
		ThreadID:             params.ThreadID,
		ResourceID:           params.ResourceID,
		ObservabilityContext: params.ObservabilityContext,
	}

	// Fire-and-forget: TS uses setImmediate inside executeHook.
	// In Go we fire the hook asynchronously via a goroutine to avoid
	// blocking the caller, matching the TS non-blocking behavior.
	go emitter.Emit(HookOnScorerRun, payload)
}
