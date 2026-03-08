// Ported from: packages/core/src/stream/base/output.ts
package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
	"github.com/brainlet/brainkit/agent-kit/core/stream/aisdk/v5/compat"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// MastraBase is a stub for ../../base MastraBase.
// Stub: real agentkit.MastraBase has private logger + RegisteredLogger Component;
// this type is defined but unused — kept for documentation parity with TS source.
type MastraBase struct {
	Component string
	Name      string
	Logger    any
}

// MessageList is a stub for ../../agent/message-list MessageList.
// Stub: real agent MessageList has rich methods (get.response, get.all, etc.);
// simplified here to avoid coupling stream/base to agent internals.
type MessageList struct {
	Messages []any
}

// MastraDBMessage is a stub for ../../agent/message-list MastraDBMessage.
// Stub: real agent.MastraDBMessage is a typed struct; kept as map alias
// to avoid coupling stream/base to agent internals.
type MastraDBMessage = map[string]any

// ProcessorRunner is a stub for ../../processors/runner ProcessorRunner.
// Stub: real ProcessorRunner has complex processor pipeline methods;
// simplified here to avoid coupling stream/base to processors internals.
type ProcessorRunner struct {
	OutputProcessors []any
}

// ProcessorState is a stub for ../../processors/runner ProcessorState.
// Stub: parallel-stubs architecture — real type has state tracking fields.
type ProcessorState struct{}

// WorkflowRunStatus mirrors ../../workflows WorkflowRunStatus.
// Stub: workflows imports stream (circular dep); must remain local definition.
type WorkflowRunStatus = string

// StreamTransport mirrors the TS StreamTransport type from types.ts.
// Stub: defined locally to avoid circular import between stream/base and stream/.
type StreamTransport struct {
	CloseOnFinish bool
	Close         func()
}

// ---------------------------------------------------------------------------
// LLMStepResult
// ---------------------------------------------------------------------------

// LLMStepResult represents the result of a single LLM execution step.
// Mirrors the TS LLMStepResult<OUTPUT> type.
type LLMStepResult struct {
	StepType         string                          `json:"stepType,omitempty"`
	Text             string                          `json:"text"`
	Reasoning        []stream.ChunkType              `json:"reasoning"`
	ReasoningText    string                          `json:"reasoningText,omitempty"`
	Sources          []stream.ChunkType              `json:"sources"`
	Files            []stream.ChunkType              `json:"files"`
	ToolCalls        []stream.ChunkType              `json:"toolCalls"`
	ToolResults      []stream.ChunkType              `json:"toolResults"`
	DynamicToolCalls []stream.ChunkType              `json:"dynamicToolCalls,omitempty"`
	DynamicToolResults []stream.ChunkType            `json:"dynamicToolResults,omitempty"`
	StaticToolCalls  []stream.ChunkType              `json:"staticToolCalls,omitempty"`
	StaticToolResults []stream.ChunkType             `json:"staticToolResults,omitempty"`
	Content          []any                           `json:"content"`
	Usage            stream.LanguageModelUsage       `json:"usage"`
	Warnings         []stream.LanguageModelV2CallWarning `json:"warnings"`
	Request          map[string]any                  `json:"request"`
	Response         map[string]any                  `json:"response"`
	ProviderMetadata stream.ProviderMetadata         `json:"providerMetadata,omitempty"`
	FinishReason     string                          `json:"finishReason,omitempty"`
	Tripwire         *stream.StepTripwireData        `json:"tripwire,omitempty"`
}

// NewEmptyLLMStepResult creates a new empty LLMStepResult with initialized fields.
func NewEmptyLLMStepResult() LLMStepResult {
	return LLMStepResult{
		Reasoning:   []stream.ChunkType{},
		Sources:     []stream.ChunkType{},
		Files:       []stream.ChunkType{},
		ToolCalls:   []stream.ChunkType{},
		ToolResults: []stream.ChunkType{},
		Content:     []any{},
		Usage: stream.LanguageModelUsage{},
		Warnings: []stream.LanguageModelV2CallWarning{},
		Request:  map[string]any{},
		Response: map[string]any{},
	}
}

// ---------------------------------------------------------------------------
// MastraModelOutputOptions
// ---------------------------------------------------------------------------

// MastraModelOutputOptions configures a MastraModelOutput instance.
// Mirrors the TS MastraModelOutputOptions<OUTPUT> type from types.ts.
type MastraModelOutputOptions struct {
	RunID             string
	StructuredOutput  *StructuredOutputOptions
	IncludeRawChunks  bool
	OutputProcessors  []any
	ProcessorStates   any
	TransportRef      any
	ReturnScorerData  bool
	IsLLMExecutionStep bool
	TracingContext    any
	RequestContext    any
	OnStepFinish     func(stepResult LLMStepResult) error
	OnFinish         func(args MastraOnFinishCallbackArgs) error
}

// MastraOnFinishCallbackArgs holds the arguments for onFinish callback.
// Mirrors the TS MastraOnFinishCallbackArgs<OUTPUT> type.
type MastraOnFinishCallbackArgs struct {
	Text             string                          `json:"text"`
	Usage            stream.LanguageModelUsage       `json:"usage"`
	TotalUsage       stream.LanguageModelUsage       `json:"totalUsage"`
	FinishReason     string                          `json:"finishReason"`
	Warnings         []stream.LanguageModelV2CallWarning `json:"warnings"`
	ProviderMetadata stream.ProviderMetadata         `json:"providerMetadata,omitempty"`
	Request          map[string]any                  `json:"request"`
	Response         map[string]any                  `json:"response"`
	Reasoning        []stream.ChunkType              `json:"reasoning"`
	ReasoningText    *string                         `json:"reasoningText,omitempty"`
	Sources          []stream.ChunkType              `json:"sources"`
	Files            []stream.ChunkType              `json:"files"`
	ToolCalls        []stream.ChunkType              `json:"toolCalls"`
	ToolResults      []stream.ChunkType              `json:"toolResults"`
	Steps            []LLMStepResult                 `json:"steps"`
	Content          []any                           `json:"content"`
	Object           any                             `json:"object,omitempty"`
	Error            error                           `json:"error,omitempty"`
	Model            *ModelInfo                      `json:"model,omitempty"`
}

