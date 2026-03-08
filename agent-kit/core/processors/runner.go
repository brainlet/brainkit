// Ported from: packages/core/src/processors/runner.ts
package processors

import (
	"context"
	"fmt"
	"sync"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// Stub types for runner dependencies
// ---------------------------------------------------------------------------

// TripWire represents an abort/tripwire error thrown by processors.
// STUB REASON: The TripWire type is defined here (not in agent) because processors
// create TripWire errors directly. Agent imports processors (circular), so
// TripWire is canonical to this package.
type TripWire struct {
	Message     string
	Options     *TripWireOptions
	ProcessorID string
}

func (tw *TripWire) Error() string {
	return tw.Message
}

// NewTripWire creates a new TripWire error.
func NewTripWire(message string, options *TripWireOptions, processorID string) *TripWire {
	return &TripWire{
		Message:     message,
		Options:     options,
		ProcessorID: processorID,
	}
}

// MastraError is resolved: uses mastraerror.NewMastraError from error package.

// ---------------------------------------------------------------------------
// ProcessorState
// ---------------------------------------------------------------------------

// ProcessorState tracks state for stream processing across chunks.
// Used by both legacy processors and workflow processors.
type ProcessorState struct {
	mu                    sync.Mutex
	inputAccumulatedText  string
	outputAccumulatedText string
	outputChunkCount      int
	CustomState           map[string]any
	StreamParts           []ChunkType
	// Span is an optional tracing span for this processor's execution.
	Span obstypes.Span
}

// ProcessorStateOptions holds optional configuration for creating a ProcessorState
// with observability support.
type ProcessorStateOptions struct {
	ProcessorName  string
	ProcessorIndex int
	CreateSpan     bool
	// ObservabilityContext provides tracing context for span creation.
	ObservabilityContext *obstypes.ObservabilityContext
}

// NewProcessorState creates a new ProcessorState without observability.
func NewProcessorState() *ProcessorState {
	return &ProcessorState{
		CustomState: make(map[string]any),
		StreamParts: make([]ChunkType, 0),
	}
}

// NewProcessorStateWithOptions creates a new ProcessorState with optional observability.
// Only creates a span if options.CreateSpan is true and a processor name is provided.
// Workflow processors handle span creation in workflow.ts, so legacy processors
// explicitly request span creation here.
func NewProcessorStateWithOptions(options *ProcessorStateOptions) *ProcessorState {
	ps := &ProcessorState{
		CustomState: make(map[string]any),
		StreamParts: make([]ChunkType, 0),
	}

	if options == nil || !options.CreateSpan || options.ProcessorName == "" {
		return ps
	}

	if options.ObservabilityContext == nil {
		return ps
	}

	currentSpan := options.ObservabilityContext.Tracing.CurrentSpan
	if currentSpan == nil {
		return ps
	}

	// Find the agent run span as parent, or use parent/current span.
	parentSpan := currentSpan.FindParent(obstypes.SpanTypeAgentRun)
	if parentSpan == nil {
		parentSpan = currentSpan
	}

	entityType := obstypes.EntityTypeOutputProcessor
	ps.Span = parentSpan.CreateChildSpan(obstypes.ChildSpanOptions{
		CreateBaseOptions: obstypes.CreateBaseOptions{
			Type:       obstypes.SpanTypeProcessorRun,
			Name:       fmt.Sprintf("output stream processor: %s", options.ProcessorName),
			EntityType: &entityType,
			EntityName: options.ProcessorName,
			Attributes: map[string]any{
				"processorExecutor": "legacy",
				"processorIndex":    options.ProcessorIndex,
			},
		},
		Input: map[string]any{
			"totalChunks": 0,
		},
	})

	return ps
}

// AddInputPart tracks an incoming chunk (before processor transformation).
func (ps *ProcessorState) AddInputPart(part ChunkType) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Extract text from text-delta chunks for accumulated text
	if part.Type == "text-delta" {
		if payload, ok := part.Payload.(map[string]any); ok {
			if text, ok := payload["text"].(string); ok {
				ps.inputAccumulatedText += text
			}
		}
	}
	ps.StreamParts = append(ps.StreamParts, part)

	// Update span input with accumulated state.
	if ps.Span != nil {
		ps.Span.Update(obstypes.UpdateSpanOptions{
			Input: map[string]any{
				"totalChunks":     len(ps.StreamParts),
				"accumulatedText": ps.inputAccumulatedText,
			},
		})
	}
}

