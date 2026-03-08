// Ported from: packages/ai/src/prompt/convert-to-language-model-prompt.ts
package prompt

import (
	"fmt"
	"net/url"
)

// LanguageModelV4TextPart represents a text part in the language model format.
type LanguageModelV4TextPart struct {
	Type            string          `json:"type"` // always "text"
	Text            string          `json:"text"`
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

// LanguageModelV4FilePart represents a file part in the language model format.
type LanguageModelV4FilePart struct {
	Type            string          `json:"type"` // always "file"
	Data            interface{}     `json:"data"` // []byte, string, or *url.URL
	Filename        *string         `json:"filename,omitempty"`
	MediaType       string          `json:"mediaType"`
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

// LanguageModelV4ToolCallPart represents a tool call part in the language model format.
type LanguageModelV4ToolCallPart struct {
	Type             string          `json:"type"` // always "tool-call"
	ToolCallID       string          `json:"toolCallId"`
	ToolName         string          `json:"toolName"`
	Input            interface{}     `json:"input"`
	ProviderExecuted *bool           `json:"providerExecuted,omitempty"`
	ProviderOptions  ProviderOptions `json:"providerOptions,omitempty"`
}

// LanguageModelV4ToolResultPart represents a tool result part in the language model format.
type LanguageModelV4ToolResultPart struct {
	Type            string           `json:"type"` // always "tool-result"
	ToolCallID      string           `json:"toolCallId"`
	ToolName        string           `json:"toolName"`
	Output          ToolResultOutput `json:"output"`
	ProviderOptions ProviderOptions  `json:"providerOptions,omitempty"`
}

// LanguageModelV4ReasoningPart represents a reasoning part in the language model format.
type LanguageModelV4ReasoningPart struct {
	Type            string          `json:"type"` // always "reasoning"
	Text            string          `json:"text"`
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

// LanguageModelV4ApprovalResponsePart represents a tool approval response in the language model format.
type LanguageModelV4ApprovalResponsePart struct {
	Type       string  `json:"type"` // always "tool-approval-response"
	ApprovalID string  `json:"approvalId"`
	Approved   bool    `json:"approved"`
	Reason     *string `json:"reason,omitempty"`
}

// LanguageModelV4Message represents a message in the language model format.
type LanguageModelV4Message struct {
	Role            string          `json:"role"`
	Content         interface{}     `json:"content"` // string or []interface{}
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

// LanguageModelV4Prompt is a slice of language model messages.
type LanguageModelV4Prompt = []LanguageModelV4Message

// DownloadedAssets maps URLs to their downloaded data.
type DownloadedAssets = map[string]DownloadedAsset

// DownloadedAsset represents a downloaded file.
type DownloadedAsset struct {
	Data      []byte
	MediaType *string
}

// MissingToolResultsError is returned when tool calls are missing results.
type MissingToolResultsError struct {
	Name        string
	Message     string
	ToolCallIDs []string
}

func (e *MissingToolResultsError) Error() string {
	return e.Message
}

// NewMissingToolResultsError creates a new MissingToolResultsError.
func NewMissingToolResultsError(toolCallIDs []string) *MissingToolResultsError {
	return &MissingToolResultsError{
		Name:        "AI_MissingToolResultsError",
		Message:     fmt.Sprintf("Missing tool results for tool call IDs: %v", toolCallIDs),
		ToolCallIDs: toolCallIDs,
	}
}

// IsMissingToolResultsError checks whether the given error is a MissingToolResultsError.
func IsMissingToolResultsError(err error) bool {
	_, ok := err.(*MissingToolResultsError)
	return ok
}

// ConvertToLanguageModelPromptOptions configures the prompt conversion.
type ConvertToLanguageModelPromptOptions struct {
	Prompt        StandardizedPrompt
	SupportedUrls map[string][]string // simplified from Record<string, RegExp[]>
	// Download is not ported as it requires async HTTP infra.
	// In Go, callers should pre-download assets and pass them via DownloadedAssets.
	DownloadedAssets DownloadedAssets
}

// ConvertToLanguageModelPrompt converts a StandardizedPrompt to a LanguageModelV4Prompt.
func ConvertToLanguageModelPrompt(opts ConvertToLanguageModelPromptOptions) (LanguageModelV4Prompt, error) {
	downloadedAssets := opts.DownloadedAssets
	if downloadedAssets == nil {
		downloadedAssets = DownloadedAssets{}
	}

	// Collect approval ID -> tool call ID mapping
	approvalIDToToolCallID := make(map[string]string)
	for _, msg := range opts.Prompt.Messages {
		if am, ok := msg.(AssistantModelMessage); ok {
			if parts, ok := am.Content.([]interface{}); ok {
				for _, part := range parts {
					if tar, ok := part.(ToolApprovalRequest); ok {
						approvalIDToToolCallID[tar.ApprovalID] = tar.ToolCallID
					}
				}
			}
		}
	}

	// Collect approved tool call IDs
	approvedToolCallIDs := make(map[string]bool)
	for _, msg := range opts.Prompt.Messages {
		if tm, ok := msg.(ToolModelMessage); ok {
			for _, part := range tm.Content {
				if tar, ok := part.(ToolApprovalResponse); ok {
					if toolCallID, exists := approvalIDToToolCallID[tar.ApprovalID]; exists {
						approvedToolCallIDs[toolCallID] = true
					}
				}
			}
		}
	}

	// Build messages
	var messages []LanguageModelV4Message

	// Add system messages
	if opts.Prompt.System != nil {
		switch sys := opts.Prompt.System.(type) {
		case string:
			messages = append(messages, LanguageModelV4Message{
				Role:    "system",
				Content: sys,
			})
		case SystemModelMessage:
			messages = append(messages, LanguageModelV4Message{
				Role:            "system",
				Content:         sys.Content,
				ProviderOptions: sys.ProviderOptions,
			})
		case []SystemModelMessage:
			for _, s := range sys {
				messages = append(messages, LanguageModelV4Message{
					Role:            "system",
					Content:         s.Content,
					ProviderOptions: s.ProviderOptions,
				})
			}
		}
	}

	// Convert each message
	for _, msg := range opts.Prompt.Messages {
		converted, err := convertToLanguageModelMessage(msg, downloadedAssets)
		if err != nil {
			return nil, err
		}
		messages = append(messages, converted)
	}

	// Combine consecutive tool messages
	var combinedMessages []LanguageModelV4Message
	for _, msg := range messages {
		if msg.Role != "tool" {
			combinedMessages = append(combinedMessages, msg)
			continue
		}

		if len(combinedMessages) > 0 {
			last := &combinedMessages[len(combinedMessages)-1]
			if last.Role == "tool" {
				// Merge tool message content
				if lastParts, ok := last.Content.([]interface{}); ok {
					if newParts, ok := msg.Content.([]interface{}); ok {
						last.Content = append(lastParts, newParts...)
						continue
					}
				}
			}
		}
		combinedMessages = append(combinedMessages, msg)
	}

	// Validate tool call IDs have matching results
	toolCallIDs := make(map[string]bool)
	for _, msg := range combinedMessages {
		switch msg.Role {
		case "assistant":
			if parts, ok := msg.Content.([]interface{}); ok {
				for _, part := range parts {
					if tc, ok := part.(LanguageModelV4ToolCallPart); ok {
						if tc.ProviderExecuted == nil || !*tc.ProviderExecuted {
							toolCallIDs[tc.ToolCallID] = true
						}
					}
				}
			}
		case "tool":
			if parts, ok := msg.Content.([]interface{}); ok {
				for _, part := range parts {
					if tr, ok := part.(LanguageModelV4ToolResultPart); ok {
						delete(toolCallIDs, tr.ToolCallID)
					}
				}
			}
		case "user", "system":
			// Remove approved tool calls before checking
			for id := range approvedToolCallIDs {
				delete(toolCallIDs, id)
			}
			if len(toolCallIDs) > 0 {
				ids := make([]string, 0, len(toolCallIDs))
				for id := range toolCallIDs {
					ids = append(ids, id)
				}
				return nil, NewMissingToolResultsError(ids)
			}
		}
	}

	// Final check
	for id := range approvedToolCallIDs {
		delete(toolCallIDs, id)
	}
	if len(toolCallIDs) > 0 {
		ids := make([]string, 0, len(toolCallIDs))
		for id := range toolCallIDs {
			ids = append(ids, id)
		}
		return nil, NewMissingToolResultsError(ids)
	}

	// Filter out empty tool messages
	var filtered LanguageModelV4Prompt
	for _, msg := range combinedMessages {
		if msg.Role == "tool" {
			if parts, ok := msg.Content.([]interface{}); ok && len(parts) == 0 {
				continue
			}
		}
		filtered = append(filtered, msg)
	}

	return filtered, nil
}

// convertToLanguageModelMessage converts a single ModelMessage to LanguageModelV4Message.
func convertToLanguageModelMessage(
	msg interface{},
	downloadedAssets DownloadedAssets,
) (LanguageModelV4Message, error) {
	switch m := msg.(type) {
	case SystemModelMessage:
		return LanguageModelV4Message{
			Role:            "system",
			Content:         m.Content,
			ProviderOptions: m.ProviderOptions,
		}, nil

	case UserModelMessage:
		switch content := m.Content.(type) {
		case string:
			return LanguageModelV4Message{
				Role:            "user",
				Content:         []interface{}{LanguageModelV4TextPart{Type: "text", Text: content}},
				ProviderOptions: m.ProviderOptions,
			}, nil
		case []interface{}:
			var parts []interface{}
			for _, part := range content {
				converted := convertPartToLanguageModelPart(part, downloadedAssets)
				// Filter empty text parts
				if tp, ok := converted.(LanguageModelV4TextPart); ok && tp.Text == "" {
					continue
				}
				parts = append(parts, converted)
			}
			return LanguageModelV4Message{
				Role:            "user",
				Content:         parts,
				ProviderOptions: m.ProviderOptions,
			}, nil
		default:
			return LanguageModelV4Message{
				Role:            "user",
				Content:         content,
				ProviderOptions: m.ProviderOptions,
			}, nil
		}

	case AssistantModelMessage:
		switch content := m.Content.(type) {
		case string:
			return LanguageModelV4Message{
				Role:            "assistant",
				Content:         []interface{}{LanguageModelV4TextPart{Type: "text", Text: content}},
				ProviderOptions: m.ProviderOptions,
			}, nil
		case []interface{}:
			var parts []interface{}
			for _, part := range content {
				switch p := part.(type) {
				case TextPart:
					if p.Text == "" && p.ProviderOptions == nil {
						continue
					}
					parts = append(parts, LanguageModelV4TextPart{
						Type:            "text",
						Text:            p.Text,
						ProviderOptions: p.ProviderOptions,
					})
				case ReasoningPart:
					parts = append(parts, LanguageModelV4ReasoningPart{
						Type:            "reasoning",
						Text:            p.Text,
						ProviderOptions: p.ProviderOptions,
					})
				case ToolCallPart:
					parts = append(parts, LanguageModelV4ToolCallPart{
						Type:             "tool-call",
						ToolCallID:       p.ToolCallID,
						ToolName:         p.ToolName,
						Input:            p.Input,
						ProviderExecuted: p.ProviderExecuted,
						ProviderOptions:  p.ProviderOptions,
					})
				case ToolResultPart:
					parts = append(parts, LanguageModelV4ToolResultPart{
						Type:            "tool-result",
						ToolCallID:      p.ToolCallID,
						ToolName:        p.ToolName,
						Output:          p.Output,
						ProviderOptions: p.ProviderOptions,
					})
				case FilePart:
					result := ConvertToLanguageModelV4DataContent(p.Data)
					mediaType := p.MediaType
					if mt, ok := result.Data.(string); ok && result.MediaType != nil {
						_ = mt
						mediaType = *result.MediaType
					}
					parts = append(parts, LanguageModelV4FilePart{
						Type:            "file",
						Data:            result.Data,
						Filename:        p.Filename,
						MediaType:       mediaType,
						ProviderOptions: p.ProviderOptions,
					})
				case ToolApprovalRequest:
					// filter out tool-approval-request
					continue
				default:
					parts = append(parts, part)
				}
			}
			return LanguageModelV4Message{
				Role:            "assistant",
				Content:         parts,
				ProviderOptions: m.ProviderOptions,
			}, nil
		default:
			return LanguageModelV4Message{
				Role:            "assistant",
				Content:         content,
				ProviderOptions: m.ProviderOptions,
			}, nil
		}

	case ToolModelMessage:
		var parts []interface{}
		for _, part := range m.Content {
			switch p := part.(type) {
			case ToolResultPart:
				parts = append(parts, LanguageModelV4ToolResultPart{
					Type:            "tool-result",
					ToolCallID:      p.ToolCallID,
					ToolName:        p.ToolName,
					Output:          p.Output,
					ProviderOptions: p.ProviderOptions,
				})
			case ToolApprovalResponse:
				if p.ProviderExecuted != nil && *p.ProviderExecuted {
					parts = append(parts, LanguageModelV4ApprovalResponsePart{
						Type:       "tool-approval-response",
						ApprovalID: p.ApprovalID,
						Approved:   p.Approved,
						Reason:     p.Reason,
					})
				}
			}
		}
		return LanguageModelV4Message{
			Role:            "tool",
			Content:         parts,
			ProviderOptions: m.ProviderOptions,
		}, nil

	default:
		role := GetMessageRole(msg)
		if role == "" {
			return LanguageModelV4Message{}, NewInvalidMessageRoleError("unknown", "")
		}
		return LanguageModelV4Message{}, NewInvalidMessageRoleError(role, "")
	}
}

// convertPartToLanguageModelPart converts a content part to language model format.
func convertPartToLanguageModelPart(part interface{}, downloadedAssets DownloadedAssets) interface{} {
	switch p := part.(type) {
	case TextPart:
		return LanguageModelV4TextPart{
			Type:            "text",
			Text:            p.Text,
			ProviderOptions: p.ProviderOptions,
		}
	case ImagePart:
		result := ConvertToLanguageModelV4DataContent(p.Image)
		mediaType := "image/*"
		if p.MediaType != nil {
			mediaType = *p.MediaType
		}
		if result.MediaType != nil {
			mediaType = *result.MediaType
		}

		data := result.Data
		// Check if URL was downloaded
		if u, ok := data.(*url.URL); ok {
			if asset, exists := downloadedAssets[u.String()]; exists {
				data = asset.Data
				if asset.MediaType != nil {
					mediaType = *asset.MediaType
				}
			}
		}

		return LanguageModelV4FilePart{
			Type:            "file",
			Data:            data,
			MediaType:       mediaType,
			ProviderOptions: p.ProviderOptions,
		}
	case FilePart:
		result := ConvertToLanguageModelV4DataContent(p.Data)
		mediaType := p.MediaType
		if result.MediaType != nil {
			mediaType = *result.MediaType
		}

		data := result.Data
		// Check if URL was downloaded
		if u, ok := data.(*url.URL); ok {
			if asset, exists := downloadedAssets[u.String()]; exists {
				data = asset.Data
				if asset.MediaType != nil {
					mediaType = *asset.MediaType
				}
			}
		}

		return LanguageModelV4FilePart{
			Type:            "file",
			Data:            data,
			Filename:        p.Filename,
			MediaType:       mediaType,
			ProviderOptions: p.ProviderOptions,
		}
	default:
		return part
	}
}
