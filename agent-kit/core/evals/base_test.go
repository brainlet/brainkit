// Ported from: packages/core/src/evals/base.test.ts
package evals

import (
	"context"
	"testing"
)

func TestNewMastraScorer(t *testing.T) {
	t.Run("creates scorer with valid config", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test-scorer",
			Name:        "Test Scorer",
			Description: "A test scorer",
		})
		if scorer.ID() != "test-scorer" {
			t.Errorf("ID() = %q, want %q", scorer.ID(), "test-scorer")
		}
		if scorer.Name() != "Test Scorer" {
			t.Errorf("Name() = %q, want %q", scorer.Name(), "Test Scorer")
		}
		if scorer.Description() != "A test scorer" {
			t.Errorf("Description() = %q, want %q", scorer.Description(), "A test scorer")
		}
	})

	t.Run("panics when ID is empty", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic, got nil")
			}
		}()
		NewMastraScorer(ScorerConfig{
			Name:        "No ID Scorer",
			Description: "Should panic",
		})
	})

	t.Run("Name falls back to ID when not provided", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "fallback-id",
			Description: "Test",
		})
		if scorer.Name() != "fallback-id" {
			t.Errorf("Name() = %q, want %q", scorer.Name(), "fallback-id")
		}
	})

	t.Run("Judge returns nil when not configured", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		})
		if scorer.Judge() != nil {
			t.Error("Judge should be nil when not configured")
		}
	})

	t.Run("Judge returns config when set", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
			Judge: &ScorerJudgeConfig{
				Model:        "mock-model",
				Instructions: "test instructions",
			},
		})
		if scorer.Judge() == nil {
			t.Fatal("Judge should not be nil")
		}
		if scorer.Judge().Instructions != "test instructions" {
			t.Errorf("Instructions = %q, want %q", scorer.Judge().Instructions, "test instructions")
		}
	})
}

func TestCreateScorer(t *testing.T) {
	t.Run("creates scorer and uses ID as default name", func(t *testing.T) {
		scorer := CreateScorer(ScorerConfig{
			ID:          "my-scorer",
			Description: "Scorer description",
		})
		if scorer.ID() != "my-scorer" {
			t.Errorf("ID() = %q, want %q", scorer.ID(), "my-scorer")
		}
		if scorer.Name() != "my-scorer" {
			t.Errorf("Name() = %q, want %q", scorer.Name(), "my-scorer")
		}
	})

	t.Run("creates scorer with explicit name", func(t *testing.T) {
		scorer := CreateScorer(ScorerConfig{
			ID:          "my-scorer",
			Name:        "My Scorer",
			Description: "Scorer description",
		})
		if scorer.Name() != "My Scorer" {
			t.Errorf("Name() = %q, want %q", scorer.Name(), "My Scorer")
		}
	})
}

func TestScorerPipeline(t *testing.T) {
	t.Run("chaining steps returns new scorers", func(t *testing.T) {
		base := CreateScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		})

		withPreprocess := base.Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return "preprocessed", nil
		}))

		if base == withPreprocess {
			t.Error("Preprocess should return a new scorer")
		}

		withScore := withPreprocess.GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			return 1.0, nil
		}))

		if withPreprocess == withScore {
			t.Error("GenerateScore should return a new scorer")
		}
	})

	t.Run("GetSteps returns step info", func(t *testing.T) {
		scorer := CreateScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return nil, nil
		})).Analyze(FunctionStep(func(ctx *StepContext) (any, error) {
			return nil, nil
		})).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			return 0, nil
		})).GenerateReason(GenerateReasonFunctionStep(func(ctx *GenerateReasonContext) (any, error) {
			return nil, nil
		}))

		steps := scorer.GetSteps()
		if len(steps) != 4 {
			t.Fatalf("expected 4 steps, got %d", len(steps))
		}

		expectedNames := []string{"preprocess", "analyze", "generateScore", "generateReason"}
		for i, name := range expectedNames {
			if steps[i].Name != name {
				t.Errorf("step[%d].Name = %q, want %q", i, steps[i].Name, name)
			}
			if steps[i].Type != "function" {
				t.Errorf("step[%d].Type = %q, want %q", i, steps[i].Type, "function")
			}
		}
	})

	t.Run("GetSteps identifies prompt objects", func(t *testing.T) {
		scorer := CreateScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		}).Analyze(&PromptObject{
			Description: "test",
			Judge: &ScorerJudgeConfig{
				Model:        "mock",
				Instructions: "test",
			},
			CreatePrompt: func(ctx *StepContext) (string, error) { return "", nil },
		}).GenerateScore(GenerateScoreFunctionStep(func(ctx *StepContext) (float64, error) {
			return 0, nil
		}))

		steps := scorer.GetSteps()
		if len(steps) != 2 {
			t.Fatalf("expected 2 steps, got %d", len(steps))
		}
		if steps[0].Type != "prompt" {
			t.Errorf("analyze step type = %q, want %q", steps[0].Type, "prompt")
		}
		if steps[1].Type != "function" {
			t.Errorf("generateScore step type = %q, want %q", steps[1].Type, "function")
		}
	})
}

