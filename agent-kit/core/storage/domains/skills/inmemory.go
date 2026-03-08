// Ported from: packages/core/src/storage/domains/skills/inmemory.ts
package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// Compile-time interface check.
var _ SkillsStorage = (*InMemorySkillsStorage)(nil)

// ---------------------------------------------------------------------------
// Local entity type — placeholder until storage/types.go is ported.
// ---------------------------------------------------------------------------

// storageSkillType is the thin skill record stored in memory.
// Note: Skills do NOT have a metadata field on the thin record (unlike other domains).
// TODO: Replace with StorageSkillType from storage/types.go once ported.
type storageSkillType struct {
	ID              string    `json:"id"`
	Status          string    `json:"status"`
	ActiveVersionID string    `json:"activeVersionId,omitempty"`
	AuthorID        string    `json:"authorId,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// configFieldNames are the skill config field names from StorageSkillSnapshotType.
// When these are present in an update, a new version is automatically created.
var configFieldNames = []string{
	"name", "description", "instructions", "license", "compatibility",
	"source", "references", "scripts", "assets", "metadata", "tree",
}

// ---------------------------------------------------------------------------
// InMemorySkillsStorage
// ---------------------------------------------------------------------------

// InMemorySkillsStorage is an in-memory implementation of SkillsStorage.
type InMemorySkillsStorage struct {
	mu       sync.RWMutex
	skills   map[string]storageSkillType
	versions map[string]SkillVersion
}

// NewInMemorySkillsStorage creates a new InMemorySkillsStorage.
func NewInMemorySkillsStorage() *InMemorySkillsStorage {
	return &InMemorySkillsStorage{
		skills:   make(map[string]storageSkillType),
		versions: make(map[string]SkillVersion),
	}
}

func (s *InMemorySkillsStorage) Init(_ context.Context) error { return nil }

func (s *InMemorySkillsStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skills = make(map[string]storageSkillType)
	s.versions = make(map[string]SkillVersion)
	return nil
}

// ==========================================================================
// Entity CRUD
// ==========================================================================

func (s *InMemorySkillsStorage) GetByID(_ context.Context, id string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sk, ok := s.skills[id]
	if !ok {
		return nil, nil
	}
	return deepCopyEntity(sk), nil
}

// Create creates a new skill with an initial version.
// If version creation fails, the skill record is rolled back.
func (s *InMemorySkillsStorage) Create(ctx context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}
	if sk, ok := inputMap["skill"].(map[string]any); ok {
		inputMap = sk
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("skill id is required")
	}
	if _, exists := s.skills[id]; exists {
		return nil, fmt.Errorf("Skill with id %s already exists", id)
	}

	now := time.Now()
	skill := storageSkillType{
		ID:        id,
		Status:    "draft",
		AuthorID:  strVal(inputMap, "authorId"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.skills[id] = skill

	// Extract config fields (skills don't have metadata on thin record).
	snapshotConfig := excludeKeys(inputMap, "id", "authorId")

	// Create version 1 — with rollback on failure.
	versionID := uuid.New().String()
	_, err := s.createVersionLocked(ctx, CreateSkillVersionInput{
		ID:            versionID,
		SkillID:       id,
		VersionNumber: 1,
		ChangedFields: mapKeys(snapshotConfig),
		ChangeMessage: "Initial version",
		Snapshot:      snapshotConfig,
	})
	if err != nil {
		// Roll back the orphaned skill record.
		delete(s.skills, id)
		return nil, err
	}

	return deepCopyEntity(skill), nil
}

// Update updates an existing skill. If config fields are present in the update,
// a new version is automatically created (auto-versioning).
func (s *InMemorySkillsStorage) Update(_ context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}
	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("skill id is required")
	}
	existing, exists := s.skills[id]
	if !exists {
		return nil, fmt.Errorf("Skill with id %s not found", id)
	}

	// Separate metadata fields from config fields.
	metadataKeys := map[string]bool{"id": true, "authorId": true, "activeVersionId": true, "status": true}
	configFields := make(map[string]any)
	for k, v := range inputMap {
		if !metadataKeys[k] {
			configFields[k] = v
		}
	}

	// Check if any config field is present.
	hasConfigUpdate := false
	configFieldSet := make(map[string]bool, len(configFieldNames))
	for _, f := range configFieldNames {
		configFieldSet[f] = true
	}
	for k := range configFields {
		if configFieldSet[k] {
			hasConfigUpdate = true
			break
		}
	}

	// Update metadata fields on the record.
	if v, ok := inputMap["authorId"]; ok {
		existing.AuthorID, _ = v.(string)
	}
	if v, ok := inputMap["activeVersionId"]; ok {
		existing.ActiveVersionID, _ = v.(string)
	}
	if v, ok := inputMap["status"]; ok {
		existing.Status, _ = v.(string)
	}
	existing.UpdatedAt = time.Now()

	// Auto-set status to 'published' when activeVersionId is set, only if status is not explicitly provided.
	if _, avSet := inputMap["activeVersionId"]; avSet {
		if _, statusSet := inputMap["status"]; !statusSet {
			existing.Status = "published"
		}
	}

	// If config fields are being updated, create a new version (auto-versioning).
	if hasConfigUpdate {
		latestVersion := s.getLatestVersionLocked(id)
		if latestVersion == nil {
			s.skills[id] = existing
			return nil, fmt.Errorf("No versions found for skill %s", id)
		}

		// Extract config from latest version via JSON round-trip.
		latestMap, _ := toMap(*latestVersion)
		versionMetaKeys := map[string]bool{
			"id": true, "skillId": true, "versionNumber": true,
			"changedFields": true, "changeMessage": true, "createdAt": true,
		}
		latestConfig := make(map[string]any)
		for k, v := range latestMap {
			if !versionMetaKeys[k] {
				latestConfig[k] = v
			}
		}

		// Merge updates into latest config.
		newConfig := make(map[string]any)
		for k, v := range latestConfig {
			newConfig[k] = v
		}
		for k, v := range configFields {
			newConfig[k] = v
		}

		// Identify which fields actually changed.
		var changedFields []string
		for _, field := range configFieldNames {
			if _, ok := configFields[field]; ok {
				oldJSON, _ := json.Marshal(latestConfig[field])
				newJSON, _ := json.Marshal(configFields[field])
				if string(oldJSON) != string(newJSON) {
					changedFields = append(changedFields, field)
				}
			}
		}

		// Only create a new version if something actually changed.
		if len(changedFields) > 0 {
			newVersionID := uuid.New().String()
			newVersionNumber := latestVersion.VersionNumber + 1
			s.createVersionLocked(context.Background(), CreateSkillVersionInput{
				ID:            newVersionID,
				SkillID:       id,
				VersionNumber: newVersionNumber,
				ChangedFields: changedFields,
				ChangeMessage: fmt.Sprintf("Updated %s", strings.Join(changedFields, ", ")),
				Snapshot:      newConfig,
			})
		}
	}

	s.skills[id] = existing
	return deepCopyEntity(existing), nil
}

func (s *InMemorySkillsStorage) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.skills, id)
	s.deleteVersionsByParentIDLocked(id)
	return nil
}

// List lists skills with optional filtering (no status filter, metadata filter always returns false).
func (s *InMemorySkillsStorage) List(_ context.Context, args any) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	argsMap, _ := toMap(args)
	page := intVal(argsMap, "page", 0)
	perPageInput := optionalIntVal(argsMap, "perPage")
	orderByMap, _ := argsMap["orderBy"].(map[string]any)
	authorIDFilter := strVal(argsMap, "authorId")
	metadataFilter := mapVal(argsMap, "metadata")

	var orderBy *domains.StorageOrderBy
	if orderByMap != nil {
		orderBy = &domains.StorageOrderBy{
			Field:     domains.ThreadOrderBy(strVal(orderByMap, "field")),
			Direction: domains.SortDirection(strVal(orderByMap, "direction")),
		}
	}
	base := domains.VersionedStorageDomainBase{}
	parsed := base.ParseOrderBy(orderBy, domains.SortDESC)
	perPage := normalizePerPage(perPageInput, 100)

	if page < 0 {
		return nil, fmt.Errorf("page must be >= 0")
	}

	var skills []storageSkillType
	for _, sk := range s.skills {
		skills = append(skills, sk)
	}

	if authorIDFilter != "" {
		skills = filterSlice(skills, func(sk storageSkillType) bool { return sk.AuthorID == authorIDFilter })
	}

	// StorageSkillType doesn't have metadata on the thin record — filter always returns false.
	if len(metadataFilter) > 0 {
		skills = filterSlice(skills, func(_ storageSkillType) bool { return false })
	}

	sortEntities(skills, parsed.Field, parsed.Direction)

	cloned := make([]any, len(skills))
	for i, sk := range skills {
		cloned[i] = deepCopyEntity(sk)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, perPageInput, perPage)
	end := minInt(offset+perPage, total)
	start := minInt(offset, total)

	return map[string]any{
		"skills":  cloned[start:end],
		"total":   total,
		"page":    page,
		"perPage": perPageResp,
		"hasMore": end < total,
	}, nil
}

// ==========================================================================
// Version Methods
// ==========================================================================

func (s *InMemorySkillsStorage) CreateVersion(ctx context.Context, input CreateSkillVersionInput) (*SkillVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createVersionLocked(ctx, input)
}

func (s *InMemorySkillsStorage) createVersionLocked(_ context.Context, input CreateSkillVersionInput) (*SkillVersion, error) {
	if _, exists := s.versions[input.ID]; exists {
		return nil, fmt.Errorf("Version with id %s already exists", input.ID)
	}
	for _, v := range s.versions {
		if v.SkillID == input.SkillID && v.VersionNumber == input.VersionNumber {
			return nil, fmt.Errorf("Version number %d already exists for skill %s", input.VersionNumber, input.SkillID)
		}
	}

	version := SkillVersion{
		ID:            input.ID,
		SkillID:       input.SkillID,
		VersionNumber: input.VersionNumber,
		ChangedFields: input.ChangedFields,
		ChangeMessage: input.ChangeMessage,
		CreatedAt:     time.Now(),
		Snapshot:      input.Snapshot,
	}

	s.versions[input.ID] = deepCopyVersion(version)
	copied := deepCopyVersion(version)
	return &copied, nil
}

func (s *InMemorySkillsStorage) GetVersion(_ context.Context, id string) (*SkillVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.versions[id]
	if !ok {
		return nil, nil
	}
	copied := deepCopyVersion(v)
	return &copied, nil
}

func (s *InMemorySkillsStorage) GetVersionByNumber(_ context.Context, skillID string, versionNumber int) (*SkillVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.versions {
		if v.SkillID == skillID && v.VersionNumber == versionNumber {
			copied := deepCopyVersion(v)
			return &copied, nil
		}
	}
	return nil, nil
}

func (s *InMemorySkillsStorage) GetLatestVersion(_ context.Context, skillID string) (*SkillVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	latest := s.getLatestVersionLocked(skillID)
	if latest == nil {
		return nil, nil
	}
	copied := deepCopyVersion(*latest)
	return &copied, nil
}

func (s *InMemorySkillsStorage) getLatestVersionLocked(skillID string) *SkillVersion {
	var latest *SkillVersion
	for _, v := range s.versions {
		if v.SkillID == skillID {
			if latest == nil || v.VersionNumber > latest.VersionNumber {
				copied := v
				latest = &copied
			}
		}
	}
	return latest
}

func (s *InMemorySkillsStorage) ListVersions(_ context.Context, input ListSkillVersionsInput) (*ListSkillVersionsOutput, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	page := 0
	if input.Page != nil {
		page = *input.Page
	}
	if page < 0 {
		return nil, fmt.Errorf("page must be >= 0")
	}
	perPage := normalizePerPage(input.PerPage, 20)

	var orderByClause *domains.VersionOrderByClause
	if input.OrderByField != nil || input.OrderByDirection != nil {
		orderByClause = &domains.VersionOrderByClause{}
		if input.OrderByField != nil {
			orderByClause.Field = domains.VersionOrderBy(*input.OrderByField)
		}
		if input.OrderByDirection != nil {
			orderByClause.Direction = domains.SortDirection(*input.OrderByDirection)
		}
	}
	base := domains.VersionedStorageDomainBase{}
	parsed := base.ParseVersionOrderBy(orderByClause, domains.SortDESC)

	var versions []SkillVersion
	for _, v := range s.versions {
		if v.SkillID == input.SkillID {
			versions = append(versions, v)
		}
	}

	sortVersions(versions, parsed.Field, parsed.Direction)

	cloned := make([]SkillVersion, len(versions))
	for i, v := range versions {
		cloned[i] = deepCopyVersion(v)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, input.PerPage, perPage)
	end := minInt(offset+perPage, total)
	start := minInt(offset, total)

	return &ListSkillVersionsOutput{
		Versions: cloned[start:end],
		Total:    total,
		Page:     page,
		PerPage:  perPageResp,
		HasMore:  end < total,
	}, nil
}

func (s *InMemorySkillsStorage) DeleteVersion(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.versions, id)
	return nil
}

func (s *InMemorySkillsStorage) DeleteVersionsByParentID(_ context.Context, skillID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteVersionsByParentIDLocked(skillID)
	return nil
}

func (s *InMemorySkillsStorage) deleteVersionsByParentIDLocked(skillID string) {
	var idsToDelete []string
	for id, v := range s.versions {
		if v.SkillID == skillID {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(s.versions, id)
	}
}

func (s *InMemorySkillsStorage) CountVersions(_ context.Context, skillID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, v := range s.versions {
		if v.SkillID == skillID {
			count++
		}
	}
	return count, nil
}

// ==========================================================================
// Resolution Methods
// ==========================================================================

func (s *InMemorySkillsStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
	entityRaw, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entityRaw == nil {
		return nil, nil
	}
	return s.resolveEntity(ctx, entityRaw, status)
}

func (s *InMemorySkillsStorage) ListResolved(ctx context.Context, args any) (any, error) {
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

func (s *InMemorySkillsStorage) resolveEntity(ctx context.Context, entityRaw any, status string) (any, error) {
	if status == "" {
		status = "published"
	}

	entityMap, ok := toMap(entityRaw)
	if !ok {
		return entityRaw, nil
	}

	entityID, _ := entityMap["id"].(string)

	var version *SkillVersion
	if status == "draft" {
		version, _ = s.GetLatestVersion(ctx, entityID)
	} else {
		activeVersionID, _ := entityMap["activeVersionId"].(string)
		if activeVersionID != "" {
			version, _ = s.GetVersion(ctx, activeVersionID)
		}
		if version == nil {
			version, _ = s.GetLatestVersion(ctx, entityID)
		}
	}

	if version != nil {
		versionMetadataFields := map[string]bool{
			"id": true, "skillId": true, "versionNumber": true,
			"changedFields": true, "changeMessage": true, "createdAt": true,
		}
		versionMap, _ := toMap(*version)
		snapshotConfig := make(map[string]any)
		for k, v := range versionMap {
			if !versionMetadataFields[k] {
				snapshotConfig[k] = v
			}
		}

		merged := make(map[string]any)
		for k, v := range entityMap {
			merged[k] = v
		}
		for k, v := range snapshotConfig {
			merged[k] = v
		}
		merged["resolvedVersionId"] = version.ID
		return merged, nil
	}

	return entityMap, nil
}

// ==========================================================================
// Helpers
// ==========================================================================

func deepCopyEntity(e storageSkillType) storageSkillType {
	return e // storageSkillType has no reference fields (no metadata on thin record).
}

func deepCopyVersion(v SkillVersion) SkillVersion {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out SkillVersion
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

func sortEntities(entities []storageSkillType, field domains.ThreadOrderBy, direction domains.SortDirection) {
	sort.Slice(entities, func(i, j int) bool {
		var aVal, bVal time.Time
		if field == domains.ThreadOrderByUpdatedAt {
			aVal, bVal = entities[i].UpdatedAt, entities[j].UpdatedAt
		} else {
			aVal, bVal = entities[i].CreatedAt, entities[j].CreatedAt
		}
		if direction == domains.SortASC {
			return aVal.Before(bVal)
		}
		return bVal.Before(aVal)
	})
}

func sortVersions(versions []SkillVersion, field domains.VersionOrderBy, direction domains.SortDirection) {
	sort.Slice(versions, func(i, j int) bool {
		var aVal, bVal float64
		if field == domains.VersionOrderByCreatedAt {
			aVal, bVal = float64(versions[i].CreatedAt.UnixNano()), float64(versions[j].CreatedAt.UnixNano())
		} else {
			aVal, bVal = float64(versions[i].VersionNumber), float64(versions[j].VersionNumber)
		}
		if direction == domains.SortASC {
			return aVal < bVal
		}
		return bVal < aVal
	})
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func toMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	if m, ok := v.(map[string]any); ok {
		return m, true
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false
	}
	return m, true
}

func strVal(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func intVal(m map[string]any, key string, defaultVal int) int {
	if m == nil {
		return defaultVal
	}
	v, ok := m[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	}
	return defaultVal
}

func optionalIntVal(m map[string]any, key string) *int {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	switch n := v.(type) {
	case int:
		return &n
	case float64:
		i := int(n)
		return &i
	case int64:
		i := int(n)
		return &i
	}
	return nil
}

func mapVal(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, _ := m[key].(map[string]any)
	return v
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func excludeKeys(m map[string]any, keys ...string) map[string]any {
	excluded := make(map[string]bool, len(keys))
	for _, k := range keys {
		excluded[k] = true
	}
	result := make(map[string]any)
	for k, v := range m {
		if !excluded[k] {
			result[k] = v
		}
	}
	return result
}

func filterSlice[T any](s []T, pred func(T) bool) []T {
	var result []T
	for _, v := range s {
		if pred(v) {
			result = append(result, v)
		}
	}
	return result
}

func normalizePerPage(perPage *int, defaultVal int) int {
	if perPage == nil {
		return defaultVal
	}
	if *perPage == domains.PerPageDisabled {
		return math.MaxInt
	}
	return *perPage
}

func calculatePagination(page int, perPageInput *int, normalizedPerPage int) (int, int) {
	offset := page * normalizedPerPage
	perPageResp := normalizedPerPage
	if perPageInput != nil && *perPageInput == domains.PerPageDisabled {
		perPageResp = domains.PerPageDisabled
	}
	return offset, perPageResp
}
