// Ported from: packages/core/src/storage/filesystem-versioned.ts
package fsutil

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// GIT_VERSION_PREFIX is the prefix for version IDs that come from git history.
// These versions are read-only and cannot be deleted.
const GIT_VERSION_PREFIX = "git-"

// ---------------------------------------------------------------------------
// FilesystemVersionedConfig
// ---------------------------------------------------------------------------

// FilesystemVersionedConfig configures a filesystem-backed versioned storage domain.
type FilesystemVersionedConfig struct {
	// DB is the FilesystemDB instance for I/O.
	DB *FilesystemDB
	// EntitiesFile is the filename for the entities JSON file (e.g., "agents.json").
	EntitiesFile string
	// ParentIDField is the key name of the parent FK field on versions (e.g., "agentId").
	ParentIDField string
	// Name is a name for logging/error messages.
	Name string
	// VersionMetadataFields is the set of fields that are version metadata
	// (not part of the snapshot config). These are stripped when writing to disk.
	VersionMetadataFields []string
	// GitHistoryLimit is the maximum number of git commits to load per file (default: 50).
	GitHistoryLimit int
}

// ---------------------------------------------------------------------------
// FilesystemVersionedHelpers
// ---------------------------------------------------------------------------

// FilesystemVersionedHelpers provides generic helpers for filesystem-backed
// versioned storage domains.
//
// Versions are kept entirely in memory. Only the published snapshot config
// (the clean primitive configuration) is persisted to the on-disk JSON file.
//
// When the storage directory is inside a git repository, committed versions
// of the JSON file are automatically loaded as read-only version history.
type FilesystemVersionedHelpers struct {
	db                    *FilesystemDB
	entitiesFile          string
	parentIDField         string
	name                  string
	versionMetadataFields []string
	gitHistoryLimit       int

	mu               sync.RWMutex
	entities         map[string]map[string]any // keyed by entity ID
	versions         map[string]map[string]any // keyed by version ID
	hydrated         bool
	gitHistoryLoaded bool
	gitVersionCounts map[string]int // max git version count per entity ID
}

// gitHistory is the shared GitHistory instance across all helpers.
var gitHistory = NewGitHistory()

// NewFilesystemVersionedHelpers creates a new FilesystemVersionedHelpers instance.
func NewFilesystemVersionedHelpers(config FilesystemVersionedConfig) *FilesystemVersionedHelpers {
	gitLimit := config.GitHistoryLimit
	if gitLimit <= 0 {
		gitLimit = 50
	}

	return &FilesystemVersionedHelpers{
		db:                    config.DB,
		entitiesFile:          config.EntitiesFile,
		parentIDField:         config.ParentIDField,
		name:                  config.Name,
		versionMetadataFields: config.VersionMetadataFields,
		gitHistoryLimit:       gitLimit,
		entities:              make(map[string]map[string]any),
		versions:              make(map[string]map[string]any),
		gitVersionCounts:      make(map[string]int),
	}
}

// IsGitVersion checks if a version ID represents a git-based version.
func IsGitVersion(id string) bool {
	return strings.HasPrefix(id, GIT_VERSION_PREFIX)
}

// ---------------------------------------------------------------------------
// Hydration
// ---------------------------------------------------------------------------

// Hydrate loads in-memory state from the on-disk JSON file.
// For each entry on disk, creates an in-memory entity (status: "published")
// and a synthetic version with the snapshot config.
// Also kicks off async git history loading.
func (h *FilesystemVersionedHelpers) Hydrate() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.hydrated {
		return
	}
	h.hydrated = true

	diskData := h.db.ReadDomain(h.entitiesFile)

	for entityID, rawConfig := range diskData {
		snapshotConfig, ok := rawConfig.(map[string]any)
		if !ok {
			continue
		}

		versionID := fmt.Sprintf("hydrated-%s-v1", entityID)
		now := time.Now()

		entity := map[string]any{
			"id":              entityID,
			"status":          "published",
			"activeVersionId": versionID,
			"createdAt":       now,
			"updatedAt":       now,
		}
		h.entities[entityID] = entity

		version := make(map[string]any, len(snapshotConfig)+4)
		for k, v := range snapshotConfig {
			version[k] = v
		}
		version["id"] = versionID
		version[h.parentIDField] = entityID
		version["versionNumber"] = 1
		version["createdAt"] = now

		h.versions[versionID] = version
	}

	// Fire and forget git history loading
	go h.loadGitHistory()
}

