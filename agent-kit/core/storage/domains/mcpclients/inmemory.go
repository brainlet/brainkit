// Ported from: packages/core/src/storage/domains/mcp-clients/inmemory.ts
package mcpclients

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
var _ MCPClientsStorage = (*InMemoryMCPClientsStorage)(nil)

// ---------------------------------------------------------------------------
// Local entity type — placeholder until storage/types.go is ported.
// ---------------------------------------------------------------------------

// storageMCPClientType is the thin MCP client record stored in memory.
// TODO: Replace with StorageMCPClientType from storage/types.go once ported.
type storageMCPClientType struct {
	ID              string         `json:"id"`
	Status          string         `json:"status"`
	ActiveVersionID string         `json:"activeVersionId,omitempty"`
	AuthorID        string         `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// ---------------------------------------------------------------------------
// InMemoryMCPClientsStorage
// ---------------------------------------------------------------------------

// InMemoryMCPClientsStorage is an in-memory implementation of MCPClientsStorage.
type InMemoryMCPClientsStorage struct {
	mu       sync.RWMutex
	clients  map[string]storageMCPClientType
	versions map[string]MCPClientVersion
}

// NewInMemoryMCPClientsStorage creates a new InMemoryMCPClientsStorage.
func NewInMemoryMCPClientsStorage() *InMemoryMCPClientsStorage {
	return &InMemoryMCPClientsStorage{
		clients:  make(map[string]storageMCPClientType),
		versions: make(map[string]MCPClientVersion),
	}
}

func (s *InMemoryMCPClientsStorage) Init(_ context.Context) error { return nil }

func (s *InMemoryMCPClientsStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients = make(map[string]storageMCPClientType)
	s.versions = make(map[string]MCPClientVersion)
	return nil
}

// ==========================================================================
// Entity CRUD
// ==========================================================================

func (s *InMemoryMCPClientsStorage) GetByID(_ context.Context, id string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clients[id]
	if !ok {
		return nil, nil
	}
	return deepCopyEntity(c), nil
}

func (s *InMemoryMCPClientsStorage) Create(ctx context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}
	if mc, ok := inputMap["mcpClient"].(map[string]any); ok {
		inputMap = mc
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("MCP client id is required")
	}
	if _, exists := s.clients[id]; exists {
		return nil, fmt.Errorf("MCP client with id %s already exists", id)
	}

	now := time.Now()
	client := storageMCPClientType{
		ID:        id,
		Status:    "draft",
		AuthorID:  strVal(inputMap, "authorId"),
		Metadata:  mapVal(inputMap, "metadata"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.clients[id] = client

	snapshotConfig := excludeKeys(inputMap, "id", "authorId", "metadata")
	versionID := uuid.New().String()
	if _, err := s.createVersionLocked(ctx, CreateMCPClientVersionInput{
		ID:            versionID,
		MCPClientID:   id,
		VersionNumber: 1,
		ChangedFields: mapKeys(snapshotConfig),
		ChangeMessage: "Initial version",
		Snapshot:      snapshotConfig,
	}); err != nil {
		return nil, err
	}

	return deepCopyEntity(client), nil
}

func (s *InMemoryMCPClientsStorage) Update(_ context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}
	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("MCP client id is required")
	}
	existing, exists := s.clients[id]
	if !exists {
		return nil, fmt.Errorf("MCP client with id %s not found", id)
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

	s.clients[id] = existing
	return deepCopyEntity(existing), nil
}

func (s *InMemoryMCPClientsStorage) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, id)
	s.deleteVersionsByParentIDLocked(id)
	return nil
}

func (s *InMemoryMCPClientsStorage) List(_ context.Context, args any) (any, error) {
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

	var clients []storageMCPClientType
	for _, c := range s.clients {
		clients = append(clients, c)
	}

	if statusFilter != "" {
		clients = filterSlice(clients, func(c storageMCPClientType) bool { return c.Status == statusFilter })
	}
	if authorIDFilter != "" {
		clients = filterSlice(clients, func(c storageMCPClientType) bool { return c.AuthorID == authorIDFilter })
	}
	if len(metadataFilter) > 0 {
		clients = filterSlice(clients, func(c storageMCPClientType) bool {
			if c.Metadata == nil {
				return false
			}
			for k, v := range metadataFilter {
				if !deepEqual(c.Metadata[k], v) {
					return false
				}
			}
			return true
		})
	}

	sortEntities(clients, parsed.Field, parsed.Direction)

	cloned := make([]any, len(clients))
	for i, c := range clients {
		cloned[i] = deepCopyEntity(c)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, perPageInput, perPage)
	end := min(offset+perPage, total)
	start := min(offset, total)

	return map[string]any{
		"mcpClients": cloned[start:end],
		"total":      total,
		"page":       page,
		"perPage":    perPageResp,
		"hasMore":    end < total,
	}, nil
}

// ==========================================================================
// Version Methods
// ==========================================================================

func (s *InMemoryMCPClientsStorage) CreateVersion(ctx context.Context, input CreateMCPClientVersionInput) (*MCPClientVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createVersionLocked(ctx, input)
}

func (s *InMemoryMCPClientsStorage) createVersionLocked(_ context.Context, input CreateMCPClientVersionInput) (*MCPClientVersion, error) {
	if _, exists := s.versions[input.ID]; exists {
		return nil, fmt.Errorf("Version with id %s already exists", input.ID)
	}
	for _, v := range s.versions {
		if v.MCPClientID == input.MCPClientID && v.VersionNumber == input.VersionNumber {
			return nil, fmt.Errorf("Version number %d already exists for MCP client %s", input.VersionNumber, input.MCPClientID)
		}
	}

	version := MCPClientVersion{
		ID:            input.ID,
		MCPClientID:   input.MCPClientID,
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

func (s *InMemoryMCPClientsStorage) GetVersion(_ context.Context, id string) (*MCPClientVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.versions[id]
	if !ok {
		return nil, nil
	}
	copied := deepCopyVersion(v)
	return &copied, nil
}

func (s *InMemoryMCPClientsStorage) GetVersionByNumber(_ context.Context, mcpClientID string, versionNumber int) (*MCPClientVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.versions {
		if v.MCPClientID == mcpClientID && v.VersionNumber == versionNumber {
			copied := deepCopyVersion(v)
			return &copied, nil
		}
	}
	return nil, nil
}

func (s *InMemoryMCPClientsStorage) GetLatestVersion(_ context.Context, mcpClientID string) (*MCPClientVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest *MCPClientVersion
	for _, v := range s.versions {
		if v.MCPClientID == mcpClientID {
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

func (s *InMemoryMCPClientsStorage) ListVersions(_ context.Context, input ListMCPClientVersionsInput) (*ListMCPClientVersionsOutput, error) {
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

	var versions []MCPClientVersion
	for _, v := range s.versions {
		if v.MCPClientID == input.MCPClientID {
			versions = append(versions, v)
		}
	}

	sortVersions(versions, parsed.Field, parsed.Direction)

	cloned := make([]MCPClientVersion, len(versions))
	for i, v := range versions {
		cloned[i] = deepCopyVersion(v)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, input.PerPage, perPage)
	end := min(offset+perPage, total)
	start := min(offset, total)

	return &ListMCPClientVersionsOutput{
		Versions: cloned[start:end],
		Total:    total,
		Page:     page,
		PerPage:  perPageResp,
		HasMore:  end < total,
	}, nil
}

func (s *InMemoryMCPClientsStorage) DeleteVersion(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.versions, id)
	return nil
}

func (s *InMemoryMCPClientsStorage) DeleteVersionsByParentID(_ context.Context, mcpClientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteVersionsByParentIDLocked(mcpClientID)
	return nil
}

func (s *InMemoryMCPClientsStorage) deleteVersionsByParentIDLocked(mcpClientID string) {
	var idsToDelete []string
	for id, v := range s.versions {
		if v.MCPClientID == mcpClientID {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(s.versions, id)
	}
}

func (s *InMemoryMCPClientsStorage) CountVersions(_ context.Context, mcpClientID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, v := range s.versions {
		if v.MCPClientID == mcpClientID {
			count++
		}
	}
	return count, nil
}

// ==========================================================================
// Resolution Methods (stubs)
// ==========================================================================

func (s *InMemoryMCPClientsStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
	entityRaw, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entityRaw == nil {
		return nil, nil
	}
	return s.resolveEntity(ctx, entityRaw, status)
}

func (s *InMemoryMCPClientsStorage) ListResolved(ctx context.Context, args any) (any, error) {
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

func (s *InMemoryMCPClientsStorage) resolveEntity(ctx context.Context, entityRaw any, status string) (any, error) {
	if status == "" {
		status = "published"
	}

	entityMap, ok := toMap(entityRaw)
	if !ok {
		return entityRaw, nil
	}

	entityID, _ := entityMap["id"].(string)

	var version *MCPClientVersion
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
			"id": true, "mcpClientId": true, "versionNumber": true,
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

func deepCopyEntity(e storageMCPClientType) storageMCPClientType {
	copied := e
	if e.Metadata != nil {
		copied.Metadata = make(map[string]any, len(e.Metadata))
		for k, v := range e.Metadata {
			copied.Metadata[k] = v
		}
	}
	return copied
}

func deepCopyVersion(v MCPClientVersion) MCPClientVersion {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out MCPClientVersion
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

func sortEntities(entities []storageMCPClientType, field domains.ThreadOrderBy, direction domains.SortDirection) {
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

func sortVersions(versions []MCPClientVersion, field domains.VersionOrderBy, direction domains.SortDirection) {
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
