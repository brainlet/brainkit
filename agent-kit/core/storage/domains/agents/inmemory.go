// Ported from: packages/core/src/storage/domains/agents/inmemory.ts
package agents

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
var _ AgentsStorage = (*InMemoryAgentsStorage)(nil)

// ---------------------------------------------------------------------------
// Local entity type — placeholder until storage/types.go is ported.
// ---------------------------------------------------------------------------

// storageAgentType is the thin agent record stored in memory.
// TODO: Replace with StorageAgentType from storage/types.go once ported.
type storageAgentType struct {
	ID              string            `json:"id"`
	Status          string            `json:"status"`
	ActiveVersionID string            `json:"activeVersionId,omitempty"`
	AuthorID        string            `json:"authorId,omitempty"`
	Metadata        map[string]any    `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

// ---------------------------------------------------------------------------
// InMemoryAgentsStorage
// ---------------------------------------------------------------------------

// InMemoryAgentsStorage is an in-memory implementation of AgentsStorage.
type InMemoryAgentsStorage struct {
	mu       sync.RWMutex
	agents   map[string]storageAgentType
	versions map[string]AgentVersion
}

// NewInMemoryAgentsStorage creates a new InMemoryAgentsStorage.
func NewInMemoryAgentsStorage() *InMemoryAgentsStorage {
	return &InMemoryAgentsStorage{
		agents:   make(map[string]storageAgentType),
		versions: make(map[string]AgentVersion),
	}
}

// Init is a no-op for the in-memory store.
func (s *InMemoryAgentsStorage) Init(_ context.Context) error {
	return nil
}

// DangerouslyClearAll clears all agents and versions.
func (s *InMemoryAgentsStorage) DangerouslyClearAll(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agents = make(map[string]storageAgentType)
	s.versions = make(map[string]AgentVersion)
	return nil
}

// ==========================================================================
// Agent CRUD Methods
// ==========================================================================

// GetByID retrieves an agent by ID.
func (s *InMemoryAgentsStorage) GetByID(_ context.Context, id string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, ok := s.agents[id]
	if !ok {
		return nil, nil
	}
	return deepCopyAgent(agent), nil
}

// Create creates a new agent with an initial version.
func (s *InMemoryAgentsStorage) Create(ctx context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Create")
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("agent id is required")
	}

	if _, exists := s.agents[id]; exists {
		return nil, fmt.Errorf("Agent with id %s already exists", id)
	}

	now := time.Now()
	agent := storageAgentType{
		ID:        id,
		Status:    "draft",
		AuthorID:  strVal(inputMap, "authorId"),
		Metadata:  mapVal(inputMap, "metadata"),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.agents[id] = agent

	// Extract config fields (everything except agent-record fields).
	snapshotConfig := excludeKeys(inputMap, "id", "authorId", "metadata")

	// Create version 1 from the config.
	versionID := uuid.New().String()
	versionInput := CreateAgentVersionInput{
		ID:            versionID,
		AgentID:       id,
		VersionNumber: 1,
		ChangedFields: mapKeys(snapshotConfig),
		ChangeMessage: "Initial version",
		Snapshot:      snapshotConfig,
	}

	if _, err := s.createVersionLocked(ctx, versionInput); err != nil {
		return nil, err
	}

	return deepCopyAgent(agent), nil
}

// Update updates an existing agent.
func (s *InMemoryAgentsStorage) Update(_ context.Context, input any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	inputMap, ok := toMap(input)
	if !ok {
		return nil, fmt.Errorf("invalid input type for Update")
	}

	id, _ := inputMap["id"].(string)
	if id == "" {
		return nil, fmt.Errorf("agent id is required")
	}

	existing, exists := s.agents[id]
	if !exists {
		return nil, fmt.Errorf("Agent with id %s not found", id)
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

	s.agents[id] = existing
	return deepCopyAgent(existing), nil
}

// Delete removes an agent by ID and all its versions.
func (s *InMemoryAgentsStorage) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Idempotent delete.
	delete(s.agents, id)
	s.deleteVersionsByParentIDLocked(id)
	return nil
}

// List lists agents with optional filtering, sorting, and pagination.
func (s *InMemoryAgentsStorage) List(_ context.Context, args any) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	argsMap, _ := toMap(args)

	page := intVal(argsMap, "page", 0)
	perPageInput := optionalIntVal(argsMap, "perPage")
	orderByMap, _ := argsMap["orderBy"].(map[string]any)
	statusFilter := strVal(argsMap, "status")
	authorIDFilter := strVal(argsMap, "authorId")
	metadataFilter := mapVal(argsMap, "metadata")

	// Parse ordering.
	var orderBy *domains.StorageOrderBy
	if orderByMap != nil {
		orderBy = &domains.StorageOrderBy{
			Field:     domains.ThreadOrderBy(strVal(orderByMap, "field")),
			Direction: domains.SortDirection(strVal(orderByMap, "direction")),
		}
	}
	base := domains.VersionedStorageDomainBase{}
	parsed := base.ParseOrderBy(orderBy, domains.SortDESC)

	// Normalize perPage.
	perPage := normalizePerPage(perPageInput, 100)

	if page < 0 {
		return nil, fmt.Errorf("page must be >= 0")
	}

	// Collect and filter agents.
	var agents []storageAgentType
	for _, a := range s.agents {
		agents = append(agents, a)
	}

	if statusFilter != "" {
		agents = filterSlice(agents, func(a storageAgentType) bool {
			return a.Status == statusFilter
		})
	}

	if authorIDFilter != "" {
		agents = filterSlice(agents, func(a storageAgentType) bool {
			return a.AuthorID == authorIDFilter
		})
	}

	if len(metadataFilter) > 0 {
		agents = filterSlice(agents, func(a storageAgentType) bool {
			if a.Metadata == nil {
				return false
			}
			for k, v := range metadataFilter {
				if !deepEqual(a.Metadata[k], v) {
					return false
				}
			}
			return true
		})
	}

	// Sort.
	sortEntities(agents, parsed.Field, parsed.Direction)

	// Deep clone.
	cloned := make([]any, len(agents))
	for i, a := range agents {
		cloned[i] = deepCopyAgent(a)
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
		"agents":  cloned[start:end],
		"total":   total,
		"page":    page,
		"perPage": perPageResp,
		"hasMore": end < total,
	}, nil
}

// ==========================================================================
// Agent Version Methods
// ==========================================================================

// CreateVersion creates a new agent version.
func (s *InMemoryAgentsStorage) CreateVersion(ctx context.Context, input CreateAgentVersionInput) (*AgentVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.createVersionLocked(ctx, input)
}

// createVersionLocked creates a version while the lock is already held.
func (s *InMemoryAgentsStorage) createVersionLocked(_ context.Context, input CreateAgentVersionInput) (*AgentVersion, error) {
	// Check for duplicate version ID.
	if _, exists := s.versions[input.ID]; exists {
		return nil, fmt.Errorf("Version with id %s already exists", input.ID)
	}

	// Check for duplicate (agentId, versionNumber) pair.
	for _, v := range s.versions {
		if v.AgentID == input.AgentID && v.VersionNumber == input.VersionNumber {
			return nil, fmt.Errorf("Version number %d already exists for agent %s", input.VersionNumber, input.AgentID)
		}
	}

	version := AgentVersion{
		ID:            input.ID,
		AgentID:       input.AgentID,
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
func (s *InMemoryAgentsStorage) GetVersion(_ context.Context, id string) (*AgentVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, ok := s.versions[id]
	if !ok {
		return nil, nil
	}
	copied := deepCopyVersion(v)
	return &copied, nil
}

// GetVersionByNumber retrieves a version by agent ID and version number.
func (s *InMemoryAgentsStorage) GetVersionByNumber(_ context.Context, agentID string, versionNumber int) (*AgentVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, v := range s.versions {
		if v.AgentID == agentID && v.VersionNumber == versionNumber {
			copied := deepCopyVersion(v)
			return &copied, nil
		}
	}
	return nil, nil
}

// GetLatestVersion retrieves the latest version for an agent.
func (s *InMemoryAgentsStorage) GetLatestVersion(_ context.Context, agentID string) (*AgentVersion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest *AgentVersion
	for _, v := range s.versions {
		if v.AgentID == agentID {
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

// ListVersions lists versions for an agent with pagination and sorting.
func (s *InMemoryAgentsStorage) ListVersions(_ context.Context, input ListVersionsInput) (*ListVersionsOutput, error) {
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

	// Parse version ordering.
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

	// Filter versions by agentId.
	var versions []AgentVersion
	for _, v := range s.versions {
		if v.AgentID == input.AgentID {
			versions = append(versions, v)
		}
	}

	// Sort.
	sortVersions(versions, parsed.Field, parsed.Direction)

	// Deep clone.
	cloned := make([]AgentVersion, len(versions))
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

	return &ListVersionsOutput{
		Versions: cloned[start:end],
		Total:    total,
		Page:     page,
		PerPage:  perPageResp,
		HasMore:  end < total,
	}, nil
}

// DeleteVersion removes a version by ID.
func (s *InMemoryAgentsStorage) DeleteVersion(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Idempotent delete.
	delete(s.versions, id)
	return nil
}

// DeleteVersionsByParentID removes all versions for an agent.
func (s *InMemoryAgentsStorage) DeleteVersionsByParentID(_ context.Context, agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deleteVersionsByParentIDLocked(agentID)
	return nil
}

func (s *InMemoryAgentsStorage) deleteVersionsByParentIDLocked(agentID string) {
	var idsToDelete []string
	for id, v := range s.versions {
		if v.AgentID == agentID {
			idsToDelete = append(idsToDelete, id)
		}
	}
	for _, id := range idsToDelete {
		delete(s.versions, id)
	}
}

// CountVersions returns the number of versions for an agent.
func (s *InMemoryAgentsStorage) CountVersions(_ context.Context, agentID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, v := range s.versions {
		if v.AgentID == agentID {
			count++
		}
	}
	return count, nil
}

// ==========================================================================
// Resolution Methods (stubs — TODO: implement once storage/types.go is ported)
// ==========================================================================

// GetByIDResolved resolves an entity by merging its thin record with
// the active or latest version config.
func (s *InMemoryAgentsStorage) GetByIDResolved(ctx context.Context, id string, status string) (any, error) {
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
func (s *InMemoryAgentsStorage) ListResolved(ctx context.Context, args any) (any, error) {
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
func (s *InMemoryAgentsStorage) resolveEntity(ctx context.Context, entityRaw any, status string) (any, error) {
	if status == "" {
		status = "published"
	}

	entityMap, ok := toMap(entityRaw)
	if !ok {
		return entityRaw, nil
	}

	entityID, _ := entityMap["id"].(string)

	var version *AgentVersion
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
			"id": true, "agentId": true, "versionNumber": true,
			"changedFields": true, "changeMessage": true, "createdAt": true,
		}

		// Extract snapshot config from version.
		versionMap, _ := toMap(*version)
		snapshotConfig := make(map[string]any)
		for k, v := range versionMap {
			if !versionMetadataFields[k] {
				snapshotConfig[k] = v
			}
		}

		// Merge entity + snapshot config + resolvedVersionId.
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

func deepCopyAgent(a storageAgentType) storageAgentType {
	copied := a
	if a.Metadata != nil {
		copied.Metadata = make(map[string]any, len(a.Metadata))
		for k, v := range a.Metadata {
			copied.Metadata[k] = v
		}
	}
	return copied
}

func deepCopyVersion(v AgentVersion) AgentVersion {
	data, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out AgentVersion
	if err := json.Unmarshal(data, &out); err != nil {
		return v
	}
	return out
}

func sortEntities(entities []storageAgentType, field domains.ThreadOrderBy, direction domains.SortDirection) {
	sort.Slice(entities, func(i, j int) bool {
		var aVal, bVal time.Time
		switch field {
		case domains.ThreadOrderByUpdatedAt:
			aVal, bVal = entities[i].UpdatedAt, entities[j].UpdatedAt
		default: // createdAt
			aVal, bVal = entities[i].CreatedAt, entities[j].CreatedAt
		}
		if direction == domains.SortASC {
			return aVal.Before(bVal)
		}
		return bVal.Before(aVal)
	})
}

func sortVersions(versions []AgentVersion, field domains.VersionOrderBy, direction domains.SortDirection) {
	sort.Slice(versions, func(i, j int) bool {
		var aVal, bVal float64
		switch field {
		case domains.VersionOrderByCreatedAt:
			aVal = float64(versions[i].CreatedAt.UnixNano())
			bVal = float64(versions[j].CreatedAt.UnixNano())
		default: // versionNumber
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
// Generic helpers
// ---------------------------------------------------------------------------

func toMap(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	if m, ok := v.(map[string]any); ok {
		return m, true
	}
	// Try JSON round-trip.
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

// normalizePerPage normalizes the PerPage value:
//   - nil → defaultVal
//   - PerPageDisabled (-1) → math.MaxInt
//   - 0 → 0
//   - positive → value
func normalizePerPage(perPage *int, defaultVal int) int {
	if perPage == nil {
		return defaultVal
	}
	if *perPage == domains.PerPageDisabled {
		return math.MaxInt
	}
	return *perPage
}

// calculatePagination returns the offset and the perPage value for the response.
func calculatePagination(page int, perPageInput *int, normalizedPerPage int) (offset int, perPageResp int) {
	offset = page * normalizedPerPage
	perPageResp = normalizedPerPage
	if perPageInput != nil && *perPageInput == domains.PerPageDisabled {
		perPageResp = domains.PerPageDisabled
	}
	return
}

// deepEqual compares two values via JSON serialization.
func deepEqual(a, b any) bool {
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return a == b
	}
	return string(aJSON) == string(bJSON)
}
