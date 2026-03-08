// Ported from: packages/core/src/storage/domains/mcp-servers/filesystem.ts
package mcpservers

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
var _ MCPServersStorage = (*FilesystemMCPServersStorage)(nil)

// ---------------------------------------------------------------------------
// FilesystemMCPServersStorage
// ---------------------------------------------------------------------------

// FilesystemMCPServersStorage is a filesystem-backed implementation of MCPServersStorage.
type FilesystemMCPServersStorage struct {
	helpers *fsutil.FilesystemVersionedHelpers
}

// NewFilesystemMCPServersStorage creates a new FilesystemMCPServersStorage.
func NewFilesystemMCPServersStorage(db *fsutil.FilesystemDB) *FilesystemMCPServersStorage {
	return &FilesystemMCPServersStorage{
		helpers: fsutil.NewFilesystemVersionedHelpers(fsutil.FilesystemVersionedConfig{
			DB:            db,
			EntitiesFile:  "mcp-servers.json",
			ParentIDField: "mcpServerId",
			Name:          "FilesystemMCPServersStorage",
			VersionMetadataFields: []string{
				"id", "mcpServerId", "versionNumber", "changedFields", "changeMessage", "createdAt",
			},
		}),
	}
}

// Init initializes the storage domain.
func (s *FilesystemMCPServersStorage) Init(_ context.Context) error {
	s.helpers.Hydrate()
	return nil
}

// DangerouslyClearAll clears all data.
func (s *FilesystemMCPServersStorage) DangerouslyClearAll(_ context.Context) error {
	return s.helpers.DangerouslyClearAll()
}

// GetByID retrieves an MCP server by ID.
func (s *FilesystemMCPServersStorage) GetByID(_ context.Context, id string) (any, error) {
	return s.helpers.GetByID(id)
}

// Create creates a new MCP server with an initial version.
func (s *FilesystemMCPServersStorage) Create(ctx context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}

	if mcpServerMap, ok := inputMap["mcpServer"].(map[string]any); ok {
		inputMap = mcpServerMap
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("mcpServer id is required")
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
		"mcpServerId":   id,
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

// Update updates an existing MCP server.
func (s *FilesystemMCPServersStorage) Update(_ context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("mcpServer id is required")
	}

	updates := excludeKeys(inputMap, "id")
	return s.helpers.UpdateEntity(id, updates)
}

// Delete removes an MCP server by ID.
func (s *FilesystemMCPServersStorage) Delete(_ context.Context, id string) error {
	return s.helpers.DeleteEntity(id)
}

// List lists MCP servers with optional filtering.
func (s *FilesystemMCPServersStorage) List(_ context.Context, args any) (any, error) {
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

	return s.helpers.ListEntities(page, perPage, orderBy, filters, "mcpServers")
}

// CreateVersion creates a new MCP server version.
func (s *FilesystemMCPServersStorage) CreateVersion(_ context.Context, input CreateMCPServerVersionInput) (*MCPServerVersion, error) {
	inputMap, _ := toMap(input)
	result, err := s.helpers.CreateVersion(inputMap)
	if err != nil {
		return nil, err
	}
	return mapToMCPServerVersion(result), nil
}

// GetVersion retrieves a version by its ID.
func (s *FilesystemMCPServersStorage) GetVersion(_ context.Context, id string) (*MCPServerVersion, error) {
	result, err := s.helpers.GetVersion(id)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToMCPServerVersion(result), nil
}

// GetVersionByNumber retrieves a version by MCP server ID and version number.
func (s *FilesystemMCPServersStorage) GetVersionByNumber(_ context.Context, mcpServerID string, versionNumber int) (*MCPServerVersion, error) {
	result, err := s.helpers.GetVersionByNumber(mcpServerID, versionNumber)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToMCPServerVersion(result), nil
}

// GetLatestVersion retrieves the latest version for an MCP server.
func (s *FilesystemMCPServersStorage) GetLatestVersion(_ context.Context, mcpServerID string) (*MCPServerVersion, error) {
	result, err := s.helpers.GetLatestVersion(mcpServerID)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToMCPServerVersion(result), nil
}

// ListVersions lists versions for an MCP server with pagination and sorting.
func (s *FilesystemMCPServersStorage) ListVersions(_ context.Context, input ListMCPServerVersionsInput) (*ListMCPServerVersionsOutput, error) {
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

	result, err := s.helpers.ListVersions(input.MCPServerID, page, perPage, orderBy)
	if err != nil {
		return nil, err
	}

	versions := make([]MCPServerVersion, len(result.Versions))
	for i, v := range result.Versions {
		if m, ok := v.(map[string]any); ok {
			versions[i] = *mapToMCPServerVersion(m)
		}
	}

	return &ListMCPServerVersionsOutput{
		Versions: versions,
		Total:    result.Total,
		Page:     result.Page,
		PerPage:  result.PerPage,
		HasMore:  result.HasMore,
	}, nil
}

// DeleteVersion removes a version by ID.
func (s *FilesystemMCPServersStorage) DeleteVersion(_ context.Context, id string) error {
	return s.helpers.DeleteVersion(id)
}

// DeleteVersionsByParentID removes all versions for an MCP server.
func (s *FilesystemMCPServersStorage) DeleteVersionsByParentID(_ context.Context, mcpServerID string) error {
	return s.helpers.DeleteVersionsByParentID(mcpServerID)
}

// CountVersions returns the number of versions for an MCP server.
func (s *FilesystemMCPServersStorage) CountVersions(_ context.Context, mcpServerID string) (int, error) {
	return s.helpers.CountVersions(mcpServerID)
}

// GetByIDResolved resolves an entity by merging its thin record with the active or latest version config.
func (s *FilesystemMCPServersStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
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
func (s *FilesystemMCPServersStorage) ListResolved(ctx context.Context, args any) (any, error) {
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

	entities, ok := resultMap["mcpServers"].([]any)
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

	resultMap["mcpServers"] = resolved
	return resultMap, nil
}

// resolveEntity merges a thin entity record with its active or latest version config.
func (s *FilesystemMCPServersStorage) resolveEntity(_ context.Context, entityRaw any, status string) (any, error) {
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
				"id", "mcpServerId", "versionNumber", "changedFields", "changeMessage", "createdAt",
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

func mapToMCPServerVersion(m map[string]any) *MCPServerVersion {
	v := &MCPServerVersion{}
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
