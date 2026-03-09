// Ported from: packages/openai/src/responses/convert-to-openai-responses-input.ts
package openai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// isFileID checks if a string is a file ID based on the given prefixes.
// Returns false if prefixes is nil (disables file ID detection).
func isFileID(data string, prefixes []string) bool {
	if prefixes == nil {
		return false
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(data, prefix) {
			return true
		}
	}
	return false
}

// ConvertToOpenAIResponsesInputOptions are the options for ConvertToOpenAIResponsesInput.
type ConvertToOpenAIResponsesInputOptions struct {
	Prompt                 languagemodel.Prompt
	ToolNameMapping        providerutils.ToolNameMapping
	SystemMessageMode      string // "system" | "developer" | "remove"
	ProviderOptionsName    string
	FileIDPrefixes         []string
	Store                  bool
	HasConversation        bool
	HasLocalShellTool      bool
	HasShellTool           bool
	HasApplyPatchTool      bool
	CustomProviderToolNames map[string]struct{}
}

// ConvertToOpenAIResponsesInputResult is the result of ConvertToOpenAIResponsesInput.
type ConvertToOpenAIResponsesInputResult struct {
	Input    OpenAIResponsesInput
	Warnings []shared.Warning
}

// ConvertToOpenAIResponsesInput converts language model prompts into
// the format expected by the OpenAI Responses API.
func ConvertToOpenAIResponsesInput(opts ConvertToOpenAIResponsesInputOptions) (*ConvertToOpenAIResponsesInputResult, error) {
	var input OpenAIResponsesInput
	var warnings []shared.Warning
	processedApprovalIDs := make(map[string]struct{})

	for _, msg := range opts.Prompt {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			switch opts.SystemMessageMode {
			case "system":
				input = append(input, OpenAIResponsesSystemMessage{
					Role:    "system",
					Content: m.Content,
				})
			case "developer":
				input = append(input, OpenAIResponsesSystemMessage{
					Role:    "developer",
					Content: m.Content,
				})
			case "remove":
				warnings = append(warnings, shared.OtherWarning{
					Message: "system messages are removed for this model",
				})
			default:
				return nil, fmt.Errorf("unsupported system message mode: %s", opts.SystemMessageMode)
			}

		case languagemodel.UserMessage:
			var contentParts []any
			for i, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					contentParts = append(contentParts, map[string]any{
						"type": "input_text",
						"text": p.Text,
					})
				case languagemodel.FilePart:
					converted, err := convertUserFilePart(p, i, opts.ProviderOptionsName, opts.FileIDPrefixes)
					if err != nil {
						return nil, err
					}
					contentParts = append(contentParts, converted)
				}
			}
			input = append(input, OpenAIResponsesUserMessage{
				Role:    "user",
				Content: contentParts,
			})

		case languagemodel.AssistantMessage:
			reasoningMessages := make(map[string]*OpenAIResponsesReasoning)

			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					id := getStringProviderOption(p.ProviderOptions, opts.ProviderOptionsName, "itemId")
					phase := getStringProviderOption(p.ProviderOptions, opts.ProviderOptionsName, "phase")

					// when using conversation, skip items that already exist
					if opts.HasConversation && id != "" {
						break
					}

					// item references reduce the payload size
					if opts.Store && id != "" {
						input = append(input, OpenAIResponsesItemReference{
							Type: "item_reference",
							ID:   id,
						})
						break
					}

					msg := OpenAIResponsesAssistantMessage{
						Role: "assistant",
						Content: []any{
							map[string]any{
								"type": "output_text",
								"text": p.Text,
							},
						},
						ID: id,
					}
					if phase != "" {
						msg.Phase = phase
					}
					input = append(input, msg)

				case languagemodel.ToolCallPart:
					id := getToolCallItemID(p, opts.ProviderOptionsName)

					if opts.HasConversation && id != "" {
						break
					}

					if p.ProviderExecuted != nil && *p.ProviderExecuted {
						if opts.Store && id != "" {
							input = append(input, OpenAIResponsesItemReference{
								Type: "item_reference",
								ID:   id,
							})
						}
						break
					}

					if opts.Store && id != "" {
						input = append(input, OpenAIResponsesItemReference{
							Type: "item_reference",
							ID:   id,
						})
						break
					}

					resolvedToolName := opts.ToolNameMapping.ToProviderToolName(p.ToolName)

					if opts.HasLocalShellTool && resolvedToolName == "local_shell" {
						localShellInput := convertLocalShellInput(p.Input)
						input = append(input, OpenAIResponsesLocalShellCall{
							Type:   "local_shell_call",
							CallID: p.ToolCallID,
							ID:     id,
							Action: OpenAIResponsesLocalShellCallAction{
						Type:    localShellInput.Action.Type,
						Command: localShellInput.Action.Command,
					},
						})
						break
					}

					if opts.HasShellTool && resolvedToolName == "shell" {
						shellInput := convertShellInput(p.Input)
						input = append(input, OpenAIResponsesShellCall{
							Type:   "shell_call",
							CallID: p.ToolCallID,
							ID:     id,
							Status: "completed",
							Action: OpenAIResponsesShellCallAction{
								Commands: shellInput.Action.Commands,
							},
						})
						break
					}

					if opts.HasApplyPatchTool && resolvedToolName == "apply_patch" {
						applyPatchInput := convertApplyPatchInput(p.Input)
						input = append(input, OpenAIResponsesApplyPatchCall{
							Type:      "apply_patch_call",
							CallID:    applyPatchInput.CallID,
							ID:        id,
							Status:    "completed",
							Operation: applyPatchInput.Operation,
						})
						break
					}

					if _, ok := opts.CustomProviderToolNames[resolvedToolName]; ok {
						inputStr := ""
						switch v := p.Input.(type) {
						case string:
							inputStr = v
						default:
							data, _ := json.Marshal(v)
							inputStr = string(data)
						}
						input = append(input, OpenAIResponsesCustomToolCall{
							Type:   "custom_tool_call",
							CallID: p.ToolCallID,
							Name:   resolvedToolName,
							Input:  inputStr,
							ID:     id,
						})
						break
					}

					argsJSON, _ := json.Marshal(p.Input)
					input = append(input, OpenAIResponsesFunctionCall{
						Type:      "function_call",
						CallID:    p.ToolCallID,
						Name:      resolvedToolName,
						Arguments: string(argsJSON),
						ID:        id,
					})

				case languagemodel.ToolResultPart:
					// assistant tool result parts are from provider-executed tools
					output := p.Output

					// Skip execution-denied results
					if isExecutionDenied(output) {
						break
					}

					if opts.HasConversation {
						break
					}

					resolvedResultToolName := opts.ToolNameMapping.ToProviderToolName(p.ToolName)

					// Shell tool results
					if opts.HasShellTool && resolvedResultToolName == "shell" {
						if jsonOutput, ok := output.(languagemodel.ToolResultOutputJSON); ok {
							shellOutput := convertShellOutput(jsonOutput.Value)
							var outputEntries []OpenAIResponsesShellOutputEntry
							for _, item := range shellOutput {
								entry := OpenAIResponsesShellOutputEntry{
									Stdout: item.Stdout,
									Stderr: item.Stderr,
								}
								if item.Outcome.Type == "timeout" {
									entry.Outcome = OpenAIResponsesShellCallOutcome{Type: "timeout"}
								} else {
									entry.Outcome = OpenAIResponsesShellCallOutcome{
										Type:     "exit",
										ExitCode: item.Outcome.ExitCode,
									}
								}
								outputEntries = append(outputEntries, entry)
							}
							input = append(input, OpenAIResponsesShellCallOutput{
								Type:   "shell_call_output",
								CallID: p.ToolCallID,
								Output: outputEntries,
							})
						}
						break
					}

					if opts.Store {
						itemID := getStringProviderOption(p.ProviderOptions, opts.ProviderOptionsName, "itemId")
						if itemID == "" {
							itemID = p.ToolCallID
						}
						input = append(input, OpenAIResponsesItemReference{
							Type: "item_reference",
							ID:   itemID,
						})
					} else {
						warnings = append(warnings, shared.OtherWarning{
							Message: fmt.Sprintf("Results for OpenAI tool %s are not sent to the API when store is false", p.ToolName),
						})
					}

				case languagemodel.ReasoningPart:
					reasoningOpts := parseReasoningProviderOptions(p.ProviderOptions, opts.ProviderOptionsName)
					reasoningID := reasoningOpts.ItemID

					if opts.HasConversation && reasoningID != "" {
						break
					}

					if reasoningID != "" {
						existing, exists := reasoningMessages[reasoningID]

						if opts.Store {
							if !exists {
								input = append(input, OpenAIResponsesItemReference{
									Type: "item_reference",
									ID:   reasoningID,
								})
								reasoningMessages[reasoningID] = &OpenAIResponsesReasoning{
									Type:    "reasoning",
									ID:      reasoningID,
									Summary: nil,
								}
							}
						} else {
							var summaryParts []OpenAIResponsesReasoningSummary
							if len(p.Text) > 0 {
								summaryParts = append(summaryParts, OpenAIResponsesReasoningSummary{
									Type: "summary_text",
									Text: p.Text,
								})
							} else if exists {
								warnings = append(warnings, shared.OtherWarning{
									Message: fmt.Sprintf("Cannot append empty reasoning part to existing reasoning sequence. Skipping reasoning part."),
								})
							}

							if !exists {
								reasoning := &OpenAIResponsesReasoning{
									Type:    "reasoning",
									ID:      reasoningID,
									Summary: summaryParts,
								}
								if reasoningOpts.EncryptedContent != nil {
									reasoning.EncryptedContent = reasoningOpts.EncryptedContent
								}
								reasoningMessages[reasoningID] = reasoning
								input = append(input, *reasoning)
							} else {
								existing.Summary = append(existing.Summary, summaryParts...)
								if reasoningOpts.EncryptedContent != nil {
									existing.EncryptedContent = reasoningOpts.EncryptedContent
								}
								// Update the copy already in the input slice
								for idx, item := range input {
									if r, ok := item.(OpenAIResponsesReasoning); ok && r.ID == reasoningID {
										input[idx] = *existing
										break
									}
								}
							}
						}
					} else {
						// No itemId - fall back to encrypted_content if available
						encryptedContent := reasoningOpts.EncryptedContent

						if encryptedContent != nil {
							var summaryParts []OpenAIResponsesReasoningSummary
							if len(p.Text) > 0 {
								summaryParts = append(summaryParts, OpenAIResponsesReasoningSummary{
									Type: "summary_text",
									Text: p.Text,
								})
							}
							input = append(input, OpenAIResponsesReasoning{
								Type:             "reasoning",
								EncryptedContent: encryptedContent,
								Summary:          summaryParts,
							})
						} else {
							warnings = append(warnings, shared.OtherWarning{
								Message: "Non-OpenAI reasoning parts are not supported. Skipping reasoning part.",
							})
						}
					}
				}
			}

		case languagemodel.ToolMessage:
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.ToolApprovalResponsePart:
					if _, ok := processedApprovalIDs[p.ApprovalID]; ok {
						continue
					}
					processedApprovalIDs[p.ApprovalID] = struct{}{}

					if opts.Store {
						input = append(input, OpenAIResponsesItemReference{
							Type: "item_reference",
							ID:   p.ApprovalID,
						})
					}

					input = append(input, OpenAIResponsesMcpApprovalResponse{
						Type:              "mcp_approval_response",
						ApprovalRequestID: p.ApprovalID,
						Approve:           p.Approved,
					})

				case languagemodel.ToolResultPart:
					output := p.Output

					// Skip execution-denied with approvalId
					if denied, ok := output.(languagemodel.ToolResultOutputExecutionDenied); ok {
						approvalID := getStringProviderOption(denied.ProviderOptions, "openai", "approvalId")
						if approvalID != "" {
							continue
						}
					}

					resolvedToolName := opts.ToolNameMapping.ToProviderToolName(p.ToolName)

					// Local shell tool results
					if opts.HasLocalShellTool && resolvedToolName == "local_shell" {
						if jsonOutput, ok := output.(languagemodel.ToolResultOutputJSON); ok {
							localShellOutput := convertLocalShellOutput(jsonOutput.Value)
							input = append(input, OpenAIResponsesLocalShellCallOutput{
								Type:   "local_shell_call_output",
								CallID: p.ToolCallID,
								Output: localShellOutput,
							})
						}
						continue
					}

					// Shell tool results
					if opts.HasShellTool && resolvedToolName == "shell" {
						if jsonOutput, ok := output.(languagemodel.ToolResultOutputJSON); ok {
							shellOutput := convertShellOutput(jsonOutput.Value)
							var outputEntries []OpenAIResponsesShellOutputEntry
							for _, item := range shellOutput {
								entry := OpenAIResponsesShellOutputEntry{
									Stdout: item.Stdout,
									Stderr: item.Stderr,
								}
								if item.Outcome.Type == "timeout" {
									entry.Outcome = OpenAIResponsesShellCallOutcome{Type: "timeout"}
								} else {
									entry.Outcome = OpenAIResponsesShellCallOutcome{
										Type:     "exit",
										ExitCode: item.Outcome.ExitCode,
									}
								}
								outputEntries = append(outputEntries, entry)
							}
							input = append(input, OpenAIResponsesShellCallOutput{
								Type:   "shell_call_output",
								CallID: p.ToolCallID,
								Output: outputEntries,
							})
						}
						continue
					}

					// Apply patch tool results
					if opts.HasApplyPatchTool && p.ToolName == "apply_patch" {
						if jsonOutput, ok := output.(languagemodel.ToolResultOutputJSON); ok {
							patchOutput := convertApplyPatchOutput(jsonOutput.Value)
							input = append(input, OpenAIResponsesApplyPatchCallOutput{
								Type:   "apply_patch_call_output",
								CallID: p.ToolCallID,
								Status: patchOutput.Status,
								Output: patchOutput.Output,
							})
						}
						continue
					}

					// Custom provider tool results
					if _, ok := opts.CustomProviderToolNames[resolvedToolName]; ok {
						outputValue := convertCustomToolOutput(output, &warnings)
						input = append(input, OpenAIResponsesCustomToolCallOutput{
							Type:   "custom_tool_call_output",
							CallID: p.ToolCallID,
							Output: outputValue,
						})
						continue
					}

					// Function call output (default)
					contentValue := convertFunctionCallOutput(output, &warnings)
					input = append(input, OpenAIResponsesFunctionCallOutput{
						Type:   "function_call_output",
						CallID: p.ToolCallID,
						Output: contentValue,
					})
				}
			}

		default:
			return nil, fmt.Errorf("unsupported role: %T", msg)
		}
	}

	return &ConvertToOpenAIResponsesInputResult{
		Input:    input,
		Warnings: warnings,
	}, nil
}