// ModelInfo holds model identification information.
type ModelInfo struct {
	ModelID  string `json:"modelId,omitempty"`
	Provider string `json:"provider,omitempty"`
	Version  string `json:"version,omitempty"`
}

// ---------------------------------------------------------------------------
// FullOutput
// ---------------------------------------------------------------------------

// FullOutput is the complete output returned by GetFullOutput().
// Mirrors the TS FullOutput<OUTPUT> type.
type FullOutput struct {
	Text             string                          `json:"text"`
	Usage            stream.LanguageModelUsage       `json:"usage"`
	Steps            []LLMStepResult                 `json:"steps"`
	FinishReason     string                          `json:"finishReason"`
	Warnings         []stream.LanguageModelV2CallWarning `json:"warnings"`
	ProviderMetadata stream.ProviderMetadata         `json:"providerMetadata,omitempty"`
	Request          map[string]any                  `json:"request"`
	Reasoning        []stream.ChunkType              `json:"reasoning"`
	ReasoningText    *string                         `json:"reasoningText,omitempty"`
	ToolCalls        []stream.ChunkType              `json:"toolCalls"`
	ToolResults      []stream.ChunkType              `json:"toolResults"`
	Sources          []stream.ChunkType              `json:"sources"`
	Files            []stream.ChunkType              `json:"files"`
	Response         map[string]any                  `json:"response"`
	TotalUsage       stream.LanguageModelUsage       `json:"totalUsage"`
	Object           any                             `json:"object,omitempty"`
	Error            error                           `json:"error,omitempty"`
	Tripwire         *stream.StepTripwireData        `json:"tripwire,omitempty"`
	TraceID          string                          `json:"traceId,omitempty"`
	RunID            string                          `json:"runId,omitempty"`
	SuspendPayload   any                             `json:"suspendPayload,omitempty"`
	ResumeSchema     any                             `json:"resumeSchema,omitempty"`
	Messages         []MastraDBMessage               `json:"messages"`
	RememberedMessages []MastraDBMessage             `json:"rememberedMessages"`
}

// ---------------------------------------------------------------------------
// MastraModelOutput
// ---------------------------------------------------------------------------

// MastraModelOutput manages a model execution's stream, accumulating
// text, tool calls, usage, structured output, and step results.
// It implements the MastraBaseStream-like interface.
//
// This is the Go equivalent of the 1544-line TS MastraModelOutput<OUTPUT> class.
// Due to Go's type system differences, some TS patterns (like Proxy for
// destructuring, EventEmitter for multiplexing) are adapted to Go idioms
// (channels, sync primitives).
type MastraModelOutput struct {
	mu sync.Mutex

	// Public fields
	RunID      string
	TraceID    string
	MessageID  string
	MessageList MessageList

	// Private state
	status               WorkflowRunStatus
	err                  error
	streamFinished       bool
	consumptionStarted   bool
	structuredOutputMode string // "direct", "processor", or ""
	returnScorerData     bool

	model ModelInfo

	// Buffered data
	bufferedChunks            []stream.ChunkType
	bufferedSteps             []LLMStepResult
	bufferedReasoningDetails  map[string]stream.ChunkType
	bufferedByStep            LLMStepResult
	bufferedText              []string
	bufferedObject            any
	bufferedTextChunks        map[string][]string
	bufferedSources           []stream.ChunkType
	bufferedReasoning         []stream.ChunkType
	bufferedFiles             []stream.ChunkType
	toolCallArgsDeltas        map[string][]string
	toolCallDeltaIDNameMap    map[string]string
	toolCalls                 []stream.ChunkType
	toolResults               []stream.ChunkType
	warnings                  []stream.LanguageModelV2CallWarning
	finishReason              string
	request                   map[string]any
	usageCount                stream.LanguageModelUsage
	tripwire                  *stream.StepTripwireData

	// Delayed promises for final values
	textPromise             *compat.DelayedPromise[string]
	reasoningPromise        *compat.DelayedPromise[[]stream.ChunkType]
	reasoningTextPromise    *compat.DelayedPromise[*string]
	sourcesPromise          *compat.DelayedPromise[[]stream.ChunkType]
	filesPromise            *compat.DelayedPromise[[]stream.ChunkType]
	toolCallsPromise        *compat.DelayedPromise[[]stream.ChunkType]
	toolResultsPromise      *compat.DelayedPromise[[]stream.ChunkType]
	usagePromise            *compat.DelayedPromise[stream.LanguageModelUsage]
	warningsPromise         *compat.DelayedPromise[[]stream.LanguageModelV2CallWarning]
	providerMetadataPromise *compat.DelayedPromise[stream.ProviderMetadata]
	responsePromise         *compat.DelayedPromise[map[string]any]
	requestPromise          *compat.DelayedPromise[map[string]any]
	objectPromise           *compat.DelayedPromise[any]
	finishReasonPromise     *compat.DelayedPromise[string]
	stepsPromise            *compat.DelayedPromise[[]LLMStepResult]
	totalUsagePromise       *compat.DelayedPromise[stream.LanguageModelUsage]
	contentPromise          *compat.DelayedPromise[[]any]
	suspendPayloadPromise   *compat.DelayedPromise[any]
	resumeSchemaPromise     *compat.DelayedPromise[any]

	// Subscribers for the evented stream
	subscribers []chan stream.ChunkType
	finishCh    chan struct{}

	options *MastraModelOutputOptions

	// ProcessorRunner (optional)
	ProcessorRunner *ProcessorRunner

	// Object stream transformer (optional, for direct structured output mode)
	objectTransformer *ObjectStreamTransformer
}

