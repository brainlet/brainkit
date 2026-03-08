// Ported from: packages/core/src/agent/agent.ts
package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/google/uuid"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	"github.com/brainlet/brainkit/agent-kit/core/action"
	"github.com/brainlet/brainkit/agent-kit/core/agent/workflows"
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/memory"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	storagepkg "github.com/brainlet/brainkit/agent-kit/core/storage"
	"github.com/brainlet/brainkit/agent-kit/core/stream"
	"github.com/brainlet/brainkit/agent-kit/core/voice"
)

// ---------------------------------------------------------------------------
// MastraLLM stub
// ---------------------------------------------------------------------------

// MastraLLM is the unified LLM interface (MastraLLMV1 | MastraLLMVNext).
// MISMATCH: real model.MastraLLMV1 is a struct with GetModel() LanguageModelV1
// (not LanguageModelLike). Real model.MastraLLMVNext is also a struct. These
// stubs are local interfaces used by llmStub; replacing with real struct types
// would require rewriting GetLLM() to return concrete types and removing llmStub.
type MastraLLM interface {
	GetModel() LanguageModelLike
}

// MastraLLMV1 is a stub for ../llm/model.MastraLLMV1.
// MISMATCH: see MastraLLM comment. Real type is a struct, not interface.
type MastraLLMV1 interface {
	MastraLLM
}

// MastraLLMVNext is a stub for ../llm/model/model.loop.MastraLLMVNext.
// MISMATCH: see MastraLLM comment. Real type is a struct, not interface.
type MastraLLMVNext interface {
	MastraLLM
}

// CoreTool is a stub for ../tools/types.CoreTool.
// MISMATCH: real tools.CoreTool is a struct with Execute, Parameters, Description,
// etc. This stub is = any because agent code uses CoreTool as a generic container
// (map[string]CoreTool) with arbitrary values assigned. Cannot wire until all
// tool conversion code (makeCoreTool, listAssignedTools, etc.) is updated to
// construct real tools.CoreTool structs.
type CoreTool = any

// MastraPrimitives is re-exported from action.
type MastraPrimitives = action.MastraPrimitives

// MastraAgentNetworkStream is re-exported from stream.
type MastraAgentNetworkStream = *stream.MastraAgentNetworkStream

// ---------------------------------------------------------------------------
// ModelFallbacks
// ---------------------------------------------------------------------------

// ModelFallback represents a single model configuration with retry and enabled settings.
type ModelFallback struct {
	ID         string         `json:"id"`
	Model      DynamicArgument `json:"model"`
	MaxRetries int            `json:"maxRetries"`
	Enabled    bool           `json:"enabled"`
}

// ModelFallbacks is a slice of model fallback configurations.
type ModelFallbacks = []ModelFallback

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

// Agent is the foundation for creating AI agents in Mastra.
// It provides methods for generating responses, streaming interactions,
// managing memory, and handling voice capabilities.
//
// In TypeScript this was generic over TAgentId, TTools, TOutput, TRequestContext.
// In Go we collapse all generics to concrete types or any.
type Agent struct {
	*agentkit.MastraBase

	// ID uniquely identifies the agent.
	ID string `json:"id"`
	// Name is the display name for the agent.
	AgentName string `json:"name"`
	// Source indicates whether the agent was created from code or storage.
	Source string `json:"source,omitempty"` // "code" | "stored"

	// Model is the language model used by the agent.
	// Can be a MastraModelConfig, DynamicModel func, or ModelFallbacks.
	Model any `json:"model"`

	// MaxRetries for model calls in case of failure. Default: 0.
	MaxRetries int `json:"maxRetries,omitempty"`

	// _agentNetworkAppend is a flag for agent network messages.
	agentNetworkAppend bool

	// Private fields (TypeScript used # prefix for these).
	instructions                DynamicArgument
	description                 string
	options                     *AgentCreateOptions
	originalModel               any
	mastra                      Mastra
	memory                      DynamicArgument
	skillsFormat                SkillFormat
	workflows                   DynamicArgument
	defaultGenerateOptionsLegacy DynamicArgument
	defaultStreamOptionsLegacy  DynamicArgument
	defaultOptions              DynamicArgument
	defaultNetworkOptions       DynamicArgument
	tools                       DynamicArgument
	scorers                     DynamicArgument
	agents                      DynamicArgument
	voice                       MastraVoice
	workspace                   DynamicArgument
	inputProcessors             DynamicArgument
	outputProcessors            DynamicArgument
	maxProcessorRetries         *int
	requestContextSchema        any
	legacyHandler               *AgentLegacyHandler
	primitives                  *MastraPrimitives

	mu sync.RWMutex
}

// NewAgent creates a new Agent instance with the specified configuration.
func NewAgent(config AgentConfig) (*Agent, error) {
	id := config.ID
	if id == "" {
		id = config.Name
	}

	base := agentkit.NewMastraBase(agentkit.MastraBaseOptions{
		Component: logger.RegisteredLoggerAgent,
		Name:      config.Name,
		RawConfig: config.RawConfig,
	})

	if config.Model == nil {
		err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_CONSTRUCTOR_MODEL_REQUIRED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Details:  map[string]any{"agentName": config.Name},
			Text:     "LanguageModel is required to create an Agent. Please provide the 'model'.",
		})
		base.Logger().Error(err.Error())
		return nil, err
	}

	a := &Agent{
		MastraBase: base,
		ID:         id,
		AgentName:  config.Name,
		Source:     "code",

		instructions:                config.Instructions,
		description:                 config.Description,
		options:                     config.Options,
		maxProcessorRetries:         config.MaxProcessorRetries,
		requestContextSchema:        config.RequestContextSchema,
		defaultGenerateOptionsLegacy: config.DefaultGenerateOptionsLegacy,
		defaultStreamOptionsLegacy:  config.DefaultStreamOptionsLegacy,
		defaultOptions:              config.DefaultOptions,
		defaultNetworkOptions:       config.DefaultNetworkOptions,
		tools:                       config.Tools,
		scorers:                     config.Scorers,
		agents:                      config.Agents,
		memory:                      config.Memory,
		workspace:                   config.Workspace,
		inputProcessors:             config.InputProcessors,
		outputProcessors:            config.OutputProcessors,
	}

	// Handle model configuration.
	if models, ok := config.Model.([]ModelWithRetries); ok {
		if len(models) == 0 {
			err := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_CONSTRUCTOR_MODEL_ARRAY_EMPTY",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details:  map[string]any{"agentName": config.Name},
				Text:     "Model array is empty. Please provide at least one model.",
			})
			base.Logger().Error(err.Error())
			return nil, err
		}

		fallbacks := make(ModelFallbacks, len(models))
		for i, mdl := range models {
			maxRetries := 0
			if mdl.MaxRetries != nil {
				maxRetries = *mdl.MaxRetries
			} else if config.MaxRetries != nil {
				maxRetries = *config.MaxRetries
			}

			enabled := true
			if mdl.Enabled != nil {
				enabled = *mdl.Enabled
			}

			fallbacks[i] = ModelFallback{
				ID:         uuid.New().String(),
				Model:      mdl.Model,
				MaxRetries: maxRetries,
				Enabled:    enabled,
			}
		}
		a.Model = fallbacks
		// Clone for original.
		orig := make(ModelFallbacks, len(fallbacks))
		copy(orig, fallbacks)
		a.originalModel = orig
	} else {
		a.Model = config.Model
		a.originalModel = config.Model
	}

	if config.MaxRetries != nil {
		a.MaxRetries = *config.MaxRetries
	}

	if config.Workflows != nil {
		a.workflows = config.Workflows
	}

	if config.SkillsFormat != "" {
		a.skillsFormat = config.SkillsFormat
	}

	if config.Voice != nil {
		a.voice = config.Voice
	}

	// Register Mastra if provided.
	if config.Mastra != nil {
		a.RegisterMastra(config.Mastra)
	}

	return a, nil
}

// ---------------------------------------------------------------------------
// Public accessor methods
// ---------------------------------------------------------------------------

// GetMastraInstance returns the Mastra instance, if registered.
func (a *Agent) GetMastraInstance() Mastra {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.mastra
}

// HasOwnMemory returns whether this agent has its own memory configured.
func (a *Agent) HasOwnMemory() bool {
	return a.memory != nil
}

// HasOwnWorkspace returns whether this agent has its own workspace configured.
func (a *Agent) HasOwnWorkspace() bool {
	return a.workspace != nil
}

// GetDescription returns the agent's description.
func (a *Agent) GetDescription() string {
	return a.description
}

// GetInstructions returns the agent's instructions, resolving function-based
// instructions if necessary.
func (a *Agent) GetInstructions(ctx context.Context, reqCtx *requestcontext.RequestContext) (any, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	if fn, ok := a.instructions.(func(*requestcontext.RequestContext, Mastra) (any, error)); ok {
		result, err := fn(reqCtx, a.mastra)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_GET_INSTRUCTIONS_FUNCTION_EMPTY_RETURN",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details:  map[string]any{"agentName": a.AgentName},
				Text:     "Instructions are required to use an Agent. The function-based instructions returned an empty value.",
			})
		}
		return result, nil
	}

	return a.instructions, nil
}

// ConvertInstructionsToString converts agent instructions to a string.
func (a *Agent) ConvertInstructionsToString(instructions any) string {
	switch inst := instructions.(type) {
	case string:
		return inst
	case []any:
		var parts []string
		for _, msg := range inst {
			switch m := msg.(type) {
			case string:
				if m != "" {
					parts = append(parts, m)
				}
			case map[string]any:
				if content, ok := m["content"].(string); ok && content != "" {
					parts = append(parts, content)
				}
			}
		}
		return strings.Join(parts, "\n\n")
	case map[string]any:
		if content, ok := inst["content"].(string); ok {
			return content
		}
	}
	return ""
}

// ListTools returns the tools configured for this agent.
func (a *Agent) ListTools(reqCtx *requestcontext.RequestContext) (ToolsInput, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	if fn, ok := a.tools.(func(*requestcontext.RequestContext, Mastra) (ToolsInput, error)); ok {
		tools, err := fn(reqCtx, a.mastra)
		if err != nil {
			return nil, err
		}
		if tools == nil {
			return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_GET_TOOLS_FUNCTION_EMPTY_RETURN",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details:  map[string]any{"agentName": a.AgentName},
				Text:     fmt.Sprintf("[Agent:%s] - Function-based tools returned empty value", a.AgentName),
			})
		}
		return tools, nil
	}

	if tools, ok := a.tools.(ToolsInput); ok {
		return tools, nil
	}
	return ToolsInput{}, nil
}

// ListAgents returns the agents configured for this agent.
func (a *Agent) ListAgents(reqCtx *requestcontext.RequestContext) (map[string]*Agent, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	if fn, ok := a.agents.(func(*requestcontext.RequestContext) (map[string]*Agent, error)); ok {
		agents, err := fn(reqCtx)
		if err != nil {
			return nil, err
		}
		if agents == nil {
			return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_GET_AGENTS_FUNCTION_EMPTY_RETURN",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details:  map[string]any{"agentName": a.AgentName},
				Text:     fmt.Sprintf("[Agent:%s] - Function-based agents returned empty value", a.AgentName),
			})
		}
		// Register mastra on sub-agents.
		for _, ag := range agents {
			if a.mastra != nil {
				ag.RegisterMastra(a.mastra)
			}
		}
		return agents, nil
	}

	if agents, ok := a.agents.(map[string]*Agent); ok {
		for _, ag := range agents {
			if a.mastra != nil {
				ag.RegisterMastra(a.mastra)
			}
		}
		return agents, nil
	}
	return map[string]*Agent{}, nil
}

// ListWorkflows returns the workflows configured for this agent.
func (a *Agent) ListWorkflows(reqCtx *requestcontext.RequestContext) (map[string]Workflow, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	if fn, ok := a.workflows.(func(*requestcontext.RequestContext, Mastra) (map[string]Workflow, error)); ok {
		wfs, err := fn(reqCtx, a.mastra)
		if err != nil {
			return nil, err
		}
		return wfs, nil
	}

	if wfs, ok := a.workflows.(map[string]Workflow); ok {
		return wfs, nil
	}
	return map[string]Workflow{}, nil
}

// ListScorers returns the scorers configured for this agent.
func (a *Agent) ListScorers(reqCtx *requestcontext.RequestContext) ([]MastraScorer, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	if fn, ok := a.scorers.(func(*requestcontext.RequestContext, Mastra) ([]MastraScorer, error)); ok {
		scorers, err := fn(reqCtx, a.mastra)
		if err != nil {
			return nil, err
		}
		if scorers == nil {
			return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_GET_SCORERS_FUNCTION_EMPTY_RETURN",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details:  map[string]any{"agentName": a.AgentName},
				Text:     fmt.Sprintf("[Agent:%s] - Function-based scorers returned empty value", a.AgentName),
			})
		}
		return scorers, nil
	}

	if scorers, ok := a.scorers.([]MastraScorer); ok {
		return scorers, nil
	}
	return nil, nil
}

// GetMemory returns the memory instance for this agent, resolving function-based
// memory if necessary. Also handles Mastra propagation and storage fallback.
// Ported from TS: getMemory({ requestContext })
//
// After resolving the memory instance, this method:
// 1. Registers Mastra with the memory (for cross-cutting concerns).
// 2. If memory doesn't have its own storage, falls back to Mastra's global storage.
func (a *Agent) GetMemory(reqCtx *requestcontext.RequestContext) (MastraMemory, error) {
	if a.memory == nil {
		return nil, nil
	}
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	var resolvedMemory MastraMemory

	if fn, ok := a.memory.(func(*requestcontext.RequestContext, Mastra) (MastraMemory, error)); ok {
		mem, err := fn(reqCtx, a.mastra)
		if err != nil {
			return nil, err
		}
		if mem == nil {
			return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_GET_MEMORY_FUNCTION_EMPTY_RETURN",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details:  map[string]any{"agentName": a.AgentName},
				Text:     fmt.Sprintf("[Agent:%s] - Function-based memory returned empty value", a.AgentName),
			})
		}
		resolvedMemory = mem
	} else if mem, ok := a.memory.(MastraMemory); ok {
		resolvedMemory = mem
	}

	if resolvedMemory == nil {
		return nil, nil
	}

	// Propagate Mastra to memory and set up storage fallback.
	// In TS:
	//   resolvedMemory.__registerMastra(this.#mastra)
	//   if (!resolvedMemory.hasOwnStorage) {
	//     const storage = this.#mastra.getStorage();
	//     if (storage) resolvedMemory.setStorage(storage);
	//   }
	if a.mastra != nil {
		// RegisterMastra expects memory.Mastra (interface with GenerateID).
		// a.mastra is agent.Mastra (empty interface{}). Type-assert to memory.Mastra.
		if mm, ok := a.mastra.(memory.Mastra); ok {
			resolvedMemory.RegisterMastra(mm)
		}

		if !resolvedMemory.HasOwnStorage() {
			// Try to get storage from the Mastra instance.
			type storageProvider interface {
				GetStorage() any
			}
			if sp, ok := a.mastra.(storageProvider); ok {
				if storage := sp.GetStorage(); storage != nil {
					// SetStorage on the memory expects a concrete storage type.
					// Pass through via the MastraMemory SetStorage method if it exists.
					if cs, ok := storage.(*storagepkg.MastraCompositeStore); ok {
						resolvedMemory.SetStorage(cs)
					}
				}
			}
		}
	}

	return resolvedMemory, nil
}

