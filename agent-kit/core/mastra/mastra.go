// Ported from: packages/core/src/mastra/index.ts
package mastra

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/brainlet/brainkit/agent-kit/core/cache"
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/events"
	"github.com/brainlet/brainkit/agent-kit/core/hooks"
	"github.com/brainlet/brainkit/agent-kit/core/interfaces"
	"github.com/brainlet/brainkit/agent-kit/core/llm/model"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/observability"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	storageworkflows "github.com/brainlet/brainkit/agent-kit/core/storage/domains/workflows"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
	"github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// Cross-package dependency interfaces
//
// Agent and MastraScorer are defined in core/interfaces to break circular
// dependencies. Other stub interfaces remain here until their dependency
// cycles are resolved.
//
// Method signatures are aligned with the real packages so that concrete
// implementations satisfy these interfaces without adapters:
//   - ID()  matches vector.MastraVector, processors.Processor, memory.MastraMemory,
//           mcp.MCPServerBase, workspace.Workspace, etc.
//   - SetLogger() comes from agentkit.MastraBase (embedded in most primitives).
//
// Import of the concrete packages is deferred until the dependency graph is
// verified cycle-free for every pair.  Until then the interfaces here act as
// the contract.
// ---------------------------------------------------------------------------

// Agent is the shared interface for agent instances.
// Defined in core/interfaces to break the circular dependency between
// mastra and agent packages. See interfaces.Agent for full documentation.
type Agent = interfaces.Agent

// AgentPrimitives holds the primitives passed to agent registration.
type AgentPrimitives struct {
	Logger  logger.IMastraLogger
	Storage *storage.MastraCompositeStore
	Agents  map[string]Agent
	TTS     map[string]MastraTTS
	Vectors map[string]MastraVector
}

// ScorerEntry pairs a scorer with metadata, used in agent/workflow scorer listings.
// Defined in core/interfaces to break circular dependencies.
type ScorerEntry = interfaces.ScorerEntry

// ToolLoopAgentLike is a stub interface for AI SDK v6 ToolLoopAgent instances.
// STUB REASON: Cannot import toolloopagent due to circular dependency:
// toolloopagent depends on agent types which depend on core. Real type is
// toolloopagent.ToolLoopAgent struct. Minimal interface for detection only.
type ToolLoopAgentLike interface {
	IsToolLoopAgent() bool
}

// MastraVector is a stub interface for vector store instances.
// STUB REASON: The real vector.MastraVector interface uses different method signatures
// (QueryVector takes context.Context + QueryVectorParams). This stub captures the
// minimal contract (ID, SetLogger) needed by Mastra registry. Real type matches on ID().
type MastraVector interface {
	// ID returns the vector store's unique identifier.
	// Matches vector.MastraVector.ID() and vector.MastraVectorBase.ID().
	ID() string
	SetLogger(l logger.IMastraLogger)
}

// MastraTTS is a stub interface for text-to-speech providers.
// STUB REASON: The real tts.MastraTTS struct has Generate/Stream methods with typed
// params (GenerateInput, StreamInput). This stub captures only SetLogger needed for
// Mastra registration. Real type embeds MastraBase → has SetLogger.
type MastraTTS interface {
	SetLogger(l logger.IMastraLogger)
}

// MastraScorer is the shared interface for scorer instances.
// Defined in core/interfaces to break the circular dependency between
// mastra and evals packages. See interfaces.MastraScorer for full documentation.
type MastraScorer = interfaces.MastraScorer

// ToolAction is a stub interface for tool instances.
// STUB REASON: The real tools.ToolAction struct has public .ID field (accessed directly,
// not via method) and many more fields/methods. This stub captures only GetID() needed
// by Mastra registry. Cannot import tools without evaluating circular dependency chain.
type ToolAction interface {
	// GetID returns the tool's unique identifier.
	// Real: tools.ToolAction.ID field (accessed directly).
	GetID() string
}

// Processor is a stub interface for processor instances.
// STUB REASON: The real processors.Processor interface has 10+ methods (ID, Name,
// Description, ProcessorIndex, ProcessInput, ProcessInputStep, ProcessOutput,
// ProcessOutputStream, etc.). This stub captures only ID() and RegisterMastra needed
// by Mastra registry. RegisterMastra comes from processors.MastraRegistrable.
type Processor interface {
	// ID returns the processor's unique identifier.
	// Matches processors.Processor.ID() and processors.BaseProcessor.ID().
	ID() string
	RegisterMastra(m any)
}

// MastraMemory is a stub interface for memory instances.
// STUB REASON: The real memory.MastraMemory interface has many methods (ID, AddMessages,
// GetMessages, CreateThread, etc.). This stub captures only ID() and SetLogger needed
// by Mastra registry. Cannot import memory without evaluating full dependency chain.
type MastraMemory interface {
	// ID returns the memory instance's unique identifier.
	// Matches memory.MastraMemory.ID() and memory.MastraMemoryBase.ID().
	ID() string
	SetLogger(l logger.IMastraLogger)
}

// AnyWorkflow is a type alias for the real workflows.Workflow pointer.
// WIRED: Replaces the former stub interface. The workflows package does NOT
// import mastra, so this import is cycle-free.
type AnyWorkflow = *workflows.Workflow

// WorkflowPrimitives is a type alias for workflows.Primitives.
// WIRED: Replaces the former stub struct.
type WorkflowPrimitives = workflows.Primitives

// WorkflowRunListOpts is a type alias for workflows.ListWorkflowRunsParams.
// WIRED: Replaces the former stub struct.
type WorkflowRunListOpts = workflows.ListWorkflowRunsParams

// WorkflowCreateRunOpts is a type alias for workflows.CreateRunOptions.
// WIRED: Replaces the former stub struct.
type WorkflowCreateRunOpts = workflows.CreateRunOptions

// WorkflowRuns is a type alias for storageworkflows.WorkflowRuns.
// WIRED: Replaces the former stub struct.
type WorkflowRuns = storageworkflows.WorkflowRuns

// WorkflowRunSnapshot is a type alias for storageworkflows.WorkflowRun.
// WIRED: Replaces the former stub struct.
type WorkflowRunSnapshot = storageworkflows.WorkflowRun

// WorkflowRun is a type alias for the real workflows.Run pointer.
// WIRED: Replaces the former stub interface.
type WorkflowRun = *workflows.Run

// Workspace is a stub interface for workspace instances.
// STUB REASON: The real workspace.Workspace struct has many more fields and methods
// beyond GetID/SetLogger. This stub captures the minimal contract needed by Mastra
// workspace registration. Real type has .ID field and SetLogger method.
type Workspace interface {
	// GetID returns the workspace's unique identifier.
	// Real: workspace.Workspace.ID field.
	GetID() string
	SetLogger(l logger.IMastraLogger)
}

// AnyWorkspace is an alias for Workspace for broader compatibility.
type AnyWorkspace = Workspace

// RegisteredWorkspace holds a workspace with its registration metadata.
// Corresponds to TS: RegisteredWorkspace type from workspace package.
type RegisteredWorkspace struct {
	Workspace Workspace
	Source    string // "mastra" | "agent"
	AgentID   string
	AgentName string
}

// MastraModelGateway is re-exported from the llm/model package.
// Wired: replaces local stub with real model.MastraModelGateway interface.
type MastraModelGateway = model.MastraModelGateway

// MCPServerBase is a stub interface for MCP server instances.
// STUB REASON: The real mcp.MCPServerBase struct has many more methods and fields.
// This stub captures the minimal contract needed by Mastra MCP server registry.
// Real type has ID(), SetID(), .Version/.ReleaseDate fields.
type MCPServerBase interface {
	// ID returns the server's unique identifier.
	// Matches mcp.MCPServerBase.ID().
	ID() string
	// Version returns the server version string.
	// Real: mcp.MCPServerBase.Version field.
	Version() string
	// ReleaseDate returns the server's release date string.
	// Real: mcp.MCPServerBase.ReleaseDate field.
	ReleaseDate() string
	SetID(id string)
	RegisterMastra(m any)
	SetLogger(l logger.IMastraLogger)
}

// MastraDeployer is a stub interface for deployment providers.
// STUB REASON: The deployer package is not yet ported. This minimal interface captures
// only SetLogger needed by Mastra registration.
type MastraDeployer interface {
	SetLogger(l logger.IMastraLogger)
}

// MastraServerBase is a stub interface for server adapters.
// STUB REASON: The real server.MastraServerBase struct has many more methods beyond
// SetLogger/GetApp. This stub captures the minimal contract for Mastra server registration.
type MastraServerBase interface {
	SetLogger(l logger.IMastraLogger)
	GetApp() any
}

// ServerConfig is a stub type for server configuration.
// STUB REASON: The real server/types.ServerConfig is a struct with typed fields.
// This stub uses map[string]any as a simplified placeholder.
type ServerConfig = map[string]any

// BundlerConfig is a stub type for bundler configuration.
// STUB REASON: The real bundler/types.BundlerConfig is a struct with typed fields.
// This stub uses map[string]any as a simplified placeholder.
type BundlerConfig = map[string]any

// MastraIdGenerator is re-exported from the types package.
// Ported from: packages/core/src/types/dynamic-argument.ts — MastraIdGenerator
type MastraIdGenerator = aktypes.MastraIdGenerator

