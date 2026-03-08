// Ported from: packages/core/src/loop/workflows/agentic-execution/is-task-complete-step.ts
package agenticexecution

import (
	"time"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// IsTaskCompleteRunResult is a stub for ../../../agent.IsTaskCompleteRunResult.
// Stub: real agent.IsTaskCompleteRunResult = agent.CompletionRunResult which has same top-level fields
// but its ScorerResult has additional fields (ScorerID, ScorerName, FinalResult) not present in local
// stub ScorerResult. Local ScorerResult uses Name instead of ScorerName. Shape mismatch in nested type.
type IsTaskCompleteRunResult struct {
	Complete         bool           `json:"complete"`
	Scorers          []ScorerResult `json:"scorers,omitempty"`
	TotalDuration    int64          `json:"totalDuration,omitempty"`
	TimedOut         bool           `json:"timedOut,omitempty"`
	CompletionReason string         `json:"completionReason,omitempty"`
}

// ScorerResult holds a single scorer's evaluation result.
type ScorerResult struct {
	Name     string  `json:"name"`
	Score    float64 `json:"score"`
	Passed   bool    `json:"passed"`
	Reason   string  `json:"reason,omitempty"`
	Duration int64   `json:"duration,omitempty"`
}

// StreamCompletionContext is a stub for ../../network/validation.StreamCompletionContext.
// Stub: real network.StreamCompletionContext has same fields but Messages is typed as
// []MastraDBMessage (map[string]any alias) instead of []any. Also ToolResultInfo.Result
// is typed as any in real vs map[string]any in stub. No import cycle exists, but switching
// would require all call sites to use network types (MastraDBMessage, ToolCallInfo, etc.).
type StreamCompletionContext struct {
	Iteration     int              `json:"iteration"`
	MaxIterations int              `json:"maxIterations,omitempty"`
	OriginalTask  string           `json:"originalTask"`
	CurrentText   string           `json:"currentText"`
	ToolCalls     []ToolCallInfo   `json:"toolCalls,omitempty"`
	Messages      []any            `json:"messages,omitempty"`
	ToolResults   []ToolResultInfo `json:"toolResults,omitempty"`
	AgentID       string           `json:"agentId"`
	AgentName     string           `json:"agentName"`
	RunID         string           `json:"runId"`
	ThreadID      string           `json:"threadId,omitempty"`
	ResourceID    string           `json:"resourceId,omitempty"`
	CustomContext map[string]any   `json:"customContext,omitempty"`
}

// ToolCallInfo holds tool call summary info for scorers.
type ToolCallInfo struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// ToolResultInfo holds tool result summary info for scorers.
type ToolResultInfo struct {
	Name   string         `json:"name"`
	Result map[string]any `json:"result,omitempty"`
}

// RunStreamCompletionScorers is a stub for ../../network/validation.RunStreamCompletionScorers.
// Stub: real signature is (context.Context, []MastraScorer, StreamCompletionContext, *CompletionScorerOptions)
// → CompletionRunResult. Stub uses ([]any, StreamCompletionContext, map[string]any) → (*IsTaskCompleteRunResult, error).
// Signature mismatch: different param types (context.Context, []MastraScorer, *CompletionScorerOptions)
// and different return (value vs pointer, no error in real). No cycle, but wiring requires type alignment.
func RunStreamCompletionScorers(
	scorers []any,
	context StreamCompletionContext,
	options map[string]any,
) (*IsTaskCompleteRunResult, error) {
	// Stub: always returns not complete.
	return &IsTaskCompleteRunResult{Complete: false}, nil
}

// FormatStreamCompletionFeedback is a stub for ../../network/validation.FormatStreamCompletionFeedback.
// Stub: real signature is (CompletionRunResult, bool) → string (value param).
// Stub uses (*IsTaskCompleteRunResult, bool) → string (pointer param). Also real function
// builds detailed markdown feedback with scorer results; stub returns simple strings.
// No cycle, but wiring requires type alignment (CompletionRunResult vs local IsTaskCompleteRunResult).
func FormatStreamCompletionFeedback(result *IsTaskCompleteRunResult, maxIterationReached bool) string {
	if result.Complete {
		return "Task is complete."
	}
	if maxIterationReached {
		return "Maximum iterations reached."
	}
	return "Task is not yet complete. Please continue."
}

// IsTaskCompleteConfig holds configuration for task completion scoring.
// Stub: real agent.IsTaskCompleteConfig = agent.CompletionConfig. Shape differences:
// Scorers: []any (stub) vs []MastraScorer (real); Parallel: bool (stub) vs *bool (real);
// OnComplete: func(IsTaskCompleteRunResult) error (stub) vs func(CompletionRunResult) error (real).
// Types are structurally different due to MastraScorer and CompletionRunResult definitions.
type IsTaskCompleteConfig struct {
	Scorers          []any                                              `json:"scorers,omitempty"`
	Strategy         string                                             `json:"strategy,omitempty"`
	Parallel         bool                                               `json:"parallel,omitempty"`
	Timeout          int                                                `json:"timeout,omitempty"`
	SuppressFeedback bool                                               `json:"suppressFeedback,omitempty"`
	OnComplete       func(result IsTaskCompleteRunResult) error         `json:"-"`
}

// ---------------------------------------------------------------------------
// Step represents a workflow step.
// ---------------------------------------------------------------------------

// Step is a stub for ../../../workflows.Step.
// Stub: can't import parent loop/workflows package (agenticexecution → workflows would create cycle).
// Real workflows.Step is also a simplified step type but lives in the parent package.
// This local Step with Execute func is the minimal contract needed for agentic workflow steps.
type Step struct {
	ID      string
	Execute func(args StepExecuteArgs) (any, error)
}

// StepExecuteArgs holds arguments passed to a step's execute function.
type StepExecuteArgs struct {
	InputData any
}

// ---------------------------------------------------------------------------
// CreateIsTaskCompleteStep
// ---------------------------------------------------------------------------

// CreateIsTaskCompleteStep creates a workflow step that evaluates whether the
// current task is complete by running configured scorers.
//
// The step:
//  1. Increments the iteration counter.
//  2. Checks if isTaskComplete scorers are configured AND the step is not
//     still continued (isContinued = false means LLM is done).
//  3. Extracts the original user task from the message list.
//  4. Builds a StreamCompletionContext with iteration count, tool calls,
//     tool results, messages, and agent metadata.
//  5. Runs the scorers (with strategy, parallel, timeout options).
//  6. Calls onComplete callback if configured.
//  7. Updates isContinued based on scorer results (complete = stop).
//  8. Adds feedback as an assistant message for the next iteration.
//  9. Emits an 'is-task-complete' chunk to the stream.
// 10. Sets isTaskCompleteCheckFailed if task is not complete.
func CreateIsTaskCompleteStep(params OuterLLMRun) *Step {
	currentIteration := 0

	return &Step{
		ID: "isTaskCompleteStep",
		Execute: func(args StepExecuteArgs) (any, error) {
			currentIteration++

			inputData, ok := args.InputData.(map[string]any)
			if !ok {
				return args.InputData, nil
			}

			// Check if isTaskComplete is configured.
			itcConfig, _ := params.IsTaskComplete.(*IsTaskCompleteConfig)
			if itcConfig == nil || len(itcConfig.Scorers) == 0 {
				return inputData, nil
			}

			// Check if step result is still continued (don't run scorers mid-loop).
			stepResult, _ := inputData["stepResult"].(map[string]any)
			if stepResult != nil {
				if isContinued, ok := stepResult["isContinued"].(bool); ok && isContinued {
					return inputData, nil
				}
			}

			// Extract original task from user messages.
			originalTask := "Unknown task"
			// Get user messages from messageList.get.input.db().
			if ml, ok := params.MessageList.(MessageListFull); ok {
				userMsgs := ml.GetInput().DB()
				if len(userMsgs) > 0 {
					if firstMsg := userMsgs[0]; firstMsg != nil {
						if content, ok := firstMsg["content"].(string); ok {
							originalTask = content
						} else if contentMap, ok := firstMsg["content"].(map[string]any); ok {
							if parts, ok := contentMap["parts"].([]any); ok && len(parts) > 0 {
								if part, ok := parts[0].(map[string]any); ok {
									if text, ok := part["text"].(string); ok {
										originalTask = text
									}
								}
							}
						}
					}
				}
			} else {
				// Fallback: try to extract from messages in inputData.
				if messages, ok := inputData["messages"].(map[string]any); ok {
					if userMsgs, ok := messages["user"].([]any); ok && len(userMsgs) > 0 {
						if firstMsg, ok := userMsgs[0].(map[string]any); ok {
							if content, ok := firstMsg["content"].(string); ok {
								originalTask = content
							} else if contentMap, ok := firstMsg["content"].(map[string]any); ok {
								if parts, ok := contentMap["parts"].([]any); ok && len(parts) > 0 {
									if part, ok := parts[0].(map[string]any); ok {
										if text, ok := part["text"].(string); ok {
											originalTask = text
										}
									}
								}
							}
						}
					}
				}
			}

			// Extract tool calls and tool results from output.
			var toolCalls []ToolCallInfo
			var toolResults []ToolResultInfo
			currentText := ""

			if output, ok := inputData["output"].(map[string]any); ok {
				if text, ok := output["text"].(string); ok {
					currentText = text
				}
				if tcs, ok := output["toolCalls"].([]any); ok {
					for _, tc := range tcs {
						if tcMap, ok := tc.(map[string]any); ok {
							name, _ := tcMap["toolName"].(string)
							args, _ := tcMap["args"].(map[string]any)
							if args == nil {
								args = make(map[string]any)
							}
							toolCalls = append(toolCalls, ToolCallInfo{
								Name: name,
								Args: args,
							})
						}
					}
				}
				if trs, ok := output["toolResults"].([]any); ok {
					for _, tr := range trs {
						if trMap, ok := tr.(map[string]any); ok {
							name, _ := trMap["toolName"].(string)
							result, _ := trMap["result"].(map[string]any)
							toolResults = append(toolResults, ToolResultInfo{
								Name:   name,
								Result: result,
							})
						}
					}
				}
			}

			// Build StreamCompletionContext.
			var threadID, resourceID string
			if params.Internal != nil {
				if internal, ok := params.Internal.(map[string]any); ok {
					threadID, _ = internal["threadId"].(string)
					resourceID, _ = internal["resourceId"].(string)
				}
			}

			var customContext map[string]any
			if params.RequestContext != nil {
				if rc, ok := params.RequestContext.(map[string]any); ok {
					customContext = rc
				}
			}

			// Get all messages from messageList for the context.
			var allMessages []any
			if ml, ok := params.MessageList.(MessageListFull); ok {
				dbMsgs := ml.GetAll().DB()
				allMessages = make([]any, len(dbMsgs))
				for i, m := range dbMsgs {
					allMessages[i] = m
				}
			}

			context := StreamCompletionContext{
				Iteration:     currentIteration,
				MaxIterations: params.MaxSteps,
				OriginalTask:  originalTask,
				CurrentText:   currentText,
				ToolCalls:     toolCalls,
				ToolResults:   toolResults,
				AgentID:       params.AgentID,
				AgentName:     params.AgentName,
				RunID:         params.RunID,
				ThreadID:      threadID,
				ResourceID:    resourceID,
				CustomContext: customContext,
				Messages:      allMessages,
			}

			// Run scorers.
			result, err := RunStreamCompletionScorers(
				itcConfig.Scorers,
				context,
				map[string]any{
					"strategy": itcConfig.Strategy,
					"parallel": itcConfig.Parallel,
					"timeout":  itcConfig.Timeout,
				},
			)
			if err != nil {
				return inputData, err
			}

			// Call onComplete callback.
			if itcConfig.OnComplete != nil {
				if err := itcConfig.OnComplete(*result); err != nil {
					return inputData, err
				}
			}

			// Update isContinued based on scorer results.
			if stepResult != nil {
				if result.Complete {
					stepResult["isContinued"] = false
				} else {
					stepResult["isContinued"] = true
				}
			}

			// Generate feedback.
			maxIterationReached := false
			if params.MaxSteps > 0 {
				maxIterationReached = currentIteration >= params.MaxSteps
			}
			feedback := FormatStreamCompletionFeedback(result, maxIterationReached)

			// Add feedback as assistant message for the LLM to see in next iteration.
			var mastraGenerateID func() string
			if params.Mastra != nil {
				if mastra, ok := params.Mastra.(map[string]any); ok {
					if genID, ok := mastra["generateId"].(func() string); ok {
						mastraGenerateID = genID
					}
				}
			}
			feedbackMsgID := ""
			if mastraGenerateID != nil {
				feedbackMsgID = mastraGenerateID()
			}

			feedbackMsg := map[string]any{
				"id":        feedbackMsgID,
				"createdAt": time.Now(),
				"type":      "text",
				"role":      "assistant",
				"content": map[string]any{
					"parts": []map[string]any{
						{
							"type": "text",
							"text": feedback,
						},
					},
					"metadata": map[string]any{
						"mode": "stream",
						"completionResult": map[string]any{
							"passed":           result.Complete,
							"suppressFeedback": itcConfig.SuppressFeedback,
						},
					},
					"format": 2,
				},
			}
			if ml, ok := params.MessageList.(MessageListFull); ok {
				ml.Add(feedbackMsg, "response")
			}

			// Emit is-task-complete chunk.
			isTaskCompleteChunk := map[string]any{
				"type":  "is-task-complete",
				"runId": params.RunID,
				"from":  ChunkFromAGENT,
				"payload": map[string]any{
					"iteration":          currentIteration,
					"passed":             result.Complete,
					"results":            result.Scorers,
					"duration":           result.TotalDuration,
					"timedOut":           result.TimedOut,
					"reason":             result.CompletionReason,
					"maxIterationReached": maxIterationReached,
					"suppressFeedback":   itcConfig.SuppressFeedback,
				},
			}
			SafeEnqueue(params.Controller, isTaskCompleteChunk)

			// Set isTaskCompleteCheckFailed if task is not complete.
			if !result.Complete {
				inputData["isTaskCompleteCheckFailed"] = true
			}

			return inputData, nil
		},
	}
}