// GetWorkspace returns the workspace instance for this agent.
// Ported from TS: getWorkspace({ requestContext })
//
// Resolution order:
// 1. If agent has its own workspace configured (static or function-based), use it.
// 2. If workspace is a function, resolve it and propagate logger.
// 3. If no agent workspace, fall back to Mastra's global workspace.
func (a *Agent) GetWorkspace(reqCtx *requestcontext.RequestContext) (AnyWorkspace, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// If agent has its own workspace configured, use it.
	if a.workspace != nil {
		// Static workspace (not a function).
		if fn, ok := a.workspace.(func(*requestcontext.RequestContext, Mastra) (AnyWorkspace, error)); ok {
			resolvedWorkspace, err := fn(reqCtx, a.mastra)
			if err != nil {
				return nil, err
			}
			if resolvedWorkspace == nil {
				return nil, nil
			}

			// Propagate logger to factory-resolved workspace.
			// TODO: Call resolvedWorkspace.__setLogger(a.Logger()) once workspace has __setLogger.

			// Auto-register dynamically created workspace with Mastra.
			// TODO: Call mastra.AddWorkspace(resolvedWorkspace, ...) once Mastra has AddWorkspace.

			return resolvedWorkspace, nil
		}

		if ws, ok := a.workspace.(AnyWorkspace); ok {
			return ws, nil
		}

		return nil, nil
	}

	// Fall back to Mastra's global workspace.
	// In TS: return this.#mastra?.getWorkspace()
	if a.mastra != nil {
		type workspaceGetter interface {
			GetWorkspace() any
		}
		if wg, ok := a.mastra.(workspaceGetter); ok {
			ws := wg.GetWorkspace()
			if ws == nil {
				return nil, nil
			}
			if anyWs, ok := ws.(AnyWorkspace); ok {
				return anyWs, nil
			}
		}
	}
	return nil, nil
}

// GetDefaultOptions returns the default options for this agent.
func (a *Agent) GetDefaultOptions(reqCtx *requestcontext.RequestContext) (AgentExecutionOptions, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	if fn, ok := a.defaultOptions.(func(*requestcontext.RequestContext, Mastra) (AgentExecutionOptions, error)); ok {
		opts, err := fn(reqCtx, a.mastra)
		if err != nil {
			return AgentExecutionOptions{}, err
		}
		return opts, nil
	}

	if opts, ok := a.defaultOptions.(AgentExecutionOptions); ok {
		return opts, nil
	}
	return AgentExecutionOptions{}, nil
}

// GetDefaultNetworkOptions returns the default network options for this agent.
func (a *Agent) GetDefaultNetworkOptions(reqCtx *requestcontext.RequestContext) (NetworkOptions, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	if fn, ok := a.defaultNetworkOptions.(func(*requestcontext.RequestContext, Mastra) (NetworkOptions, error)); ok {
		opts, err := fn(reqCtx, a.mastra)
		if err != nil {
			return NetworkOptions{}, err
		}
		return opts, nil
	}

	if opts, ok := a.defaultNetworkOptions.(NetworkOptions); ok {
		return opts, nil
	}
	return NetworkOptions{}, nil
}

// GetDefaultGenerateOptionsLegacy returns the default generate options for legacy mode.
func (a *Agent) GetDefaultGenerateOptionsLegacy(reqCtx *requestcontext.RequestContext) (AgentGenerateOptions, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	if fn, ok := a.defaultGenerateOptionsLegacy.(func(*requestcontext.RequestContext, Mastra) (AgentGenerateOptions, error)); ok {
		opts, err := fn(reqCtx, a.mastra)
		if err != nil {
			return AgentGenerateOptions{}, err
		}
		return opts, nil
	}

	if opts, ok := a.defaultGenerateOptionsLegacy.(AgentGenerateOptions); ok {
		return opts, nil
	}
	return AgentGenerateOptions{}, nil
}

// GetDefaultStreamOptionsLegacy returns the default stream options for legacy mode.
func (a *Agent) GetDefaultStreamOptionsLegacy(reqCtx *requestcontext.RequestContext) (AgentStreamOptions, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	if fn, ok := a.defaultStreamOptionsLegacy.(func(*requestcontext.RequestContext, Mastra) (AgentStreamOptions, error)); ok {
		opts, err := fn(reqCtx, a.mastra)
		if err != nil {
			return AgentStreamOptions{}, err
		}
		return opts, nil
	}

	if opts, ok := a.defaultStreamOptionsLegacy.(AgentStreamOptions); ok {
		return opts, nil
	}
	return AgentStreamOptions{}, nil
}

// GetLLM returns an LLM instance based on the provided or configured model.
// Ported from TS: getLLM({ requestContext, model })
//
// If model is provided, resolves it; otherwise uses the agent's model.
// For supported vNext models (v2/v3), creates a MastraLLMVNext.
// For v1 models, creates a MastraLLMV1.
// Applies stored primitives and mastra registration to the resolved LLM.
//
// TODO: Complete implementation once llm/model package is ported.
// Currently returns the resolved model wrapped in a minimal LLM stub.
func (a *Agent) GetLLM(reqCtx *requestcontext.RequestContext, model DynamicArgument) (MastraLLM, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// Resolve the model config to a LanguageModel instance.
	resolvedModel, err := a.GetModel(reqCtx, model)
	if err != nil {
		return nil, err
	}

	// Check if the resolved model satisfies LanguageModelLike for version checking.
	langModel, ok := resolvedModel.(LanguageModelLike)
	if !ok {
		// If the model doesn't implement LanguageModelLike, we can't determine
		// its version. Return a generic error.
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_GET_LLM_INVALID_MODEL",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Details:  map[string]any{"agentName": a.AgentName},
			Text:     fmt.Sprintf("[Agent:%s] - Resolved model does not implement LanguageModelLike", a.AgentName),
		})
	}

	if IsSupportedLanguageModel(langModel) {
		// vNext path: use prepareModels to resolve all model configurations,
		// then create MastraLLMVNext with the prepared models.
		models, err := a.prepareModels(reqCtx, model)
		if err != nil {
			return nil, err
		}

		// TODO: Construct real MastraLLMVNext once llm/model/model.loop package is ported.
		// In TS: const llm = new MastraLLMVNext({ models, ... })
		//   llm.__registerPrimitives(this.#primitives)
		//   if (this.#mastra) llm.__registerMastra(this.#mastra)
		_ = models
		return &llmStub{model: langModel}, nil
	}

	// v1 (legacy) path: create MastraLLMV1.
	// TODO: Implement MastraLLMV1 construction once llm/model package is ported.
	return &llmStub{model: langModel}, nil
}

// llmStub is a minimal MastraLLM implementation wrapping a resolved model.
// It also satisfies workflows.MastraLLMVNext so it can be used in the
// prepare-stream workflow.
// TODO: Replace with real MastraLLMV1/MastraLLMVNext once llm/model is ported.
type llmStub struct {
	model LanguageModelLike
}

func (l *llmStub) GetModel() LanguageModelLike {
	return l.model
}

// Stream satisfies the workflows.MastraLLMVNext interface.
// TODO: Implement actual streaming once llm/model package is ported.
func (l *llmStub) Stream(args any) any {
	return nil
}

// GetModel returns the model instance, resolving it if it's a function or model configuration.
// Ported from TS: getModel({ requestContext, modelConfig })
//
// When modelConfig is not an array, resolves it directly via resolveModelConfig.
// When the agent has multiple models (ModelFallbacks), returns the first enabled model.
// If modelConfig is nil, uses the agent's own Model field.
func (a *Agent) GetModel(reqCtx *requestcontext.RequestContext, modelConfig any) (any, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// Use agent's model if no explicit config provided.
	configToUse := modelConfig
	if configToUse == nil {
		configToUse = a.Model
	}

	// Check if the config is a ModelFallbacks (array of models).
	if fallbacks, ok := configToUse.(ModelFallbacks); ok {
		if len(fallbacks) == 0 || fallbacks[0].Model == nil {
			return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_GET_MODEL_MISSING_MODEL_INSTANCE",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details:  map[string]any{"agentName": a.AgentName},
				Text:     fmt.Sprintf("[Agent:%s] - Empty model list provided", a.AgentName),
			})
		}
		// Return the first model's resolved config.
		return a.resolveModelConfig(fallbacks[0].Model, reqCtx)
	}

	// Single model config - resolve directly.
	return a.resolveModelConfig(configToUse, reqCtx)
}

// resolveModelConfig resolves a model configuration to a model instance.
// Ported from TS: resolveModelConfig(modelConfig, requestContext)
//
// Handles:
//   - Function-based configs (dynamic resolution)
//   - Objects that are already LanguageModelLike instances (pass-through)
//   - String-based magic model IDs (e.g., "openai/gpt-5")
//
// TODO: Implement full resolution via the llm package's resolveModelConfig
// once that package is ported. Currently supports pass-through and function-based configs.
func (a *Agent) resolveModelConfig(modelConfig any, reqCtx *requestcontext.RequestContext) (any, error) {
	if modelConfig == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_GET_MODEL_MISSING_MODEL_INSTANCE",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Details:  map[string]any{"agentName": a.AgentName},
			Text:     fmt.Sprintf("[Agent:%s] - No model configuration provided", a.AgentName),
		})
	}

	// If it's already a LanguageModelLike, return it directly.
	if _, ok := modelConfig.(LanguageModelLike); ok {
		return modelConfig, nil
	}

	// If it's a dynamic function, resolve it.
	if fn, ok := modelConfig.(func(*requestcontext.RequestContext, Mastra) (any, error)); ok {
		result, err := fn(reqCtx, a.mastra)
		if err != nil {
			return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_GET_MODEL_MISSING_MODEL_INSTANCE",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details: map[string]any{
					"agentName":     a.AgentName,
					"originalError": err.Error(),
				},
				Text: fmt.Sprintf("[Agent:%s] - Failed to resolve model configuration", a.AgentName),
			})
		}
		return result, nil
	}

	// TODO: Handle string-based model IDs (e.g., "openai/gpt-5") via the
	// llm package's resolveModelConfig once ported.
	// For now, return the config as-is and let downstream consumers validate.
	return modelConfig, nil
}

// GetModelList returns the list of configured models if the agent has multiple models.
func (a *Agent) GetModelList() []ModelFallback {
	if fallbacks, ok := a.Model.(ModelFallbacks); ok {
		return fallbacks
	}
	return nil
}

// prepareModels resolves and prepares model configurations for the LLM.
// Ported from TS: prepareModels(requestContext, model?)
//
// When a model override is provided (or the agent has a single model), wraps it
// in a single-element AgentModelManagerConfig slice.
// When the agent has multiple models (ModelFallbacks), resolves each one and
// validates that all are vNext-compatible (v2/v3).
func (a *Agent) prepareModels(reqCtx *requestcontext.RequestContext, model any) ([]AgentModelManagerConfig, error) {
	if model != nil || !isModelFallbacksSlice(a.Model) {
		// Single model path: resolve and wrap.
		modelToUse := model
		if modelToUse == nil {
			modelToUse = a.Model
		}
		resolvedModel, err := a.resolveModelConfig(modelToUse, reqCtx)
		if err != nil {
			return nil, err
		}

		// Validate it's a vNext model.
		langModel, ok := resolvedModel.(LanguageModelLike)
		if !ok || !IsSupportedLanguageModel(langModel) {
			return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_PREPARE_MODELS_INCOMPATIBLE_WITH_MODEL_ARRAY_V1",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details:  map[string]any{"agentName": a.AgentName},
				Text:     fmt.Sprintf("[Agent:%s] - Only v2/v3 models are allowed when an array of models is provided", a.AgentName),
			})
		}

		// Extract headers from ModelRouterLanguageModel if available.
		// TODO: Check for ModelRouterLanguageModel interface once llm/model/router is ported.
		var headers map[string]string
		if hdr, ok := resolvedModel.(interface{ GetHeaders() map[string]string }); ok {
			headers = hdr.GetHeaders()
		}

		maxRetries := a.MaxRetries
		return []AgentModelManagerConfig{
			{
				ModelManagerModelConfig: ModelManagerModelConfig{
					ID:         "main",
					Model:      resolvedModel,
					MaxRetries: maxRetries,
					Headers:    headers,
				},
				Enabled: true,
			},
		}, nil
	}

	// Multiple models path: resolve each from the fallbacks array.
	fallbacks, _ := a.Model.(ModelFallbacks)
	configs := make([]AgentModelManagerConfig, 0, len(fallbacks))

	for _, modelConfig := range fallbacks {
		resolvedModel, err := a.resolveModelConfig(modelConfig.Model, reqCtx)
		if err != nil {
			return nil, err
		}

		// Validate it's a vNext model.
		langModel, ok := resolvedModel.(LanguageModelLike)
		if !ok || !IsSupportedLanguageModel(langModel) {
			return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_PREPARE_MODELS_INCOMPATIBLE_WITH_MODEL_ARRAY_V1",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details:  map[string]any{"agentName": a.AgentName},
				Text:     fmt.Sprintf("[Agent:%s] - Only v2/v3 models are allowed when an array of models is provided", a.AgentName),
			})
		}

		// Determine model ID.
		modelID := modelConfig.ID
		if modelID == "" {
			if mid, ok := resolvedModel.(interface{ ModelID() string }); ok {
				modelID = mid.ModelID()
			}
		}
		if modelID == "" {
			return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_PREPARE_MODELS_MISSING_MODEL_ID",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details:  map[string]any{"agentName": a.AgentName},
				Text:     fmt.Sprintf("[Agent:%s] - Unable to determine model ID. Please provide an explicit ID in the model configuration.", a.AgentName),
			})
		}

		// Extract headers from ModelRouterLanguageModel if available.
		var headers map[string]string
		if hdr, ok := resolvedModel.(interface{ GetHeaders() map[string]string }); ok {
			headers = hdr.GetHeaders()
		}

		configs = append(configs, AgentModelManagerConfig{
			ModelManagerModelConfig: ModelManagerModelConfig{
				ID:         modelID,
				Model:      resolvedModel,
				MaxRetries: modelConfig.MaxRetries,
				Headers:    headers,
			},
			Enabled: modelConfig.Enabled,
		})
	}

	return configs, nil
}

// isModelFallbacksSlice checks if the value is a ModelFallbacks slice.
func isModelFallbacksSlice(model any) bool {
	_, ok := model.(ModelFallbacks)
	return ok
}

