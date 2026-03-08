// Ported from: packages/core/src/storage/domains/workspaces/inmemory.ts
package workspaces

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
var _ WorkspacesStorage = (*InMemoryWorkspacesStorage)(nil)

// ---------------------------------------------------------------------------
// Local entity type — placeholder until storage/types.go is ported.
// ---------------------------------------------------------------------------

// storageWorkspaceType is the thin workspace record stored in memory.
// TODO: Replace with StorageWorkspaceType from storage/types.go once ported.
type storageWorkspaceType struct {
	ID              string         `json:"id"`
	Status          string         `json:"status"`
	ActiveVersionID string         `json:"activeVersionId,omitempty"`
	AuthorID        string         `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// configFieldNames are the workspace config field names from StorageWorkspaceSnapshotType.
// When these are present in an update, a new version is automatically created.
var configFieldNames = []string{
	"name", "description", "filesystem", "sandbox", "mounts",
	"search", "skills", "tools", "autoSync", "operationTimeout",
}

// ---------------------------------------------------------------------------
// InMemoryWorkspacesStorage
// ---------------------------------------------------------------------------

// InMemoryWorkspacesStorage is an in-memory implementation of WorkspacesStorage.
type InMemoryWorkspacesStorage struct {
	mu         sync.RWMutex
	workspaces map[string]storageWorkspaceType
	versions   map[string]WorkspaceVersion
}

// NewInMemoryWorkspacesStorage creates a new InMemoryWorkspacesStorage.
func NewInMemoryWorkspacesStorage() *InMemoryWorkspacesStorage {
	return &InMemoryWorkspacesStorage{
		workspaces: make(map[string]storageWorkspaceType),
		versions:   make(map[string]WorkspaceVersion),
	}
}

func (s *InMemoryWorkspacesStorage) Init(_ context.Context) error { return nil }

func (s *InMemoryWorkspacesStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspaces = make(map[string]storageWorkspaceType)
	s.versions = make(map[string]WorkspaceVersion)
	return nil
}

// ==========================================================================
// Entity CRUD
// ==========================================================================

func (s *InMemoryWorkspacesStorage) GetByID(_ context.Context, id string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ws, ok := s.workspaces[id]
	if !ok {
		return nil, nil
	}
	return deepCopyEntity(ws), nil
}

func (s *InMemoryWorkspacesStorage) Create(ctx context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}
	if ws, ok := inputMap["workspace"].(map[string]any); ok {
		inputMap = ws
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	if _, exists := s.workspaces[id]; exists {
		return nil, fmt.Errorf("Workspace with id %s already exists", id)
	}

	now := time.Now()
	ws := storageWorkspaceType{
		ID:        id,
		Status:    "draft",
		AuthorID:  strVal(inputMap, "authorId"),
		Metadata:  mapVal(inputMap, "metadata"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.workspaces[id] = ws

	snapshotConfig := excludeKeys(inputMap, "id", "authorId", "metadata")
	versionID := uuid.New().String()
	if _, err := s.createVersionLocked(ctx, CreateWorkspaceVersionInput{
		ID:            versionID,
		WorkspaceID:   id,
		VersionNumber: 1,
		ChangedFields: mapKeys(snapshotConfig),
		ChangeMessage: "Initial version",
		Snapshot:      snapshotConfig,
	}); err != nil {
		return nil, err
	}

	return deepCopyEntity(ws), nil
}

// Update updates an existing workspace. If config fields are present in the update,
// a new version is automatically created (auto-versioning).
func (s *InMemoryWorkspacesStorage) Update(_ context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}
	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("workspace id is required")
	}
	existing, exists := s.workspaces[id]
	if !exists {
		return nil, fmt.Errorf("Workspace with id %s not found", id)
	}

	// Separate metadata fields from config fields.
	authorID := strVal(inputMap, "authorId")
	activeVersionID := strVal(inputMap, "activeVersionId")
	status := strVal(inputMap, "status")
	newMeta := mapVal(inputMap, "metadata")

	// Build configFields map from all non-metadata update keys.
	metadataKeys := map[string]bool{"id": true, "authorId": true, "activeVersionId": true, "metadata": true, "status": true}
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
	if _, ok := inputMap["authorId"]; ok {
		existing.AuthorID = authorID
	}
	if _, ok := inputMap["activeVersionId"]; ok {
		existing.ActiveVersionID = activeVersionID
	}
	if _, ok := inputMap["status"]; ok {
		existing.Status = status
	}
	if newMeta != nil {
		if existing.Metadata == nil {
			existing.Metadata = make(map[string]any)
		}
		for k, val := range newMeta {
			existing.Metadata[k] = val
		}
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
			s.workspaces[id] = existing
			return nil, fmt.Errorf("No versions found for workspace %s", id)
		}

		// Extract config from latest version via JSON round-trip.
		latestMap, _ := toMap(*latestVersion)
		versionMetaKeys := map[string]bool{
			"id": true, "workspaceId": true, "versionNumber": true,
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
			s.createVersionLocked(context.Background(), CreateWorkspaceVersionInput{
				ID:            newVersionID,
				WorkspaceID:   id,
				VersionNumber: newVersionNumber,
				ChangedFields: changedFields,
				ChangeMessage: fmt.Sprintf("Updated %s", strings.Join(changedFields, ", ")),
				Snapshot:      newConfig,
			})
		}
	}

	s.workspaces[id] = existing
	return deepCopyEntity(existing), nil
}

func (s *InMemoryWorkspacesStorage) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.workspaces, id)
	s.deleteVersionsByParentIDLocked(id)
	return nil
}

// List lists workspaces with optional filtering (no status filter, unlike other domains).
func (s *InMemoryWorkspacesStorage) List(_ context.Context, args any) (any, error) {
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

	var workspaces []storageWorkspaceType
	for _, ws := range s.workspaces {
		workspaces = append(workspaces, ws)
	}

	if authorIDFilter != "" {
		workspaces = filterSlice(workspaces, func(ws storageWorkspaceType) bool { return ws.AuthorID == authorIDFilter })
	}
	if len(metadataFilter) > 0 {
		workspaces = filterSlice(workspaces, func(ws storageWorkspaceType) bool {
			if ws.Metadata == nil {
				return false
			}
			for k, v := range metadataFilter {
				if !deepEqual(ws.Metadata[k], v) {
					return false
				}
			}
			return true
		})
	}

	sortEntities(workspaces, parsed.Field, parsed.Direction)

	cloned := make([]any, len(workspaces))
	for i, ws := range workspaces {
		cloned[i] = deepCopyEntity(ws)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, perPageInput, perPage)
	end := minInt(offset+perPage, total)
	start := minInt(offset, total)

	return map[string]any{
		"workspaces": cloned[start:end],
		"total":      total,
		"page":       page,
		"perPage":    perPageResp,
		"hasMore":    end < total,
	}, nil
}

// ==========================================================================
// Version Methods
// ==========================================================================

func (s *InMemoryWorkspacesStorage) CreateVersion(ctx context.Context, input CreateWorkspaceVersionInput) (*WorkspaceVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createVersionLocked(ctx, input)
}

func (s *InMemoryWorkspacesStorage) createVersionLocked(_ context.Context, input CreateWorkspaceVersionInput) (*WorkspaceVersion, error) {
	if _, exists := s.versions[input.ID]; exists {
		return nil, fmt.Errorf("Version with id %s already exists", input.ID)
	}
	for _, v := range s.versions {
		if v.WorkspaceID == input.WorkspaceID && v.VersionNumber == input.VersionNumber {
			return nil, fmt.Errorf("Version number %d already exists for workspace %s", input.VersionNumber, input.WorkspaceID)
		}
	}

	version := WorkspaceVersion{
		ID:            input.ID,
		WorkspaceID:   input.WorkspaceID,
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

func (s *InMemoryWorkspacesStorage) GetVersion(_ context.Context, id string) (*WorkspaceVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.versions[id]
	if !ok {
		return nil, nil
	}
	copied := deepCopyVersion(v)
	return &copied, nil
}

func (s *InMemoryWorkspacesStorage) GetVersionByNumber(_ context.Context, workspaceID string, versionNumber int) (*WorkspaceVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.versions {
		if v.WorkspaceID == workspaceID && v.VersionNumber == versionNumber {
			copied := deepCopyVersion(v)
			return &copied, nil
		}
	}
	return nil, nil
}

func (s *InMemoryWorkspacesStorage) GetLatestVersion(_ context.Context, workspaceID string) (*WorkspaceVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	latest := s.getLatestVersionLocked(workspaceID)
	if latest == nil {
		return nil, nil
	}
	copied := deepCopyVersion(*latest)
	return &copied, nil
}

func (s *InMemoryWorkspacesStorage) getLatestVersionLocked(workspaceID string) *WorkspaceVersion {
	var latest *WorkspaceVersion
	for _, v := range s.versions {
		if v.WorkspaceID == workspaceID {
			if latest == nil || v.VersionNumber > latest.VersionNumber {
				copied := v
				latest = &copied
			}
		}
	}
	return latest
}

func (s *InMemoryWorkspacesStorage) ListVersions(_ context.Context, input ListWorkspaceVersionsInput) (*ListWorkspaceVersionsOutput, error) {
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

	var versions []WorkspaceVersion
	for _, v := range s.versions {
		if v.WorkspaceID == input.WorkspaceID {
			versions = append(versions, v)
		}
	}

	sortVersions(versions, parsed.Field, parsed.Direction)

	cloned := make([]WorkspaceVersion, len(versions))
	for i, v := range versions {
		cloned[i] = deepCopyVersion(v)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, input.PerPage, perPage)
	end := minInt(offset+perPage, total)
	start := minInt(offset, total)

	return &ListWorkspaceVersionsOutput{
		Versions: cloned[start:end],
		Total:    total,
		Page:     page,
		PerPage:  perPageResp,
		HasMore:  end < total,
	}, nil
}

func (s *InMemoryWorkspacesStorage) DeleteVersion(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.versions, id)
	return nil
}

func (s *InMemoryWorkspacesStorage) DeleteVersionsByParentID(_ context.Context, workspaceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteVersionsByParentIDLocked(workspaceID)
	return nil
}

func (s *InMemoryWorkspacesStorage) deleteVersionsByParentIDLocked(workspaceID string) {
	var idsToDelete []string
	for id, v := range s.versions {
		if v.WorkspaceID == workspaceID {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(s.versions, id)
	}
}

func (s *InMemoryWorkspacesStorage) CountVersions(_ context.Context, workspaceID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, v := range s.versions {
		if v.WorkspaceID == workspaceID {
			count++
		}
	}
	return count, nil
}

// ==========================================================================
// Resolution Methods
// ==========================================================================

func (s *InMemoryWorkspacesStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
	entityRaw, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entityRaw == nil {
		return nil, nil
	}
	return s.resolveEntity(ctx, entityRaw, status)
}

func (s *InMemoryWorkspacesStorage) ListResolved(ctx context.Context, args any) (any, error) {
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

func (s *InMemoryWorkspacesStorage) resolveEntity(ctx context.Context, entityRaw any, status string) (any, error) {
	if status == "" {
		status = "published"
	}

	entityMap, ok := toMap(entityRaw)
	if !ok {
		return entityRaw, nil
	}

	entityID, _ := entityMap["id"].(string)

	var version *WorkspaceVersion
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
			"id": true, "workspaceId": true, "versionNumber": true,
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

func deepCopyEntity(e storageWorkspaceType) storageWorkspaceType {
	copied := e
	if e.Metadata != nil {
		copied.Metadata = make(map[string]any, len(e.Metadata))
		for k, v := range e.Metadata {
			copied.Metadata[k] = v
		}
	}
	return copied
}

func deepCopyVersion(v WorkspaceVersion) WorkspaceVersion {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out WorkspaceVersion
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

func sortEntities(entities []storageWorkspaceType, field domains.ThreadOrderBy, direction domains.SortDirection) {
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

func sortVersions(versions []WorkspaceVersion, field domains.VersionOrderBy, direction domains.SortDirection) {
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

func deepEqual(a, b any) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}