// AddOutputPart tracks an outgoing chunk (after processor transformation).
func (ps *ProcessorState) AddOutputPart(part *ChunkType) {
	if part == nil {
		return
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.outputChunkCount++
	if part.Type == "text-delta" {
		if payload, ok := part.Payload.(map[string]any); ok {
			if text, ok := payload["text"].(string); ok {
				ps.outputAccumulatedText += text
			}
		}
	}
}

// ProcessorStateFinalOutput holds the final output from a ProcessorState.
type ProcessorStateFinalOutput struct {
	TotalChunks    int    `json:"totalChunks"`
	AccumulatedText string `json:"accumulatedText"`
}

// GetFinalOutput returns the final output for the span.
func (ps *ProcessorState) GetFinalOutput() ProcessorStateFinalOutput {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ProcessorStateFinalOutput{
		TotalChunks:     ps.outputChunkCount,
		AccumulatedText: ps.outputAccumulatedText,
	}
}

// ---------------------------------------------------------------------------
// ProcessPartResult
// ---------------------------------------------------------------------------

// ProcessPartResult holds the result of processing a stream part.
type ProcessPartResult struct {
	Part            *ChunkType
	Blocked         bool
	Reason          string
	TripwireOptions *TripWireOptions
	ProcessorID     string
}

// ---------------------------------------------------------------------------
// ProcessorRunner
// ---------------------------------------------------------------------------

// ProcessorRunner orchestrates the 5-phase processor pipeline.
// It runs input processors, output processors, stream processors,
// input step processors, and output step processors.
type ProcessorRunner struct {
	InputProcessors  []any // Processor or ProcessorWorkflow
	OutputProcessors []any // Processor or ProcessorWorkflow
	logger           logger.IMastraLogger
	agentName        string

	// processorStates is shared processor state that persists across loop iterations.
	// Used by all processor methods (input and output) to share state.
	// Keyed by processor ID.
	processorStates sync.Map
}

// ProcessorRunnerConfig holds configuration for creating a ProcessorRunner.
type ProcessorRunnerConfig struct {
	InputProcessors  []any
	OutputProcessors []any
	Logger           logger.IMastraLogger
	AgentName        string
}

// NewProcessorRunner creates a new ProcessorRunner.
func NewProcessorRunner(cfg ProcessorRunnerConfig) *ProcessorRunner {
	inputProcessors := cfg.InputProcessors
	if inputProcessors == nil {
		inputProcessors = []any{}
	}
	outputProcessors := cfg.OutputProcessors
	if outputProcessors == nil {
		outputProcessors = []any{}
	}
	return &ProcessorRunner{
		InputProcessors:  inputProcessors,
		OutputProcessors: outputProcessors,
		logger:           cfg.Logger,
		agentName:        cfg.AgentName,
	}
}

// getProcessorState gets or creates ProcessorState for the given processor ID.
// This state persists across loop iterations and is shared between
// all processor methods (input and output).
func (pr *ProcessorRunner) getProcessorState(processorID string) *ProcessorState {
	val, loaded := pr.processorStates.LoadOrStore(processorID, NewProcessorState())
	if loaded {
		return val.(*ProcessorState)
	}
	return val.(*ProcessorState)
}

// ---------------------------------------------------------------------------
// executeWorkflowAsProcessor
// ---------------------------------------------------------------------------

// executeWorkflowAsProcessor executes a workflow as a processor and handles
// the result. Returns the processed output and any tripwire information.
//
// Corresponds to TS: private async executeWorkflowAsProcessor(workflow, input, observabilityContext?, requestContext?, writer?, abortSignal?)
func (pr *ProcessorRunner) executeWorkflowAsProcessor(
	workflow ProcessorWorkflow,
	input ProcessorStepOutput,
	observabilityContext *obstypes.ObservabilityContext,
	requestCtx *requestcontext.RequestContext,
	writer ProcessorStreamWriter,
	abortSignal context.Context,
) (*ProcessorStepOutput, error) {
	run, err := workflow.CreateRun()
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow run: %w", err)
	}

	var outputWriter func(data any) error
	if writer != nil {
		outputWriter = func(data any) error {
			if dc, ok := data.(DataChunkType); ok {
				return writer.Custom(dc)
			}
			return nil
		}
	}

	result, err := run.Start(WorkflowRunStartOpts{
		InputData:      input,
		RequestContext: requestCtx,
		OutputWriter:   outputWriter,
	})
	if err != nil {
		return nil, fmt.Errorf("workflow run failed: %w", err)
	}

	// Check for tripwire status
	if result.Status == "tripwire" {
		tripwireData := result.Tripwire
		reason := fmt.Sprintf("Tripwire triggered in workflow %s", workflow.GetID())
		if tripwireData != nil && tripwireData.Reason != "" {
			reason = tripwireData.Reason
		}
		opts := &TripWireOptions{}
		pid := workflow.GetID()
		if tripwireData != nil {
			opts.Retry = tripwireData.Retry
			opts.Metadata = tripwireData.Metadata
			if tripwireData.ProcessorID != "" {
				pid = tripwireData.ProcessorID
			}
		}
		return nil, NewTripWire(reason, opts, pid)
	}

	// Check for execution failure
	if result.Status != "success" {
		details := ""
		if result.Error != nil {
			details = result.Error.Message
		}
		for stepID, step := range result.Steps {
			if step.Status == "failed" && step.Error != nil && step.Error.Message != "" {
				if details != "" {
					details += "; "
				}
				details += fmt.Sprintf("step %s: %s", stepID, step.Error.Message)
			}
		}
		text := fmt.Sprintf("Processor workflow %s failed with status: %s", workflow.GetID(), result.Status)
		if details != "" {
			text += " — " + details
		}
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "PROCESSOR_WORKFLOW_FAILED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Text:     text,
		})
	}

	// Extract and validate output
	if result.Result == nil {
		return &input, nil
	}

	output := result.Result
	if output.Phase == "" {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "PROCESSOR_WORKFLOW_INVALID_OUTPUT",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Text:     fmt.Sprintf("Processor workflow %s returned invalid output format. Expected ProcessorStepOutput.", workflow.GetID()),
		})
	}

	return output, nil
}

// ---------------------------------------------------------------------------
// RunOutputProcessors
// ---------------------------------------------------------------------------

