// Ported from: packages/core/src/datasets/dataset.ts
package datasets

import (
	"context"
	"errors"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/datasets/experiment"
	"github.com/brainlet/brainkit/agent-kit/core/mastra"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
)

// ---------------------------------------------------------------------------
// Cross-package interfaces (real imports, no circular dependency)
// ---------------------------------------------------------------------------

// Mastra is the narrow interface for the Mastra orchestrator used by datasets.
// No circular dependency: mastra does not import datasets.
// Ported from: packages/core/src/datasets/dataset.ts — uses mastra instance
type Mastra interface {
	GetStorage() *storage.MastraCompositeStore
	GetScorerByID(id string) mastra.MastraScorer
	GetAgentByID(id string) (mastra.Agent, error)
	GetAgent(name string) (mastra.Agent, error)
	GetWorkflowByID(id string) (mastra.AnyWorkflow, error)
	GetWorkflow(name string) (mastra.AnyWorkflow, error)
}

// MastraCompositeStore is re-exported from the storage package.
// No circular dependency: storage does not import datasets.
type MastraCompositeStore = storage.MastraCompositeStore

// DatasetsStorage is a stub for storage/domains/datasets DatasetsStorage.
// STUB REASON: The real storage/domains/datasets.DatasetsStorage is itself a stub
// (type alias for any). This interface defines the expected contract locally.
type DatasetsStorage interface {
	GetDatasetByID(ctx context.Context, id string) (DatasetRecord, error)
	UpdateDataset(ctx context.Context, args map[string]any) (DatasetRecord, error)
	AddItem(ctx context.Context, args map[string]any) (DatasetItem, error)
	BatchInsertItems(ctx context.Context, args map[string]any) ([]DatasetItem, error)
	GetItemByID(ctx context.Context, args map[string]any) (DatasetItem, error)
	ListItems(ctx context.Context, args map[string]any) (any, error)
	GetItemsByVersion(ctx context.Context, args map[string]any) ([]DatasetItem, error)
	UpdateItem(ctx context.Context, args map[string]any) (DatasetItem, error)
	DeleteItem(ctx context.Context, args map[string]any) error
	BatchDeleteItems(ctx context.Context, args map[string]any) error
	ListDatasetVersions(ctx context.Context, args map[string]any) (ListVersionsOutput, error)
	GetItemHistory(ctx context.Context, itemID string) ([]DatasetItemRow, error)
}

// ExperimentsStorage is a stub for storage/domains/experiments ExperimentsStorage.
// STUB REASON: Same as DatasetsStorage — the real type is a stub alias for any.
type ExperimentsStorage interface {
	CreateExperiment(ctx context.Context, input map[string]any) (ExperimentRecord, error)
	ListExperiments(ctx context.Context, args map[string]any) (any, error)
	GetExperimentByID(ctx context.Context, id string) (any, error)
	ListExperimentResults(ctx context.Context, args map[string]any) (any, error)
	DeleteExperiment(ctx context.Context, id string) error
}

// DatasetRecord is a stub for storage/types DatasetRecord.
// STUB REASON: The real storage.DatasetRecord is a struct with typed fields.
// Replacing would require updating all code that constructs/accesses these as maps.
type DatasetRecord = map[string]any

// DatasetItem is a stub for storage/types DatasetItem.
// STUB REASON: Same as DatasetRecord — real type is a struct.
type DatasetItem = map[string]any

// DatasetItemRow is a stub for storage/types DatasetItemRow.
// STUB REASON: Same as DatasetRecord — real type is a struct.
type DatasetItemRow = map[string]any

// DatasetVersion is a stub for storage/types DatasetVersion.
// STUB REASON: Same as DatasetRecord — real type is a struct.
type DatasetVersion = map[string]any

// ExperimentRecord is a stub for experiment record.
// STUB REASON: Same as DatasetRecord — real type is a struct.
type ExperimentRecord = map[string]any

// ListVersionsOutput is a stub for list versions output.
// Defined locally as this is a datasets-specific output type.
type ListVersionsOutput struct {
	Versions   []DatasetVersion `json:"versions"`
	Pagination PaginationInfo   `json:"pagination"`
}

// PaginationInfo is a stub for pagination metadata.
// Defined locally as a simplified version of the storage pagination types.
type PaginationInfo struct {
	Total   int  `json:"total"`
	Page    int  `json:"page"`
	PerPage any  `json:"perPage"` // int or false
	HasMore bool `json:"hasMore"`
}

// ============================================================================
// Dataset
// ============================================================================

