// Ported from: packages/core/src/datasets/dataset.ts
package datasets

import (
	"context"
	"errors"

	"github.com/brainlet/brainkit/agent-kit/core/datasets/experiment"
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/mastra"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	storagedatasets "github.com/brainlet/brainkit/agent-kit/core/storage/domains/datasets"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
	storageexperiments "github.com/brainlet/brainkit/agent-kit/core/storage/domains/experiments"
)

// ---------------------------------------------------------------------------
// Cross-package types (real imports, no circular dependency)
// ---------------------------------------------------------------------------

// Mastra is the narrow interface for the Mastra orchestrator used by datasets.
// Ported from: packages/core/src/datasets/dataset.ts — uses mastra instance.
type Mastra interface {
	GetStorage() *storage.MastraCompositeStore
	GetScorerByID(id string) mastra.MastraScorer
	GetAgentByID(id string) (mastra.Agent, error)
	GetAgent(name string) (mastra.Agent, error)
	GetWorkflowByID(id string) (mastra.AnyWorkflow, error)
	GetWorkflow(name string) (mastra.AnyWorkflow, error)
}

// MastraCompositeStore is re-exported from the storage package.
type MastraCompositeStore = storage.MastraCompositeStore

// DatasetsStorage is the real storage interface from storage/domains/datasets.
type DatasetsStorage = storagedatasets.DatasetsStorage

// ExperimentsStorage is the real storage interface from storage/domains/experiments.
type ExperimentsStorage = storageexperiments.ExperimentsStorage

// DatasetRecord is a dataset record from storage.
type DatasetRecord = storagedatasets.DatasetRecord

// DatasetItem is an item within a dataset from storage.
type DatasetItem = storagedatasets.DatasetItem

// DatasetItemRow is the raw database row for a dataset item (includes versioning fields).
type DatasetItemRow = storagedatasets.DatasetItemRow

// DatasetVersion represents a dataset version record.
type DatasetVersion = storagedatasets.DatasetVersion

// ExperimentRecord is an experiment record from storage.
type ExperimentRecord = storageexperiments.Experiment