// RunOutputProcessors runs processOutputResult for all output processors.
//
// Corresponds to TS: async runOutputProcessors(messageList, observabilityContext?, requestContext?, retryCount, writer?)
func (pr *ProcessorRunner) RunOutputProcessors(
	messageList *MessageList,
	requestCtx *requestcontext.RequestContext,
	retryCount int,
	writer ProcessorStreamWriter,
	observabilityContext ...*obstypes.ObservabilityContext,
) (*MessageList, error) {
	var obsCtx *obstypes.ObservabilityContext
	if len(observabilityContext) > 0 {
		obsCtx = observabilityContext[0]
	}
	for index, processorOrWorkflow := range pr.OutputProcessors {
		processableMessages := messageList.GetResponseDB()
		idsBeforeProcessing := extractMessageIDs(processableMessages)
		check := messageList.MakeMessageSourceChecker()

		// Handle workflow as processor
		if wf, ok := processorOrWorkflow.(ProcessorWorkflow); ok {
			_, err := pr.executeWorkflowAsProcessor(
				wf,
				ProcessorStepOutput{
					Phase:       ProcessorPhaseOutputResult,
					MessageList: messageList,
				},
				obsCtx,
				requestCtx,
				writer,
				nil,
			)
			if err != nil {
				return nil, err
			}
			continue
		}

		// Handle regular processor
		processor, ok := processorOrWorkflow.(OutputProcessor)
		if !ok {
			continue
		}

		abort := func(reason string, options *TripWireOptions) error {
			if reason == "" {
				reason = fmt.Sprintf("Tripwire triggered by %s", processor.ID())
			}
			return NewTripWire(reason, options, processor.ID())
		}

		// Create processor span for observability
		var processorSpan obstypes.Span
		if obsCtx != nil {
			currentSpan := obsCtx.Tracing.CurrentSpan
			if currentSpan != nil {
				parentSpan := currentSpan.FindParent(obstypes.SpanTypeAgentRun)
				if parentSpan == nil {
					parentSpan = currentSpan
				}
				entityType := obstypes.EntityTypeOutputProcessor
				processorSpan = parentSpan.CreateChildSpan(obstypes.ChildSpanOptions{
					CreateBaseOptions: obstypes.CreateBaseOptions{
						Type:       obstypes.SpanTypeProcessorRun,
						Name:       fmt.Sprintf("output processor: %s", processor.ID()),
						EntityType: &entityType,
						EntityID:   processor.ID(),
						EntityName: processor.Name(),
						Attributes: map[string]any{
							"processorExecutor": "legacy",
							"processorIndex":    index,
						},
					},
					Input: processableMessages,
				})
			}
		}

		// Start recording MessageList mutations for this processor
		messageList.StartRecording()

		processorState := pr.getProcessorState(processor.ID())

		messages, returnedList, err := processor.ProcessOutputResult(ProcessOutputResultArgs{
			ProcessorMessageContext: ProcessorMessageContext{
				ProcessorContext: ProcessorContext{
					Abort:          abort,
					RequestContext: requestCtx,
					RetryCount:     retryCount,
					Writer:         writer,
				},
				MessageList: messageList,
			},
			State: processorState.CustomState,
		})
		if err != nil {
			messageList.StopRecording()
			return nil, err
		}

		// Stop recording and get mutations for this processor
		mutations := messageList.StopRecording()

		// Handle return types
		if returnedList != nil {
			if returnedList != messageList {
				return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "PROCESSOR_RETURNED_EXTERNAL_MESSAGE_LIST",
					Domain:   mastraerror.ErrorDomainAgent,
					Category: mastraerror.ErrorCategoryUser,
					Text:     fmt.Sprintf("Processor %s returned a MessageList instance other than the one that was passed in.", processor.ID()),
				})
			}
			// MessageList was mutated in place
			if len(mutations) > 0 {
				processableMessages = returnedList.GetResponseDB()
			}
		} else if messages != nil {
			// Processor returned an array — apply to messageList
			applyMessagesToMessageList(messages, messageList, idsBeforeProcessing, check, "response")
			processableMessages = messageList.GetResponseDB()
		}

		// End the processor span
		if processorSpan != nil {
			endOpts := &obstypes.EndSpanOptions{
				Output: processableMessages,
			}
			if len(mutations) > 0 {
				endOpts.Attributes = map[string]any{"messageListMutations": mutations}
			}
			processorSpan.End(endOpts)
		}

		_ = index // suppress unused warning if observability is nil
	}

	return messageList, nil
}

// ---------------------------------------------------------------------------
// ProcessPart
// ---------------------------------------------------------------------------