// GetMostRecentUserMessage returns the most recent user message from a slice.
func (a *Agent) GetMostRecentUserMessage(messages []MastraDBMessage) *MastraDBMessage {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			msg := messages[i]
			return &msg
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Generate / Stream / Network (stubs)
// ---------------------------------------------------------------------------

// Generate generates a response from the agent.
// Ported from TS: generate(messages, options)
//
// 1. Validates the request context against the agent's requestContextSchema.
// 2. Merges default options with call-specific options.
// 3. Resolves the LLM and validates the model specification version.
// 4. Delegates to #execute with methodType="generate".
// 5. Returns the full output or throws on failure.
func (a *Agent) Generate(ctx context.Context, messages MessageListInput, opts AgentExecutionOptions) (*FullOutput, error) {
	// Validate request context if schema is provided.
	if err := a.validateRequestContext(opts.RequestContext); err != nil {
		return nil, err
	}

	// Merge default options with call-specific options.
	defaultOpts, err := a.GetDefaultOptions(opts.RequestContext)
	if err != nil {
		return nil, err
	}
	mergedOpts := mergeExecutionOptions(defaultOpts, opts)

	// Resolve the LLM.
	llm, err := a.GetLLM(mergedOpts.RequestContext, nil)
	if err != nil {
		return nil, err
	}

	// Validate model specification version.
	modelInfo := llm.GetModel()
	if !IsSupportedLanguageModel(modelInfo) {
		specVersion := modelInfo.SpecificationVersion()
		var text string
		if specVersion == "v1" {
			text = fmt.Sprintf(
				"Agent %q is using AI SDK v4 model which is not compatible with Generate(). "+
					"Please use AI SDK v5+ models or call GenerateLegacy() instead.",
				a.AgentName,
			)
		} else {
			text = fmt.Sprintf(
				"Agent %q has a model with unrecognized specificationVersion %q. "+
					"Supported versions: v1 (legacy), v2 (AI SDK v5), v3 (AI SDK v6).",
				a.AgentName, specVersion,
			)
		}
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_GENERATE_V1_MODEL_NOT_SUPPORTED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Details: map[string]any{
				"agentName":            a.AgentName,
				"specificationVersion": specVersion,
			},
			Text: text,
		})
	}

	// Build inner execution options.
	innerOpts := InnerAgentExecutionOptions{
		AgentExecutionOptionsBase: mergedOpts.AgentExecutionOptionsBase,
		StructuredOutput:          mergedOpts.StructuredOutput,
		Messages:                  messages,
		MethodType:                AgentMethodTypeGenerate,
	}

	// Use agent's maxProcessorRetries as default, allow options to override.
	if innerOpts.MaxProcessorRetries == nil {
		innerOpts.MaxProcessorRetries = a.maxProcessorRetries
	}

	// Delegate to #execute.
	result, err := a.execute(ctx, innerOpts)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Stream streams a response from the agent.
// Ported from TS: stream(messages, streamOptions)
//
// Mirrors Generate() but returns a streaming MastraModelOutput.
// 1. Validates the request context.
// 2. Merges default options with call-specific options.
// 3. Resolves the LLM and validates the model specification version.
// 4. Delegates to #execute with methodType="stream".
// 5. Returns the stream result.
func (a *Agent) Stream(ctx context.Context, messages MessageListInput, opts AgentExecutionOptions) (*MastraModelOutputStub, error) {
	// Validate request context if schema is provided.
	if err := a.validateRequestContext(opts.RequestContext); err != nil {
		return nil, err
	}

	// Merge default options with call-specific options.
	defaultOpts, err := a.GetDefaultOptions(opts.RequestContext)
	if err != nil {
		return nil, err
	}
	mergedOpts := mergeExecutionOptions(defaultOpts, opts)

	// Resolve the LLM.
	llm, err := a.GetLLM(mergedOpts.RequestContext, nil)
	if err != nil {
		return nil, err
	}

	// Validate model specification version.
	modelInfo := llm.GetModel()
	if !IsSupportedLanguageModel(modelInfo) {
		specVersion := modelInfo.SpecificationVersion()
		var text string
		if specVersion == "v1" {
			text = fmt.Sprintf(
				"Agent %q is using AI SDK v4 model which is not compatible with Stream(). "+
					"Please use AI SDK v5+ models or call StreamLegacy() instead.",
				a.AgentName,
			)
		} else {
			text = fmt.Sprintf(
				"Agent %q has a model with unrecognized specificationVersion %q. "+
					"Supported versions: v1 (legacy), v2 (AI SDK v5), v3 (AI SDK v6).",
				a.AgentName, specVersion,
			)
		}
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_STREAM_V1_MODEL_NOT_SUPPORTED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Details: map[string]any{
				"agentName":            a.AgentName,
				"specificationVersion": specVersion,
			},
			Text: text,
		})
	}

	// Build inner execution options.
	innerOpts := InnerAgentExecutionOptions{
		AgentExecutionOptionsBase: mergedOpts.AgentExecutionOptionsBase,
		StructuredOutput:          mergedOpts.StructuredOutput,
		Messages:                  messages,
		MethodType:                AgentMethodTypeStream,
	}

	// Use agent's maxProcessorRetries as default, allow options to override.
	if innerOpts.MaxProcessorRetries == nil {
		innerOpts.MaxProcessorRetries = a.maxProcessorRetries
	}

	// Delegate to #execute.
	result, err := a.execute(ctx, innerOpts)
	if err != nil {
		return nil, err
	}

	// For stream, we return a MastraModelOutputStub wrapping the full output.
	return &MastraModelOutputStub{Object: result}, nil
}

// Network executes the agent as a network orchestrator.
// Ported from TS: network(messages, options)
//
// Orchestrates multi-agent collaboration via the networkLoop:
// 1. Merges default network options with call-specific options.
// 2. Resolves thread/resource IDs from request context (reserved keys take precedence).
// 3. Delegates to networkLoop with routing agent, messages, and all config.
//
// TODO: Implement fully once the loop/network package is ported.
func (a *Agent) Network(ctx context.Context, messages MessageListInput, opts *NetworkOptions) (MastraAgentNetworkStream, error) {
	var effectiveOpts NetworkOptions
	if opts != nil {
		effectiveOpts = *opts
	}

	reqCtx := effectiveOpts.RequestContext
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// Merge default network options with call-specific options.
	defaultNetworkOpts, err := a.GetDefaultNetworkOptions(reqCtx)
	if err != nil {
		return nil, err
	}

	// Deep merge: call-specific opts override defaults, but nested objects are merged.
	mergedOpts := defaultNetworkOpts
	if opts != nil {
		if opts.Memory != nil {
			mergedOpts.Memory = opts.Memory
		}
		if opts.RunID != "" {
			mergedOpts.RunID = opts.RunID
		}
		if opts.RequestContext != nil {
			mergedOpts.RequestContext = opts.RequestContext
		}
		if opts.MaxSteps != nil {
			mergedOpts.MaxSteps = opts.MaxSteps
		}
		if opts.ModelSettings != nil {
			mergedOpts.ModelSettings = opts.ModelSettings
		}
		if opts.Completion != nil {
			mergedOpts.Completion = opts.Completion
		}
		if opts.Routing != nil {
			mergedOpts.Routing = opts.Routing
		}
		if opts.OnIterationComplete != nil {
			mergedOpts.OnIterationComplete = opts.OnIterationComplete
		}
		if opts.StructuredOutput != nil {
			mergedOpts.StructuredOutput = opts.StructuredOutput
		}
		if opts.OnStepFinish != nil {
			mergedOpts.OnStepFinish = opts.OnStepFinish
		}
		if opts.OnError != nil {
			mergedOpts.OnError = opts.OnError
		}
		if opts.OnAbort != nil {
			mergedOpts.OnAbort = opts.OnAbort
		}
		mergedOpts.AutoResumeSuspendedTools = opts.AutoResumeSuspendedTools
	}

	// Generate run ID if not provided.
	runID := mergedOpts.RunID
	if runID == "" {
		runID = uuid.New().String()
	}

	// Resolve thread/resource IDs.
	// Reserved keys from requestContext take precedence for security.
	var threadID, resourceID string
	if mergedOpts.Memory != nil {
		switch t := mergedOpts.Memory.Thread.(type) {
		case string:
			threadID = t
		case map[string]any:
			if id, ok := t["id"].(string); ok {
				threadID = id
			}
		}
		resourceID = mergedOpts.Memory.Resource
	}

	_ = ctx
	_ = messages
	_ = runID
	_ = threadID
	_ = resourceID

	// TODO: Implement full networkLoop call once the loop/network package is ported.
	// The TS code calls: networkLoop({ networkName, requestContext, runId, routingAgent,
	//   routingAgentOptions, generateId, maxIterations, messages, threadId, resourceId,
	//   validation, routing, onIterationComplete, autoResumeSuspendedTools, mastra,
	//   structuredOutput, onStepFinish, onError, onAbort, abortSignal })
	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "AGENT_NETWORK_NOT_FULLY_IMPLEMENTED",
		Domain:   mastraerror.ErrorDomainAgent,
		Category: mastraerror.ErrorCategorySystem,
		Text:     "Network pipeline requires the loop/network package to be ported.",
	})
}

// GenerateLegacy is the legacy implementation of generate using AI SDK v4 models.
// TODO: Implement once the legacy handler is complete.
func (a *Agent) GenerateLegacy(ctx context.Context, messages MessageListInput, opts AgentGenerateOptions) (any, error) {
	handler := a.getLegacyHandler()
	return handler.GenerateLegacy(ctx, messages, opts)
}

// StreamLegacy is the legacy implementation of stream using AI SDK v4 models.
// TODO: Implement once the legacy handler is complete.
func (a *Agent) StreamLegacy(ctx context.Context, messages MessageListInput, opts AgentStreamOptions) (any, error) {
	handler := a.getLegacyHandler()
	return handler.StreamLegacy(ctx, messages, opts)
}

// ---------------------------------------------------------------------------
// Resume methods
// ---------------------------------------------------------------------------

// ResumeStream resumes a previously suspended stream execution.
// Used to continue execution after a suspension point (e.g., tool approval, workflow suspend).
//
// Ported from TS: async resumeStream(resumeData, streamOptions)
func (a *Agent) ResumeStream(ctx context.Context, resumeData any, opts AgentExecutionOptions) (any, error) {
	defaultOpts, err := a.GetDefaultOptions(opts.RequestContext)
	if err != nil {
		return nil, err
	}
	mergedOpts := mergeExecutionOptions(defaultOpts, opts)

	llm, err := a.GetLLM(mergedOpts.RequestContext, nil)
	if err != nil {
		return nil, err
	}

	if !IsSupportedLanguageModel(llm.GetModel()) {
		specVersion := llm.GetModel().SpecificationVersion()
		text := fmt.Sprintf(`Model has unrecognized specificationVersion "%s". Supported versions: v1 (legacy), v2 (AI SDK v5), v3 (AI SDK v6).`, specVersion)
		if specVersion == "v1" {
			text = "V1 models are not supported for resumeStream. Please use streamLegacy instead."
		}
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_STREAM_V1_MODEL_NOT_SUPPORTED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Text:     text,
		})
	}

	// Load existing snapshot from workflow storage
	// TODO: implement once mastra storage/workflows is ported:
	//   workflowsStore := a.mastra.GetStorage().GetStore("workflows")
	//   existingSnapshot := workflowsStore.LoadWorkflowSnapshot({workflowName: "agentic-loop", runId: opts.RunID})
	var existingSnapshot any

	// Call execute with resume context
	result, err := a.execute(ctx, InnerAgentExecutionOptions{
		AgentExecutionOptionsBase: mergedOpts.AgentExecutionOptionsBase,
		Messages:                  nil, // empty messages for resume
		ResumeContext: &ResumeContext{
			ResumeData: resumeData,
			Snapshot:   existingSnapshot,
		},
		MethodType:       "stream",
		StructuredOutput: mergedOpts.StructuredOutput,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ResumeGenerate resumes a previously suspended generate execution.
// Used to continue execution after a suspension point (e.g., tool approval, workflow suspend).
//
// Ported from TS: async resumeGenerate(resumeData, options)
func (a *Agent) ResumeGenerate(ctx context.Context, resumeData any, opts AgentExecutionOptions) (any, error) {
	defaultOpts, err := a.GetDefaultOptions(opts.RequestContext)
	if err != nil {
		return nil, err
	}
	mergedOpts := mergeExecutionOptions(defaultOpts, opts)

	llm, err := a.GetLLM(mergedOpts.RequestContext, nil)
	if err != nil {
		return nil, err
	}

	if !IsSupportedLanguageModel(llm.GetModel()) {
		specVersion := llm.GetModel().SpecificationVersion()
		text := fmt.Sprintf(`Agent "%s" has a model with unrecognized specificationVersion "%s".`, a.AgentName, specVersion)
		if specVersion == "v1" {
			text = fmt.Sprintf(`Agent "%s" is using AI SDK v4 model which is not compatible with generate(). Please use AI SDK v5+ models or call generateLegacy() instead.`, a.AgentName)
		}
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_GENERATE_V1_MODEL_NOT_SUPPORTED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Text:     text,
		})
	}

	// Load existing snapshot from workflow storage
	// TODO: implement once mastra storage/workflows is ported
	var existingSnapshot any

	// Call execute with resume context
	result, err := a.execute(ctx, InnerAgentExecutionOptions{
		AgentExecutionOptionsBase: mergedOpts.AgentExecutionOptionsBase,
		Messages:                  nil,
		ResumeContext: &ResumeContext{
			ResumeData: resumeData,
			Snapshot:   existingSnapshot,
		},
		MethodType:       "generate",
		StructuredOutput: mergedOpts.StructuredOutput,
	})
	if err != nil {
		return nil, err
	}

	// For generate, get the full output
	// TODO: call result.GetFullOutput() once MastraModelOutput is ported
	return result, nil
}

// ApproveToolCall approves a pending tool call and resumes execution (streaming).
// Used when RequireToolApproval is enabled to allow the agent to proceed with a tool call.
//
// Ported from TS: async approveToolCall(options)
func (a *Agent) ApproveToolCall(ctx context.Context, opts AgentExecutionOptions) (any, error) {
	return a.ResumeStream(ctx, map[string]any{"approved": true}, opts)
}

// DeclineToolCall declines a pending tool call and resumes execution (streaming).
// Used when RequireToolApproval is enabled to prevent the agent from executing a tool call.
//
// Ported from TS: async declineToolCall(options)
func (a *Agent) DeclineToolCall(ctx context.Context, opts AgentExecutionOptions) (any, error) {
	return a.ResumeStream(ctx, map[string]any{"approved": false}, opts)
}

// ApproveToolCallGenerate approves a pending tool call and returns the complete result (non-streaming).
// Used when RequireToolApproval is enabled with Generate() to allow the agent to proceed.
//
// Ported from TS: async approveToolCallGenerate(options)
func (a *Agent) ApproveToolCallGenerate(ctx context.Context, opts AgentExecutionOptions) (any, error) {
	return a.ResumeGenerate(ctx, map[string]any{"approved": true}, opts)
}

// DeclineToolCallGenerate declines a pending tool call and returns the complete result (non-streaming).
// Used when RequireToolApproval is enabled with Generate() to prevent tool execution.
//
// Ported from TS: async declineToolCallGenerate(options)
func (a *Agent) DeclineToolCallGenerate(ctx context.Context, opts AgentExecutionOptions) (any, error) {
	return a.ResumeGenerate(ctx, map[string]any{"approved": false}, opts)
}

// ---------------------------------------------------------------------------
// Tool formatting
// ---------------------------------------------------------------------------

