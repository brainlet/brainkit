// Ported from: packages/core/src/storage/domains/workspaces/filesystem.ts
package workspaces

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
var _ WorkspacesStorage = (*FilesystemWorkspacesStorage)(nil)

// ---------------------------------------------------------------------------
// FilesystemWorkspacesStorage
// ---------------------------------------------------------------------------

// FilesystemWorkspacesStorage is a filesystem-backed implementation of WorkspacesStorage.
type FilesystemWorkspacesStorage struct {
	helpers *fsutil.FilesystemVersionedHelpers
}

// NewFilesystemWorkspacesStorage creates a new FilesystemWorkspacesStorage.
func NewFilesystemWorkspacesStorage(db *fsutil.FilesystemDB) *FilesystemWorkspacesStorage {
	return &FilesystemWorkspacesStorage{
		helpers: fsutil.NewFilesystemVersionedHelpers(fsutil.FilesystemVersionedConfig{
			DB:            db,
			EntitiesFile:  "workspaces.json",
			ParentIDField: "workspaceId",
			Name:          "FilesystemWorkspacesStorage",
			VersionMetadataFields: []string{
				"id", "workspaceId", "versionNumber", "changedFields", "changeMessage", "createdAt",
			},
		}),
	}
}

// Init initializes the storage domain.
func (s *FilesystemWorkspacesStorage) Init(_ context.Context) error {
	s.helpers.Hydrate()
	return nil
}

// DangerouslyClearAll clears all data.
func (s *FilesystemWorkspacesStorage) DangerouslyClearAll(_ context.Context) error {
	return s.helpers.DangerouslyClearAll()
}

// GetByID retrieves a workspace by ID.
func (s *FilesystemWorkspacesStorage) GetByID(_ context.Context, id string) (any, error) {
	return s.helpers.GetByID(id)
}

// Create creates a new workspace with an initial version.
func (s *FilesystemWorkspacesStorage) Create(ctx context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}

	if wsMap, ok := inputMap["workspace"].(map[string]any); ok {
		inputMap = wsMap
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("workspace id is required")
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
		"id":            versionID,
		"workspaceId":   id,
		"versionNumber": 1,
		"changedFields": mapKeys(snapshotConfig),
		"changeMessage": "Initial version",
	}
	for k, v := range snapshotConfig {
		versionInput[k] = v
	}

	if _, err := s.helpers.CreateVersion(versionInput); err != nil {
		return nil, err
	}

	return cloneMap(entity), nil
}

// Update updates an existing workspace.
func (s *FilesystemWorkspacesStorage) Update(_ context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("workspace id is required")
	}

	updates := excludeKeys(inputMap, "id")
	return s.helpers.UpdateEntity(id, updates)
}

// Delete removes a workspace by ID.
func (s *FilesystemWorkspacesStorage) Delete(_ context.Context, id string) error {
	return s.helpers.DeleteEntity(id)
}

// List lists workspaces with optional filtering.
func (s *FilesystemWorkspacesStorage) List(_ context.Context, args any) (any, error) {
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

	return s.helpers.ListEntities(page, perPage, orderBy, filters, "workspaces")
}

// CreateVersion creates a new workspace version.
func (s *FilesystemWorkspacesStorage) CreateVersion(_ context.Context, input CreateWorkspaceVersionInput) (*WorkspaceVersion, error) {
	inputMap, _ := toMap(input)
	result, err := s.helpers.CreateVersion(inputMap)
	if err != nil {
		return nil, err
	}
	return mapToWorkspaceVersion(result), nil
}

// GetVersion retrieves a version by its ID.
func (s *FilesystemWorkspacesStorage) GetVersion(_ context.Context, id string) (*WorkspaceVersion, error) {
	result, err := s.helpers.GetVersion(id)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToWorkspaceVersion(result), nil
}

// GetVersionByNumber retrieves a version by workspace ID and version number.
func (s *FilesystemWorkspacesStorage) GetVersionByNumber(_ context.Context, workspaceID string, versionNumber int) (*WorkspaceVersion, error) {
	result, err := s.helpers.GetVersionByNumber(workspaceID, versionNumber)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToWorkspaceVersion(result), nil
}

// GetLatestVersion retrieves the latest version for a workspace.
func (s *FilesystemWorkspacesStorage) GetLatestVersion(_ context.Context, workspaceID string) (*WorkspaceVersion, error) {
	result, err := s.helpers.GetLatestVersion(workspaceID)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToWorkspaceVersion(result), nil
}

// ListVersions lists versions for a workspace with pagination and sorting.
func (s *FilesystemWorkspacesStorage) ListVersions(_ context.Context, input ListWorkspaceVersionsInput) (*ListWorkspaceVersionsOutput, error) {
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

	result, err := s.helpers.ListVersions(input.WorkspaceID, page, perPage, orderBy)
	if err != nil {
		return nil, err
	}

	versions := make([]WorkspaceVersion, len(result.Versions))
	for i, v := range result.Versions {
		if m, ok := v.(map[string]any); ok {
			versions[i] = *mapToWorkspaceVersion(m)
		}
	}

	return &ListWorkspaceVersionsOutput{
		Versions: versions,
		Total:    result.Total,
		Page:     result.Page,
		PerPage:  result.PerPage,
		HasMore:  result.HasMore,
	}, nil
}

// DeleteVersion removes a version by ID.
func (s *FilesystemWorkspacesStorage) DeleteVersion(_ context.Context, id string) error {
	return s.helpers.DeleteVersion(id)
}

// DeleteVersionsByParentID removes all versions for a workspace.
func (s *FilesystemWorkspacesStorage) DeleteVersionsByParentID(_ context.Context, workspaceID string) error {
	return s.helpers.DeleteVersionsByParentID(workspaceID)
}

// CountVersions returns the number of versions for a workspace.
func (s *FilesystemWorkspacesStorage) CountVersions(_ context.Context, workspaceID string) (int, error) {
	return s.helpers.CountVersions(workspaceID)
}

// GetByIDResolved resolves an entity by merging its thin record with the active or latest version config.
func (s *FilesystemWorkspacesStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
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
func (s *FilesystemWorkspacesStorage) ListResolved(ctx context.Context, args any) (any, error) {
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

	entities, ok := resultMap["workspaces"].([]any)
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

	resultMap["workspaces"] = resolved
	return resultMap, nil
}

// resolveEntity merges a thin entity record with its active or latest version config.
func (s *FilesystemWorkspacesStorage) resolveEntity(_ context.Context, entityRaw any, status string) (any, error) {
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
				"id", "workspaceId", "versionNumber", "changedFields", "changeMessage", "createdAt",
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

func mapToWorkspaceVersion(m map[string]any) *WorkspaceVersion {
	v := &WorkspaceVersion{}
	data, err := json.Marshal(m)
	if err != nil {
		return v
	}
	_ = json.Unmarshal(data, v)
	v.Snapshot = m
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
