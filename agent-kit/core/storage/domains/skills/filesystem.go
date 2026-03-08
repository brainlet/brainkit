// Ported from: packages/core/src/storage/domains/skills/filesystem.ts
package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
	"github.com/brainlet/brainkit/agent-kit/core/storage/fsutil"
)

// Compile-time interface check.
var _ SkillsStorage = (*FilesystemSkillsStorage)(nil)

// skillConfigFieldNames are the config field names from StorageSkillSnapshotType.
// Used during update to detect config changes that require a new version.
var skillConfigFieldNames = []string{
	"name",
	"description",
	"instructions",
	"license",
	"compatibility",
	"source",
	"references",
	"scripts",
	"assets",
	"metadata",
	"tree",
}

// ---------------------------------------------------------------------------
// FilesystemSkillsStorage
// ---------------------------------------------------------------------------

// FilesystemSkillsStorage is a filesystem-backed implementation of SkillsStorage.
type FilesystemSkillsStorage struct {
	helpers *fsutil.FilesystemVersionedHelpers
}

// NewFilesystemSkillsStorage creates a new FilesystemSkillsStorage.
func NewFilesystemSkillsStorage(db *fsutil.FilesystemDB) *FilesystemSkillsStorage {
	return &FilesystemSkillsStorage{
		helpers: fsutil.NewFilesystemVersionedHelpers(fsutil.FilesystemVersionedConfig{
			DB:            db,
			EntitiesFile:  "skills.json",
			ParentIDField: "skillId",
			Name:          "FilesystemSkillsStorage",
			VersionMetadataFields: []string{
				"id", "skillId", "versionNumber", "changedFields", "changeMessage", "createdAt",
			},
		}),
	}
}

// Init initializes the storage domain.
func (s *FilesystemSkillsStorage) Init(_ context.Context) error {
	s.helpers.Hydrate()
	return nil
}

// DangerouslyClearAll clears all data.
func (s *FilesystemSkillsStorage) DangerouslyClearAll(_ context.Context) error {
	return s.helpers.DangerouslyClearAll()
}

// GetByID retrieves a skill by ID.
func (s *FilesystemSkillsStorage) GetByID(_ context.Context, id string) (any, error) {
	return s.helpers.GetByID(id)
}

// Create creates a new skill with an initial version.
func (s *FilesystemSkillsStorage) Create(ctx context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}

	if skillMap, ok := inputMap["skill"].(map[string]any); ok {
		inputMap = skillMap
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("skill id is required")
	}

	now := time.Now()
	entity := map[string]any{
		"id":        id,
		"status":    "draft",
		"authorId":  inputMap["authorId"],
		"createdAt": now,
		"updatedAt": now,
	}

	if _, err := s.helpers.CreateEntity(id, entity); err != nil {
		return nil, err
	}

	// Skills don't have metadata on the thin record, so only exclude id and authorId.
	snapshotConfig := excludeKeys(inputMap, "id", "authorId")
	versionID := uuid.New().String()
	versionInput := map[string]any{
		"id":            versionID,
		"skillId":       id,
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

// Update updates an existing skill.
// If config fields are present in the update, a new version is automatically created.
func (s *FilesystemSkillsStorage) Update(_ context.Context, input any) (any, error) {
	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("skill id is required")
	}

	existing, err := s.helpers.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("FilesystemSkillsStorage: skill with id %s not found", id)
	}

	updates := excludeKeys(inputMap, "id")

	// Separate entity-level fields from config fields.
	authorID := updates["authorId"]
	activeVersionID := updates["activeVersionId"]
	status := updates["status"]

	configFields := excludeKeys(updates, "authorId", "activeVersionId", "status")

	// Check if any config fields are present.
	hasConfigUpdate := false
	for _, field := range skillConfigFieldNames {
		if _, ok := configFields[field]; ok {
			hasConfigUpdate = true
			break
		}
	}

	// If config fields are being updated, create a new version.
	if hasConfigUpdate {
		latestVersion, err := s.helpers.GetLatestVersion(id)
		if err != nil {
			return nil, err
		}
		if latestVersion == nil {
			return nil, fmt.Errorf("no versions found for skill %s", id)
		}

		// Extract version metadata fields to get just the config from the latest version.
		latestConfig := make(map[string]any)
		for k, v := range latestVersion {
			switch k {
			case "id", "skillId", "versionNumber", "changedFields", "changeMessage", "createdAt":
				continue
			default:
				latestConfig[k] = v
			}
		}

		// Merge latest config with new config fields.
		newConfig := cloneMap(latestConfig)
		for k, v := range configFields {
			newConfig[k] = v
		}

		// Determine which fields actually changed.
		var changedFields []string
		for _, field := range skillConfigFieldNames {
			newVal, inNew := configFields[field]
			if !inNew {
				continue
			}
			oldVal, inOld := latestConfig[field]
			if !inOld {
				changedFields = append(changedFields, field)
				continue
			}
			// Compare via JSON serialization (matching TS behavior).
			newJSON, _ := json.Marshal(newVal)
			oldJSON, _ := json.Marshal(oldVal)
			if string(newJSON) != string(oldJSON) {
				changedFields = append(changedFields, field)
			}
		}

		if len(changedFields) > 0 {
			newVersionID := uuid.New().String()
			versionNumber, _ := latestVersion["versionNumber"]
			var newVersionNumber int
			switch n := versionNumber.(type) {
			case int:
				newVersionNumber = n + 1
			case float64:
				newVersionNumber = int(n) + 1
			default:
				newVersionNumber = 2
			}

			versionInput := map[string]any{
				"id":            newVersionID,
				"skillId":       id,
				"versionNumber": newVersionNumber,
				"changedFields": changedFields,
				"changeMessage": "Updated " + strings.Join(changedFields, ", "),
			}
			for k, v := range newConfig {
				versionInput[k] = v
			}

			if _, err := s.helpers.CreateVersion(versionInput); err != nil {
				return nil, err
			}
		}
	}

	// Build the entity-level updates for the helpers.
	entityUpdates := make(map[string]any)
	if authorID != nil {
		entityUpdates["authorId"] = authorID
	}
	if activeVersionID != nil {
		entityUpdates["activeVersionId"] = activeVersionID
	}
	if status != nil {
		entityUpdates["status"] = status
	}
	// Auto-set status to 'published' when activeVersionId is set without explicit status.
	if activeVersionID != nil && status == nil {
		entityUpdates["status"] = "published"
	}

	return s.helpers.UpdateEntity(id, entityUpdates)
}