// convertUserFilePart converts a file part in a user message to the OpenAI format.
func convertUserFilePart(p languagemodel.FilePart, index int, providerOptionsName string, fileIDPrefixes []string) (any, error) {
	if strings.HasPrefix(p.MediaType, "image/") {
		mediaType := p.MediaType
		if mediaType == "image/*" {
			mediaType = "image/jpeg"
		}

		result := map[string]any{"type": "input_image"}

		switch d := p.Data.(type) {
		case languagemodel.DataContentString:
			if strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://") {
				result["image_url"] = d.Value
			} else if isFileID(d.Value, fileIDPrefixes) {
				result["file_id"] = d.Value
			} else {
				result["image_url"] = fmt.Sprintf("data:%s;base64,%s", mediaType, d.Value)
			}
		case languagemodel.DataContentBytes:
			b64 := providerutils.ConvertBytesToBase64(d.Data)
			result["image_url"] = fmt.Sprintf("data:%s;base64,%s", mediaType, b64)
		}

		// Check for imageDetail provider option
		detail := getProviderOptionValue(p.ProviderOptions, providerOptionsName, "imageDetail")
		if detail != nil {
			result["detail"] = detail
		}

		return result, nil
	} else if p.MediaType == "application/pdf" {
		result := map[string]any{"type": "input_file"}

		switch d := p.Data.(type) {
		case languagemodel.DataContentString:
			if strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://") {
				result["file_url"] = d.Value
			} else if isFileID(d.Value, fileIDPrefixes) {
				result["file_id"] = d.Value
			} else {
				filename := fmt.Sprintf("part-%d.pdf", index)
				if p.Filename != nil {
					filename = *p.Filename
				}
				result["filename"] = filename
				result["file_data"] = fmt.Sprintf("data:application/pdf;base64,%s", d.Value)
			}
		case languagemodel.DataContentBytes:
			filename := fmt.Sprintf("part-%d.pdf", index)
			if p.Filename != nil {
				filename = *p.Filename
			}
			b64 := providerutils.ConvertBytesToBase64(d.Data)
			result["filename"] = filename
			result["file_data"] = fmt.Sprintf("data:application/pdf;base64,%s", b64)
		}

		return result, nil
	}

	return nil, fmt.Errorf("unsupported file part media type: %s", p.MediaType)
}