// MastraModelOutputParams are the constructor parameters.
type MastraModelOutputParams struct {
	Model        ModelInfo
	Stream       <-chan stream.ChunkType
	MessageList  MessageList
	Options      MastraModelOutputOptions
	MessageID    string
	InitialState any
}

// NewMastraModelOutput creates a new MastraModelOutput and starts consuming
// the input stream in a background goroutine.
func NewMastraModelOutput(params MastraModelOutputParams) *MastraModelOutput {
	m := &MastraModelOutput{
		status:                   "running",
		model:                    params.Model,
		RunID:                    params.Options.RunID,
		MessageID:                params.MessageID,
		MessageList:              params.MessageList,
		options:                  &params.Options,
		returnScorerData:         params.Options.ReturnScorerData,

		bufferedChunks:            []stream.ChunkType{},
		bufferedSteps:             []LLMStepResult{},
		bufferedReasoningDetails:  map[string]stream.ChunkType{},
		bufferedByStep:            NewEmptyLLMStepResult(),
		bufferedText:              []string{},
		bufferedTextChunks:        map[string][]string{},
		bufferedSources:           []stream.ChunkType{},
		bufferedReasoning:         []stream.ChunkType{},
		bufferedFiles:             []stream.ChunkType{},
		toolCallArgsDeltas:        map[string][]string{},
		toolCallDeltaIDNameMap:    map[string]string{},
		toolCalls:                 []stream.ChunkType{},
		toolResults:               []stream.ChunkType{},
		warnings:                  []stream.LanguageModelV2CallWarning{},
		request:                   map[string]any{},
		usageCount:                stream.LanguageModelUsage{},

		textPromise:             compat.NewDelayedPromise[string](),
		reasoningPromise:        compat.NewDelayedPromise[[]stream.ChunkType](),
		reasoningTextPromise:    compat.NewDelayedPromise[*string](),
		sourcesPromise:          compat.NewDelayedPromise[[]stream.ChunkType](),
		filesPromise:            compat.NewDelayedPromise[[]stream.ChunkType](),
		toolCallsPromise:        compat.NewDelayedPromise[[]stream.ChunkType](),
		toolResultsPromise:      compat.NewDelayedPromise[[]stream.ChunkType](),
		usagePromise:            compat.NewDelayedPromise[stream.LanguageModelUsage](),
		warningsPromise:         compat.NewDelayedPromise[[]stream.LanguageModelV2CallWarning](),
		providerMetadataPromise: compat.NewDelayedPromise[stream.ProviderMetadata](),
		responsePromise:         compat.NewDelayedPromise[map[string]any](),
		requestPromise:          compat.NewDelayedPromise[map[string]any](),
		objectPromise:           compat.NewDelayedPromise[any](),
		finishReasonPromise:     compat.NewDelayedPromise[string](),
		stepsPromise:            compat.NewDelayedPromise[[]LLMStepResult](),
		totalUsagePromise:       compat.NewDelayedPromise[stream.LanguageModelUsage](),
		contentPromise:          compat.NewDelayedPromise[[]any](),
		suspendPayloadPromise:   compat.NewDelayedPromise[any](),
		resumeSchemaPromise:     compat.NewDelayedPromise[any](),

		finishCh: make(chan struct{}),
	}

	// Determine structured output mode
	if params.Options.StructuredOutput != nil && params.Options.StructuredOutput.Schema != nil {
		if params.Options.StructuredOutput.Model != nil {
			m.structuredOutputMode = "processor"
		} else {
			m.structuredOutputMode = "direct"
		}
	}

	// Set up object stream transformer for direct mode
	if m.structuredOutputMode == "direct" && params.Options.IsLLMExecutionStep {
		m.objectTransformer = NewObjectStreamTransformer(params.Options.StructuredOutput)
	}

	go m.consumeInput(params.Stream)
	return m
}

// consumeInput reads from the input stream and processes each chunk.
func (m *MastraModelOutput) consumeInput(inputStream <-chan stream.ChunkType) {
	defer func() {
		// Flush: reject any pending promises
		m.mu.Lock()
		m.streamFinished = true
		m.mu.Unlock()

		// Flush object transformer if present
		if m.objectTransformer != nil {
			for _, chunk := range m.objectTransformer.Flush() {
				m.processChunk(chunk)
			}
		}

		// Resolve object promise if still pending
		if m.objectPromise.Status().Type == compat.DelayedPromiseStatusPending {
			m.objectPromise.Resolve(nil)
		}

		// Reject any unresolved promises
		m.rejectPendingPromises()

		// Close subscribers
		m.mu.Lock()
		for _, sub := range m.subscribers {
			close(sub)
		}
		m.subscribers = nil
		close(m.finishCh)
		m.mu.Unlock()
	}()

	for rawChunk := range inputStream {
		// Apply object transformer if present
		if m.objectTransformer != nil {
			chunks := m.objectTransformer.Transform(rawChunk)
			for _, chunk := range chunks {
				m.processChunk(chunk)
			}
		} else {
			m.processChunk(rawChunk)
		}
	}
}

