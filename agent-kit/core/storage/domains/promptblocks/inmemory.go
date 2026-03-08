// Ported from: packages/core/src/storage/domains/prompt-blocks/inmemory.ts
package promptblocks

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
var _ PromptBlocksStorage = (*InMemoryPromptBlocksStorage)(nil)

// ---------------------------------------------------------------------------
// Local entity type — placeholder until storage/types.go is ported.
// ---------------------------------------------------------------------------

// storagePromptBlockType is the thin prompt block record stored in memory.
// TODO: Replace with StoragePromptBlockType from storage/types.go once ported.
type storagePromptBlockType struct {
	ID              string         `json:"id"`
	Status          string         `json:"status"`
	ActiveVersionID string         `json:"activeVersionId,omitempty"`
	AuthorID        string         `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// ---------------------------------------------------------------------------
// InMemoryPromptBlocksStorage
// ---------------------------------------------------------------------------

// InMemoryPromptBlocksStorage is an in-memory implementation of PromptBlocksStorage.
type InMemoryPromptBlocksStorage struct {
	mu       sync.RWMutex
	blocks   map[string]storagePromptBlockType
	versions map[string]PromptBlockVersion
}

// NewInMemoryPromptBlocksStorage creates a new InMemoryPromptBlocksStorage.
func NewInMemoryPromptBlocksStorage() *InMemoryPromptBlocksStorage {
	return &InMemoryPromptBlocksStorage{
		blocks:   make(map[string]storagePromptBlockType),
		versions: make(map[string]PromptBlockVersion),
	}
}

// Init is a no-op for the in-memory store.
func (s *InMemoryPromptBlocksStorage) Init(_ context.Context) error {
	return nil
}

// DangerouslyClearAll clears all prompt blocks and versions.
func (s *InMemoryPromptBlocksStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.blocks = make(map[string]storagePromptBlockType)
	s.versions = make(map[string]PromptBlockVersion)
	return nil
}

// ==========================================================================
// Prompt Block CRUD Methods
// ==========================================================================

// GetByID retrieves a prompt block by ID.
func (s *InMemoryPromptBlocksStorage) GetByID(_ context.Context, id string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	block, ok := s.blocks[id]
	if !ok {
		return nil, nil
	}
	return deepCopyBlock(block), nil
}

// Create creates a new prompt block with an initial version.
func (s *InMemoryPromptBlocksStorage) Create(ctx context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}

	// Extract the nested promptBlock field if present.
	if pb, ok := inputMap["promptBlock"].(map[string]any); ok {
		inputMap = pb
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("prompt block id is required")
	}

	if _, exists := s.blocks[id]; exists {
		return nil, fmt.Errorf("Prompt block with id %s already exists", id)
	}

	now := time.Now()
	block := storagePromptBlockType{
		ID:        id,
		Status:    "draft",
		AuthorID:  strVal(inputMap, "authorId"),
		Metadata:  mapVal(inputMap, "metadata"),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.blocks[id] = block

	// Extract config fields.
	snapshotConfig := excludeKeys(inputMap, "id", "authorId", "metadata")

	// Create version 1.
	versionID := uuid.New().String()
	versionInput := CreatePromptBlockVersionInput{
		ID:            versionID,
		BlockID:       id,
		VersionNumber: 1,
		ChangedFields: mapKeys(snapshotConfig),
		ChangeMessage: "Initial version",
		Snapshot:      snapshotConfig,
	}

	if _, err := s.createVersionLocked(ctx, versionInput); err != nil {
		return nil, err
	}

	return deepCopyBlock(block), nil
}

// Update updates an existing prompt block.
func (s *InMemoryPromptBlocksStorage) Update(_ context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("prompt block id is required")
	}

	existing, exists := s.blocks[id]
	if !exists {
		return nil, fmt.Errorf("Prompt block with id %s not found", id)
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

	s.blocks[id] = existing
	return deepCopyBlock(existing), nil
}

// Delete removes a prompt block by ID and all its versions.
func (s *InMemoryPromptBlocksStorage) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.blocks, id)
	s.deleteVersionsByParentIDLocked(id)
	return nil
}

// List lists prompt blocks with optional filtering, sorting, and pagination.
func (s *InMemoryPromptBlocksStorage) List(_ context.Context, args any) (any, error) {
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

	var blocks []storagePromptBlockType
	for _, b := range s.blocks {
		blocks = append(blocks, b)
	}

	if statusFilter != "" {
		blocks = filterSlice(blocks, func(b storagePromptBlockType) bool {
			return b.Status == statusFilter
		})
	}

	if authorIDFilter != "" {
		blocks = filterSlice(blocks, func(b storagePromptBlockType) bool {
			return b.AuthorID == authorIDFilter
		})
	}

	if len(metadataFilter) > 0 {
		blocks = filterSlice(blocks, func(b storagePromptBlockType) bool {
			if b.Metadata == nil {
				return false
			}
			for k, v := range metadataFilter {
				if !deepEqual(b.Metadata[k], v) {
					return false
				}
			}
			return true
		})
	}

	sortBlocks(blocks, parsed.Field, parsed.Direction)

	cloned := make([]any, len(blocks))
	for i, b := range blocks {
		cloned[i] = deepCopyBlock(b)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, perPageInput, perPage)

	end := offset + perPage
	if end > total {
		end = total
	}
	start := offset
	if start > total {
		start = total
	}

	return map[string]any{
		"promptBlocks": cloned[start:end],
		"total":        total,
		"page":         page,
		"perPage":      perPageResp,
		"hasMore":      end < total,
	}, nil
}

// ==========================================================================
// Prompt Block Version Methods
// ==========================================================================

// CreateVersion creates a new prompt block version.
func (s *InMemoryPromptBlocksStorage) CreateVersion(ctx context.Context, input CreatePromptBlockVersionInput) (*PromptBlockVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.createVersionLocked(ctx, input)
}

func (s *InMemoryPromptBlocksStorage) createVersionLocked(_ context.Context, input CreatePromptBlockVersionInput) (*PromptBlockVersion, error) {
	if _, exists := s.versions[input.ID]; exists {
		return nil, fmt.Errorf("Version with id %s already exists", input.ID)
	}

	for _, v := range s.versions {
		if v.BlockID == input.BlockID && v.VersionNumber == input.VersionNumber {
			return nil, fmt.Errorf("Version number %d already exists for prompt block %s", input.VersionNumber, input.BlockID)
		}
	}

	version := PromptBlockVersion{
		ID:            input.ID,
		BlockID:       input.BlockID,
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

// GetVersion retrieves a version by its ID.
func (s *InMemoryPromptBlocksStorage) GetVersion(_ context.Context, id string) (*PromptBlockVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.versions[id]
	if !ok {
		return nil, nil
	}
	copied := deepCopyVersion(v)
	return &copied, nil
}

// GetVersionByNumber retrieves a version by block ID and version number.
func (s *InMemoryPromptBlocksStorage) GetVersionByNumber(_ context.Context, blockID string, versionNumber int) (*PromptBlockVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.versions {
		if v.BlockID == blockID && v.VersionNumber == versionNumber {
			copied := deepCopyVersion(v)
			return &copied, nil
		}
	}
	return nil, nil
}

// GetLatestVersion retrieves the latest version for a prompt block.
func (s *InMemoryPromptBlocksStorage) GetLatestVersion(_ context.Context, blockID string) (*PromptBlockVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest *PromptBlockVersion
	for _, v := range s.versions {
		if v.BlockID == blockID {
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

// ListVersions lists versions for a prompt block with pagination and sorting.
func (s *InMemoryPromptBlocksStorage) ListVersions(_ context.Context, input ListPromptBlockVersionsInput) (*ListPromptBlockVersionsOutput, error) {
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

	var versions []PromptBlockVersion
	for _, v := range s.versions {
		if v.BlockID == input.BlockID {
			versions = append(versions, v)
		}
	}

	sortVersions(versions, parsed.Field, parsed.Direction)

	cloned := make([]PromptBlockVersion, len(versions))
	for i, v := range versions {
		cloned[i] = deepCopyVersion(v)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, input.PerPage, perPage)

	end := offset + perPage
	if end > total {
		end = total
	}
	start := offset
	if start > total {
		start = total
	}

	return &ListPromptBlockVersionsOutput{
		Versions: cloned[start:end],
		Total:    total,
		Page:     page,
		PerPage:  perPageResp,
		HasMore:  end < total,
	}, nil
}

// DeleteVersion removes a version by ID.
func (s *InMemoryPromptBlocksStorage) DeleteVersion(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.versions, id)
	return nil
}

// DeleteVersionsByParentID removes all versions for a prompt block.
func (s *InMemoryPromptBlocksStorage) DeleteVersionsByParentID(_ context.Context, blockID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deleteVersionsByParentIDLocked(blockID)
	return nil
}

func (s *InMemoryPromptBlocksStorage) deleteVersionsByParentIDLocked(blockID string) {
	var idsToDelete []string
	for id, v := range s.versions {
		if v.BlockID == blockID {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(s.versions, id)
	}
}

// CountVersions returns the number of versions for a prompt block.
func (s *InMemoryPromptBlocksStorage) CountVersions(_ context.Context, blockID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, v := range s.versions {
		if v.BlockID == blockID {
			count++
		}
	}
	return count, nil
}

// ==========================================================================
// Resolution Methods
// ==========================================================================

func (s *InMemoryPromptBlocksStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
	entityRaw, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entityRaw == nil {
		return nil, nil
	}
	return s.resolveEntity(ctx, entityRaw, status)
}

func (s *InMemoryPromptBlocksStorage) ListResolved(ctx context.Context, args any) (any, error) {
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

	entities, ok := resultMap["promptBlocks"].([]any)
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

	resultMap["promptBlocks"] = resolved
	return resultMap, nil
}

func (s *InMemoryPromptBlocksStorage) resolveEntity(ctx context.Context, entityRaw any, status string) (any, error) {
	if status == "" {
		status = "published"
	}

	entityMap, ok := toMap(entityRaw)
	if !ok {
		return entityRaw, nil
	}

	entityID, _ := entityMap["id"].(string)

	var version *PromptBlockVersion
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
			"id": true, "blockId": true, "versionNumber": true,
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
// Private Helper Methods
// ==========================================================================

func deepCopyBlock(b storagePromptBlockType) storagePromptBlockType {
	copied := b
	if b.Metadata != nil {
		copied.Metadata = make(map[string]any, len(b.Metadata))
		for k, v := range b.Metadata {
			copied.Metadata[k] = v
		}
	}
	return copied
}

func deepCopyVersion(v PromptBlockVersion) PromptBlockVersion {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out PromptBlockVersion
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

func sortBlocks(blocks []storagePromptBlockType, field domains.ThreadOrderBy, direction domains.SortDirection) {
	sort.Slice(blocks, func(i, j int) bool {
		var aVal, bVal time.Time
		switch field {
		case domains.ThreadOrderByUpdatedAt:
			aVal, bVal = blocks[i].UpdatedAt, blocks[j].UpdatedAt
		default:
			aVal, bVal = blocks[i].CreatedAt, blocks[j].CreatedAt
		}
		if direction == domains.SortASC {
			return aVal.Before(bVal)
		}
		return bVal.Before(aVal)
	})
}

func sortVersions(versions []PromptBlockVersion, field domains.VersionOrderBy, direction domains.SortDirection) {
	sort.Slice(versions, func(i, j int) bool {
		var aVal, bVal float64
		switch field {
		case domains.VersionOrderByCreatedAt:
			aVal = float64(versions[i].CreatedAt.UnixNano())
			bVal = float64(versions[j].CreatedAt.UnixNano())
		default:
			aVal = float64(versions[i].VersionNumber)
			bVal = float64(versions[j].VersionNumber)
		}
		if direction == domains.SortASC {
			return aVal < bVal
		}
		return bVal < aVal
	})
}

// ---------------------------------------------------------------------------
// Generic helpers (duplicated per package to avoid circular deps)
// ---------------------------------------------------------------------------

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
	default:
		return defaultVal
	}
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
	default:
		return nil
	}
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

func calculatePagination(page int, perPageInput *int, normalizedPerPage int) (offset int, perPageResp int) {
	offset = page * normalizedPerPage
	perPageResp = normalizedPerPage
	if perPageInput != nil && *perPageInput == domains.PerPageDisabled {
		perPageResp = domains.PerPageDisabled
	}
	return
}

func deepEqual(a, b any) bool {
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return a == b
	}
	return string(aJSON) == string(bJSON)
}
