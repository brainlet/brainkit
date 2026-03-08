// Ported from: packages/core/src/evals/types.test.ts
package evals

import (
	"testing"
)

func TestScoringSamplingConfigType(t *testing.T) {
	t.Run("has correct values", func(t *testing.T) {
		cases := []struct {
			configType ScoringSamplingConfigType
			want       string
		}{
			{ScoringSamplingNone, "none"},
			{ScoringSamplingRatio, "ratio"},
		}
		for _, tc := range cases {
			if string(tc.configType) != tc.want {
				t.Errorf("type = %q, want %q", tc.configType, tc.want)
			}
		}
	})

	t.Run("types are distinct", func(t *testing.T) {
		if ScoringSamplingNone == ScoringSamplingRatio {
			t.Error("sampling types should be distinct")
		}
	})
}

func TestScoringSource(t *testing.T) {
	t.Run("has correct values", func(t *testing.T) {
		cases := []struct {
			source ScoringSource
			want   string
		}{
			{ScoringSourceLive, "LIVE"},
			{ScoringSourceTest, "TEST"},
		}
		for _, tc := range cases {
			if string(tc.source) != tc.want {
				t.Errorf("source = %q, want %q", tc.source, tc.want)
			}
		}
	})

	t.Run("sources are distinct", func(t *testing.T) {
		if ScoringSourceLive == ScoringSourceTest {
			t.Error("scoring sources should be distinct")
		}
	})
}

func TestScoringEntityType(t *testing.T) {
	t.Run("has agent and workflow types", func(t *testing.T) {
		if string(ScoringEntityTypeAgent) != "AGENT" {
			t.Errorf("ScoringEntityTypeAgent = %q, want %q", ScoringEntityTypeAgent, "AGENT")
		}
		if string(ScoringEntityTypeWorkflow) != "WORKFLOW" {
			t.Errorf("ScoringEntityTypeWorkflow = %q, want %q", ScoringEntityTypeWorkflow, "WORKFLOW")
		}
	})

	t.Run("has span-derived entity types", func(t *testing.T) {
		spanTypes := []struct {
			entityType ScoringEntityType
			want       string
		}{
			{ScoringEntityTypeAgentRun, "agent_run"},
			{ScoringEntityTypeGeneric, "generic"},
			{ScoringEntityTypeModelGeneration, "model_generation"},
			{ScoringEntityTypeModelStep, "model_step"},
			{ScoringEntityTypeModelChunk, "model_chunk"},
			{ScoringEntityTypeMCPToolCall, "mcp_tool_call"},
			{ScoringEntityTypeProcessorRun, "processor_run"},
			{ScoringEntityTypeToolCall, "tool_call"},
			{ScoringEntityTypeWorkflowRun, "workflow_run"},
			{ScoringEntityTypeWorkflowStep, "workflow_step"},
			{ScoringEntityTypeWorkflowConditional, "workflow_conditional"},
			{ScoringEntityTypeWorkflowConditionalEval, "workflow_conditional_eval"},
			{ScoringEntityTypeWorkflowParallel, "workflow_parallel"},
			{ScoringEntityTypeWorkflowLoop, "workflow_loop"},
			{ScoringEntityTypeWorkflowSleep, "workflow_sleep"},
			{ScoringEntityTypeWorkflowWaitEvent, "workflow_wait_event"},
		}
		for _, tc := range spanTypes {
			if string(tc.entityType) != tc.want {
				t.Errorf("entity type = %q, want %q", tc.entityType, tc.want)
			}
		}
	})

	t.Run("all entity types are distinct", func(t *testing.T) {
		all := []ScoringEntityType{
			ScoringEntityTypeAgent,
			ScoringEntityTypeWorkflow,
			ScoringEntityTypeAgentRun,
			ScoringEntityTypeGeneric,
			ScoringEntityTypeModelGeneration,
			ScoringEntityTypeModelStep,
			ScoringEntityTypeModelChunk,
			ScoringEntityTypeMCPToolCall,
			ScoringEntityTypeProcessorRun,
			ScoringEntityTypeToolCall,
			ScoringEntityTypeWorkflowRun,
			ScoringEntityTypeWorkflowStep,
			ScoringEntityTypeWorkflowConditional,
			ScoringEntityTypeWorkflowConditionalEval,
			ScoringEntityTypeWorkflowParallel,
			ScoringEntityTypeWorkflowLoop,
			ScoringEntityTypeWorkflowSleep,
			ScoringEntityTypeWorkflowWaitEvent,
		}
		seen := make(map[ScoringEntityType]bool)
		for _, et := range all {
			if seen[et] {
				t.Errorf("duplicate entity type: %q", et)
			}
			seen[et] = true
		}
	})
}

func TestScoringSamplingConfig(t *testing.T) {
	t.Run("none type has zero rate by default", func(t *testing.T) {
		cfg := ScoringSamplingConfig{Type: ScoringSamplingNone}
		if cfg.Rate != 0 {
			t.Errorf("Rate = %f, want 0", cfg.Rate)
		}
	})

	t.Run("ratio type accepts rate", func(t *testing.T) {
		cfg := ScoringSamplingConfig{Type: ScoringSamplingRatio, Rate: 0.5}
		if cfg.Rate != 0.5 {
			t.Errorf("Rate = %f, want 0.5", cfg.Rate)
		}
	})
}