// IdGeneratorContext is re-exported from the types package.
// Ported from: packages/core/src/types/dynamic-argument.ts — IdGeneratorContext
type IdGeneratorContext = aktypes.IdGeneratorContext

// IMastraEditor is a stub interface for the editor system.
// STUB REASON: The editor package is not yet ported. This minimal interface captures
// only RegisterWithMastra needed by Mastra editor registration.
type IMastraEditor interface {
	RegisterWithMastra(m any)
}

// StorageResolvedPromptBlockType is re-exported from the storage package.
// Wired: replaces local stub with real storage.StorageResolvedPromptBlockType.
type StorageResolvedPromptBlockType = storage.StorageResolvedPromptBlockType

// DatasetsManager is a stub type for the datasets manager.
// STUB REASON: Cannot import datasets due to circular dependency through core.
// The real datasets.DatasetsManager has methods for CRUD operations on datasets.
// This stub holds only a reference to the Mastra instance.
type DatasetsManager struct {
	mastra *Mastra
}

// Middleware is a stub type for server middleware.
// STUB REASON: The real server/types.Middleware is a typed function signature.
// Using `= any` as a simplified placeholder until server types are wired.
type Middleware = any

// EventHandler is a function that handles events.
type EventHandler func(event events.Event, cb func() error)

// ProcessorConfiguration tracks which agents use which processors.
type ProcessorConfiguration struct {
	Processor Processor
	AgentID   string
	Type      string // "input" | "output"
}

// ServerMiddlewareEntry holds a middleware handler and its path.
type ServerMiddlewareEntry struct {
	Handler func(c any, next func() error) error
	Path    string
}

// ---------------------------------------------------------------------------
// AvailableHooks — mirrors packages/core/src/hooks/index.ts
// ---------------------------------------------------------------------------

// AvailableHooks enumerates the available hook types.
// This is ported here because the hooks package only has mitt.go;
// the hook registry (index.ts) hasn't been ported to a separate file yet.
const (
	HookOnEvaluation = "onEvaluation"
	HookOnGeneration = "onGeneration"
	HookOnScorerRun  = "onScorerRun"
)

// globalHooks is the package-level hook emitter, mirroring TS: const hooks = mitt()
var globalHooks = hooks.New()

// RegisterHook registers a hook handler for a specific hook type.
// Corresponds to TS: export function registerHook(hook, action)
func RegisterHook(hook string, action hooks.Handler) {
	globalHooks.On(hook, action)
}

// ExecuteHook fires a hook asynchronously (non-blocking).
// Corresponds to TS: export function executeHook(hook, data) { setImmediate(() => hooks.emit(hook, data)) }
func ExecuteHook(hook string, data any) {
	go globalHooks.Emit(hook, data)
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

// Config holds all optional components for initializing a Mastra instance.
//
// Corresponds to TS: export interface Config<TAgents, TWorkflows, TVectors, TTTS, TLogger, TMCPServers, TScorers, TTools, TProcessors, TMemory>
type Config struct {
	// Agents are autonomous systems that can make decisions and take actions.
	Agents map[string]Agent

	// Storage provider for persisting data, conversation history, and workflow state.
	Storage *storage.MastraCompositeStore

	// Vectors are vector stores for semantic search and RAG.
	Vectors map[string]MastraVector

	// Logger implementation for application logging and debugging.
	// Set to nil to use the default ConsoleLogger.
	Logger logger.IMastraLogger

	// DisableLogger set to true to disable logging entirely (uses NoopLogger).
	DisableLogger bool

	// Workflows provide type-safe, composable task execution.
	Workflows map[string]AnyWorkflow

	// TTS holds text-to-speech providers.
	TTS map[string]MastraTTS

	// Observability entrypoint for tracking model interactions and tracing.
	Observability obstypes.ObservabilityEntrypoint

	// IDGenerator is a custom ID generator function.
	IDGenerator MastraIdGenerator

	// Deployer is a deployment provider.
	Deployer MastraDeployer

	// Server is server configuration for HTTP endpoints and middleware.
	Server ServerConfig

	// MCPServers provide tools and resources that agents can use.
	MCPServers map[string]MCPServerBase

	// Bundler is bundler configuration for packaging and deployment.
	Bundler BundlerConfig

	// PubSub is a pub/sub system for event-driven communication.
	PubSub events.PubSub

	// Scorers assess quality of agent responses and workflow outputs.
	Scorers map[string]MastraScorer

	// Tools are reusable functions agents can use to interact with external systems.
	Tools map[string]ToolAction

	// Processors transform inputs and outputs for agents and workflows.
	Processors map[string]Processor

	// Memory instances that can be referenced by stored agents.
	Memory map[string]MastraMemory

	// Workspace provides file storage, skills, and code execution.
	Workspace AnyWorkspace

	// Gateways are custom model router gateways for accessing LLM providers.
	Gateways map[string]MastraModelGateway

	// Events holds event handlers keyed by topic.
	Events map[string][]EventHandler

	// Editor instance for handling agent instantiation and configuration.
	Editor IMastraEditor
}

// ---------------------------------------------------------------------------
// createUndefinedPrimitiveError
// ---------------------------------------------------------------------------

// createUndefinedPrimitiveError creates an error for when a nil value is passed to an add* method.
// This commonly occurs when config was spread and the original had getters or non-enumerable properties.
//
// Corresponds to TS: function createUndefinedPrimitiveError(type, value, key?)
func createUndefinedPrimitiveError(primitiveType string, key string) *mastraerror.MastraError {
	typeLabel := primitiveType
	if primitiveType == "mcp-server" {
		typeLabel = "MCP server"
	}
	errorID := fmt.Sprintf("MASTRA_ADD_%s_UNDEFINED", strings.ToUpper(strings.ReplaceAll(primitiveType, "-", "_")))
	details := map[string]any{"status": 400}
	if key != "" {
		details["key"] = key
	}
	return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       errorID,
		Domain:   mastraerror.ErrorDomainMastra,
		Category: mastraerror.ErrorCategoryUser,
		Text:     fmt.Sprintf("Cannot add %s: %s is nil. This may occur if config was spread and the original object had getters or non-enumerable properties.", typeLabel, typeLabel),
		Details:  details,
	})
}

// ---------------------------------------------------------------------------
// Mastra
// ---------------------------------------------------------------------------

// Mastra is the central orchestrator for Mastra applications, managing agents,
// workflows, storage, logging, observability, and more.
//
// Corresponds to TS: export class Mastra<TAgents, TWorkflows, TVectors, TTTS, TLogger, TMCPServers, TScorers, TTools, TProcessors, TMemory>
type Mastra struct {
	mu sync.RWMutex

	vectors       map[string]MastraVector
	agents        map[string]Agent
	logger        logger.IMastraLogger
	workflows     map[string]AnyWorkflow
	observability obstypes.ObservabilityEntrypoint
	tts           map[string]MastraTTS
	deployer      MastraDeployer

	serverMiddleware []ServerMiddlewareEntry

	storage          *storage.MastraCompositeStore
	augmentedStorage *storage.AugmentedStore
	scorers          map[string]MastraScorer
	tools      map[string]ToolAction
	processors map[string]Processor

	processorConfigurations map[string][]ProcessorConfiguration

	memory     map[string]MastraMemory
	workspace  Workspace
	workspaces map[string]RegisteredWorkspace
	server     ServerConfig
	serverAdapter MastraServerBase
	mcpServers map[string]MCPServerBase
	bundler    BundlerConfig
	idGenerator MastraIdGenerator
	pubsub     events.PubSub
	gateways   map[string]MastraModelGateway

	events map[string][]EventHandler

	internalMastraWorkflows map[string]AnyWorkflow

	// serverCache is only used internally for server handlers that require temporary persistence.
	serverCache cache.MastraServerCache

	// storedAgentsCache allows in-memory modifications to persist across requests.
	storedAgentsCache map[string]Agent

	// storedScorersCache allows in-memory modifications to persist across requests.
	storedScorersCache map[string]MastraScorer

	// promptBlocks is a registry for prompt blocks (stored or code-defined).
	promptBlocks map[string]StorageResolvedPromptBlockType

	// editor handles agent instantiation and configuration.
	editor IMastraEditor

	// datasets is lazily initialized.
	datasets *DatasetsManager
}

