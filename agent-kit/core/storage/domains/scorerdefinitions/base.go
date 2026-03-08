// Ported from: packages/core/src/storage/domains/scorer-definitions/base.ts
package scorerdefinitions

import (
	"context"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/evals"
)

// ---------------------------------------------------------------------------
// Scorer Definition Snapshot Config
// ---------------------------------------------------------------------------

// ScorerScoreRange defines the min/max score range for a scorer.
type ScorerScoreRange struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

// ScorerModelConfig is the model configuration for LLM judge scorers.
// Mirrors storage.StorageModelConfig but defined locally to avoid circular imports
// (storage -> scorerdefinitions -> storage).
type ScorerModelConfig struct {
	Provider         string   `json:"provider"`
	Name             string   `json:"name"`
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"topP,omitempty"`
	FrequencyPenalty *float64 `json:"frequencyPenalty,omitempty"`
	PresencePenalty  *float64 `json:"presencePenalty,omitempty"`
	MaxTokens        *int     `json:"maxTokens,omitempty"`
	MaxSteps         *int     `json:"maxSteps,omitempty"`
	Seed             *int     `json:"seed,omitempty"`
}

// ScorerDefinitionSnapshotConfig contains ALL scorer configuration fields.
// These fields live exclusively in version snapshot rows, not on the scorer record.
// Mirrors storage.StorageScorerDefinitionSnapshotType.
type ScorerDefinitionSnapshotConfig struct {
	// Display name of the scorer.
	Name string `json:"name"`
	// Purpose description.
	Description *string `json:"description,omitempty"`
	// Scorer type — determines how the scorer is instantiated at runtime.
	// Values: "llm-judge", "answer-relevancy", "bias", "faithfulness", etc.
	Type string `json:"type"`
	// Model configuration — used for LLM judge; for presets, overrides the default model.
	Model *ScorerModelConfig `json:"model,omitempty"`
	// System instructions for the judge LLM (used when type === "llm-judge").
	Instructions *string `json:"instructions,omitempty"`
	// Score range configuration (used when type === "llm-judge").
	ScoreRange *ScorerScoreRange `json:"scoreRange,omitempty"`
	// Serializable config options for preset scorers (e.g., { scale: 10, context: [...] }).
	PresetConfig map[string]any `json:"presetConfig,omitempty"`
	// Default sampling configuration.
	DefaultSampling *evals.ScoringSamplingConfig `json:"defaultSampling,omitempty"`
}

// ---------------------------------------------------------------------------
// Scorer Definition Version Types
// ---------------------------------------------------------------------------

// ScorerDefinitionVersion represents a stored version of a scorer definition's content.
// Config fields are top-level on the version row (no nested snapshot object).
type ScorerDefinitionVersion struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the scorer definition this version belongs to.
	ScorerDefinitionID string `json:"scorerDefinitionId"`
	// Sequential version number (1, 2, 3, ...).
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`
	// When this version was created.
	CreatedAt time.Time `json:"createdAt"`

	// Scorer configuration fields (from StorageScorerDefinitionSnapshotType).
	ScorerDefinitionSnapshotConfig
}

// CreateScorerDefinitionVersionInput is the input for creating a new scorer definition version.
type CreateScorerDefinitionVersionInput struct {
	// UUID identifier for this version.
	ID string `json:"id"`
	// ID of the scorer definition this version belongs to.
	ScorerDefinitionID string `json:"scorerDefinitionId"`
	// Sequential version number.
	VersionNumber int `json:"versionNumber"`
	// Array of field names that changed from the previous version.
	ChangedFields []string `json:"changedFields,omitempty"`
	// Optional message describing the changes.
	ChangeMessage string `json:"changeMessage,omitempty"`

	// Scorer configuration fields (from StorageScorerDefinitionSnapshotType).
	ScorerDefinitionSnapshotConfig
}

// ScorerDefinitionVersionSortDirection is the sort direction for version listings.
type ScorerDefinitionVersionSortDirection string

const (
	ScorerDefinitionVersionSortASC  ScorerDefinitionVersionSortDirection = "ASC"
	ScorerDefinitionVersionSortDESC ScorerDefinitionVersionSortDirection = "DESC"
)