func TestScoringPrompts(t *testing.T) {
	t.Run("holds description and prompt", func(t *testing.T) {
		sp := ScoringPrompts{
			Description: "test desc",
			Prompt:      "test prompt",
		}
		if sp.Description != "test desc" {
			t.Errorf("Description = %q, want %q", sp.Description, "test desc")
		}
		if sp.Prompt != "test prompt" {
			t.Errorf("Prompt = %q, want %q", sp.Prompt, "test prompt")
		}
	})
}

func TestScoringInput(t *testing.T) {
	t.Run("holds input and output", func(t *testing.T) {
		si := ScoringInput{
			RunID:  "run-1",
			Input:  "test input",
			Output: "test output",
		}
		if si.RunID != "run-1" {
			t.Errorf("RunID = %q, want %q", si.RunID, "run-1")
		}
		if si.Input != "test input" {
			t.Errorf("Input = %v, want %q", si.Input, "test input")
		}
		if si.Output != "test output" {
			t.Errorf("Output = %v, want %q", si.Output, "test output")
		}
	})

	t.Run("accepts additional context", func(t *testing.T) {
		si := ScoringInput{
			Output:            "out",
			AdditionalContext: map[string]any{"key": "val"},
		}
		if si.AdditionalContext["key"] != "val" {
			t.Errorf("AdditionalContext[key] = %v, want %q", si.AdditionalContext["key"], "val")
		}
	})
}

func TestScoringHookInput(t *testing.T) {
	t.Run("holds all scoring hook fields", func(t *testing.T) {
		structuredOutput := true
		shi := ScoringHookInput{
			RunID:            "run-1",
			Scorer:           map[string]any{"id": "s1"},
			Input:            "in",
			Output:           "out",
			Source:           ScoringSourceLive,
			Entity:           map[string]any{"id": "e1"},
			EntityType:       ScoringEntityTypeAgent,
			StructuredOutput: &structuredOutput,
			TraceID:          "trace-1",
			SpanID:           "span-1",
			ResourceID:       "res-1",
			ThreadID:         "thread-1",
		}
		if shi.RunID != "run-1" {
			t.Errorf("RunID = %q, want %q", shi.RunID, "run-1")
		}
		if shi.Source != ScoringSourceLive {
			t.Errorf("Source = %q, want %q", shi.Source, ScoringSourceLive)
		}
		if shi.EntityType != ScoringEntityTypeAgent {
			t.Errorf("EntityType = %q, want %q", shi.EntityType, ScoringEntityTypeAgent)
		}
		if shi.StructuredOutput == nil || !*shi.StructuredOutput {
			t.Error("StructuredOutput should be true")
		}
	})
}

func TestScorerOptions(t *testing.T) {
	t.Run("holds scorer configuration", func(t *testing.T) {
		opts := ScorerOptions{
			Name:        "test-scorer",
			Description: "A test scorer",
			IsLLMScorer: true,
			Metadata:    map[string]any{"version": "1.0"},
		}
		if opts.Name != "test-scorer" {
			t.Errorf("Name = %q, want %q", opts.Name, "test-scorer")
		}
		if opts.Description != "A test scorer" {
			t.Errorf("Description = %q, want %q", opts.Description, "A test scorer")
		}
		if !opts.IsLLMScorer {
			t.Error("IsLLMScorer should be true")
		}
		if opts.Metadata["version"] != "1.0" {
			t.Errorf("Metadata[version] = %v, want %q", opts.Metadata["version"], "1.0")
		}
	})
}

func TestReasonResult(t *testing.T) {
	t.Run("holds reason and prompt", func(t *testing.T) {
		rr := ReasonResult{
			Reason:       "the output matches",
			ReasonPrompt: "explain why",
		}
		if rr.Reason != "the output matches" {
			t.Errorf("Reason = %q, want %q", rr.Reason, "the output matches")
		}
		if rr.ReasonPrompt != "explain why" {
			t.Errorf("ReasonPrompt = %q, want %q", rr.ReasonPrompt, "explain why")
		}
	})
}

func TestMastraScorerEntry(t *testing.T) {
	t.Run("holds scorer with optional sampling", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		})
		entry := MastraScorerEntry{
			Scorer: scorer,
			Sampling: &ScoringSamplingConfig{
				Type: ScoringSamplingRatio,
				Rate: 0.5,
			},
		}
		if entry.Scorer == nil {
			t.Fatal("Scorer should not be nil")
		}
		if entry.Sampling == nil {
			t.Fatal("Sampling should not be nil")
		}
		if entry.Sampling.Rate != 0.5 {
			t.Errorf("Sampling.Rate = %f, want 0.5", entry.Sampling.Rate)
		}
	})

	t.Run("sampling can be nil", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		})
		entry := MastraScorerEntry{
			Scorer: scorer,
		}
		if entry.Sampling != nil {
			t.Error("Sampling should be nil when not provided")
		}
	})
}
