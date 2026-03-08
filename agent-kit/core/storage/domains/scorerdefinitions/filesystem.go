// Ported from: packages/core/src/storage/domains/scorer-definitions/filesystem.ts
package scorerdefinitions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
	"github.com/brainlet/brainkit/agent-kit/core/storage/fsutil"
)

// Compile-time interface check.
var _ ScorerDefinitionsStorage = (*FilesystemScorerDefinitionsStorage)(nil)

// ---------------------------------------------------------------------------
// FilesystemScorerDefinitionsStorage
// ---------------------------------------------------------------------------

// FilesystemScorerDefinitionsStorage is a filesystem-backed implementation of ScorerDefinitionsStorage.
type FilesystemScorerDefinitionsStorage struct {
	helpers *fsutil.FilesystemVersionedHelpers
}

// NewFilesystemScorerDefinitionsStorage creates a new FilesystemScorerDefinitionsStorage.
func NewFilesystemScorerDefinitionsStorage(db *fsutil.FilesystemDB) *FilesystemScorerDefinitionsStorage {
	return &FilesystemScorerDefinitionsStorage{
		helpers: fsutil.NewFilesystemVersionedHelpers(fsutil.FilesystemVersionedConfig{
			DB:            db,
			EntitiesFile:  "scorer-definitions.json",
			ParentIDField: "scorerDefinitionId",
			Name:          "FilesystemScorerDefinitionsStorage",
			VersionMetadataFields: []string{
				"id", "scorerDefinitionId", "versionNumber", "changedFields", "changeMessage", "createdAt",
			},
		}),
	}
}

// Init initializes the storage domain.
func (s *FilesystemScorerDefinitionsStorage) Init(_ context.Context) error {
	s.helpers.Hydrate()
	return nil
}

// DangerouslyClearAll clears all data.
func (s *FilesystemScorerDefinitionsStorage) DangerouslyClearAll(_ context.Context) error {
	return s.helpers.DangerouslyClearAll()
}

// GetByID retrieves a scorer definition by ID.
func (s *FilesystemScorerDefinitionsStorage) GetByID(_ context.Context, id string) (any, error) {
	return s.helpers.GetByID(id)
}

// Create creates a new scorer definition with an initial version.
func (s *FilesystemScorerDefinitionsStorage) Create(ctx context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}

	if sdMap, ok := inputMap["scorerDefinition"].(map[string]any); ok {
		inputMap = sdMap
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("scorerDefinition id is required")
	}

	now := time.Now()
	entity := map[string]any{
		"id":        id,
		"status":    "draft",
		"authorId":  inputMap["authorId"],
		"metadata":  inputMap["metadata"],
		"createdAt": now,
		"updatedAt": now,
	}

	if _, err := s.helpers.CreateEntity(id, entity); err != nil {
		return nil, err
	}

	snapshotConfig := excludeKeys(inputMap, "id", "authorId", "metadata")
	versionID := uuid.New().String()
	versionInput := map[string]any{
		"id":                   versionID,
		"scorerDefinitionId":   id,
		"versionNumber":        1,
		"changedFields":        mapKeys(snapshotConfig),
		"changeMessage":        "Initial version",
	}
	for k, v := range snapshotConfig {
		versionInput[k] = v
	}

	if _, err := s.helpers.CreateVersion(versionInput); err != nil {
		return nil, err
	}

	return cloneMap(entity), nil
}

// Update updates an existing scorer definition.
func (s *FilesystemScorerDefinitionsStorage) Update(_ context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("scorerDefinition id is required")
	}

	updates := excludeKeys(inputMap, "id")
	return s.helpers.UpdateEntity(id, updates)
}

// Delete removes a scorer definition by ID.
func (s *FilesystemScorerDefinitionsStorage) Delete(_ context.Context, id string) error {
	return s.helpers.DeleteEntity(id)
}

// List lists scorer definitions with optional filtering.
func (s *FilesystemScorerDefinitionsStorage) List(_ context.Context, args any) (any, error) {
	argsMap, _ := toMap(args)

	page := intVal(argsMap, "page", 0)
	perPage := intVal(argsMap, "perPage", 100)
	orderByRaw := mapVal(argsMap, "orderBy")

	var orderBy *domains.StorageOrderBy
	if orderByRaw != nil {
		orderBy = &domains.StorageOrderBy{
			Field:     domains.ThreadOrderBy(strVal(orderByRaw, "field")),
			Direction: domains.SortDirection(strVal(orderByRaw, "direction")),
		}
	}

	filters := make(map[string]any)
	if v := strVal(argsMap, "authorId"); v != "" {
		filters["authorId"] = v
	}
	if v := mapVal(argsMap, "metadata"); v != nil {
		filters["metadata"] = v
	}
	if v := strVal(argsMap, "status"); v != "" {
		filters["status"] = v
	}

	return s.helpers.ListEntities(page, perPage, orderBy, filters, "scorerDefinitions")
}

// CreateVersion creates a new scorer definition version.
func (s *FilesystemScorerDefinitionsStorage) CreateVersion(_ context.Context, input CreateScorerDefinitionVersionInput) (*ScorerDefinitionVersion, error) {
	inputMap, _ := toMap(input)
	result, err := s.helpers.CreateVersion(inputMap)
	if err != nil {
		return nil, err
	}
	return mapToScorerDefinitionVersion(result), nil
}

// GetVersion retrieves a version by its ID.
func (s *FilesystemScorerDefinitionsStorage) GetVersion(_ context.Context, id string) (*ScorerDefinitionVersion, error) {
	result, err := s.helpers.GetVersion(id)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToScorerDefinitionVersion(result), nil
}