// NewMastra creates a new Mastra instance with the provided configuration.
//
// Corresponds to TS: constructor(config?: Config<...>)
func NewMastra(config *Config) *Mastra {
	m := &Mastra{
		serverMiddleware:        make([]ServerMiddlewareEntry, 0),
		processorConfigurations: make(map[string][]ProcessorConfiguration),
		workspaces:              make(map[string]RegisteredWorkspace),
		events:                  make(map[string][]EventHandler),
		internalMastraWorkflows: make(map[string]AnyWorkflow),
		storedAgentsCache:       make(map[string]Agent),
		storedScorersCache:      make(map[string]MastraScorer),
		promptBlocks:            make(map[string]StorageResolvedPromptBlockType),
	}

	// Server cache
	m.serverCache = cache.NewInMemoryServerCache()

	if config == nil {
		config = &Config{}
	}

	// Editor
	m.editor = config.Editor
	if m.editor != nil {
		m.editor.RegisterWithMastra(m)
	}

	// PubSub
	if config.PubSub != nil {
		m.pubsub = config.PubSub
	} else {
		m.pubsub = events.NewEventEmitterPubSub()
	}

	// Events — normalize single handlers into slices
	for topic, handlers := range config.Events {
		m.events[topic] = handlers
	}

	// WorkflowEventProcessor — register a workflow event callback.
	// In TS: const workflowEventProcessor = new WorkflowEventProcessor({ mastra: this });
	// The full WorkflowEventProcessor from workflows/evented is not yet ported;
	// we wire up a minimal event callback that delegates to it when available.
	workflowEventCb := EventHandler(func(event events.Event, cb func() error) {
		// Placeholder: the full WorkflowEventProcessor.Process(event, cb) will be
		// wired here once the workflows/evented package is fully ported.
		// For now, invoke the callback if provided to avoid stalling event pipelines.
		if cb != nil {
			if err := cb(); err != nil {
				m.GetLogger().Error(fmt.Sprintf("Error in workflow event callback: %v", err))
			}
		}
	})
	if handlers, ok := m.events["workflows"]; ok {
		m.events["workflows"] = append(handlers, workflowEventCb)
	} else {
		m.events["workflows"] = []EventHandler{workflowEventCb}
	}

	// Logger
	var l logger.IMastraLogger
	if config.DisableLogger {
		l = logger.NoopLogger
	} else if config.Logger != nil {
		l = config.Logger
	} else {
		// Default: INFO in dev, WARN in production
		level := logger.LogLevelInfo
		env := os.Getenv("NODE_ENV")
		mastraDev := os.Getenv("MASTRA_DEV")
		if env == "production" && mastraDev != "true" {
			level = logger.LogLevelWarn
		}
		l = logger.NewConsoleLogger(&logger.ConsoleLoggerOptions{
			Name:  "Mastra",
			Level: level,
		})
	}
	m.logger = l

	// ID Generator
	m.idGenerator = config.IDGenerator

	// Storage — augment with auto-init wrapper matching TS: storage = augmentWithInit(storage)
	if config.Storage != nil {
		augmented := storage.AugmentWithInit(config.Storage)
		m.storage = augmented.MastraCompositeStore
		m.augmentedStorage = augmented
	}

	// Observability
	if config.Observability != nil {
		m.observability = config.Observability
		m.observability.SetLogger(m.logger)
	} else {
		m.observability = &observability.NoOpObservability{}
	}

	// Initialize all primitive registries (empty maps)
	m.vectors = make(map[string]MastraVector)
	m.mcpServers = make(map[string]MCPServerBase)
	m.tts = make(map[string]MastraTTS)
	m.agents = make(map[string]Agent)
	m.scorers = make(map[string]MastraScorer)
	m.tools = make(map[string]ToolAction)
	m.processors = make(map[string]Processor)
	m.memory = make(map[string]MastraMemory)
	m.workflows = make(map[string]AnyWorkflow)
	m.gateways = make(map[string]MastraModelGateway)

	// Add primitives — order matters for auto-registration.
	// Tools and processors should be added before agents and MCP servers.
	for key, tool := range config.Tools {
		if tool != nil {
			m.AddTool(tool, key)
		}
	}
	for key, proc := range config.Processors {
		if proc != nil {
			m.AddProcessor(proc, key)
		}
	}
	for key, mem := range config.Memory {
		if mem != nil {
			m.AddMemory(mem, key)
		}
	}
	for key, vec := range config.Vectors {
		if vec != nil {
			m.AddVector(vec, key)
		}
	}
	if config.Workspace != nil {
		m.workspace = config.Workspace
		m.AddWorkspace(config.Workspace, "", &WorkspaceMetadata{Source: "mastra"})
	}
	for key, scorer := range config.Scorers {
		if scorer != nil {
			m.AddScorer(scorer, key, nil)
		}
	}
	for key, wf := range config.Workflows {
		if wf != nil {
			m.AddWorkflow(wf, key)
		}
	}
	for key, gw := range config.Gateways {
		if gw != nil {
			m.AddGateway(gw, key)
		}
	}
	// MCP servers and agents last since they might reference other primitives
	for key, srv := range config.MCPServers {
		if srv != nil {
			m.AddMCPServer(srv, key)
		}
	}
	for key, agent := range config.Agents {
		if agent != nil {
			m.AddAgent(agent, key, nil)
		}
	}
	for key, tts := range config.TTS {
		if tts != nil {
			m.tts[key] = tts
		}
	}

	if config.Server != nil {
		m.server = config.Server
	}

	// Register the scorer hook
	RegisterHook(HookOnScorerRun, createOnScorerHook(m))

	// Initialize observability with Mastra context
	m.observability.SetMastraContext(m)

	// Propagate logger to all registered primitives
	m.SetLogger(l)

	// Deployer
	m.deployer = config.Deployer

	// Bundler
	m.bundler = config.Bundler

	return m
}

// ---------------------------------------------------------------------------
// PubSub
// ---------------------------------------------------------------------------

// PubSub returns the configured PubSub instance.
func (m *Mastra) PubSub() events.PubSub {
	return m.pubsub
}

// ---------------------------------------------------------------------------
// Datasets
// ---------------------------------------------------------------------------

// Datasets returns the DatasetsManager, creating it lazily.
func (m *Mastra) Datasets() *DatasetsManager {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.datasets == nil {
		m.datasets = &DatasetsManager{mastra: m}
	}
	return m.datasets
}

// ---------------------------------------------------------------------------
// ID Generation
// ---------------------------------------------------------------------------

// GetIDGenerator returns the configured ID generator function, or nil.
//
// Corresponds to TS: public getIdGenerator()
func (m *Mastra) GetIDGenerator() MastraIdGenerator {
	return m.idGenerator
}

// SetIDGenerator sets a custom ID generator function.
//
// Corresponds to TS: public setIdGenerator(idGenerator)
func (m *Mastra) SetIDGenerator(gen MastraIdGenerator) {
	m.idGenerator = gen
}

// GenerateID generates a unique identifier using the configured generator or defaults to uuid.New().
//
// Corresponds to TS: public generateId(context?)
func (m *Mastra) GenerateID(ctx *IdGeneratorContext) string {
	if m.idGenerator != nil {
		id := m.idGenerator(ctx)
		if id == "" {
			err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "MASTRA_ID_GENERATOR_RETURNED_EMPTY_STRING",
				Domain:   mastraerror.ErrorDomainMastra,
				Category: mastraerror.ErrorCategoryUser,
				Text:     "ID generator returned an empty string, which is not allowed",
			})
			m.logger.TrackException(err)
			panic(err)
		}
		return id
	}
	return uuid.New().String()
}

// ---------------------------------------------------------------------------
// Editor
// ---------------------------------------------------------------------------

// GetEditor returns the configured editor instance.
//
// Corresponds to TS: public getEditor()
func (m *Mastra) GetEditor() IMastraEditor {
	return m.editor
}

// ---------------------------------------------------------------------------
// Stored Agent / Scorer Caches
// ---------------------------------------------------------------------------

// GetStoredAgentCache returns the stored agents cache.
//
// Corresponds to TS: public getStoredAgentCache()
func (m *Mastra) GetStoredAgentCache() map[string]Agent {
	return m.storedAgentsCache
}

// GetStoredScorerCache returns the stored scorers cache.
//
// Corresponds to TS: public getStoredScorerCache()
func (m *Mastra) GetStoredScorerCache() map[string]MastraScorer {
	return m.storedScorersCache
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

// SetServer sets the server configuration.
//
// Corresponds to TS: public setServer(server)
func (m *Mastra) SetServer(server ServerConfig) {
	m.server = server
}

// GetServer returns the server configuration.
//
// Corresponds to TS: public getServer()
func (m *Mastra) GetServer() ServerConfig {
	return m.server
}

// SetMastraServer sets the server adapter.
//
// Corresponds to TS: public setMastraServer(adapter)
func (m *Mastra) SetMastraServer(adapter MastraServerBase) {
	if m.serverAdapter != nil {
		m.logger.Debug("Replacing existing server adapter. Only one adapter should be registered per Mastra instance.")
	}
	m.serverAdapter = adapter
	if m.logger != nil {
		adapter.SetLogger(m.logger)
	}
}

// GetMastraServer returns the server adapter.
//
// Corresponds to TS: public getMastraServer()
func (m *Mastra) GetMastraServer() MastraServerBase {
	return m.serverAdapter
}

// GetServerApp returns the server app from the adapter.
//
// Corresponds to TS: public getServerApp<T>()
func (m *Mastra) GetServerApp() any {
	if m.serverAdapter == nil {
		return nil
	}
	return m.serverAdapter.GetApp()
}

// GetServerMiddleware returns the configured server middleware.
//
// Corresponds to TS: public getServerMiddleware()
func (m *Mastra) GetServerMiddleware() []ServerMiddlewareEntry {
	return m.serverMiddleware
}

// SetServerMiddleware sets the server middleware with normalization.
// Each entry is ensured to have a path — defaults to "/api/*" if not set.
//
// Corresponds to TS: public setServerMiddleware(serverMiddleware)
func (m *Mastra) SetServerMiddleware(middleware []ServerMiddlewareEntry) {
	normalized := make([]ServerMiddlewareEntry, 0, len(middleware))
	for _, mw := range middleware {
		entry := mw
		if entry.Path == "" {
			entry.Path = "/api/*"
		}
		normalized = append(normalized, entry)
	}
	m.serverMiddleware = normalized
}

// GetServerCache returns the internal server cache.
//
// Corresponds to TS: public getServerCache() / public get serverCache()
func (m *Mastra) GetServerCache() cache.MastraServerCache {
	return m.serverCache
}

// ---------------------------------------------------------------------------
// Agents
// ---------------------------------------------------------------------------

// GetAgent retrieves a registered agent by its registration key.
//
// Corresponds to TS: public getAgent<TAgentName>(name)
func (m *Mastra) GetAgent(name string) (Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, ok := m.agents[name]
	if !ok {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_AGENT_BY_NAME_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Agent with name %s not found", name),
			Details: map[string]any{
				"status":    404,
				"agentName": name,
				"agents":    joinMapKeys(m.agents),
			},
		})
		m.logger.TrackException(err)
		return nil, err
	}
	return agent, nil
}

