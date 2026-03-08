// Ported from: packages/core/src/storage/domains/datasets/base.ts
package datasets

import (
	"context"
	"time"

	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// Dataset Record Types
// ---------------------------------------------------------------------------

// DatasetRecord is a dataset record.
type DatasetRecord struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Description       *string        `json:"description,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	InputSchema       map[string]any `json:"inputSchema,omitempty"`
	GroundTruthSchema map[string]any `json:"groundTruthSchema,omitempty"`
	Version           int            `json:"version"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

// DatasetItem is an item within a dataset.
type DatasetItem struct {
	ID             string         `json:"id"`
	DatasetID      string         `json:"datasetId"`
	DatasetVersion int            `json:"datasetVersion"`
	Input          any            `json:"input"`
	GroundTruth    any            `json:"groundTruth,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// DatasetItemRow is the raw database row for a dataset item (includes versioning fields).
type DatasetItemRow struct {
	ID             string         `json:"id"`
	DatasetID      string         `json:"datasetId"`
	DatasetVersion int            `json:"datasetVersion"`
	ValidTo        *int           `json:"validTo"`  // null means current
	IsDeleted      bool           `json:"isDeleted"`
	Input          any            `json:"input"`
	GroundTruth    any            `json:"groundTruth,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// DatasetVersion represents a dataset version record.
type DatasetVersion struct {
	ID        string    `json:"id"`
	DatasetID string    `json:"datasetId"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
}

// ---------------------------------------------------------------------------
// Dataset Input/Output Types
// ---------------------------------------------------------------------------

// CreateDatasetInput is the input for creating a new dataset.
type CreateDatasetInput struct {
	Name              string         `json:"name"`
	Description       *string        `json:"description,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	InputSchema       map[string]any `json:"inputSchema,omitempty"`
	GroundTruthSchema map[string]any `json:"groundTruthSchema,omitempty"`
}

// UpdateDatasetInput is the input for updating a dataset.
type UpdateDatasetInput struct {
	ID                string         `json:"id"`
	Name              *string        `json:"name,omitempty"`
	Description       *string        `json:"description,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
	InputSchema       map[string]any `json:"inputSchema,omitempty"`
	GroundTruthSchema map[string]any `json:"groundTruthSchema,omitempty"`
}

// AddDatasetItemInput is the input for adding an item to a dataset.
type AddDatasetItemInput struct {
	DatasetID   string         `json:"datasetId"`
	Input       any            `json:"input"`
	GroundTruth any            `json:"groundTruth,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// UpdateDatasetItemInput is the input for updating a dataset item.
type UpdateDatasetItemInput struct {
	ID          string         `json:"id"`
	DatasetID   string         `json:"datasetId"`
	Input       any            `json:"input,omitempty"`
	GroundTruth any            `json:"groundTruth,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ListDatasetsInput is the input for listing datasets.
type ListDatasetsInput struct {
	Pagination domains.StoragePagination `json:"pagination"`
}

// ListDatasetsOutput is the paginated output for listing datasets.
type ListDatasetsOutput struct {
	Datasets   []DatasetRecord      `json:"datasets"`
	Pagination domains.PaginationInfo `json:"pagination"`
}

// ListDatasetItemsInput is the input for listing dataset items.
type ListDatasetItemsInput struct {
	DatasetID  string                    `json:"datasetId"`
	Version    *int                      `json:"version,omitempty"`
	Search     *string                   `json:"search,omitempty"`
	Pagination domains.StoragePagination `json:"pagination"`
}

// ListDatasetItemsOutput is the paginated output for listing dataset items.
type ListDatasetItemsOutput struct {
	Items      []DatasetItem        `json:"items"`
	Pagination domains.PaginationInfo `json:"pagination"`
}

// ListDatasetVersionsInput is the input for listing dataset versions.
type ListDatasetVersionsInput struct {
	DatasetID  string                    `json:"datasetId"`
	Pagination domains.StoragePagination `json:"pagination"`
}

// ListDatasetVersionsOutput is the paginated output for listing dataset versions.
type ListDatasetVersionsOutput struct {
	Versions   []DatasetVersion     `json:"versions"`
	Pagination domains.PaginationInfo `json:"pagination"`
}

// BatchInsertItemsInput is the input for batch inserting dataset items.
type BatchInsertItemsInput struct {
	DatasetID string                        `json:"datasetId"`
	Items     []domains.BatchInsertItemInput `json:"items"`
}

// BatchDeleteItemsInput is the input for batch deleting dataset items.
type BatchDeleteItemsInput struct {
	DatasetID string   `json:"datasetId"`
	ItemIDs   []string `json:"itemIds"`
}

// DeleteItemArgs holds the arguments for deleting an item.
type DeleteItemArgs struct {
	ID        string `json:"id"`
	DatasetID string `json:"datasetId"`
}

// GetItemByIDArgs holds the arguments for getting an item by ID.
type GetItemByIDArgs struct {
	ID             string `json:"id"`
	DatasetVersion *int   `json:"datasetVersion,omitempty"`
}

// GetItemsByVersionArgs holds the arguments for getting items by version.
type GetItemsByVersionArgs struct {
	DatasetID string `json:"datasetId"`
	Version   int    `json:"version"`
}

// ---------------------------------------------------------------------------
// DatasetsStorage Interface
// ---------------------------------------------------------------------------

// DatasetsStorage is the storage interface for the datasets domain.
// Provides the contract for dataset and dataset item CRUD operations.
//
// Schema validation (Template Method pattern in TS) is expected to be
// handled by wrapper logic in Go — the interface exposes the raw storage
// operations.
type DatasetsStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// --- Dataset CRUD ---

	// CreateDataset creates a new dataset.
	CreateDataset(ctx context.Context, input CreateDatasetInput) (DatasetRecord, error)

	// GetDatasetByID retrieves a dataset by ID.
	GetDatasetByID(ctx context.Context, id string) (DatasetRecord, error)

	// DeleteDataset removes a dataset by ID.
	DeleteDataset(ctx context.Context, id string) error

	// ListDatasets lists datasets with optional filtering.
	ListDatasets(ctx context.Context, args ListDatasetsInput) (ListDatasetsOutput, error)

	// UpdateDataset updates a dataset.
	// Note: In TS, schema validation is done in the base class before delegating
	// to the protected _doUpdateDataset method. In Go, validation should be
	// handled by a wrapper or middleware.
	UpdateDataset(ctx context.Context, args UpdateDatasetInput) (DatasetRecord, error)

	// --- Item CRUD ---

	// AddItem adds an item to a dataset.
	// In TS, validation happens before delegation to _doAddItem. In Go,
	// validation should be handled externally.
	AddItem(ctx context.Context, args AddDatasetItemInput) (DatasetItem, error)

	// UpdateItem updates an item in a dataset.
	UpdateItem(ctx context.Context, args UpdateDatasetItemInput) (DatasetItem, error)

	// DeleteItem deletes an item from a dataset (creates a tombstone row via SCD-2).
	DeleteItem(ctx context.Context, args DeleteItemArgs) error

	// ListItems lists items in a dataset.
	ListItems(ctx context.Context, args ListDatasetItemsInput) (ListDatasetItemsOutput, error)

	// GetItemByID retrieves an item by ID, optionally at a specific dataset version.
	GetItemByID(ctx context.Context, args GetItemByIDArgs) (DatasetItem, error)

	// --- SCD-2 Queries ---

	// GetItemsByVersion retrieves all items at a specific dataset version.
	GetItemsByVersion(ctx context.Context, args GetItemsByVersionArgs) ([]DatasetItem, error)

	// GetItemHistory retrieves the full SCD-2 history for an item.
	GetItemHistory(ctx context.Context, itemID string) ([]DatasetItemRow, error)

	// --- Dataset Version Methods ---

	// CreateDatasetVersion creates a new dataset version record.
	CreateDatasetVersion(ctx context.Context, datasetID string, version int) (DatasetVersion, error)

	// ListDatasetVersions lists dataset versions with pagination.
	ListDatasetVersions(ctx context.Context, input ListDatasetVersionsInput) (ListDatasetVersionsOutput, error)

	// --- Batch Operations ---

	// BatchInsertItems inserts multiple items in a batch.
	// Validation should be handled externally in Go.
	BatchInsertItems(ctx context.Context, input BatchInsertItemsInput) ([]DatasetItem, error)

	// BatchDeleteItems deletes multiple items in a batch (creates tombstone rows).
	BatchDeleteItems(ctx context.Context, input BatchDeleteItemsInput) error
}