// processChunk processes a single chunk, updating internal state.
func (m *MastraModelOutput) processChunk(chunk stream.ChunkType) {
	m.mu.Lock()

	switch chunk.Type {
	case "tool-call-suspended", "tool-call-approval":
		m.status = "suspended"
		m.suspendPayloadPromise.Resolve(chunk.Payload)
		// Extract resumeSchema from payload (mirrors TS: chunk.payload.resumeSchema)
		var resumeSchema any
		if payloadMap, ok := chunk.Payload.(map[string]any); ok {
			resumeSchema = payloadMap["resumeSchema"]
		}
		m.resumeSchemaPromise.Resolve(resumeSchema)
		m.mu.Unlock()
		m.emitChunk(chunk)
		return

	case "raw":
		if !m.options.IncludeRawChunks {
			m.mu.Unlock()
			return
		}

	case "object-result":
		m.bufferedObject = chunk.Object
		if m.objectPromise.Status().Type == compat.DelayedPromiseStatusPending {
			m.objectPromise.Resolve(chunk.Object)
		}

	case "source":
		m.bufferedSources = append(m.bufferedSources, chunk)
		m.bufferedByStep.Sources = append(m.bufferedByStep.Sources, chunk)

	case "text-delta":
		text := extractTextFromPayload(chunk.Payload)
		m.bufferedText = append(m.bufferedText, text)
		m.bufferedByStep.Text += text
		id := extractIDFromPayload(chunk.Payload)
		if id != "" {
			m.bufferedTextChunks[id] = append(m.bufferedTextChunks[id], text)
		}

	case "tool-call-input-streaming-start":
		if p, ok := chunk.Payload.(map[string]any); ok {
			toolCallID, _ := p["toolCallId"].(string)
			toolName, _ := p["toolName"].(string)
			if toolCallID != "" {
				m.toolCallDeltaIDNameMap[toolCallID] = toolName
			}
		}

	case "tool-call-delta":
		if p, ok := chunk.Payload.(map[string]any); ok {
			toolCallID, _ := p["toolCallId"].(string)
			argsTextDelta, _ := p["argsTextDelta"].(string)
			if toolCallID != "" {
				m.toolCallArgsDeltas[toolCallID] = append(m.toolCallArgsDeltas[toolCallID], argsTextDelta)
				// Mutate chunk to add toolname
				if _, ok := p["toolName"]; !ok || p["toolName"] == "" {
					p["toolName"] = m.toolCallDeltaIDNameMap[toolCallID]
				}
			}
		}

	case "file":
		m.bufferedFiles = append(m.bufferedFiles, chunk)
		m.bufferedByStep.Files = append(m.bufferedByStep.Files, chunk)

	case "reasoning-start":
		id := extractIDFromPayload(chunk.Payload)
		m.bufferedReasoningDetails[id] = stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: chunk.RunID,
				From:  chunk.From,
			},
			Type: "reasoning",
			Payload: map[string]any{
				"id":               id,
				"providerMetadata": extractProviderMetadataFromPayload(chunk.Payload),
				"text":             "",
			},
		}

	case "reasoning-delta":
		reasoningChunk := stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: chunk.RunID,
				From:  chunk.From,
			},
			Type:    "reasoning",
			Payload: chunk.Payload,
		}
		m.bufferedReasoning = append(m.bufferedReasoning, reasoningChunk)
		m.bufferedByStep.Reasoning = append(m.bufferedByStep.Reasoning, reasoningChunk)

		id := extractIDFromPayload(chunk.Payload)
		text := extractTextFromPayload(chunk.Payload)
		if existing, ok := m.bufferedReasoningDetails[id]; ok {
			if p, ok := existing.Payload.(map[string]any); ok {
				p["text"] = p["text"].(string) + text
			}
		}

	case "reasoning-end":
		// Update provider metadata on the buffered reasoning detail
		id := extractIDFromPayload(chunk.Payload)
		if existing, ok := m.bufferedReasoningDetails[id]; ok {
			pm := extractProviderMetadataFromPayload(chunk.Payload)
			if pm != nil {
				if p, ok := existing.Payload.(map[string]any); ok {
					p["providerMetadata"] = pm
				}
			}
		}

	case "tool-call":
		m.toolCalls = append(m.toolCalls, chunk)
		m.bufferedByStep.ToolCalls = append(m.bufferedByStep.ToolCalls, chunk)

	case "tool-result":
		m.toolResults = append(m.toolResults, chunk)
		m.bufferedByStep.ToolResults = append(m.bufferedByStep.ToolResults, chunk)

	case "step-finish":
		m.processStepFinish(chunk)

	case "tripwire":
		m.processTripwire(chunk)
		m.mu.Unlock()
		m.emitChunk(chunk)
		return

	case "finish":
		m.processFinish(chunk)

	case "error":
		m.processError(chunk)
	}

	m.mu.Unlock()
	m.emitChunk(chunk)
}

// processStepFinish handles step-finish chunks. Must be called with mu held.
func (m *MastraModelOutput) processStepFinish(chunk stream.ChunkType) {
	payload := extractStepFinishPayload(chunk.Payload)
	if payload == nil {
		return
	}

	m.updateUsageCount(payload.OutputUsage)
	m.warnings = payload.Warnings

	if payload.Request != nil {
		m.request = payload.Request
	}

	// Build step text - if tripwire, text should be empty
	stepText := m.bufferedByStep.Text
	if payload.StepTripwire != nil {
		stepText = ""
	}

	stepResult := LLMStepResult{
		Text:          stepText,
		Sources:       m.bufferedByStep.Sources,
		Files:         m.bufferedByStep.Files,
		ToolCalls:     m.bufferedByStep.ToolCalls,
		ToolResults:   m.bufferedByStep.ToolResults,
		Reasoning:     valuesFromReasoningDetails(m.bufferedReasoningDetails),
		FinishReason:  payload.StepResultReason,
		Usage:         payload.OutputUsage,
		Warnings:      m.warnings,
		Request:       payload.Request,
		Response:      payload.Response,
		ProviderMetadata: payload.ProviderMetadata,
		Tripwire:      payload.StepTripwire,
	}

	if len(m.bufferedSteps) == 0 {
		stepResult.StepType = "initial"
	} else {
		stepResult.StepType = "tool-result"
	}

	// Build reasoning text
	var reasoningTexts []string
	for _, r := range m.bufferedReasoning {
		reasoningTexts = append(reasoningTexts, extractTextFromPayload(r.Payload))
	}
	if len(reasoningTexts) > 0 {
		combined := ""
		for _, t := range reasoningTexts {
			combined += t
		}
		stepResult.ReasoningText = combined
	}

	m.bufferedSteps = append(m.bufferedSteps, stepResult)

	// Reset per-step buffers
	m.bufferedByStep = NewEmptyLLMStepResult()
}

