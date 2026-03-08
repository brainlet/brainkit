// Ported from: packages/core/src/storage/base.ts
package storage

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ---------------------------------------------------------------------------
// StorageDomains — the set of all domain-specific stores
// ---------------------------------------------------------------------------

// StorageDomains holds references to domain-specific storage interfaces.
// Required domains (Workflows, Scores, Memory) must be non-nil for a fully
// functional composite store. Optional domains may be nil.
type StorageDomains struct {
	// Required domains
	Workflows domains.StorageDomain // WorkflowsStorage
	Scores    domains.StorageDomain // ScoresStorage
	Memory    domains.StorageDomain // MemoryStorage

	// Optional domains
	Observability    domains.StorageDomain // ObservabilityStorage
	Agents           domains.StorageDomain // AgentsStorage
	Datasets         domains.StorageDomain // DatasetsStorage
	Experiments      domains.StorageDomain // ExperimentsStorage
	PromptBlocks     domains.StorageDomain // PromptBlocksStorage
	ScorerDefinitions domains.StorageDomain // ScorerDefinitionsStorage
	MCPClients       domains.StorageDomain // MCPClientsStorage
	MCPServers       domains.StorageDomain // MCPServersStorage
	Workspaces       domains.StorageDomain // WorkspacesStorage
	Skills           domains.StorageDomain // SkillsStorage
	Blobs            domains.StorageDomain // BlobStore
}