var (
	invalidCharRegex  = regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
	startingCharRegex = regexp.MustCompile(`^[a-zA-Z_]`)
)

// FormatTools validates and normalizes tool names to comply with naming restrictions.
func (a *Agent) FormatTools(tools map[string]CoreTool) (map[string]CoreTool, error) {
	for key := range tools {
		if tools[key] == nil {
			continue
		}
		if len(key) > 63 || invalidCharRegex.MatchString(key) || !startingCharRegex.MatchString(key) {
			newKey := invalidCharRegex.ReplaceAllString(key, "_")
			if !startingCharRegex.MatchString(newKey) {
				newKey = "_" + newKey
			}
			if len(newKey) > 63 {
				newKey = newKey[:63]
			}

			if _, exists := tools[newKey]; exists {
				return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "AGENT_TOOL_NAME_COLLISION",
					Domain:   mastraerror.ErrorDomainAgent,
					Category: mastraerror.ErrorCategoryUser,
					Details: map[string]any{
						"agentName": a.AgentName,
						"toolName":  newKey,
					},
					Text: fmt.Sprintf("Two or more tools resolve to the same name %q. Please rename one of the tools to avoid this collision.", newKey),
				})
			}

			tools[newKey] = tools[key]
			delete(tools, key)
		}
	}
	return tools, nil
}

// ConvertTools assembles all tools from 7 sources into a unified CoreTool dictionary.
// Ported from TS: convertTools({ toolsets, clientTools, threadId, resourceId, ... })
//
// Sources (in merge order, later sources override earlier):
// 1. Assigned tools (agent's configured tools)
// 2. Memory tools (working memory, thread management)
// 3. Toolset tools (additional tool sets passed per-call)
// 4. Client-side tools (tools provided by the client)
// 5. Agent tools (sub-agent delegation tools)
// 6. Workflow tools (workflow execution tools)
// 7. Workspace tools (file/code workspace tools)
//
// After merging, tool names are validated and normalized via FormatTools.
func (a *Agent) ConvertTools(ctx context.Context, params ConvertToolsParams) (map[string]CoreTool, error) {
	reqCtx := params.RequestContext
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// TODO: Create mastraProxy once createMastraProxy is ported.
	// In TS: let mastraProxy = undefined;
	// if (this.#mastra) { mastraProxy = createMastraProxy({ mastra: this.#mastra, logger }); }

	// 1. Assigned tools - the agent's configured tools
	assignedTools, err := a.listAssignedTools(ctx, listToolsArgs{
		RunID:                   params.RunID,
		ResourceID:              params.ResourceID,
		ThreadID:                params.ThreadID,
		RequestContext:          reqCtx,
		OutputWriter:            params.OutputWriter,
		AutoResumeSuspendedTools: params.AutoResumeSuspended,
	})
	if err != nil {
		return nil, err
	}

	// 2. Memory tools
	memoryTools, err := a.listMemoryTools(ctx, listMemoryToolsArgs{
		RunID:                   params.RunID,
		ResourceID:              params.ResourceID,
		ThreadID:                params.ThreadID,
		RequestContext:          reqCtx,
		MemoryConfig:            params.MemoryConfig,
		AutoResumeSuspendedTools: params.AutoResumeSuspended,
	})
	if err != nil {
		return nil, err
	}

	// 3. Toolset tools
	toolsetTools, err := a.listToolsetTools(ctx, listToolsetToolsArgs{
		RunID:                   params.RunID,
		ResourceID:              params.ResourceID,
		ThreadID:                params.ThreadID,
		RequestContext:          reqCtx,
		Toolsets:                params.Toolsets,
		AutoResumeSuspendedTools: params.AutoResumeSuspended,
	})
	if err != nil {
		return nil, err
	}

	// 4. Client-side tools
	clientSideTools, err := a.listClientTools(ctx, listClientToolsArgs{
		RunID:                   params.RunID,
		ResourceID:              params.ResourceID,
		ThreadID:                params.ThreadID,
		RequestContext:          reqCtx,
		ClientTools:             params.ClientTools,
		AutoResumeSuspendedTools: params.AutoResumeSuspended,
	})
	if err != nil {
		return nil, err
	}

	// 5. Agent tools (sub-agent delegation)
	agentTools, err := a.listAgentToolsMethod(ctx, listAgentToolsArgs{
		RunID:                   params.RunID,
		ResourceID:              params.ResourceID,
		ThreadID:                params.ThreadID,
		RequestContext:          reqCtx,
		MethodType:              params.MethodType,
		AutoResumeSuspendedTools: params.AutoResumeSuspended,
		Delegation:              params.Delegation,
	})
	if err != nil {
		return nil, err
	}

	// 6. Workflow tools
	workflowTools, err := a.listWorkflowTools(ctx, listWorkflowToolsArgs{
		RunID:                   params.RunID,
		ResourceID:              params.ResourceID,
		ThreadID:                params.ThreadID,
		RequestContext:          reqCtx,
		MethodType:              params.MethodType,
		AutoResumeSuspendedTools: params.AutoResumeSuspended,
	})
	if err != nil {
		return nil, err
	}

	// 7. Workspace tools
	workspaceTools, err := a.listWorkspaceTools(ctx, listWorkspaceToolsArgs{
		RunID:                   params.RunID,
		ResourceID:              params.ResourceID,
		ThreadID:                params.ThreadID,
		RequestContext:          reqCtx,
		AutoResumeSuspendedTools: params.AutoResumeSuspended,
	})
	if err != nil {
		return nil, err
	}

	// Merge all tools (later sources override earlier).
	allTools := make(map[string]CoreTool)
	for k, v := range assignedTools {
		allTools[k] = v
	}
	for k, v := range memoryTools {
		allTools[k] = v
	}
	for k, v := range toolsetTools {
		allTools[k] = v
	}
	for k, v := range clientSideTools {
		allTools[k] = v
	}
	for k, v := range agentTools {
		allTools[k] = v
	}
	for k, v := range workflowTools {
		allTools[k] = v
	}
	for k, v := range workspaceTools {
		allTools[k] = v
	}

	return a.FormatTools(allTools)
}

// ---------------------------------------------------------------------------
// Tool listing internal methods
// ---------------------------------------------------------------------------

// listToolsArgs holds common arguments for tool listing methods.
type listToolsArgs struct {
	RunID                   string
	ResourceID              string
	ThreadID                string
	RequestContext          *requestcontext.RequestContext
	OutputWriter            OutputWriter
	AutoResumeSuspendedTools bool
}

// listAssignedTools retrieves and converts the agent's configured tools.
// Ported from TS: listAssignedTools({ runId, resourceId, threadId, requestContext, ... })
func (a *Agent) listAssignedTools(_ context.Context, args listToolsArgs) (map[string]CoreTool, error) {
	a.Logger().Debug(fmt.Sprintf("[Agents:%s] - Assembling assigned tools", a.AgentName),
		"runId", args.RunID, "threadId", args.ThreadID, "resourceId", args.ResourceID)

	assignedTools, err := a.ListTools(args.RequestContext)
	if err != nil {
		return nil, err
	}

	// TODO: Convert each tool via makeCoreTool once the tools package is ported.
	// In TS, each tool goes through makeCoreTool(tool, options, undefined, autoResumeSuspendedTools)
	// which wraps it with logging, observability, memory injection, etc.
	// For now, pass tools through as-is since CoreTool = any.
	result := make(map[string]CoreTool, len(assignedTools))
	for k, v := range assignedTools {
		if v != nil {
			result[k] = v
		}
	}
	return result, nil
}

// listMemoryToolsArgs extends listToolsArgs with memory config.
type listMemoryToolsArgs struct {
	RunID                   string
	ResourceID              string
	ThreadID                string
	RequestContext          *requestcontext.RequestContext
	MemoryConfig            *MemoryConfig
	AutoResumeSuspendedTools bool
}

// listMemoryTools retrieves memory-related tools (working memory, thread management).
// Ported from TS: listMemoryTools({ runId, resourceId, threadId, requestContext, memoryConfig, ... })
func (a *Agent) listMemoryTools(_ context.Context, args listMemoryToolsArgs) (map[string]CoreTool, error) {
	convertedMemoryTools := make(map[string]CoreTool)

	// Skip memory tools in agent network context.
	if a.agentNetworkAppend {
		a.Logger().Debug(fmt.Sprintf("[Agent:%s] - Skipping memory tools (agent network context)", a.AgentName),
			"runId", args.RunID)
		return convertedMemoryTools, nil
	}

	// Get memory instance.
	memory, err := a.GetMemory(args.RequestContext)
	if err != nil {
		return nil, err
	}

	// Skip memory tools if there's no usable context — thread-scoped needs threadId, resource-scoped needs resourceId.
	if args.ThreadID == "" && args.ResourceID == "" {
		a.Logger().Debug(fmt.Sprintf("[Agent:%s] - Skipping memory tools (no thread or resource context)", a.AgentName),
			"runId", args.RunID)
		return convertedMemoryTools, nil
	}

	if memory == nil {
		return convertedMemoryTools, nil
	}

	// Get memory tools from the memory instance.
	memoryTools := memory.ListTools(args.MemoryConfig)
	if len(memoryTools) == 0 {
		return convertedMemoryTools, nil
	}

	toolNames := make([]string, 0, len(memoryTools))
	for k := range memoryTools {
		toolNames = append(toolNames, k)
	}

	a.Logger().Debug(
		fmt.Sprintf("[Agent:%s] - Adding tools from memory %s", a.AgentName, strings.Join(toolNames, ", ")),
		"runId", args.RunID,
	)

	// Convert each memory tool. In TS this goes through makeCoreTool with ToolOptions.
	// Since makeCoreTool is not yet ported as a Go function, we pass tools through as-is.
	// The tool values from memory.ListTools() are already ToolAction-compatible.
	for toolName, tool := range memoryTools {
		if tool != nil {
			convertedMemoryTools[toolName] = tool
		}
	}

	return convertedMemoryTools, nil
}

// listToolsetToolsArgs extends args with toolsets.
type listToolsetToolsArgs struct {
	RunID                   string
	ResourceID              string
	ThreadID                string
	RequestContext          *requestcontext.RequestContext
	Toolsets                ToolsetsInput
	AutoResumeSuspendedTools bool
}

// listToolsetTools retrieves tools from additional tool sets.
// Ported from TS: listToolsets({ runId, threadId, resourceId, toolsets, ... })
func (a *Agent) listToolsetTools(_ context.Context, args listToolsetToolsArgs) (map[string]CoreTool, error) {
	result := make(map[string]CoreTool)
	if len(args.Toolsets) == 0 {
		return result, nil
	}

	a.Logger().Debug(fmt.Sprintf("[Agent:%s] - Adding tools from toolsets", a.AgentName), "runId", args.RunID)

	// TODO: Convert each tool via makeCoreTool once the tools package is ported.
	for _, toolset := range args.Toolsets {
		for toolName, tool := range toolset {
			if tool != nil {
				result[toolName] = tool
			}
		}
	}
	return result, nil
}

// listClientToolsArgs extends args with client tools.
type listClientToolsArgs struct {
	RunID                   string
	ResourceID              string
	ThreadID                string
	RequestContext          *requestcontext.RequestContext
	ClientTools             ToolsInput
	AutoResumeSuspendedTools bool
}

// listClientTools retrieves and converts client-side tools.
// Ported from TS: listClientTools({ runId, threadId, resourceId, clientTools, ... })
func (a *Agent) listClientTools(_ context.Context, args listClientToolsArgs) (map[string]CoreTool, error) {
	result := make(map[string]CoreTool)
	if len(args.ClientTools) == 0 {
		return result, nil
	}

	a.Logger().Debug(fmt.Sprintf("[Agent:%s] - Adding client tools", a.AgentName), "runId", args.RunID)

	// TODO: Convert client tools via makeCoreTool once the tools package is ported.
	// In TS, client tools are marked with `type: 'client'` which prevents server-side execution.
	for k, v := range args.ClientTools {
		if v != nil {
			result[k] = v
		}
	}
	return result, nil
}

// listAgentToolsArgs extends args with method type and delegation.
type listAgentToolsArgs struct {
	RunID                   string
	ResourceID              string
	ThreadID                string
	RequestContext          *requestcontext.RequestContext
	MethodType              AgentMethodType
	AutoResumeSuspendedTools bool
	Delegation              *DelegationConfig
}

// listAgentToolsMethod retrieves sub-agent delegation tools.
// Named listAgentToolsMethod to avoid collision with ListAgents.
// Ported from TS: listAgentTools({ runId, resourceId, threadId, requestContext, methodType, delegation, ... })
func (a *Agent) listAgentToolsMethod(_ context.Context, args listAgentToolsArgs) (map[string]CoreTool, error) {
	// TODO: Implement agent tool creation once the delegation system is ported.
	// In TS, this is ~600 lines that creates a tool for each sub-agent allowing
	// the parent agent to delegate work. Each tool handles:
	// - onDelegationStart/onDelegationComplete hooks
	// - messageFilter for controlling context
	// - Memory thread management for sub-agents
	// - Streaming/generation delegation
	_ = args
	return map[string]CoreTool{}, nil
}

// listWorkflowToolsArgs extends args with method type.
type listWorkflowToolsArgs struct {
	RunID                   string
	ResourceID              string
	ThreadID                string
	RequestContext          *requestcontext.RequestContext
	MethodType              AgentMethodType
	AutoResumeSuspendedTools bool
}

// listWorkflowTools retrieves workflow execution tools.
// Ported from TS: listWorkflowTools({ runId, resourceId, threadId, requestContext, methodType, ... })
func (a *Agent) listWorkflowTools(_ context.Context, args listWorkflowToolsArgs) (map[string]CoreTool, error) {
	// TODO: Implement workflow tool creation once the workflows package is integrated.
	// In TS, this creates a tool for each configured workflow that allows
	// the agent to execute workflows as tool calls.
	_ = args
	return map[string]CoreTool{}, nil
}

// listWorkspaceToolsArgs extends args for workspace.
type listWorkspaceToolsArgs struct {
	RunID                   string
	ResourceID              string
	ThreadID                string
	RequestContext          *requestcontext.RequestContext
	AutoResumeSuspendedTools bool
}

// listWorkspaceTools retrieves workspace-related tools.
// Ported from TS: listWorkspaceTools({ runId, resourceId, threadId, requestContext, ... })
func (a *Agent) listWorkspaceTools(_ context.Context, args listWorkspaceToolsArgs) (map[string]CoreTool, error) {
	convertedWorkspaceTools := make(map[string]CoreTool)

	// Skip workspace tools in agent network context.
	if a.agentNetworkAppend {
		a.Logger().Debug(fmt.Sprintf("[Agent:%s] - Skipping workspace tools (agent network context)", a.AgentName),
			"runId", args.RunID)
		return convertedWorkspaceTools, nil
	}

	// Get workspace instance.
	ws, err := a.GetWorkspace(args.RequestContext)
	if err != nil {
		return nil, err
	}
	if ws == nil {
		return convertedWorkspaceTools, nil
	}

	// In TS: const workspaceTools = createWorkspaceTools(workspace)
	// The workspace/tools.CreateWorkspaceTools function accepts a WorkspaceAccessor interface.
	// workspace.Workspace.GetToolsConfig() returns a value (not pointer), which doesn't
	// satisfy workspace/tools.WorkspaceAccessor. Call workspace/tools.CreateWorkspaceTools
	// via an any-based wrapper or use the workspace's tool config directly.
	// Since AnyWorkspace is *workspace.Workspace (concrete type), we can call methods directly.
	toolsCfg := ws.GetToolsConfig()
	_ = toolsCfg // Tool config is available but CreateWorkspaceTools has interface mismatch.
	// TODO: Wire workspace/tools.CreateWorkspaceTools once GetToolsConfig signature is aligned.
	// For now, workspace tools are not added until the interface is unified.
	a.Logger().Debug(fmt.Sprintf("[Agent:%s] - Workspace tools pending interface alignment", a.AgentName),
		"runId", args.RunID)

	return convertedWorkspaceTools, nil
}

