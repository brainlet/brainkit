// Ported from: packages/ai/src/prompt/convert-to-language-model-prompt.validation.test.ts
package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolValidation(t *testing.T) {
	t.Run("should pass validation for provider-executed tools (deferred results)", func(t *testing.T) {
		providerExecuted := true
		result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
			Prompt: StandardizedPrompt{
				Messages: []ModelMessage{
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							ToolCallPart{
								Type:             "tool-call",
								ToolCallID:       "call_1",
								ToolName:         "code_interpreter",
								Input:            map[string]interface{}{"code": "print(\"hello\")"},
								ProviderExecuted: &providerExecuted,
							},
						},
					},
				},
			},
			SupportedUrls: map[string][]string{},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Len(t, result, 1)
		assert.Equal(t, "assistant", result[0].Role)
	})

	t.Run("should pass validation for tool-approval-response", func(t *testing.T) {
		result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
			Prompt: StandardizedPrompt{
				Messages: []ModelMessage{
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							ToolCallPart{
								Type:       "tool-call",
								ToolCallID: "call_to_approve",
								ToolName:   "dangerous_action",
								Input:      map[string]interface{}{"action": "delete_db"},
							},
							ToolApprovalRequest{
								Type:       "tool-approval-request",
								ToolCallID: "call_to_approve",
								ApprovalID: "approval_123",
							},
						},
					},
					ToolModelMessage{
						Role: "tool",
						Content: []interface{}{
							ToolApprovalResponse{
								Type:       "tool-approval-response",
								ApprovalID: "approval_123",
								Approved:   true,
							},
						},
					},
				},
			},
			SupportedUrls: map[string][]string{},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("should preserve provider-executed tool-approval-response", func(t *testing.T) {
		providerExecuted := true
		result, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
			Prompt: StandardizedPrompt{
				Messages: []ModelMessage{
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							ToolCallPart{
								Type:             "tool-call",
								ToolCallID:       "call_provider_executed",
								ToolName:         "mcp_tool",
								Input:            map[string]interface{}{"action": "execute"},
								ProviderExecuted: &providerExecuted,
							},
							ToolApprovalRequest{
								Type:       "tool-approval-request",
								ToolCallID: "call_provider_executed",
								ApprovalID: "approval_provider",
							},
						},
					},
					ToolModelMessage{
						Role: "tool",
						Content: []interface{}{
							ToolApprovalResponse{
								Type:             "tool-approval-response",
								ApprovalID:       "approval_provider",
								Approved:         true,
								ProviderExecuted: &providerExecuted,
							},
						},
					},
				},
			},
			SupportedUrls: map[string][]string{},
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify the provider-executed tool-approval-response is preserved in the output
		// The tool call is provider-executed so it shouldn't require a tool result
		require.Len(t, result, 2)
		assert.Equal(t, "assistant", result[0].Role)
		assert.Equal(t, "tool", result[1].Role)

		// The tool message should contain the approval response (since it's providerExecuted)
		toolParts, ok := result[1].Content.([]interface{})
		require.True(t, ok)
		require.Len(t, toolParts, 1)
		approvalResp, ok := toolParts[0].(LanguageModelV4ApprovalResponsePart)
		require.True(t, ok)
		assert.Equal(t, "tool-approval-response", approvalResp.Type)
		assert.Equal(t, "approval_provider", approvalResp.ApprovalID)
		assert.True(t, approvalResp.Approved)
	})

	t.Run("should throw error for actual missing results", func(t *testing.T) {
		_, err := ConvertToLanguageModelPrompt(ConvertToLanguageModelPromptOptions{
			Prompt: StandardizedPrompt{
				Messages: []ModelMessage{
					AssistantModelMessage{
						Role: "assistant",
						Content: []interface{}{
							ToolCallPart{
								Type:       "tool-call",
								ToolCallID: "call_missing_result",
								ToolName:   "regular_tool",
								Input:      map[string]interface{}{},
							},
						},
					},
				},
			},
			SupportedUrls: map[string][]string{},
		})

		require.Error(t, err)
		assert.True(t, IsMissingToolResultsError(err))
	})
}