// ProcessPart processes a stream part through all output processors with state management.
//
// Corresponds to TS: async processPart(part, processorStates, observabilityContext?, requestContext?, messageList?, retryCount, writer?)
func (pr *ProcessorRunner) ProcessPart(
	part ChunkType,
	processorStates *sync.Map,
	requestCtx *requestcontext.RequestContext,
	messageList *MessageList,
	retryCount int,
	writer ProcessorStreamWriter,
	observabilityContext ...*obstypes.ObservabilityContext,
) ProcessPartResult {
	var obsCtx *obstypes.ObservabilityContext
	if len(observabilityContext) > 0 {
		obsCtx = observabilityContext[0]
	}
	if len(pr.OutputProcessors) == 0 {
		return ProcessPartResult{Part: &part, Blocked: false}
	}

	processedPart := &part
	isFinishChunk := part.Type == "finish"

	for index, processorOrWorkflow := range pr.OutputProcessors {
		// Handle workflows for stream processing
		if wf, ok := processorOrWorkflow.(ProcessorWorkflow); ok {
			if processedPart == nil {
				continue
			}
			workflowID := wf.GetID()
			stateVal, _ := processorStates.LoadOrStore(workflowID, NewProcessorState())
			state := stateVal.(*ProcessorState)
			state.AddInputPart(*processedPart)

			result, err := pr.executeWorkflowAsProcessor(
				wf,
				ProcessorStepOutput{
					Phase:       ProcessorPhaseOutputStream,
					Part:        processedPart,
					StreamParts: anySliceFromChunks(state.StreamParts),
					State:       state.CustomState,
					MessageList: messageList,
				},
				obsCtx,
				requestCtx,
				nil,
				nil,
			)
			if err != nil {
				if tw, ok := err.(*TripWire); ok {
					return ProcessPartResult{
						Part:            nil,
						Blocked:         true,
						Reason:          tw.Message,
						TripwireOptions: tw.Options,
						ProcessorID:     tw.ProcessorID,
					}
				}
				pr.logger.Error(fmt.Sprintf("[Agent:%s] - Output processor workflow %s failed:", pr.agentName, workflowID), err)
				continue
			}

			if result != nil && result.Part != nil {
				if chunk, ok := result.Part.(*ChunkType); ok {
					processedPart = chunk
				} else if chunk, ok := result.Part.(ChunkType); ok {
					processedPart = &chunk
				}
			}
			state.AddOutputPart(processedPart)
			continue
		}

		// Handle regular processor
		processor, ok := processorOrWorkflow.(OutputProcessor)
		if !ok {
			continue
		}

		if processedPart == nil {
			continue
		}

		// Get or create state for this processor, with span creation for legacy processors
		stateVal, loaded := processorStates.LoadOrStore(processor.ID(), (*ProcessorState)(nil))
		if !loaded || stateVal == nil {
			state := NewProcessorStateWithOptions(&ProcessorStateOptions{
				ProcessorName:        processor.Name(),
				ObservabilityContext: obsCtx,
				ProcessorIndex:       index,
				CreateSpan:           true,
			})
			processorStates.Store(processor.ID(), state)
			stateVal = state
		}
		state := stateVal.(*ProcessorState)
		state.AddInputPart(*processedPart)

		abort := func(reason string, options *TripWireOptions) error {
			if reason == "" {
				reason = fmt.Sprintf("Stream part blocked by %s", processor.ID())
			}
			return NewTripWire(reason, options, processor.ID())
		}

		result, err := processor.ProcessOutputStream(ProcessOutputStreamArgs{
			ProcessorContext: ProcessorContext{
				Abort:          abort,
				RequestContext: requestCtx,
				RetryCount:     retryCount,
				Writer:         writer,
			},
			Part:        *processedPart,
			StreamParts: state.StreamParts,
			State:       state.CustomState,
			MessageList: messageList,
		})
		if err != nil {
			if tw, ok := err.(*TripWire); ok {
				// End span with blocked metadata
				if state.Span != nil {
					state.Span.End(&obstypes.EndSpanOptions{
						Metadata: map[string]any{
							"blocked": true,
							"reason":  tw.Message,
							"retry":   tw.Options != nil && tw.Options.Retry,
						},
					})
				}
				return ProcessPartResult{
					Part:            nil,
					Blocked:         true,
					Reason:          tw.Message,
					TripwireOptions: tw.Options,
					ProcessorID:     processor.ID(),
				}
			}
			// End span with error
			if state.Span != nil {
				state.Span.Error(obstypes.ErrorSpanOptions{
					Error:   err,
					EndSpan: true,
				})
			}
			pr.logger.Error(fmt.Sprintf("[Agent:%s] - Output processor %s failed:", pr.agentName, processor.ID()), err)
			continue
		}

		processedPart = result
		state.AddOutputPart(processedPart)
	}

	// If this was a finish chunk, end all processor spans AFTER processing
	if isFinishChunk {
		processorStates.Range(func(key, value any) bool {
			state := value.(*ProcessorState)
			if state.Span != nil {
				state.Span.End(&obstypes.EndSpanOptions{
					Output: state.GetFinalOutput(),
				})
			} else {
				_ = state.GetFinalOutput()
			}
			return true
		})
	}

	return ProcessPartResult{Part: processedPart, Blocked: false}
}

// ---------------------------------------------------------------------------
// RunInputProcessors
// ---------------------------------------------------------------------------