// Dataset is the public API for interacting with a single dataset.
//
// Provides methods for item CRUD, versioning, and experiment management.
// Obtained via DatasetsManager.Get() or DatasetsManager.Create().
type Dataset struct {
	// ID is the dataset ID.
	ID string

	mastra          Mastra
	datasetsStore   DatasetsStorage
	experimentsStore ExperimentsStorage
}

// NewDataset creates a new Dataset handle.
func NewDataset(id string, mastra Mastra) *Dataset {
	return &Dataset{
		ID:     id,
		mastra: mastra,
	}
}

// ---------------------------------------------------------------------------
// Lazy storage resolution
// ---------------------------------------------------------------------------

func (d *Dataset) getDatasetsStore(ctx context.Context) (DatasetsStorage, error) {
	if d.datasetsStore != nil {
		return d.datasetsStore, nil
	}

	cs := d.mastra.GetStorage()
	if cs == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "DATASETS_STORAGE_NOT_CONFIGURED",
			Text:     "Storage not configured. Configure storage in Mastra instance.",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategoryUser,
		})
	}

	store := cs.GetStore(storage.DomainDatasets)
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

	d.datasetsStore = ds
	return ds, nil
}

func (d *Dataset) getExperimentsStore(ctx context.Context) (ExperimentsStorage, error) {
	if d.experimentsStore != nil {
		return d.experimentsStore, nil
	}

	cs := d.mastra.GetStorage()
	if cs == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "DATASETS_STORAGE_NOT_CONFIGURED",
			Text:     "Storage not configured. Configure storage in Mastra instance.",
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategoryUser,
		})
	}

	store := cs.GetStore(storage.DomainExperiments)
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

	d.experimentsStore = es
	return es, nil
}

// ---------------------------------------------------------------------------
// Dataset metadata
// ---------------------------------------------------------------------------

// GetDetails gets the full dataset record from storage.
func (d *Dataset) GetDetails(ctx context.Context) (DatasetRecord, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}

	record, err := store.GetDatasetByID(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "DATASET_NOT_FOUND",
			Text:     "Dataset not found: " + d.ID,
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategoryUser,
		})
	}
	return record, nil
}

// UpdateInput holds the fields for updating dataset metadata.
type UpdateInput struct {
	Name             *string        `json:"name,omitempty"`
	Description      *string        `json:"description,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	InputSchema      any            `json:"inputSchema,omitempty"`
	GroundTruthSchema any           `json:"groundTruthSchema,omitempty"`
}

// Update updates dataset metadata and/or schemas.
func (d *Dataset) Update(ctx context.Context, input UpdateInput) (DatasetRecord, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}

	args := map[string]any{
		"id": d.ID,
	}
	if input.Name != nil {
		args["name"] = *input.Name
	}
	if input.Description != nil {
		args["description"] = *input.Description
	}
	if input.Metadata != nil {
		args["metadata"] = input.Metadata
	}
	if input.InputSchema != nil {
		args["inputSchema"] = input.InputSchema
	}
	if input.GroundTruthSchema != nil {
		args["groundTruthSchema"] = input.GroundTruthSchema
	}

	return store.UpdateDataset(ctx, args)
}

// ---------------------------------------------------------------------------
// Item CRUD
// ---------------------------------------------------------------------------

// AddItemInput holds the fields for adding a single item.
type AddItemInput struct {
	Input       any            `json:"input"`
	GroundTruth any            `json:"groundTruth,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// AddItem adds a single item to the dataset.
func (d *Dataset) AddItem(ctx context.Context, input AddItemInput) (DatasetItem, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}
	return store.AddItem(ctx, map[string]any{
		"datasetId":   d.ID,
		"input":       input.Input,
		"groundTruth": input.GroundTruth,
		"metadata":    input.Metadata,
	})
}

// AddItemsInput holds the fields for adding multiple items.
type AddItemsInput struct {
	Items []AddItemInput `json:"items"`
}

// AddItems adds multiple items to the dataset in bulk.
func (d *Dataset) AddItems(ctx context.Context, input AddItemsInput) ([]DatasetItem, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}
	return store.BatchInsertItems(ctx, map[string]any{
		"datasetId": d.ID,
		"items":     input.Items,
	})
}

// GetItemArgs holds the arguments for getting a single item.
type GetItemArgs struct {
	ItemID  string `json:"itemId"`
	Version *int   `json:"version,omitempty"`
}

