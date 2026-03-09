// Ported from: packages/core/src/datasets/experiment/analytics/compare.ts
package analytics

import (
	"context"
	"errors"

	storage "github.com/brainlet/brainkit/agent-kit/core/storage"
)

// ---------------------------------------------------------------------------
// Local interface types
// ---------------------------------------------------------------------------

// Mastra is the narrow interface for the Mastra orchestrator used by analytics.
// Only GetStorage() is needed — callers resolve domain stores externally.
type Mastra interface {
	GetStorage() *storage.MastraCompositeStore
}

// ---------------------------------------------------------------------------
// Compatibility interfaces for storage access
// ---------------------------------------------------------------------------
// These interfaces abstract over the concrete storage types so that
// CompareExperiments doesn't depend on the domain package internals directly.
// TODO: Remove once a unified Mastra type with typed GetStore is available.

// ExperimentRecord holds the experiment fields needed by CompareExperiments.
type ExperimentRecord struct {
	ID             string `json:"id"`
	DatasetVersion *int   `json:"datasetVersion"`
}

// ExperimentResultRecord holds the experiment result fields needed by CompareExperiments.
type ExperimentResultRecord struct {
	ItemID      string `json:"itemId"`
	Input       any    `json:"input"`
	Output      any    `json:"output"`
	GroundTruth any    `json:"groundTruth"`
}

// ListExperimentResultsOutput is the output for listing experiment results.
type ListExperimentResultsOutput struct {
	Results []ExperimentResultRecord `json:"results"`
}

// ListScoresOutput is the output for listing scores.
type ListScoresOutput struct {
	Scores []ScoreRowData `json:"scores"`
}

// ExperimentsStorageCompat is the subset of ExperimentsStorage needed by CompareExperiments.
type ExperimentsStorageCompat interface {
	GetExperimentByID(ctx context.Context, id string) (*ExperimentRecord, error)
	ListExperimentResults(ctx context.Context, experimentID string) (*ListExperimentResultsOutput, error)
}

// ScoresStorageCompat is the subset of ScoresStorage needed by CompareExperiments.
type ScoresStorageCompat interface {
	ListScoresByRunID(ctx context.Context, runID string) (*ListScoresOutput, error)
}

// DefaultThreshold is used when no scorer-specific threshold is specified.
var DefaultThreshold = ScorerThreshold{
	Value:     0,
	Direction: HigherIsBetter,
}

// DefaultPassThreshold is the default pass threshold for computing pass rate.
const DefaultPassThreshold = 0.5