// ScorerDefinitionVersionOrderBy defines fields for ordering version listings.
type ScorerDefinitionVersionOrderBy string

const (
	ScorerDefinitionVersionOrderByVersionNumber ScorerDefinitionVersionOrderBy = "versionNumber"
	ScorerDefinitionVersionOrderByCreatedAt     ScorerDefinitionVersionOrderBy = "createdAt"
)

// ListScorerDefinitionVersionsInput is the input for listing scorer definition versions.
type ListScorerDefinitionVersionsInput struct {
	// ID of the scorer definition to list versions for.
	ScorerDefinitionID string `json:"scorerDefinitionId"`
	// Zero-indexed page number.
	Page *int `json:"page,omitempty"`
	// Number of items per page. Use -1 to fetch all records without limit.
	PerPage *int `json:"perPage,omitempty"`
	// Sorting options.
	OrderByField     *ScorerDefinitionVersionOrderBy      `json:"orderByField,omitempty"`
	OrderByDirection *ScorerDefinitionVersionSortDirection `json:"orderByDirection,omitempty"`
}

// ListScorerDefinitionVersionsOutput is the output for listing scorer definition versions.
type ListScorerDefinitionVersionsOutput struct {
	Versions []ScorerDefinitionVersion `json:"versions"`
	Total    int                       `json:"total"`
	Page     int                       `json:"page"`
	PerPage  int                       `json:"perPage"`
	HasMore  bool                      `json:"hasMore"`
}

// ---------------------------------------------------------------------------
// ScorerDefinitionsStorage Interface
// ---------------------------------------------------------------------------

// ScorerDefinitionsStorage is the storage interface for the scorer definitions domain.
// It extends VersionedStorageDomain with scorer definition-specific types.
//
// Entity CRUD methods use `any` for input/output because the concrete types
// (StorageScorerDefinitionType, StorageCreateScorerDefinitionInput, etc.)
// live in the storage package, which would create a circular import.
type ScorerDefinitionsStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// --- Entity CRUD ---

	// GetByID retrieves a scorer definition by ID.
	GetByID(ctx context.Context, id string) (any, error)

	// Create creates a new scorer definition.
	Create(ctx context.Context, input any) (any, error)

	// Update updates an existing scorer definition.
	Update(ctx context.Context, input any) (any, error)

	// Delete removes a scorer definition by ID.
	Delete(ctx context.Context, id string) error

	// List lists scorer definitions with optional filtering.
	List(ctx context.Context, args any) (any, error)

	// --- Version Methods ---

	// CreateVersion creates a new scorer definition version.
	CreateVersion(ctx context.Context, input CreateScorerDefinitionVersionInput) (*ScorerDefinitionVersion, error)

	// GetVersion retrieves a version by its ID.
	GetVersion(ctx context.Context, id string) (*ScorerDefinitionVersion, error)

	// GetVersionByNumber retrieves a version by scorer definition ID and version number.
	GetVersionByNumber(ctx context.Context, scorerDefinitionID string, versionNumber int) (*ScorerDefinitionVersion, error)

	// GetLatestVersion retrieves the latest version for a scorer definition.
	GetLatestVersion(ctx context.Context, scorerDefinitionID string) (*ScorerDefinitionVersion, error)

	// ListVersions lists versions for a scorer definition with pagination and sorting.
	ListVersions(ctx context.Context, input ListScorerDefinitionVersionsInput) (*ListScorerDefinitionVersionsOutput, error)

	// DeleteVersion removes a version by ID.
	DeleteVersion(ctx context.Context, id string) error

	// DeleteVersionsByParentID removes all versions for a scorer definition.
	DeleteVersionsByParentID(ctx context.Context, scorerDefinitionID string) error

	// CountVersions returns the number of versions for a scorer definition.
	CountVersions(ctx context.Context, scorerDefinitionID string) (int, error)

	// --- Resolution Methods ---

	// GetByIDResolved resolves an entity by merging its thin record with
	// the active or latest version config.
	GetByIDResolved(ctx context.Context, id string, status string) (any, error)

	// ListResolved lists entities with version resolution.
	ListResolved(ctx context.Context, args any) (any, error)
}