// RunInputProcessors runs processInput for all input processors.
//
// Corresponds to TS: async runInputProcessors(messageList, observabilityContext?, requestContext?, retryCount)
func (pr *ProcessorRunner) RunInputProcessors(
	messageList *MessageList,
	requestCtx *requestcontext.RequestContext,
	retryCount int,
	observabilityContext ...*obstypes.ObservabilityContext,
) (*MessageList, error) {
	var obsCtx *obstypes.ObservabilityContext
	if len(observabilityContext) > 0 {
		obsCtx = observabilityContext[0]
	}

	for index, processorOrWorkflow := range pr.InputProcessors {
		processableMessages := messageList.GetInputDB()
		inputIDs := extractMessageIDs(processableMessages)
		check := messageList.MakeMessageSourceChecker()

		// Handle workflow as processor
		if wf, ok := processorOrWorkflow.(ProcessorWorkflow); ok {
			currentSystemMessages := messageList.GetAllSystemMessages()
			_, err := pr.executeWorkflowAsProcessor(
				wf,
				ProcessorStepOutput{
					Phase:       ProcessorPhaseInput,
					MessageList: messageList,
				},
				obsCtx,
				requestCtx,
				nil,
				nil,
			)
			_ = currentSystemMessages // system messages passed via ProcessorStepOutput
			if err != nil {
				return nil, err
			}
			continue
		}

		// Handle regular processor
		processor, ok := processorOrWorkflow.(InputProcessor)
		if !ok {
			continue
		}

		abort := func(reason string, options *TripWireOptions) error {
			if reason == "" {
				reason = fmt.Sprintf("Tripwire triggered by %s", processor.ID())
			}
			return NewTripWire(reason, options, processor.ID())
		}

		// Create processor span for observability
		var processorSpan obstypes.Span
		if obsCtx != nil {
			currentSpan := obsCtx.Tracing.CurrentSpan
			if currentSpan != nil {
				parentSpan := currentSpan.FindParent(obstypes.SpanTypeAgentRun)
				if parentSpan == nil {
					parentSpan = currentSpan
				}
				entityType := obstypes.EntityTypeInputProcessor
				processorSpan = parentSpan.CreateChildSpan(obstypes.ChildSpanOptions{
					CreateBaseOptions: obstypes.CreateBaseOptions{
						Type:       obstypes.SpanTypeProcessorRun,
						Name:       fmt.Sprintf("input processor: %s", processor.ID()),
						EntityType: &entityType,
						EntityID:   processor.ID(),
						EntityName: processor.Name(),
						Attributes: map[string]any{
							"processorExecutor": "legacy",
							"processorIndex":    index,
						},
					},
					Input: processableMessages,
				})
			}
		}

		// Start recording MessageList mutations for this processor
		messageList.StartRecording()

		// Get all system messages to pass to the processor
		currentSystemMessages := messageList.GetAllSystemMessages()

		processorState := pr.getProcessorState(processor.ID())

		messages, returnedList, withSys, err := processor.ProcessInput(ProcessInputArgs{
			ProcessorMessageContext: ProcessorMessageContext{
				ProcessorContext: ProcessorContext{
					Abort:          abort,
					RequestContext: requestCtx,
					RetryCount:     retryCount,
				},
				MessageList: messageList,
			},
			SystemMessages: currentSystemMessages,
			State:          processorState.CustomState,
		})
		if err != nil {
			messageList.StopRecording()
			return nil, err
		}

		// Stop recording and get mutations
		mutations := messageList.StopRecording()

		// Handle return types
		if returnedList != nil {
			if returnedList != messageList {
				return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "PROCESSOR_RETURNED_EXTERNAL_MESSAGE_LIST",
					Domain:   mastraerror.ErrorDomainAgent,
					Category: mastraerror.ErrorCategoryUser,
					Text:     fmt.Sprintf("Processor %s returned a MessageList instance other than the one that was passed in.", processor.ID()),
				})
			}
			// MessageList was mutated in place
			if len(mutations) > 0 {
				processableMessages = returnedList.GetInputDB()
			}
		} else if withSys != nil {
			// Handle ProcessInputResultWithSystemMessages
			// Replace system messages with the modified ones
			messageList.ReplaceAllSystemMessages(withSys.SystemMessages)

			// Handle regular messages
			if withSys.Messages != nil {
				applyMessagesWithSystemHandling(withSys.Messages, messageList, inputIDs, check, "input")
			}
			processableMessages = messageList.GetInputDB()
		} else if messages != nil {
			// Processor returned an array — apply to messageList
			applyMessagesWithSystemHandling(messages, messageList, inputIDs, check, "input")
			processableMessages = messageList.GetInputDB()
		}

		// End the processor span
		if processorSpan != nil {
			endOpts := &obstypes.EndSpanOptions{
				Output: processableMessages,
			}
			if len(mutations) > 0 {
				endOpts.Attributes = map[string]any{"messageListMutations": mutations}
			}
			processorSpan.End(endOpts)
		}
	}

	return messageList, nil
}

// ---------------------------------------------------------------------------
// RunProcessInputStep
// ---------------------------------------------------------------------------