// CompareExperiments compares two experiments to detect score regressions.
//
// Returns per-scorer deltas and per-item score diffs.
//
// In the TypeScript source, this accesses storage via Mastra.getStorage().getStore().
// In Go, callers must provide the pre-resolved storage interfaces via the compat
// parameters since the domain storage types use `any` placeholder types.
func CompareExperiments(
	ctx context.Context,
	expStore ExperimentsStorageCompat,
	scrStore ScoresStorageCompat,
	config CompareExperimentsConfig,
) (*ComparisonResult, error) {
	experimentIDA := config.ExperimentIDA
	experimentIDB := config.ExperimentIDB
	thresholds := config.Thresholds
	if thresholds == nil {
		thresholds = make(map[string]ScorerThreshold)
	}
	var warnings []string

	// 1. Load both experiments
	experimentA, err := expStore.GetExperimentByID(ctx, experimentIDA)
	if err != nil {
		return nil, err
	}
	if experimentA == nil {
		return nil, errors.New("Experiment not found: " + experimentIDA)
	}

	experimentB, err := expStore.GetExperimentByID(ctx, experimentIDB)
	if err != nil {
		return nil, err
	}
	if experimentB == nil {
		return nil, errors.New("Experiment not found: " + experimentIDB)
	}

	// 2. Check version mismatch
	versionMismatch := !intPtrEqual(experimentA.DatasetVersion, experimentB.DatasetVersion)
	if versionMismatch {
		warnings = append(warnings, "Experiments have different dataset versions")
	}

	// 3. Load results for both experiments
	resultsA, err := expStore.ListExperimentResults(ctx, experimentIDA)
	if err != nil {
		return nil, err
	}
	resultsB, err := expStore.ListExperimentResults(ctx, experimentIDB)
	if err != nil {
		return nil, err
	}

	// 4. Load scores for both experiments
	scoresA, err := scrStore.ListScoresByRunID(ctx, experimentIDA)
	if err != nil {
		return nil, err
	}
	scoresB, err := scrStore.ListScoresByRunID(ctx, experimentIDB)
	if err != nil {
		return nil, err
	}

	// 5. Handle empty experiments
	if len(resultsA.Results) == 0 && len(resultsB.Results) == 0 {
		warnings = append(warnings, "Both experiments have no results.")
		return buildEmptyResult(experimentA, experimentB, versionMismatch, warnings), nil
	}
	if len(resultsA.Results) == 0 {
		warnings = append(warnings, "Experiment A has no results.")
	}
	if len(resultsB.Results) == 0 {
		warnings = append(warnings, "Experiment B has no results.")
	}

	// 6. Find overlapping items
	itemIDsA := make(map[string]struct{})
	for _, r := range resultsA.Results {
		itemIDsA[r.ItemID] = struct{}{}
	}
	itemIDsB := make(map[string]struct{})
	for _, r := range resultsB.Results {
		itemIDsB[r.ItemID] = struct{}{}
	}

	overlappingCount := 0
	for id := range itemIDsA {
		if _, ok := itemIDsB[id]; ok {
			overlappingCount++
		}
	}
	if overlappingCount == 0 {
		warnings = append(warnings, "No overlapping items between experiments.")
	}

	// 7. Group scores by scorer and item
	scoresMapA := groupScoresByScorerAndItem(scoresA.Scores)
	scoresMapB := groupScoresByScorerAndItem(scoresB.Scores)

	// 8. Find all unique scorers
	allScorerIDs := make(map[string]struct{})
	for k := range scoresMapA {
		allScorerIDs[k] = struct{}{}
	}
	for k := range scoresMapB {
		allScorerIDs[k] = struct{}{}
	}

	// 9. Build per-scorer comparison
	scorers := make(map[string]ScorerComparison)
	hasRegression := false

	for scorerID := range allScorerIDs {
		scorerScoresA := scoresMapA[scorerID]
		if scorerScoresA == nil {
			scorerScoresA = make(map[string]ScoreRowData)
		}
		scorerScoresB := scoresMapB[scorerID]
		if scorerScoresB == nil {
			scorerScoresB = make(map[string]ScoreRowData)
		}

		// Get scores as slices for stats computation
		scoresArrayA := mapValues(scorerScoresA)
		scoresArrayB := mapValues(scorerScoresB)

		// Get threshold config for this scorer
		thresholdConfig, exists := thresholds[scorerID]
		if !exists {
			thresholdConfig = DefaultThreshold
		}
		threshold := thresholdConfig.Value
		direction := thresholdConfig.Direction
		if direction == "" {
			direction = HigherIsBetter
		}

		// Compute stats
		statsA := ComputeScorerStats(scoresArrayA, DefaultPassThreshold)
		statsB := ComputeScorerStats(scoresArrayB, DefaultPassThreshold)

		// Compute delta and check regression
		delta := statsB.AvgScore - statsA.AvgScore
		regressed := IsRegression(delta, threshold, direction)

		if regressed {
			hasRegression = true
		}

		scorers[scorerID] = ScorerComparison{
			StatsA:    statsA,
			StatsB:    statsB,
			Delta:     delta,
			Regressed: regressed,
			Threshold: threshold,
		}
	}

	// 10. Build per-item comparison
	allItemIDs := make(map[string]struct{})
	for id := range itemIDsA {
		allItemIDs[id] = struct{}{}
	}
	for id := range itemIDsB {
		allItemIDs[id] = struct{}{}
	}

	var items []ItemComparison
	for itemID := range allItemIDs {
		_, inA := itemIDsA[itemID]
		_, inB := itemIDsB[itemID]
		inBothExperiments := inA && inB

		// Build scores for this item
		itemScoresA := make(map[string]*float64)
		itemScoresB := make(map[string]*float64)

		for scorerID := range allScorerIDs {
			if scoreA, ok := scoresMapA[scorerID][itemID]; ok {
				itemScoresA[scorerID] = scoreA.Score
			} else {
				itemScoresA[scorerID] = nil
			}

			if scoreB, ok := scoresMapB[scorerID][itemID]; ok {
				itemScoresB[scorerID] = scoreB.Score
			} else {
				itemScoresB[scorerID] = nil
			}
		}

		items = append(items, ItemComparison{
			ItemID:            itemID,
			InBothExperiments: inBothExperiments,
			ScoresA:           itemScoresA,
			ScoresB:           itemScoresB,
		})
	}

	return &ComparisonResult{
		ExperimentA: ExperimentRef{
			ID:             experimentA.ID,
			DatasetVersion: experimentA.DatasetVersion,
		},
		ExperimentB: ExperimentRef{
			ID:             experimentB.ID,
			DatasetVersion: experimentB.DatasetVersion,
		},
		VersionMismatch: versionMismatch,
		HasRegression:   hasRegression,
		Scorers:         scorers,
		Items:           items,
		Warnings:        warnings,
	}, nil
}

// groupScoresByScorerAndItem groups scores by scorer ID, then by item ID.
func groupScoresByScorerAndItem(scores []ScoreRowData) map[string]map[string]ScoreRowData {
	result := make(map[string]map[string]ScoreRowData)

	for _, score := range scores {
		scorerID := score.ScorerID
		itemID := score.EntityID // entityId is the item ID for experiment scores

		if result[scorerID] == nil {
			result[scorerID] = make(map[string]ScoreRowData)
		}
		result[scorerID][itemID] = score
	}

	return result
}

// buildEmptyResult builds an empty comparison result for edge cases.
func buildEmptyResult(
	experimentA, experimentB *ExperimentRecord,
	versionMismatch bool,
	warnings []string,
) *ComparisonResult {
	return &ComparisonResult{
		ExperimentA: ExperimentRef{
			ID:             experimentA.ID,
			DatasetVersion: experimentA.DatasetVersion,
		},
		ExperimentB: ExperimentRef{
			ID:             experimentB.ID,
			DatasetVersion: experimentB.DatasetVersion,
		},
		VersionMismatch: versionMismatch,
		HasRegression:   false,
		Scorers:         make(map[string]ScorerComparison),
		Items:           nil,
		Warnings:        warnings,
	}
}

// intPtrEqual compares two *int values for equality.
func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// mapValues extracts all values from a map into a slice.
func mapValues(m map[string]ScoreRowData) []ScoreRowData {
	result := make([]ScoreRowData, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}