// Helper types for reasoning provider options parsing.
type reasoningProviderOptions struct {
	ItemID           string
	EncryptedContent *string
}

func parseReasoningProviderOptions(providerOpts shared.ProviderOptions, providerName string) reasoningProviderOptions {
	result := reasoningProviderOptions{}
	if providerOpts == nil {
		return result
	}

	optsMap, ok := providerOpts[providerName]
	if !ok {
		return result
	}

	if id, ok := optsMap["itemId"].(string); ok {
		result.ItemID = id
	}
	if ec, ok := optsMap["reasoningEncryptedContent"].(string); ok {
		result.EncryptedContent = &ec
	}

	return result
}

// getStringProviderOption extracts a string value from provider options.
func getStringProviderOption(providerOpts shared.ProviderOptions, providerName string, key string) string {
	if providerOpts == nil {
		return ""
	}
	optsMap, ok := providerOpts[providerName]
	if !ok {
		return ""
	}
	val, ok := optsMap[key].(string)
	if !ok {
		return ""
	}
	return val
}

// getProviderOptionValue extracts a value from provider options.
func getProviderOptionValue(providerOpts shared.ProviderOptions, providerName string, key string) any {
	if providerOpts == nil {
		return nil
	}
	optsMap, ok := providerOpts[providerName]
	if !ok {
		return nil
	}
	return optsMap[key]
}