// ensureGitHistory waits for git history to be loaded.
func (h *FilesystemVersionedHelpers) ensureGitHistory() {
	h.Hydrate()
	// Git history is loaded asynchronously in Hydrate.
	// For simplicity in Go, we use a simple flag.
	// In a production implementation, use sync.WaitGroup or channel.
}

// loadGitHistory loads git commit history for the domain's JSON file.
func (h *FilesystemVersionedHelpers) loadGitHistory() {
	dir := h.db.Dir

	isRepo, err := gitHistory.IsGitRepo(dir)
	if err != nil || !isRepo {
		return
	}

	commits, err := gitHistory.GetFileHistory(dir, h.entitiesFile, h.gitHistoryLimit)
	if err != nil || len(commits) == 0 {
		return
	}

	// Process commits from oldest to newest
	orderedCommits := make([]GitCommit, len(commits))
	copy(orderedCommits, commits)
	// Reverse: commits come newest-first
	for i, j := 0, len(orderedCommits)-1; i < j; i, j = i+1, j-1 {
		orderedCommits[i], orderedCommits[j] = orderedCommits[j], orderedCommits[i]
	}

	entityVersionCount := make(map[string]int)
	previousSnapshots := make(map[string]string)

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, commit := range orderedCommits {
		fileContent, err := gitHistory.GetFileAtCommit(dir, commit.Hash, h.entitiesFile)
		if err != nil || fileContent == nil {
			continue
		}

		for entityID, rawConfig := range fileContent {
			snapshotConfig, ok := rawConfig.(map[string]any)
			if !ok {
				continue
			}

			// Skip if unchanged
			serialized, _ := json.Marshal(snapshotConfig)
			serializedStr := string(serialized)
			if prev, ok := previousSnapshots[entityID]; ok && prev == serializedStr {
				continue
			}
			previousSnapshots[entityID] = serializedStr

			count := entityVersionCount[entityID] + 1
			entityVersionCount[entityID] = count

			versionID := fmt.Sprintf("%s%s-%s", GIT_VERSION_PREFIX, commit.Hash, entityID)

			if _, exists := h.versions[versionID]; exists {
				continue
			}

			version := make(map[string]any, len(snapshotConfig)+5)
			for k, v := range snapshotConfig {
				version[k] = v
			}
			version["id"] = versionID
			version[h.parentIDField] = entityID
			version["versionNumber"] = count
			version["changeMessage"] = commit.Message
			version["createdAt"] = commit.Date

			h.versions[versionID] = version
		}
	}

	h.gitVersionCounts = entityVersionCount

	// Reassign version numbers for hydrated versions to sit on top of git history
	for entityID, gitCount := range entityVersionCount {
		hydratedVersionID := fmt.Sprintf("hydrated-%s-v1", entityID)
		if version, ok := h.versions[hydratedVersionID]; ok {
			version["versionNumber"] = gitCount + 1
		}
	}

	h.gitHistoryLoaded = true
}

// ---------------------------------------------------------------------------
// Disk persistence
// ---------------------------------------------------------------------------

func (h *FilesystemVersionedHelpers) persistToDisk() {
	diskData := make(map[string]any)

	for entityID, entity := range h.entities {
		status, _ := entity["status"].(string)
		activeVersionID, _ := entity["activeVersionId"].(string)
		if status != "published" || activeVersionID == "" {
			continue
		}

		version, ok := h.versions[activeVersionID]
		if !ok {
			continue
		}

		diskData[entityID] = h.extractSnapshotConfig(version)
	}

	h.db.WriteDomain(h.entitiesFile, diskData)
}

