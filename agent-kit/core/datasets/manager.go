// Ported from: packages/core/src/datasets/manager.ts
package datasets

import (
	"context"
	"errors"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	storagedatasets "github.com/brainlet/brainkit/agent-kit/core/storage/domains/datasets"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
	storageexperiments "github.com/brainlet/brainkit/agent-kit/core/storage/domains/experiments"
	"github.com/brainlet/brainkit/agent-kit/core/datasets/experiment/analytics"
)

// ============================================================================
// DatasetsManager
// ============================================================================

// DatasetsManager is the public API for managing datasets.
//
// Provides methods for dataset CRUD and cross-dataset experiment operations.
// Typically accessed via mastra.Datasets (Phase 4).
type DatasetsManager struct {
	mastra           Mastra
	datasetsStore    DatasetsStorage
	experimentsStore ExperimentsStorage
}

// NewDatasetsManager creates a new DatasetsManager.
func NewDatasetsManager(mastra Mastra) *DatasetsManager {
	return &DatasetsManager{
		mastra: mastra,
	}
}

// ---------------------------------------------------------------------------
// Lazy storage resolution
// ---------------------------------------------------------------------------

func (m *DatasetsManager) getDatasetsStore(ctx context.Context) (DatasetsStorage, error) {
	if m.datasetsStore != nil {
		return m.datasetsStore, nil
	}

	st := m.mastra.GetStorage()
	if st == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "DATASETS_STORAGE_NOT_CONFIGURED",
			Text:     "Storage not configured. Configure storage in Mastra instance.",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategoryUser,
		})
	}

	store := st.GetStore(storage.DomainDatasets)
	if store == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "DATASETS_STORE_NOT_AVAILABLE",
			Text:     "Datasets store not available. Ensure your storage adapter provides a datasets domain.",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategoryUser,
		})
	}

	ds, ok := store.(DatasetsStorage)
	if !ok {
		return nil, errors.New("datasets store does not implement DatasetsStorage interface")
	}

	m.datasetsStore = ds
	return ds, nil
}

func (m *DatasetsManager) getExperimentsStore(ctx context.Context) (ExperimentsStorage, error) {
	if m.experimentsStore != nil {
		return m.experimentsStore, nil
	}

	st := m.mastra.GetStorage()
	if st == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "DATASETS_STORAGE_NOT_CONFIGURED",
			Text:     "Storage not configured. Configure storage in Mastra instance.",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategoryUser,
		})
	}

	store := st.GetStore(storage.DomainExperiments)
	if store == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "EXPERIMENTS_STORE_NOT_AVAILABLE",
			Text:     "Experiments store not available. Ensure your storage adapter provides an experiments domain.",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategoryUser,
		})
	}

	es, ok := store.(ExperimentsStorage)
	if !ok {
		return nil, errors.New("experiments store does not implement ExperimentsStorage interface")
	}

	m.experimentsStore = es
	return es, nil
}

// ---------------------------------------------------------------------------
// Dataset CRUD
// ---------------------------------------------------------------------------