// getToolCallItemID extracts the item ID from a tool call part.
func getToolCallItemID(p languagemodel.ToolCallPart, providerOptionsName string) string {
	if id := getStringProviderOption(p.ProviderOptions, providerOptionsName, "itemId"); id != "" {
		return id
	}
	return ""
}

// isExecutionDenied checks if the output is an execution-denied result.
func isExecutionDenied(output languagemodel.ToolResultOutput) bool {
	if _, ok := output.(languagemodel.ToolResultOutputExecutionDenied); ok {
		return true
	}
	// Also check for JSON output that contains an execution-denied type
	if jsonOutput, ok := output.(languagemodel.ToolResultOutputJSON); ok {
		if m, ok := jsonOutput.Value.(map[string]any); ok {
			if t, ok := m["type"].(string); ok && t == "execution-denied" {
				return true
			}
		}
	}
	return false
}

// convertLocalShellInput converts a tool call input into a LocalShellInput.
func convertLocalShellInput(input any) LocalShellInput {
	var result LocalShellInput
	data, err := json.Marshal(input)
	if err == nil {
		json.Unmarshal(data, &result)
	}
	return result
}

// convertLocalShellOutput extracts the output string from a local shell tool result.
func convertLocalShellOutput(value any) string {
	if m, ok := value.(map[string]any); ok {
		if output, ok := m["output"].(string); ok {
			return output
		}
	}
	return ""
}