// allDomains returns every non-nil domain as a slice for iteration.
func (s *StorageDomains) allDomains() []domains.StorageDomain {
	candidates := []domains.StorageDomain{
		s.Memory,
		s.Workflows,
		s.Scores,
		s.Observability,
		s.Agents,
		s.Datasets,
		s.Experiments,
		s.PromptBlocks,
		s.ScorerDefinitions,
		s.MCPClients,
		s.MCPServers,
		s.Workspaces,
		s.Skills,
		s.Blobs,
	}
	out := make([]domains.StorageDomain, 0, len(candidates))
	for _, d := range candidates {
		if d != nil {
			out = append(out, d)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// EDITOR_DOMAINS
// ---------------------------------------------------------------------------

// EditorDomain identifies a storage domain used by the Mastra Editor.
type EditorDomain string

const (
	EditorDomainAgents           EditorDomain = "agents"
	EditorDomainPromptBlocks     EditorDomain = "promptBlocks"
	EditorDomainScorerDefinitions EditorDomain = "scorerDefinitions"
	EditorDomainMCPClients       EditorDomain = "mcpClients"
	EditorDomainMCPServers       EditorDomain = "mcpServers"
	EditorDomainWorkspaces       EditorDomain = "workspaces"
	EditorDomainSkills           EditorDomain = "skills"
)

// EditorDomains lists every domain key used by the Mastra Editor.
// Used by the "editor" shorthand on MastraCompositeStoreConfig to route all
// editor-related domains to a single store.
var EditorDomains = []EditorDomain{
	EditorDomainAgents,
	EditorDomainPromptBlocks,
	EditorDomainScorerDefinitions,
	EditorDomainMCPClients,
	EditorDomainMCPServers,
	EditorDomainWorkspaces,
	EditorDomainSkills,
}

// editorDomainSet provides O(1) membership lookups for editor domains.
var editorDomainSet = func() map[EditorDomain]struct{} {
	s := make(map[EditorDomain]struct{}, len(EditorDomains))
	for _, d := range EditorDomains {
		s[d] = struct{}{}
	}
	return s
}()

// isEditorDomain reports whether the given domain name is an editor domain.
func isEditorDomain(name string) bool {
	_, ok := editorDomainSet[EditorDomain(name)]
	return ok
}

// ---------------------------------------------------------------------------
// Pagination helpers
// ---------------------------------------------------------------------------

// NormalizePerPage normalizes a perPage input for pagination queries.
//
// Semantics (matching the TypeScript source):
//   - perPage == nil  → use defaultValue
//   - *perPage == PerPageDisabled (-1) → math.MaxInt (get all results)
//   - *perPage == 0   → 0 (return zero results)
//   - *perPage > 0    → use that value
//   - *perPage < 0 && != PerPageDisabled → error
//
// PerPageDisabled is defined in domains/shared.go as -1.
func NormalizePerPage(perPage *int, defaultValue int) (int, error) {
	if perPage == nil {
		return defaultValue, nil
	}
	v := *perPage
	if v == domains.PerPageDisabled {
		return math.MaxInt, nil
	}
	if v == 0 {
		return 0, nil
	}
	if v > 0 {
		return v, nil
	}
	// Negative (but not PerPageDisabled)
	return 0, errors.New("perPage must be >= 0")
}

// PaginationResult holds the computed offset and perPage for a paginated query.
type PaginationResult struct {
	// Offset is the number of items to skip.
	Offset int
	// PerPage is the normalized per-page value, or domains.PerPageDisabled (-1) when
	// pagination is disabled (fetch all).
	PerPage int
}

// CalculatePagination computes the offset and the response perPage value.
// When the original perPageInput is PerPageDisabled (false in TS), offset is
// always 0 regardless of page.
//
//   - page: zero-indexed page number
//   - perPageInput: original input (nil = undefined, PerPageDisabled = false)
//   - normalizedPerPage: result of NormalizePerPage
func CalculatePagination(page int, perPageInput *int, normalizedPerPage int) PaginationResult {
	if perPageInput != nil && *perPageInput == domains.PerPageDisabled {
		return PaginationResult{
			Offset:  0,
			PerPage: domains.PerPageDisabled,
		}
	}
	return PaginationResult{
		Offset:  page * normalizedPerPage,
		PerPage: normalizedPerPage,
	}
}

// ---------------------------------------------------------------------------
// MastraCompositeStoreConfig
// ---------------------------------------------------------------------------

// MastraStorageDomains is a partial set of domain overrides.
// Each field, when non-nil, takes precedence over both the editor and default
// storage for that domain.
type MastraStorageDomains struct {
	Workflows         domains.StorageDomain
	Scores            domains.StorageDomain
	Memory            domains.StorageDomain
	Observability     domains.StorageDomain
	Agents            domains.StorageDomain
	Datasets          domains.StorageDomain
	Experiments       domains.StorageDomain
	PromptBlocks      domains.StorageDomain
	ScorerDefinitions domains.StorageDomain
	MCPClients        domains.StorageDomain
	MCPServers        domains.StorageDomain
	Workspaces        domains.StorageDomain
	Skills            domains.StorageDomain
	Blobs             domains.StorageDomain
}

// hasAny reports whether at least one field is non-nil.
func (d *MastraStorageDomains) hasAny() bool {
	if d == nil {
		return false
	}
	return d.Workflows != nil || d.Scores != nil || d.Memory != nil ||
		d.Observability != nil || d.Agents != nil || d.Datasets != nil ||
		d.Experiments != nil || d.PromptBlocks != nil || d.ScorerDefinitions != nil ||
		d.MCPClients != nil || d.MCPServers != nil || d.Workspaces != nil ||
		d.Skills != nil || d.Blobs != nil
}

// MastraCompositeStoreConfig holds the configuration for MastraCompositeStore.
//
// Can be used in two ways:
//  1. By store implementations: set ID and Name, then populate Stores directly.
//  2. For composition: set ID with Default, Editor, and/or Domains to compose
//     domains from multiple stores.
type MastraCompositeStoreConfig struct {
	// ID is a unique identifier for this storage instance (required, non-empty).
	ID string

	// Name of the storage adapter (used for logging).
	// Defaults to "MastraCompositeStore" when empty.
	Name string

	// Default storage adapter used as fallback for domains not explicitly specified.
	Default *MastraCompositeStore

	// Editor is a shorthand that routes all editor-related domains
	// (agents, promptBlocks, scorerDefinitions, mcpClients, mcpServers,
	// workspaces, skills) to a single store.
	//
	// Priority: Domains > Editor > Default.
	Editor *MastraCompositeStore

	// Domains provides individual domain overrides. Each domain can come from
	// a different storage adapter. These take precedence over both Editor and Default.
	Domains *MastraStorageDomains

	// DisableInit when true prevents automatic initialization (table creation /
	// migrations). You must call Init explicitly in your CI/CD scripts.
	DisableInit bool
}

// ---------------------------------------------------------------------------
// MastraCompositeStore
// ---------------------------------------------------------------------------

// MastraCompositeStore is the base type for all Mastra storage adapters.
//
// It can be used in two ways:
//
//  1. Extended by store implementations (embedding this struct) which populate
//     Stores with their domain implementations.
//
//  2. Directly instantiated for composition — mixing domains from multiple
//     storage backends using Default, Editor, and Domains options.
//
// All domain-specific operations should be accessed through GetStore.
type MastraCompositeStore struct {
	*agentkit.MastraBase

	ID          string
	Stores      *StorageDomains
	DisableInit bool

	// shouldCacheInit controls whether Init results are cached (default true).
	shouldCacheInit bool

	mu             sync.Mutex
	hasInitialized bool
	initErr        error
}

// NewMastraCompositeStore creates a new MastraCompositeStore with the given config.
//
// When composition config is provided (Default, Editor, or Domains), stores are
// composed with the priority: Domains > Editor (for editor domains) > Default.
//
// When none of Default/Editor/Domains are provided, the caller (a store
// implementation) is expected to populate Stores directly.
func NewMastraCompositeStore(cfg MastraCompositeStoreConfig) (*MastraCompositeStore, error) {
	name := cfg.Name
	if name == "" {
		name = "MastraCompositeStore"
	}

	id := cfg.ID
	if id == "" {
		return nil, fmt.Errorf("%s: id must be provided and cannot be empty", name)
	}

	base := agentkit.NewMastraBase(agentkit.MastraBaseOptions{
		Component: logger.RegisteredLoggerStorage,
		Name:      name,
	})

	cs := &MastraCompositeStore{
		MastraBase:      base,
		ID:              id,
		DisableInit:     cfg.DisableInit,
		shouldCacheInit: true,
	}

	// If composition config is provided, compose the stores.
	if cfg.Default != nil || cfg.Editor != nil || cfg.Domains != nil {
		var defaultStores *StorageDomains
		if cfg.Default != nil {
			defaultStores = cfg.Default.Stores
		}
		var editorStores *StorageDomains
		if cfg.Editor != nil {
			editorStores = cfg.Editor.Stores
		}
		domainOverrides := cfg.Domains

		// Validate that at least one storage source provides domains.
		hasDefaultDomains := defaultStores != nil && len(defaultStores.allDomains()) > 0
		hasEditorDomains := editorStores != nil && len(editorStores.allDomains()) > 0
		hasOverrideDomains := domainOverrides.hasAny()

		if !hasDefaultDomains && !hasEditorDomains && !hasOverrideDomains {
			return nil, errors.New(
				"MastraCompositeStore requires at least one storage source. " +
					"Provide a default storage, an editor storage, or domain overrides",
			)
		}

		// resolve returns the highest-priority domain store for the given key.
		// Priority: domains > editor (for editor domains) > default.
		resolve := func(key string, fromOverride domains.StorageDomain, fromEditor, fromDefault func() domains.StorageDomain) domains.StorageDomain {
			// 1. Explicit domain override
			if fromOverride != nil {
				return fromOverride
			}
			// 2. Editor store (only for editor domains)
			if isEditorDomain(key) && editorStores != nil {
				if d := fromEditor(); d != nil {
					return d
				}
			}
			// 3. Default store
			if defaultStores != nil {
				return fromDefault()
			}
			return nil
		}

		// overrideField returns nil safely when domainOverrides is nil.
		var ov MastraStorageDomains
		if domainOverrides != nil {
			ov = *domainOverrides
		}

		editorField := func(get func(*StorageDomains) domains.StorageDomain) func() domains.StorageDomain {
			return func() domains.StorageDomain { return get(editorStores) }
		}
		defaultField := func(get func(*StorageDomains) domains.StorageDomain) func() domains.StorageDomain {
			return func() domains.StorageDomain { return get(defaultStores) }
		}
		mem := func(s *StorageDomains) domains.StorageDomain { return s.Memory }
		wf := func(s *StorageDomains) domains.StorageDomain { return s.Workflows }
		sc := func(s *StorageDomains) domains.StorageDomain { return s.Scores }
		obs := func(s *StorageDomains) domains.StorageDomain { return s.Observability }
		ag := func(s *StorageDomains) domains.StorageDomain { return s.Agents }
		ds := func(s *StorageDomains) domains.StorageDomain { return s.Datasets }
		exp := func(s *StorageDomains) domains.StorageDomain { return s.Experiments }
		pb := func(s *StorageDomains) domains.StorageDomain { return s.PromptBlocks }
		sd := func(s *StorageDomains) domains.StorageDomain { return s.ScorerDefinitions }
		mc := func(s *StorageDomains) domains.StorageDomain { return s.MCPClients }
		ms := func(s *StorageDomains) domains.StorageDomain { return s.MCPServers }
		ws := func(s *StorageDomains) domains.StorageDomain { return s.Workspaces }
		sk := func(s *StorageDomains) domains.StorageDomain { return s.Skills }
		bl := func(s *StorageDomains) domains.StorageDomain { return s.Blobs }

		cs.Stores = &StorageDomains{
			Memory:            resolve("memory", ov.Memory, editorField(mem), defaultField(mem)),
			Workflows:         resolve("workflows", ov.Workflows, editorField(wf), defaultField(wf)),
			Scores:            resolve("scores", ov.Scores, editorField(sc), defaultField(sc)),
			Observability:     resolve("observability", ov.Observability, editorField(obs), defaultField(obs)),
			Agents:            resolve("agents", ov.Agents, editorField(ag), defaultField(ag)),
			Datasets:          resolve("datasets", ov.Datasets, editorField(ds), defaultField(ds)),
			Experiments:       resolve("experiments", ov.Experiments, editorField(exp), defaultField(exp)),
			PromptBlocks:      resolve("promptBlocks", ov.PromptBlocks, editorField(pb), defaultField(pb)),
			ScorerDefinitions: resolve("scorerDefinitions", ov.ScorerDefinitions, editorField(sd), defaultField(sd)),
			MCPClients:        resolve("mcpClients", ov.MCPClients, editorField(mc), defaultField(mc)),
			MCPServers:        resolve("mcpServers", ov.MCPServers, editorField(ms), defaultField(ms)),
			Workspaces:        resolve("workspaces", ov.Workspaces, editorField(ws), defaultField(ws)),
			Skills:            resolve("skills", ov.Skills, editorField(sk), defaultField(sk)),
			Blobs:             resolve("blobs", ov.Blobs, editorField(bl), defaultField(bl)),
		}
	}

	return cs, nil
}

// ---------------------------------------------------------------------------
// Domain access
// ---------------------------------------------------------------------------

// DomainName identifies a named storage domain for use with GetStore.
type DomainName string

const (
	DomainMemory            DomainName = "memory"
	DomainWorkflows         DomainName = "workflows"
	DomainScores            DomainName = "scores"
	DomainObservability     DomainName = "observability"
	DomainAgents            DomainName = "agents"
	DomainDatasets          DomainName = "datasets"
	DomainExperiments       DomainName = "experiments"
	DomainPromptBlocks      DomainName = "promptBlocks"
	DomainScorerDefinitions DomainName = "scorerDefinitions"
	DomainMCPClients        DomainName = "mcpClients"
	DomainMCPServers        DomainName = "mcpServers"
	DomainWorkspaces        DomainName = "workspaces"
	DomainSkills            DomainName = "skills"
	DomainBlobs             DomainName = "blobs"
)

// GetStore returns the domain-specific storage interface for the given domain,
// or nil if no store is configured for that domain.
//
// In the TypeScript source this is async to support lazy init; in Go the
// caller should use AugmentWithInit for auto-initialization if needed.
func (cs *MastraCompositeStore) GetStore(name DomainName) domains.StorageDomain {
	if cs.Stores == nil {
		return nil
	}
	switch name {
	case DomainMemory:
		return cs.Stores.Memory
	case DomainWorkflows:
		return cs.Stores.Workflows
	case DomainScores:
		return cs.Stores.Scores
	case DomainObservability:
		return cs.Stores.Observability
	case DomainAgents:
		return cs.Stores.Agents
	case DomainDatasets:
		return cs.Stores.Datasets
	case DomainExperiments:
		return cs.Stores.Experiments
	case DomainPromptBlocks:
		return cs.Stores.PromptBlocks
	case DomainScorerDefinitions:
		return cs.Stores.ScorerDefinitions
	case DomainMCPClients:
		return cs.Stores.MCPClients
	case DomainMCPServers:
		return cs.Stores.MCPServers
	case DomainWorkspaces:
		return cs.Stores.Workspaces
	case DomainSkills:
		return cs.Stores.Skills
	case DomainBlobs:
		return cs.Stores.Blobs
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

// Init initializes all domain stores. This creates necessary tables, indexes,
// and performs any required migrations.
//
// Init is safe to call concurrently. When shouldCacheInit is true (the default),
// subsequent calls after a successful first init are no-ops.
func (cs *MastraCompositeStore) Init(ctx context.Context) error {
	cs.mu.Lock()
	if cs.shouldCacheInit && cs.hasInitialized {
		err := cs.initErr
		cs.mu.Unlock()
		return err
	}
	cs.mu.Unlock()

	if cs.Stores == nil {
		cs.mu.Lock()
		cs.hasInitialized = true
		cs.mu.Unlock()
		return nil
	}

	// Initialize all domain stores concurrently.
	allDomains := cs.Stores.allDomains()
	errs := make([]error, len(allDomains))
	var wg sync.WaitGroup
	wg.Add(len(allDomains))
	for i, d := range allDomains {
		go func(idx int, dom domains.StorageDomain) {
			defer wg.Done()
			errs[idx] = dom.Init(ctx)
		}(i, d)
	}
	wg.Wait()

	initErr := errors.Join(errs...)

	cs.mu.Lock()
	cs.hasInitialized = true
	cs.initErr = initErr
	cs.mu.Unlock()

	return initErr
}

// ---------------------------------------------------------------------------
// Deprecated aliases
// ---------------------------------------------------------------------------

// MastraStorageConfig is a deprecated alias for MastraCompositeStoreConfig.
// Deprecated: Use MastraCompositeStoreConfig instead. This alias will be removed in a future version.
type MastraStorageConfig = MastraCompositeStoreConfig

// MastraStorage is a deprecated alias for MastraCompositeStore.
// Deprecated: Use MastraCompositeStore instead. This alias will be removed in a future version.
type MastraStorage = MastraCompositeStore
