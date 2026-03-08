// Ported from: packages/core/src/storage/filesystem.ts
package storage

import (
	"path/filepath"

	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
	"github.com/brainlet/brainkit/agent-kit/core/storage/fsutil"
)

// ---------------------------------------------------------------------------
// FilesystemStoreConfig
// ---------------------------------------------------------------------------

// FilesystemStoreConfig holds configuration for the filesystem-based store.
type FilesystemStoreConfig struct {
	// Dir is the directory to store JSON files in.
	// Defaults to ".mastra-storage/" relative to the current working directory.
	Dir string
}

// ---------------------------------------------------------------------------
// FilesystemStore
// ---------------------------------------------------------------------------

// FilesystemStore is a filesystem-based storage adapter for the Mastra Editor.
//
// Stores editor primitives (agents, prompt blocks, scorer definitions,
// MCP clients, MCP servers, workspaces, skills) as JSON files on disk.
// This enables Git-based version tracking instead of database-based versioning.
//
// Only implements the 7 editor domains. Other domains (memory, workflows, scores,
// observability, datasets, experiments, blobs) are left nil and should be
// provided by a separate store via the "editor" shorthand on MastraCompositeStore.
type FilesystemStore struct {
	*MastraCompositeStore
	db  *fsutil.FilesystemDB
	dir string
}

// NewFilesystemStore creates a new FilesystemStore.
func NewFilesystemStore(config FilesystemStoreConfig) (*FilesystemStore, error) {
	dir := config.Dir
	if dir == "" {
		dir = ".mastra-storage"
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	cs, err := NewMastraCompositeStore(MastraCompositeStoreConfig{
		ID:   "filesystem",
		Name: "FilesystemStore",
	})
	if err != nil {
		return nil, err
	}

	db := fsutil.NewFilesystemDB(dir)

	fs := &FilesystemStore{
		MastraCompositeStore: cs,
		db:                   db,
		dir:                  dir,
	}

	// Only editor domains are provided; other domains (workflows, scores, memory, etc.)
	// should come from a default store when using the "editor" shorthand on
	// MastraCompositeStore.
	//
	// TODO: create FilesystemAgentsStorage, FilesystemPromptBlocksStorage, etc.
	// and assign them here when the filesystem domain implementations are ported.
	// For now, we assign nil placeholders so the structure exists.
	fs.Stores = &StorageDomains{
		Agents:            nil, // TODO: FilesystemAgentsStorage
		PromptBlocks:      nil, // TODO: FilesystemPromptBlocksStorage
		ScorerDefinitions: nil, // TODO: FilesystemScorerDefinitionsStorage
		MCPClients:        nil, // TODO: FilesystemMCPClientsStorage
		MCPServers:        nil, // TODO: FilesystemMCPServersStorage
		Workspaces:        nil, // TODO: FilesystemWorkspacesStorage
		Skills:            nil, // TODO: FilesystemSkillsStorage
	}

	// Keep db reference accessible for filesystem domain impls
	_ = fs.db

	return fs, nil
}

// Dir returns the absolute path to the storage directory.
func (fs *FilesystemStore) Dir() string {
	return fs.dir
}

// DB returns the underlying FilesystemDB instance.
// This is useful for filesystem domain implementations that need direct access.
func (fs *FilesystemStore) DB() *fsutil.FilesystemDB {
	return fs.db
}

// SetDomainStore sets a specific domain store on the filesystem store.
// This allows filesystem domain implementations to be registered after creation.
func (fs *FilesystemStore) SetDomainStore(name DomainName, store domains.StorageDomain) {
	if fs.Stores == nil {
		fs.Stores = &StorageDomains{}
	}
	switch name {
	case DomainAgents:
		fs.Stores.Agents = store
	case DomainPromptBlocks:
		fs.Stores.PromptBlocks = store
	case DomainScorerDefinitions:
		fs.Stores.ScorerDefinitions = store
	case DomainMCPClients:
		fs.Stores.MCPClients = store
	case DomainMCPServers:
		fs.Stores.MCPServers = store
	case DomainWorkspaces:
		fs.Stores.Workspaces = store
	case DomainSkills:
		fs.Stores.Skills = store
	}
}
