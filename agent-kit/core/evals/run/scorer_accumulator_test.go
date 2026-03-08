// Ported from: packages/core/src/evals/run/scorerAccumulator.test.ts
package run

import (
	"math"
	"testing"
)

func TestNewScoreAccumulator(t *testing.T) {
	t.Run("creates empty accumulator", func(t *testing.T) {
		sa := NewScoreAccumulator()
		result := sa.GetAverageScores()
		if len(result) != 0 {
			t.Errorf("expected empty result, got %v", result)
		}
	})
}

func TestScoreAccumulatorAddFlatScores(t *testing.T) {
	t.Run("accumulates flat scores", func(t *testing.T) {
		sa := NewScoreAccumulator()
		sa.AddScores(map[string]any{
			"scorer1": map[string]any{"score": 0.8},
			"scorer2": map[string]any{"score": 0.6},
		})
		sa.AddScores(map[string]any{
			"scorer1": map[string]any{"score": 1.0},
			"scorer2": map[string]any{"score": 0.4},
		})

		result := sa.GetAverageScores()
		s1, ok := result["scorer1"].(float64)
		if !ok {
			t.Fatalf("scorer1 not found or not float64: %v", result["scorer1"])
		}
		if math.Abs(s1-0.9) > 0.001 {
			t.Errorf("scorer1 average = %f, want 0.9", s1)
		}

		s2, ok := result["scorer2"].(float64)
		if !ok {
			t.Fatalf("scorer2 not found or not float64: %v", result["scorer2"])
		}
		if math.Abs(s2-0.5) > 0.001 {
			t.Errorf("scorer2 average = %f, want 0.5", s2)
		}
	})
}

func TestScoreAccumulatorAddNestedScores(t *testing.T) {
	t.Run("accumulates workflow-level scores", func(t *testing.T) {
		sa := NewScoreAccumulator()
		sa.AddScores(map[string]any{
			"steps": map[string]any{},
			"workflow": map[string]any{
				"wf-scorer": map[string]any{"score": 0.7},
			},
		})
		sa.AddScores(map[string]any{
			"steps": map[string]any{},
			"workflow": map[string]any{
				"wf-scorer": map[string]any{"score": 0.3},
			},
		})

		result := sa.GetAverageScores()
		wf, ok := result["workflow"].(map[string]any)
		if !ok {
			t.Fatalf("workflow key not found or not map: %v", result["workflow"])
		}
		avg, ok := wf["wf-scorer"].(float64)
		if !ok {
			t.Fatalf("wf-scorer not found or not float64: %v", wf["wf-scorer"])
		}
		if math.Abs(avg-0.5) > 0.001 {
			t.Errorf("wf-scorer average = %f, want 0.5", avg)
		}
	})

	t.Run("accumulates step-level scores", func(t *testing.T) {
		sa := NewScoreAccumulator()
		sa.AddScores(map[string]any{
			"steps": map[string]any{
				"step1": map[string]any{
					"step-scorer": map[string]any{"score": 1.0},
				},
			},
		})
		sa.AddScores(map[string]any{
			"steps": map[string]any{
				"step1": map[string]any{
					"step-scorer": map[string]any{"score": 0.0},
				},
			},
		})

		result := sa.GetAverageScores()
		steps, ok := result["steps"].(map[string]any)
		if !ok {
			t.Fatalf("steps key not found or not map: %v", result["steps"])
		}
		step1, ok := steps["step1"].(map[string]any)
		if !ok {
			t.Fatalf("step1 not found or not map: %v", steps["step1"])
		}
		avg, ok := step1["step-scorer"].(float64)
		if !ok {
			t.Fatalf("step-scorer not found or not float64: %v", step1["step-scorer"])
		}
		if math.Abs(avg-0.5) > 0.001 {
			t.Errorf("step-scorer average = %f, want 0.5", avg)
		}
	})
}

func TestScoreAccumulatorAddStepScores(t *testing.T) {
	t.Run("adds step scores directly", func(t *testing.T) {
		sa := NewScoreAccumulator()
		sa.AddStepScores(map[string]map[string]any{
			"step-a": {
				"scorer-x": map[string]any{"score": 0.5},
			},
		})
		sa.AddStepScores(map[string]map[string]any{
			"step-a": {
				"scorer-x": map[string]any{"score": 1.0},
			},
		})

		result := sa.GetAverageScores()
		steps, ok := result["steps"].(map[string]any)
		if !ok {
			t.Fatalf("steps key not found or not map: %v", result["steps"])
		}
		stepA, ok := steps["step-a"].(map[string]any)
		if !ok {
			t.Fatalf("step-a not found or not map: %v", steps["step-a"])
		}
		avg, ok := stepA["scorer-x"].(float64)
		if !ok {
			t.Fatalf("scorer-x not found or not float64: %v", stepA["scorer-x"])
		}
		if math.Abs(avg-0.75) > 0.001 {
			t.Errorf("scorer-x average = %f, want 0.75", avg)
		}
	})
}

func TestExtractScore(t *testing.T) {
	t.Run("extracts float64 score from map", func(t *testing.T) {
		result := extractScore(map[string]any{"score": 0.85})
		if math.Abs(result-0.85) > 0.001 {
			t.Errorf("got %f, want 0.85", result)
		}
	})

	t.Run("extracts int score from map", func(t *testing.T) {
		result := extractScore(map[string]any{"score": 1})
		if result != 1.0 {
			t.Errorf("got %f, want 1.0", result)
		}
	})

	t.Run("extracts direct float64", func(t *testing.T) {
		result := extractScore(0.5)
		if math.Abs(result-0.5) > 0.001 {
			t.Errorf("got %f, want 0.5", result)
		}
	})

	t.Run("returns 0 for unrecognized types", func(t *testing.T) {
		result := extractScore("not a score")
		if result != 0 {
			t.Errorf("got %f, want 0", result)
		}
	})

	t.Run("returns 0 for nil", func(t *testing.T) {
		result := extractScore(nil)
		if result != 0 {
			t.Errorf("got %f, want 0", result)
		}
	})
}

func TestGetAverageScore(t *testing.T) {
	t.Run("returns 0 for empty slice", func(t *testing.T) {
		result := getAverageScore([]float64{})
		if result != 0 {
			t.Errorf("got %f, want 0", result)
		}
	})

	t.Run("returns the value for single element", func(t *testing.T) {
		result := getAverageScore([]float64{0.7})
		if math.Abs(result-0.7) > 0.001 {
			t.Errorf("got %f, want 0.7", result)
		}
	})

	t.Run("computes correct average", func(t *testing.T) {
		result := getAverageScore([]float64{0.2, 0.4, 0.6, 0.8})
		if math.Abs(result-0.5) > 0.001 {
			t.Errorf("got %f, want 0.5", result)
		}
	})
}