// GetAgentByID retrieves a registered agent by its unique ID.
//
// Corresponds to TS: public getAgentById(id)
func (m *Mastra) GetAgentByID(id string) (Agent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Search by internal ID
	for _, agent := range m.agents {
		if agent.ID() == id {
			return agent, nil
		}
	}

	// Fallback: try by registration key
	if agent, ok := m.agents[id]; ok {
		return agent, nil
	}

	err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "MASTRA_GET_AGENT_BY_AGENT_ID_NOT_FOUND",
		Domain:   mastraerror.ErrorDomainMastra,
		Category: mastraerror.ErrorCategoryUser,
		Text:     fmt.Sprintf("Agent with id %s not found", id),
		Details: map[string]any{
			"status":  404,
			"agentId": id,
			"agents":  joinMapKeys(m.agents),
		},
	})
	m.logger.TrackException(err)
	return nil, err
}

// ListAgents returns all registered agents.
//
// Corresponds to TS: public listAgents()
func (m *Mastra) ListAgents() map[string]Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents
}

// AddAgent adds a new agent to the Mastra instance.
//
// Corresponds to TS: public addAgent(agent, key?, options?)
func (m *Mastra) AddAgent(agent Agent, key string, options *AddPrimitiveOptions) {
	if agent == nil {
		panic(createUndefinedPrimitiveError("agent", key))
	}

	agentKey := key
	if agentKey == "" {
		agentKey = agent.ID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.agents[agentKey]; exists {
		m.logger.Debug(fmt.Sprintf("Agent with key %s already exists. Skipping addition.", agentKey))
		return
	}

	// Initialize the agent
	agent.SetLogger(m.logger)
	agent.RegisterMastra(m)
	agent.RegisterPrimitives(AgentPrimitives{
		Logger:  m.logger,
		Storage: m.storage,
		Agents:  m.agents,
		TTS:     m.tts,
		Vectors: m.vectors,
	})

	if options != nil && options.Source != "" {
		agent.SetSource(options.Source)
	}

	m.agents[agentKey] = agent

	// Register configured processor workflows from the agent asynchronously.
	// In TS this uses .then() to handle async without blocking the constructor.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Debug(fmt.Sprintf("Failed to register processor workflows for agent %s: %v", agentKey, r))
			}
		}()
		processorWorkflows := agent.GetConfiguredProcessorWorkflows()
		for _, wfAny := range processorWorkflows {
			if wf, ok := wfAny.(AnyWorkflow); ok && wf != nil {
				m.AddWorkflow(wf, wf.GetID())
			}
		}
	}()

	// Register agent workspace in the workspaces registry for direct lookup.
	// Dynamic workspace functions may return nil without request context — that's fine.
	if agent.HasOwnWorkspace() {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					m.logger.Debug(fmt.Sprintf("Failed to register workspace for agent %s: %v", agentKey, r))
				}
			}()
			wsAny := agent.GetWorkspace()
			if ws, ok := wsAny.(Workspace); ok && ws != nil {
				agentID := agent.ID()
				if agentID == "" {
					agentID = agentKey
				}
				m.AddWorkspace(ws, "", &WorkspaceMetadata{
					Source:    "agent",
					AgentID:   agentID,
					AgentName: agent.Name(),
				})
			}
		}()
	}

	// Register scorers from the agent to the Mastra instance.
	// This makes agent-level scorers discoverable via mastra.GetScorer()/GetScorerByID().
	go func() {
		defer func() {
			if r := recover(); r != nil {
				m.logger.Debug(fmt.Sprintf("Failed to register scorers from agent %s: %v", agentKey, r))
			}
		}()
		agentScorers := agent.ListScorers()
		for _, entry := range agentScorers {
			if entry != nil && entry.Scorer != nil {
				m.AddScorer(entry.Scorer, "", nil)
			}
		}
	}()
}

// RemoveAgent removes an agent by its key or ID.
//
// Corresponds to TS: public removeAgent(keyOrId)
func (m *Mastra) RemoveAgent(keyOrID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try direct key lookup
	if agent, ok := m.agents[keyOrID]; ok {
		agentID := agent.ID()
		delete(m.agents, keyOrID)
		if agentID != "" {
			delete(m.storedAgentsCache, agentID)
		}
		return true
	}

	// Try finding by ID
	for k, agent := range m.agents {
		if agent.ID() == keyOrID {
			agentID := agent.ID()
			delete(m.agents, k)
			if agentID != "" {
				delete(m.storedAgentsCache, agentID)
			}
			return true
		}
	}

	return false
}

// ---------------------------------------------------------------------------
// Vectors
// ---------------------------------------------------------------------------

// GetVector retrieves a registered vector store by its name.
//
// Corresponds to TS: public getVector(name)
func (m *Mastra) GetVector(name string) (MastraVector, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vec, ok := m.vectors[name]
	if !ok {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_VECTOR_BY_NAME_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Vector with name %s not found", name),
			Details: map[string]any{
				"status":     404,
				"vectorName": name,
				"vectors":    joinMapKeysStr(m.vectors),
			},
		})
		m.logger.TrackException(err)
		return nil, err
	}
	return vec, nil
}

// GetVectorByID retrieves a vector store by its internal ID.
//
// Corresponds to TS: public getVectorById(id)
func (m *Mastra) GetVectorByID(id string) (MastraVector, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Search by internal ID
	for _, vec := range m.vectors {
		if vec.ID() == id {
			return vec, nil
		}
	}

	// Fallback: by registration key
	if vec, ok := m.vectors[id]; ok {
		return vec, nil
	}

	err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "MASTRA_GET_VECTOR_BY_ID_NOT_FOUND",
		Domain:   mastraerror.ErrorDomainMastra,
		Category: mastraerror.ErrorCategoryUser,
		Text:     fmt.Sprintf("Vector store with id %s not found", id),
		Details: map[string]any{
			"status":   404,
			"vectorId": id,
			"vectors":  joinMapKeysStr(m.vectors),
		},
	})
	m.logger.TrackException(err)
	return nil, err
}

// ListVectors returns all registered vector stores.
//
// Corresponds to TS: public listVectors()
func (m *Mastra) ListVectors() map[string]MastraVector {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.vectors
}

// AddVector adds a new vector store.
//
// Corresponds to TS: public addVector(vector, key?)
func (m *Mastra) AddVector(vector MastraVector, key string) {
	if vector == nil {
		panic(createUndefinedPrimitiveError("vector", key))
	}
	vectorKey := key
	if vectorKey == "" {
		vectorKey = vector.ID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.vectors[vectorKey]; exists {
		m.logger.Debug(fmt.Sprintf("Vector with key %s already exists. Skipping addition.", vectorKey))
		return
	}

	vector.SetLogger(m.logger)
	m.vectors[vectorKey] = vector
}

// ---------------------------------------------------------------------------
// Deployer
// ---------------------------------------------------------------------------

// GetDeployer returns the configured deployment provider.
//
// Corresponds to TS: public getDeployer()
func (m *Mastra) GetDeployer() MastraDeployer {
	return m.deployer
}

// ---------------------------------------------------------------------------
// Workspace
// ---------------------------------------------------------------------------

// GetWorkspace returns the global workspace instance.
//
// Corresponds to TS: public getWorkspace()
func (m *Mastra) GetWorkspace() Workspace {
	return m.workspace
}

// GetWorkspaceByID retrieves a registered workspace by its ID.
//
// Corresponds to TS: public getWorkspaceById(id)
func (m *Mastra) GetWorkspaceByID(id string) (Workspace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.workspaces[id]
	if !ok {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_WORKSPACE_BY_ID_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Workspace with id %s not found", id),
			Details: map[string]any{
				"status":       404,
				"workspaceId":  id,
				"availableIds": joinRegisteredWorkspaceKeys(m.workspaces),
			},
		})
		m.logger.TrackException(err)
		return nil, err
	}
	return entry.Workspace, nil
}

// ListWorkspaces returns all registered workspaces.
//
// Corresponds to TS: public listWorkspaces()
func (m *Mastra) ListWorkspaces() map[string]RegisteredWorkspace {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]RegisteredWorkspace, len(m.workspaces))
	for k, v := range m.workspaces {
		result[k] = v
	}
	return result
}