// ConvertToolsParams holds the parameters for ConvertTools.
type ConvertToolsParams struct {
	Toolsets             ToolsetsInput
	ClientTools          ToolsInput
	ThreadID             string
	ResourceID           string
	RunID                string
	RequestContext       *requestcontext.RequestContext
	OutputWriter         OutputWriter
	MethodType           AgentMethodType
	MemoryConfig         *MemoryConfig
	AutoResumeSuspended  bool
	Delegation           *DelegationConfig
	ObservabilityContext
}

// ---------------------------------------------------------------------------
// Model management
// ---------------------------------------------------------------------------

// UpdateInstructions updates the agent's instructions.
func (a *Agent) UpdateInstructions(newInstructions DynamicArgument) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.instructions = newInstructions
	a.Logger().Debug(fmt.Sprintf("[Agents:%s] Instructions updated.", a.AgentName))
}

// UpdateModel updates the agent's model configuration.
func (a *Agent) UpdateModel(model any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Model = model
	a.Logger().Debug(fmt.Sprintf("[Agents:%s] Model updated.", a.AgentName))
}

// ResetToOriginalModel resets the agent's model to the original model set during construction.
func (a *Agent) ResetToOriginalModel() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if fallbacks, ok := a.originalModel.(ModelFallbacks); ok {
		cloned := make(ModelFallbacks, len(fallbacks))
		copy(cloned, fallbacks)
		a.Model = cloned
	} else {
		a.Model = a.originalModel
	}
	a.Logger().Debug(fmt.Sprintf("[Agents:%s] Model reset to original.", a.AgentName))
}

// ReorderModels reorders the model list based on the given model IDs.
func (a *Agent) ReorderModels(modelIDs []string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	fallbacks, ok := a.Model.(ModelFallbacks)
	if !ok {
		a.Logger().Warn(fmt.Sprintf("[Agents:%s] model is not an array", a.AgentName))
		return
	}

	idxMap := make(map[string]int)
	for i, id := range modelIDs {
		idxMap[id] = i
	}

	// Simple sort: put models in the order specified by modelIDs.
	sorted := make(ModelFallbacks, len(fallbacks))
	copy(sorted, fallbacks)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			iIdx, iOk := idxMap[sorted[i].ID]
			jIdx, jOk := idxMap[sorted[j].ID]
			iPrio := len(modelIDs)
			jPrio := len(modelIDs)
			if iOk {
				iPrio = iIdx
			}
			if jOk {
				jPrio = jIdx
			}
			if jPrio < iPrio {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	a.Model = sorted
	a.Logger().Debug(fmt.Sprintf("[Agents:%s] Models reordered", a.AgentName))
}

// UpdateModelInModelList updates a specific model in the model list.
func (a *Agent) UpdateModelInModelList(id string, model DynamicArgument, enabled *bool, maxRetries *int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	fallbacks, ok := a.Model.(ModelFallbacks)
	if !ok {
		a.Logger().Warn(fmt.Sprintf("[Agents:%s] model is not an array", a.AgentName))
		return
	}

	found := false
	for i, mdl := range fallbacks {
		if mdl.ID == id {
			found = true
			if model != nil {
				fallbacks[i].Model = model
			}
			if enabled != nil {
				fallbacks[i].Enabled = *enabled
			}
			if maxRetries != nil {
				fallbacks[i].MaxRetries = *maxRetries
			}
			break
		}
	}

	if !found {
		a.Logger().Warn(fmt.Sprintf("[Agents:%s] model %s not found", a.AgentName, id))
		return
	}

	a.Model = fallbacks
	a.Logger().Debug(fmt.Sprintf("[Agents:%s] model %s updated", a.AgentName, id))
}

// GetOverridableFields returns a snapshot of the raw field values that may be overridden by stored config.
func (a *Agent) GetOverridableFields() map[string]any {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return map[string]any{
		"instructions": a.instructions,
		"model":        a.Model,
		"tools":        a.tools,
		"workspace":    a.workspace,
	}
}

// ---------------------------------------------------------------------------
// Registration
// ---------------------------------------------------------------------------

// RegisterPrimitives registers logger primitives with the agent.
func (a *Agent) RegisterPrimitives(p MastraPrimitives) {
	if p.Logger != nil {
		a.SetLogger(p.Logger)
	}
	a.mu.Lock()
	a.primitives = &p
	a.mu.Unlock()
	a.Logger().Debug(fmt.Sprintf("[Agents:%s] initialized.", a.AgentName))
}

// RegisterMastra registers the Mastra instance with the agent and auto-registers
// tools and processors.
// Ported from TS: __registerMastra(mastra)
//
// This method:
// 1. Stores the mastra reference.
// 2. Propagates logger to workspace if it's a direct instance (not a function).
// 3. Auto-registers tools with the Mastra instance (if tools are static objects).
// 4. Auto-registers input/output processors with the Mastra instance.
func (a *Agent) RegisterMastra(mastra Mastra) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.mastra = mastra

	// Propagate logger to workspace if it's a direct instance (not a factory function).
	// In TS: if (this.#workspace && typeof this.#workspace !== 'function') { workspace.__setLogger(this.logger); }
	// TODO: Implement workspace logger propagation once workspace package has __setLogger.

	// Auto-register tools with the Mastra instance.
	// Only register tools that are static objects (not functions).
	if a.tools != nil {
		if toolsMap, ok := a.tools.(ToolsInput); ok {
			for key, tool := range toolsMap {
				if tool == nil {
					continue
				}
				// Only add tools that have an id property (ToolAction type).
				if toolObj, ok := tool.(map[string]any); ok {
					if _, hasID := toolObj["id"]; hasID {
						toolKey := key
						if idStr, ok := toolObj["id"].(string); ok && idStr != "" {
							toolKey = idStr
						}
						// TODO: Call mastra.AddTool(tool, toolKey) once Mastra interface has AddTool.
						_ = toolKey
					}
				}
			}
		}
	}

	// Auto-register input processors with the Mastra instance.
	if a.inputProcessors != nil {
		if processors, ok := a.inputProcessors.([]InputProcessorOrWorkflow); ok {
			for _, processor := range processors {
				// TODO: Call mastra.AddProcessor(processor) once Mastra interface has AddProcessor.
				// TODO: Call mastra.AddProcessorConfiguration(processor, a.ID, "input")
				_ = processor
			}
		}
	}

	// Auto-register output processors with the Mastra instance.
	if a.outputProcessors != nil {
		if processors, ok := a.outputProcessors.([]OutputProcessorOrWorkflow); ok {
			for _, processor := range processors {
				// TODO: Call mastra.AddProcessor(processor) once Mastra interface has AddProcessor.
				// TODO: Call mastra.AddProcessorConfiguration(processor, a.ID, "output")
				_ = processor
			}
		}
	}
}

// SetTools sets the concrete tools for the agent.
func (a *Agent) SetTools(tools DynamicArgument) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tools = tools
	a.Logger().Debug(fmt.Sprintf("[Agents:%s] Tools set for agent %s", a.AgentName, a.AgentName))
}

// SetMemory sets the memory for the agent.
func (a *Agent) SetMemory(memory DynamicArgument) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.memory = memory
}

// SetWorkspace sets the workspace for the agent.
func (a *Agent) SetWorkspace(workspace DynamicArgument) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.workspace = workspace

	// In TS, __setWorkspace also propagates the logger and registers with Mastra:
	//   if (this.#mastra && workspace && typeof workspace !== 'function') {
	//     workspace.__setLogger(this.logger);
	//     this.#mastra.addWorkspace(workspace, undefined, { source: 'agent', agentId: this.id, agentName: this.name });
	//   }
	// TODO: Propagate logger and register workspace with Mastra once workspace package has
	// __setLogger and Mastra interface has AddWorkspace.
}

// ---------------------------------------------------------------------------
// Title generation
// ---------------------------------------------------------------------------

// ResolveTitleGenerationConfig resolves the configuration for title generation.
func (a *Agent) ResolveTitleGenerationConfig(config any) TitleGenerationResult {
	switch c := config.(type) {
	case bool:
		return TitleGenerationResult{ShouldGenerate: c}
	case map[string]any:
		result := TitleGenerationResult{ShouldGenerate: true}
		if model, ok := c["model"]; ok {
			result.Model = model
		}
		if instructions, ok := c["instructions"]; ok {
			result.Instructions = instructions
		}
		return result
	default:
		return TitleGenerationResult{ShouldGenerate: false}
	}
}

// TitleGenerationResult holds the resolved title generation configuration.
type TitleGenerationResult struct {
	ShouldGenerate bool
	Model          any
	Instructions   any
}

// ResolveTitleInstructions resolves title generation instructions.
func (a *Agent) ResolveTitleInstructions(reqCtx *requestcontext.RequestContext, instructions DynamicArgument) (string, error) {
	const defaultInstructions = `- you will generate a short title based on the first message a user begins a conversation with
- ensure it is not more than 80 characters long
- the title should be a summary of the user's message
- do not use quotes or colons
- the entire text you return will be used as the title`

	if instructions == nil {
		return defaultInstructions, nil
	}

	if s, ok := instructions.(string); ok {
		return s, nil
	}

	if fn, ok := instructions.(func(*requestcontext.RequestContext, Mastra) (string, error)); ok {
		result, err := fn(reqCtx, a.mastra)
		if err != nil {
			return defaultInstructions, err
		}
		if result == "" {
			return defaultInstructions, nil
		}
		return result, nil
	}

	return defaultInstructions, nil
}

// GenerateTitleFromUserMessage generates a title from a user message using the LLM.
// Ported from TS: generateTitleFromUserMessage({ message, requestContext, model, instructions, ... })
//
// This method:
// 1. Resolves the LLM (optionally using a custom model for title generation).
// 2. Normalizes the message into text parts.
// 3. Resolves title instructions (custom or default).
// 4. Calls the LLM with system instructions and the user message.
// 5. Strips any think tags from the response.
func (a *Agent) GenerateTitleFromUserMessage(
	message any,
	reqCtx *requestcontext.RequestContext,
	model DynamicArgument,
	instructions DynamicArgument,
) (string, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	llm, err := a.GetLLM(reqCtx, model)
	if err != nil {
		return "", err
	}

	// Resolve instructions.
	systemInstructions, err := a.ResolveTitleInstructions(reqCtx, instructions)
	if err != nil {
		// Fall back to default instructions on error.
		a.Logger().Warn(fmt.Sprintf("[Agent:%s] - Error resolving title instructions: %v", a.AgentName, err))
	}

	// Normalize the message into text for the LLM prompt.
	// In TS, this uses MessageList().add(message, 'user').get.all.aiV5.ui().at(-1)
	// then extracts text parts from the normalized message.
	// Since MessageList is not yet fully ported, do a simplified text extraction.
	var messageText string
	switch m := message.(type) {
	case string:
		messageText = m
	case map[string]any:
		// Try to extract content from a message-like map.
		if content, ok := m["content"].(string); ok {
			messageText = content
		} else if content, ok := m["text"].(string); ok {
			messageText = content
		}
	default:
		// For complex message types, marshal to JSON as a fallback.
		// In TS: JSON.stringify(partsToGen)
		messageText = fmt.Sprintf("%v", m)
	}

	if messageText == "" {
		return "", fmt.Errorf("could not generate title from input %v", message)
	}

	// TODO: Call LLM to generate title once LLM streaming is fully ported.
	// In TS:
	//   if (isSupportedLanguageModel(llm.getModel())) {
	//     const result = (llm as MastraLLMVNext).stream({
	//       methodType: 'generate',
	//       messageList: [system: systemInstructions, user: messageText],
	//       ...
	//     });
	//     text = await result.text;
	//   } else {
	//     const result = await (llm as MastraLLMV1).__text({
	//       messages: [{role:'system', content: systemInstructions}, {role:'user', content: messageText}],
	//     });
	//     text = result.text;
	//   }
	_ = llm
	_ = systemInstructions

	// Strip any <think>...</think> tags from the response.
	// In TS: text.replace(/<think>[\s\S]*?<\/think>/g, '').trim()
	thinkTagRegex := regexp.MustCompile(`(?s)<think>.*?</think>`)
	cleanedText := thinkTagRegex.ReplaceAllString(messageText, "")
	cleanedText = strings.TrimSpace(cleanedText)

	// Until LLM is wired, return empty string (no title generated).
	_ = cleanedText
	return "", nil
}

// GenTitle generates a title from a user message, with error handling.
// Ported from TS: genTitle(userMessage, requestContext, observabilityContext, model, instructions)
func (a *Agent) GenTitle(
	userMessage any,
	reqCtx *requestcontext.RequestContext,
	model DynamicArgument,
	instructions DynamicArgument,
) string {
	if userMessage == nil {
		return ""
	}

	title, err := a.GenerateTitleFromUserMessage(userMessage, reqCtx, model, instructions)
	if err != nil {
		a.Logger().Error(fmt.Sprintf("[Agent:%s] - Error generating title: %v", a.AgentName, err))
		return ""
	}
	return title
}

// ---------------------------------------------------------------------------
// Voice methods
// ---------------------------------------------------------------------------

// GetVoice returns the voice instance for this agent with tools and instructions configured.
// Ported from TS: getVoice({ requestContext })
//
// The voice instance enables text-to-speech and speech-to-text capabilities.
func (a *Agent) GetVoice(reqCtx *requestcontext.RequestContext) (MastraVoice, error) {
	if a.voice != nil {
		// TODO: Once voice package is ported, call:
		//   voice.AddTools(a.ListTools(reqCtx))
		//   instructions, _ := a.GetInstructions(ctx, reqCtx)
		//   voice.AddInstructions(a.ConvertInstructionsToString(instructions))
		return a.voice, nil
	}
	// Return a DefaultVoice if no voice configured.
	return voice.NewDefaultVoice(), nil
}

// ---------------------------------------------------------------------------
// Processor methods
// ---------------------------------------------------------------------------

