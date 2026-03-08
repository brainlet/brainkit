// Ported from: packages/core/src/loop/workflows/agentic-execution/index.ts
package agenticexecution

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// ToolSet is a stub for @internal/ai-sdk-v5.ToolSet.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 ToolSet remains local.
// model.ToolSet = map[string]Tool where Tool = any — same shape but different V5 context.
type ToolSet = map[string]any

// InternalSpans is a stub for ../../../observability.InternalSpans.
// Stub: real observability package defines span name constants differently.
// This constant is kept local for the workflow span naming pattern.
const InternalSpansWORKFLOW = "workflow"

// OuterLLMRun is a stub for ../../types.OuterLLMRun.
// Stub: can't import parent loop package (loop → loop/workflows → agenticexecution
// would create cycle). Uses simplified any-typed fields. Cycle risk + shape mismatch.
type OuterLLMRun struct {
	Models              []any          `json:"models"`
	Internal            any            `json:"_internal,omitempty"`
	MessageID           string         `json:"messageId"`
	RunID               string         `json:"runId"`
	MessageList         any            `json:"-"`
	Controller          any            `json:"-"`
	OutputWriter        any            `json:"-"`
	StreamState         any            `json:"-"`
	Tools               ToolSet        `json:"-"`
	ToolChoice          any            `json:"toolChoice,omitempty"`
	ModelSettings       any            `json:"modelSettings,omitempty"`
	Options             any            `json:"-"`
	Logger              any            `json:"-"`
	AgentID             string         `json:"agentId"`
	AgentName           string         `json:"agentName,omitempty"`
	MaxSteps            int            `json:"maxSteps,omitempty"`
	RequireToolApproval bool           `json:"requireToolApproval,omitempty"`
	ToolCallConcurrency int            `json:"toolCallConcurrency,omitempty"`
	IsTaskComplete      any            `json:"isTaskComplete,omitempty"`
	OutputProcessors    []any          `json:"-"`
	InputProcessors     []any          `json:"-"`
	ModelSpanTracker    any            `json:"-"`
	RequestContext      any            `json:"-"`
	ProcessorStates     map[string]any `json:"-"`
	Mastra              any            `json:"-"`
	// ExperimentalGenerateMessageID is a function to generate message IDs.
	ExperimentalGenerateMessageID func() string `json:"-"`
}

// Workflow is a stub for ../../../workflows.Workflow.
// Stub: can't import external workflows package. Real workflows.Workflow has a full graph-based
// execution engine (stepFlow, serializedStepFlow, Committed flag, sync.Map for runs, pub/sub events).
// This stub uses a simplified linear step chain with *Step (local type). Shape mismatch + different domain.
type Workflow struct {
	ID string

	// Internal step chain for execution.
	steps []*Step
	// Foreach config.
	foreachStep       *Step
	foreachConcurrency int
}

// RegisterMastra is a stub for workflow.__registerMastra.
func (w *Workflow) RegisterMastra(mastra any) {}

// DeleteWorkflowRunByID is a stub.
func (w *Workflow) DeleteWorkflowRunByID(runID string) error { return nil }

// Execute runs the workflow chain with given input data.
// This is a simplified synchronous execution model.
// TODO: implement full async workflow engine with suspend/resume.
func (w *Workflow) Execute(inputData any) (any, error) {
	current := inputData
	for _, step := range w.steps {
		result, err := step.Execute(StepExecuteArgs{InputData: current})
		if err != nil {
			return nil, err
		}
		current = result
	}
	return current, nil
}

// LLMIterationData is a stub for ../schema.LLMIterationData.
// Stub: can't import parent package loop/workflows (agenticexecution → workflows would create cycle).
// Real LLMIterationData has typed sub-structs (LLMIterationMessages, LLMIterationOutput,
// LLMIterationMetadata, LLMIterationStepResult) + additional fields (ProcessorRetryCount,
// ProcessorRetryFeedback, IsTaskCompleteCheckFailed). This stub uses any-typed fields. Shape mismatch + cycle.
type LLMIterationData struct {
	MessageID  string `json:"messageId"`
	Messages   any    `json:"messages"`
	Output     any    `json:"output"`
	Metadata   any    `json:"metadata"`
	StepResult any    `json:"stepResult"`
}