// processFinish handles finish chunks. Must be called with mu held.
func (m *MastraModelOutput) processFinish(chunk stream.ChunkType) {
	m.status = "success"

	payload := extractFinishPayload(chunk.Payload)
	if payload != nil {
		if payload.StepResultReason != "" {
			m.finishReason = payload.StepResultReason
		}
		m.populateUsageCount(payload.OutputUsage)
	}

	// Build text from buffered text
	text := ""
	for _, t := range m.bufferedText {
		text += t
	}

	m.textPromise.Resolve(text)
	m.finishReasonPromise.Resolve(m.finishReason)
	m.usagePromise.Resolve(m.usageCount)
	m.warningsPromise.Resolve(m.warnings)
	m.requestPromise.Resolve(m.request)
	m.stepsPromise.Resolve(m.bufferedSteps)
	m.totalUsagePromise.Resolve(m.getTotalUsage())
	m.sourcesPromise.Resolve(m.bufferedSources)
	m.filesPromise.Resolve(m.bufferedFiles)
	m.toolCallsPromise.Resolve(m.toolCalls)
	m.toolResultsPromise.Resolve(m.toolResults)

	// Reasoning
	reasoningDetails := valuesFromReasoningDetails(m.bufferedReasoningDetails)
	m.reasoningPromise.Resolve(reasoningDetails)

	if len(m.bufferedReasoning) > 0 {
		combined := ""
		for _, r := range m.bufferedReasoning {
			combined += extractTextFromPayload(r.Payload)
		}
		m.reasoningTextPromise.Resolve(&combined)
	} else {
		m.reasoningTextPromise.Resolve(nil)
	}

	// Provider metadata
	if payload != nil {
		m.providerMetadataPromise.Resolve(payload.ProviderMetadata)
		m.responsePromise.Resolve(payload.Response)
	}
	m.contentPromise.Resolve([]any{})
	m.suspendPayloadPromise.Resolve(nil)
	m.resumeSchemaPromise.Resolve(nil)
}

// processTripwire handles tripwire chunks. Must be called with mu held.
func (m *MastraModelOutput) processTripwire(chunk stream.ChunkType) {
	if p, ok := chunk.Payload.(map[string]any); ok {
		reason, _ := p["reason"].(string)
		if reason == "" {
			reason = "Content blocked"
		}
		m.tripwire = &stream.StepTripwireData{
			Reason:      reason,
			ProcessorID: stringFromAny(p["processorId"]),
		}
	}
	m.finishReason = "other"
	m.streamFinished = true

	// Resolve promises
	text := ""
	for _, t := range m.bufferedText {
		text += t
	}
	m.textPromise.Resolve(text)
	m.finishReasonPromise.Resolve("other")
	m.usagePromise.Resolve(m.usageCount)
	m.warningsPromise.Resolve(m.warnings)
	m.objectPromise.Resolve(nil)
	m.reasoningPromise.Resolve([]stream.ChunkType{})
	m.reasoningTextPromise.Resolve(nil)
	m.sourcesPromise.Resolve([]stream.ChunkType{})
	m.filesPromise.Resolve([]stream.ChunkType{})
	m.toolCallsPromise.Resolve([]stream.ChunkType{})
	m.toolResultsPromise.Resolve([]stream.ChunkType{})
	m.stepsPromise.Resolve(m.bufferedSteps)
	m.totalUsagePromise.Resolve(m.usageCount)
	m.contentPromise.Resolve([]any{})
	m.suspendPayloadPromise.Resolve(nil)
	m.resumeSchemaPromise.Resolve(nil)
	m.requestPromise.Resolve(map[string]any{})
	m.responsePromise.Resolve(map[string]any{})
	m.providerMetadataPromise.Resolve(nil)
}

// processError handles error chunks. Must be called with mu held.
func (m *MastraModelOutput) processError(chunk stream.ChunkType) {
	var chunkErr error
	if p, ok := chunk.Payload.(map[string]any); ok {
		if e, ok := p["error"].(error); ok {
			chunkErr = e
		} else if msg, ok := p["error"].(string); ok {
			chunkErr = errors.New(msg)
		}
	}
	if chunkErr == nil {
		chunkErr = errors.New("unknown error chunk in stream")
	}

	m.err = chunkErr
	m.status = "failed"
	m.streamFinished = true

	// Reject all pending promises
	m.rejectAllPending(chunkErr)
}

// rejectAllPending rejects all pending delayed promises with the given error.
func (m *MastraModelOutput) rejectAllPending(err error) {
	promises := []interface {
		Reject(error)
		Status() compat.DelayedPromiseStatus[any]
	}{}
	// We can't easily iterate generic promises, so we handle them individually
	rejectIfPending(m.textPromise, err)
	rejectIfPending(m.reasoningPromise, err)
	rejectIfPending(m.reasoningTextPromise, err)
	rejectIfPending(m.sourcesPromise, err)
	rejectIfPending(m.filesPromise, err)
	rejectIfPending(m.toolCallsPromise, err)
	rejectIfPending(m.toolResultsPromise, err)
	rejectIfPending(m.usagePromise, err)
	rejectIfPending(m.warningsPromise, err)
	rejectIfPending(m.providerMetadataPromise, err)
	rejectIfPending(m.responsePromise, err)
	rejectIfPending(m.requestPromise, err)
	rejectIfPending(m.objectPromise, err)
	rejectIfPending(m.finishReasonPromise, err)
	rejectIfPending(m.stepsPromise, err)
	rejectIfPending(m.totalUsagePromise, err)
	rejectIfPending(m.contentPromise, err)
	rejectIfPending(m.suspendPayloadPromise, err)
	rejectIfPending(m.resumeSchemaPromise, err)
	_ = promises
}