// ListResolvedInputProcessors resolves and returns the input processors.
// Ported from TS: listResolvedInputProcessors(requestContext, configuredProcessorOverrides)
//
// Processors are resolved from multiple sources and combined:
// 1. Memory processors (run first - fetch history, semantic recall, working memory)
// 2. Workspace instructions processors
// 3. Skills processors
// 4. User-configured processors (from agent config or per-call overrides)
//
// All are combined into a single processor workflow.
func (a *Agent) ListResolvedInputProcessors(
	reqCtx *requestcontext.RequestContext,
	overrides []InputProcessorOrWorkflow,
) ([]InputProcessorOrWorkflow, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// Get configured input processors - use overrides if provided,
	// otherwise use agent constructor processors.
	var configuredProcessors []InputProcessorOrWorkflow
	if overrides != nil {
		configuredProcessors = overrides
	} else if a.inputProcessors != nil {
		if fn, ok := a.inputProcessors.(func(*requestcontext.RequestContext) ([]InputProcessorOrWorkflow, error)); ok {
			procs, err := fn(reqCtx)
			if err != nil {
				return nil, err
			}
			configuredProcessors = procs
		} else if procs, ok := a.inputProcessors.([]InputProcessorOrWorkflow); ok {
			configuredProcessors = procs
		}
	}
	if configuredProcessors == nil {
		configuredProcessors = []InputProcessorOrWorkflow{}
	}

	// Get memory input processors.
	// In TS: const memoryProcessors = memory ? await memory.getInputProcessors(configuredProcessors, requestContext) : [];
	var memoryProcessors []InputProcessorOrWorkflow
	mem, memErr := a.GetMemory(reqCtx)
	if memErr != nil {
		return nil, memErr
	}
	if mem != nil {
		// Convert configuredProcessors to memory package's type (both are interface{}).
		memConfigured := make([]memory.InputProcessorOrWorkflow, len(configuredProcessors))
		for i, p := range configuredProcessors {
			memConfigured[i] = p
		}
		memProcs, err := mem.GetInputProcessors(context.Background(), memConfigured, reqCtx)
		if err != nil {
			return nil, err
		}
		for _, mp := range memProcs {
			memoryProcessors = append(memoryProcessors, mp)
		}
	}

	// Get workspace instructions processors.
	// In TS: const workspaceProcessors = await this.getWorkspaceInstructionsProcessors(configuredProcessors, requestContext);
	workspaceProcessors, wsErr := a.getWorkspaceInstructionsProcessors(configuredProcessors, reqCtx)
	if wsErr != nil {
		return nil, wsErr
	}

	// Get skills processors.
	// In TS: const skillsProcessors = await this.getSkillsProcessors(configuredProcessors, requestContext);
	skillsProcessors, skErr := a.getSkillsProcessors(configuredProcessors, reqCtx)
	if skErr != nil {
		return nil, skErr
	}

	// Combine all processors.
	// Memory processors should run first, then workspace, then skills, then user-configured.
	allProcessors := make([]InputProcessorOrWorkflow, 0,
		len(memoryProcessors)+len(workspaceProcessors)+len(skillsProcessors)+len(configuredProcessors))
	allProcessors = append(allProcessors, memoryProcessors...)
	allProcessors = append(allProcessors, workspaceProcessors...)
	allProcessors = append(allProcessors, skillsProcessors...)
	allProcessors = append(allProcessors, configuredProcessors...)

	// TODO: Combine into a single processor workflow via combineProcessorsIntoWorkflow
	// once the processors/workflow packages are ported.
	return allProcessors, nil
}

// ListResolvedOutputProcessors resolves and returns the output processors.
// Ported from TS: listResolvedOutputProcessors(requestContext, configuredProcessorOverrides)
//
// Processors are resolved from:
// 1. User-configured processors (from agent config or per-call overrides)
// 2. Memory processors (run last - to persist messages after other processing)
func (a *Agent) ListResolvedOutputProcessors(
	reqCtx *requestcontext.RequestContext,
	overrides []OutputProcessorOrWorkflow,
) ([]OutputProcessorOrWorkflow, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// Get configured output processors - use overrides if provided,
	// otherwise use agent constructor processors.
	var configuredProcessors []OutputProcessorOrWorkflow
	if overrides != nil {
		configuredProcessors = overrides
	} else if a.outputProcessors != nil {
		if fn, ok := a.outputProcessors.(func(*requestcontext.RequestContext) ([]OutputProcessorOrWorkflow, error)); ok {
			procs, err := fn(reqCtx)
			if err != nil {
				return nil, err
			}
			configuredProcessors = procs
		} else if procs, ok := a.outputProcessors.([]OutputProcessorOrWorkflow); ok {
			configuredProcessors = procs
		}
	}
	if configuredProcessors == nil {
		configuredProcessors = []OutputProcessorOrWorkflow{}
	}

	// Get memory output processors.
	// In TS: const memoryProcessors = memory ? await memory.getOutputProcessors(configuredProcessors, requestContext) : [];
	var memoryProcessors []OutputProcessorOrWorkflow
	mem, memErr := a.GetMemory(reqCtx)
	if memErr != nil {
		return nil, memErr
	}
	if mem != nil {
		memConfigured := make([]memory.OutputProcessorOrWorkflow, len(configuredProcessors))
		for i, p := range configuredProcessors {
			memConfigured[i] = p
		}
		memProcs, err := mem.GetOutputProcessors(context.Background(), memConfigured, reqCtx)
		if err != nil {
			return nil, err
		}
		for _, mp := range memProcs {
			memoryProcessors = append(memoryProcessors, mp)
		}
	}

	// Combine all processors.
	// Memory processors should run last (to persist messages after other processing).
	allProcessors := make([]OutputProcessorOrWorkflow, 0, len(configuredProcessors)+len(memoryProcessors))
	allProcessors = append(allProcessors, configuredProcessors...)
	allProcessors = append(allProcessors, memoryProcessors...)

	// TODO: Combine into a single processor workflow via combineProcessorsIntoWorkflow
	// once the processors/workflow packages are ported.
	return allProcessors, nil
}

// ListInputProcessors returns the input processors for this agent, resolving
// function-based processors if necessary. This includes both memory-derived
// and user-configured processors.
// Ported from TS: listInputProcessors(requestContext)
func (a *Agent) ListInputProcessors(reqCtx *requestcontext.RequestContext) ([]InputProcessorOrWorkflow, error) {
	return a.ListResolvedInputProcessors(reqCtx, nil)
}

// ListOutputProcessors returns the output processors for this agent, resolving
// function-based processors if necessary. This includes both memory-derived
// and user-configured processors.
// Ported from TS: listOutputProcessors(requestContext)
func (a *Agent) ListOutputProcessors(reqCtx *requestcontext.RequestContext) ([]OutputProcessorOrWorkflow, error) {
	return a.ListResolvedOutputProcessors(reqCtx, nil)
}

// ListConfiguredInputProcessors returns only the user-configured input processors,
// excluding memory-derived processors. Useful for scenarios where memory processors
// should not be applied (e.g., network routing agents).
// Ported from TS: listConfiguredInputProcessors(requestContext)
func (a *Agent) ListConfiguredInputProcessors(reqCtx *requestcontext.RequestContext) ([]InputProcessorOrWorkflow, error) {
	if a.inputProcessors == nil {
		return []InputProcessorOrWorkflow{}, nil
	}
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	var configuredProcessors []InputProcessorOrWorkflow
	if fn, ok := a.inputProcessors.(func(*requestcontext.RequestContext) ([]InputProcessorOrWorkflow, error)); ok {
		procs, err := fn(reqCtx)
		if err != nil {
			return nil, err
		}
		configuredProcessors = procs
	} else if procs, ok := a.inputProcessors.([]InputProcessorOrWorkflow); ok {
		configuredProcessors = procs
	}
	if configuredProcessors == nil {
		return []InputProcessorOrWorkflow{}, nil
	}

	// TODO: Combine into workflow via combineProcessorsIntoWorkflow once processors package is ported.
	return configuredProcessors, nil
}

// ListConfiguredOutputProcessors returns only the user-configured output processors,
// excluding memory-derived processors. Useful for scenarios where memory processors
// should not be applied (e.g., network routing agents).
// Ported from TS: listConfiguredOutputProcessors(requestContext)
func (a *Agent) ListConfiguredOutputProcessors(reqCtx *requestcontext.RequestContext) ([]OutputProcessorOrWorkflow, error) {
	if a.outputProcessors == nil {
		return []OutputProcessorOrWorkflow{}, nil
	}
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	var configuredProcessors []OutputProcessorOrWorkflow
	if fn, ok := a.outputProcessors.(func(*requestcontext.RequestContext) ([]OutputProcessorOrWorkflow, error)); ok {
		procs, err := fn(reqCtx)
		if err != nil {
			return nil, err
		}
		configuredProcessors = procs
	} else if procs, ok := a.outputProcessors.([]OutputProcessorOrWorkflow); ok {
		configuredProcessors = procs
	}
	if configuredProcessors == nil {
		return []OutputProcessorOrWorkflow{}, nil
	}

	// TODO: Combine into workflow via combineProcessorsIntoWorkflow once processors package is ported.
	return configuredProcessors, nil
}

// GetConfiguredProcessorIds returns the IDs of the raw configured input and output
// processors, without combining them into workflows. Used by the editor to clone
// agent processor configuration to storage.
// Ported from TS: getConfiguredProcessorIds(requestContext)
func (a *Agent) GetConfiguredProcessorIds(reqCtx *requestcontext.RequestContext) (inputIDs []string, outputIDs []string, err error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// Resolve input processor IDs.
	if a.inputProcessors != nil {
		var processors []InputProcessorOrWorkflow
		if fn, ok := a.inputProcessors.(func(*requestcontext.RequestContext) ([]InputProcessorOrWorkflow, error)); ok {
			processors, err = fn(reqCtx)
			if err != nil {
				return nil, nil, err
			}
		} else if procs, ok := a.inputProcessors.([]InputProcessorOrWorkflow); ok {
			processors = procs
		}
		for _, p := range processors {
			if proc, ok := p.(Processor); ok {
				if id := proc.GetID(); id != "" {
					inputIDs = append(inputIDs, id)
				}
			}
		}
	}

	// Resolve output processor IDs.
	if a.outputProcessors != nil {
		var processors []OutputProcessorOrWorkflow
		if fn, ok := a.outputProcessors.(func(*requestcontext.RequestContext) ([]OutputProcessorOrWorkflow, error)); ok {
			processors, err = fn(reqCtx)
			if err != nil {
				return nil, nil, err
			}
		} else if procs, ok := a.outputProcessors.([]OutputProcessorOrWorkflow); ok {
			processors = procs
		}
		for _, p := range processors {
			if proc, ok := p.(Processor); ok {
				if id := proc.GetID(); id != "" {
					outputIDs = append(outputIDs, id)
				}
			}
		}
	}

	return inputIDs, outputIDs, nil
}

// GetConfiguredProcessorWorkflows returns configured processor workflows for registration
// with Mastra. This excludes memory-derived processors to avoid triggering memory
// factory functions.
// Ported from TS: getConfiguredProcessorWorkflows()
func (a *Agent) GetConfiguredProcessorWorkflows() ([]ProcessorWorkflow, error) {
	var result []ProcessorWorkflow

	// Get input processor workflows.
	if a.inputProcessors != nil {
		var inputProcessors []InputProcessorOrWorkflow
		if fn, ok := a.inputProcessors.(func(*requestcontext.RequestContext) ([]InputProcessorOrWorkflow, error)); ok {
			procs, err := fn(requestcontext.NewRequestContext())
			if err != nil {
				return nil, err
			}
			inputProcessors = procs
		} else if procs, ok := a.inputProcessors.([]InputProcessorOrWorkflow); ok {
			inputProcessors = procs
		}

		// TODO: Combine into workflow via combineProcessorsIntoWorkflow once processors package is ported.
		// For now, check if any are already workflows.
		for _, p := range inputProcessors {
			if pw, ok := p.(ProcessorWorkflow); ok {
				result = append(result, pw)
			}
		}
	}

	// Get output processor workflows.
	if a.outputProcessors != nil {
		var outputProcessors []OutputProcessorOrWorkflow
		if fn, ok := a.outputProcessors.(func(*requestcontext.RequestContext) ([]OutputProcessorOrWorkflow, error)); ok {
			procs, err := fn(requestcontext.NewRequestContext())
			if err != nil {
				return nil, err
			}
			outputProcessors = procs
		} else if procs, ok := a.outputProcessors.([]OutputProcessorOrWorkflow); ok {
			outputProcessors = procs
		}

		// TODO: Combine into workflow via combineProcessorsIntoWorkflow once processors package is ported.
		for _, p := range outputProcessors {
			if pw, ok := p.(ProcessorWorkflow); ok {
				result = append(result, pw)
			}
		}
	}

	return result, nil
}

// ResolveProcessorById resolves a processor by its ID from both input and output processors.
// This method resolves dynamic processor functions and includes memory-derived processors.
// Returns the processor if found, nil otherwise.
// Ported from TS: resolveProcessorById(processorId, requestContext)
func (a *Agent) ResolveProcessorById(processorID string, reqCtx *requestcontext.RequestContext) (Processor, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// Get raw input processors (before combining into workflow).
	var configuredInputProcessors []InputProcessorOrWorkflow
	if a.inputProcessors != nil {
		if fn, ok := a.inputProcessors.(func(*requestcontext.RequestContext) ([]InputProcessorOrWorkflow, error)); ok {
			procs, err := fn(reqCtx)
			if err != nil {
				return nil, err
			}
			configuredInputProcessors = procs
		} else if procs, ok := a.inputProcessors.([]InputProcessorOrWorkflow); ok {
			configuredInputProcessors = procs
		}
	}

	// Get memory input processors.
	// In TS: const memoryInputProcessors = memory ? await memory.getInputProcessors(...) : [];
	mem, memErr := a.GetMemory(reqCtx)
	if memErr != nil {
		return nil, memErr
	}
	if mem != nil {
		memConfigured := make([]memory.InputProcessorOrWorkflow, len(configuredInputProcessors))
		for i, p := range configuredInputProcessors {
			memConfigured[i] = p
		}
		memProcs, err := mem.GetInputProcessors(context.Background(), memConfigured, reqCtx)
		if err != nil {
			return nil, err
		}
		for _, mp := range memProcs {
			if proc, ok := mp.(Processor); ok && proc.GetID() == processorID {
				return proc, nil
			}
		}
	}

	// Search all configured input processors.
	for _, p := range configuredInputProcessors {
		if proc, ok := p.(Processor); ok && proc.GetID() == processorID {
			return proc, nil
		}
	}

	// Get raw output processors (before combining into workflow).
	var configuredOutputProcessors []OutputProcessorOrWorkflow
	if a.outputProcessors != nil {
		if fn, ok := a.outputProcessors.(func(*requestcontext.RequestContext) ([]OutputProcessorOrWorkflow, error)); ok {
			procs, err := fn(reqCtx)
			if err != nil {
				return nil, err
			}
			configuredOutputProcessors = procs
		} else if procs, ok := a.outputProcessors.([]OutputProcessorOrWorkflow); ok {
			configuredOutputProcessors = procs
		}
	}

	// Get memory output processors.
	// In TS: const memoryOutputProcessors = memory ? await memory.getOutputProcessors(...) : [];
	if mem != nil {
		memConfigured := make([]memory.OutputProcessorOrWorkflow, len(configuredOutputProcessors))
		for i, p := range configuredOutputProcessors {
			memConfigured[i] = p
		}
		memProcs, err := mem.GetOutputProcessors(context.Background(), memConfigured, reqCtx)
		if err != nil {
			return nil, err
		}
		for _, mp := range memProcs {
			if proc, ok := mp.(Processor); ok && proc.GetID() == processorID {
				return proc, nil
			}
		}
	}

	// Search all configured output processors.
	for _, p := range configuredOutputProcessors {
		if proc, ok := p.(Processor); ok && proc.GetID() == processorID {
			return proc, nil
		}
	}

	return nil, nil
}