// WorkspaceMetadata holds optional metadata for workspace registration.
type WorkspaceMetadata struct {
	Source    string // "mastra" | "agent"
	AgentID  string
	AgentName string
}

// AddWorkspace adds a new workspace.
//
// Corresponds to TS: public addWorkspace(workspace, key?, metadata?)
func (m *Mastra) AddWorkspace(workspace AnyWorkspace, key string, metadata *WorkspaceMetadata) {
	if workspace == nil {
		panic(createUndefinedPrimitiveError("workspace", key))
	}

	source := "mastra"
	if metadata != nil && metadata.Source != "" {
		source = metadata.Source
	} else if metadata != nil && (metadata.AgentID != "" || metadata.AgentName != "") {
		source = "agent"
	}

	if source == "agent" && (metadata == nil || metadata.AgentID == "" || metadata.AgentName == "") {
		panic(mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_ADD_WORKSPACE_MISSING_AGENT_METADATA",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "Agent workspaces must include agentId and agentName.",
			Details:  map[string]any{"status": 400, "workspaceId": key},
		}))
	}

	workspaceKey := key
	if workspaceKey == "" {
		workspaceKey = workspace.GetID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.workspaces[workspaceKey]; exists {
		m.logger.Debug(fmt.Sprintf("Workspace with key %s already exists. Skipping addition.", workspaceKey))
		return
	}

	entry := RegisteredWorkspace{
		Workspace: workspace,
		Source:    source,
	}
	if metadata != nil {
		entry.AgentID = metadata.AgentID
		entry.AgentName = metadata.AgentName
	}
	m.workspaces[workspaceKey] = entry
}

// ---------------------------------------------------------------------------
// Workflows
// ---------------------------------------------------------------------------

// GetWorkflow retrieves a registered workflow by its registration key.
//
// Corresponds to TS: public getWorkflow(id, { serialized }?)
func (m *Mastra) GetWorkflow(id string) (AnyWorkflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	wf, ok := m.workflows[id]
	if !ok {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_WORKFLOW_BY_ID_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Workflow with ID %s not found", id),
			Details: map[string]any{
				"status":     404,
				"workflowId": id,
				"workflows":  joinMapKeysWf(m.workflows),
			},
		})
		m.logger.TrackException(err)
		return nil, err
	}
	return wf, nil
}

// GetWorkflowByID retrieves a workflow by its internal ID.
//
// Corresponds to TS: public getWorkflowById(id)
func (m *Mastra) GetWorkflowByID(id string) (AnyWorkflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Search by internal ID
	for _, wf := range m.workflows {
		if wf.GetID() == id {
			return wf, nil
		}
	}

	// Fallback: by registration key
	if wf, ok := m.workflows[id]; ok {
		return wf, nil
	}

	err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "MASTRA_GET_WORKFLOW_BY_ID_NOT_FOUND",
		Domain:   mastraerror.ErrorDomainMastra,
		Category: mastraerror.ErrorCategoryUser,
		Text:     fmt.Sprintf("Workflow with id %s not found", id),
		Details: map[string]any{
			"status":     404,
			"workflowId": id,
			"workflows":  joinMapKeysWf(m.workflows),
		},
	})
	m.logger.TrackException(err)
	return nil, err
}

// RegisterInternalWorkflow registers an internal workflow.
//
// Corresponds to TS: __registerInternalWorkflow(workflow)
func (m *Mastra) RegisterInternalWorkflow(workflow AnyWorkflow) {
	workflow.RegisterMastra(m)
	workflow.RegisterPrimitives(WorkflowPrimitives{
		Logger: m.GetLogger(),
	})
	m.mu.Lock()
	m.internalMastraWorkflows[workflow.GetID()] = workflow
	m.mu.Unlock()
}

// HasInternalWorkflow checks if an internal workflow exists by ID.
//
// Corresponds to TS: __hasInternalWorkflow(id)
func (m *Mastra) HasInternalWorkflow(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, wf := range m.internalMastraWorkflows {
		if wf.GetID() == id {
			return true
		}
	}
	return false
}

// GetInternalWorkflow retrieves an internal workflow by ID.
//
// Corresponds to TS: __getInternalWorkflow(id)
func (m *Mastra) GetInternalWorkflow(id string) (AnyWorkflow, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, wf := range m.internalMastraWorkflows {
		if wf.GetID() == id {
			return wf, nil
		}
	}

	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "MASTRA_GET_INTERNAL_WORKFLOW_BY_ID_NOT_FOUND",
		Domain:   mastraerror.ErrorDomainMastra,
		Category: mastraerror.ErrorCategorySystem,
		Text:     fmt.Sprintf("Workflow with id %s not found", id),
		Details: map[string]any{
			"status":     404,
			"workflowId": id,
		},
	})
}

// ListActiveWorkflowRuns returns all active (running/waiting) workflow runs.
//
// Corresponds to TS: public async listActiveWorkflowRuns()
func (m *Mastra) ListActiveWorkflowRuns() (*WorkflowRuns, error) {
	if m.storage == nil {
		m.logger.Debug("Cannot get active workflow runs. Mastra storage is not initialized")
		return &WorkflowRuns{Runs: []WorkflowRunSnapshot{}, Total: 0}, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var allRuns []WorkflowRunSnapshot
	var allTotal int

	for _, wf := range m.workflows {
		if wf.GetEngineType() != "default" {
			continue
		}

		runningRuns, err := wf.ListWorkflowRuns(&WorkflowRunListOpts{Status: "running"})
		if err != nil {
			return nil, err
		}
		waitingRuns, err := wf.ListWorkflowRuns(&WorkflowRunListOpts{Status: "waiting"})
		if err != nil {
			return nil, err
		}

		allRuns = append(allRuns, runningRuns.Runs...)
		allRuns = append(allRuns, waitingRuns.Runs...)
		allTotal += runningRuns.Total + waitingRuns.Total
	}

	if allRuns == nil {
		allRuns = []WorkflowRunSnapshot{}
	}
	return &WorkflowRuns{Runs: allRuns, Total: allTotal}, nil
}

// RestartAllActiveWorkflowRuns restarts all currently active workflow runs.
//
// Corresponds to TS: public async restartAllActiveWorkflowRuns()
func (m *Mastra) RestartAllActiveWorkflowRuns() error {
	activeRuns, err := m.ListActiveWorkflowRuns()
	if err != nil {
		return err
	}

	if len(activeRuns.Runs) > 0 {
		plural := ""
		if len(activeRuns.Runs) > 1 {
			plural = "s"
		}
		m.logger.Debug(fmt.Sprintf("Restarting %d active workflow run%s", len(activeRuns.Runs), plural))
	}

	for _, snapshot := range activeRuns.Runs {
		wf, wfErr := m.GetWorkflowByID(snapshot.WorkflowName)
		if wfErr != nil {
			m.logger.Error(fmt.Sprintf("Failed to find workflow %s for restart: %v", snapshot.WorkflowName, wfErr))
			continue
		}
		run, runErr := wf.CreateRun(&WorkflowCreateRunOpts{RunID: snapshot.RunID})
		if runErr != nil {
			m.logger.Error(fmt.Sprintf("Failed to restart %s workflow run %s: %v", snapshot.WorkflowName, snapshot.RunID, runErr))
			continue
		}
		if _, restartErr := run.Restart(workflows.RestartParams{}); restartErr != nil {
			m.logger.Error(fmt.Sprintf("Failed to restart %s workflow run %s: %v", snapshot.WorkflowName, snapshot.RunID, restartErr))
			continue
		}
		m.logger.Debug(fmt.Sprintf("Restarted %s workflow run %s", snapshot.WorkflowName, snapshot.RunID))
	}
	return nil
}

// ListWorkflows returns all registered workflows.
//
// Corresponds to TS: public listWorkflows(props?)
func (m *Mastra) ListWorkflows() map[string]AnyWorkflow {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.workflows
}

// AddWorkflow adds a new workflow.
//
// Corresponds to TS: public addWorkflow(workflow, key?)
func (m *Mastra) AddWorkflow(workflow AnyWorkflow, key string) {
	if workflow == nil {
		panic(createUndefinedPrimitiveError("workflow", key))
	}
	workflowKey := key
	if workflowKey == "" {
		workflowKey = workflow.GetID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.workflows[workflowKey]; exists {
		m.logger.Debug(fmt.Sprintf("Workflow with key %s already exists. Skipping addition.", workflowKey))
		return
	}

	workflow.RegisterMastra(m)
	workflow.RegisterPrimitives(WorkflowPrimitives{
		Logger:  m.logger,
		Storage: m.storage,
	})
	if !workflow.IsCommitted() {
		workflow.Commit()
	}
	m.workflows[workflowKey] = workflow
}

// ---------------------------------------------------------------------------
// Scorers
// ---------------------------------------------------------------------------

// ListScorers returns all registered scorers.
//
// Corresponds to TS: public listScorers()
func (m *Mastra) ListScorers() map[string]MastraScorer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.scorers
}

// AddPrimitiveOptions holds optional parameters for adding primitives.
type AddPrimitiveOptions struct {
	Source string // "code" | "stored"
}

// AddScorer adds a new scorer.
//
// Corresponds to TS: public addScorer(scorer, key?, options?)
func (m *Mastra) AddScorer(scorer MastraScorer, key string, options *AddPrimitiveOptions) {
	if scorer == nil {
		panic(createUndefinedPrimitiveError("scorer", key))
	}
	scorerKey := key
	if scorerKey == "" {
		scorerKey = scorer.ID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.scorers[scorerKey]; exists {
		m.logger.Debug(fmt.Sprintf("Scorer with key %s already exists. Skipping addition.", scorerKey))
		return
	}

	scorer.RegisterMastra(m)

	if options != nil && options.Source != "" {
		scorer.SetSource(options.Source)
	}

	m.scorers[scorerKey] = scorer
}

// GetScorer retrieves a registered scorer by its key.
//
// Corresponds to TS: public getScorer(key)
func (m *Mastra) GetScorer(key string) (MastraScorer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	scorer, ok := m.scorers[key]
	if !ok {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_SCORER_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Scorer with %s not found", key),
		})
		m.logger.TrackException(err)
		return nil, err
	}
	return scorer, nil
}