// GetItem gets a single item by ID, optionally at a specific version.
func (d *Dataset) GetItem(ctx context.Context, args GetItemArgs) (DatasetItem, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}
	return store.GetItemByID(ctx, map[string]any{
		"id":             args.ItemID,
		"datasetVersion": args.Version,
	})
}

// ListItemsArgs holds the arguments for listing items.
type ListItemsArgs struct {
	Version *int   `json:"version,omitempty"`
	Page    *int   `json:"page,omitempty"`
	PerPage *int   `json:"perPage,omitempty"`
	Search  string `json:"search,omitempty"`
}

// ListItems lists items in the dataset, optionally at a specific version.
func (d *Dataset) ListItems(ctx context.Context, args *ListItemsArgs) (any, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}

	if args != nil && args.Version != nil {
		return store.GetItemsByVersion(ctx, map[string]any{
			"datasetId": d.ID,
			"version":   *args.Version,
		})
	}

	page := 0
	perPage := 20
	var search string
	if args != nil {
		if args.Page != nil {
			page = *args.Page
		}
		if args.PerPage != nil {
			perPage = *args.PerPage
		}
		search = args.Search
	}

	return store.ListItems(ctx, map[string]any{
		"datasetId": d.ID,
		"search":    search,
		"pagination": map[string]any{
			"page":    page,
			"perPage": perPage,
		},
	})
}

