// Ported from: packages/ai/src/generate-text/to-response-messages.ts
package generatetext

// ToResponseMessages converts the result of a generateText or streamText call
// to a list of response messages.
func ToResponseMessages(content []ContentPart, tools ToolSet) []ResponseMessage {
	var responseMessages []ResponseMessage

	var assistantContent []ModelMessageContent
	for _, part := range content {
		// Skip sources - they are response-only content
		if part.Type == "source" {
			continue
		}

		// Skip non-provider-executed tool results/errors (they go in the tool message)
		if part.Type == "tool-result" && part.ToolResult != nil && !part.ToolResult.ProviderExecuted {
			continue
		}
		if part.Type == "tool-error" && part.ToolError != nil && !part.ToolError.ProviderExecuted {
			continue
		}

		// Skip empty text
		if part.Type == "text" && part.Text == "" {
			continue
		}

		switch part.Type {
		case "text":
			assistantContent = append(assistantContent, ModelMessageContent{
				Type:            "text",
				Text:            part.Text,
				ProviderOptions: part.ProviderMetadata,
			})
		case "reasoning":
			assistantContent = append(assistantContent, ModelMessageContent{
				Type:            "reasoning",
				Text:            part.Text,
				ProviderOptions: part.ProviderMetadata,
			})
		case "file":
			if part.File != nil {
				assistantContent = append(assistantContent, ModelMessageContent{
					Type:            "file",
					Data:            part.File.Base64(),
					MediaType:       part.File.GetMediaType(),
					ProviderOptions: part.ProviderMetadata,
				})
			}
		case "tool-call":
			if part.ToolCall != nil {
				assistantContent = append(assistantContent, ModelMessageContent{
					Type:             "tool-call",
					ToolCallID:       part.ToolCall.ToolCallID,
					ToolName:         part.ToolCall.ToolName,
					Input:            part.ToolCall.Input,
					ProviderExecuted: part.ToolCall.ProviderExecuted,
					ProviderOptions:  part.ProviderMetadata,
				})
			}
		case "tool-result":
			if part.ToolResult != nil {
				assistantContent = append(assistantContent, ModelMessageContent{
					Type:            "tool-result",
					ToolCallID:      part.ToolResult.ToolCallID,
					ToolName:        part.ToolResult.ToolName,
					Output:          part.ToolResult.Output,
					ProviderOptions: part.ProviderMetadata,
				})
			}
		case "tool-error":
			if part.ToolError != nil {
				assistantContent = append(assistantContent, ModelMessageContent{
					Type:            "tool-result",
					ToolCallID:      part.ToolError.ToolCallID,
					ToolName:        part.ToolError.ToolName,
					Output:          part.ToolError.Error,
					ProviderOptions: part.ProviderMetadata,
				})
			}
		case "tool-approval-request":
			if part.ToolApprovalRequest != nil {
				assistantContent = append(assistantContent, ModelMessageContent{
					Type:       "tool-approval-request",
					ApprovalID: part.ToolApprovalRequest.ApprovalID,
					ToolCallID: part.ToolApprovalRequest.ToolCall.ToolCallID,
				})
			}
		}
	}

	if len(assistantContent) > 0 {
		responseMessages = append(responseMessages, ModelMessage{
			Role:    "assistant",
			Content: assistantContent,
		})
	}

	// Build tool result content
	var toolResultContent []ModelMessageContent
	for _, part := range content {
		if part.Type == "tool-result" && part.ToolResult != nil && !part.ToolResult.ProviderExecuted {
			toolResultContent = append(toolResultContent, ModelMessageContent{
				Type:            "tool-result",
				ToolCallID:      part.ToolResult.ToolCallID,
				ToolName:        part.ToolResult.ToolName,
				Output:          part.ToolResult.Output,
				ProviderOptions: part.ProviderMetadata,
			})
		}
		if part.Type == "tool-error" && part.ToolError != nil && !part.ToolError.ProviderExecuted {
			toolResultContent = append(toolResultContent, ModelMessageContent{
				Type:            "tool-result",
				ToolCallID:      part.ToolError.ToolCallID,
				ToolName:        part.ToolError.ToolName,
				Output:          part.ToolError.Error,
				ProviderOptions: part.ProviderMetadata,
			})
		}
	}

	if len(toolResultContent) > 0 {
		responseMessages = append(responseMessages, ModelMessage{
			Role:    "tool",
			Content: toolResultContent,
		})
	}

	return responseMessages
}