// GetScorerByID retrieves a scorer by its ID or name.
// Returns nil if not found (used by hooks internally).
//
// Corresponds to TS: public getScorerById(id)
func (m *Mastra) GetScorerByID(id string) MastraScorer {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, scorer := range m.scorers {
		if scorer.ID() == id || scorer.Name() == id {
			return scorer
		}
	}
	return nil
}

// RemoveScorer removes a scorer by its key or ID.
//
// Corresponds to TS: public removeScorer(keyOrId)
func (m *Mastra) RemoveScorer(keyOrID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Direct key lookup
	if scorer, ok := m.scorers[keyOrID]; ok {
		scorerID := scorer.ID()
		delete(m.scorers, keyOrID)
		if scorerID != "" {
			delete(m.storedScorersCache, scorerID)
		}
		return true
	}

	// Search by ID or name
	for k, scorer := range m.scorers {
		if scorer.ID() == keyOrID || scorer.Name() == keyOrID {
			scorerID := scorer.ID()
			delete(m.scorers, k)
			if scorerID != "" {
				delete(m.storedScorersCache, scorerID)
			}
			return true
		}
	}

	return false
}

// ---------------------------------------------------------------------------
// Prompt Blocks
// ---------------------------------------------------------------------------

// ListPromptBlocks returns all registered prompt blocks.
//
// Corresponds to TS: public listPromptBlocks()
func (m *Mastra) ListPromptBlocks() map[string]StorageResolvedPromptBlockType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.promptBlocks
}

// AddPromptBlock registers a prompt block.
//
// Corresponds to TS: public addPromptBlock(promptBlock, key?)
func (m *Mastra) AddPromptBlock(block StorageResolvedPromptBlockType, key string) {
	blockKey := key
	if blockKey == "" {
		blockKey = block.ID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.promptBlocks[blockKey]; exists {
		m.logger.Debug(fmt.Sprintf("Prompt block with key %s already exists. Skipping addition.", blockKey))
		return
	}
	m.promptBlocks[blockKey] = block
}

// GetPromptBlock retrieves a prompt block by its key.
//
// Corresponds to TS: public getPromptBlock(key)
func (m *Mastra) GetPromptBlock(key string) (StorageResolvedPromptBlockType, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	block, ok := m.promptBlocks[key]
	if !ok {
		return StorageResolvedPromptBlockType{}, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_PROMPT_BLOCK_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Prompt block with key %s not found", key),
		})
	}
	return block, nil
}

// GetPromptBlockByID retrieves a prompt block by its ID.
//
// Corresponds to TS: public getPromptBlockById(id)
func (m *Mastra) GetPromptBlockByID(id string) (StorageResolvedPromptBlockType, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, block := range m.promptBlocks {
		if block.ID == id {
			return block, nil
		}
	}

	return StorageResolvedPromptBlockType{}, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "MASTRA_GET_PROMPT_BLOCK_BY_ID_NOT_FOUND",
		Domain:   mastraerror.ErrorDomainMastra,
		Category: mastraerror.ErrorCategoryUser,
		Text:     fmt.Sprintf("Prompt block with id %s not found", id),
	})
}

// RemovePromptBlock removes a prompt block by key or ID.
//
// Corresponds to TS: public removePromptBlock(keyOrId)
func (m *Mastra) RemovePromptBlock(keyOrID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.promptBlocks[keyOrID]; ok {
		delete(m.promptBlocks, keyOrID)
		return true
	}

	for k, block := range m.promptBlocks {
		if block.ID == keyOrID {
			delete(m.promptBlocks, k)
			return true
		}
	}

	return false
}

// ---------------------------------------------------------------------------
// Tools
// ---------------------------------------------------------------------------

// GetTool retrieves a tool by its registration key.
//
// Corresponds to TS: public getTool(name)
func (m *Mastra) GetTool(name string) (ToolAction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tool, ok := m.tools[name]
	if !ok {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_TOOL_BY_NAME_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Tool with name %s not found", name),
			Details: map[string]any{
				"status":   404,
				"toolName": name,
				"tools":    joinMapKeysTool(m.tools),
			},
		})
		m.logger.TrackException(err)
		return nil, err
	}
	return tool, nil
}

// GetToolByID retrieves a tool by its internal ID.
//
// Corresponds to TS: public getToolById(id)
func (m *Mastra) GetToolByID(id string) (ToolAction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Search by internal ID
	for _, tool := range m.tools {
		if tool.GetID() == id {
			return tool, nil
		}
	}

	// Fallback: by registration key
	if tool, ok := m.tools[id]; ok {
		return tool, nil
	}

	err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "MASTRA_GET_TOOL_BY_ID_NOT_FOUND",
		Domain:   mastraerror.ErrorDomainMastra,
		Category: mastraerror.ErrorCategoryUser,
		Text:     fmt.Sprintf("Tool with id %s not found", id),
		Details: map[string]any{
			"status": 404,
			"toolId": id,
			"tools":  joinMapKeysTool(m.tools),
		},
	})
	m.logger.TrackException(err)
	return nil, err
}

// ListTools returns all registered tools.
//
// Corresponds to TS: public listTools()
func (m *Mastra) ListTools() map[string]ToolAction {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tools
}

// AddTool adds a new tool.
//
// Corresponds to TS: public addTool(tool, key?)
func (m *Mastra) AddTool(tool ToolAction, key string) {
	if tool == nil {
		panic(createUndefinedPrimitiveError("tool", key))
	}
	toolKey := key
	if toolKey == "" {
		toolKey = tool.GetID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tools[toolKey]; exists {
		m.logger.Debug(fmt.Sprintf("Tool with key %s already exists. Skipping addition.", toolKey))
		return
	}

	m.tools[toolKey] = tool
}

// ---------------------------------------------------------------------------
// Processors
// ---------------------------------------------------------------------------

// GetProcessor retrieves a processor by its registration key.
//
// Corresponds to TS: public getProcessor(name)
func (m *Mastra) GetProcessor(name string) (Processor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	proc, ok := m.processors[name]
	if !ok {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_PROCESSOR_BY_NAME_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Processor with name %s not found", name),
			Details: map[string]any{
				"status":        404,
				"processorName": name,
				"processors":    joinMapKeysProc(m.processors),
			},
		})
		m.logger.TrackException(err)
		return nil, err
	}
	return proc, nil
}

// GetProcessorByID retrieves a processor by its internal ID.
//
// Corresponds to TS: public getProcessorById(id)
func (m *Mastra) GetProcessorByID(id string) (Processor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Search by internal ID
	for _, proc := range m.processors {
		if proc.ID() == id {
			return proc, nil
		}
	}

	// Fallback: by registration key
	if proc, ok := m.processors[id]; ok {
		return proc, nil
	}

	err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "MASTRA_GET_PROCESSOR_BY_ID_NOT_FOUND",
		Domain:   mastraerror.ErrorDomainMastra,
		Category: mastraerror.ErrorCategoryUser,
		Text:     fmt.Sprintf("Processor with id %s not found", id),
		Details: map[string]any{
			"status":      404,
			"processorId": id,
			"processors":  joinMapKeysProc(m.processors),
		},
	})
	m.logger.TrackException(err)
	return nil, err
}

// ListProcessors returns all registered processors.
//
// Corresponds to TS: public listProcessors()
func (m *Mastra) ListProcessors() map[string]Processor {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processors
}

// AddProcessor adds a new processor.
//
// Corresponds to TS: public addProcessor(processor, key?)
func (m *Mastra) AddProcessor(processor Processor, key string) {
	if processor == nil {
		panic(createUndefinedPrimitiveError("processor", key))
	}
	processorKey := key
	if processorKey == "" {
		processorKey = processor.ID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.processors[processorKey]; exists {
		m.logger.Debug(fmt.Sprintf("Processor with key %s already exists. Skipping addition.", processorKey))
		return
	}

	processor.RegisterMastra(m)
	m.processors[processorKey] = processor
}

// AddProcessorConfiguration registers a processor configuration with agent context.
//
// Corresponds to TS: public addProcessorConfiguration(processor, agentId, type)
func (m *Mastra) AddProcessorConfiguration(processor Processor, agentID string, procType string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	processorID := processor.ID()
	configs := m.processorConfigurations[processorID]

	// Check if this exact configuration already exists
	for _, c := range configs {
		if c.AgentID == agentID && c.Type == procType {
			return
		}
	}

	m.processorConfigurations[processorID] = append(configs, ProcessorConfiguration{
		Processor: processor,
		AgentID:   agentID,
		Type:      procType,
	})
}

// GetProcessorConfigurations returns all configurations for a processor ID.
//
// Corresponds to TS: public getProcessorConfigurations(processorId)
func (m *Mastra) GetProcessorConfigurations(processorID string) []ProcessorConfiguration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processorConfigurations[processorID]
}

