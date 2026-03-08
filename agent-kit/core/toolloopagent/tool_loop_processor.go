// Ported from: packages/core/src/tool-loop-agent/tool-loop-processor.ts
package toolloopagent

import (
	"errors"
	"fmt"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// MastraLanguageModel is a stub for ../llm/model/shared.types.MastraLanguageModel.
// STUB REASON: The real model.MastraLanguageModel is also `= any` (a union type).
// Importing would add a dependency for no type-safety gain. Keep local alias.
type MastraLanguageModel = any

// AgentInstructions is a stub for ../agent.AgentInstructions (SystemMessage).
// STUB REASON: The real agent.AgentInstructions = SystemMessage = any (chain of aliases).
// Importing would add a dependency for no type-safety gain. Keep local alias.
type AgentInstructions = any

// ProcessInputStepArgs mirrors ../processors.ProcessInputStepArgs.
// STUB REASON: The real processors.ProcessInputStepArgs embeds ProcessorMessageContext,
// uses typed fields (StepResult, CoreMessageV4, MastraLanguageModel, ToolChoice,
// StructuredOutputOptions, ObservabilityContext) and has ~20 fields. This stub uses
// simplified types ([]any, map[string]any). Structural mismatch prevents replacement.
type ProcessInputStepArgs struct {
	StepNumber     int            `json:"stepNumber"`
	Steps          []any          `json:"steps,omitempty"`
	Messages       []any          `json:"messages,omitempty"`
	SystemMessages []any          `json:"systemMessages,omitempty"`
	State          map[string]any `json:"state,omitempty"`

	Model           any            `json:"model,omitempty"`
	Tools           map[string]any `json:"tools,omitempty"`
	ToolChoice      any            `json:"toolChoice,omitempty"`
	ActiveTools     []string       `json:"activeTools,omitempty"`
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`
	ModelSettings   *ModelSettings `json:"modelSettings,omitempty"`
}

// ModelSettings mirrors Omit<CallSettings, 'abortSignal'> from the AI SDK.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 CallSettings remain local stubs.
type ModelSettings struct {
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"topP,omitempty"`
	TopK             *int     `json:"topK,omitempty"`
	Seed             *int     `json:"seed,omitempty"`
	MaxOutputTokens  *int     `json:"maxOutputTokens,omitempty"`
	PresencePenalty  *float64 `json:"presencePenalty,omitempty"`
	FrequencyPenalty *float64 `json:"frequencyPenalty,omitempty"`
	StopSequences    []string `json:"stopSequences,omitempty"`
}

// ProcessInputStepResult mirrors ../processors.ProcessInputStepResult.
// STUB REASON: The real processors.ProcessInputStepResult uses typed fields (ToolChoice,
// MastraDBMessage, MessageList, SharedProviderOptions, StructuredOutputOptions) and has
// ModelSettings as map[string]any. This stub uses simplified types and *ModelSettings struct.
// Also has fewer fields (no MessageList, StructuredOutput, RetryCount). Structural mismatch.
type ProcessInputStepResult struct {
	Model           any                       `json:"model,omitempty"`
	Tools           map[string]any            `json:"tools,omitempty"`
	ToolChoice      any                       `json:"toolChoice,omitempty"`
	ActiveTools     []string                  `json:"activeTools,omitempty"`
	Messages        []any                     `json:"messages,omitempty"`
	SystemMessages  []any                     `json:"systemMessages,omitempty"`
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`
	ModelSettings   *ModelSettings            `json:"modelSettings,omitempty"`
}

// Processor is a stub for ../processors.Processor.
// STUB REASON: The real processors.Processor interface uses ID()/Name() (not GetID()/GetName())
// and has many more methods (Description, ProcessorIndex, ProcessInput, ProcessInputStep,
// ProcessOutput, ProcessOutputStream). Method name mismatch prevents replacement.
type Processor interface {
	GetID() string
	GetName() string
}

// AgentExecutionOptions is a stub for ../agent.AgentExecutionOptions.
// STUB REASON: The real agent.AgentExecutionOptions embeds AgentExecutionOptionsBase (20+
// fields: Memory, Delegation, ObservabilityContext, etc.) plus StructuredOutput. This stub
// has only 6 flat fields. Structural mismatch prevents direct replacement.
type AgentExecutionOptions struct {
	ToolChoice      any            `json:"toolChoice,omitempty"`
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`
	ModelSettings   *ModelSettings `json:"modelSettings,omitempty"`
	StopWhen        any            `json:"stopWhen,omitempty"`
	OnStepFinish    any            `json:"onStepFinish,omitempty"`
	OnFinish        any            `json:"onFinish,omitempty"`
}

// resolveModelConfig is a stub for ../llm/model/resolve-model.ResolveModelConfig.
// STUB REASON: The real model.ResolveModelConfig takes (modelConfig any, customGateways
// []MastraModelGateway, requestContext ...any) — different signature requiring gateway
// and request context params. This stub takes only model and returns it as-is.
func resolveModelConfig(model any) (any, error) {
	// In the real implementation this resolves string/config to a LanguageModelV2.
	return model, nil
}

// isSupportedLanguageModel is a stub for ../agent.IsSupportedLanguageModel.
// STUB REASON: The real agent.IsSupportedLanguageModel takes LanguageModelLike interface
// (requires SpecificationVersion() method) while this stub takes any. Different parameter
// type prevents direct replacement without refactoring call sites.
func isSupportedLanguageModel(model any) bool {
	// In the real implementation this checks if the model satisfies LanguageModelV2.
	return model != nil
}

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
	Steps               []any  `json:"steps,omitempty"`
	StepNumber          int    `json:"stepNumber"`
	Model               any    `json:"model,omitempty"`
	Messages            []any  `json:"messages,omitempty"`
	ExperimentalContext any    `json:"experimental_context,omitempty"`
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
	id   string
	name string

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

// GetID implements Processor.
func (p *ToolLoopAgentProcessor) GetID() string { return p.id }

// GetName implements Processor.
func (p *ToolLoopAgentProcessor) GetName() string { return p.name }

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

	// Model settings.
	ms := &ModelSettings{}
	hasModelSettings := false

	if p.settings.Temperature != nil {
		ms.Temperature = p.settings.Temperature
		hasModelSettings = true
	}
	if p.settings.TopP != nil {
		ms.TopP = p.settings.TopP
		hasModelSettings = true
	}
	if p.settings.TopK != nil {
		ms.TopK = p.settings.TopK
		hasModelSettings = true
	}
	if p.settings.Seed != nil {
		ms.Seed = p.settings.Seed
		hasModelSettings = true
	}
	if p.settings.MaxOutputTokens != nil {
		ms.MaxOutputTokens = p.settings.MaxOutputTokens
		hasModelSettings = true
	}
	if p.settings.PresencePenalty != nil {
		ms.PresencePenalty = p.settings.PresencePenalty
		hasModelSettings = true
	}
	if p.settings.FrequencyPenalty != nil {
		ms.FrequencyPenalty = p.settings.FrequencyPenalty
		hasModelSettings = true
	}
	if p.settings.StopSequences != nil {
		ms.StopSequences = p.settings.StopSequences
		hasModelSettings = true
	}
	if hasModelSettings {
		defaultOptions.ModelSettings = ms
		hasDefaults = true
	}

	// Callbacks.
	if p.settings.StopWhen != nil {
		// TODO: The callback signatures differ (StepResult vs event are incompatible).
		defaultOptions.StopWhen = p.settings.StopWhen
		hasDefaults = true
	}
	if p.settings.OnStepFinish != nil {
		// TODO: The callback signatures differ (StepResult vs event are incompatible).
		defaultOptions.OnStepFinish = p.settings.OnStepFinish
		hasDefaults = true
	}
	if p.settings.OnFinish != nil {
		// TODO: The callback signatures differ ('event' and 'event' are incompatible).
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
	if po, ok := result["providerOptions"]; ok && po != nil {
		if poMap, ok := po.(map[string]map[string]any); ok {
			stepResult.ProviderOptions = poMap
			populated = true
		}
	}

	// Map model settings (prepareCall can return individual settings).
	ms := &ModelSettings{}
	hasMS := false
	if v, ok := result["temperature"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			ms.Temperature = &f
			hasMS = true
		}
	}
	if v, ok := result["topP"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			ms.TopP = &f
			hasMS = true
		}
	}
	if v, ok := result["topK"]; ok && v != nil {
		if i, ok := v.(int); ok {
			ms.TopK = &i
			hasMS = true
		}
	}
	if v, ok := result["maxOutputTokens"]; ok && v != nil {
		if i, ok := v.(int); ok {
			ms.MaxOutputTokens = &i
			hasMS = true
		}
	}
	if v, ok := result["presencePenalty"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			ms.PresencePenalty = &f
			hasMS = true
		}
	}
	if v, ok := result["frequencyPenalty"]; ok && v != nil {
		if f, ok := v.(float64); ok {
			ms.FrequencyPenalty = &f
			hasMS = true
		}
	}
	if v, ok := result["stopSequences"]; ok && v != nil {
		if ss, ok := v.([]string); ok {
			ms.StopSequences = ss
			hasMS = true
		}
	}
	if v, ok := result["seed"]; ok && v != nil {
		if i, ok := v.(int); ok {
			ms.Seed = &i
			hasMS = true
		}
	}
	if hasMS {
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
	if args.ProviderOptions != nil {
		input.ProviderOptions = args.ProviderOptions
	}

	// Model settings.
	if args.ModelSettings != nil {
		input.Temperature = args.ModelSettings.Temperature
		input.TopP = args.ModelSettings.TopP
		input.TopK = args.ModelSettings.TopK
		input.MaxOutputTokens = args.ModelSettings.MaxOutputTokens
		input.PresencePenalty = args.ModelSettings.PresencePenalty
		input.FrequencyPenalty = args.ModelSettings.FrequencyPenalty
		input.StopSequences = args.ModelSettings.StopSequences
		input.Seed = args.ModelSettings.Seed
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
	if src.SystemMessages != nil {
		dst.SystemMessages = src.SystemMessages
	}
	if src.ProviderOptions != nil {
		dst.ProviderOptions = src.ProviderOptions
	}
	if src.ModelSettings != nil {
		dst.ModelSettings = src.ModelSettings
	}
}