// Delete removes a skill by ID.
func (s *FilesystemSkillsStorage) Delete(_ context.Context, id string) error {
	return s.helpers.DeleteEntity(id)
}

// List lists skills with optional filtering.
func (s *FilesystemSkillsStorage) List(_ context.Context, args any) (any, error) {
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

	return s.helpers.ListEntities(page, perPage, orderBy, filters, "skills")
}

// CreateVersion creates a new skill version.
func (s *FilesystemSkillsStorage) CreateVersion(_ context.Context, input CreateSkillVersionInput) (*SkillVersion, error) {
	inputMap, _ := toMap(input)
	result, err := s.helpers.CreateVersion(inputMap)
	if err != nil {
		return nil, err
	}
	return mapToSkillVersion(result), nil
}

// GetVersion retrieves a version by its ID.
func (s *FilesystemSkillsStorage) GetVersion(_ context.Context, id string) (*SkillVersion, error) {
	result, err := s.helpers.GetVersion(id)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToSkillVersion(result), nil
}

// GetVersionByNumber retrieves a version by skill ID and version number.
func (s *FilesystemSkillsStorage) GetVersionByNumber(_ context.Context, skillID string, versionNumber int) (*SkillVersion, error) {
	result, err := s.helpers.GetVersionByNumber(skillID, versionNumber)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToSkillVersion(result), nil
}

// GetLatestVersion retrieves the latest version for a skill.
func (s *FilesystemSkillsStorage) GetLatestVersion(_ context.Context, skillID string) (*SkillVersion, error) {
	result, err := s.helpers.GetLatestVersion(skillID)
	if err != nil || result == nil {
		return nil, err
	}
	return mapToSkillVersion(result), nil
}

// ListVersions lists versions for a skill with pagination and sorting.
func (s *FilesystemSkillsStorage) ListVersions(_ context.Context, input ListSkillVersionsInput) (*ListSkillVersionsOutput, error) {
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

	result, err := s.helpers.ListVersions(input.SkillID, page, perPage, orderBy)
	if err != nil {
		return nil, err
	}

	versions := make([]SkillVersion, len(result.Versions))
	for i, v := range result.Versions {
		if m, ok := v.(map[string]any); ok {
			versions[i] = *mapToSkillVersion(m)
		}
	}

	return &ListSkillVersionsOutput{
		Versions: versions,
		Total:    result.Total,
		Page:     result.Page,
		PerPage:  result.PerPage,
		HasMore:  result.HasMore,
	}, nil
}

// DeleteVersion removes a version by ID.
func (s *FilesystemSkillsStorage) DeleteVersion(_ context.Context, id string) error {
	return s.helpers.DeleteVersion(id)
}

// DeleteVersionsByParentID removes all versions for a skill.
func (s *FilesystemSkillsStorage) DeleteVersionsByParentID(_ context.Context, skillID string) error {
	return s.helpers.DeleteVersionsByParentID(skillID)
}

// CountVersions returns the number of versions for a skill.
func (s *FilesystemSkillsStorage) CountVersions(_ context.Context, skillID string) (int, error) {
	return s.helpers.CountVersions(skillID)
}

// GetByIDResolved resolves an entity by merging its thin record with the active or latest version config.
func (s *FilesystemSkillsStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
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
func (s *FilesystemSkillsStorage) ListResolved(ctx context.Context, args any) (any, error) {
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

	entities, ok := resultMap["skills"].([]any)
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

	resultMap["skills"] = resolved
	return resultMap, nil
}

// resolveEntity merges a thin entity record with its active or latest version config.
func (s *FilesystemSkillsStorage) resolveEntity(_ context.Context, entityRaw any, status string) (any, error) {
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
				"id", "skillId", "versionNumber", "changedFields", "changeMessage", "createdAt",
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

func mapToSkillVersion(m map[string]any) *SkillVersion {
	v := &SkillVersion{}
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