// convertShellInput converts a tool call input into a ShellInput.
func convertShellInput(input any) ShellInput {
	var result ShellInput
	data, err := json.Marshal(input)
	if err == nil {
		json.Unmarshal(data, &result)
	}
	return result
}

// shellOutputEntry is used for parsing shell output from JSON values.
type shellOutputEntry struct {
	Stdout  string
	Stderr  string
	Outcome struct {
		Type     string
		ExitCode *int
	}
}

// convertShellOutput converts a JSON value into shell output entries.
func convertShellOutput(value any) []shellOutputEntry {
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	outputArr, ok := m["output"].([]any)
	if !ok {
		return nil
	}

	var entries []shellOutputEntry
	for _, item := range outputArr {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := shellOutputEntry{}
		if v, ok := itemMap["stdout"].(string); ok {
			entry.Stdout = v
		}
		if v, ok := itemMap["stderr"].(string); ok {
			entry.Stderr = v
		}
		if outcome, ok := itemMap["outcome"].(map[string]any); ok {
			if t, ok := outcome["type"].(string); ok {
				entry.Outcome.Type = t
			}
			if ec, ok := outcome["exitCode"]; ok {
				switch v := ec.(type) {
				case float64:
					i := int(v)
					entry.Outcome.ExitCode = &i
				case int:
					entry.Outcome.ExitCode = &v
				}
			}
		}
		entries = append(entries, entry)
	}
	return entries
}