// UpdateItemInput holds the fields for updating an item.
type UpdateItemInput struct {
	ItemID      string         `json:"itemId"`
	Input       any            `json:"input,omitempty"`
	GroundTruth any            `json:"groundTruth,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// UpdateItem updates an existing item in the dataset.
func (d *Dataset) UpdateItem(ctx context.Context, input UpdateItemInput) (DatasetItem, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}
	return store.UpdateItem(ctx, map[string]any{
		"id":          input.ItemID,
		"datasetId":   d.ID,
		"input":       input.Input,
		"groundTruth": input.GroundTruth,
		"metadata":    input.Metadata,
	})
}

// DeleteItem deletes a single item from the dataset.
func (d *Dataset) DeleteItem(ctx context.Context, itemID string) error {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return err
	}
	return store.DeleteItem(ctx, map[string]any{
		"id":        itemID,
		"datasetId": d.ID,
	})
}

// DeleteItems deletes multiple items from the dataset in bulk.
func (d *Dataset) DeleteItems(ctx context.Context, itemIDs []string) error {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return err
	}
	return store.BatchDeleteItems(ctx, map[string]any{
		"datasetId": d.ID,
		"itemIds":   itemIDs,
	})
}

// ---------------------------------------------------------------------------
// Versioning
// ---------------------------------------------------------------------------

// ListVersionsArgs holds the arguments for listing versions.
type ListVersionsArgs struct {
	Page    *int `json:"page,omitempty"`
	PerPage *int `json:"perPage,omitempty"`
}

// ListVersions lists all versions of this dataset.
func (d *Dataset) ListVersions(ctx context.Context, args *ListVersionsArgs) (*ListVersionsOutput, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
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

	out, err := store.ListDatasetVersions(ctx, map[string]any{
		"datasetId": d.ID,
		"pagination": map[string]any{
			"page":    page,
			"perPage": perPage,
		},
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetItemHistory gets the full SCD-2 history of a specific item across all dataset versions.
func (d *Dataset) GetItemHistory(ctx context.Context, itemID string) ([]DatasetItemRow, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}
	return store.GetItemHistory(ctx, itemID)
}

// ---------------------------------------------------------------------------
// Experiments
// ---------------------------------------------------------------------------

// StartExperiment runs an experiment on this dataset and waits for completion.
//
// This wraps experiment.RunExperiment with the dataset's ID injected.
// TODO: Call experiment.RunExperiment once the runner is fully wired up.
func (d *Dataset) StartExperiment(ctx context.Context, config experiment.StartExperimentConfig) (*experiment.ExperimentSummary, error) {
	// Convert StartExperimentConfig to ExperimentConfig with datasetId injected
	fullConfig := experiment.ExperimentConfig{
		DatasetID:      d.ID,
		TargetType:     config.TargetType,
		TargetID:       config.TargetID,
		Task:           config.Task,
		Scorers:        config.Scorers,
		Version:        config.Version,
		MaxConcurrency: config.MaxConcurrency,
		ItemTimeout:    config.ItemTimeout,
		MaxRetries:     config.MaxRetries,
		Name:           config.Name,
		Description:    config.Description,
		Metadata:       config.Metadata,
	}

	// TODO: Call experiment.RunExperiment(ctx, mastra, fullConfig) once
	// the experiment runner is fully wired with Mastra adapter compatibility.
	_ = fullConfig
	return nil, errors.New("StartExperiment: experiment runner not yet wired — TODO: implement RunExperiment integration")
}

// StartExperimentAsyncResult is the result of an async experiment start.
type StartExperimentAsyncResult struct {
	ExperimentID string `json:"experimentId"`
	Status       string `json:"status"`
}

// StartExperimentAsync starts an experiment asynchronously (fire-and-forget).
// Returns immediately with the experiment ID and pending status.
func (d *Dataset) StartExperimentAsync(ctx context.Context, config experiment.StartExperimentConfig) (*StartExperimentAsyncResult, error) {
	expStore, err := d.getExperimentsStore(ctx)
	if err != nil {
		return nil, err
	}

	dsStore, err := d.getDatasetsStore(ctx)
	if err != nil {
		return nil, err
	}

	dataset, err := dsStore.GetDatasetByID(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	if dataset == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "DATASET_NOT_FOUND",
			Text:     "Dataset not found: " + d.ID,
			Domain:   mastraerror.ErrorDomainStorage,
			Category: mastraerror.ErrorCategoryUser,
		})
	}

	targetType := config.TargetType
	if targetType == "" {
		targetType = experiment.TargetTypeAgent
	}
	targetID := config.TargetID
	if targetID == "" {
		targetID = "inline"
	}

	run, err := expStore.CreateExperiment(ctx, map[string]any{
		"datasetId":      d.ID,
		"datasetVersion": dataset["version"],
		"targetType":     string(targetType),
		"targetId":       targetID,
		"totalItems":     0,
		"name":           config.Name,
		"description":    config.Description,
		"metadata":       config.Metadata,
	})
	if err != nil {
		return nil, err
	}

	experimentID, _ := run["id"].(string)

	// Fire-and-forget — errors are silently caught
	// TODO: Launch goroutine calling experiment.RunExperiment once wired.
	// go func() {
	//     _ = experiment.RunExperiment(ctx, d.mastra, experiment.ExperimentConfig{
	//         DatasetID:    d.ID,
	//         ExperimentID: experimentID,
	//         ...config,
	//     })
	// }()

	return &StartExperimentAsyncResult{
		ExperimentID: experimentID,
		Status:       "pending",
	}, nil
}

// ListExperimentsArgs holds the arguments for listing experiments.
type ListExperimentsArgs struct {
	Page    *int `json:"page,omitempty"`
	PerPage *int `json:"perPage,omitempty"`
}

// ListExperiments lists all experiments (runs) for this dataset.
func (d *Dataset) ListExperiments(ctx context.Context, args *ListExperimentsArgs) (any, error) {
	expStore, err := d.getExperimentsStore(ctx)
	if err != nil {
		return nil, err
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

	return expStore.ListExperiments(ctx, map[string]any{
		"datasetId": d.ID,
		"pagination": map[string]any{
			"page":    page,
			"perPage": perPage,
		},
	})
}

// GetExperiment gets a specific experiment (run) by ID.
func (d *Dataset) GetExperiment(ctx context.Context, experimentID string) (any, error) {
	expStore, err := d.getExperimentsStore(ctx)
	if err != nil {
		return nil, err
	}
	return expStore.GetExperimentByID(ctx, experimentID)
}

// ListExperimentResultsArgs holds the arguments for listing experiment results.
type ListExperimentResultsArgs struct {
	ExperimentID string `json:"experimentId"`
	Page         *int   `json:"page,omitempty"`
	PerPage      *int   `json:"perPage,omitempty"`
}

// ListExperimentResults lists results for a specific experiment.
func (d *Dataset) ListExperimentResults(ctx context.Context, args ListExperimentResultsArgs) (any, error) {
	expStore, err := d.getExperimentsStore(ctx)
	if err != nil {
		return nil, err
	}

	page := 0
	perPage := 20
	if args.Page != nil {
		page = *args.Page
	}
	if args.PerPage != nil {
		perPage = *args.PerPage
	}

	return expStore.ListExperimentResults(ctx, map[string]any{
		"experimentId": args.ExperimentID,
		"pagination": map[string]any{
			"page":    page,
			"perPage": perPage,
		},
	})
}

// DeleteExperiment deletes an experiment (run) by ID.
func (d *Dataset) DeleteExperiment(ctx context.Context, experimentID string) error {
	expStore, err := d.getExperimentsStore(ctx)
	if err != nil {
		return err
	}
	return expStore.DeleteExperiment(ctx, experimentID)
}
