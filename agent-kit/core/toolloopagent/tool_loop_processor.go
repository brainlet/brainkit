// Ported from: packages/core/src/tool-loop-agent/tool-loop-processor.ts
package toolloopagent

import (
	"errors"
	"fmt"

	"github.com/brainlet/brainkit/agent-kit/core/agent"
	"github.com/brainlet/brainkit/agent-kit/core/llm/model"
	"github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// Type aliases wired to real packages
// ---------------------------------------------------------------------------

// MastraLanguageModel is the real model.MastraLanguageModel interface.
// No circular dependency: toolloopagent does not import any package that
// imports toolloopagent.
type MastraLanguageModel = model.MastraLanguageModel

// AgentInstructions is the real agent.AgentInstructions type (= SystemMessage = any).
type AgentInstructions = agent.AgentInstructions

// ---------------------------------------------------------------------------
// ProcessInputStepArgs — structurally compatible with processors.ProcessInputStepArgs
// ---------------------------------------------------------------------------
//
// The real processors.ProcessInputStepArgs embeds ProcessorMessageContext (which
// itself embeds ProcessorContext) and uses typed fields. We keep a local struct
// because importing the real type would require constructing ProcessorContext
// (with Abort func, RequestContext, Writer, etc.) at every call site.
// Fields are named and typed to match the real struct so that future migration
// is a search-and-replace.
type ProcessInputStepArgs struct {
	// --- ProcessorMessageContext fields (flattened) ---
	// The real type embeds ProcessorMessageContext which embeds ProcessorContext.
	// We flatten them here for simplicity while keeping the same field names.

	// Messages is the current messages being processed.
	// Real type: []processors.MastraDBMessage (state.MastraDBMessage).
	Messages []any `json:"messages,omitempty"`

	// MessageList is the MessageList instance for managing message sources.
	// Real type: *processors.MessageList.
	MessageList any `json:"messageList,omitempty"`

	// --- ProcessorContext fields ---

	// Abort aborts processing with an optional reason and options.
	Abort func(reason string, options any) error `json:"-"`

	// RequestContext holds optional runtime context with execution metadata.
	RequestContext any `json:"requestContext,omitempty"`

	// RetryCount is the number of times processors have triggered retry.
	RetryCount int `json:"retryCount,omitempty"`

	// Writer is an optional stream writer for emitting custom data chunks.
	Writer any `json:"writer,omitempty"`

	// AbortSignal is an optional context for cancellation.
	AbortSignal any `json:"-"`

	// ObservabilityContext provides tracing context for processors.
	ObservabilityContext any `json:"observabilityContext,omitempty"`

	// --- ProcessInputStepArgs own fields ---

	// StepNumber is the current step number (0-indexed).
	StepNumber int `json:"stepNumber"`

	// Steps contains results from previous steps.
	// Real type: []processors.StepResult.
	Steps []any `json:"steps,omitempty"`

	// SystemMessages contains all system messages for read/modify access.
	// Real type: []processors.CoreMessageV4.
	SystemMessages []any `json:"systemMessages,omitempty"`

	// State is per-processor state that persists across all method calls.
	State map[string]any `json:"state,omitempty"`

	// Model is the current model for this step.
	// Real type: processors.MastraLanguageModel (= model.MastraLanguageModel interface).
	// Kept as any because callers may pass unresolved configs (string, struct).
	Model any `json:"model,omitempty"`

	// Tools contains the current tools available for this step.
	Tools map[string]any `json:"tools,omitempty"`

	// ToolChoice specifies how tools should be selected.
	// Real type: processors.ToolChoice (= any).
	ToolChoice any `json:"toolChoice,omitempty"`

	// ActiveTools lists the currently active tools.
	ActiveTools []string `json:"activeTools,omitempty"`

	// ProviderOptions contains provider-specific options.
	// Real type: processors.SharedProviderOptions (= model.SharedProviderOptions = map[string]any).
	ProviderOptions map[string]any `json:"providerOptions,omitempty"`

	// ModelSettings contains model settings (temperature, etc.), excluding AbortSignal.
	// Real type: map[string]any (not a struct).
	ModelSettings map[string]any `json:"modelSettings,omitempty"`

	// StructuredOutput contains structured output configuration.
	// Real type: *processors.StructuredOutputOptions.
	StructuredOutput any `json:"structuredOutput,omitempty"`
}

// ---------------------------------------------------------------------------
// ProcessInputStepResult — structurally compatible with processors.ProcessInputStepResult
// ---------------------------------------------------------------------------
//
// Matches the real processors.ProcessInputStepResult field set and types.
type ProcessInputStepResult struct {
	Model            any            `json:"model,omitempty"`
	Tools            map[string]any `json:"tools,omitempty"`
	ToolChoice       any            `json:"toolChoice,omitempty"`
	ActiveTools      []string       `json:"activeTools,omitempty"`
	Messages         []any          `json:"messages,omitempty"`
	MessageList      any            `json:"messageList,omitempty"`
	SystemMessages   []any          `json:"systemMessages,omitempty"`
	ProviderOptions  map[string]any `json:"providerOptions,omitempty"`
	ModelSettings    map[string]any `json:"modelSettings,omitempty"`
	StructuredOutput any            `json:"structuredOutput,omitempty"`
	RetryCount       *int           `json:"retryCount,omitempty"`
}

// ---------------------------------------------------------------------------
// Processor — matches processors.Processor interface signatures
// ---------------------------------------------------------------------------
//
// Uses ID()/Name() (not GetID()/GetName()) and includes all methods from the
// real processors.Processor interface.
type Processor interface {
	// ID returns the processor's unique identifier.
	ID() string

	// Name returns the processor's optional human-readable name.
	Name() string

	// Description returns the processor's optional description.
	Description() string

	// ProcessorIndex returns the index of this processor in the workflow.
	ProcessorIndex() int

	// SetProcessorIndex sets the index of this processor in the workflow.
	SetProcessorIndex(index int)
}

// ---------------------------------------------------------------------------
// AgentExecutionOptions — structurally compatible with agent.AgentExecutionOptions
// ---------------------------------------------------------------------------
//
// The real agent.AgentExecutionOptions embeds AgentExecutionOptionsBase (20+ fields)
// plus StructuredOutput. We include the most relevant fields here while keeping
// the structure flat for backward compatibility. Fields match the real type names.
type AgentExecutionOptions struct {
	// --- AgentExecutionOptionsBase fields ---

	// Instructions to override the agent's default instructions for this execution.
	Instructions any `json:"instructions,omitempty"`

	// System is a custom system message to include in the prompt.
	System any `json:"system,omitempty"`

	// Context holds additional context messages.
	Context []any `json:"context,omitempty"`

	// Memory configures conversation persistence and retrieval.
	Memory any `json:"memory,omitempty"`

	// RunID is a unique identifier for this execution run.
	RunID string `json:"runId,omitempty"`

	// SavePerStep saves messages incrementally after each stream step completes.
	SavePerStep bool `json:"savePerStep,omitempty"`

	// RequestContext holds dynamic configuration and state.
	RequestContext any `json:"requestContext,omitempty"`

	// MaxSteps is the maximum number of steps to run.
	MaxSteps *int `json:"maxSteps,omitempty"`

	// StopWhen holds conditions for stopping execution.
	StopWhen any `json:"stopWhen,omitempty"`

	// ProviderOptions holds provider-specific options passed to the language model.
	// Real type: agent.ProviderOptions (= map[string]map[string]any).
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`

	// OnStepFinish is called after each execution step.
	OnStepFinish any `json:"-"`
	// OnFinish is called when execution completes.
	OnFinish any `json:"-"`
	// OnChunk is called for each streaming chunk received.
	OnChunk any `json:"-"`
	// OnError is called when an error occurs during streaming.
	OnError func(err error) error `json:"-"`
	// OnAbort is called when streaming is aborted.
	OnAbort func(event any) error `json:"-"`

	// ActiveTools lists tools that are active for this execution.
	ActiveTools []string `json:"activeTools,omitempty"`

	// AbortSignal to abort the streaming operation.
	AbortSignal any `json:"-"`

	// InputProcessors to use for this execution (overrides agent's default).
	InputProcessors []any `json:"inputProcessors,omitempty"`
	// OutputProcessors to use for this execution (overrides agent's default).
	OutputProcessors []any `json:"outputProcessors,omitempty"`
	// MaxProcessorRetries overrides agent's default maxProcessorRetries.
	MaxProcessorRetries *int `json:"maxProcessorRetries,omitempty"`

	// Toolsets are additional tool sets for this execution.
	Toolsets any `json:"toolsets,omitempty"`
	// ClientTools are client-side tools available during execution.
	ClientTools any `json:"clientTools,omitempty"`

	// ToolChoice controls tool selection strategy.
	ToolChoice any `json:"toolChoice,omitempty"`

	// ModelSettings holds model-specific settings like temperature, maxTokens, topP, etc.
	ModelSettings any `json:"modelSettings,omitempty"`

	// Scorers are evaluation scorers to run on the execution results.
	Scorers any `json:"scorers,omitempty"`
	// ReturnScorerData indicates whether to return detailed scoring data.
	ReturnScorerData bool `json:"returnScorerData,omitempty"`
	// TracingOptions for starting new traces.
	TracingOptions any `json:"tracingOptions,omitempty"`

	// PrepareStep is a callback function called before each step.
	PrepareStep any `json:"-"`

	// IsTaskComplete is the scoring configuration for supervisor patterns.
	IsTaskComplete any `json:"isTaskComplete,omitempty"`

	// RequireToolApproval requires approval for all tool calls.
	RequireToolApproval bool `json:"requireToolApproval,omitempty"`

	// AutoResumeSuspendedTools automatically resumes suspended tools.
	AutoResumeSuspendedTools bool `json:"autoResumeSuspendedTools,omitempty"`

	// ToolCallConcurrency is the maximum number of concurrent tool calls.
	ToolCallConcurrency *int `json:"toolCallConcurrency,omitempty"`

	// IncludeRawChunks includes raw chunks in the stream output.
	IncludeRawChunks bool `json:"includeRawChunks,omitempty"`

	// OnIterationComplete is called after each iteration completes.
	OnIterationComplete any `json:"-"`

	// Delegation configures sub-agent and workflow tool call delegation.
	Delegation any `json:"delegation,omitempty"`

	// --- AgentExecutionOptions own field ---

	// StructuredOutput configures structured output for this execution.
	StructuredOutput any `json:"structuredOutput,omitempty"`
}

// ---------------------------------------------------------------------------
// resolveModelConfig — delegates to model.ResolveModelConfig
// ---------------------------------------------------------------------------

func resolveModelConfig(modelCfg any) (any, error) {
	return model.ResolveModelConfig(modelCfg, nil)
}

// ---------------------------------------------------------------------------
// isSupportedLanguageModel — delegates to agent.IsSupportedLanguageModel
// ---------------------------------------------------------------------------

func isSupportedLanguageModel(m any) bool {
	lm, ok := m.(agent.LanguageModelLike)
	if !ok {
		return false
	}
	return agent.IsSupportedLanguageModel(lm)
}

// ---------------------------------------------------------------------------
// modelSettingsToMap / modelSettingsFromMap helpers
// ---------------------------------------------------------------------------
//
// Since the real ProcessInputStepResult.ModelSettings is map[string]any and
// the ToolLoopAgent settings use individual typed fields, we provide helpers
// to convert between the two representations.

// modelSettingsToMap converts individual model setting fields into map[string]any.
func modelSettingsToMap(
	temperature *float64,
	topP *float64,
	topK *int,
	seed *int,
	maxOutputTokens *int,
	presencePenalty *float64,
	frequencyPenalty *float64,
	stopSequences []string,
) map[string]any {
	m := make(map[string]any)
	if temperature != nil {
		m["temperature"] = *temperature
	}
	if topP != nil {
		m["topP"] = *topP
	}
	if topK != nil {
		m["topK"] = *topK
	}
	if seed != nil {
		m["seed"] = *seed
	}
	if maxOutputTokens != nil {
		m["maxOutputTokens"] = *maxOutputTokens
	}
	if presencePenalty != nil {
		m["presencePenalty"] = *presencePenalty
	}
	if frequencyPenalty != nil {
		m["frequencyPenalty"] = *frequencyPenalty
	}
	if stopSequences != nil {
		m["stopSequences"] = stopSequences
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// modelSettingsGetFloat64 safely extracts a *float64 from model settings map.
func modelSettingsGetFloat64(ms map[string]any, key string) *float64 {
	if ms == nil {
		return nil
	}
	v, ok := ms[key]
	if !ok || v == nil {
		return nil
	}
	switch f := v.(type) {
	case float64:
		return &f
	case int:
		fv := float64(f)
		return &fv
	}
	return nil
}

// modelSettingsGetInt safely extracts a *int from model settings map.
func modelSettingsGetInt(ms map[string]any, key string) *int {
	if ms == nil {
		return nil
	}
	v, ok := ms[key]
	if !ok || v == nil {
		return nil
	}
	switch i := v.(type) {
	case int:
		return &i
	case float64:
		iv := int(i)
		return &iv
	}
	return nil
}

// modelSettingsGetStringSlice safely extracts a []string from model settings map.
func modelSettingsGetStringSlice(ms map[string]any, key string) []string {
	if ms == nil {
		return nil
	}
	v, ok := ms[key]
	if !ok || v == nil {
		return nil
	}
	if ss, ok := v.([]string); ok {
		return ss
	}
	return nil
}

// Compile-time proof that we reference the processors package to avoid
// an "imported and not used" error, while documenting the type contract.
var _ = (processors.ProcessInputStepResult)(processors.ProcessInputStepResult{})

// ---------------------------------------------------------------------------
// PrepareCallInput
// ---------------------------------------------------------------------------

// PrepareCallInput combines AgentCallParameters with picked ToolLoopAgentSettings fields.
// Mirrors the TS type: AgentCallParameters<never> & Pick<ToolLoopAgentSettings, ...>.
type PrepareCallInput struct {
	Messages         []any                     `json:"messages,omitempty"`
	Model            any                       `json:"model,omitempty"`
	Tools            map[string]any            `json:"tools,omitempty"`
	Instructions     any                       `json:"instructions,omitempty"`
	StopWhen         any                       `json:"stopWhen,omitempty"`
	ActiveTools      []string                  `json:"activeTools,omitempty"`
	ProviderOptions  map[string]map[string]any `json:"providerOptions,omitempty"`
	MaxOutputTokens  *int                      `json:"maxOutputTokens,omitempty"`
	Temperature      *float64                  `json:"temperature,omitempty"`
	TopP             *float64                  `json:"topP,omitempty"`
	TopK             *int                      `json:"topK,omitempty"`
	PresencePenalty  *float64                  `json:"presencePenalty,omitempty"`
	FrequencyPenalty *float64                  `json:"frequencyPenalty,omitempty"`
	StopSequences    []string                  `json:"stopSequences,omitempty"`
	Seed             *int                      `json:"seed,omitempty"`
	Headers          map[string]string         `json:"headers,omitempty"`
}

// PrepareStepInput is the argument passed to ToolLoopAgentSettings.PrepareStep.
type PrepareStepInput struct {
	Steps               []any `json:"steps,omitempty"`
	StepNumber          int   `json:"stepNumber"`
	Model               any   `json:"model,omitempty"`
	Messages            []any `json:"messages,omitempty"`
	ExperimentalContext any   `json:"experimental_context,omitempty"`
}

// ---------------------------------------------------------------------------
// AgentConfig (returned by GetAgentConfig)
// ---------------------------------------------------------------------------

// AgentConfig holds the configuration extracted from a ToolLoopAgent for
// constructing a Mastra Agent.
type AgentConfig struct {
	ID             string                 `json:"id,omitempty"`
	Name           string                 `json:"name,omitempty"`
	Instructions   AgentInstructions      `json:"instructions,omitempty"`
	Model          any                    `json:"model,omitempty"`
	Tools          map[string]any         `json:"tools,omitempty"`
	MaxRetries     *int                   `json:"maxRetries,omitempty"`
	DefaultOptions *AgentExecutionOptions `json:"defaultOptions,omitempty"`
}

// ---------------------------------------------------------------------------
// ToolLoopAgentProcessor
// ---------------------------------------------------------------------------

// ToolLoopAgentProcessor implements the Processor interface to adapt a
// ToolLoopAgent (AI SDK v6) into the Mastra processor pipeline.
type ToolLoopAgentProcessor struct {
	id             string
	name           string
	description    string
	processorIndex int

	agent             ToolLoopAgentLike
	settings          *ToolLoopAgentSettings
	prepareCallResult map[string]any
}

// Ensure ToolLoopAgentProcessor satisfies Processor.
var _ Processor = (*ToolLoopAgentProcessor)(nil)

// NewToolLoopAgentProcessor creates a new ToolLoopAgentProcessor from a ToolLoopAgentLike.
func NewToolLoopAgentProcessor(agent ToolLoopAgentLike) (*ToolLoopAgentProcessor, error) {
	settings, err := GetSettings(agent)
	if err != nil {
		return nil, fmt.Errorf("NewToolLoopAgentProcessor: %w", err)
	}
	return &ToolLoopAgentProcessor{
		id:       "tool-loop-agent-processor",
		name:     "ToolLoop to Mastra Agent Processor",
		agent:    agent,
		settings: settings,
	}, nil
}

// ID implements Processor.
func (p *ToolLoopAgentProcessor) ID() string { return p.id }

// Name implements Processor.
func (p *ToolLoopAgentProcessor) Name() string { return p.name }

// Description implements Processor.
func (p *ToolLoopAgentProcessor) Description() string { return p.description }

// ProcessorIndex implements Processor.
func (p *ToolLoopAgentProcessor) ProcessorIndex() int { return p.processorIndex }

// SetProcessorIndex implements Processor.
func (p *ToolLoopAgentProcessor) SetProcessorIndex(index int) { p.processorIndex = index }

// ---------------------------------------------------------------------------
// GetAgentConfig
// ---------------------------------------------------------------------------

// GetAgentConfig extracts a Mastra-compatible AgentConfig from the ToolLoopAgent settings.
func (p *ToolLoopAgentProcessor) GetAgentConfig() *AgentConfig {
	tools := p.agent.GetTools()

	// Build default options from ToolLoopAgent config params.
	defaultOptions := &AgentExecutionOptions{}
	hasDefaults := false

	// ToolChoice
	if p.settings.ToolChoice != nil {
		defaultOptions.ToolChoice = p.settings.ToolChoice
		hasDefaults = true
	}
	// ProviderOptions
	if p.settings.ProviderOptions != nil {
		defaultOptions.ProviderOptions = p.settings.ProviderOptions
		hasDefaults = true
	}

	// Model settings — build as map[string]any to match real type.
	ms := modelSettingsToMap(
		p.settings.Temperature,
		p.settings.TopP,
		p.settings.TopK,
		p.settings.Seed,
		p.settings.MaxOutputTokens,
		p.settings.PresencePenalty,
		p.settings.FrequencyPenalty,
		p.settings.StopSequences,
	)
	if ms != nil {
		defaultOptions.ModelSettings = ms
		hasDefaults = true
	}

	// Callbacks.
	if p.settings.StopWhen != nil {
		defaultOptions.StopWhen = p.settings.StopWhen
		hasDefaults = true
	}
	if p.settings.OnStepFinish != nil {
		defaultOptions.OnStepFinish = p.settings.OnStepFinish
		hasDefaults = true
	}
	if p.settings.OnFinish != nil {
		defaultOptions.OnFinish = p.settings.OnFinish
		hasDefaults = true
	}

	var instructions AgentInstructions
	if p.settings.Instructions != nil {
		instructions = p.settings.Instructions
	} else {
		instructions = ""
	}

	var opts *AgentExecutionOptions
	if hasDefaults {
		opts = defaultOptions
	}

	return &AgentConfig{
		ID:             p.settings.ID,
		Name:           p.settings.ID,
		Instructions:   instructions,
		Model:          p.settings.Model,
		Tools:          tools,
		MaxRetries:     p.settings.MaxRetries,
		DefaultOptions: opts,
	}
}

// ---------------------------------------------------------------------------
// mapToProcessInputStepResult
// ---------------------------------------------------------------------------

// mapToProcessInputStepResult maps a prepareCall or prepareStep result (returned as
// a generic map) into a ProcessInputStepResult. Both hooks return similar structures
// that can override model, tools, activeTools, etc.
func (p *ToolLoopAgentProcessor) mapToProcessInputStepResult(result map[string]any) *ProcessInputStepResult {
	if result == nil {
		return nil
	}

	stepResult := &ProcessInputStepResult{}
	populated := false

	// Map model.
	if model, ok := result["model"]; ok && model != nil {
		stepResult.Model = model
		populated = true
	}

	// Map tools (prepareCall can return this).
	if tools, ok := result["tools"]; ok && tools != nil {
		if t, ok := tools.(map[string]any); ok {
			stepResult.Tools = t
			populated = true
		}
	}

	// Map toolChoice (prepareStep can return this).
	if tc, ok := result["toolChoice"]; ok && tc != nil {
		stepResult.ToolChoice = tc
		populated = true
	}

	// Map activeTools (both can return this).
	if at, ok := result["activeTools"]; ok && at != nil {
		if atSlice, ok := at.([]string); ok {
			stepResult.ActiveTools = atSlice
			populated = true
		}
	}

	// Map providerOptions (prepareCall can return this).
	// Real type is SharedProviderOptions = map[string]any, but we accept
	// both map[string]any and map[string]map[string]any for flexibility.
	if po, ok := result["providerOptions"]; ok && po != nil {
		switch poTyped := po.(type) {
		case map[string]any:
			stepResult.ProviderOptions = poTyped
			populated = true
		case map[string]map[string]any:
			// Flatten to map[string]any for compatibility with real type.
			flat := make(map[string]any, len(poTyped))
			for k, v := range poTyped {
				flat[k] = v
			}
			stepResult.ProviderOptions = flat
			populated = true
		}
	}

	// Map model settings (prepareCall can return individual settings).
	// Build as map[string]any to match real ProcessInputStepResult.ModelSettings type.
	ms := make(map[string]any)
	if v, ok := result["temperature"]; ok && v != nil {
		ms["temperature"] = v
	}
	if v, ok := result["topP"]; ok && v != nil {
		ms["topP"] = v
	}
	if v, ok := result["topK"]; ok && v != nil {
		ms["topK"] = v
	}
	if v, ok := result["maxOutputTokens"]; ok && v != nil {
		ms["maxOutputTokens"] = v
	}
	if v, ok := result["presencePenalty"]; ok && v != nil {
		ms["presencePenalty"] = v
	}
	if v, ok := result["frequencyPenalty"]; ok && v != nil {
		ms["frequencyPenalty"] = v
	}
	if v, ok := result["stopSequences"]; ok && v != nil {
		ms["stopSequences"] = v
	}
	if v, ok := result["seed"]; ok && v != nil {
		ms["seed"] = v
	}
	if len(ms) > 0 {
		stepResult.ModelSettings = ms
		populated = true
	}

	// Map system/instructions to systemMessages.
	// prepareCall returns "instructions", prepareStep returns "system".
	var systemContent any
	if instr, ok := result["instructions"]; ok {
		systemContent = instr
	} else if sys, ok := result["system"]; ok {
		systemContent = sys
	}
	if systemContent != nil {
		populated = true
		switch sc := systemContent.(type) {
		case string:
			stepResult.SystemMessages = []any{
				map[string]any{"role": "system", "content": sc},
			}
		case []any:
			msgs := make([]any, 0, len(sc))
			for _, msg := range sc {
				if s, ok := msg.(string); ok {
					msgs = append(msgs, map[string]any{"role": "system", "content": s})
				} else {
					msgs = append(msgs, msg)
				}
			}
			stepResult.SystemMessages = msgs
		case map[string]any:
			// Single system message object with role and content.
			stepResult.SystemMessages = []any{sc}
		}
	}

	// Map messages if prepareStep returns them.
	if msgs, ok := result["messages"]; ok && msgs != nil {
		if msgSlice, ok := msgs.([]any); ok {
			stepResult.Messages = msgSlice
			populated = true
		}
	}

	if !populated {
		return nil
	}
	return stepResult
}

// ---------------------------------------------------------------------------
// handlePrepareCall
// ---------------------------------------------------------------------------

// handlePrepareCall invokes the ToolLoopAgentSettings.PrepareCall hook.
func (p *ToolLoopAgentProcessor) handlePrepareCall(args *ProcessInputStepArgs) error {
	if p.settings.PrepareCall == nil {
		return nil
	}

	// Build the prepareCall input object.
	input := &PrepareCallInput{
		Messages:     args.Messages,
		Model:        args.Model,
		Tools:        args.Tools,
		Instructions: p.settings.Instructions,
		StopWhen:     p.settings.StopWhen,
		ActiveTools:  args.ActiveTools,
	}

	// Provider options.
	// Real type is map[string]any; convert to map[string]map[string]any for PrepareCallInput.
	if args.ProviderOptions != nil {
		po := make(map[string]map[string]any)
		for k, v := range args.ProviderOptions {
			if m, ok := v.(map[string]any); ok {
				po[k] = m
			}
		}
		if len(po) > 0 {
			input.ProviderOptions = po
		}
	}

	// Model settings — extract typed fields from map[string]any.
	if args.ModelSettings != nil {
		input.Temperature = modelSettingsGetFloat64(args.ModelSettings, "temperature")
		input.TopP = modelSettingsGetFloat64(args.ModelSettings, "topP")
		input.TopK = modelSettingsGetInt(args.ModelSettings, "topK")
		input.MaxOutputTokens = modelSettingsGetInt(args.ModelSettings, "maxOutputTokens")
		input.PresencePenalty = modelSettingsGetFloat64(args.ModelSettings, "presencePenalty")
		input.FrequencyPenalty = modelSettingsGetFloat64(args.ModelSettings, "frequencyPenalty")
		input.StopSequences = modelSettingsGetStringSlice(args.ModelSettings, "stopSequences")
		input.Seed = modelSettingsGetInt(args.ModelSettings, "seed")
	}

	result, err := p.settings.PrepareCall(input)
	if err != nil {
		return fmt.Errorf("prepareCall: %w", err)
	}
	p.prepareCallResult = result
	return nil
}

// ---------------------------------------------------------------------------
// handlePrepareStep
// ---------------------------------------------------------------------------

// handlePrepareStep invokes the ToolLoopAgentSettings.PrepareStep hook.
func (p *ToolLoopAgentProcessor) handlePrepareStep(args *ProcessInputStepArgs, currentResult *ProcessInputStepResult) (map[string]any, error) {
	if p.settings.PrepareStep == nil {
		return nil, nil
	}

	model := args.Model
	if currentResult != nil && currentResult.Model != nil {
		resolved, err := resolveModelConfig(currentResult.Model)
		if err != nil {
			return nil, fmt.Errorf("handlePrepareStep: resolveModelConfig: %w", err)
		}
		if !isSupportedLanguageModel(resolved) {
			return nil, errors.New("prepareStep returned an unsupported model version")
		}
		model = resolved
	}

	input := &PrepareStepInput{
		Model:               model,
		Messages:            args.Messages,
		Steps:               args.Steps,
		StepNumber:          args.StepNumber,
		ExperimentalContext: nil,
	}

	result, err := p.settings.PrepareStep(input)
	if err != nil {
		return nil, fmt.Errorf("prepareStep: %w", err)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// ProcessInputStep
// ---------------------------------------------------------------------------

// ProcessInputStep is the main entry point called at each step of the agentic loop.
// It applies prepareCall (on step 0) and prepareStep (on every step) overrides.
func (p *ToolLoopAgentProcessor) ProcessInputStep(args *ProcessInputStepArgs) (*ProcessInputStepResult, error) {
	// On step 0, invoke prepareCall if configured.
	if args.StepNumber == 0 && p.settings.PrepareCall != nil {
		if err := p.handlePrepareCall(args); err != nil {
			return nil, err
		}
	}

	result := &ProcessInputStepResult{}

	// Apply prepareCall result (cached from step 0).
	if p.prepareCallResult != nil {
		mapped := p.mapToProcessInputStepResult(p.prepareCallResult)
		if mapped != nil {
			mergeProcessInputStepResult(result, mapped)
		}
	}

	// Apply prepareStep result (called on every step).
	if p.settings.PrepareStep != nil {
		prepareStepResult, err := p.handlePrepareStep(args, result)
		if err != nil {
			return nil, err
		}
		if prepareStepResult != nil {
			mapped := p.mapToProcessInputStepResult(prepareStepResult)
			if mapped != nil {
				// prepareStep overrides prepareCall for this step.
				mergeProcessInputStepResult(result, mapped)
			}
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// mergeProcessInputStepResult
// ---------------------------------------------------------------------------

// mergeProcessInputStepResult merges src into dst, overwriting non-nil fields.
func mergeProcessInputStepResult(dst, src *ProcessInputStepResult) {
	if src.Model != nil {
		dst.Model = src.Model
	}
	if src.Tools != nil {
		dst.Tools = src.Tools
	}
	if src.ToolChoice != nil {
		dst.ToolChoice = src.ToolChoice
	}
	if src.ActiveTools != nil {
		dst.ActiveTools = src.ActiveTools
	}
	if src.Messages != nil {
		dst.Messages = src.Messages
	}
	if src.MessageList != nil {
		dst.MessageList = src.MessageList
	}
	if src.SystemMessages != nil {
		dst.SystemMessages = src.SystemMessages
	}
	if src.ProviderOptions != nil {
		dst.ProviderOptions = src.ProviderOptions
	}
	if src.ModelSettings != nil {
		dst.ModelSettings = src.ModelSettings
	}
	if src.StructuredOutput != nil {
		dst.StructuredOutput = src.StructuredOutput
	}
	if src.RetryCount != nil {
		dst.RetryCount = src.RetryCount
	}
}