// GetVersionByNumber retrieves a version by scorer definition ID and version number.
func (s *FilesystemScorerDefinitionsStorage) GetVersionByNumber(_ context.Context, scorerDefinitionID string, versionNumber int) (*ScorerDefinitionVersion, error) {
	result, err := s.helpers.GetVersionByNumber(scorerDefinitionID, versionNumber)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToScorerDefinitionVersion(result), nil
}

// GetLatestVersion retrieves the latest version for a scorer definition.
func (s *FilesystemScorerDefinitionsStorage) GetLatestVersion(_ context.Context, scorerDefinitionID string) (*ScorerDefinitionVersion, error) {
	result, err := s.helpers.GetLatestVersion(scorerDefinitionID)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToScorerDefinitionVersion(result), nil
}

// ListVersions lists versions for a scorer definition with pagination and sorting.
func (s *FilesystemScorerDefinitionsStorage) ListVersions(_ context.Context, input ListScorerDefinitionVersionsInput) (*ListScorerDefinitionVersionsOutput, error) {
	page := 0
	if input.Page != nil {
		page = *input.Page
	}
	perPage := 20
	if input.PerPage != nil {
		perPage = *input.PerPage
	}

	var orderBy *domains.VersionOrderByClause
	if input.OrderByField != nil || input.OrderByDirection != nil {
		orderBy = &domains.VersionOrderByClause{}
		if input.OrderByField != nil {
			orderBy.Field = domains.VersionOrderBy(*input.OrderByField)
		}
		if input.OrderByDirection != nil {
			orderBy.Direction = domains.SortDirection(*input.OrderByDirection)
		}
	}

	result, err := s.helpers.ListVersions(input.ScorerDefinitionID, page, perPage, orderBy)
	if err != nil {
		return nil, err
	}

	versions := make([]ScorerDefinitionVersion, len(result.Versions))
	for i, v := range result.Versions {
		if m, ok := v.(map[string]any); ok {
			versions[i] = *mapToScorerDefinitionVersion(m)
		}
	}

	return &ListScorerDefinitionVersionsOutput{
		Versions: versions,
		Total:    result.Total,
		Page:     result.Page,
		PerPage:  result.PerPage,
		HasMore:  result.HasMore,
	}, nil
}

// DeleteVersion removes a version by ID.
func (s *FilesystemScorerDefinitionsStorage) DeleteVersion(_ context.Context, id string) error {
	return s.helpers.DeleteVersion(id)
}

// DeleteVersionsByParentID removes all versions for a scorer definition.
func (s *FilesystemScorerDefinitionsStorage) DeleteVersionsByParentID(_ context.Context, scorerDefinitionID string) error {
	return s.helpers.DeleteVersionsByParentID(scorerDefinitionID)
}

// CountVersions returns the number of versions for a scorer definition.
func (s *FilesystemScorerDefinitionsStorage) CountVersions(_ context.Context, scorerDefinitionID string) (int, error) {
	return s.helpers.CountVersions(scorerDefinitionID)
}

// GetByIDResolved resolves an entity by merging its thin record with the active or latest version config.
func (s *FilesystemScorerDefinitionsStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
	entityRaw, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entityRaw == nil {
		return nil, nil
	}
	return s.resolveEntity(ctx, entityRaw, status)
}

// ListResolved lists entities with version resolution.
func (s *FilesystemScorerDefinitionsStorage) ListResolved(ctx context.Context, args any) (any, error) {
	result, err := s.List(ctx, args)
	if err != nil {
		return nil, err
	}
	resultMap, ok := result.(map[string]any)
	if !ok {
		return result, nil
	}

	argsMap, _ := toMap(args)
	resolveStatus := strVal(argsMap, "status")

	entities, ok := resultMap["scorerDefinitions"].([]any)
	if !ok {
		return result, nil
	}

	resolved := make([]any, len(entities))
	for i, entity := range entities {
		r, err := s.resolveEntity(ctx, entity, resolveStatus)
		if err != nil {
			return nil, err
		}
		resolved[i] = r
	}

	resultMap["scorerDefinitions"] = resolved
	return resultMap, nil
}

// resolveEntity merges a thin entity record with its active or latest version config.
func (s *FilesystemScorerDefinitionsStorage) resolveEntity(_ context.Context, entityRaw any, status string) (any, error) {
	if status == "" {
		status = "published"
	}

	entityMap, ok := toMap(entityRaw)
	if !ok {
		return entityRaw, nil
	}

	entityID, _ := entityMap["id"].(string)

	var versionMap map[string]any
	if status == "draft" {
		versionMap, _ = s.helpers.GetLatestVersion(entityID)
	} else {
		activeVersionID, _ := entityMap["activeVersionId"].(string)
		if activeVersionID != "" {
			versionMap, _ = s.helpers.GetVersion(activeVersionID)
		}
		if versionMap == nil {
			versionMap, _ = s.helpers.GetLatestVersion(entityID)
		}
	}

	if versionMap != nil {
		base := domains.VersionedStorageDomainBase{
			VersionMetadataFields: []string{
				"id", "scorerDefinitionId", "versionNumber", "changedFields", "changeMessage", "createdAt",
			},
		}
		snapshotConfig := base.ExtractSnapshotConfig(versionMap)

		merged := cloneMap(entityMap)
		for k, v := range snapshotConfig {
			merged[k] = v
		}
		merged["resolvedVersionId"] = versionMap["id"]
		return merged, nil
	}

	return entityMap, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mapToScorerDefinitionVersion(m map[string]any) *ScorerDefinitionVersion {
	v := &ScorerDefinitionVersion{}
	data, err := json.Marshal(m)
	if err != nil {
		return v
	}
	_ = json.Unmarshal(data, v)
	return v
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