// rejectPendingPromises rejects all still-pending promises with a generic error.
func (m *MastraModelOutput) rejectPendingPromises() {
	genericErr := errors.New("promise was not resolved or rejected when stream finished")
	m.rejectAllPending(genericErr)
}

// emitChunk buffers a chunk and sends it to all current subscribers.
func (m *MastraModelOutput) emitChunk(chunk stream.ChunkType) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bufferedChunks = append(m.bufferedChunks, chunk)
	for _, sub := range m.subscribers {
		select {
		case sub <- chunk:
		default:
		}
	}
}

// updateUsageCount updates the usage count with values from a LanguageModelUsage.
// Must be called with mu held.
func (m *MastraModelOutput) updateUsageCount(usage stream.LanguageModelUsage) {
	m.usageCount.InputTokens += usage.InputTokens
	m.usageCount.OutputTokens += usage.OutputTokens
	m.usageCount.TotalTokens += usage.TotalTokens
	m.usageCount.ReasoningTokens += usage.ReasoningTokens
	m.usageCount.CachedInputTokens += usage.CachedInputTokens
}

// populateUsageCount populates usage count only if not already set.
// Must be called with mu held.
func (m *MastraModelOutput) populateUsageCount(usage stream.LanguageModelUsage) {
	if m.usageCount.InputTokens == 0 {
		m.usageCount.InputTokens = usage.InputTokens
	}
	if m.usageCount.OutputTokens == 0 {
		m.usageCount.OutputTokens = usage.OutputTokens
	}
	if m.usageCount.TotalTokens == 0 {
		m.usageCount.TotalTokens = usage.TotalTokens
	}
	if m.usageCount.ReasoningTokens == 0 {
		m.usageCount.ReasoningTokens = usage.ReasoningTokens
	}
	if m.usageCount.CachedInputTokens == 0 {
		m.usageCount.CachedInputTokens = usage.CachedInputTokens
	}
}

