// Ported from: packages/core/src/evals/run/scorerAccumulator.ts
package run

// ScoreAccumulator accumulates scores from multiple scorer runs and computes
// averages. It handles both flat score results (for agents) and nested
// workflow/step score results.
//
// Corresponds to TS: export class ScoreAccumulator
type ScoreAccumulator struct {
	flatScores     map[string][]float64
	workflowScores map[string][]float64
	stepScores     map[string]map[string][]float64
}

// NewScoreAccumulator creates a new ScoreAccumulator.
func NewScoreAccumulator() *ScoreAccumulator {
	return &ScoreAccumulator{
		flatScores:     make(map[string][]float64),
		workflowScores: make(map[string][]float64),
		stepScores:     make(map[string]map[string][]float64),
	}
}

// AddScores adds scorer results to the accumulator. It auto-detects whether
// the results are flat (agent) or nested (workflow with steps).
//
// Corresponds to TS: addScores(scorerResults: Record<string, any>)
func (sa *ScoreAccumulator) AddScores(scorerResults map[string]any) {
	// Check if this is a workflow target with step scores.
	_, hasSteps := scorerResults["steps"]
	if hasSteps {
		sa.addNestedScores(scorerResults)
	} else {
		sa.addFlatScores(scorerResults)
	}
}

// addFlatScores adds flat (non-nested) scorer results.
//
// Corresponds to TS: private addFlatScores(scorerResults: Record<string, any>)
func (sa *ScoreAccumulator) addFlatScores(scorerResults map[string]any) {
	for scorerName, result := range scorerResults {
		if sa.flatScores[scorerName] == nil {
			sa.flatScores[scorerName] = []float64{}
		}
		score := extractScore(result)
		sa.flatScores[scorerName] = append(sa.flatScores[scorerName], score)
	}
}

// addNestedScores adds workflow-level and step-level scorer results.
//
// Corresponds to TS: private addNestedScores(scorerResults: Record<string, any>)
func (sa *ScoreAccumulator) addNestedScores(scorerResults map[string]any) {
	// Handle workflow-level scores.
	if workflowResults, ok := scorerResults["workflow"]; ok && workflowResults != nil {
		if wMap, ok := workflowResults.(map[string]any); ok {
			for scorerName, result := range wMap {
				if sa.workflowScores[scorerName] == nil {
					sa.workflowScores[scorerName] = []float64{}
				}
				score := extractScore(result)
				sa.workflowScores[scorerName] = append(sa.workflowScores[scorerName], score)
			}
		}
	}

	// Handle step-level scores.
	if stepsResults, ok := scorerResults["steps"]; ok && stepsResults != nil {
		if sMap, ok := stepsResults.(map[string]any); ok {
			for stepID, stepResults := range sMap {
				if sa.stepScores[stepID] == nil {
					sa.stepScores[stepID] = make(map[string][]float64)
				}
				if srMap, ok := stepResults.(map[string]any); ok {
					for scorerName, result := range srMap {
						if sa.stepScores[stepID][scorerName] == nil {
							sa.stepScores[stepID][scorerName] = []float64{}
						}
						score := extractScore(result)
						sa.stepScores[stepID][scorerName] = append(sa.stepScores[stepID][scorerName], score)
					}
				}
			}
		}
	}
}

// AddStepScores adds step-level scorer results directly.
//
// Corresponds to TS: addStepScores(stepScorerResults: Record<string, Record<string, any>>)
func (sa *ScoreAccumulator) AddStepScores(stepScorerResults map[string]map[string]any) {
	for stepID, stepResults := range stepScorerResults {
		if sa.stepScores[stepID] == nil {
			sa.stepScores[stepID] = make(map[string][]float64)
		}
		for scorerName, result := range stepResults {
			if sa.stepScores[stepID][scorerName] == nil {
				sa.stepScores[stepID][scorerName] = []float64{}
			}
			score := extractScore(result)
			sa.stepScores[stepID][scorerName] = append(sa.stepScores[stepID][scorerName], score)
		}
	}
}

// GetAverageScores computes and returns the average scores across all
// accumulated results. Returns a map that may include top-level scorer
// averages, a "workflow" key with workflow-level averages, and a "steps"
// key with per-step averages.
//
// Corresponds to TS: getAverageScores(): Record<string, any>
func (sa *ScoreAccumulator) GetAverageScores() map[string]any {
	result := make(map[string]any)

	// Flat scores.
	for scorerName, scoreArray := range sa.flatScores {
		result[scorerName] = getAverageScore(scoreArray)
	}

	// Workflow scores.
	if len(sa.workflowScores) > 0 {
		workflow := make(map[string]any)
		for scorerName, scoreArray := range sa.workflowScores {
			workflow[scorerName] = getAverageScore(scoreArray)
		}
		result["workflow"] = workflow
	}

	// Step scores.
	if len(sa.stepScores) > 0 {
		steps := make(map[string]any)
		for stepID, stepScorers := range sa.stepScores {
			stepMap := make(map[string]any)
			for scorerName, scoreArray := range stepScorers {
				stepMap[scorerName] = getAverageScore(scoreArray)
			}
			steps[stepID] = stepMap
		}
		result["steps"] = steps
	}

	return result
}

// getAverageScore computes the average of a float64 slice. Returns 0 for empty slices.
//
// Corresponds to TS: private getAverageScore(scoreArray: number[]): number
func getAverageScore(scoreArray []float64) float64 {
	if len(scoreArray) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range scoreArray {
		sum += v
	}
	return sum / float64(len(scoreArray))
}

// extractScore extracts the numeric score from a scorer result.
// Expects the result to have a "score" field (map[string]any) or be a struct
// with a Score field. Falls back to 0 if extraction fails.
func extractScore(result any) float64 {
	if m, ok := result.(map[string]any); ok {
		if s, ok := m["score"]; ok {
			switch v := s.(type) {
			case float64:
				return v
			case int:
				return float64(v)
			case int64:
				return float64(v)
			}
		}
	}
	// Try direct float64 assertion (for pre-extracted scores).
	if v, ok := result.(float64); ok {
		return v
	}
	return 0
}