// RunProcessInputStep runs processInputStep for all processors that implement it.
// Called at each step of the agentic loop, before the LLM is invoked.
//
// Corresponds to TS: async runProcessInputStep(args: RunProcessInputStepArgs)
func (pr *ProcessorRunner) RunProcessInputStep(args RunProcessInputStepArgs) (*RunProcessInputStepResult, error) {
	messageList := args.MessageList
	observabilityContext := args.ObservabilityContext

	// Initialize with all provided values
	stepInput := &RunProcessInputStepResult{
		Tools:            args.Tools,
		ToolChoice:       args.ToolChoice,
		Model:            args.Model,
		ActiveTools:      args.ActiveTools,
		ProviderOptions:  args.ProviderOptions,
		ModelSettings:    args.ModelSettings,
		StructuredOutput: args.StructuredOutput,
		RetryCount:       args.RetryCount,
	}

	// Append the trailing assistant guard when the resolved model is Claude 4.6
	processors := pr.InputProcessors
	if stepInput.Model != nil && IsMaybeClaude46(stepInput.Model) {
		processors = append(append([]any{}, processors...), NewTrailingAssistantGuard())
	}

	for index, processorOrWorkflow := range processors {
		processableMessages := messageList.GetAllDB()
		idsBeforeProcessing := extractMessageIDs(processableMessages)
		check := messageList.MakeMessageSourceChecker()

		// Handle workflow as processor with inputStep phase
		if wf, ok := processorOrWorkflow.(ProcessorWorkflow); ok {
			currentSystemMessages := messageList.GetAllSystemMessages()
			result, err := pr.executeWorkflowAsProcessor(
				wf,
				ProcessorStepOutput{
					Phase:            ProcessorPhaseInputStep,
					MessageList:      messageList,
					Model:            stepInput.Model,
					Tools:            stepInput.Tools,
					ToolChoice:       stepInput.ToolChoice,
					ActiveTools:      stepInput.ActiveTools,
					ProviderOptions:  stepInput.ProviderOptions,
					ModelSettings:    stepInput.ModelSettings,
					StructuredOutput: stepInput.StructuredOutput,
				},
				observabilityContext,
				args.RequestContext,
				args.Writer,
				args.AbortSignal,
			)
			_ = currentSystemMessages // system messages passed via ProcessorStepOutput
			if err != nil {
				return nil, err
			}
			if result != nil {
				applyStepOutputToResult(result, stepInput)
			}
			continue
		}

		// Handle regular processor
		processor, ok := processorOrWorkflow.(InputProcessor)
		if !ok {
			continue
		}

		abort := func(reason string, options *TripWireOptions) error {
			if reason == "" {
				reason = fmt.Sprintf("Tripwire triggered by %s", processor.ID())
			}
			return NewTripWire(reason, options, processor.ID())
		}

		// Get all system messages to pass to the processor
		currentSystemMessages := messageList.GetAllSystemMessages()

		// Create processor span for observability
		// Use the current span (the step span) as the parent for processor spans.
		var processorSpan obstypes.Span
		if observabilityContext != nil {
			currentSpan := observabilityContext.Tracing.CurrentSpan
			if currentSpan != nil {
				entityType := obstypes.EntityTypeInputStepProcessor
				processorSpan = currentSpan.CreateChildSpan(obstypes.ChildSpanOptions{
					CreateBaseOptions: obstypes.CreateBaseOptions{
						Type:       obstypes.SpanTypeProcessorRun,
						Name:       fmt.Sprintf("input step processor: %s", processor.ID()),
						EntityType: &entityType,
						EntityID:   processor.ID(),
						EntityName: processor.Name(),
						Attributes: map[string]any{
							"processorExecutor": "legacy",
							"processorIndex":    index,
						},
					},
					Input: map[string]any{
						"messages":   processableMessages,
						"stepNumber": args.StepNumber,
					},
				})
			}
		}

		// Start recording MessageList mutations for this processor
		messageList.StartRecording()

		processorState := pr.getProcessorState(processor.ID())

		stepResult, messages, err := processor.ProcessInputStep(ProcessInputStepArgs{
			ProcessorMessageContext: ProcessorMessageContext{
				ProcessorContext: ProcessorContext{
					Abort:          abort,
					RequestContext: args.RequestContext,
					RetryCount:     args.RetryCount,
					Writer:         args.Writer,
					AbortSignal:    args.AbortSignal,
				},
				MessageList: messageList,
			},
			StepNumber:       args.StepNumber,
			Steps:            args.Steps,
			SystemMessages:   currentSystemMessages,
			State:            processorState.CustomState,
			Model:            stepInput.Model,
			Tools:            stepInput.Tools,
			ToolChoice:       stepInput.ToolChoice,
			ActiveTools:      stepInput.ActiveTools,
			ProviderOptions:  stepInput.ProviderOptions,
			ModelSettings:    stepInput.ModelSettings,
			StructuredOutput: stepInput.StructuredOutput,
		})
		if err != nil {
			// Stop recording on error
			messageList.StopRecording()

			if tw, ok := err.(*TripWire); ok {
				if processorSpan != nil {
					processorSpan.End(&obstypes.EndSpanOptions{
						Metadata: map[string]any{"blocked": true, "reason": tw.Message},
					})
				}
				return nil, tw
			}
			if processorSpan != nil {
				processorSpan.Error(obstypes.ErrorSpanOptions{
					Error:   err,
					EndSpan: true,
				})
			}
			pr.logger.Error(fmt.Sprintf("[Agent:%s] - Input step processor %s failed:", pr.agentName, processor.ID()), err)
			return nil, err
		}

		if messages != nil {
			// Processor returned messages array — apply to messageList
			applyMessagesToMessageList(messages, messageList, idsBeforeProcessing, check, "input")
		}

		if stepResult != nil {
			// Apply step result fields to stepInput
			if stepResult.Tools != nil {
				stepInput.Tools = stepResult.Tools
			}
			if stepResult.ToolChoice != nil {
				stepInput.ToolChoice = stepResult.ToolChoice
			}
			if stepResult.ActiveTools != nil {
				stepInput.ActiveTools = stepResult.ActiveTools
			}
			if stepResult.ProviderOptions != nil {
				stepInput.ProviderOptions = stepResult.ProviderOptions
			}
			if stepResult.ModelSettings != nil {
				stepInput.ModelSettings = stepResult.ModelSettings
			}
			if stepResult.StructuredOutput != nil {
				stepInput.StructuredOutput = stepResult.StructuredOutput
			}
			if stepResult.RetryCount != nil {
				stepInput.RetryCount = *stepResult.RetryCount
			}
			if stepResult.Messages != nil {
				// Apply messages to messageList
				applyMessagesToMessageList(stepResult.Messages, messageList, idsBeforeProcessing, check, "input")
			}
			if stepResult.SystemMessages != nil {
				// Replace system messages on messageList
				messageList.ReplaceAllSystemMessages(stepResult.SystemMessages)
			}
		}

		// Stop recording and get mutations for this processor
		mutations := messageList.StopRecording()

		// End the processor span
		if processorSpan != nil {
			endOpts := &obstypes.EndSpanOptions{
				Output: map[string]any{
					"messages":       messageList.GetAllDB(),
					"systemMessages": messageList.GetAllSystemMessages(),
				},
			}
			if len(mutations) > 0 {
				endOpts.Attributes = map[string]any{"messageListMutations": mutations}
			}
			processorSpan.End(endOpts)
		}
	}

	return stepInput, nil
}

// ---------------------------------------------------------------------------
// RunProcessOutputStep
// ---------------------------------------------------------------------------