// ListProcessorConfigurations returns all processor configurations.
//
// Corresponds to TS: public listProcessorConfigurations()
func (m *Mastra) ListProcessorConfigurations() map[string][]ProcessorConfiguration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processorConfigurations
}

// ---------------------------------------------------------------------------
// Memory
// ---------------------------------------------------------------------------

// GetMemory retrieves a memory instance by its registration key.
//
// Corresponds to TS: public getMemory(name)
func (m *Mastra) GetMemory(name string) (MastraMemory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mem, ok := m.memory[name]
	if !ok {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_MEMORY_BY_KEY_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Memory with key %s not found", name),
			Details: map[string]any{
				"status":    404,
				"memoryKey": name,
				"memory":    joinMapKeysMem(m.memory),
			},
		})
		m.logger.TrackException(err)
		return nil, err
	}
	return mem, nil
}

// GetMemoryByID retrieves a memory instance by its ID.
//
// Corresponds to TS: public getMemoryById(id)
func (m *Mastra) GetMemoryByID(id string) (MastraMemory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, mem := range m.memory {
		if mem.ID() == id {
			return mem, nil
		}
	}

	// Build available IDs for error details
	var availableIDs []string
	for _, mem := range m.memory {
		availableIDs = append(availableIDs, mem.ID())
	}

	err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "MASTRA_GET_MEMORY_BY_ID_NOT_FOUND",
		Domain:   mastraerror.ErrorDomainMastra,
		Category: mastraerror.ErrorCategoryUser,
		Text:     fmt.Sprintf("Memory with id %s not found", id),
		Details: map[string]any{
			"status":       404,
			"memoryId":     id,
			"availableIds": strings.Join(availableIDs, ", "),
		},
	})
	m.logger.TrackException(err)
	return nil, err
}

// ListMemory returns all registered memory instances.
//
// Corresponds to TS: public listMemory()
func (m *Mastra) ListMemory() map[string]MastraMemory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.memory
}

// AddMemory adds a new memory instance.
//
// Corresponds to TS: public addMemory(memory, key?)
func (m *Mastra) AddMemory(mem MastraMemory, key string) {
	if mem == nil {
		panic(createUndefinedPrimitiveError("memory", key))
	}
	memoryKey := key
	if memoryKey == "" {
		memoryKey = mem.ID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.memory[memoryKey]; exists {
		m.logger.Debug(fmt.Sprintf("Memory with key %s already exists. Skipping addition.", memoryKey))
		return
	}

	m.memory[memoryKey] = mem
}

// ---------------------------------------------------------------------------
// Storage
// ---------------------------------------------------------------------------

// SetStorage sets the storage provider.
//
// Corresponds to TS: public setStorage(storage)
func (m *Mastra) SetStorage(stg *storage.MastraCompositeStore) {
	if stg != nil {
		augmented := storage.AugmentWithInit(stg)
		m.storage = augmented.MastraCompositeStore
		m.augmentedStorage = augmented
	} else {
		m.storage = nil
		m.augmentedStorage = nil
	}
}

// GetStorage returns the configured storage provider.
//
// Corresponds to TS: public getStorage()
func (m *Mastra) GetStorage() *storage.MastraCompositeStore {
	return m.storage
}

// GetWorkflowsStore returns the workflows storage domain from the composite store.
// This satisfies the workflows.Mastra interface so that *Mastra can be passed
// directly to Workflow.RegisterMastra without stub indirection.
func (m *Mastra) GetWorkflowsStore() storageworkflows.WorkflowsStorage {
	if m.storage == nil || m.storage.Stores == nil || m.storage.Stores.Workflows == nil {
		return nil
	}
	if ws, ok := m.storage.Stores.Workflows.(storageworkflows.WorkflowsStorage); ok {
		return ws
	}
	return nil
}

// ---------------------------------------------------------------------------
// Logger
// ---------------------------------------------------------------------------

// SetLogger sets the logger and propagates it to all registered primitives.
//
// Corresponds to TS: public setLogger({ logger })
func (m *Mastra) SetLogger(l logger.IMastraLogger) {
	m.logger = l

	for _, agent := range m.agents {
		if agent != nil {
			agent.SetLogger(m.logger)
		}
	}

	if m.deployer != nil {
		m.deployer.SetLogger(m.logger)
	}

	for _, tts := range m.tts {
		if tts != nil {
			tts.SetLogger(m.logger)
		}
	}

	if m.storage != nil {
		m.storage.SetLogger(m.logger)
	}

	for _, vec := range m.vectors {
		if vec != nil {
			vec.SetLogger(m.logger)
		}
	}

	for _, srv := range m.mcpServers {
		if srv != nil {
			srv.SetLogger(m.logger)
		}
	}

	for _, wf := range m.workflows {
		if wf != nil {
			wf.SetLogger(m.logger)
		}
	}

	if m.serverAdapter != nil {
		m.serverAdapter.SetLogger(m.logger)
	}

	if m.workspace != nil {
		m.workspace.SetLogger(m.logger)
	}

	for _, mem := range m.memory {
		if mem != nil {
			mem.SetLogger(m.logger)
		}
	}

	m.observability.SetLogger(m.logger)
}

// GetLogger returns the configured logger.
//
// Corresponds to TS: public getLogger()
func (m *Mastra) GetLogger() logger.IMastraLogger {
	return m.logger
}

// ---------------------------------------------------------------------------
// Observability
// ---------------------------------------------------------------------------

// Observability returns the configured observability entrypoint.
//
// Corresponds to TS: get observability()
func (m *Mastra) Observability() obstypes.ObservabilityEntrypoint {
	return m.observability
}

// LoggerVNext returns the structured logging API for observability.
//
// Corresponds to TS: get loggerVNext()
func (m *Mastra) LoggerVNext() obstypes.LoggerContext {
	inst := m.observability.GetDefaultInstance()
	if inst != nil {
		return inst.GetLoggerContext(nil)
	}
	return observability.NoOpLoggerContext
}

// Metrics returns the direct metrics API.
//
// Corresponds to TS: get metrics()
func (m *Mastra) Metrics() obstypes.MetricsContext {
	inst := m.observability.GetDefaultInstance()
	if inst != nil {
		return inst.GetMetricsContext(nil)
	}
	return observability.NoOpMetricsContext
}

// ---------------------------------------------------------------------------
// TTS
// ---------------------------------------------------------------------------

// GetTTS returns all registered TTS providers.
//
// Corresponds to TS: public getTTS()
func (m *Mastra) GetTTS() map[string]MastraTTS {
	return m.tts
}

// ---------------------------------------------------------------------------
// Logs
// ---------------------------------------------------------------------------

// ListLogsByRunID lists logs by run ID.
//
// Corresponds to TS: public async listLogsByRunId(...)
func (m *Mastra) ListLogsByRunID(runID, transportID string, params *logger.ListLogsParams) (logger.LogResult, error) {
	if transportID == "" {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_LIST_LOGS_BY_RUN_ID_MISSING_TRANSPORT",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "Transport ID is required",
			Details: map[string]any{
				"runId":       runID,
				"transportId": transportID,
			},
		})
		m.logger.TrackException(err)
		return logger.LogResult{}, err
	}

	if m.logger == nil {
		return logger.LogResult{}, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_LOGS_BY_RUN_ID_LOGGER_NOT_CONFIGURED",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Logger is not configured or does not support listLogsByRunId operation",
			Details: map[string]any{
				"runId":       runID,
				"transportId": transportID,
			},
		})
	}

	return m.logger.ListLogsByRunID(&logger.ListLogsByRunIDFullArgs{
		TransportID: transportID,
		ListLogsByRunIDArgs: logger.ListLogsByRunIDArgs{
			RunID: runID,
		},
	})
}

// ListLogs lists logs from a transport.
//
// Corresponds to TS: public async listLogs(transportId, params?)
func (m *Mastra) ListLogs(transportID string, params *logger.ListLogsParams) (logger.LogResult, error) {
	if transportID == "" {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_LOGS_MISSING_TRANSPORT",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "Transport ID is required",
			Details: map[string]any{
				"transportId": transportID,
			},
		})
		m.logger.TrackException(err)
		return logger.LogResult{}, err
	}

	if m.logger == nil {
		return logger.LogResult{}, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_LOGS_LOGGER_NOT_CONFIGURED",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategorySystem,
			Text:     "Logger is not set",
			Details: map[string]any{
				"transportId": transportID,
			},
		})
	}

	return m.logger.ListLogs(transportID, params)
}

// ---------------------------------------------------------------------------
// MCP Servers
// ---------------------------------------------------------------------------

// ListMCPServers returns all registered MCP server instances.
//
// Corresponds to TS: public listMCPServers()
func (m *Mastra) ListMCPServers() map[string]MCPServerBase {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.mcpServers
}

