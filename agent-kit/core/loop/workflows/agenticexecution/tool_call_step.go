// Ported from: packages/core/src/loop/workflows/agentic-execution/tool-call-step.ts
package agenticexecution

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported (tool-call-specific)
// ---------------------------------------------------------------------------

// ToolNotFoundError is a stub — re-declared here to avoid circular import.
// Stub: real ToolNotFoundError is in loop/workflows (parent package). Importing would create
// cycle: agenticexecution → workflows → agenticexecution. Identical shape (Message string field,
// Error() string method, NewToolNotFoundError constructor). Kept local to break the cycle.
type ToolNotFoundError struct {
	Message string
}

func (e *ToolNotFoundError) Error() string {
	return e.Message
}

// NewToolNotFoundError creates a new ToolNotFoundError.
func NewToolNotFoundError(message string) *ToolNotFoundError {
	return &ToolNotFoundError{Message: message}
}

// MastraToolInvocationOptions is a stub for ../../../tools/types.MastraToolInvocationOptions.
// Stub: real MastraToolInvocationOptions embeds *ObservabilityContext, has typed fields
// (OutputWriter as tools.OutputWriter interface, Workspace as workspace.Workspace,
// RequestContext as *requestcontext.RequestContext, MCP *MCPToolExecutionContext).
// Stub uses all-any fields (AbortSignal any, Messages []any, OutputWriter any, etc.).
// Shape mismatch: embedded struct, typed fields, and additional MCP field in real.
type MastraToolInvocationOptions struct {
	AbortSignal    any            `json:"-"`
	ToolCallID     string         `json:"toolCallId"`
	Messages       []any          `json:"messages,omitempty"`
	OutputWriter   any            `json:"-"`
	TracingContext any            `json:"-"`
	Workspace      any            `json:"-"`
	RequestContext any            `json:"-"`
	Suspend        func(payload any, opts *SuspendOptions) (any, error) `json:"-"`
	ResumeData     any            `json:"resumeData,omitempty"`
}

// SuspendOptions is a stub for ../../../workflows.SuspendOptions.
// Stub: real workflows.SuspendOptions has fields {ResumeLabel []string, Extra map[string]any}.
// Stub has {RequireToolApproval bool, ResumeSchema string, ResumeLabel string, RunID string}.
// Completely different field sets — real is generic workflow suspend; stub is tool-specific suspend.
// Shape mismatch: different purpose and field names/types.
type SuspendOptions struct {
	RequireToolApproval bool   `json:"requireToolApproval,omitempty"`
	ResumeSchema        string `json:"resumeSchema,omitempty"`
	ResumeLabel         string `json:"resumeLabel,omitempty"`
	RunID               string `json:"runId,omitempty"`
}