// RunProcessOutputStepArgs holds arguments for RunProcessOutputStep.
type RunProcessOutputStepArgs struct {
	Steps                []StepResult
	Messages             []MastraDBMessage
	MessageList          *MessageList
	StepNumber           int
	FinishReason         string
	ToolCalls            []ToolCallInfo
	Text                 string
	RequestContext       *requestcontext.RequestContext
	RetryCount           int
	Writer               ProcessorStreamWriter
	ObservabilityContext *obstypes.ObservabilityContext
}

// RunProcessOutputStep runs processOutputStep for all processors that implement it.
// Called after each LLM response in the agentic loop, before tool execution.
//
// Corresponds to TS: async runProcessOutputStep(args)
func (pr *ProcessorRunner) RunProcessOutputStep(args RunProcessOutputStepArgs) (*MessageList, error) {
	messageList := args.MessageList
	observabilityContext := args.ObservabilityContext

	for index, processorOrWorkflow := range pr.OutputProcessors {
		processableMessages := messageList.GetAllDB()
		idsBeforeProcessing := extractMessageIDs(processableMessages)
		check := messageList.MakeMessageSourceChecker()

		// Handle workflow as processor with outputStep phase
		if wf, ok := processorOrWorkflow.(ProcessorWorkflow); ok {
			currentSystemMessages := messageList.GetAllSystemMessages()
			_, err := pr.executeWorkflowAsProcessor(
				wf,
				ProcessorStepOutput{
					Phase:       ProcessorPhaseOutputStep,
					MessageList: messageList,
				},
				observabilityContext,
				args.RequestContext,
				args.Writer,
				nil,
			)
			_ = currentSystemMessages // system messages passed via ProcessorStepOutput
			if err != nil {
				return nil, err
			}
			continue
		}

		// Handle regular processor
		processor, ok := processorOrWorkflow.(OutputProcessor)
		if !ok {
			continue
		}

		abort := func(reason string, options *TripWireOptions) error {
			if reason == "" {
				reason = fmt.Sprintf("Tripwire triggered by %s", processor.ID())
			}
			return NewTripWire(reason, options, processor.ID())
		}

		// Create processor span for observability
		var processorSpan obstypes.Span
		if observabilityContext != nil {
			currentSpan := observabilityContext.Tracing.CurrentSpan
			if currentSpan != nil {
				parentSpan := currentSpan.FindParent(obstypes.SpanTypeAgentRun)
				if parentSpan == nil {
					parentSpan = currentSpan
				}
				entityType := obstypes.EntityTypeOutputStepProcessor
				processorSpan = parentSpan.CreateChildSpan(obstypes.ChildSpanOptions{
					CreateBaseOptions: obstypes.CreateBaseOptions{
						Type:       obstypes.SpanTypeProcessorRun,
						Name:       fmt.Sprintf("output step processor: %s", processor.ID()),
						EntityType: &entityType,
						EntityID:   processor.ID(),
						EntityName: processor.Name(),
						Attributes: map[string]any{
							"processorExecutor": "legacy",
							"processorIndex":    index,
						},
					},
					Input: map[string]any{
						"messages":     processableMessages,
						"stepNumber":   args.StepNumber,
						"finishReason": args.FinishReason,
						"toolCalls":    args.ToolCalls,
						"text":         args.Text,
					},
				})
			}
		}

		// Start recording MessageList mutations for this processor
		messageList.StartRecording()

		// Get all system messages to pass to the processor
		currentSystemMessages := messageList.GetAllSystemMessages()

		processorState := pr.getProcessorState(processor.ID())

		messages, returnedList, err := processor.ProcessOutputStep(ProcessOutputStepArgs{
			ProcessorMessageContext: ProcessorMessageContext{
				ProcessorContext: ProcessorContext{
					Abort:          abort,
					RequestContext: args.RequestContext,
					RetryCount:     args.RetryCount,
					Writer:         args.Writer,
				},
				Messages:    args.Messages,
				MessageList: messageList,
			},
			StepNumber:     args.StepNumber,
			FinishReason:   args.FinishReason,
			ToolCalls:      args.ToolCalls,
			Text:           args.Text,
			SystemMessages: currentSystemMessages,
			Steps:          args.Steps,
			State:          processorState.CustomState,
		})
		if err != nil {
			// Stop recording on error
			messageList.StopRecording()

			if tw, ok := err.(*TripWire); ok {
				if processorSpan != nil {
					processorSpan.End(&obstypes.EndSpanOptions{
						Metadata: map[string]any{
							"blocked":  true,
							"reason":   tw.Message,
							"retry":    tw.Options != nil && tw.Options.Retry,
							"metadata": tw.Options,
						},
					})
				}
				return nil, tw
			}
			if processorSpan != nil {
				processorSpan.Error(obstypes.ErrorSpanOptions{
					Error:   err,
					EndSpan: true,
				})
			}
			pr.logger.Error(fmt.Sprintf("[Agent:%s] - Output step processor %s failed:", pr.agentName, processor.ID()), err)
			return nil, err
		}

		// Stop recording and get mutations for this processor
		mutations := messageList.StopRecording()

		// Handle return types
		if returnedList != nil {
			if returnedList != messageList {
				return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "PROCESSOR_RETURNED_EXTERNAL_MESSAGE_LIST",
					Domain:   mastraerror.ErrorDomainAgent,
					Category: mastraerror.ErrorCategoryUser,
					Text:     fmt.Sprintf("Processor %s returned a MessageList instance other than the one that was passed in.", processor.ID()),
				})
			}
			// Processor returned the same messageList - mutations have been applied
		} else if messages != nil {
			// Processor returned an array — apply changes to messageList
			applyMessagesWithSystemHandling(messages, messageList, idsBeforeProcessing, check, "response")
		}

		// End the processor span
		if processorSpan != nil {
			endOpts := &obstypes.EndSpanOptions{
				Output: messageList.GetAllDB(),
			}
			if len(mutations) > 0 {
				endOpts.Attributes = map[string]any{"messageListMutations": mutations}
			}
			processorSpan.End(endOpts)
		}
	}

	return messageList, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// anySliceFromChunks converts []ChunkType to []any for ProcessorStepOutput.
