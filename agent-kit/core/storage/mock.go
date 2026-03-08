// Ported from: packages/core/src/storage/mock.ts
package storage

import (
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/agents"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/blobs"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/datasets"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/experiments"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/mcpclients"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/mcpservers"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/observability"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/operations"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/promptblocks"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/scorerdefinitions"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/scores"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/skills"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/workspaces"
)

// ---------------------------------------------------------------------------
// InMemoryStore
// ---------------------------------------------------------------------------

// InMemoryStore is an in-memory storage implementation for testing and development.
//
// All data is stored in memory and will be lost when the process ends.
// Access domain-specific storage via GetStore().
type InMemoryStore struct {
	*MastraCompositeStore

	// db is the internal database layer shared across all domains.
	db *domains.InMemoryDB
}

// NewInMemoryStore creates a new InMemoryStore.
func NewInMemoryStore(id string) (*InMemoryStore, error) {
	if id == "" {
		id = "in-memory"
	}

	cs, err := NewMastraCompositeStore(MastraCompositeStoreConfig{
		ID:   id,
		Name: "InMemoryStorage",
	})
	if err != nil {
		return nil, err
	}

	db := domains.NewInMemoryDB()

	store := &InMemoryStore{
		MastraCompositeStore: cs,
		db:                   db,
	}

	// Create all domain instances with the shared db.
	// Each domain maps to a concrete in-memory implementation.
	store.Stores = &StorageDomains{
		// TODO: InMemoryMemory is not imported here because the memory domain's
		// inmemory implementation requires types not yet in scope.
		// Assign nil; callers can set it post-construction.
		Memory:            nil, // TODO: memory.NewInMemoryMemory(db)
		Workflows:         workflows.NewInMemoryWorkflowsStorage(),
		Scores:            scores.NewInMemoryScoresStorage(),
		Observability:     observability.NewInMemoryObservabilityStorage(),
		Agents:            agents.NewInMemoryAgentsStorage(),
		Datasets:          datasets.NewInMemoryDatasetsStorage(),
		Experiments:       experiments.NewInMemoryExperimentsStorage(),
		PromptBlocks:      promptblocks.NewInMemoryPromptBlocksStorage(),
		ScorerDefinitions: scorerdefinitions.NewInMemoryScorerDefinitionsStorage(),
		MCPClients:        mcpclients.NewInMemoryMCPClientsStorage(),
		MCPServers:        mcpservers.NewInMemoryMCPServersStorage(),
		Workspaces:        workspaces.NewInMemoryWorkspacesStorage(),
		Skills:            skills.NewInMemorySkillsStorage(),
		Blobs:             blobs.NewInMemoryBlobStore(),
	}

	// Mark as initialized (in-memory doesn't need async init).
	store.hasInitialized = true

	return store, nil
}

// Clear clears all data from the in-memory database.
// Useful for testing.
//
// Deprecated: Use DangerouslyClearAll() on individual domains instead.
func (s *InMemoryStore) Clear() {
	s.db.Clear()
}

// ---------------------------------------------------------------------------
// MockStore (alias)
// ---------------------------------------------------------------------------

// MockStore is an alias for InMemoryStore, matching the TypeScript export.
type MockStore = InMemoryStore

// NewMockStore creates a new MockStore (alias for NewInMemoryStore).
func NewMockStore() (*MockStore, error) {
	return NewInMemoryStore("in-memory")
}

// ---------------------------------------------------------------------------
// Check that domain constructors exist
// ---------------------------------------------------------------------------

// These compile-time checks verify that the domain constructor functions
// referenced above exist. If any package doesn't export the expected
// constructor, this file won't compile.
var (
	_ = workflows.NewInMemoryWorkflowsStorage
	_ = scores.NewInMemoryScoresStorage
	_ = observability.NewInMemoryObservabilityStorage
	_ = agents.NewInMemoryAgentsStorage
	_ = datasets.NewInMemoryDatasetsStorage
	_ = experiments.NewInMemoryExperimentsStorage
	_ = promptblocks.NewInMemoryPromptBlocksStorage
	_ = scorerdefinitions.NewInMemoryScorerDefinitionsStorage
	_ = mcpclients.NewInMemoryMCPClientsStorage
	_ = mcpservers.NewInMemoryMCPServersStorage
	_ = workspaces.NewInMemoryWorkspacesStorage
	_ = skills.NewInMemorySkillsStorage
	_ = blobs.NewInMemoryBlobStore
	_ = operations.NewInMemoryStoreOperations
)