// AddMCPServer adds a new MCP server.
//
// Corresponds to TS: public addMCPServer(server, key?)
func (m *Mastra) AddMCPServer(server MCPServerBase, key string) {
	if server == nil {
		panic(createUndefinedPrimitiveError("mcp-server", key))
	}

	if key != "" {
		server.SetID(key)
	}

	resolvedID := server.ID()
	if resolvedID == "" {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_ADD_MCP_SERVER_MISSING_ID",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "MCP server must expose an id or be registered under one",
			Details:  map[string]any{"status": 400},
		})
		m.logger.TrackException(err)
		panic(err)
	}

	serverKey := key
	if serverKey == "" {
		serverKey = resolvedID
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.mcpServers[serverKey]; exists {
		m.logger.Debug(fmt.Sprintf("MCP server with key %s already exists. Skipping addition.", serverKey))
		return
	}

	server.RegisterMastra(m)
	server.SetLogger(m.GetLogger())
	m.mcpServers[serverKey] = server
}

// GetMCPServer retrieves an MCP server by registration key.
//
// Corresponds to TS: public getMCPServer(name)
func (m *Mastra) GetMCPServer(name string) MCPServerBase {
	m.mu.RLock()
	defer m.mu.RUnlock()

	srv, ok := m.mcpServers[name]
	if !ok {
		m.logger.Debug(fmt.Sprintf("MCP server with name %s not found", name))
		return nil
	}
	return srv
}

// GetMCPServerByID retrieves an MCP server by its logical ID.
//
// Corresponds to TS: public getMCPServerById(serverId, version?)
func (m *Mastra) GetMCPServerByID(serverID string, version string) MCPServerBase {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matching []MCPServerBase
	for _, srv := range m.mcpServers {
		if srv.ID() == serverID {
			matching = append(matching, srv)
		}
	}

	if len(matching) == 0 {
		m.logger.Debug(fmt.Sprintf("No MCP servers found with logical ID: %s", serverID))
		return nil
	}

	if version != "" {
		for _, srv := range matching {
			if srv.Version() == version {
				return srv
			}
		}
		m.logger.Debug(fmt.Sprintf("MCP server with logical ID '%s' found, but not version '%s'.", serverID, version))
		return nil
	}

	// No version specified; return first match (or latest by releaseDate if multiple exist)
	if len(matching) == 1 {
		return matching[0]
	}

	// Sort by releaseDate descending (latest first), matching the TS implementation.
	// Servers with missing/invalid release dates are treated as older.
	sort.Slice(matching, func(i, j int) bool {
		dateA := parseReleaseDate(matching[i].ReleaseDate())
		dateB := parseReleaseDate(matching[j].ReleaseDate())
		if dateA.IsZero() && dateB.IsZero() {
			return false
		}
		if dateA.IsZero() {
			return false // invalid dates sort to the end
		}
		if dateB.IsZero() {
			return true // valid dates sort before invalid ones
		}
		return dateA.After(dateB)
	})

	// After sorting, the first element is the latest if its date is valid
	latest := matching[0]
	latestDate := parseReleaseDate(latest.ReleaseDate())
	if !latestDate.IsZero() {
		return latest
	}

	m.logger.Warn(fmt.Sprintf("Could not determine the latest server for logical ID '%s' due to invalid or missing release dates, or no servers left after filtering.", serverID))
	return nil
}

// ---------------------------------------------------------------------------
// Gateways
// ---------------------------------------------------------------------------

// GetGateway retrieves a gateway by its key.
//
// Corresponds to TS: public getGateway(key)
func (m *Mastra) GetGateway(key string) (MastraModelGateway, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	gw, ok := m.gateways[key]
	if !ok {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_GET_GATEWAY_BY_KEY_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainMastra,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Gateway with key %s not found", key),
			Details: map[string]any{
				"status":     404,
				"gatewayKey": key,
				"gateways":   joinMapKeysGw(m.gateways),
			},
		})
		m.logger.TrackException(err)
		return nil, err
	}
	return gw, nil
}

// GetGatewayByID retrieves a gateway by its ID.
//
// Corresponds to TS: public getGatewayById(id)
func (m *Mastra) GetGatewayByID(id string) (MastraModelGateway, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, gw := range m.gateways {
		if gw.ID() == id {
			return gw, nil
		}
	}

	var availableIDs []string
	for _, gw := range m.gateways {
		availableIDs = append(availableIDs, gw.ID())
	}

	err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "MASTRA_GET_GATEWAY_BY_ID_NOT_FOUND",
		Domain:   mastraerror.ErrorDomainMastra,
		Category: mastraerror.ErrorCategoryUser,
		Text:     fmt.Sprintf("Gateway with ID %s not found", id),
		Details: map[string]any{
			"status":       404,
			"gatewayId":    id,
			"availableIds": strings.Join(availableIDs, ", "),
		},
	})
	m.logger.TrackException(err)
	return nil, err
}

// ListGateways returns all registered gateways.
//
// Corresponds to TS: public listGateways()
func (m *Mastra) ListGateways() map[string]MastraModelGateway {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gateways
}

// AddGateway adds a new gateway.
//
// Corresponds to TS: public addGateway(gateway, key?)
func (m *Mastra) AddGateway(gateway MastraModelGateway, key string) {
	if gateway == nil {
		panic(createUndefinedPrimitiveError("gateway", key))
	}
	gatewayKey := key
	if gatewayKey == "" {
		gatewayKey = gateway.ID()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.gateways[gatewayKey]; exists {
		m.logger.Debug(fmt.Sprintf("Gateway with key %s already exists. Skipping addition.", gatewayKey))
		return
	}

	m.gateways[gatewayKey] = gateway

	// TODO: In TS, this calls #syncGatewayRegistry() for dev-mode type generation.
	// Skipped in Go port as this is a TS-specific dev feature.
}

// ---------------------------------------------------------------------------
// Events
// ---------------------------------------------------------------------------

// AddTopicListener subscribes a listener to a topic.
//
// Corresponds to TS: public async addTopicListener(topic, listener)
func (m *Mastra) AddTopicListener(topic string, listener events.SubscribeCallback) error {
	return m.pubsub.Subscribe(topic, listener)
}

// RemoveTopicListener unsubscribes a listener from a topic.
//
// Corresponds to TS: public async removeTopicListener(topic, listener)
func (m *Mastra) RemoveTopicListener(topic string, listener events.SubscribeCallback) error {
	return m.pubsub.Unsubscribe(topic, listener)
}

// StartEventEngine subscribes all configured event handlers.
//
// Corresponds to TS: public async startEventEngine()
func (m *Mastra) StartEventEngine() error {
	for topic, handlers := range m.events {
		for _, handler := range handlers {
			h := handler // capture for closure
			if err := m.pubsub.Subscribe(topic, func(event events.Event, ack events.AckFunc) {
				h(event, func() error {
					if ack != nil {
						return ack()
					}
					return nil
				})
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// StopEventEngine unsubscribes all configured event handlers and flushes.
//
// Corresponds to TS: public async stopEventEngine()
func (m *Mastra) StopEventEngine() error {
	// Unsubscribe all event handlers by topic
	for topic := range m.events {
		// Since Go func values aren't comparable, we use Unsubscribe which removes
		// the oldest subscription for each handler we added.
		for range m.events[topic] {
			if err := m.pubsub.Unsubscribe(topic, nil); err != nil {
				return err
			}
		}
	}
	return m.pubsub.Flush()
}

// ---------------------------------------------------------------------------
// Bundler
// ---------------------------------------------------------------------------

// GetBundlerConfig returns the bundler configuration.
//
// Corresponds to TS: public getBundlerConfig()
func (m *Mastra) GetBundlerConfig() BundlerConfig {
	return m.bundler
}

// ---------------------------------------------------------------------------
// Shutdown
// ---------------------------------------------------------------------------

// Shutdown gracefully shuts down the Mastra instance and cleans up all resources.
//
// Corresponds to TS: async shutdown()
func (m *Mastra) Shutdown() error {
	if err := m.StopEventEngine(); err != nil {
		m.logger.Error(fmt.Sprintf("Error stopping event engine during shutdown: %v", err))
	}

	if err := m.observability.Shutdown(); err != nil {
		m.logger.Error(fmt.Sprintf("Error shutting down observability: %v", err))
	}

	m.logger.Info("Mastra shutdown completed")
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// joinMapKeys joins the keys of a map[string]Agent into a comma-separated string.
func joinMapKeys(m map[string]Agent) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

func joinMapKeysStr(m map[string]MastraVector) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

func joinMapKeysWf(m map[string]AnyWorkflow) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

func joinMapKeysTool(m map[string]ToolAction) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

func joinMapKeysProc(m map[string]Processor) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

func joinMapKeysMem(m map[string]MastraMemory) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

func joinMapKeysGw(m map[string]MastraModelGateway) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

func joinRegisteredWorkspaceKeys(m map[string]RegisteredWorkspace) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

// parseReleaseDate parses a release date string into a time.Time.
// Supports RFC3339 (e.g. "2025-01-15T10:00:00Z") and date-only (e.g. "2025-01-15").
// Returns zero time if the string is empty or unparseable, matching TS new Date() → NaN.
func parseReleaseDate(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	// Try RFC3339 first (most common for ISO timestamps).
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	// Try date-only format.
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t
	}
	return time.Time{}
}