// CreateInput holds the fields for creating a new dataset.
type CreateInput struct {
	Name              string         `json:"name"`
	Description       string         `json:"description,omitempty"`
	InputSchema       map[string]any `json:"inputSchema,omitempty"`
	GroundTruthSchema map[string]any `json:"groundTruthSchema,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

// Create creates a new dataset.
func (m *DatasetsManager) Create(ctx context.Context, input CreateInput) (*Dataset, error) {
	store, err := m.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}

	var desc *string
	if input.Description != "" {
		desc = &input.Description
	}

	result, err := store.CreateDataset(ctx, storagedatasets.CreateDatasetInput{
		Name:              input.Name,
		Description:       desc,
		InputSchema:       input.InputSchema,
		GroundTruthSchema: input.GroundTruthSchema,
		Metadata:          input.Metadata,
	})
	if err != nil {
		return nil, err
	}

	return NewDataset(result.ID, m.mastra), nil
}

// Get gets an existing dataset by ID. Returns an error if the dataset does not exist.
func (m *DatasetsManager) Get(ctx context.Context, id string) (*Dataset, error) {
	store, err := m.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}

	record, err := store.GetDatasetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if record.ID == "" {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "DATASET_NOT_FOUND",
			Text:     "Dataset not found",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategoryUser,
		})
	}
	return NewDataset(id, m.mastra), nil
}

// ListArgs holds the arguments for listing datasets.
type ListArgs struct {
	Page    *int `json:"page,omitempty"`
	PerPage *int `json:"perPage,omitempty"`
}

// List lists all datasets with pagination.
func (m *DatasetsManager) List(ctx context.Context, args *ListArgs) (storagedatasets.ListDatasetsOutput, error) {
	store, err := m.getDatasetsStore(ctx)
	if err != nil {
		return storagedatasets.ListDatasetsOutput{}, err
	}

	page := 0
	perPage := 20
	if args != nil {
		if args.Page != nil {
			page = *args.Page
		}
		if args.PerPage != nil {
			perPage = *args.PerPage
		}
	}

	return store.ListDatasets(ctx, storagedatasets.ListDatasetsInput{
		Pagination: domains.StoragePagination{
			Page:    page,
			PerPage: perPage,
		},
	})
}

// Delete deletes a dataset by ID.
func (m *DatasetsManager) Delete(ctx context.Context, id string) error {
	store, err := m.getDatasetsStore(ctx)
	if err != nil {
		return err
	}
	return store.DeleteDataset(ctx, id)
}

// ---------------------------------------------------------------------------
// Cross-dataset experiment operations
// ---------------------------------------------------------------------------

// GetExperiment gets a specific experiment (run) by ID.
func (m *DatasetsManager) GetExperiment(ctx context.Context, experimentID string) (storageexperiments.Experiment, error) {
	expStore, err := m.getExperimentsStore(ctx)
	if err != nil {
		return storageexperiments.Experiment{}, err
	}
	return expStore.GetExperimentByID(ctx, experimentID)
}

// CompareExperimentsArgs holds the arguments for comparing experiments.
type CompareExperimentsArgs struct {
	ExperimentIDs []string `json:"experimentIds"`
	BaselineID    string   `json:"baselineId,omitempty"`
}

// CompareExperimentsItem is a per-item comparison result for the public API.
type CompareExperimentsItem struct {
	ItemID      string                              `json:"itemId"`
	Input       any                                 `json:"input"`
	GroundTruth any                                 `json:"groundTruth"`
	Results     map[string]*CompareExperimentResult `json:"results"`
}

// CompareExperimentResult holds one experiment's result for a single item.
type CompareExperimentResult struct {
	Output any                 `json:"output"`
	Scores map[string]*float64 `json:"scores"`
}

// CompareExperimentsOutput is the output of comparing experiments.
type CompareExperimentsOutput struct {
	BaselineID string                   `json:"baselineId"`
	Items      []CompareExperimentsItem `json:"items"`
}

// CompareExperiments compares two or more experiments.
//
// Uses the internal compareExperiments function for pairwise comparison,
// then enriches results with per-item input/groundTruth/output data.
func (m *DatasetsManager) CompareExperiments(ctx context.Context, args CompareExperimentsArgs) (*CompareExperimentsOutput, error) {
	experimentIDs := args.ExperimentIDs
	baselineID := args.BaselineID

	if len(experimentIDs) < 2 {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "COMPARE_INVALID_INPUT",
			Text:     "compareExperiments requires at least 2 experiment IDs.",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategoryUser,
		})
	}

	resolvedBaseline := baselineID
	if resolvedBaseline == "" {
		resolvedBaseline = experimentIDs[0]
	}

	var otherExperimentID string
	for _, id := range experimentIDs {
		if id != resolvedBaseline {
			otherExperimentID = id
			break
		}
	}
	if otherExperimentID == "" && len(experimentIDs) > 1 {
		otherExperimentID = experimentIDs[1]
	}

	// Get the storage interfaces needed by CompareExperiments
	st := m.mastra.GetStorage()
	if st == nil {
		return nil, errors.New("storage not configured")
	}

	expStoreRaw := st.GetStore(storage.DomainExperiments)
	scrStoreRaw := st.GetStore(storage.DomainScores)

	expStore, ok := expStoreRaw.(analytics.ExperimentsStorageCompat)
	if !ok {
		return nil, errors.New("experiments store does not implement ExperimentsStorageCompat")
	}
	scrStore, ok := scrStoreRaw.(analytics.ScoresStorageCompat)
	if !ok {
		return nil, errors.New("scores store does not implement ScoresStorageCompat")
	}

	internal, err := analytics.CompareExperiments(ctx, expStore, scrStore, analytics.CompareExperimentsConfig{
		ExperimentIDA: resolvedBaseline,
		ExperimentIDB: otherExperimentID,
	})
	if err != nil {
		return nil, err
	}

	// Load results for both runs to get input/groundTruth/output
	expMgrStore, err := m.getExperimentsStore(ctx)
	if err != nil {
		return nil, err
	}

	resultsA, err := expMgrStore.ListExperimentResults(ctx, storageexperiments.ListExperimentResultsInput{
		ExperimentID: resolvedBaseline,
		Pagination: domains.StoragePagination{
			Page:    0,
			PerPage: domains.PerPageDisabled,
		},
	})
	if err != nil {
		return nil, err
	}
	resultsB, err := expMgrStore.ListExperimentResults(ctx, storageexperiments.ListExperimentResultsInput{
		ExperimentID: otherExperimentID,
		Pagination: domains.StoragePagination{
			Page:    0,
			PerPage: domains.PerPageDisabled,
		},
	})
	if err != nil {
		return nil, err
	}

	// Build results maps by itemId
	resultsMapA := buildResultsMap(resultsA.Results)
	resultsMapB := buildResultsMap(resultsB.Results)

	// Transform internal items to MVP shape
	var items []CompareExperimentsItem
	for _, item := range internal.Items {
		resultA := resultsMapA[item.ItemID]
		resultB := resultsMapB[item.ItemID]

		var input any
		var groundTruth any
		if resultA != nil {
			input = resultA.Input
			groundTruth = resultA.GroundTruth
		} else if resultB != nil {
			input = resultB.Input
			groundTruth = resultB.GroundTruth
		}

		results := make(map[string]*CompareExperimentResult)
		if resultA != nil {
			results[resolvedBaseline] = &CompareExperimentResult{
				Output: resultA.Output,
				Scores: item.ScoresA,
			}
		}
		if resultB != nil {
			results[otherExperimentID] = &CompareExperimentResult{
				Output: resultB.Output,
				Scores: item.ScoresB,
			}
		}

		items = append(items, CompareExperimentsItem{
			ItemID:      item.ItemID,
			Input:       input,
			GroundTruth: groundTruth,
			Results:     results,
		})
	}

	return &CompareExperimentsOutput{
		BaselineID: resolvedBaseline,
		Items:      items,
	}, nil
}

// buildResultsMap builds a map from itemId to experiment result record.
func buildResultsMap(results []storageexperiments.ExperimentResult) map[string]*storageexperiments.ExperimentResult {
	m := make(map[string]*storageexperiments.ExperimentResult, len(results))
	for i := range results {
		m[results[i].ItemID] = &results[i]
	}
	return m
}
