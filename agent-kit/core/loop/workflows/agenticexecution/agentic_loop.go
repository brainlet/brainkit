// Ported from: packages/core/src/loop/workflows/agentic-loop/index.ts
package agenticexecution

import (
	"crypto/rand"
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported (agentic-loop-specific)
// ---------------------------------------------------------------------------

// StepResultContent represents content parts from a step result.
type StepResultContent = []map[string]any

// StepResultFull holds a full step result for stopWhen evaluation.
type StepResultFull struct {
	Content            StepResultContent `json:"content,omitempty"`
	Usage              map[string]any    `json:"usage,omitempty"`
	FinishReason       string            `json:"finishReason,omitempty"`
	Warnings           []any             `json:"warnings,omitempty"`
	Request            map[string]any    `json:"request,omitempty"`
	Response           map[string]any    `json:"response,omitempty"`
	Text               string            `json:"text,omitempty"`
	Reasoning          []any             `json:"reasoning,omitempty"`
	ReasoningText      string            `json:"reasoningText,omitempty"`
	Files              []any             `json:"files,omitempty"`
	ToolCalls          []any             `json:"toolCalls,omitempty"`
	ToolResults        []any             `json:"toolResults,omitempty"`
	Sources            []any             `json:"sources,omitempty"`
	StaticToolCalls    []any             `json:"staticToolCalls,omitempty"`
	DynamicToolCalls   []any             `json:"dynamicToolCalls,omitempty"`
	StaticToolResults  []any             `json:"staticToolResults,omitempty"`
	DynamicToolResults []any             `json:"dynamicToolResults,omitempty"`
	ProviderMetadata   any               `json:"providerMetadata,omitempty"`
}

// IterationContext holds context passed to onIterationComplete callbacks.
type IterationContext struct {
	Iteration     int            `json:"iteration"`
	MaxIterations int            `json:"maxIterations,omitempty"`
	Text          string         `json:"text"`
	ToolCalls     []ToolCallRef  `json:"toolCalls,omitempty"`
	ToolResults   []ToolResultRef `json:"toolResults,omitempty"`
	IsFinal       bool           `json:"isFinal"`
	FinishReason  string         `json:"finishReason"`
	RunID         string         `json:"runId"`
	ThreadID      string         `json:"threadId,omitempty"`
	ResourceID    string         `json:"resourceId,omitempty"`
	AgentID       string         `json:"agentId"`
	AgentName     string         `json:"agentName"`
	Messages      []any          `json:"messages,omitempty"`
}

// ToolCallRef is a simplified tool call reference for callbacks.
type ToolCallRef struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// ToolResultRef is a simplified tool result reference for callbacks.
type ToolResultRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Result any    `json:"result,omitempty"`
	Error  any    `json:"error,omitempty"`
}

// IterationResult is the optional return from onIterationComplete.
type IterationResult struct {
	Feedback string `json:"feedback,omitempty"`
	Continue *bool  `json:"continue,omitempty"`
}

// OnIterationCompleteHandler is the callback type for iteration completion.
type OnIterationCompleteHandler func(ctx IterationContext) (*IterationResult, error)

// ---------------------------------------------------------------------------
// AgenticLoopParams
// ---------------------------------------------------------------------------

// AgenticLoopParams extends OuterLLMRun with controller and writer for the
// agentic loop.
type AgenticLoopParams struct {
	OuterLLMRun
	OnIterationComplete OnIterationCompleteHandler       `json:"-"`
	StopWhen            []func(args map[string]any) bool `json:"-"`
}

// ---------------------------------------------------------------------------
// CreateAgenticLoopWorkflow
// ---------------------------------------------------------------------------