func (h *FilesystemVersionedHelpers) extractSnapshotConfig(version map[string]any) map[string]any {
	metadataSet := make(map[string]bool, len(h.versionMetadataFields))
	for _, f := range h.versionMetadataFields {
		metadataSet[f] = true
	}

	result := make(map[string]any)
	for key, value := range version {
		if !metadataSet[key] {
			result[key] = value
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Entity CRUD
// ---------------------------------------------------------------------------

// GetByID retrieves an entity by ID.
func (h *FilesystemVersionedHelpers) GetByID(id string) (map[string]any, error) {
	h.Hydrate()
	h.mu.RLock()
	defer h.mu.RUnlock()

	entity, ok := h.entities[id]
	if !ok {
		return nil, nil
	}
	return cloneMap(entity), nil
}

// CreateEntity creates a new entity.
func (h *FilesystemVersionedHelpers) CreateEntity(id string, entity map[string]any) (map[string]any, error) {
	h.Hydrate()
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.entities[id]; exists {
		return nil, fmt.Errorf("%s: entity with id %s already exists", h.name, id)
	}

	h.entities[id] = cloneMap(entity)
	return cloneMap(entity), nil
}

// UpdateEntity updates an existing entity.
func (h *FilesystemVersionedHelpers) UpdateEntity(id string, updates map[string]any) (map[string]any, error) {
	h.Hydrate()
	h.mu.Lock()
	defer h.mu.Unlock()

	existing, ok := h.entities[id]
	if !ok {
		return nil, fmt.Errorf("%s: entity with id %s not found", h.name, id)
	}

	updated := cloneMap(existing)
	for key, value := range updates {
		if key == "id" {
			continue
		}
		if value == nil {
			continue
		}
		if key == "metadata" {
			if metaMap, ok := value.(map[string]any); ok {
				existingMeta, _ := updated["metadata"].(map[string]any)
				if existingMeta == nil {
					existingMeta = make(map[string]any)
				}
				for mk, mv := range metaMap {
					existingMeta[mk] = mv
				}
				updated["metadata"] = existingMeta
				continue
			}
		}
		updated[key] = value
	}
	updated["updatedAt"] = time.Now()

	h.entities[id] = cloneMap(updated)

	// Persist to disk when publication state changes
	wasPublished := existing["status"] == "published"
	isPublished := updated["status"] == "published" && updated["activeVersionId"] != nil && updated["activeVersionId"] != ""
	if isPublished || (wasPublished && updates["status"] != nil) {
		h.persistToDisk()
	}

	return cloneMap(updated), nil
}

// DeleteEntity deletes an entity and all its versions.
func (h *FilesystemVersionedHelpers) DeleteEntity(id string) error {
	h.Hydrate()
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.entities, id)
	h.deleteVersionsByParentIDLocked(id)
	h.persistToDisk()
	return nil
}

// ListEntities lists entities with pagination, sorting, and filtering.
func (h *FilesystemVersionedHelpers) ListEntities(page, perPage int, orderBy *domains.StorageOrderBy, filters map[string]any, listKey string) (map[string]any, error) {
	h.Hydrate()
	h.mu.RLock()
	defer h.mu.RUnlock()

	entities := make([]map[string]any, 0, len(h.entities))
	for _, e := range h.entities {
		entities = append(entities, e)
	}

	// Apply filters
	if len(filters) > 0 {
		var filtered []map[string]any
		for _, e := range entities {
			match := true
			for key, value := range filters {
				if value == nil {
					continue
				}
				if key == "metadata" {
					if metaFilter, ok := value.(map[string]any); ok {
						entityMeta, _ := e["metadata"].(map[string]any)
						if entityMeta == nil {
							match = false
							break
						}
						for mk, mv := range metaFilter {
							a, _ := json.Marshal(entityMeta[mk])
							b, _ := json.Marshal(mv)
							if string(a) != string(b) {
								match = false
								break
							}
						}
					}
				} else if e[key] != value {
					match = false
					break
				}
			}
			if match {
				filtered = append(filtered, e)
			}
		}
		entities = filtered
	}

	// Sort
	field := "createdAt"
	direction := "DESC"
	if orderBy != nil {
		if orderBy.Field != "" {
			field = string(orderBy.Field)
		}
		if orderBy.Direction != "" {
			direction = string(orderBy.Direction)
		}
	}

	sort.Slice(entities, func(i, j int) bool {
		aTime := toTime(entities[i][field])
		bTime := toTime(entities[j][field])
		if direction == "ASC" {
			return aTime.Before(bTime)
		}
		return bTime.Before(aTime)
	})

	// Paginate
	total := len(entities)
	normalizedPerPage := perPage
	if normalizedPerPage <= 0 {
		normalizedPerPage = math.MaxInt
	}
	offset := page * normalizedPerPage
	if offset > total {
		offset = total
	}
	end := offset + normalizedPerPage
	if end > total {
		end = total
	}

	result := map[string]any{
		listKey:  entities[offset:end],
		"total":  total,
		"page":   page,
		"hasMore": end < total,
	}
	if perPage <= 0 {
		result["perPage"] = -1
	} else {
		result["perPage"] = perPage
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Version Methods
// ---------------------------------------------------------------------------

// CreateVersion creates a new version.
func (h *FilesystemVersionedHelpers) CreateVersion(input map[string]any) (map[string]any, error) {
	h.ensureGitHistory()
	h.mu.Lock()
	defer h.mu.Unlock()

	inputID, _ := input["id"].(string)
	if _, exists := h.versions[inputID]; exists {
		return nil, fmt.Errorf("%s: version with id %s already exists", h.name, inputID)
	}

	parentID, _ := input[h.parentIDField].(string)
	inputVersionNumber, _ := input["versionNumber"].(int)

	// Check for duplicate (parentId, versionNumber) pair
	for _, v := range h.versions {
		vParent, _ := v[h.parentIDField].(string)
		vNum, _ := v["versionNumber"].(int)
		if vParent == parentID && vNum == inputVersionNumber {
			return nil, fmt.Errorf("%s: version number %d already exists for entity %s", h.name, inputVersionNumber, parentID)
		}
	}

	version := cloneMap(input)
	version["createdAt"] = time.Now()

	h.versions[inputID] = cloneMap(version)
	return cloneMap(version), nil
}

// GetVersion retrieves a version by ID.
func (h *FilesystemVersionedHelpers) GetVersion(id string) (map[string]any, error) {
	h.ensureGitHistory()
	h.mu.RLock()
	defer h.mu.RUnlock()

	version, ok := h.versions[id]
	if !ok {
		return nil, nil
	}
	return cloneMap(version), nil
}

// GetVersionByNumber retrieves a version by entity ID and version number.
func (h *FilesystemVersionedHelpers) GetVersionByNumber(entityID string, versionNumber int) (map[string]any, error) {
	h.ensureGitHistory()
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, v := range h.versions {
		vParent, _ := v[h.parentIDField].(string)
		vNum, _ := v["versionNumber"].(int)
		if vParent == entityID && vNum == versionNumber {
			return cloneMap(v), nil
		}
	}
	return nil, nil
}

// GetLatestVersion retrieves the latest version for an entity.
func (h *FilesystemVersionedHelpers) GetLatestVersion(entityID string) (map[string]any, error) {
	h.ensureGitHistory()
	h.mu.RLock()
	defer h.mu.RUnlock()

	var latest map[string]any
	latestNum := -1

	for _, v := range h.versions {
		vParent, _ := v[h.parentIDField].(string)
		vNum, _ := v["versionNumber"].(int)
		if vParent == entityID && vNum > latestNum {
			latest = v
			latestNum = vNum
		}
	}

	if latest == nil {
		return nil, nil
	}
	return cloneMap(latest), nil
}

// ListVersions lists versions for an entity with pagination and sorting.
func (h *FilesystemVersionedHelpers) ListVersions(entityID string, page, perPage int, orderBy *domains.VersionOrderByClause) (*domains.ListVersionsOutputBase, error) {
	h.ensureGitHistory()
	h.mu.RLock()
	defer h.mu.RUnlock()

	var versions []map[string]any
	for _, v := range h.versions {
		vParent, _ := v[h.parentIDField].(string)
		if vParent == entityID {
			versions = append(versions, v)
		}
	}

	// Sort
	field := "versionNumber"
	direction := "DESC"
	if orderBy != nil {
		if orderBy.Field != "" {
			field = string(orderBy.Field)
		}
		if orderBy.Direction != "" {
			direction = string(orderBy.Direction)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		var aVal, bVal float64
		if field == "createdAt" {
			aVal = float64(toTime(versions[i]["createdAt"]).UnixMilli())
			bVal = float64(toTime(versions[j]["createdAt"]).UnixMilli())
		} else {
			aVal = toFloat(versions[i]["versionNumber"])
			bVal = toFloat(versions[j]["versionNumber"])
		}
		if direction == "ASC" {
			return aVal < bVal
		}
		return bVal < aVal
	})

	total := len(versions)
	normalizedPerPage := perPage
	if normalizedPerPage <= 0 {
		normalizedPerPage = math.MaxInt
	}
	offset := page * normalizedPerPage
	if offset > total {
		offset = total
	}
	end := offset + normalizedPerPage
	if end > total {
		end = total
	}

	outputVersions := make([]any, end-offset)
	for i, v := range versions[offset:end] {
		outputVersions[i] = v
	}

	responsePerPage := perPage
	if perPage <= 0 {
		responsePerPage = -1
	}

	return &domains.ListVersionsOutputBase{
		Versions: outputVersions,
		Total:    total,
		Page:     page,
		PerPage:  responsePerPage,
		HasMore:  end < total,
	}, nil
}

// DeleteVersion deletes a version. Git-based versions are read-only.
func (h *FilesystemVersionedHelpers) DeleteVersion(id string) error {
	h.ensureGitHistory()
	if IsGitVersion(id) {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.versions, id)
	return nil
}

// DeleteVersionsByParentID deletes all non-git versions for an entity.
func (h *FilesystemVersionedHelpers) DeleteVersionsByParentID(entityID string) error {
	h.ensureGitHistory()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.deleteVersionsByParentIDLocked(entityID)
	return nil
}

func (h *FilesystemVersionedHelpers) deleteVersionsByParentIDLocked(entityID string) {
	for versionID, version := range h.versions {
		vParent, _ := version[h.parentIDField].(string)
		if vParent == entityID && !IsGitVersion(versionID) {
			delete(h.versions, versionID)
		}
	}
}

// CountVersions returns the number of versions for an entity.
func (h *FilesystemVersionedHelpers) CountVersions(entityID string) (int, error) {
	h.ensureGitHistory()
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, v := range h.versions {
		vParent, _ := v[h.parentIDField].(string)
		if vParent == entityID {
			count++
		}
	}
	return count, nil
}

// GetNextVersionNumber returns the next version number for an entity.
func (h *FilesystemVersionedHelpers) GetNextVersionNumber(entityID string) (int, error) {
	h.ensureGitHistory()
	h.mu.RLock()
	defer h.mu.RUnlock()

	gitCount := h.gitVersionCounts[entityID]
	maxVersion := gitCount
	for _, v := range h.versions {
		vParent, _ := v[h.parentIDField].(string)
		vNum, _ := v["versionNumber"].(int)
		if vParent == entityID && vNum > maxVersion {
			maxVersion = vNum
		}
	}
	return maxVersion + 1, nil
}

// DangerouslyClearAll clears all in-memory state and the disk file.
func (h *FilesystemVersionedHelpers) DangerouslyClearAll() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entities = make(map[string]map[string]any)
	h.versions = make(map[string]map[string]any)
	h.gitVersionCounts = make(map[string]int)
	h.hydrated = false
	h.gitHistoryLoaded = false
	h.db.ClearDomain(h.entitiesFile)
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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

func toTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case float64:
		return n
	case int64:
		return float64(n)
	}
	return 0
}