// ToolCallInput represents the input for a single tool call step.
type ToolCallInput struct {
	ToolCallID       string         `json:"toolCallId"`
	ToolName         string         `json:"toolName"`
	Args             map[string]any `json:"args"`
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`
	ProviderExecuted *bool          `json:"providerExecuted,omitempty"`
	Output           any            `json:"output,omitempty"`
}

// AddToolMetadataOptions configures how tool metadata is added to messages.
type AddToolMetadataOptions struct {
	ToolCallID         string `json:"toolCallId"`
	ToolName           string `json:"toolName"`
	Args               any    `json:"args"`
	ResumeSchema       string `json:"resumeSchema"`
	SuspendedToolRunID string `json:"suspendedToolRunId,omitempty"`
	Type               string `json:"type"` // "approval" or "suspension"
	SuspendPayload     any    `json:"suspendPayload,omitempty"`
}

// approvalResumeSchema is the JSON schema for approval resume data.
// Mirrors the TS zod schema: z.object({ approved: z.boolean() }).
const approvalResumeSchema = `{"type":"object","properties":{"approved":{"type":"boolean","description":"Controls if the tool call is approved or not, should be true when approved and false when declined"}},"required":["approved"]}`

// ---------------------------------------------------------------------------
// CreateToolCallStep
// ---------------------------------------------------------------------------

// CreateToolCallStep creates the workflow step that executes a single tool
// call. This step is invoked via foreach over all tool calls in an iteration.
//
// The step:
//  1. Resolves the tool from stepTools (set by llmExecutionStep via
//     prepareStep/processInputStep) or falls back to the original tools.
//  2. Handles provider-executed tools (skip execution, return output).
//  3. Returns ToolNotFoundError for unknown tools (with available names).
//  4. Calls tool.onInputAvailable if available.
//  5. Handles tool approval flow:
//     - If requireApproval: emit approval chunk, persist metadata, suspend.
//     - On resume with approved=true: continue execution.
//     - On resume with approved=false: return "not approved" result.
//  6. Handles tool suspension flow:
//     - Tool calls suspend() during execution.
//     - Emit suspension chunk, persist metadata, suspend workflow.
//  7. Handles sub-agent tools (agent-* prefix):
//     - Pass threadId/resourceId from _internal.
//     - Pass full conversation context.
//     - Handle suspended tool run ID propagation.
//  8. Handles workflow tools (workflow-* prefix) similarly.
//  9. Calls tool.execute with parsed args and tool options.
// 10. Calls tool.onOutput if available.
// 11. Returns {result, ...inputData} on success or {error, ...inputData} on failure.
//
// Message persistence:
//   - Before any suspension, flushMessagesBeforeSuspension() ensures all
//     pending messages are persisted.
//   - Tool metadata (pendingToolApprovals, suspendedTools) is added to the
//     last assistant message for resumption state tracking.
//   - On resume, tool metadata is removed to clean up state.
func CreateToolCallStep(params OuterLLMRun) *Step {
	tools := params.Tools

	return &Step{
		ID: "toolCallStep",
		Execute: func(args StepExecuteArgs) (any, error) {
			inputData, ok := args.InputData.(ToolCallInput)
			if !ok {
				inputDataMap, ok := args.InputData.(map[string]any)
				if !ok {
					return args.InputData, nil
				}
				// Try to unmarshal from map.
				b, _ := json.Marshal(inputDataMap)
				if err := json.Unmarshal(b, &inputData); err != nil {
					return args.InputData, nil
				}
			}

			// Use stepTools from _internal if available (set by llmExecutionStep
			// via prepareStep/processInputStep). Falls back to original tools.
			stepTools := tools
			if params.Internal != nil {
				if internal, ok := params.Internal.(map[string]any); ok {
					if st, ok := internal["stepTools"].(ToolSet); ok && st != nil {
						stepTools = st
					}
				}
			}

			// Find the tool by name or by id field.
			tool, found := stepTools[inputData.ToolName]
			if !found {
				for _, t := range stepTools {
					tm, ok := t.(map[string]any)
					if ok {
						if id, ok := tm["id"].(string); ok && id == inputData.ToolName {
							tool = t
							found = true
							break
						}
					}
				}
			}

			// Handle provider-executed tools — skip execution, return output.
			if inputData.ProviderExecuted != nil && *inputData.ProviderExecuted {
				result := inputData.Output
				if result == nil {
					result = map[string]any{
						"providerExecuted": true,
						"toolName":         inputData.ToolName,
					}
				}
				return ToolCallOutput{
					ToolCallID:       inputData.ToolCallID,
					ToolName:         inputData.ToolName,
					Args:             inputData.Args,
					ProviderMetadata: inputData.ProviderMetadata,
					ProviderExecuted: inputData.ProviderExecuted,
					Result:           result,
				}, nil
			}

			// Tool not found — return error with available tool names.
			if !found {
				availableNames := make([]string, 0, len(stepTools))
				for name := range stepTools {
					availableNames = append(availableNames, name)
				}
				var availableStr string
				if len(availableNames) > 0 {
					availableStr = fmt.Sprintf(" Available tools: %s", joinStrings(availableNames, ", "))
				}
				return ToolCallOutput{
					ToolCallID:       inputData.ToolCallID,
					ToolName:         inputData.ToolName,
					Args:             inputData.Args,
					ProviderMetadata: inputData.ProviderMetadata,
					Error: NewToolNotFoundError(
						fmt.Sprintf(`Tool "%s" not found.%s. Call tools by their exact name only — never add prefixes, namespaces, or colons.`,
							inputData.ToolName, availableStr),
					),
				}, nil
			}

			// Call onInputAvailable hook if tool supports it.
			toolMap, isMap := tool.(map[string]any)
			if isMap {
				if onInputAvailable, ok := toolMap["onInputAvailable"].(func(args map[string]any) error); ok {
					if err := onInputAvailable(map[string]any{
						"toolCallId": inputData.ToolCallID,
						"input":      inputData.Args,
					}); err != nil {
						if logger := toLogger(params.Logger); logger != nil {
							logger.Error("Error calling onInputAvailable", err)
						}
					}
				}
			}

			// Check for execute function.
			if isMap {
				executeFn, hasExecute := toolMap["execute"]
				if !hasExecute || executeFn == nil {
					return ToolCallOutput{
						ToolCallID:       inputData.ToolCallID,
						ToolName:         inputData.ToolName,
						Args:             inputData.Args,
						ProviderMetadata: inputData.ProviderMetadata,
					}, nil
				}
			}

			// --- Approval / Suspension / Execution logic ---

			// Extract resumeData from args if present.
			var resumeData map[string]any
			toolArgs := inputData.Args

			if toolArgs != nil {
				if rd, ok := toolArgs["resumeData"]; ok {
					if rdMap, ok := rd.(map[string]any); ok {
						resumeData = rdMap
					}
					// Remove resumeData from args to pass clean args to tool.
					cleanArgs := make(map[string]any, len(toolArgs))
					for k, v := range toolArgs {
						if k != "resumeData" {
							cleanArgs[k] = v
						}
					}
					toolArgs = cleanArgs
				}
			}

			// In TS: resumeData = resumeDataFromArgs ?? workflowResumeData
			// workflowResumeData comes from the workflow suspend/resume system (not yet ported).
			// When the workflow engine is ported, merge workflowResumeData here as a fallback.

			isResumeToolCall := resumeData != nil

			// Check if approval is required.
			// requireApproval can be: bool (from createTool), undefined (no approval), or from requestContext.
			var toolRequiresApproval bool
			if params.RequireToolApproval {
				toolRequiresApproval = true
			}
			if isMap {
				if ra, ok := toolMap["requireApproval"].(bool); ok && ra {
					toolRequiresApproval = true
				}
				// Check needsApprovalFn.
				if needsApprovalFn, ok := toolMap["needsApprovalFn"].(func(args map[string]any) (bool, error)); ok {
					result, err := needsApprovalFn(toolArgs)
					if err != nil {
						if logger := toLogger(params.Logger); logger != nil {
							logger.Error(fmt.Sprintf("Error evaluating needsApprovalFn for tool %s", inputData.ToolName), err)
						}
						toolRequiresApproval = true
					} else {
						toolRequiresApproval = result
					}
				}
			}

			// --- Tool approval flow ---
			if toolRequiresApproval {
				if resumeData == nil {
					// First call — emit approval chunk and suspend.
					SafeEnqueue(params.Controller, map[string]any{
						"type":  "tool-call-approval",
						"runId": params.RunID,
						"from":  ChunkFromAGENT,
						"payload": map[string]any{
							"toolCallId":   inputData.ToolCallID,
							"toolName":     inputData.ToolName,
							"args":         inputData.Args,
							"resumeSchema": approvalResumeSchema,
						},
					})

					// Add approval metadata to the last assistant message for resumption tracking.
					addToolMetadata(params.MessageList, AddToolMetadataOptions{
						ToolCallID:   inputData.ToolCallID,
						ToolName:     inputData.ToolName,
						Args:         inputData.Args,
						Type:         "approval",
						ResumeSchema: approvalResumeSchema,
					}, params.RunID)

					// Flush messages before suspension to ensure persistence.
					flushMessagesBeforeSuspension(params.Internal, params.Logger)

					// Suspend workflow — in a full implementation this would call suspend().
					// For now, return a suspended-like result signaling no tool result.
					return ToolCallOutput{
						ToolCallID:       inputData.ToolCallID,
						ToolName:         inputData.ToolName,
						Args:             inputData.Args,
						ProviderMetadata: inputData.ProviderMetadata,
					}, nil
				}

				// Resuming — remove approval metadata since we're either approved or declined.
				removeToolMetadata(params.MessageList, params.Internal, params.Logger, inputData.ToolName, "approval")
				if approved, ok := resumeData["approved"].(bool); ok && !approved {
					return ToolCallOutput{
						ToolCallID:       inputData.ToolCallID,
						ToolName:         inputData.ToolName,
						Args:             inputData.Args,
						ProviderMetadata: inputData.ProviderMetadata,
						Result:           "Tool call was not approved by the user",
					}, nil
				}
			} else if isResumeToolCall {
				// Not requiring approval but resuming — remove suspension metadata.
				removeToolMetadata(params.MessageList, params.Internal, params.Logger, inputData.ToolName, "suspension")
			}

			// Determine if this is an agent or workflow tool.
			isAgentTool := strings.HasPrefix(inputData.ToolName, "agent-")
			isWorkflowTool := strings.HasPrefix(inputData.ToolName, "workflow-")

			// Determine resumeData to pass to tool options.
			var resumeDataToPass any
			if isAgentTool || !toolRequiresApproval || (resumeData != nil && len(resumeData) != 1) {
				resumeDataToPass = resumeData
			} else if resumeData != nil {
				if _, hasApproved := resumeData["approved"]; hasApproved && len(resumeData) == 1 {
					// Only 'approved' key — don't pass to tool.
					resumeDataToPass = nil
				} else {
					resumeDataToPass = resumeData
				}
			}

			// Build tool options.
			toolOptions := MastraToolInvocationOptions{
				ToolCallID:     inputData.ToolCallID,
				OutputWriter:   params.OutputWriter,
				RequestContext: params.RequestContext,
				ResumeData:     resumeDataToPass,
			}

			// For agent tools, pass full conversation messages.
			if isAgentTool {
				if ml, ok := params.MessageList.(MessageListFull); ok {
					toolOptions.Messages = ml.GetAll().AIV5Model()
				}
			} else {
				if ml, ok := params.MessageList.(MessageListFull); ok {
					toolOptions.Messages = ml.GetInput().AIV5Model()
				}
			}

			// Set up suspend function on tool options for tool-initiated suspension.
			toolOptions.Suspend = func(suspendPayload any, opts *SuspendOptions) (any, error) {
				if opts != nil && opts.RequireToolApproval {
					// Tool requesting approval during execution.
					SafeEnqueue(params.Controller, map[string]any{
						"type":  "tool-call-approval",
						"runId": params.RunID,
						"from":  ChunkFromAGENT,
						"payload": map[string]any{
							"toolCallId":   inputData.ToolCallID,
							"toolName":     inputData.ToolName,
							"args":         inputData.Args,
							"resumeSchema": approvalResumeSchema,
						},
					})

					// Add approval metadata to message before persisting.
					addToolMetadata(params.MessageList, AddToolMetadataOptions{
						ToolCallID:         inputData.ToolCallID,
						ToolName:           inputData.ToolName,
						Args:               inputData.Args,
						Type:               "approval",
						SuspendedToolRunID: opts.RunID,
						ResumeSchema:       approvalResumeSchema,
					}, params.RunID)

					// Flush messages before suspension.
					flushMessagesBeforeSuspension(params.Internal, params.Logger)

					// In a full implementation, this would call suspend() to pause the workflow.
				} else {
					// Tool requesting suspension during execution.
					var resumeSchema string
					var suspendedToolRunID string
					if opts != nil {
						resumeSchema = opts.ResumeSchema
						suspendedToolRunID = opts.RunID
					}
					SafeEnqueue(params.Controller, map[string]any{
						"type":  "tool-call-suspended",
						"runId": params.RunID,
						"from":  ChunkFromAGENT,
						"payload": map[string]any{
							"toolCallId":     inputData.ToolCallID,
							"toolName":       inputData.ToolName,
							"suspendPayload": suspendPayload,
							"args":           inputData.Args,
							"resumeSchema":   resumeSchema,
						},
					})

					// Add suspension metadata to message before persisting.
					addToolMetadata(params.MessageList, AddToolMetadataOptions{
						ToolCallID:         inputData.ToolCallID,
						ToolName:           inputData.ToolName,
						Args:               toolArgs,
						SuspendPayload:     suspendPayload,
						SuspendedToolRunID: suspendedToolRunID,
						Type:               "suspension",
						ResumeSchema:       resumeSchema,
					}, params.RunID)

					// Flush messages before suspension.
					flushMessagesBeforeSuspension(params.Internal, params.Logger)

					// In a full implementation, this would call suspend() to pause the workflow.
				}
				return nil, nil
			}

			// Pass workspace from _internal.
			if params.Internal != nil {
				if internal, ok := params.Internal.(map[string]any); ok {
					toolOptions.Workspace = internal["stepWorkspace"]
				}
			}

			// For sub-agent or workflow tools resuming, find the suspended tool's runId.
			if resumeDataToPass != nil && (isAgentTool || isWorkflowTool) && !isResumeToolCall {
				// Search assistant messages for suspendedTools/pendingToolApprovals
				// metadata to find the suspended tool's runId.
				if ml, ok := params.MessageList.(MessageListFull); ok {
					allMsgs := ml.GetAll().DB()
					// Search in reverse order (most recent first).
					for i := len(allMsgs) - 1; i >= 0; i-- {
						msg := allMsgs[i]
						if msg == nil {
							continue
						}
						role, _ := msg["role"].(string)
						if role != "assistant" {
							continue
						}
						content, _ := msg["content"].(map[string]any)
						if content == nil {
							continue
						}
						metadata, _ := content["metadata"].(map[string]any)
						if metadata != nil {
							for _, key := range []string{"suspendedTools", "pendingToolApprovals"} {
								if pendingTools, ok := metadata[key].(map[string]any); ok {
									if toolData, ok := pendingTools[inputData.ToolName].(map[string]any); ok {
										if runID, ok := toolData["runId"].(string); ok && runID != "" {
											toolArgs["suspendedToolRunId"] = runID
											goto foundRunID
										}
									}
								}
							}
						}
						// Also check data-tool-call-suspended parts.
						if parts, ok := content["parts"].([]any); ok {
							for _, p := range parts {
								part, ok := p.(map[string]any)
								if !ok {
									continue
								}
								partType, _ := part["type"].(string)
								if partType == "data-tool-call-suspended" || partType == "data-tool-call-approval" {
									if resumed, ok := part["data"].(map[string]any); ok {
										if isResumed, _ := resumed["resumed"].(bool); !isResumed {
											if tn, ok := resumed["toolName"].(string); ok && tn == inputData.ToolName {
												if runID, ok := resumed["runId"].(string); ok && runID != "" {
													toolArgs["suspendedToolRunId"] = runID
													goto foundRunID
												}
											}
										}
									}
								}
							}
						}
					}
				foundRunID:
				}
			}

			// Validate args.
			if toolArgs == nil {
				return ToolCallOutput{
					ToolCallID:       inputData.ToolCallID,
					ToolName:         inputData.ToolName,
					Args:             inputData.Args,
					ProviderMetadata: inputData.ProviderMetadata,
					Error: fmt.Errorf(
						`Tool "%s" received invalid arguments — the provided JSON could not be parsed. Please provide valid JSON arguments.`,
						inputData.ToolName),
				}, nil
			}

			// For agent tools, inject threadId/resourceId.
			if isAgentTool {
				if _, hasPrompt := toolArgs["prompt"]; hasPrompt {
					if params.Internal != nil {
						if internal, ok := params.Internal.(map[string]any); ok {
							if _, hasThread := toolArgs["threadId"]; !hasThread {
								if tid, ok := internal["threadId"].(string); ok {
									toolArgs["threadId"] = tid
								}
							}
							if _, hasResource := toolArgs["resourceId"]; !hasResource {
								if rid, ok := internal["resourceId"].(string); ok {
									toolArgs["resourceId"] = rid
								}
							}
						}
					}
				}
			}

			// Execute the tool.
			if isMap {
				executeFn, ok := toolMap["execute"].(func(args map[string]any, opts MastraToolInvocationOptions) (any, error))
				if ok {
					result, err := executeFn(toolArgs, toolOptions)
					if err != nil {
						return ToolCallOutput{
							ToolCallID:       inputData.ToolCallID,
							ToolName:         inputData.ToolName,
							Args:             inputData.Args,
							ProviderMetadata: inputData.ProviderMetadata,
							Error:            err,
						}, nil
					}

					// Call onOutput hook after successful execution.
					if onOutput, ok := toolMap["onOutput"].(func(args map[string]any) error); ok {
						if err := onOutput(map[string]any{
							"toolCallId": inputData.ToolCallID,
							"toolName":   inputData.ToolName,
							"output":     result,
						}); err != nil {
							if logger := toLogger(params.Logger); logger != nil {
								logger.Error("Error calling onOutput", err)
							}
						}
					}

					return ToolCallOutput{
						ToolCallID:       inputData.ToolCallID,
						ToolName:         inputData.ToolName,
						Args:             inputData.Args,
						ProviderMetadata: inputData.ProviderMetadata,
						Result:           result,
					}, nil
				}
			}

			return ToolCallOutput{
				ToolCallID:       inputData.ToolCallID,
				ToolName:         inputData.ToolName,
				Args:             inputData.Args,
				ProviderMetadata: inputData.ProviderMetadata,
			}, nil
		},
	}
}

// addToolMetadata adds tool metadata (pendingToolApprovals or suspendedTools)
// to the last assistant message in the response for resumption tracking.
func addToolMetadata(messageList any, opts AddToolMetadataOptions, runID string) {
	ml, ok := messageList.(MessageListFull)
	if !ok {
		return
	}

	metadataKey := "suspendedTools"
	if opts.Type == "approval" {
		metadataKey = "pendingToolApprovals"
	}

	// Find the last assistant message in the response.
	responseMsgs := ml.GetResponse().DB()
	if len(responseMsgs) == 0 {
		return
	}

	// Search from the end for the last assistant message.
	for i := len(responseMsgs) - 1; i >= 0; i-- {
		msg := responseMsgs[i]
		if msg == nil {
			continue
		}
		role, _ := msg["role"].(string)
		if role != "assistant" {
			continue
		}

		content, _ := msg["content"].(map[string]any)
		if content == nil {
			continue
		}

		// Get or create metadata on the content.
		metadata, _ := content["metadata"].(map[string]any)
		if metadata == nil {
			metadata = make(map[string]any)
			content["metadata"] = metadata
		}

		// Get or create the pending tools map.
		pendingTools, _ := metadata[metadataKey].(map[string]any)
		if pendingTools == nil {
			pendingTools = make(map[string]any)
		}

		// Key by toolName (not toolCallId) to track one suspension state per unique tool.
		toolEntry := map[string]any{
			"toolCallId": opts.ToolCallID,
			"toolName":   opts.ToolName,
			"args":       opts.Args,
			"type":       opts.Type,
			"runId":      runID,
		}
		if opts.SuspendedToolRunID != "" {
			toolEntry["runId"] = opts.SuspendedToolRunID
		}
		if opts.Type == "suspension" {
			toolEntry["suspendPayload"] = opts.SuspendPayload
		}
		if opts.ResumeSchema != "" {
			toolEntry["resumeSchema"] = opts.ResumeSchema
		}
		pendingTools[opts.ToolName] = toolEntry
		metadata[metadataKey] = pendingTools
		break
	}
}

// removeToolMetadata removes tool metadata from the message list after resumption.
func removeToolMetadata(messageList any, internal any, logger any, toolName string, metadataType string) {
	ml, ok := messageList.(MessageListFull)
	if !ok {
		return
	}

	metadataKey := "suspendedTools"
	if metadataType == "approval" {
		metadataKey = "pendingToolApprovals"
	}

	// Search all messages in reverse for the one with this tool's metadata.
	allMsgs := ml.GetAll().DB()
	for i := len(allMsgs) - 1; i >= 0; i-- {
		msg := allMsgs[i]
		if msg == nil {
			continue
		}
		role, _ := msg["role"].(string)
		if role != "assistant" {
			continue
		}
		content, _ := msg["content"].(map[string]any)
		if content == nil {
			continue
		}
		metadata, _ := content["metadata"].(map[string]any)
		if metadata != nil {
			pendingTools, _ := metadata[metadataKey].(map[string]any)
			if pendingTools != nil {
				if _, found := pendingTools[toolName]; found {
					delete(pendingTools, toolName)
					if len(pendingTools) == 0 {
						delete(metadata, metadataKey)
					}
					// Flush to persist the metadata removal.
					flushMessagesBeforeSuspension(internal, logger)
					return
				}
			}
		}
		// Also check parts.
		if parts, ok := content["parts"].([]any); ok {
			for _, p := range parts {
				part, ok := p.(map[string]any)
				if !ok {
					continue
				}
				partType, _ := part["type"].(string)
				if partType == "data-tool-call-suspended" || partType == "data-tool-call-approval" {
					if data, ok := part["data"].(map[string]any); ok {
						if tn, ok := data["toolName"].(string); ok && tn == toolName {
							// Mark as resumed.
							data["resumed"] = true
							flushMessagesBeforeSuspension(internal, logger)
							return
						}
					}
				}
			}
		}
	}
}

// flushMessagesBeforeSuspension ensures all pending messages are persisted
// before a workflow suspension. Calls saveQueueManager.flushMessages if available.
func flushMessagesBeforeSuspension(internal any, logger any) {
	if internal == nil {
		return
	}
	internalMap, ok := internal.(map[string]any)
	if !ok {
		return
	}

	// Check for saveQueueManager and threadId.
	saveQueueManager := internalMap["saveQueueManager"]
	threadID, _ := internalMap["threadId"].(string)
	if saveQueueManager == nil || threadID == "" {
		return
	}

	// Call flushMessages if it's a function.
	if flushFn, ok := saveQueueManager.(interface {
		FlushMessages(messageList any, threadID string, memoryConfig any) error
	}); ok {
		memoryConfig := internalMap["memoryConfig"]
		if err := flushFn.FlushMessages(internalMap["messageList"], threadID, memoryConfig); err != nil {
			if l := toLogger(logger); l != nil {
				l.Error("Error flushing messages before suspension", err)
			}
		}
	}
}

// joinStrings joins string slices with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for _, s := range strs[1:] {
		result += sep + s
	}
	return result
}