func anySliceFromChunks(chunks []ChunkType) []any {
	result := make([]any, len(chunks))
	for i, c := range chunks {
		result[i] = c
	}
	return result
}

// applyStepOutputToResult copies non-nil fields from a ProcessorStepOutput
// onto a RunProcessInputStepResult.
func applyStepOutputToResult(output *ProcessorStepOutput, result *RunProcessInputStepResult) {
	if output.Model != nil {
		result.Model = output.Model
	}
	if output.Tools != nil {
		result.Tools = output.Tools
	}
	if output.ToolChoice != nil {
		result.ToolChoice = output.ToolChoice
	}
	if output.ActiveTools != nil {
		result.ActiveTools = output.ActiveTools
	}
	if output.ProviderOptions != nil {
		result.ProviderOptions = output.ProviderOptions
	}
	if output.ModelSettings != nil {
		result.ModelSettings = output.ModelSettings
	}
	if output.StructuredOutput != nil {
		result.StructuredOutput = output.StructuredOutput
	}
	if output.RetryCount != nil {
		result.RetryCount = *output.RetryCount
	}
}

// ---------------------------------------------------------------------------
// MessageList manipulation helpers
// ---------------------------------------------------------------------------

// extractMessageIDs extracts IDs from a slice of MastraDBMessage.
func extractMessageIDs(messages []MastraDBMessage) []string {
	ids := make([]string, len(messages))
	for i, m := range messages {
		ids[i] = m.ID
	}
	return ids
}

// applyMessagesToMessageList applies a processor-returned message array to a MessageList.
// This handles removing deleted messages and re-adding remaining ones with correct sources.
//
// Corresponds to TS: ProcessorRunner.applyMessagesToMessageList(messages, messageList, idsBeforeProcessing, check, defaultSource)
func applyMessagesToMessageList(
	messages []MastraDBMessage,
	messageList *MessageList,
	idsBeforeProcessing []string,
	check *MessageSourceChecker,
	defaultSource string,
) {
	// Find deleted IDs (IDs present before but not in the result)
	resultIDSet := make(map[string]bool, len(messages))
	for _, m := range messages {
		resultIDSet[m.ID] = true
	}
	var deletedIDs []string
	for _, id := range idsBeforeProcessing {
		if !resultIDSet[id] {
			deletedIDs = append(deletedIDs, id)
		}
	}
	if len(deletedIDs) > 0 {
		messageList.RemoveByIds(deletedIDs)
	}

	// Re-add messages with correct sources
	for _, message := range messages {
		messageList.RemoveByIds([]string{message.ID})
		if message.Role == "system" {
			systemText := extractSystemText(message)
			messageList.AddSystem(systemText)
		} else {
			source := check.GetSource(message)
			if source == "" {
				source = defaultSource
			}
			messageList.Add(message, source)
		}
	}
}

// applyMessagesWithSystemHandling applies a processor-returned message array to a MessageList,
// separating system messages from non-system messages for proper handling.
// This is the full TS pattern that handles both system and non-system messages.
func applyMessagesWithSystemHandling(
	messages []MastraDBMessage,
	messageList *MessageList,
	idsBeforeProcessing []string,
	check *MessageSourceChecker,
	defaultSource string,
) {
	// Find deleted IDs
	resultIDSet := make(map[string]bool, len(messages))
	for _, m := range messages {
		resultIDSet[m.ID] = true
	}
	var deletedIDs []string
	for _, id := range idsBeforeProcessing {
		if !resultIDSet[id] {
			deletedIDs = append(deletedIDs, id)
		}
	}
	if len(deletedIDs) > 0 {
		messageList.RemoveByIds(deletedIDs)
	}

	// Separate system messages from non-system messages
	var systemMessages []MastraDBMessage
	var nonSystemMessages []MastraDBMessage
	for _, m := range messages {
		if m.Role == "system" {
			systemMessages = append(systemMessages, m)
		} else {
			nonSystemMessages = append(nonSystemMessages, m)
		}
	}

	// Add system messages using AddSystem
	for _, sysMsg := range systemMessages {
		systemText := extractSystemText(sysMsg)
		messageList.AddSystem(systemText)
	}

	// Add non-system messages normally
	for _, message := range nonSystemMessages {
		messageList.RemoveByIds([]string{message.ID})
		source := check.GetSource(message)
		if source == "" {
			source = defaultSource
		}
		messageList.Add(message, source)
	}
}

// extractSystemText extracts the text content from a system message.
// Tries Content.Content first, then concatenates text parts.
// Matches TS: (sysMsg.content.content as string) ?? sysMsg.content.parts?.map(p => p.type === 'text' ? p.text : '').join('\n')
func extractSystemText(msg MastraDBMessage) string {
	if msg.Content.Content != "" {
		return msg.Content.Content
	}
	if len(msg.Content.Parts) > 0 {
		var parts []string
		for _, p := range msg.Content.Parts {
			if p.Type == "text" {
				parts = append(parts, p.Text)
			}
		}
		if len(parts) > 0 {
			result := ""
			for i, p := range parts {
				if i > 0 {
					result += "\n"
				}
				result += p
			}
			return result
		}
	}
	return ""
}
