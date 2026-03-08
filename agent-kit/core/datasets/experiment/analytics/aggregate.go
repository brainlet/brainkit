// Ported from: packages/core/src/datasets/experiment/analytics/aggregate.ts
package analytics

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// ScoreRowData is a stub for evals/types ScoreRowData.
// STUB REASON: The real evals package has ScoreRowData but with different field
// types (e.g., EntityType as enum vs string). Kept as simplified local struct.
type ScoreRowData struct {
	ID         string   `json:"id"`
	ScorerID   string   `json:"scorerId"`
	Score      *float64 `json:"score"` // nil means error/missing
	EntityID   string   `json:"entityId,omitempty"`
	EntityType string   `json:"entityType,omitempty"`
	RunID      string   `json:"runId,omitempty"`
	Source     string   `json:"source,omitempty"`
}

// ComputeMean computes the arithmetic mean of an array of numbers.
// Returns 0 if the slice is empty.
func ComputeMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// ComputeScorerStats computes aggregate statistics for a set of scores.
//
// Metrics:
//   - ErrorRate: proportion of items with nil scores (errors)
//   - PassRate: proportion of scored items meeting threshold
//   - AvgScore: mean of non-nil scores
//
// passThreshold defaults to 0.5 if <= 0.
func ComputeScorerStats(scores []ScoreRowData, passThreshold float64) ScorerStats {
	if passThreshold <= 0 {
		passThreshold = 0.5
	}

	totalItems := len(scores)

	if totalItems == 0 {
		return ScorerStats{
			ErrorRate:  0,
			ErrorCount: 0,
			PassRate:   0,
			PassCount:  0,
			AvgScore:   0,
			ScoreCount: 0,
			TotalItems: 0,
		}
	}

	// Separate nil scores (errors) from valid scores
	var validScores []float64
	errorCount := 0

	for _, score := range scores {
		if score.Score == nil {
			errorCount++
		} else {
			validScores = append(validScores, *score.Score)
		}
	}

	scoreCount := len(validScores)
	errorRate := float64(errorCount) / float64(totalItems)

	// Pass rate is computed over items with valid scores only
	passCount := 0
	for _, s := range validScores {
		if s >= passThreshold {
			passCount++
		}
	}

	var passRate float64
	if scoreCount > 0 {
		passRate = float64(passCount) / float64(scoreCount)
	}

	// Average score excludes errors
	avgScore := ComputeMean(validScores)

	return ScorerStats{
		ErrorRate:  errorRate,
		ErrorCount: errorCount,
		PassRate:   passRate,
		PassCount:  passCount,
		AvgScore:   avgScore,
		ScoreCount: scoreCount,
		TotalItems: totalItems,
	}
}

// IsRegression determines if a score delta represents a regression.
//
// For "higher-is-better" (default): negative delta past threshold is a regression.
// For "lower-is-better": positive delta past threshold is a regression.
//
// Examples:
//
//	IsRegression(-0.1, 0.05, HigherIsBetter)  // true  (dropped more than 0.05)
//	IsRegression(-0.01, 0.05, HigherIsBetter) // false (within tolerance)
//	IsRegression(0.1, 0.05, LowerIsBetter)    // true  (increased more than 0.05)
func IsRegression(delta float64, threshold float64, direction ScoreDirection) bool {
	if direction == "" {
		direction = HigherIsBetter
	}

	if direction == HigherIsBetter {
		// Regression if score dropped below threshold
		// delta < -threshold means score dropped by more than threshold
		return delta < -threshold
	}
	// Regression if score increased above threshold
	// delta > threshold means score increased by more than threshold
	return delta > threshold
}
