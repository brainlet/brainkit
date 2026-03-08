// Ported from: packages/core/src/storage/domains/mcp-servers/inmemory.ts
package mcpservers

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
var _ MCPServersStorage = (*InMemoryMCPServersStorage)(nil)

// ---------------------------------------------------------------------------
// Local entity type — placeholder until storage/types.go is ported.
// ---------------------------------------------------------------------------

// storageMCPServerType is the thin MCP server record stored in memory.
// TODO: Replace with StorageMCPServerType from storage/types.go once ported.
type storageMCPServerType struct {
	ID              string         `json:"id"`
	Status          string         `json:"status"`
	ActiveVersionID string         `json:"activeVersionId,omitempty"`
	AuthorID        string         `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// ---------------------------------------------------------------------------
// InMemoryMCPServersStorage
// ---------------------------------------------------------------------------

// InMemoryMCPServersStorage is an in-memory implementation of MCPServersStorage.
type InMemoryMCPServersStorage struct {
	mu       sync.RWMutex
	servers  map[string]storageMCPServerType
	versions map[string]MCPServerVersion
}

// NewInMemoryMCPServersStorage creates a new InMemoryMCPServersStorage.
func NewInMemoryMCPServersStorage() *InMemoryMCPServersStorage {
	return &InMemoryMCPServersStorage{
		servers:  make(map[string]storageMCPServerType),
		versions: make(map[string]MCPServerVersion),
	}
}

func (s *InMemoryMCPServersStorage) Init(_ context.Context) error { return nil }

func (s *InMemoryMCPServersStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.servers = make(map[string]storageMCPServerType)
	s.versions = make(map[string]MCPServerVersion)
	return nil
}

// ==========================================================================
// Entity CRUD
// ==========================================================================

func (s *InMemoryMCPServersStorage) GetByID(_ context.Context, id string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	srv, ok := s.servers[id]
	if !ok {
		return nil, nil
	}
	return deepCopyEntity(srv), nil
}

func (s *InMemoryMCPServersStorage) Create(ctx context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}
	if ms, ok := inputMap["mcpServer"].(map[string]any); ok {
		inputMap = ms
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("MCP server id is required")
	}
	if _, exists := s.servers[id]; exists {
		return nil, fmt.Errorf("MCP server with id %s already exists", id)
	}

	now := time.Now()
	srv := storageMCPServerType{
		ID:        id,
		Status:    "draft",
		AuthorID:  strVal(inputMap, "authorId"),
		Metadata:  mapVal(inputMap, "metadata"),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.servers[id] = srv

	snapshotConfig := excludeKeys(inputMap, "id", "authorId", "metadata")
	versionID := uuid.New().String()
	if _, err := s.createVersionLocked(ctx, CreateMCPServerVersionInput{
		ID:            versionID,
		MCPServerID:   id,
		VersionNumber: 1,
		ChangedFields: mapKeys(snapshotConfig),
		ChangeMessage: "Initial version",
		Snapshot:      snapshotConfig,
	}); err != nil {
		return nil, err
	}

	return deepCopyEntity(srv), nil
}

func (s *InMemoryMCPServersStorage) Update(_ context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}
	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("MCP server id is required")
	}
	existing, exists := s.servers[id]
	if !exists {
		return nil, fmt.Errorf("MCP server with id %s not found", id)
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

	s.servers[id] = existing
	return deepCopyEntity(existing), nil
}

func (s *InMemoryMCPServersStorage) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.servers, id)
	s.deleteVersionsByParentIDLocked(id)
	return nil
}

// List lists MCP servers with optional filtering, sorting, and pagination.
// NOTE: Unlike other domains, the default status filter is "published".
func (s *InMemoryMCPServersStorage) List(_ context.Context, args any) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	argsMap, _ := toMap(args)
	page := intVal(argsMap, "page", 0)
	perPageInput := optionalIntVal(argsMap, "perPage")
	orderByMap, _ := argsMap["orderBy"].(map[string]any)
	authorIDFilter := strVal(argsMap, "authorId")
	metadataFilter := mapVal(argsMap, "metadata")

	// Default status to "published" — this is the key difference from other domains.
	statusFilter := strVal(argsMap, "status")
	if statusFilter == "" {
		statusFilter = "published"
	}

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

	var servers []storageMCPServerType
	for _, srv := range s.servers {
		servers = append(servers, srv)
	}

	if statusFilter != "" {
		servers = filterSlice(servers, func(srv storageMCPServerType) bool { return srv.Status == statusFilter })
	}
	if authorIDFilter != "" {
		servers = filterSlice(servers, func(srv storageMCPServerType) bool { return srv.AuthorID == authorIDFilter })
	}
	if len(metadataFilter) > 0 {
		servers = filterSlice(servers, func(srv storageMCPServerType) bool {
			if srv.Metadata == nil {
				return false
			}
			for k, v := range metadataFilter {
				if !deepEqual(srv.Metadata[k], v) {
					return false
				}
			}
			return true
		})
	}

	sortEntities(servers, parsed.Field, parsed.Direction)

	cloned := make([]any, len(servers))
	for i, srv := range servers {
		cloned[i] = deepCopyEntity(srv)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, perPageInput, perPage)
	end := min(offset+perPage, total)
	start := min(offset, total)

	return map[string]any{
		"mcpServers": cloned[start:end],
		"total":      total,
		"page":       page,
		"perPage":    perPageResp,
		"hasMore":    end < total,
	}, nil
}

// ==========================================================================
// Version Methods
// ==========================================================================

func (s *InMemoryMCPServersStorage) CreateVersion(ctx context.Context, input CreateMCPServerVersionInput) (*MCPServerVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createVersionLocked(ctx, input)
}

func (s *InMemoryMCPServersStorage) createVersionLocked(_ context.Context, input CreateMCPServerVersionInput) (*MCPServerVersion, error) {
	if _, exists := s.versions[input.ID]; exists {
		return nil, fmt.Errorf("Version with id %s already exists", input.ID)
	}
	for _, v := range s.versions {
		if v.MCPServerID == input.MCPServerID && v.VersionNumber == input.VersionNumber {
			return nil, fmt.Errorf("Version number %d already exists for MCP server %s", input.VersionNumber, input.MCPServerID)
		}
	}

	version := MCPServerVersion{
		ID:            input.ID,
		MCPServerID:   input.MCPServerID,
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

func (s *InMemoryMCPServersStorage) GetVersion(_ context.Context, id string) (*MCPServerVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.versions[id]
	if !ok {
		return nil, nil
	}
	copied := deepCopyVersion(v)
	return &copied, nil
}

func (s *InMemoryMCPServersStorage) GetVersionByNumber(_ context.Context, mcpServerID string, versionNumber int) (*MCPServerVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.versions {
		if v.MCPServerID == mcpServerID && v.VersionNumber == versionNumber {
			copied := deepCopyVersion(v)
			return &copied, nil
		}
	}
	return nil, nil
}

func (s *InMemoryMCPServersStorage) GetLatestVersion(_ context.Context, mcpServerID string) (*MCPServerVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var latest *MCPServerVersion
	for _, v := range s.versions {
		if v.MCPServerID == mcpServerID {
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

func (s *InMemoryMCPServersStorage) ListVersions(_ context.Context, input ListMCPServerVersionsInput) (*ListMCPServerVersionsOutput, error) {
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

	var versions []MCPServerVersion
	for _, v := range s.versions {
		if v.MCPServerID == input.MCPServerID {
			versions = append(versions, v)
		}
	}

	sortVersions(versions, parsed.Field, parsed.Direction)

	cloned := make([]MCPServerVersion, len(versions))
	for i, v := range versions {
		cloned[i] = deepCopyVersion(v)
	}

	total := len(cloned)
	offset, perPageResp := calculatePagination(page, input.PerPage, perPage)
	end := min(offset+perPage, total)
	start := min(offset, total)

	return &ListMCPServerVersionsOutput{
		Versions: cloned[start:end],
		Total:    total,
		Page:     page,
		PerPage:  perPageResp,
		HasMore:  end < total,
	}, nil
}

func (s *InMemoryMCPServersStorage) DeleteVersion(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.versions, id)
	return nil
}

func (s *InMemoryMCPServersStorage) DeleteVersionsByParentID(_ context.Context, mcpServerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteVersionsByParentIDLocked(mcpServerID)
	return nil
}

func (s *InMemoryMCPServersStorage) deleteVersionsByParentIDLocked(mcpServerID string) {
	var idsToDelete []string
	for id, v := range s.versions {
		if v.MCPServerID == mcpServerID {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(s.versions, id)
	}
}

func (s *InMemoryMCPServersStorage) CountVersions(_ context.Context, mcpServerID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, v := range s.versions {
		if v.MCPServerID == mcpServerID {
			count++
		}
	}
	return count, nil
}

// ==========================================================================
// Resolution Methods
// ==========================================================================

func (s *InMemoryMCPServersStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
	entityRaw, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if entityRaw == nil {
		return nil, nil
	}
	return s.resolveEntity(ctx, entityRaw, status)
}

func (s *InMemoryMCPServersStorage) ListResolved(ctx context.Context, args any) (any, error) {
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

func (s *InMemoryMCPServersStorage) resolveEntity(ctx context.Context, entityRaw any, status string) (any, error) {
	if status == "" {
		status = "published"
	}

	entityMap, ok := toMap(entityRaw)
	if !ok {
		return entityRaw, nil
	}

	entityID, _ := entityMap["id"].(string)

	var version *MCPServerVersion
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
			"id": true, "mcpServerId": true, "versionNumber": true,
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

func deepCopyEntity(e storageMCPServerType) storageMCPServerType {
	copied := e
	if e.Metadata != nil {
		copied.Metadata = make(map[string]any, len(e.Metadata))
		for k, v := range e.Metadata {
			copied.Metadata[k] = v
		}
	}
	return copied
}

func deepCopyVersion(v MCPServerVersion) MCPServerVersion {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out MCPServerVersion
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

func sortEntities(entities []storageMCPServerType, field domains.ThreadOrderBy, direction domains.SortDirection) {
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

func sortVersions(versions []MCPServerVersion, field domains.VersionOrderBy, direction domains.SortDirection) {
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