func TestScorerRun(t *testing.T) {
	t.Run("errors when generateScore is missing", func(t *testing.T) {
		scorer := CreateScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		}).Preprocess(FunctionStep(func(ctx *StepContext) (any, error) {
			return nil, nil
		}))

		_, err := scorer.Run(context.Background(), &ScorerRun{
			Input:  "test input",
			Output: "test output",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("basic scorer runs successfully", func(t *testing.T) {
		scorer := FunctionBasedScorerBuilders.Basic()
		result, err := scorer.Run(context.Background(), &ScorerRun{
			Input:  "test input",
			Output: "test output",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("result should not be nil")
		}
		score, ok := result.Score.(float64)
		if !ok {
			t.Fatalf("Score should be float64, got %T", result.Score)
		}
		if score != 1.0 {
			t.Errorf("Score = %v, want 1.0", score)
		}
	})

	t.Run("scorer with preprocess runs successfully", func(t *testing.T) {
		scorer := FunctionBasedScorerBuilders.WithPreprocess()
		result, err := scorer.Run(context.Background(), &ScorerRun{
			Input:  "test input",
			Output: "test output",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		score, ok := result.Score.(float64)
		if !ok {
			t.Fatalf("Score should be float64, got %T", result.Score)
		}
		if score != 1.0 {
			t.Errorf("Score = %v, want 1.0", score)
		}
	})

	t.Run("scorer with preprocess and analyze runs successfully", func(t *testing.T) {
		scorer := FunctionBasedScorerBuilders.WithPreprocessAndAnalyze()
		result, err := scorer.Run(context.Background(), &ScorerRun{
			Input:  "test input",
			Output: "test output",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		score, ok := result.Score.(float64)
		if !ok {
			t.Fatalf("Score should be float64, got %T", result.Score)
		}
		if score != 1.0 {
			t.Errorf("Score = %v, want 1.0", score)
		}
	})

	t.Run("scorer with all steps runs successfully", func(t *testing.T) {
		scorer := FunctionBasedScorerBuilders.WithPreprocessAndAnalyzeAndReason()
		result, err := scorer.Run(context.Background(), &ScorerRun{
			Input:  "test input",
			Output: "test output",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		score, ok := result.Score.(float64)
		if !ok {
			t.Fatalf("Score should be float64, got %T", result.Score)
		}
		if score != 1.0 {
			t.Errorf("Score = %v, want 1.0", score)
		}
		if result.Reason == nil {
			t.Error("Reason should not be nil")
		}
		reason, ok := result.Reason.(string)
		if !ok {
			t.Fatalf("Reason should be string, got %T", result.Reason)
		}
		if reason == "" {
			t.Error("Reason should not be empty")
		}
	})

	t.Run("scorer with reason only runs successfully", func(t *testing.T) {
		scorer := FunctionBasedScorerBuilders.WithReason()
		result, err := scorer.Run(context.Background(), &ScorerRun{
			Input:  "test input",
			Output: "test output",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		score, ok := result.Score.(float64)
		if !ok {
			t.Fatalf("Score should be float64, got %T", result.Score)
		}
		if score != 1.0 {
			t.Errorf("Score = %v, want 1.0", score)
		}
		if result.Reason == nil {
			t.Error("Reason should not be nil")
		}
	})

	t.Run("sets run ID when not provided", func(t *testing.T) {
		scorer := FunctionBasedScorerBuilders.Basic()
		result, err := scorer.Run(context.Background(), &ScorerRun{
			Input:  "input",
			Output: "output",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.RunID == "" {
			t.Error("RunID should be auto-generated")
		}
	})

	t.Run("preserves provided run ID", func(t *testing.T) {
		scorer := FunctionBasedScorerBuilders.Basic()
		result, err := scorer.Run(context.Background(), &ScorerRun{
			RunID:  "custom-run-id",
			Input:  "input",
			Output: "output",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.RunID != "custom-run-id" {
			t.Errorf("RunID = %q, want %q", result.RunID, "custom-run-id")
		}
	})

	t.Run("passes input and output through to result", func(t *testing.T) {
		scorer := FunctionBasedScorerBuilders.Basic()
		result, err := scorer.Run(context.Background(), &ScorerRun{
			Input:  "my input",
			Output: "my output",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Input != "my input" {
			t.Errorf("Input = %v, want %q", result.Input, "my input")
		}
		if result.Output != "my output" {
			t.Errorf("Output = %v, want %q", result.Output, "my output")
		}
	})

	t.Run("passes ground truth through to result", func(t *testing.T) {
		scorer := FunctionBasedScorerBuilders.Basic()
		result, err := scorer.Run(context.Background(), &ScorerRun{
			Input:       "input",
			Output:      "output",
			GroundTruth: "expected answer",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.GroundTruth != "expected answer" {
			t.Errorf("GroundTruth = %v, want %q", result.GroundTruth, "expected answer")
		}
	})
}

func TestRegisterMastra(t *testing.T) {
	t.Run("registers mastra instance", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		})
		scorer.RegisterMastra("mastra-instance")
		// No panic means success — mastra field is private
	})
}

func TestRawConfig(t *testing.T) {
	t.Run("ToRawConfig returns nil initially", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		})
		if scorer.ToRawConfig() != nil {
			t.Error("ToRawConfig should return nil initially")
		}
	})

	t.Run("SetRawConfig and ToRawConfig round-trip", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		})
		raw := map[string]any{"key": "value"}
		scorer.SetRawConfig(raw)
		got := scorer.ToRawConfig()
		if got == nil {
			t.Fatal("ToRawConfig should not be nil after set")
		}
		if got["key"] != "value" {
			t.Errorf("ToRawConfig()[key] = %v, want %q", got["key"], "value")
		}
	})
}

func TestGetType(t *testing.T) {
	t.Run("returns nil when not set", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
		})
		if scorer.GetType() != nil {
			t.Error("GetType should return nil when not set")
		}
	})

	t.Run("returns the configured type", func(t *testing.T) {
		scorer := NewMastraScorer(ScorerConfig{
			ID:          "test",
			Description: "test",
			Type:        ScorerTypeAgent,
		})
		if scorer.GetType() != ScorerTypeAgent {
			t.Errorf("GetType() = %v, want %v", scorer.GetType(), ScorerTypeAgent)
		}
	})
}

func TestIsPromptObject(t *testing.T) {
	t.Run("returns true for PromptObject", func(t *testing.T) {
		po := &PromptObject{Description: "test"}
		if !isPromptObject(po) {
			t.Error("should be true for *PromptObject")
		}
	})

	t.Run("returns true for GenerateScorePromptObject", func(t *testing.T) {
		po := &GenerateScorePromptObject{Description: "test"}
		if !isPromptObject(po) {
			t.Error("should be true for *GenerateScorePromptObject")
		}
	})

	t.Run("returns true for GenerateReasonPromptObject", func(t *testing.T) {
		po := &GenerateReasonPromptObject{Description: "test"}
		if !isPromptObject(po) {
			t.Error("should be true for *GenerateReasonPromptObject")
		}
	})

	t.Run("returns false for FunctionStep", func(t *testing.T) {
		fn := FunctionStep(func(ctx *StepContext) (any, error) { return nil, nil })
		if isPromptObject(fn) {
			t.Error("should be false for FunctionStep")
		}
	})

	t.Run("returns false for nil", func(t *testing.T) {
		if isPromptObject(nil) {
			t.Error("should be false for nil")
		}
	})
}