// getSkillsProcessors returns skills processors to add to input processors
// when workspace has skills configured. Checks for existing SkillsProcessor
// in configured processors to avoid duplicates.
// Ported from TS: getSkillsProcessors(configuredProcessors, requestContext)
func (a *Agent) getSkillsProcessors(configuredProcessors []InputProcessorOrWorkflow, reqCtx *requestcontext.RequestContext) ([]InputProcessorOrWorkflow, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// Check if workspace has skills configured.
	workspace, err := a.GetWorkspace(reqCtx)
	if err != nil {
		return nil, err
	}
	// In TS: if (!workspace?.skills) return [];
	if workspace == nil {
		return nil, nil
	}

	// Check for existing SkillsProcessor in configured processors to avoid duplicates.
	for _, p := range configuredProcessors {
		if proc, ok := p.(Processor); ok && proc.GetID() == "skills-processor" {
			return nil, nil
		}
	}

	// Check if workspace has skills configured.
	// AnyWorkspace is *workspace.Workspace (concrete type), so call Skills() directly.
	skills := workspace.Skills()
	if skills == nil {
		return nil, nil
	}

	// Cannot create SkillsProcessor directly because processors/processors package
	// uses its own local Workspace interface stub (not workspace.Workspace).
	// The SkillsProcessor needs a Workspace that satisfies its local interface,
	// which would require the workspace to implement its Skills() method matching
	// the processors/processors.Workspace interface.
	// For now, return nil until the interface alignment is resolved.
	// TODO: Wire once processors/processors.Workspace interface matches workspace.Workspace.
	return nil, nil
}

// getWorkspaceInstructionsProcessors returns workspace-instructions processors to add
// when the workspace has a filesystem or sandbox.
// Ported from TS: getWorkspaceInstructionsProcessors(configuredProcessors, requestContext)
func (a *Agent) getWorkspaceInstructionsProcessors(configuredProcessors []InputProcessorOrWorkflow, reqCtx *requestcontext.RequestContext) ([]InputProcessorOrWorkflow, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	workspace, err := a.GetWorkspace(reqCtx)
	if err != nil {
		return nil, err
	}
	if workspace == nil {
		return nil, nil
	}

	// Check if workspace has filesystem or sandbox configured.
	// In TS: if (!workspace.filesystem && !workspace.sandbox) return [];
	// AnyWorkspace is *workspace.Workspace (concrete type), so call methods directly.
	if workspace.Filesystem() == nil && workspace.Sandbox() == nil {
		return nil, nil
	}

	// Check for existing processor to avoid duplicates.
	for _, p := range configuredProcessors {
		if proc, ok := p.(Processor); ok && proc.GetID() == "workspace-instructions-processor" {
			return nil, nil
		}
	}

	// Cannot create WorkspaceInstructionsProcessor directly because
	// processors/processors package uses its own local AnyWorkspace interface stub
	// (with GetInstructions(opts map[string]any) string) which workspace.Workspace
	// may not directly satisfy. The interface alignment needs to be resolved.
	// TODO: Wire once processors/processors.AnyWorkspace interface matches workspace.Workspace.
	return nil, nil
}

// getProcessorRunner creates and returns a ProcessorRunner with resolved
// input/output processors.
// Ported from TS: getProcessorRunner({ requestContext, inputProcessorOverrides, ... })
func (a *Agent) getProcessorRunner(
	reqCtx *requestcontext.RequestContext,
	inputProcessorOverrides []InputProcessorOrWorkflow,
	outputProcessorOverrides []OutputProcessorOrWorkflow,
	processorStates map[string]ProcessorState,
) (*ProcessorRunner, error) {
	// Resolve processors - overrides replace user-configured but auto-derived (memory, skills) are kept.
	inputProcessors, err := a.ListResolvedInputProcessors(reqCtx, inputProcessorOverrides)
	if err != nil {
		return nil, err
	}
	outputProcessors, err := a.ListResolvedOutputProcessors(reqCtx, outputProcessorOverrides)
	if err != nil {
		return nil, err
	}

	// TODO: Create full ProcessorRunner with logger and agentName once processors/runner package is ported.
	// In TS:
	//   return new ProcessorRunner({
	//     inputProcessors, outputProcessors, logger: this.logger,
	//     agentName: this.name, processorStates,
	//   });
	_ = processorStates
	return &ProcessorRunner{
		InputProcessors:  inputProcessors,
		OutputProcessors: outputProcessors,
	}, nil
}

// combineProcessorsIntoWorkflow combines multiple processors into a single workflow.
// Each processor becomes a step in the workflow, chained together.
// If there's only one item and it's already a workflow, returns it as-is.
// Ported from TS: combineProcessorsIntoWorkflow(processors, workflowId)
//
// TODO: Implement full workflow creation once the processors/workflow packages are ported.
// For now, returns the processors as-is without combining them into a workflow.
func (a *Agent) combineProcessorsIntoWorkflow(processors []InputProcessorOrWorkflow, workflowID string) []InputProcessorOrWorkflow {
	if len(processors) == 0 {
		return nil
	}

	// TODO: Full implementation should:
	// 1. If single item is a workflow, mark as processor type and return
	// 2. Filter invalid processors
	// 3. Create a single workflow with all processors chained
	// Currently returns processors as-is until workflow/processors packages are ported.
	_ = workflowID
	return processors
}

// combineOutputProcessorsIntoWorkflow is the OutputProcessorOrWorkflow variant of
// combineProcessorsIntoWorkflow.
// Ported from TS: combineProcessorsIntoWorkflow (generic over Input/Output)
func (a *Agent) combineOutputProcessorsIntoWorkflow(processors []OutputProcessorOrWorkflow, workflowID string) []OutputProcessorOrWorkflow {
	if len(processors) == 0 {
		return nil
	}

	// TODO: Full implementation once workflow/processors packages are ported.
	_ = workflowID
	return processors
}

// ---------------------------------------------------------------------------
// Scorer methods
// ---------------------------------------------------------------------------

// RunScorers runs configured scorers against the agent's message history.
// Ported from TS: #runScorers({ messageList, runId, requestContext, ... })
func (a *Agent) RunScorers(
	runID string,
	reqCtx *requestcontext.RequestContext,
	structuredOutput bool,
	overrideScorers any,
) error {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// TODO: Implement full scorer execution once the evals package is ported.
	// In TS, this:
	// 1. Resolves scorers (override or agent-configured)
	// 2. Builds scorer input from message list
	// 3. Runs each scorer via runScorer()
	_ = runID
	_ = structuredOutput
	_ = overrideScorers

	return nil
}

// ResolveOverrideScorerReferences resolves scorer name references to actual scorer instances.
// Ported from TS: resolveOverrideScorerReferences(overrideScorers)
//
// If scorers are provided as name strings, they are resolved from the Mastra instance.
// If they are direct scorer objects, they are used as-is.
func (a *Agent) ResolveOverrideScorerReferences(overrideScorers any) (map[string]ScorerWithSampling, error) {
	result := make(map[string]ScorerWithSampling)

	switch scorers := overrideScorers.(type) {
	case map[string]ScorerWithSampling:
		for id, scorerObj := range scorers {
			// If the scorer is a string reference, resolve from Mastra.
			if scorerName, ok := scorerObj.Scorer.(string); ok {
				if a.mastra == nil {
					return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
						ID:       "AGENT_GENERATE_SCORER_NOT_FOUND",
						Domain:   mastraerror.ErrorDomainAgent,
						Category: mastraerror.ErrorCategoryUser,
						Text:     "Mastra not found when fetching scorer. Make sure to fetch agent from mastra.getAgent()",
					})
				}
				// TODO: Resolve scorer by name from Mastra once getScorerById is available.
				_ = scorerName
				a.Logger().Warn(fmt.Sprintf("[Agent:%s] - Failed to resolve scorer reference %q", a.AgentName, scorerName))
				continue
			}
			result[id] = scorerObj
		}
	case []MastraScorer:
		for _, scorer := range scorers {
			result[scorer.ID()] = ScorerWithSampling{Scorer: scorer}
		}
	}

	if len(result) == 0 && overrideScorers != nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_GENERATE_SCORER_NOT_FOUND",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "No scorers found in overrideScorers",
		})
	}

	return result, nil
}

// ScorerWithSampling pairs a scorer with optional sampling configuration.
type ScorerWithSampling struct {
	Scorer   any                  `json:"scorer"`
	Sampling *ScoringSamplingConfig `json:"sampling,omitempty"`
}

// ---------------------------------------------------------------------------
// ExecuteOnFinish
// ---------------------------------------------------------------------------

// ExecuteOnFinish handles post-execution tasks including memory persistence
// and title generation.
// Ported from TS: #executeOnFinish({ result, readOnlyMemory, thread, ... })
func (a *Agent) ExecuteOnFinish(opts AgentExecuteOnFinishOptions) error {
	a.Logger().Debug(fmt.Sprintf("[Agent:%s] - Post processing LLM response", a.AgentName),
		"runId", opts.RunID,
		"threadId", opts.ThreadID,
		"resourceId", opts.ResourceID,
	)

	// Get memory instance.
	mem, err := a.GetMemory(opts.RequestContext)
	if err != nil {
		return err
	}

	thread := opts.Thread

	// If memory is configured and not read-only, handle persistence.
	if mem != nil && opts.ResourceID != "" && thread != nil && !opts.ReadOnlyMemory {
		// TODO: Add LLM response messages to messageList once MessageList is ported.
		// In TS: messageList.add(result.response.messages, 'response')

		// Create thread if it doesn't exist.
		if !opts.ThreadExists {
			saveThread := true
			_, createErr := mem.CreateThread(context.Background(), memory.CreateThreadOpts{
				ThreadID:     thread.ID,
				ResourceID:   thread.ResourceID,
				Title:        thread.Title,
				Metadata:     thread.Metadata,
				MemoryConfig: opts.MemoryConfig,
				SaveThread:   &saveThread,
			})
			if createErr != nil {
				return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "AGENT_MEMORY_PERSIST_RESPONSE_MESSAGES_FAILED",
					Domain:   mastraerror.ErrorDomainAgent,
					Category: mastraerror.ErrorCategorySystem,
					Details: map[string]any{
						"agentName": a.AgentName,
						"runId":     opts.RunID,
						"threadId":  opts.ThreadID,
					},
					Text: fmt.Sprintf("Failed to create thread: %v", createErr),
				})
			}
		}

		// Title generation.
		// In TS: config = memory.getMergedThreadConfig(memoryConfig)
		// {shouldGenerate, model, instructions} = resolveTitleGenerationConfig(config.generateTitle)
		// if (shouldGenerate && !thread.title) { generate and save }
		if thread.Title == "" {
			config := a.ResolveTitleGenerationConfig(nil)
			if config.ShouldGenerate {
				title := a.GenTitle(nil, opts.RequestContext, config.Model, config.Instructions)
				if title != "" {
					saveThread := true
					mem.CreateThread(context.Background(), memory.CreateThreadOpts{
						ThreadID:     thread.ID,
						ResourceID:   opts.ResourceID,
						MemoryConfig: opts.MemoryConfig,
						Title:        title,
						Metadata:     thread.Metadata,
						SaveThread:   &saveThread,
					})
				}
			}
		}
	}

	// Run scorers.
	// TODO: Wire scorer execution once MessageList is available to build scorer input.
	// In TS: this.#runScorers({messageList, runId, requestContext, structuredOutput, overrideScorers})

	// End agent span.
	if opts.AgentSpan != nil {
		// TODO: Call agentSpan.End({output: {text, object, files}}) once Span interface has End.
	}

	return nil
}

// SaveStepMessages adds response messages from a step to the MessageList.
// Ported from TS: saveStepMessages({ result, messageList, runId })
func (a *Agent) SaveStepMessages(result any, messageList any, runID string) error {
	// TODO: Implement once MessageList has Add method.
	// In TS: messageList.add(result.response.messages, 'response')
	_ = result
	_ = messageList
	_ = runID
	return nil
}

// GetMemoryMessages retrieves memory messages for the agent.
// Ported from TS: getMemoryMessages({ resourceId, threadId, vectorMessageSearch, memoryConfig, requestContext })
func (a *Agent) GetMemoryMessages(reqCtx *requestcontext.RequestContext, threadID string, resourceID string) ([]any, error) {
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	mem, err := a.GetMemory(reqCtx)
	if err != nil {
		return nil, err
	}
	if mem == nil {
		return nil, nil
	}

	// In TS: const threadConfig = memory.getMergedThreadConfig(memoryConfig || {});
	// The memory's Recall method handles the merged config internally.
	result, err := mem.Recall(context.Background(), memory.RecallArgs{
		StorageListMessagesInput: map[string]any{
			"threadId":   threadID,
			"resourceId": resourceID,
		},
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	// Convert []MastraDBMessage to []any.
	messages := make([]any, len(result.Messages))
	for i, m := range result.Messages {
		messages[i] = m
	}
	return messages, nil
}

// ---------------------------------------------------------------------------
// Private / internal helpers
// ---------------------------------------------------------------------------

// validateRequestContext validates the request context against the agent's
// requestContextSchema, if one is configured.
// Ported from TS: #validateRequestContext(requestContext)
//
// In the TS source, this uses Zod's safeParseAsync to validate context values.
// In Go, we check if a requestContextSchema is configured and if so, use it
// to validate. Since Zod schemas are not available in Go, the validation
// is delegated to whatever schema validator is assigned.
func (a *Agent) validateRequestContext(reqCtx *requestcontext.RequestContext) error {
	if a.requestContextSchema == nil {
		return nil
	}

	// If a validator function is provided, use it.
	if validator, ok := a.requestContextSchema.(func(map[string]any) error); ok {
		var contextValues map[string]any
		if reqCtx != nil {
			contextValues = reqCtx.All()
		} else {
			contextValues = map[string]any{}
		}
		if err := validator(contextValues); err != nil {
			return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "AGENT_REQUEST_CONTEXT_VALIDATION_FAILED",
				Domain:   mastraerror.ErrorDomainAgent,
				Category: mastraerror.ErrorCategoryUser,
				Details: map[string]any{
					"agentId":   a.ID,
					"agentName": a.AgentName,
				},
				Text: fmt.Sprintf("Request context validation failed for agent %q: %v", a.ID, err),
			})
		}
	}

	// TODO: Add Zod-like schema validation once a Go schema validation
	// library is integrated. For now, non-function schemas are silently accepted.
	return nil
}