// ---------------------------------------------------------------------------
// CreateAgenticExecutionWorkflow
// ---------------------------------------------------------------------------

// CreateAgenticExecutionWorkflow builds the execution workflow that
// orchestrates a single iteration of the agentic loop:
//
//  1. capture-response-count: Snapshot the response model message count
//     BEFORE the LLM runs, to later add only NEW messages.
//  2. llmExecutionStep: Call the LLM (with fallback models and retry).
//  3. add-response-to-messagelist: Add new assistant messages to the list.
//  4. map-tool-calls: Extract tool calls from the LLM output.
//  5. toolCallStep (foreach): Execute each tool call (sequential if
//     approval/suspend required, otherwise concurrent).
//  6. llmMappingStep: Map tool results back and build chunks.
//  7. isTaskCompleteStep: Run completion scorers if configured.
//
// The concurrency of tool call execution depends on:
//   - toolCallConcurrency setting (default 10)
//   - Whether requireToolApproval is set (forces sequential)
//   - Whether any tool has suspendSchema (forces sequential)
//   - Whether any tool has requireApproval flag (forces sequential)
func CreateAgenticExecutionWorkflow(params OuterLLMRun) *Workflow {
	// Track existing response model count before each LLM call.
	existingResponseModelCount := 0

	// Determine tool call concurrency.
	toolCallConcurrency := 10
	if params.ToolCallConcurrency > 0 {
		toolCallConcurrency = params.ToolCallConcurrency
	}

	// Check for sequential execution requirements.
	hasRequireToolApproval := params.RequireToolApproval

	hasSuspendSchema := false
	hasRequireApproval := false

	if params.Tools != nil {
		for _, tool := range params.Tools {
			toolMap, ok := tool.(map[string]any)
			if !ok {
				continue
			}
			if _, ok := toolMap["hasSuspendSchema"]; ok {
				hasSuspendSchema = true
			}
			if _, ok := toolMap["requireApproval"]; ok {
				hasRequireApproval = true
			}
			if hasSuspendSchema || hasRequireApproval {
				break
			}
		}
	}

	sequentialExecutionRequired := hasRequireToolApproval || hasSuspendSchema || hasRequireApproval
	if sequentialExecutionRequired {
		toolCallConcurrency = 1
	}

	// Create the workflow steps.
	llmExecStep := CreateLLMExecutionStep(params)
	toolCallStep := CreateToolCallStep(params)
	llmMapStep := CreateLLMMappingStep(params, llmExecStep)
	isTaskComplStep := CreateIsTaskCompleteStep(params)

	// capture-response-count: Snapshot response count before LLM.
	captureResponseCountStep := &Step{
		ID: "capture-response-count",
		Execute: func(args StepExecuteArgs) (any, error) {
			// Snapshot the response model message count BEFORE the LLM runs.
			// This lets add-response-to-messagelist only add truly NEW messages.
			if ml, ok := params.MessageList.(MessageListFull); ok {
				existingResponseModelCount = len(ml.GetResponse().AIV5Model())
			} else {
				existingResponseModelCount = 0
			}
			return args.InputData, nil
		},
	}

	// add-response-to-messagelist: Add new assistant messages after LLM.
	addResponseStep := &Step{
		ID: "add-response-to-messagelist",
		Execute: func(args StepExecuteArgs) (any, error) {
			inputData, ok := args.InputData.(map[string]any)
			if !ok {
				return args.InputData, nil
			}
			messages, _ := inputData["messages"].(map[string]any)
			if messages != nil {
				if nonUser, ok := messages["nonUser"].([]any); ok {
					newMessages := make([]any, 0)
					if len(nonUser) > existingResponseModelCount {
						newMessages = nonUser[existingResponseModelCount:]
					}
					if len(newMessages) > 0 {
						if ml, ok := params.MessageList.(MessageListFull); ok {
							ml.Add(newMessages, "response")
						}
					}
				}
			}
			return inputData, nil
		},
	}

	// map-tool-calls: Extract tool calls from the LLM output.
	mapToolCallsStep := &Step{
		ID: "map-tool-calls",
		Execute: func(args StepExecuteArgs) (any, error) {
			inputData, ok := args.InputData.(map[string]any)
			if !ok {
				return []ToolCallInput{}, nil
			}
			output, _ := inputData["output"].(map[string]any)
			if output == nil {
				return []ToolCallInput{}, nil
			}
			toolCalls, _ := output["toolCalls"].([]any)
			if len(toolCalls) == 0 {
				toolCallMaps, _ := output["toolCalls"].([]map[string]any)
				if len(toolCallMaps) == 0 {
					return []ToolCallInput{}, nil
				}
				for _, tc := range toolCallMaps {
					toolCalls = append(toolCalls, tc)
				}
			}

			inputs := make([]ToolCallInput, 0, len(toolCalls))
			for _, tc := range toolCalls {
				tcMap, ok := tc.(map[string]any)
				if !ok {
					continue
				}
				input := ToolCallInput{
					ToolCallID: getStr(tcMap, "toolCallId"),
					ToolName:   getStr(tcMap, "toolName"),
				}
				if args, ok := tcMap["args"].(map[string]any); ok {
					input.Args = args
				}
				if pm, ok := tcMap["providerMetadata"].(map[string]any); ok {
					input.ProviderMetadata = pm
				}
				if pe, ok := tcMap["providerExecuted"].(bool); ok {
					input.ProviderExecuted = &pe
				}
				if out, ok := tcMap["output"]; ok {
					input.Output = out
				}
				inputs = append(inputs, input)
			}
			return inputs, nil
		},
	}

	// foreach-tool-calls: Execute each tool call with configured concurrency.
	// In a full implementation, this runs toolCallStep for each item in the
	// tool calls list with the specified concurrency.
	foreachToolCallsStep := &Step{
		ID: "foreach-tool-calls",
		Execute: func(args StepExecuteArgs) (any, error) {
			inputs, ok := args.InputData.([]ToolCallInput)
			if !ok {
				return []ToolCallOutput{}, nil
			}

			if len(inputs) == 0 {
				return []ToolCallOutput{}, nil
			}

			results := make([]ToolCallOutput, len(inputs))

			if toolCallConcurrency <= 1 || len(inputs) == 1 {
				// Sequential execution.
				for i, input := range inputs {
					result, err := toolCallStep.Execute(StepExecuteArgs{InputData: input})
					if err != nil {
						return nil, err
					}
					if tc, ok := result.(ToolCallOutput); ok {
						results[i] = tc
					}
				}
			} else {
				// Concurrent execution with semaphore.
				type indexedResult struct {
					index  int
					result ToolCallOutput
					err    error
				}
				ch := make(chan indexedResult, len(inputs))
				sem := make(chan struct{}, toolCallConcurrency)

				for i, input := range inputs {
					sem <- struct{}{}
					go func(idx int, inp ToolCallInput) {
						defer func() { <-sem }()
						result, err := toolCallStep.Execute(StepExecuteArgs{InputData: inp})
						if err != nil {
							ch <- indexedResult{index: idx, err: err}
							return
						}
						if tc, ok := result.(ToolCallOutput); ok {
							ch <- indexedResult{index: idx, result: tc}
						} else {
							ch <- indexedResult{index: idx}
						}
					}(i, input)
				}

				// Collect results preserving order.
				var firstErr error
				for range inputs {
					ir := <-ch
					if ir.err != nil && firstErr == nil {
						firstErr = ir.err
					}
					results[ir.index] = ir.result
				}
				if firstErr != nil {
					return nil, firstErr
				}
			}

			return results, nil
		},
	}

	// Build the workflow step chain.
	workflow := &Workflow{
		ID: "executionWorkflow",
		steps: []*Step{
			captureResponseCountStep,
			llmExecStep,
			addResponseStep,
			mapToolCallsStep,
			foreachToolCallsStep,
			llmMapStep,
			isTaskComplStep,
		},
	}

	return workflow
}

// getStr safely extracts a string from a map.
func getStr(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