// getTotalUsage computes total usage from the accumulated counts.
func (m *MastraModelOutput) getTotalUsage() stream.LanguageModelUsage {
	total := m.usageCount.TotalTokens
	if total == 0 {
		total = m.usageCount.InputTokens + m.usageCount.OutputTokens + m.usageCount.ReasoningTokens
	}
	return stream.LanguageModelUsage{
		InputTokens:      m.usageCount.InputTokens,
		OutputTokens:     m.usageCount.OutputTokens,
		TotalTokens:      total,
		ReasoningTokens:  m.usageCount.ReasoningTokens,
		CachedInputTokens: m.usageCount.CachedInputTokens,
	}
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// FullStream returns a channel that replays all buffered chunks and then
// delivers new chunks in real time until the stream finishes.
func (m *MastraModelOutput) FullStream() <-chan stream.ChunkType {
	out := make(chan stream.ChunkType, 256)
	go func() {
		defer close(out)

		m.mu.Lock()
		// Replay buffered chunks
		for _, chunk := range m.bufferedChunks {
			out <- chunk
		}

		if m.streamFinished {
			m.mu.Unlock()
			return
		}

		// Subscribe for new chunks
		sub := make(chan stream.ChunkType, 256)
		m.subscribers = append(m.subscribers, sub)
		m.mu.Unlock()

		// Start consumption if needed
		m.startConsumption()

		for chunk := range sub {
			out <- chunk
		}
	}()
	return out
}

// ConsumeStream reads through the full stream to drive processing.
func (m *MastraModelOutput) ConsumeStream(onError func(error)) {
	m.startConsumption()
}

// startConsumption ensures the stream is being consumed.
func (m *MastraModelOutput) startConsumption() {
	m.mu.Lock()
	if m.consumptionStarted {
		m.mu.Unlock()
		return
	}
	m.consumptionStarted = true
	m.mu.Unlock()
}

// AwaitText blocks until the text promise is resolved.
func (m *MastraModelOutput) AwaitText() (string, error) {
	m.startConsumption()
	return m.textPromise.Await()
}

// AwaitObject blocks until the object promise is resolved.
func (m *MastraModelOutput) AwaitObject() (any, error) {
	m.startConsumption()
	return m.objectPromise.Await()
}

// AwaitUsage blocks until the usage promise is resolved.
func (m *MastraModelOutput) AwaitUsage() (stream.LanguageModelUsage, error) {
	m.startConsumption()
	return m.usagePromise.Await()
}

// AwaitFinishReason blocks until the finishReason promise is resolved.
func (m *MastraModelOutput) AwaitFinishReason() (string, error) {
	m.startConsumption()
	return m.finishReasonPromise.Await()
}

// AwaitSteps blocks until the steps promise is resolved.
func (m *MastraModelOutput) AwaitSteps() ([]LLMStepResult, error) {
	m.startConsumption()
	return m.stepsPromise.Await()
}

// AwaitToolCalls blocks until the toolCalls promise is resolved.
func (m *MastraModelOutput) AwaitToolCalls() ([]stream.ChunkType, error) {
	m.startConsumption()
	return m.toolCallsPromise.Await()
}

// AwaitToolResults blocks until the toolResults promise is resolved.
func (m *MastraModelOutput) AwaitToolResults() ([]stream.ChunkType, error) {
	m.startConsumption()
	return m.toolResultsPromise.Await()
}

// AwaitWarnings blocks until the warnings promise is resolved.
func (m *MastraModelOutput) AwaitWarnings() ([]stream.LanguageModelV2CallWarning, error) {
	m.startConsumption()
	return m.warningsPromise.Await()
}

// AwaitTotalUsage blocks until the totalUsage promise is resolved.
func (m *MastraModelOutput) AwaitTotalUsage() (stream.LanguageModelUsage, error) {
	m.startConsumption()
	return m.totalUsagePromise.Await()
}

// AwaitReasoning blocks until the reasoning promise is resolved.
func (m *MastraModelOutput) AwaitReasoning() ([]stream.ChunkType, error) {
	m.startConsumption()
	return m.reasoningPromise.Await()
}

// AwaitReasoningText blocks until the reasoningText promise is resolved.
func (m *MastraModelOutput) AwaitReasoningText() (*string, error) {
	m.startConsumption()
	return m.reasoningTextPromise.Await()
}

// AwaitSources blocks until the sources promise is resolved.
func (m *MastraModelOutput) AwaitSources() ([]stream.ChunkType, error) {
	m.startConsumption()
	return m.sourcesPromise.Await()
}

// AwaitFiles blocks until the files promise is resolved.
func (m *MastraModelOutput) AwaitFiles() ([]stream.ChunkType, error) {
	m.startConsumption()
	return m.filesPromise.Await()
}

// AwaitResponse blocks until the response promise is resolved.
func (m *MastraModelOutput) AwaitResponse() (map[string]any, error) {
	m.startConsumption()
	return m.responsePromise.Await()
}

// AwaitRequest blocks until the request promise is resolved.
func (m *MastraModelOutput) AwaitRequest() (map[string]any, error) {
	m.startConsumption()
	return m.requestPromise.Await()
}

// GetFullOutput returns the complete output after the stream is consumed.
func (m *MastraModelOutput) GetFullOutput() (*FullOutput, error) {
	m.startConsumption()
	<-m.finishCh

	_, _ = m.textPromise.Await() // text is computed from steps below
	usage, _ := m.usagePromise.Await()
	steps, _ := m.stepsPromise.Await()
	finishReason, _ := m.finishReasonPromise.Await()
	warnings, _ := m.warningsPromise.Await()
	providerMetadata, _ := m.providerMetadataPromise.Await()
	request, _ := m.requestPromise.Await()
	reasoning, _ := m.reasoningPromise.Await()
	reasoningText, _ := m.reasoningTextPromise.Await()
	toolCalls, _ := m.toolCallsPromise.Await()
	toolResults, _ := m.toolResultsPromise.Await()
	sources, _ := m.sourcesPromise.Await()
	files, _ := m.filesPromise.Await()
	response, _ := m.responsePromise.Await()
	totalUsage, _ := m.totalUsagePromise.Await()
	object, _ := m.objectPromise.Await()
	suspendPayload, _ := m.suspendPayloadPromise.Await()
	resumeSchema, _ := m.resumeSchemaPromise.Await()

	// Calculate text from steps (respects tripwire)
	textFromSteps := ""
	for _, step := range steps {
		textFromSteps += step.Text
	}

	return &FullOutput{
		Text:             textFromSteps,
		Usage:            usage,
		Steps:            steps,
		FinishReason:     finishReason,
		Warnings:         warnings,
		ProviderMetadata: providerMetadata,
		Request:          request,
		Reasoning:        reasoning,
		ReasoningText:    reasoningText,
		ToolCalls:        toolCalls,
		ToolResults:      toolResults,
		Sources:          sources,
		Files:            files,
		Response:         response,
		TotalUsage:       totalUsage,
		Object:           object,
		Error:            m.err,
		Tripwire:         m.tripwire,
		TraceID:          m.TraceID,
		RunID:            m.RunID,
		SuspendPayload:   suspendPayload,
		ResumeSchema:     resumeSchema,
		Messages:         []MastraDBMessage{},
		RememberedMessages: []MastraDBMessage{},
	}, nil
}

// Status returns the current status.
func (m *MastraModelOutput) Status() WorkflowRunStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

// Error returns the error if one occurred.
func (m *MastraModelOutput) Error() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.err
}

// Tripwire returns the tripwire data if the stream was aborted.
func (m *MastraModelOutput) Tripwire() *stream.StepTripwireData {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tripwire
}

// ImmediateText returns the immediately available text.
func (m *MastraModelOutput) ImmediateText() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	text := ""
	for _, t := range m.bufferedText {
		text += t
	}
	return text
}

// ImmediateObject returns the immediately available object.
func (m *MastraModelOutput) ImmediateObject() any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bufferedObject
}

// ImmediateUsage returns the immediately available usage.
func (m *MastraModelOutput) ImmediateUsage() stream.LanguageModelUsage {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.usageCount
}

// ImmediateFinishReason returns the immediately available finish reason.
func (m *MastraModelOutput) ImmediateFinishReason() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.finishReason
}

// WaitForFinish blocks until the stream has fully finished.
func (m *MastraModelOutput) WaitForFinish() {
	<-m.finishCh
}

