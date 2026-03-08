// Ported from: packages/core/src/storage/domains/agents/filesystem.ts
package agents

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
var _ AgentsStorage = (*FilesystemAgentsStorage)(nil)

// persistedSnapshotFields is the set of fields persisted for filesystem-stored agents.
// Only fields that applyStoredOverrides actually uses plus the
// minimum required by the storage schema (name, model).
var persistedSnapshotFields = map[string]bool{
	"name":                 true,
	"instructions":         true,
	"model":                true,
	"tools":                true,
	"integrationTools":     true,
	"mcpClients":           true,
	"requestContextSchema": true,
}

// stripUnusedFields returns a copy of obj containing only persisted snapshot fields.
func stripUnusedFields(obj map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range obj {
		if persistedSnapshotFields[key] {
			result[key] = value
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// FilesystemAgentsStorage
// ---------------------------------------------------------------------------

// FilesystemAgentsStorage is a filesystem-backed implementation of AgentsStorage.
type FilesystemAgentsStorage struct {
	helpers *fsutil.FilesystemVersionedHelpers
}

// NewFilesystemAgentsStorage creates a new FilesystemAgentsStorage.
func NewFilesystemAgentsStorage(db *fsutil.FilesystemDB) *FilesystemAgentsStorage {
	return &FilesystemAgentsStorage{
		helpers: fsutil.NewFilesystemVersionedHelpers(fsutil.FilesystemVersionedConfig{
			DB:            db,
			EntitiesFile:  "agents.json",
			ParentIDField: "agentId",
			Name:          "FilesystemAgentsStorage",
			VersionMetadataFields: []string{
				"id", "agentId", "versionNumber", "changedFields", "changeMessage", "createdAt",
			},
		}),
	}
}

// Init initializes the storage domain.
func (s *FilesystemAgentsStorage) Init(_ context.Context) error {
	s.helpers.Hydrate()
	return nil
}

// DangerouslyClearAll clears all data.
func (s *FilesystemAgentsStorage) DangerouslyClearAll(_ context.Context) error {
	return s.helpers.DangerouslyClearAll()
}

// GetByID retrieves an agent by ID.
func (s *FilesystemAgentsStorage) GetByID(_ context.Context, id string) (any, error) {
	return s.helpers.GetByID(id)
}

// Create creates a new agent with an initial version.
func (s *FilesystemAgentsStorage) Create(ctx context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}

	// Extract the agent from the input wrapper if present
	if agentMap, ok := inputMap["agent"].(map[string]any); ok {
		inputMap = agentMap
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("agent id is required")
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

	// Extract config fields (everything except agent-record fields)
	snapshotConfig := excludeKeys(inputMap, "id", "authorId", "metadata")
	filtered := stripUnusedFields(snapshotConfig)

	versionID := uuid.New().String()
	versionInput := map[string]any{
		"id":            versionID,
		"agentId":       id,
		"versionNumber": 1,
		"changedFields": mapKeys(filtered),
		"changeMessage": "Initial version",
	}
	for k, v := range filtered {
		versionInput[k] = v
	}

	if _, err := s.helpers.CreateVersion(versionInput); err != nil {
		return nil, err
	}

	return cloneMap(entity), nil
}

// Update updates an existing agent.
func (s *FilesystemAgentsStorage) Update(_ context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("agent id is required")
	}

	// Strip snapshot config fields that don't belong on the entity record
	entityFields := map[string]bool{
		"authorId": true, "metadata": true, "activeVersionId": true, "status": true,
	}
	entityUpdates := make(map[string]any)
	for key, value := range inputMap {
		if key == "id" {
			continue
		}
		if entityFields[key] {
			entityUpdates[key] = value
		}
	}

	return s.helpers.UpdateEntity(id, entityUpdates)
}

// Delete removes an agent by ID.
func (s *FilesystemAgentsStorage) Delete(_ context.Context, id string) error {
	return s.helpers.DeleteEntity(id)
}

// List lists agents with optional filtering.
func (s *FilesystemAgentsStorage) List(_ context.Context, args any) (any, error) {
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

	return s.helpers.ListEntities(page, perPage, orderBy, filters, "agents")
}

// CreateVersion creates a new agent version.
func (s *FilesystemAgentsStorage) CreateVersion(_ context.Context, input CreateAgentVersionInput) (*AgentVersion, error) {
	// Convert to map, strip unused fields from snapshot
	inputMap, _ := toMap(input)
	snapshotFields := excludeKeys(inputMap, "id", "agentId", "versionNumber", "changedFields", "changeMessage")
	filtered := stripUnusedFields(snapshotFields)

	versionInput := map[string]any{
		"id":            input.ID,
		"agentId":       input.AgentID,
		"versionNumber": input.VersionNumber,
		"changedFields": input.ChangedFields,
		"changeMessage": input.ChangeMessage,
	}
	for k, v := range filtered {
		versionInput[k] = v
	}

	result, err := s.helpers.CreateVersion(versionInput)
	if err != nil {
		return nil, err
	}
	return mapToAgentVersion(result), nil
}

// GetVersion retrieves a version by its ID.
func (s *FilesystemAgentsStorage) GetVersion(_ context.Context, id string) (*AgentVersion, error) {
	result, err := s.helpers.GetVersion(id)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToAgentVersion(result), nil
}

// GetVersionByNumber retrieves a version by agent ID and version number.
func (s *FilesystemAgentsStorage) GetVersionByNumber(_ context.Context, agentID string, versionNumber int) (*AgentVersion, error) {
	result, err := s.helpers.GetVersionByNumber(agentID, versionNumber)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToAgentVersion(result), nil
}

// GetLatestVersion retrieves the latest version for an agent.
func (s *FilesystemAgentsStorage) GetLatestVersion(_ context.Context, agentID string) (*AgentVersion, error) {
	result, err := s.helpers.GetLatestVersion(agentID)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToAgentVersion(result), nil
}

// ListVersions lists versions for an agent with pagination and sorting.
func (s *FilesystemAgentsStorage) ListVersions(_ context.Context, input ListVersionsInput) (*ListVersionsOutput, error) {
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

	result, err := s.helpers.ListVersions(input.AgentID, page, perPage, orderBy)
	if err != nil {
		return nil, err
	}

	versions := make([]AgentVersion, len(result.Versions))
	for i, v := range result.Versions {
		if m, ok := v.(map[string]any); ok {
			versions[i] = *mapToAgentVersion(m)
		}
	}

	return &ListVersionsOutput{
		Versions: versions,
		Total:    result.Total,
		Page:     result.Page,
		PerPage:  result.PerPage,
		HasMore:  result.HasMore,
	}, nil
}

// DeleteVersion removes a version by ID.
func (s *FilesystemAgentsStorage) DeleteVersion(_ context.Context, id string) error {
	return s.helpers.DeleteVersion(id)
}

// DeleteVersionsByParentID removes all versions for an agent.
func (s *FilesystemAgentsStorage) DeleteVersionsByParentID(_ context.Context, agentID string) error {
	return s.helpers.DeleteVersionsByParentID(agentID)
}

// CountVersions returns the number of versions for an agent.
func (s *FilesystemAgentsStorage) CountVersions(_ context.Context, agentID string) (int, error) {
	return s.helpers.CountVersions(agentID)
}

// GetByIDResolved resolves an entity by merging its thin record with the active or latest version config.
func (s *FilesystemAgentsStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
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
func (s *FilesystemAgentsStorage) ListResolved(ctx context.Context, args any) (any, error) {
	result, err := s.List(ctx, args)
	if err != nil {
		return nil, err
	}
	resultMap, ok := result.(map[string]any)
	if !ok {
		return result, nil
	}

	// Extract status from args for resolution.
	argsMap, _ := toMap(args)
	resolveStatus := strVal(argsMap, "status")

	entities, ok := resultMap["agents"].([]any)
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

	resultMap["agents"] = resolved
	return resultMap, nil
}

// resolveEntity merges a thin entity record with its active or latest version config.
func (s *FilesystemAgentsStorage) resolveEntity(_ context.Context, entityRaw any, status string) (any, error) {
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
				"id", "agentId", "versionNumber", "changedFields", "changeMessage", "createdAt",
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

func mapToAgentVersion(m map[string]any) *AgentVersion {
	v := &AgentVersion{}
	data, err := json.Marshal(m)
	if err != nil {
		return v
	}
	_ = json.Unmarshal(data, v)
	// Preserve snapshot fields that aren't in the typed struct
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