// convertApplyPatchInput converts a tool call input into an ApplyPatchInput.
func convertApplyPatchInput(input any) ApplyPatchInput {
	var result ApplyPatchInput
	data, err := json.Marshal(input)
	if err == nil {
		json.Unmarshal(data, &result)
	}
	return result
}

// convertApplyPatchOutput converts a JSON value into an ApplyPatchOutput.
func convertApplyPatchOutput(value any) ApplyPatchOutput {
	var result ApplyPatchOutput
	data, err := json.Marshal(value)
	if err == nil {
		json.Unmarshal(data, &result)
	}
	return result
}

// convertCustomToolOutput converts tool result output for custom tools.
func convertCustomToolOutput(output languagemodel.ToolResultOutput, warnings *[]shared.Warning) any {
	switch o := output.(type) {
	case languagemodel.ToolResultOutputText:
		return o.Value
	case languagemodel.ToolResultOutputErrorText:
		return o.Value
	case languagemodel.ToolResultOutputExecutionDenied:
		if o.Reason != nil {
			return *o.Reason
		}
		return "Tool execution denied."
	case languagemodel.ToolResultOutputJSON:
		data, _ := json.Marshal(o.Value)
		return string(data)
	case languagemodel.ToolResultOutputErrorJSON:
		data, _ := json.Marshal(o.Value)
		return string(data)
	case languagemodel.ToolResultOutputContent:
		return convertContentOutputParts(o.Value, warnings)
	default:
		return ""
	}
}

// convertFunctionCallOutput converts tool result output for function call outputs.
func convertFunctionCallOutput(output languagemodel.ToolResultOutput, warnings *[]shared.Warning) any {
	switch o := output.(type) {
	case languagemodel.ToolResultOutputText:
		return o.Value
	case languagemodel.ToolResultOutputErrorText:
		return o.Value
	case languagemodel.ToolResultOutputExecutionDenied:
		if o.Reason != nil {
			return *o.Reason
		}
		return "Tool execution denied."
	case languagemodel.ToolResultOutputJSON:
		data, _ := json.Marshal(o.Value)
		return string(data)
	case languagemodel.ToolResultOutputErrorJSON:
		data, _ := json.Marshal(o.Value)
		return string(data)
	case languagemodel.ToolResultOutputContent:
		return convertContentOutputParts(o.Value, warnings)
	default:
		return ""
	}
}

// convertContentOutputParts converts content parts to OpenAI API format.
func convertContentOutputParts(parts []languagemodel.ToolResultContentPart, warnings *[]shared.Warning) []any {
	var result []any
	for _, part := range parts {
		switch p := part.(type) {
		case languagemodel.ToolResultContentText:
			result = append(result, map[string]any{
				"type": "input_text",
				"text": p.Text,
			})
		case languagemodel.ToolResultContentImageData:
			result = append(result, map[string]any{
				"type":      "input_image",
				"image_url": fmt.Sprintf("data:%s;base64,%s", p.MediaType, p.Data),
			})
		case languagemodel.ToolResultContentImageURL:
			result = append(result, map[string]any{
				"type":      "input_image",
				"image_url": p.URL,
			})
		case languagemodel.ToolResultContentFileData:
			filename := "data"
			if p.Filename != nil {
				filename = *p.Filename
			}
			result = append(result, map[string]any{
				"type":      "input_file",
				"filename":  filename,
				"file_data": fmt.Sprintf("data:%s;base64,%s", p.MediaType, p.Data),
			})
		default:
			if warnings != nil {
				*warnings = append(*warnings, shared.OtherWarning{
					Message: fmt.Sprintf("unsupported tool content part type: %T", part),
				})
			}
		}
	}
	return result
}