// mergeExecutionOptions performs a shallow merge of default options with
// call-specific options, where call-specific values take precedence.
// Ported from TS: deepMerge(defaultOptions, options)
//
// In the TypeScript source, this is a generic deepMerge utility. In Go, we
// perform a field-by-field merge where non-zero call-specific values override
// defaults. This mirrors the TS behavior for the AgentExecutionOptions struct.
func mergeExecutionOptions(defaults, overrides AgentExecutionOptions) AgentExecutionOptions {
	merged := defaults

	// AgentExecutionOptionsBase fields
	if overrides.Instructions != nil {
		merged.Instructions = overrides.Instructions
	}
	if overrides.System != nil {
		merged.System = overrides.System
	}
	if len(overrides.Context) > 0 {
		merged.Context = overrides.Context
	}
	if overrides.Memory != nil {
		merged.Memory = overrides.Memory
	}
	if overrides.RunID != "" {
		merged.RunID = overrides.RunID
	}
	if overrides.SavePerStep {
		merged.SavePerStep = overrides.SavePerStep
	}
	if overrides.RequestContext != nil {
		merged.RequestContext = overrides.RequestContext
	}
	if overrides.MaxSteps != nil {
		merged.MaxSteps = overrides.MaxSteps
	}
	if overrides.StopWhen != nil {
		merged.StopWhen = overrides.StopWhen
	}
	if overrides.ProviderOptions != nil {
		merged.ProviderOptions = overrides.ProviderOptions
	}
	if overrides.OnStepFinish != nil {
		merged.OnStepFinish = overrides.OnStepFinish
	}
	if overrides.OnFinish != nil {
		merged.OnFinish = overrides.OnFinish
	}
	if overrides.OnChunk != nil {
		merged.OnChunk = overrides.OnChunk
	}
	if overrides.OnError != nil {
		merged.OnError = overrides.OnError
	}
	if overrides.OnAbort != nil {
		merged.OnAbort = overrides.OnAbort
	}
	if len(overrides.ActiveTools) > 0 {
		merged.ActiveTools = overrides.ActiveTools
	}
	if overrides.AbortSignal != nil {
		merged.AbortSignal = overrides.AbortSignal
	}
	if len(overrides.InputProcessors) > 0 {
		merged.InputProcessors = overrides.InputProcessors
	}
	if len(overrides.OutputProcessors) > 0 {
		merged.OutputProcessors = overrides.OutputProcessors
	}
	if overrides.MaxProcessorRetries != nil {
		merged.MaxProcessorRetries = overrides.MaxProcessorRetries
	}
	if overrides.Toolsets != nil {
		merged.Toolsets = overrides.Toolsets
	}
	if overrides.ClientTools != nil {
		merged.ClientTools = overrides.ClientTools
	}
	if overrides.ToolChoice != nil {
		merged.ToolChoice = overrides.ToolChoice
	}
	if overrides.ModelSettings != nil {
		merged.ModelSettings = overrides.ModelSettings
	}
	if overrides.Scorers != nil {
		merged.Scorers = overrides.Scorers
	}
	if overrides.ReturnScorerData {
		merged.ReturnScorerData = overrides.ReturnScorerData
	}
	if overrides.TracingOptions != nil {
		merged.TracingOptions = overrides.TracingOptions
	}
	if overrides.PrepareStep != nil {
		merged.PrepareStep = overrides.PrepareStep
	}
	if overrides.IsTaskComplete != nil {
		merged.IsTaskComplete = overrides.IsTaskComplete
	}
	if overrides.RequireToolApproval {
		merged.RequireToolApproval = overrides.RequireToolApproval
	}
	if overrides.AutoResumeSuspendedTools {
		merged.AutoResumeSuspendedTools = overrides.AutoResumeSuspendedTools
	}
	if overrides.ToolCallConcurrency != nil {
		merged.ToolCallConcurrency = overrides.ToolCallConcurrency
	}
	if overrides.IncludeRawChunks {
		merged.IncludeRawChunks = overrides.IncludeRawChunks
	}
	if overrides.OnIterationComplete != nil {
		merged.OnIterationComplete = overrides.OnIterationComplete
	}
	if overrides.Delegation != nil {
		merged.Delegation = overrides.Delegation
	}

	// AgentExecutionOptions-specific fields
	if overrides.StructuredOutput != nil {
		merged.StructuredOutput = overrides.StructuredOutput
	}

	return merged
}

// execute is the core vNext execution pipeline.
// Ported from TS: #execute({ methodType, resumeContext, ...options })
//
// This method:
// 1. Resolves snapshot memory info from resume context (if resuming).
// 2. Resolves thread/resource IDs (reserved requestContext keys take precedence).
// 3. Resolves the LLM and creates the capabilities object.
// 4. Creates the prepare-stream workflow with all context.
// 5. Executes the workflow (parallel tool prep + memory prep -> map -> stream).
// 6. Returns the result.
func (a *Agent) execute(ctx context.Context, opts InnerAgentExecutionOptions) (*FullOutput, error) {
	_ = ctx // reserved for context.Context propagation

	methodType := opts.MethodType
	resumeContext := opts.ResumeContext

	// Resolve snapshot memory info for resume scenarios.
	var snapshotMemoryThreadID, snapshotMemoryResourceID string
	if resumeContext != nil && resumeContext.Snapshot != nil {
		if snapshot, ok := resumeContext.Snapshot.(map[string]any); ok {
			if ctxMap, ok := snapshot["context"].(map[string]any); ok {
				for _, step := range ctxMap {
					if stepMap, ok := step.(map[string]any); ok {
						if status, _ := stepMap["status"].(string); status == "suspended" {
							if payload, ok := stepMap["suspendPayload"].(map[string]any); ok {
								if streamState, ok := payload["__streamState"].(map[string]any); ok {
									if msgList, ok := streamState["messageList"].(map[string]any); ok {
										if memInfo, ok := msgList["memoryInfo"].(map[string]any); ok {
											snapshotMemoryThreadID, _ = memInfo["threadId"].(string)
											snapshotMemoryResourceID, _ = memInfo["resourceId"].(string)
										}
									}
								}
							}
							break
						}
					}
				}
			}
		}
	}

	reqCtx := opts.RequestContext
	if reqCtx == nil {
		reqCtx = requestcontext.NewRequestContext()
	}

	// Reserved keys from requestContext take precedence for security.
	// This allows middleware to securely set resourceId/threadId based on authenticated user,
	// preventing attackers from hijacking another user's memory by passing different values in the body.
	resourceIDFromContext, _ := reqCtx.Get(requestcontext.MastraResourceIDKey).(string)
	threadIDFromContext, _ := reqCtx.Get(requestcontext.MastraThreadIDKey).(string)

	var threadFromArgs *StorageThreadType
	if threadIDFromContext != "" {
		threadFromArgs = &StorageThreadType{ID: threadIDFromContext}
	} else {
		// Build memory option with snapshot fallback for thread.
		memOpt := opts.Memory
		if memOpt == nil && snapshotMemoryThreadID != "" {
			memOpt = &AgentMemoryOption{Thread: snapshotMemoryThreadID}
		} else if memOpt != nil && memOpt.Thread == nil && snapshotMemoryThreadID != "" {
			memCopy := *memOpt
			memCopy.Thread = snapshotMemoryThreadID
			memOpt = &memCopy
		}
		resolved := ResolveThreadIdFromArgs(ResolveThreadArgs{Memory: memOpt})
		if resolved != nil {
			threadFromArgs = &resolved.StorageThreadType
		}
	}

	var resourceID string
	if resourceIDFromContext != "" {
		resourceID = resourceIDFromContext
	} else if opts.Memory != nil && opts.Memory.Resource != "" {
		resourceID = opts.Memory.Resource
	} else if snapshotMemoryResourceID != "" {
		resourceID = snapshotMemoryResourceID
	}

	var memoryConfig *MemoryConfig
	if opts.Memory != nil && opts.Memory.Options != nil {
		memoryConfig = opts.Memory.Options
	}

	// Warn if memory args provided but no memory configured.
	if resourceID != "" && threadFromArgs != nil && !a.HasOwnMemory() {
		a.Logger().Warn(
			fmt.Sprintf("[Agent:%s] - No memory is configured but resourceId and threadId were passed in args. This will not work.", a.AgentName),
		)
	}

	// Resolve the LLM.
	llm, err := a.GetLLM(reqCtx, opts.Model)
	if err != nil {
		return nil, err
	}

	// Generate run ID.
	runID := opts.RunID
	if runID == "" {
		runID = uuid.New().String()
	}

	// Resolve instructions.
	instructions := opts.Instructions
	if instructions == nil {
		instructions, err = a.GetInstructions(ctx, reqCtx)
		if err != nil {
			return nil, err
		}
	}

	// TODO: Set tracing context / create agent span once observability package is integrated.
	// The TS code calls getOrCreateSpan({ type: SpanType.AGENT_RUN, ... })
	var agentSpan Span

	// Resolve memory and workspace.
	memory, err := a.GetMemory(reqCtx)
	if err != nil {
		return nil, err
	}

	workspace, err := a.GetWorkspace(reqCtx)
	if err != nil {
		return nil, err
	}

	// TODO: Create SaveQueueManager once the savequeue package is integrated.

	a.Logger().Debug(fmt.Sprintf("[Agents:%s] - Starting generation", a.AgentName), "runId", runID)

	// Cast LLM to MastraLLMVNext as required by the workflow capabilities.
	// In the TS source: `const llm = (await this.getLLM(...)) as MastraLLMVNext`
	llmVNext, ok := llm.(workflows.MastraLLMVNext)
	if !ok {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_EXECUTE_LLM_NOT_VNEXT",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategorySystem,
			Details:  map[string]any{"agentName": a.AgentName},
			Text:     fmt.Sprintf("[Agent:%s] - LLM does not satisfy MastraLLMVNext interface", a.AgentName),
		})
	}

	// Create a capabilities object with bound methods.
	// This mirrors the TS capabilities object that is passed to createPrepareStreamWorkflow.
	capabilities := workflows.AgentCapabilities{
		AgentName:          a.AgentName,
		Logger:             a.Logger(),
		AgentNetworkAppend: a.agentNetworkAppend,
		LLM:                llmVNext,
		ConvertTools: func(args any) (map[string]any, error) {
			// Convert the generic args to ConvertToolsParams.
			if params, ok := args.(ConvertToolsParams); ok {
				result, err := a.ConvertTools(ctx, params)
				if err != nil {
					return nil, err
				}
				out := make(map[string]any, len(result))
				for k, v := range result {
					out[k] = v
				}
				return out, nil
			}
			if m, ok := args.(map[string]any); ok {
				return m, nil
			}
			return map[string]any{}, nil
		},
		GetMemory: func() any {
			mem, _ := a.GetMemory(reqCtx)
			return mem
		},
		GetModel: func(modelOpts any) (any, error) {
			return a.GetModel(reqCtx, modelOpts)
		},
		GenerateMessageID: func() string {
			return uuid.New().String()
		},
		SaveStepMessages: func(args any) error {
			if m, ok := args.(map[string]any); ok {
				return a.SaveStepMessages(m["result"], m["messageList"], m["runId"].(string))
			}
			return nil
		},
		ExecuteOnFinish: func(args workflows.AgentExecuteOnFinishOptions) error {
			// workflows.AgentExecuteOnFinishOptions = any, cast to our type.
			if opts, ok := args.(AgentExecuteOnFinishOptions); ok {
				return a.ExecuteOnFinish(opts)
			}
			return nil
		},
		RunInputProcessors: func(args any) (workflows.InputProcessorsResult, error) {
			// TODO: Wire once full processor runner is implemented.
			return workflows.InputProcessorsResult{}, nil
		},
		OutputProcessors: a.outputProcessors,
		InputProcessors:  a.inputProcessors,
	}

	// Determine toolCallConcurrency default.
	toolCallConcurrency := 10
	if opts.ToolCallConcurrency != nil {
		toolCallConcurrency = *opts.ToolCallConcurrency
	} else if opts.RequireToolApproval {
		toolCallConcurrency = 1
	}

	// Convert agent InnerAgentExecutionOptions to the workflow's stub type.
	var workflowMemory *workflows.MemoryWrapper
	if opts.Memory != nil {
		workflowMemory = &workflows.MemoryWrapper{
			Options: opts.Memory.Options,
		}
	}
	// Convert ToolsetsInput (map[string]map[string]any) to map[string]any for the workflow.
	var wfToolsets map[string]any
	if opts.Toolsets != nil {
		wfToolsets = make(map[string]any, len(opts.Toolsets))
		for k, v := range opts.Toolsets {
			wfToolsets[k] = v
		}
	}
	workflowOpts := workflows.InnerAgentExecutionOptions{
		Toolsets:                wfToolsets,
		ClientTools:            opts.ClientTools,
		Memory:                 workflowMemory,
		OutputWriter:           opts.OutputWriter,
		AutoResumeSuspendedTools: opts.AutoResumeSuspendedTools,
		Delegation:             opts.Delegation,
	}

	// Convert threadFromArgs to workflow type if needed.
	var wfThread *workflows.StorageThreadType
	if threadFromArgs != nil {
		wfThread = &workflows.StorageThreadType{
			ID:         threadFromArgs.ID,
			Title:      threadFromArgs.Title,
			ResourceID: threadFromArgs.ResourceID,
			Metadata:   threadFromArgs.Metadata,
		}
	}

	// Create the prepare-stream workflow with all necessary context.
	executionWorkflow := workflows.CreatePrepareStreamWorkflow(workflows.CreatePrepareStreamWorkflowOptions{
		Capabilities:        capabilities,
		Options:             workflowOpts,
		ThreadFromArgs:      wfThread,
		ResourceID:          resourceID,
		RunID:               runID,
		RequestContext:      reqCtx,
		AgentSpan:           agentSpan,
		MethodType:          workflows.AgentMethodType(methodType),
		Instructions:        instructions,
		MemoryConfig:        memoryConfig,
		Memory:              memory,
		ReturnScorerData:    opts.ReturnScorerData,
		RequireToolApproval: opts.RequireToolApproval,
		ToolCallConcurrency: toolCallConcurrency,
		ResumeContext:       resumeContextToWorkflow(resumeContext),
		AgentID:             a.ID,
		AgentName:           a.AgentName,
		ToolCallID:          opts.ToolCallID,
		Workspace:           workspace,
	})

	// Execute the workflow: parallel(prepareTools, prepareMemory) -> mapResults -> stream
	result, err := executionWorkflow.Execute()
	if err != nil {
		return nil, err
	}

	// The workflow returns a MastraModelOutput (any).
	// Convert to FullOutput for the caller.
	if result == nil {
		return &FullOutput{}, nil
	}
	if fo, ok := result.(*FullOutput); ok {
		return fo, nil
	}

	// Best-effort conversion from generic result.
	return &FullOutput{Object: result}, nil
}

// resumeContextToWorkflow converts agent ResumeContext to workflow ResumeContext.
func resumeContextToWorkflow(rc *ResumeContext) *workflows.ResumeContext {
	if rc == nil {
		return nil
	}
	return &workflows.ResumeContext{
		ResumeData: rc.ResumeData,
		Snapshot:   rc.Snapshot,
	}
}

// getLegacyHandler lazily initializes and returns the legacy handler.
func (a *Agent) getLegacyHandler() *AgentLegacyHandler {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.legacyHandler == nil {
		a.legacyHandler = NewAgentLegacyHandler(a)
	}
	return a.legacyHandler
}
