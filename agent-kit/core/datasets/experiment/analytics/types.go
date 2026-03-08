// Ported from: packages/core/src/datasets/experiment/analytics/types.ts
package analytics

// ============================================================================
// Per-Scorer Statistics
// ============================================================================

// ScorerStats holds aggregate statistics for a single scorer across an experiment.
type ScorerStats struct {
	// ErrorRate is items with null score / total items.
	ErrorRate float64 `json:"errorRate"`
	// ErrorCount is count of items with null score.
	ErrorCount int `json:"errorCount"`
	// PassRate is items >= threshold / items with scores.
	PassRate float64 `json:"passRate"`
	// PassCount is count of items that passed threshold.
	PassCount int `json:"passCount"`
	// AvgScore is mean of non-null scores.
	AvgScore float64 `json:"avgScore"`
	// ScoreCount is count of items with non-null scores.
	ScoreCount int `json:"scoreCount"`
	// TotalItems is total items evaluated by this scorer.
	TotalItems int `json:"totalItems"`
}

// ============================================================================
// Comparison Types
// ============================================================================

// ScorerComparison is per-scorer comparison between two experiments.
type ScorerComparison struct {
	// StatsA is stats from experiment A (baseline).
	StatsA ScorerStats `json:"statsA"`
	// StatsB is stats from experiment B (candidate).
	StatsB ScorerStats `json:"statsB"`
	// Delta is avgScore difference: StatsB.AvgScore - StatsA.AvgScore.
	Delta float64 `json:"delta"`
	// Regressed indicates whether this scorer regressed (delta below threshold).
	Regressed bool `json:"regressed"`
	// Threshold is the threshold used for regression detection.
	Threshold float64 `json:"threshold"`
}

// ItemComparison is per-item comparison showing score differences.
type ItemComparison struct {
	// ItemID is the dataset item ID.
	ItemID string `json:"itemId"`
	// InBothExperiments indicates whether item exists in both experiments.
	InBothExperiments bool `json:"inBothExperiments"`
	// ScoresA is scores from experiment A by scorer ID (nil value means no score).
	ScoresA map[string]*float64 `json:"scoresA"`
	// ScoresB is scores from experiment B by scorer ID (nil value means no score).
	ScoresB map[string]*float64 `json:"scoresB"`
}

// ComparisonResult is top-level comparison result.
type ComparisonResult struct {
	// ExperimentA is experiment A metadata.
	ExperimentA ExperimentRef `json:"experimentA"`
	// ExperimentB is experiment B metadata.
	ExperimentB ExperimentRef `json:"experimentB"`
	// VersionMismatch is true if experiments used different dataset versions.
	VersionMismatch bool `json:"versionMismatch"`
	// HasRegression is true if any scorer regressed (for CI quick check).
	HasRegression bool `json:"hasRegression"`
	// Scorers is per-scorer comparison results, keyed by scorer ID.
	Scorers map[string]ScorerComparison `json:"scorers"`
	// Items is per-item comparison details.
	Items []ItemComparison `json:"items"`
	// Warnings is warning messages (e.g., version mismatch, no overlap).
	Warnings []string `json:"warnings"`
}

// ExperimentRef holds reference metadata for an experiment in a comparison.
type ExperimentRef struct {
	// ID is the experiment ID.
	ID string `json:"id"`
	// DatasetVersion is the dataset version (nil if unknown).
	DatasetVersion *int `json:"datasetVersion"`
}

// ============================================================================
// Configuration Types
// ============================================================================

// ScorerThreshold is threshold configuration for a single scorer.
type ScorerThreshold struct {
	// Value is absolute threshold value for regression detection.
	Value float64 `json:"value"`
	// Direction is score direction: "higher-is-better" (default) or "lower-is-better".
	Direction ScoreDirection `json:"direction,omitempty"`
}

// ScoreDirection represents the direction of scoring.
type ScoreDirection string

const (
	// HigherIsBetter means a higher score is better (default).
	HigherIsBetter ScoreDirection = "higher-is-better"
	// LowerIsBetter means a lower score is better.
	LowerIsBetter ScoreDirection = "lower-is-better"
)

// CompareExperimentsConfig is configuration for compareExperiments function.
type CompareExperimentsConfig struct {
	// ExperimentIDA is ID of experiment A (baseline).
	ExperimentIDA string `json:"experimentIdA"`
	// ExperimentIDB is ID of experiment B (candidate).
	ExperimentIDB string `json:"experimentIdB"`
	// Thresholds is per-scorer thresholds for regression detection.
	// Key is scorer ID, value is threshold config.
	// Default when not specified: { Value: 0, Direction: HigherIsBetter }.
	Thresholds map[string]ScorerThreshold `json:"thresholds,omitempty"`
}
