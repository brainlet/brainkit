// Ported from: packages/core/src/storage/domains/mcp-clients/filesystem.ts
package mcpclients

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
var _ MCPClientsStorage = (*FilesystemMCPClientsStorage)(nil)

// ---------------------------------------------------------------------------
// FilesystemMCPClientsStorage
// ---------------------------------------------------------------------------

// FilesystemMCPClientsStorage is a filesystem-backed implementation of MCPClientsStorage.
type FilesystemMCPClientsStorage struct {
	helpers *fsutil.FilesystemVersionedHelpers
}

// NewFilesystemMCPClientsStorage creates a new FilesystemMCPClientsStorage.
func NewFilesystemMCPClientsStorage(db *fsutil.FilesystemDB) *FilesystemMCPClientsStorage {
	return &FilesystemMCPClientsStorage{
		helpers: fsutil.NewFilesystemVersionedHelpers(fsutil.FilesystemVersionedConfig{
			DB:            db,
			EntitiesFile:  "mcp-clients.json",
			ParentIDField: "mcpClientId",
			Name:          "FilesystemMCPClientsStorage",
			VersionMetadataFields: []string{
				"id", "mcpClientId", "versionNumber", "changedFields", "changeMessage", "createdAt",
			},
		}),
	}
}

// Init initializes the storage domain.
func (s *FilesystemMCPClientsStorage) Init(_ context.Context) error {
	s.helpers.Hydrate()
	return nil
}

// DangerouslyClearAll clears all data.
func (s *FilesystemMCPClientsStorage) DangerouslyClearAll(_ context.Context) error {
	return s.helpers.DangerouslyClearAll()
}

// GetByID retrieves an MCP client by ID.
func (s *FilesystemMCPClientsStorage) GetByID(_ context.Context, id string) (any, error) {
	return s.helpers.GetByID(id)
}

// Create creates a new MCP client with an initial version.
func (s *FilesystemMCPClientsStorage) Create(ctx context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}

	if mcpClientMap, ok := inputMap["mcpClient"].(map[string]any); ok {
		inputMap = mcpClientMap
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("mcpClient id is required")
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
		"mcpClientId":   id,
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

// Update updates an existing MCP client.
func (s *FilesystemMCPClientsStorage) Update(_ context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("mcpClient id is required")
	}

	updates := excludeKeys(inputMap, "id")
	return s.helpers.UpdateEntity(id, updates)
}

// Delete removes an MCP client by ID.
func (s *FilesystemMCPClientsStorage) Delete(_ context.Context, id string) error {
	return s.helpers.DeleteEntity(id)
}

// List lists MCP clients with optional filtering.
func (s *FilesystemMCPClientsStorage) List(_ context.Context, args any) (any, error) {
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

	return s.helpers.ListEntities(page, perPage, orderBy, filters, "mcpClients")
}

// CreateVersion creates a new MCP client version.
func (s *FilesystemMCPClientsStorage) CreateVersion(_ context.Context, input CreateMCPClientVersionInput) (*MCPClientVersion, error) {
	inputMap, _ := toMap(input)
	result, err := s.helpers.CreateVersion(inputMap)
	if err != nil {
		return nil, err
	}
	return mapToMCPClientVersion(result), nil
}

// GetVersion retrieves a version by its ID.
func (s *FilesystemMCPClientsStorage) GetVersion(_ context.Context, id string) (*MCPClientVersion, error) {
	result, err := s.helpers.GetVersion(id)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToMCPClientVersion(result), nil
}

// GetVersionByNumber retrieves a version by MCP client ID and version number.
func (s *FilesystemMCPClientsStorage) GetVersionByNumber(_ context.Context, mcpClientID string, versionNumber int) (*MCPClientVersion, error) {
	result, err := s.helpers.GetVersionByNumber(mcpClientID, versionNumber)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToMCPClientVersion(result), nil
}

// GetLatestVersion retrieves the latest version for an MCP client.
func (s *FilesystemMCPClientsStorage) GetLatestVersion(_ context.Context, mcpClientID string) (*MCPClientVersion, error) {
	result, err := s.helpers.GetLatestVersion(mcpClientID)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToMCPClientVersion(result), nil
}

// ListVersions lists versions for an MCP client with pagination and sorting.
func (s *FilesystemMCPClientsStorage) ListVersions(_ context.Context, input ListMCPClientVersionsInput) (*ListMCPClientVersionsOutput, error) {
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

	result, err := s.helpers.ListVersions(input.MCPClientID, page, perPage, orderBy)
	if err != nil {
		return nil, err
	}

	versions := make([]MCPClientVersion, len(result.Versions))
	for i, v := range result.Versions {
		if m, ok := v.(map[string]any); ok {
			versions[i] = *mapToMCPClientVersion(m)
		}
	}

	return &ListMCPClientVersionsOutput{
		Versions: versions,
		Total:    result.Total,
		Page:     result.Page,
		PerPage:  result.PerPage,
		HasMore:  result.HasMore,
	}, nil
}

// DeleteVersion removes a version by ID.
func (s *FilesystemMCPClientsStorage) DeleteVersion(_ context.Context, id string) error {
	return s.helpers.DeleteVersion(id)
}

// DeleteVersionsByParentID removes all versions for an MCP client.
func (s *FilesystemMCPClientsStorage) DeleteVersionsByParentID(_ context.Context, mcpClientID string) error {
	return s.helpers.DeleteVersionsByParentID(mcpClientID)
}

// CountVersions returns the number of versions for an MCP client.
func (s *FilesystemMCPClientsStorage) CountVersions(_ context.Context, mcpClientID string) (int, error) {
	return s.helpers.CountVersions(mcpClientID)
}

// GetByIDResolved resolves an entity by merging its thin record with the active or latest version config.
func (s *FilesystemMCPClientsStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
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
func (s *FilesystemMCPClientsStorage) ListResolved(ctx context.Context, args any) (any, error) {
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

	entities, ok := resultMap["mcpClients"].([]any)
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

	resultMap["mcpClients"] = resolved
	return resultMap, nil
}

// resolveEntity merges a thin entity record with its active or latest version config.
func (s *FilesystemMCPClientsStorage) resolveEntity(_ context.Context, entityRaw any, status string) (any, error) {
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
				"id", "mcpClientId", "versionNumber", "changedFields", "changeMessage", "createdAt",
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

func mapToMCPClientVersion(m map[string]any) *MCPClientVersion {
	v := &MCPClientVersion{}
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
