// Ported from: packages/core/src/storage/domains/scorer-definitions/inmemory.ts
package scorerdefinitions

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	domains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// Compile-time interface check.
var _ ScorerDefinitionsStorage = (*InMemoryScorerDefinitionsStorage)(nil)

// ---------------------------------------------------------------------------
// Local entity type — mirrors storage.StorageScorerDefinitionType.
// Defined locally to avoid circular import (storage -> scorerdefinitions -> storage).
// ---------------------------------------------------------------------------

// storageScorerDefinitionType is the thin scorer definition record stored in memory.
// Fields match storage.StorageScorerDefinitionType.
type storageScorerDefinitionType struct {
	ID              string         `json:"id"`
	Status          string         `json:"status"`
	ActiveVersionID string         `json:"activeVersionId,omitempty"`
	AuthorID        string         `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// ---------------------------------------------------------------------------
// InMemoryScorerDefinitionsStorage
// ---------------------------------------------------------------------------

// InMemoryScorerDefinitionsStorage is an in-memory implementation of ScorerDefinitionsStorage.
type InMemoryScorerDefinitionsStorage struct {
	mu       sync.RWMutex
	scorers  map[string]storageScorerDefinitionType
	versions map[string]ScorerDefinitionVersion
}

// NewInMemoryScorerDefinitionsStorage creates a new InMemoryScorerDefinitionsStorage.
func NewInMemoryScorerDefinitionsStorage() *InMemoryScorerDefinitionsStorage {
	return &InMemoryScorerDefinitionsStorage{
		scorers:  make(map[string]storageScorerDefinitionType),
		versions: make(map[string]ScorerDefinitionVersion),
	}
}

func (s *InMemoryScorerDefinitionsStorage) Init(_ context.Context) error { return nil }

func (s *InMemoryScorerDefinitionsStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scorers = make(map[string]storageScorerDefinitionType)
	s.versions = make(map[string]ScorerDefinitionVersion)
	return nil
}

// ==========================================================================
// Entity CRUD
// ==========================================================================

func (s *InMemoryScorerDefinitionsStorage) GetByID(_ context.Context, id string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	scorer, ok := s.scorers[id]
	if !ok {
		return nil, nil
	}
	return deepCopyEntity(scorer), nil
}

func (s *InMemoryScorerDefinitionsStorage) Create(ctx context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}
	if sd, ok := inputMap["scorerDefinition"].(map[string]any); ok {
		inputMap = sd
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("scorer definition id is required")
	}
	if _, exists := s.scorers[id]; exists {
		return nil, fmt.Errorf("Scorer definition with id %s already exists", id)
	}

	now := time.Now()
	scorer := storageScorerDefinitionType{
		ID:        id,
		Status:    "draft",
		AuthorID:  strVal(inputMap, "authorId"),
		Metadata:  mapVal(inputMap, "metadata"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.scorers[id] = scorer

	snapshotConfig := excludeKeys(inputMap, "id", "authorId", "metadata")
	versionID := uuid.New().String()

	// Populate the embedded snapshot config from the map.
	var snapCfg ScorerDefinitionSnapshotConfig
	if snapBytes, err := json.Marshal(snapshotConfig); err == nil {
		_ = json.Unmarshal(snapBytes, &snapCfg)
	}

	if _, err := s.createVersionLocked(ctx, CreateScorerDefinitionVersionInput{
		ID:                             versionID,
		ScorerDefinitionID:             id,
		VersionNumber:                  1,
		ChangedFields:                  mapKeys(snapshotConfig),
		ChangeMessage:                  "Initial version",
		ScorerDefinitionSnapshotConfig: snapCfg,
	}); err != nil {
		return nil, err
	}

	return deepCopyEntity(scorer), nil
}

func (s *InMemoryScorerDefinitionsStorage) Update(_ context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}
	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("scorer definition id is required")
	}
	existing, exists := s.scorers[id]
	if !exists {
		return nil, fmt.Errorf("Scorer definition with id %s not found", id)
	}

	if v, ok := inputMap["authorId"]; ok {
		existing.AuthorID, _ = v.(string)
	}
	if v, ok := inputMap["activeVersionId"]; ok {
		existing.ActiveVersionID, _ = v.(string)
	}
	if v, ok := inputMap["status"]; ok {
		existing.Status, _ = v.(string)
	}
	if v, ok := inputMap["metadata"]; ok {
		if newMeta, ok := v.(map[string]any); ok {
			if existing.Metadata == nil {
				existing.Metadata = make(map[string]any)
			}
			for k, val := range newMeta {
				existing.Metadata[k] = val
			}
		}
	}
	existing.UpdatedAt = time.Now()

	s.scorers[id] = existing
	return deepCopyEntity(existing), nil
}

func (s *InMemoryScorerDefinitionsStorage) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.scorers, id)
	s.deleteVersionsByParentIDLocked(id)
	return nil
}

func (s *InMemoryScorerDefinitionsStorage) List(_ context.Context, args any) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	argsMap, _ := toMap(args)
	page := intVal(argsMap, "page", 0)
	perPageInput := optionalIntVal(argsMap, "perPage")
	orderByMap, _ := argsMap["orderBy"].(map[string]any)
	statusFilter := strVal(argsMap, "status")
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

	var scorers []storageScorerDefinitionType
	for _, sc := range s.scorers {
		scorers = append(scorers, sc)
	}

	if statusFilter != "" {
		scorers = filterSlice(scorers, func(sc storageScorerDefinitionType) bool { return sc.Status == statusFilter })
	}
	if authorIDFilter != "" {
		scorers = filterSlice(scorers, func(sc storageScorerDefinitionType) bool { return sc.AuthorID == authorIDFilter })
	}
	if len(metadataFilter) > 0 {
		scorers = filterSlice(scorers, func(sc storageScorerDefinitionType) bool {
			if sc.Metadata == nil {
				return false
			}
			for k, v := range metadataFilter {
				if !deepEqual(sc.Metadata[k], v) {
					return false
				}
			}
			return true
		})
	}

	sortEntities(scorers, parsed.Field, parsed.Direction)

	cloned := make([]any, len(scorers))
	for i, sc := range scorers {
		cloned[i] = deepCopyEntity(sc)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, perPageInput, perPage)
	end := min(offset+perPage, total)
	start := min(offset, total)

	return map[string]any{
		"scorerDefinitions": cloned[start:end],
		"total":             total,
		"page":              page,
		"perPage":           perPageResp,
		"hasMore":           end < total,
	}, nil
}

// ==========================================================================
// Version Methods
// ==========================================================================

func (s *InMemoryScorerDefinitionsStorage) CreateVersion(ctx context.Context, input CreateScorerDefinitionVersionInput) (*ScorerDefinitionVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createVersionLocked(ctx, input)
}

func (s *InMemoryScorerDefinitionsStorage) createVersionLocked(_ context.Context, input CreateScorerDefinitionVersionInput) (*ScorerDefinitionVersion, error) {
	if _, exists := s.versions[input.ID]; exists {
		return nil, fmt.Errorf("Version with id %s already exists", input.ID)
	}
	for _, v := range s.versions {
		if v.ScorerDefinitionID == input.ScorerDefinitionID && v.VersionNumber == input.VersionNumber {
			return nil, fmt.Errorf("Version number %d already exists for scorer definition %s", input.VersionNumber, input.ScorerDefinitionID)
		}
	}

	version := ScorerDefinitionVersion{
		ID:                             input.ID,
		ScorerDefinitionID:             input.ScorerDefinitionID,
		VersionNumber:                  input.VersionNumber,
		ChangedFields:                  input.ChangedFields,
		ChangeMessage:                  input.ChangeMessage,
		CreatedAt:                      time.Now(),
		ScorerDefinitionSnapshotConfig: input.ScorerDefinitionSnapshotConfig,
	}

	s.versions[input.ID] = deepCopyVersion(version)
	copied := deepCopyVersion(version)
	return &copied, nil
}

func (s *InMemoryScorerDefinitionsStorage) GetVersion(_ context.Context, id string) (*ScorerDefinitionVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.versions[id]
	if !ok {
		return nil, nil
	}
	copied := deepCopyVersion(v)
	return &copied, nil
}

func (s *InMemoryScorerDefinitionsStorage) GetVersionByNumber(_ context.Context, scorerDefinitionID string, versionNumber int) (*ScorerDefinitionVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.versions {
		if v.ScorerDefinitionID == scorerDefinitionID && v.VersionNumber == versionNumber {
			copied := deepCopyVersion(v)
			return &copied, nil
		}
	}
	return nil, nil
}

func (s *InMemoryScorerDefinitionsStorage) GetLatestVersion(_ context.Context, scorerDefinitionID string) (*ScorerDefinitionVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest *ScorerDefinitionVersion
	for _, v := range s.versions {
		if v.ScorerDefinitionID == scorerDefinitionID {
			if latest == nil || v.VersionNumber > latest.VersionNumber {
				copied := v
				latest = &copied
			}
		}
	}
	if latest == nil {
		return nil, nil
	}
	copied := deepCopyVersion(*latest)
	return &copied, nil
}

func (s *InMemoryScorerDefinitionsStorage) ListVersions(_ context.Context, input ListScorerDefinitionVersionsInput) (*ListScorerDefinitionVersionsOutput, error) {
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

	var versions []ScorerDefinitionVersion
	for _, v := range s.versions {
		if v.ScorerDefinitionID == input.ScorerDefinitionID {
			versions = append(versions, v)
		}
	}

	sortVersions(versions, parsed.Field, parsed.Direction)

	cloned := make([]ScorerDefinitionVersion, len(versions))
	for i, v := range versions {
		cloned[i] = deepCopyVersion(v)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, input.PerPage, perPage)
	end := min(offset+perPage, total)
	start := min(offset, total)

	return &ListScorerDefinitionVersionsOutput{
		Versions: cloned[start:end],
		Total:    total,
		Page:     page,
		PerPage:  perPageResp,
		HasMore:  end < total,
	}, nil
}

func (s *InMemoryScorerDefinitionsStorage) DeleteVersion(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.versions, id)
	return nil
}

func (s *InMemoryScorerDefinitionsStorage) DeleteVersionsByParentID(_ context.Context, scorerDefinitionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteVersionsByParentIDLocked(scorerDefinitionID)
	return nil
}

func (s *InMemoryScorerDefinitionsStorage) deleteVersionsByParentIDLocked(scorerDefinitionID string) {
	var idsToDelete []string
	for id, v := range s.versions {
		if v.ScorerDefinitionID == scorerDefinitionID {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(s.versions, id)
	}
}

func (s *InMemoryScorerDefinitionsStorage) CountVersions(_ context.Context, scorerDefinitionID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, v := range s.versions {
		if v.ScorerDefinitionID == scorerDefinitionID {
			count++
		}
	}
	return count, nil
}

// ==========================================================================
// Resolution Methods
// ==========================================================================

func (s *InMemoryScorerDefinitionsStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
	entityRaw, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entityRaw == nil {
		return nil, nil
	}
	return s.resolveEntity(ctx, entityRaw, status)
}

func (s *InMemoryScorerDefinitionsStorage) ListResolved(ctx context.Context, args any) (any, error) {
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

func (s *InMemoryScorerDefinitionsStorage) resolveEntity(ctx context.Context, entityRaw any, status string) (any, error) {
	if status == "" {
		status = "published"
	}

	entityMap, ok := toMap(entityRaw)
	if !ok {
		return entityRaw, nil
	}

	entityID, _ := entityMap["id"].(string)

	var version *ScorerDefinitionVersion
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
			"id": true, "scorerDefinitionId": true, "versionNumber": true,
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

func deepCopyEntity(e storageScorerDefinitionType) storageScorerDefinitionType {
	copied := e
	if e.Metadata != nil {
		copied.Metadata = make(map[string]any, len(e.Metadata))
		for k, v := range e.Metadata {
			copied.Metadata[k] = v
		}
	}
	return copied
}

func deepCopyVersion(v ScorerDefinitionVersion) ScorerDefinitionVersion {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out ScorerDefinitionVersion
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

func sortEntities(entities []storageScorerDefinitionType, field domains.ThreadOrderBy, direction domains.SortDirection) {
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

func sortVersions(versions []ScorerDefinitionVersion, field domains.VersionOrderBy, direction domains.SortDirection) {
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

func min(a, b int) int {
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