// SerializeState returns the current state as a serializable map.
func (m *MastraModelOutput) SerializeState() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]any{
		"status":                   m.status,
		"bufferedSteps":            m.bufferedSteps,
		"bufferedReasoningDetails": m.bufferedReasoningDetails,
		"bufferedByStep":           m.bufferedByStep,
		"bufferedText":             m.bufferedText,
		"bufferedTextChunks":       m.bufferedTextChunks,
		"bufferedSources":          m.bufferedSources,
		"bufferedReasoning":        m.bufferedReasoning,
		"bufferedFiles":            m.bufferedFiles,
		"toolCallArgsDeltas":       m.toolCallArgsDeltas,
		"toolCallDeltaIdNameMap":   m.toolCallDeltaIDNameMap,
		"toolCalls":                m.toolCalls,
		"toolResults":              m.toolResults,
		"warnings":                 m.warnings,
		"finishReason":             m.finishReason,
		"request":                  m.request,
		"usageCount":               m.usageCount,
		"tripwire":                 m.tripwire,
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// rejectIfPending is a helper to reject a delayed promise if it's still pending.
func rejectIfPending[T any](dp *compat.DelayedPromise[T], err error) {
	if dp.Status().Type == compat.DelayedPromiseStatusPending {
		dp.Reject(err)
	}
}

// stepFinishPayloadData holds extracted step-finish payload data.
type stepFinishPayloadData struct {
	OutputUsage      stream.LanguageModelUsage
	Warnings         []stream.LanguageModelV2CallWarning
	Request          map[string]any
	Response         map[string]any
	ProviderMetadata stream.ProviderMetadata
	StepResultReason string
	StepTripwire     *stream.StepTripwireData
}

// extractStepFinishPayload extracts data from a step-finish payload.
func extractStepFinishPayload(payload any) *stepFinishPayloadData {
	if payload == nil {
		return nil
	}
	if p, ok := payload.(*stream.StepFinishPayload); ok {
		return &stepFinishPayloadData{
			OutputUsage:      p.Output.Usage,
			Warnings:         p.StepResult.Warnings,
			StepResultReason: string(p.StepResult.Reason),
			ProviderMetadata: p.ProviderMetadata,
		}
	}
	if p, ok := payload.(stream.StepFinishPayload); ok {
		return &stepFinishPayloadData{
			OutputUsage:      p.Output.Usage,
			Warnings:         p.StepResult.Warnings,
			StepResultReason: string(p.StepResult.Reason),
			ProviderMetadata: p.ProviderMetadata,
		}
	}
	if p, ok := payload.(map[string]any); ok {
		data := &stepFinishPayloadData{}
		if output, ok := p["output"].(map[string]any); ok {
			if usage, ok := output["usage"].(map[string]any); ok {
				data.OutputUsage = usageFromMap(usage)
			}
		}
		if stepResult, ok := p["stepResult"].(map[string]any); ok {
			data.StepResultReason, _ = stepResult["reason"].(string)
		}
		if metadata, ok := p["metadata"].(map[string]any); ok {
			if req, ok := metadata["request"].(map[string]any); ok {
				data.Request = req
			}
			data.ProviderMetadata, _ = metadata["providerMetadata"].(stream.ProviderMetadata)
			data.Response = metadata
		}
		return data
	}
	return nil
}

// finishPayloadData holds extracted finish payload data.
type finishPayloadData struct {
	OutputUsage      stream.LanguageModelUsage
	StepResultReason string
	ProviderMetadata stream.ProviderMetadata
	Response         map[string]any
}

// extractFinishPayload extracts data from a finish payload.
func extractFinishPayload(payload any) *finishPayloadData {
	if payload == nil {
		return nil
	}
	if p, ok := payload.(map[string]any); ok {
		data := &finishPayloadData{}
		if output, ok := p["output"].(map[string]any); ok {
			if usage, ok := output["usage"].(map[string]any); ok {
				data.OutputUsage = usageFromMap(usage)
			}
		}
		if stepResult, ok := p["stepResult"].(map[string]any); ok {
			data.StepResultReason, _ = stepResult["reason"].(string)
		}
		if metadata, ok := p["metadata"].(map[string]any); ok {
			data.ProviderMetadata, _ = metadata["providerMetadata"].(stream.ProviderMetadata)
			data.Response = metadata
		}
		return data
	}
	return nil
}

// usageFromMap converts a map to LanguageModelUsage.
func usageFromMap(m map[string]any) stream.LanguageModelUsage {
	return stream.LanguageModelUsage{
		InputTokens:      intFromAny(m["inputTokens"]),
		OutputTokens:     intFromAny(m["outputTokens"]),
		TotalTokens:      intFromAny(m["totalTokens"]),
		ReasoningTokens:  intFromAny(m["reasoningTokens"]),
		CachedInputTokens: intFromAny(m["cachedInputTokens"]),
	}
}

func intFromAny(v any) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case json.Number:
		n, _ := val.Int64()
		return int(n)
	default:
		return 0
	}
}

func stringFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func extractTextFromPayload(payload any) string {
	if p, ok := payload.(map[string]any); ok {
		if text, ok := p["text"].(string); ok {
			return text
		}
	}
	if p, ok := payload.(*stream.TextDeltaPayload); ok {
		return p.Text
	}
	if p, ok := payload.(stream.TextDeltaPayload); ok {
		return p.Text
	}
	return ""
}

func extractIDFromPayload(payload any) string {
	if p, ok := payload.(map[string]any); ok {
		if id, ok := p["id"].(string); ok {
			return id
		}
	}
	if p, ok := payload.(*stream.TextDeltaPayload); ok {
		return p.ID
	}
	if p, ok := payload.(stream.TextDeltaPayload); ok {
		return p.ID
	}
	return ""
}

func extractProviderMetadataFromPayload(payload any) stream.ProviderMetadata {
	if p, ok := payload.(map[string]any); ok {
		if pm, ok := p["providerMetadata"].(stream.ProviderMetadata); ok {
			return pm
		}
	}
	return nil
}

func valuesFromReasoningDetails(details map[string]stream.ChunkType) []stream.ChunkType {
	result := make([]stream.ChunkType, 0, len(details))
	for _, v := range details {
		result = append(result, v)
	}
	return result
}