// ListVersionsOutput is the output for listing dataset versions.
type ListVersionsOutput = storagedatasets.ListDatasetVersionsOutput

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

	mastra           Mastra
	datasetsStore    DatasetsStorage
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
		return DatasetRecord{}, err
	}

	record, err := store.GetDatasetByID(ctx, d.ID)
	if err != nil {
		return DatasetRecord{}, err
	}
	if record.ID == "" {
		return DatasetRecord{}, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
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
	Name              *string        `json:"name,omitempty"`
	Description       *string        `json:"description,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	InputSchema       map[string]any `json:"inputSchema,omitempty"`
	GroundTruthSchema map[string]any `json:"groundTruthSchema,omitempty"`
}

// Update updates dataset metadata and/or schemas.
func (d *Dataset) Update(ctx context.Context, input UpdateInput) (DatasetRecord, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return DatasetRecord{}, err
	}

	return store.UpdateDataset(ctx, storagedatasets.UpdateDatasetInput{
		ID:                d.ID,
		Name:              input.Name,
		Description:       input.Description,
		Metadata:          input.Metadata,
		InputSchema:       input.InputSchema,
		GroundTruthSchema: input.GroundTruthSchema,
	})
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
		return DatasetItem{}, err
	}
	return store.AddItem(ctx, storagedatasets.AddDatasetItemInput{
		DatasetID:   d.ID,
		Input:       input.Input,
		GroundTruth: input.GroundTruth,
		Metadata:    input.Metadata,
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

	batchItems := make([]domains.BatchInsertItemInput, len(input.Items))
	for i, item := range input.Items {
		batchItems[i] = domains.BatchInsertItemInput{
			Input:       item.Input,
			GroundTruth: item.GroundTruth,
			Metadata:    item.Metadata,
		}
	}

	return store.BatchInsertItems(ctx, storagedatasets.BatchInsertItemsInput{
		DatasetID: d.ID,
		Items:     batchItems,
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
		return DatasetItem{}, err
	}
	return store.GetItemByID(ctx, storagedatasets.GetItemByIDArgs{
		ID:             args.ItemID,
		DatasetVersion: args.Version,
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
func (d *Dataset) ListItems(ctx context.Context, args *ListItemsArgs) (storagedatasets.ListDatasetItemsOutput, error) {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return storagedatasets.ListDatasetItemsOutput{}, err
	}

	if args != nil && args.Version != nil {
		items, err := store.GetItemsByVersion(ctx, storagedatasets.GetItemsByVersionArgs{
			DatasetID: d.ID,
			Version:   *args.Version,
		})
		if err != nil {
			return storagedatasets.ListDatasetItemsOutput{}, err
		}
		return storagedatasets.ListDatasetItemsOutput{
			Items: items,
		}, nil
	}

	page := 0
	perPage := 20
	var search *string
	if args != nil {
		if args.Page != nil {
			page = *args.Page
		}
		if args.PerPage != nil {
			perPage = *args.PerPage
		}
		if args.Search != "" {
			search = &args.Search
		}
	}

	return store.ListItems(ctx, storagedatasets.ListDatasetItemsInput{
		DatasetID: d.ID,
		Search:    search,
		Pagination: domains.StoragePagination{
			Page:    page,
			PerPage: perPage,
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
		return DatasetItem{}, err
	}
	return store.UpdateItem(ctx, storagedatasets.UpdateDatasetItemInput{
		ID:          input.ItemID,
		DatasetID:   d.ID,
		Input:       input.Input,
		GroundTruth: input.GroundTruth,
		Metadata:    input.Metadata,
	})
}

// DeleteItem deletes a single item from the dataset.
func (d *Dataset) DeleteItem(ctx context.Context, itemID string) error {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return err
	}
	return store.DeleteItem(ctx, storagedatasets.DeleteItemArgs{
		ID:        itemID,
		DatasetID: d.ID,
	})
}

// DeleteItems deletes multiple items from the dataset in bulk.
func (d *Dataset) DeleteItems(ctx context.Context, itemIDs []string) error {
	store, err := d.getDatasetsStore(ctx)
	if err != nil {
		return err
	}
	return store.BatchDeleteItems(ctx, storagedatasets.BatchDeleteItemsInput{
		DatasetID: d.ID,
		ItemIDs:   itemIDs,
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

	out, err := store.ListDatasetVersions(ctx, storagedatasets.ListDatasetVersionsInput{
		DatasetID: d.ID,
		Pagination: domains.StoragePagination{
			Page:    page,
			PerPage: perPage,
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
	if dataset.ID == "" {
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

	datasetID := d.ID
	datasetVersion := dataset.Version

	run, err := expStore.CreateExperiment(ctx, storageexperiments.CreateExperimentInput{
		DatasetID:      &datasetID,
		DatasetVersion: &datasetVersion,
		TargetType:     domains.TargetType(targetType),
		TargetID:       targetID,
		TotalItems:     0,
		Name:           nilIfEmpty(config.Name),
		Description:    nilIfEmpty(config.Description),
		Metadata:       config.Metadata,
	})
	if err != nil {
		return nil, err
	}

	// Fire-and-forget — errors are silently caught
	// TODO: Launch goroutine calling experiment.RunExperiment once wired.
	// go func() {
	//     _ = experiment.RunExperiment(ctx, d.mastra, experiment.ExperimentConfig{
	//         DatasetID:    d.ID,
	//         ExperimentID: run.ID,
	//         ...config,
	//     })
	// }()

	return &StartExperimentAsyncResult{
		ExperimentID: run.ID,
		Status:       "pending",
	}, nil
}

// nilIfEmpty returns a pointer to s if non-empty, otherwise nil.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ListExperimentsArgs holds the arguments for listing experiments.
type ListExperimentsArgs struct {
	Page    *int `json:"page,omitempty"`
	PerPage *int `json:"perPage,omitempty"`
}

// ListExperiments lists all experiments (runs) for this dataset.
func (d *Dataset) ListExperiments(ctx context.Context, args *ListExperimentsArgs) (storageexperiments.ListExperimentsOutput, error) {
	expStore, err := d.getExperimentsStore(ctx)
	if err != nil {
		return storageexperiments.ListExperimentsOutput{}, err
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

	datasetID := d.ID
	return expStore.ListExperiments(ctx, storageexperiments.ListExperimentsInput{
		DatasetID: &datasetID,
		Pagination: domains.StoragePagination{
			Page:    page,
			PerPage: perPage,
		},
	})
}

// GetExperiment gets a specific experiment (run) by ID.
func (d *Dataset) GetExperiment(ctx context.Context, experimentID string) (storageexperiments.Experiment, error) {
	expStore, err := d.getExperimentsStore(ctx)
	if err != nil {
		return storageexperiments.Experiment{}, err
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
func (d *Dataset) ListExperimentResults(ctx context.Context, args ListExperimentResultsArgs) (storageexperiments.ListExperimentResultsOutput, error) {
	expStore, err := d.getExperimentsStore(ctx)
	if err != nil {
		return storageexperiments.ListExperimentResultsOutput{}, err
	}

	page := 0
	perPage := 20
	if args.Page != nil {
		page = *args.Page
	}
	if args.PerPage != nil {
		perPage = *args.PerPage
	}

	return expStore.ListExperimentResults(ctx, storageexperiments.ListExperimentResultsInput{
		ExperimentID: args.ExperimentID,
		Pagination: domains.StoragePagination{
			Page:    page,
			PerPage: perPage,
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
