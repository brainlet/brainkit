// Ported from: packages/core/src/storage/domains/inmemory-db.ts
package domains

import "sync"

// InMemoryDB is a thin database layer for in-memory storage.
// It holds all the maps that store data, similar to how a real database
// connection (pgx pool, libsql client) is shared across domains.
//
// Each domain receives a reference to this db and operates on the relevant maps.
// All operations are protected by a sync.RWMutex for thread safety.
type InMemoryDB struct {
	mu sync.RWMutex

	// Core domain maps.
	// In TypeScript these are typed Map<string, T> for each domain.
	// In Go we use map[string]any — concrete domain implementations cast as needed.

	Threads   map[string]any // StorageThreadType
	Messages  map[string]any // StorageMessageType
	Resources map[string]any // StorageResourceType
	Workflows map[string]any // StorageWorkflowRun
	Scores    map[string]any // ScoreRowData
	Traces    map[string]any // TraceEntry

	// Versioned entity domain maps.
	Agents                  map[string]any // StorageAgentType
	AgentVersions           map[string]any // AgentVersion
	PromptBlocks            map[string]any // StoragePromptBlockType
	PromptBlockVersions     map[string]any // PromptBlockVersion
	ScorerDefinitions       map[string]any // StorageScorerDefinitionType
	ScorerDefinitionVersions map[string]any // ScorerDefinitionVersion
	MCPClients              map[string]any // StorageMCPClientType
	MCPClientVersions       map[string]any // MCPClientVersion
	MCPServers              map[string]any // StorageMCPServerType
	MCPServerVersions       map[string]any // MCPServerVersion
	Workspaces              map[string]any // StorageWorkspaceType
	WorkspaceVersions       map[string]any // WorkspaceVersion
	Skills                  map[string]any // StorageSkillType
	SkillVersions           map[string]any // SkillVersion

	// Observational memory records, keyed by resourceId.
	// Each value is a []any representing an array of records (generations).
	ObservationalMemory map[string][]any // ObservationalMemoryRecord

	// Dataset domain maps.
	Datasets        map[string]any   // DatasetRecord
	DatasetItems    map[string][]any // DatasetItemRow (array per dataset)
	DatasetVersions map[string]any   // DatasetVersion

	// Experiment domain maps.
	Experiments       map[string]any // Experiment
	ExperimentResults map[string]any // ExperimentResult
}

// NewInMemoryDB creates a new InMemoryDB with all maps initialized.
func NewInMemoryDB() *InMemoryDB {
	db := &InMemoryDB{}
	db.initMaps()
	return db
}

// initMaps initializes all internal maps. Called by NewInMemoryDB and Clear.
func (db *InMemoryDB) initMaps() {
	db.Threads = make(map[string]any)
	db.Messages = make(map[string]any)
	db.Resources = make(map[string]any)
	db.Workflows = make(map[string]any)
	db.Scores = make(map[string]any)
	db.Traces = make(map[string]any)
	db.Agents = make(map[string]any)
	db.AgentVersions = make(map[string]any)
	db.PromptBlocks = make(map[string]any)
	db.PromptBlockVersions = make(map[string]any)
	db.ScorerDefinitions = make(map[string]any)
	db.ScorerDefinitionVersions = make(map[string]any)
	db.MCPClients = make(map[string]any)
	db.MCPClientVersions = make(map[string]any)
	db.MCPServers = make(map[string]any)
	db.MCPServerVersions = make(map[string]any)
	db.Workspaces = make(map[string]any)
	db.WorkspaceVersions = make(map[string]any)
	db.Skills = make(map[string]any)
	db.SkillVersions = make(map[string]any)
	db.ObservationalMemory = make(map[string][]any)
	db.Datasets = make(map[string]any)
	db.DatasetItems = make(map[string][]any)
	db.DatasetVersions = make(map[string]any)
	db.Experiments = make(map[string]any)
	db.ExperimentResults = make(map[string]any)
}

// Clear removes all data from all collections. Useful for testing.
// Acquires a write lock for the duration of the operation.
func (db *InMemoryDB) Clear() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.initMaps()
}

// RLock acquires a read lock on the database.
// Callers must call RUnlock when done.
func (db *InMemoryDB) RLock() {
	db.mu.RLock()
}

// RUnlock releases the read lock on the database.
func (db *InMemoryDB) RUnlock() {
	db.mu.RUnlock()
}

// Lock acquires a write lock on the database.
// Callers must call Unlock when done.
func (db *InMemoryDB) Lock() {
	db.mu.Lock()
}

// Unlock releases the write lock on the database.
func (db *InMemoryDB) Unlock() {
	db.mu.Unlock()
}