// CreateAgenticLoopWorkflow builds the outer agentic loop workflow that
// repeatedly executes the agentic execution workflow until a stop condition
// is met.
//
// The loop:
//  1. Creates the agentic execution workflow (the inner loop).
//  2. Uses dowhile to repeatedly execute the inner workflow.
//  3. On each iteration:
//     a. Extracts new content from the iteration (slicing past previous).
//     b. Builds a StepResultFull with all the relevant data.
//     c. Accumulates steps for stopWhen evaluation.
//     d. Evaluates stopWhen conditions (if any).
//     e. Calls onIterationComplete hook (if configured):
//        - If feedback is returned AND still continuing: add feedback to
//          message list.
//        - If continue=false with feedback: set pendingFeedbackStop
//          (allow one more LLM turn then stop).
//        - If continue=false without feedback: stop immediately.
//     f. Checks delegation bail flag.
//     g. Emits step-finish chunk (unless tripwire with no steps).
//     h. Returns whether to continue the loop.
//  4. Commits the workflow.
func CreateAgenticLoopWorkflow(params AgenticLoopParams) *Workflow {
	// Track accumulated steps across iterations.
	accumulatedSteps := make([]StepResultFull, 0)
	// Track previous content to determine what's new.
	previousContentLength := 0
	// When continue:false + feedback, allow one more LLM turn then stop.
	pendingFeedbackStop := false

	// Create the inner execution workflow.
	agenticExecutionWorkflow := CreateAgenticExecutionWorkflow(params.OuterLLMRun)

	// dowhile condition: evaluate whether to continue looping.
	evaluateCondition := func(inputData map[string]any) bool {
		hasFinishedSteps := false

		if pendingFeedbackStop {
			hasFinishedSteps = true
			pendingFeedbackStop = false
		}

		// Extract content from nonUser messages.
		messages, _ := inputData["messages"].(map[string]any)
		var allContent []any
		if messages != nil {
			if nonUser, ok := messages["nonUser"].([]any); ok {
				for _, msg := range nonUser {
					if msgMap, ok := msg.(map[string]any); ok {
						if content, ok := msgMap["content"]; ok {
							allContent = append(allContent, content)
						}
					}
				}
			}
		}

		// Extract new content for this iteration.
		var currentContent []map[string]any
		if len(allContent) > previousContentLength {
			for _, c := range allContent[previousContentLength:] {
				if cm, ok := c.(map[string]any); ok {
					currentContent = append(currentContent, cm)
				}
			}
		}
		previousContentLength = len(allContent)

		// Filter tool result parts.
		var toolResultParts []map[string]any
		for _, part := range currentContent {
			if partType, ok := part["type"].(string); ok && partType == "tool-result" {
				toolResultParts = append(toolResultParts, part)
			}
		}

		// Build step result.
		output, _ := inputData["output"].(map[string]any)
		stepResult, _ := inputData["stepResult"].(map[string]any)
		metadata, _ := inputData["metadata"].(map[string]any)

		var usage map[string]any
		if output != nil {
			usage, _ = output["usage"].(map[string]any)
		}
		if usage == nil {
			usage = map[string]any{"inputTokens": 0, "outputTokens": 0, "totalTokens": 0}
		}

		finishReason := "unknown"
		if stepResult != nil {
			if r, ok := stepResult["reason"].(string); ok {
				finishReason = r
			}
		}

		warnings := []any{}
		if stepResult != nil {
			if w, ok := stepResult["warnings"].([]any); ok {
				warnings = w
			}
		}

		text := ""
		if output != nil {
			if t, ok := output["text"].(string); ok {
				text = t
			}
		}

		currentStepResult := StepResultFull{
			Content:      currentContent,
			Usage:        usage,
			FinishReason: finishReason,
			Warnings:     warnings,
			Text:         text,
			ToolResults:  toAnySlice(toolResultParts),
		}

		if metadata != nil {
			currentStepResult.Request, _ = metadata["request"].(map[string]any)
			modelID, _ := metadata["modelId"].(string)
			model, _ := metadata["model"].(string)
			if modelID == "" {
				modelID = model
			}
			currentStepResult.Response = map[string]any{
				"modelId": modelID,
			}
			for k, v := range metadata {
				if k != "request" {
					currentStepResult.Response[k] = v
				}
			}
			currentStepResult.ProviderMetadata = metadata["providerMetadata"]
		}

		if output != nil {
			currentStepResult.Reasoning, _ = output["reasoning"].([]any)
			currentStepResult.ReasoningText, _ = output["reasoningText"].(string)
			currentStepResult.Files, _ = output["files"].([]any)
			currentStepResult.ToolCalls, _ = output["toolCalls"].([]any)
			currentStepResult.Sources, _ = output["sources"].([]any)
		}

		accumulatedSteps = append(accumulatedSteps, currentStepResult)

		// Evaluate stopWhen conditions.
		isContinued := false
		if stepResult != nil {
			if ic, ok := stepResult["isContinued"].(bool); ok {
				isContinued = ic
			}
		}

		if len(params.StopWhen) > 0 && isContinued && len(accumulatedSteps) > 0 {
			for _, condition := range params.StopWhen {
				if condition(map[string]any{"steps": accumulatedSteps}) {
					hasFinishedSteps = true
					break
				}
			}
		}

		// Call onIterationComplete hook.
		if params.OnIterationComplete != nil {
			isFinal := !isContinued || hasFinishedSteps

			// Build tool call refs.
			var toolCallRefs []ToolCallRef
			if output != nil {
				if tcs, ok := output["toolCalls"].([]any); ok {
					for _, tc := range tcs {
						if tcMap, ok := tc.(map[string]any); ok {
							ref := ToolCallRef{
								ID:   getStr(tcMap, "toolCallId"),
								Name: getStr(tcMap, "toolName"),
							}
							if args, ok := tcMap["args"].(map[string]any); ok {
								ref.Args = args
							}
							toolCallRefs = append(toolCallRefs, ref)
						}
					}
				}
			}

			// Build tool result refs.
			var toolResultRefs []ToolResultRef
			if output != nil {
				if trs, ok := output["toolResults"].([]any); ok {
					for _, tr := range trs {
						if trMap, ok := tr.(map[string]any); ok {
							ref := ToolResultRef{
								ID:     getStr(trMap, "toolCallId"),
								Name:   getStr(trMap, "toolName"),
								Result: trMap["result"],
								Error:  trMap["error"],
							}
							toolResultRefs = append(toolResultRefs, ref)
						}
					}
				}
			}

			var threadID, resourceID string
			if params.Internal != nil {
				if internal, ok := params.Internal.(map[string]any); ok {
					threadID, _ = internal["threadId"].(string)
					resourceID, _ = internal["resourceId"].(string)
				}
			}

			iterCtx := IterationContext{
				Iteration:     len(accumulatedSteps),
				MaxIterations: params.MaxSteps,
				Text:          text,
				ToolCalls:     toolCallRefs,
				ToolResults:   toolResultRefs,
				IsFinal:       isFinal,
				FinishReason:  finishReason,
				RunID:         params.RunID,
				ThreadID:      threadID,
				ResourceID:    resourceID,
				AgentID:       params.AgentID,
				AgentName:     params.AgentName,
			}
			if ml, ok := params.MessageList.(MessageListFull); ok {
				dbMsgs := ml.GetAll().DB()
				iterCtx.Messages = make([]any, len(dbMsgs))
				for mi, m := range dbMsgs {
					iterCtx.Messages[mi] = m
				}
			}

			iterResult, err := params.OnIterationComplete(iterCtx)
			if err != nil {
				if logger := toLogger(params.Logger); logger != nil {
					logger.Error("Error in onIterationComplete hook:", err)
				}
			}

			if iterResult != nil {
				if iterResult.Feedback != "" && isContinued {
					// Add feedback as assistant message.
					feedbackMsg := map[string]any{
						"id":        generateUUID(),
						"createdAt": time.Now(),
						"type":      "text",
						"role":      "assistant",
						"content": map[string]any{
							"parts": []map[string]any{
								{
									"type": "text",
									"text": iterResult.Feedback,
								},
							},
							"metadata": map[string]any{
								"mode": "stream",
								"completionResult": map[string]any{
									"suppressFeedback": true,
								},
							},
							"format": 2,
						},
					}
					if ml, ok := params.MessageList.(MessageListFull); ok {
						ml.Add(feedbackMsg, "response")
					}

					if iterResult.Continue != nil && !*iterResult.Continue {
						pendingFeedbackStop = true
					} else if !hasFinishedSteps && params.MaxSteps > 0 && len(accumulatedSteps) < params.MaxSteps {
						hasFinishedSteps = false
						if stepResult != nil {
							stepResult["isContinued"] = true
						}
						isContinued = true
					}
				} else if iterResult.Continue != nil && !*iterResult.Continue && !hasFinishedSteps {
					hasFinishedSteps = true
				}
			}
		}

		// Check delegation bail flag.
		if !hasFinishedSteps {
			if params.Internal != nil {
				if internal, ok := params.Internal.(map[string]any); ok {
					if bailed, ok := internal["_delegationBailed"].(bool); ok && bailed {
						hasFinishedSteps = true
						internal["_delegationBailed"] = false
					}
				}
			}
		}

		// Update isContinued based on finished status.
		if stepResult != nil {
			if hasFinishedSteps {
				stepResult["isContinued"] = false
			}
		}

		// Emit step-finish chunk.
		// Skip if tripwire with no steps.
		var hasSteps bool
		if output != nil {
			if steps, ok := output["steps"].([]any); ok {
				hasSteps = len(steps) > 0
			}
		}
		reason := ""
		if stepResult != nil {
			reason, _ = stepResult["reason"].(string)
		}
		shouldEmitStepFinish := reason != "tripwire" || hasSteps

		if shouldEmitStepFinish {
			SafeEnqueue(params.Controller, map[string]any{
				"type":    "step-finish",
				"runId":   params.RunID,
				"from":    ChunkFromAGENT,
				"payload": inputData,
			})
		}

		if reason == "" {
			return false
		}

		// Return whether to continue.
		finalIsContinued := false
		if stepResult != nil {
			if ic, ok := stepResult["isContinued"].(bool); ok {
				finalIsContinued = ic
			}
		}
		return finalIsContinued
	}

	// Build the dowhile workflow.
	doWhileStep := &Step{
		ID: "dowhile",
		Execute: func(args StepExecuteArgs) (any, error) {
			current := args.InputData
			for {
				// Execute the inner workflow.
				result, err := agenticExecutionWorkflow.Execute(current)
				if err != nil {
					return nil, err
				}

				// Evaluate the condition.
				resultMap, ok := result.(map[string]any)
				if !ok {
					return result, nil
				}

				shouldContinue := evaluateCondition(resultMap)
				if !shouldContinue {
					return result, nil
				}

				current = result
			}
		},
	}

	return &Workflow{
		ID:    "agentic-loop",
		steps: []*Step{doWhileStep},
	}
}

// generateUUID generates a random UUID v4 string.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// toAnySlice converts []map[string]any to []any.
func toAnySlice(in []map[string]any) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}
