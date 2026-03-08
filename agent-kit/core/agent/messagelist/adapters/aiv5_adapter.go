// Ported from: packages/core/src/agent/message-list/adapters/AIV5Adapter.ts
package adapters

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/prompt"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/utils"
)

// AIV5Adapter handles conversions between MastraDBMessage and AI SDK V5 formats.
type AIV5Adapter struct{}

// getToolName extracts tool name from AI SDK v5 tool type string.
// V5 format: "tool-${toolName}" or "dynamic-tool"
func getToolName(partType string) string {
	if partType == "dynamic-tool" {
		return "dynamic-tool"
	}
	if strings.HasPrefix(partType, "tool-") {
		return utils.SanitizeToolName(partType[5:])
	}
	return utils.SanitizeToolName(partType)
}

// AIV5UIMessage is a local representation of an AIV5 UIMessage.
type AIV5UIMessage struct {
	ID       string             `json:"id"`
	Role     string             `json:"role"`
	Parts    []AIV5UIPart       `json:"parts"`
	Metadata map[string]any     `json:"metadata,omitempty"`
}

// AIV5UIPart is a local representation of an AIV5 UIMessage part.
type AIV5UIPart struct {
	Type                 string           `json:"type"`
	Text                 string           `json:"text,omitempty"`
	ToolCallID           string           `json:"toolCallId,omitempty"`
	State                string           `json:"state,omitempty"`
	Input                any              `json:"input,omitempty"`
	Output               any              `json:"output,omitempty"`
	URL                  string           `json:"url,omitempty"`
	MediaType            string           `json:"mediaType,omitempty"`
	Filename             string           `json:"filename,omitempty"`
	SourceID             string           `json:"sourceId,omitempty"`
	Title                string           `json:"title,omitempty"`
	ProviderMetadata     ProviderMetadata `json:"providerMetadata,omitempty"`
	ProviderExecuted     *bool            `json:"providerExecuted,omitempty"`
	CallProviderMetadata ProviderMetadata `json:"callProviderMetadata,omitempty"`
	DataPayload          any              `json:"data,omitempty"`
}

// IsToolUIPart checks if a part is a tool UI part (type starts with "tool-" or is "dynamic-tool").
func IsToolUIPart(part AIV5UIPart) bool {
	return strings.HasPrefix(part.Type, "tool-") || part.Type == "dynamic-tool"
}

// AIV5ToUIMessage converts MastraDBMessage to AIV5 UIMessage.
func AIV5ToUIMessage(dbMsg *MastraDBMessage) *AIV5UIMessage {
	var parts []AIV5UIPart
	metadata := make(map[string]any)

	// Copy existing metadata
	if dbMsg.Content.Metadata != nil {
		for k, v := range dbMsg.Content.Metadata {
			metadata[k] = v
		}
	}

	// Add Mastra-specific metadata
	if !dbMsg.CreatedAt.IsZero() {
		metadata["createdAt"] = dbMsg.CreatedAt
	}
	if dbMsg.ThreadID != "" {
		metadata["threadId"] = dbMsg.ThreadID
	}
	if dbMsg.ResourceID != "" {
		metadata["resourceId"] = dbMsg.ResourceID
	}
	if dbMsg.Content.ProviderMetadata != nil {
		metadata["providerMetadata"] = dbMsg.Content.ProviderMetadata
	}

	// 1. Handle tool invocations (only if not already in parts array)
	hasToolInvocationParts := false
	for _, p := range dbMsg.Content.Parts {
		if p.Type == "tool-invocation" {
			hasToolInvocationParts = true
			break
		}
	}

	if len(dbMsg.Content.ToolInvocations) > 0 && !hasToolInvocationParts {
		for _, inv := range dbMsg.Content.ToolInvocations {
			if inv.State == "result" {
				parts = append(parts, AIV5UIPart{
					Type:       fmt.Sprintf("tool-%s", inv.ToolName),
					ToolCallID: inv.ToolCallID,
					State:      "output-available",
					Input:      inv.Args,
					Output:     inv.Result,
				})
			} else {
				st := "input-available"
				if inv.State == "partial-call" {
					st = "input-streaming"
				}
				parts = append(parts, AIV5UIPart{
					Type:       fmt.Sprintf("tool-%s", inv.ToolName),
					ToolCallID: inv.ToolCallID,
					State:      st,
					Input:      inv.Args,
				})
			}
		}
	}

	// 2. Check for parts with providerMetadata
	hasReasoningInParts := false
	hasFileInParts := false
	for _, p := range dbMsg.Content.Parts {
		if p.Type == "reasoning" {
			hasReasoningInParts = true
		}
		if p.Type == "file" {
			hasFileInParts = true
		}
	}

	// 3. Handle reasoning
	if dbMsg.Content.Reasoning != "" && !hasReasoningInParts {
		parts = append(parts, AIV5UIPart{
			Type: "reasoning",
			Text: dbMsg.Content.Reasoning,
		})
	}

	// 4. Handle files from experimental_attachments
	attachmentUrls := make(map[string]struct{})
	if len(dbMsg.Content.ExperimentalAttachments) > 0 && !hasFileInParts {
		for _, att := range dbMsg.Content.ExperimentalAttachments {
			attachmentUrls[att.URL] = struct{}{}
			parts = append(parts, AIV5UIPart{
				Type:      "file",
				URL:       att.URL,
				MediaType: att.ContentType,
			})
		}
	}

	// 5. Handle parts directly
	hasNonToolReasoningParts := false
	for _, part := range dbMsg.Content.Parts {
		switch {
		case part.Type == "tool-invocation" && part.ToolInvocation != nil:
			inv := part.ToolInvocation
			if inv.State == "result" {
				parts = append(parts, AIV5UIPart{
					Type:                 fmt.Sprintf("tool-%s", inv.ToolName),
					ToolCallID:           inv.ToolCallID,
					Input:                inv.Args,
					Output:               inv.Result,
					State:                "output-available",
					CallProviderMetadata: part.ProviderMetadata,
					ProviderExecuted:     part.ProviderExecuted,
				})
			} else {
				parts = append(parts, AIV5UIPart{
					Type:                 fmt.Sprintf("tool-%s", inv.ToolName),
					ToolCallID:           inv.ToolCallID,
					Input:                inv.Args,
					State:                "input-available",
					CallProviderMetadata: part.ProviderMetadata,
					ProviderExecuted:     part.ProviderExecuted,
				})
			}

		case part.Type == "reasoning":
			text := part.Reasoning
			if text == "" {
				for _, d := range part.Details {
					if d.Type == "text" && d.Text != "" {
						text += d.Text
					}
				}
			}
			if text != "" || len(part.Details) > 0 {
				p := AIV5UIPart{
					Type:  "reasoning",
					Text:  text,
					State: "done",
				}
				if part.ProviderMetadata != nil {
					p.ProviderMetadata = part.ProviderMetadata
				}
				parts = append(parts, p)
			}

		case part.Type == "tool-invocation" || strings.HasPrefix(part.Type, "tool-"):
			continue

		case part.Type == "file":
			if _, skip := attachmentUrls[part.Data]; skip {
				continue
			}
			categorized := prompt.CategorizeFileData(part.Data, part.MimeType)
			if categorized.Type == "url" {
				p := AIV5UIPart{
					Type:      "file",
					URL:       part.Data,
					MediaType: categorized.MimeType,
				}
				if p.MediaType == "" {
					p.MediaType = "image/png"
				}
				if part.ProviderMetadata != nil {
					p.ProviderMetadata = part.ProviderMetadata
				}
				parts = append(parts, p)
			} else {
				parsed := prompt.ParseDataUri(part.Data)
				filePartData := part.Data
				extractedMimeType := part.MimeType
				if parsed.IsDataUri {
					filePartData = parsed.Base64Content
					if parsed.MimeType != "" && extractedMimeType == "" {
						extractedMimeType = parsed.MimeType
					}
				}
				finalMimeType := extractedMimeType
				if finalMimeType == "" {
					finalMimeType = "image/png"
				}
				dataUri := filePartData
				if !strings.HasPrefix(filePartData, "data:") {
					dataUri = prompt.CreateDataUri(filePartData, finalMimeType)
				}
				p := AIV5UIPart{
					Type:      "file",
					URL:       dataUri,
					MediaType: finalMimeType,
				}
				if part.ProviderMetadata != nil {
					p.ProviderMetadata = part.ProviderMetadata
				}
				parts = append(parts, p)
			}

		case part.Type == "source" && part.Source != nil:
			p := AIV5UIPart{
				Type:     "source-url",
				URL:      part.Source.URL,
				SourceID: part.Source.ID,
				Title:    part.Source.Title,
			}
			if part.ProviderMetadata != nil {
				p.ProviderMetadata = part.ProviderMetadata
			}
			parts = append(parts, p)

		case part.Type == "text":
			p := AIV5UIPart{
				Type: "text",
				Text: part.Text,
			}
			if part.ProviderMetadata != nil {
				p.ProviderMetadata = part.ProviderMetadata
			}
			parts = append(parts, p)
			hasNonToolReasoningParts = true

		default:
			parts = append(parts, AIV5UIPart{Type: part.Type})
			hasNonToolReasoningParts = true
		}
	}

	// 6. Handle text content (fallback if no parts)
	if dbMsg.Content.Content != "" && !hasNonToolReasoningParts {
		parts = append(parts, AIV5UIPart{Type: "text", Text: dbMsg.Content.Content})
	}

	return &AIV5UIMessage{
		ID:       dbMsg.ID,
		Role:     dbMsg.Role,
		Metadata: metadata,
		Parts:    parts,
	}
}

// AIV5FromUIMessage converts AIV5 UIMessage to MastraDBMessage.
func AIV5FromUIMessage(uiMsg *AIV5UIMessage) *MastraDBMessage {
	metadata := make(map[string]any)
	if uiMsg.Metadata != nil {
		for k, v := range uiMsg.Metadata {
			metadata[k] = v
		}
	}

	// Extract Mastra-specific metadata
	createdAt := time.Now()
	if v, ok := metadata["createdAt"]; ok {
		switch ct := v.(type) {
		case string:
			if t, err := time.Parse(time.RFC3339, ct); err == nil {
				createdAt = t
			}
		case time.Time:
			createdAt = ct
		}
	}
	threadID, _ := metadata["threadId"].(string)
	resourceID, _ := metadata["resourceId"].(string)

	cleanMetadata := make(map[string]any)
	for k, v := range metadata {
		if k != "createdAt" && k != "threadId" && k != "resourceId" {
			cleanMetadata[k] = v
		}
	}

	// Process parts
	var toolInvocations []ToolInvocation
	var reasoningParts []string
	var experimentalAttachments []state.ExperimentalAttachment
	var contentStr string
	var v2Parts []MastraMessagePart

	for _, part := range uiMsg.Parts {
		if IsToolUIPart(part) {
			toolName := getToolName(part.Type)
			if part.State == "output-available" {
				output := part.Output
				// Unwrap {value: ...} wrapper
				if m, ok := output.(map[string]any); ok {
					if v, exists := m["value"]; exists {
						output = v
					}
				}
				toolInvocations = append(toolInvocations, ToolInvocation{
					Args:       anyToArgs(part.Input),
					Result:     output,
					ToolCallID: part.ToolCallID,
					ToolName:   toolName,
					State:      "result",
				})
				v2Parts = append(v2Parts, MastraMessagePart{
					Type: "tool-invocation",
					ToolInvocation: &ToolInvocation{
						ToolCallID: part.ToolCallID,
						ToolName:   toolName,
						Args:       anyToArgs(part.Input),
						Result:     output,
						State:      "result",
					},
					ProviderMetadata: part.CallProviderMetadata,
				})
			} else {
				toolInvocations = append(toolInvocations, ToolInvocation{
					Args:       anyToArgs(part.Input),
					ToolCallID: part.ToolCallID,
					ToolName:   toolName,
					State:      "call",
				})
				v2Parts = append(v2Parts, MastraMessagePart{
					Type: "tool-invocation",
					ToolInvocation: &ToolInvocation{
						ToolCallID: part.ToolCallID,
						ToolName:   toolName,
						Args:       anyToArgs(part.Input),
						State:      "call",
					},
					ProviderMetadata: part.CallProviderMetadata,
				})
			}
		} else if part.Type == "reasoning" {
			reasoningParts = append(reasoningParts, part.Text)
			v2Parts = append(v2Parts, MastraMessagePart{
				Type:             "reasoning",
				Reasoning:        "",
				Details:          []state.ReasoningDetail{{Type: "text", Text: part.Text}},
				ProviderMetadata: part.ProviderMetadata,
			})
		} else if part.Type == "file" {
			experimentalAttachments = append(experimentalAttachments, state.ExperimentalAttachment{
				URL:         part.URL,
				ContentType: part.MediaType,
			})
			v2Parts = append(v2Parts, MastraMessagePart{
				Type:             "file",
				MimeType:         part.MediaType,
				Data:             part.URL,
				ProviderMetadata: part.ProviderMetadata,
				Filename:         part.Filename,
			})
		} else if part.Type == "source-url" {
			v2Parts = append(v2Parts, MastraMessagePart{
				Type: "source",
				Source: &state.SourceInfo{
					URL:        part.URL,
					SourceType: "url",
					ID:         part.URL,
				},
				ProviderMetadata: part.ProviderMetadata,
			})
		} else if part.Type == "text" {
			contentStr += part.Text
			v2Parts = append(v2Parts, MastraMessagePart{
				Type:             "text",
				Text:             part.Text,
				ProviderMetadata: part.ProviderMetadata,
			})
		} else if part.Type == "step-start" {
			v2Parts = append(v2Parts, MastraMessagePart{Type: "step-start"})
		} else if strings.HasPrefix(part.Type, "data-") {
			v2Parts = append(v2Parts, MastraMessagePart{
				Type:        part.Type,
				DataPayload: part.DataPayload,
			})
		}
	}

	filteredV2Parts := filterEmptyTextParts(v2Parts)

	content := MastraMessageContentV2{
		Format: 2,
		Parts:  filteredV2Parts,
	}
	if len(toolInvocations) > 0 {
		content.ToolInvocations = toolInvocations
	}
	if len(reasoningParts) > 0 {
		content.Reasoning = strings.Join(reasoningParts, "\n")
	}
	if len(experimentalAttachments) > 0 {
		content.ExperimentalAttachments = experimentalAttachments
	}
	if contentStr != "" {
		content.Content = contentStr
	}
	if len(cleanMetadata) > 0 {
		content.Metadata = cleanMetadata
	}

	return &MastraDBMessage{
		MastraMessageShared: state.MastraMessageShared{
			ID:         uiMsg.ID,
			Role:       uiMsg.Role,
			CreatedAt:  createdAt,
			ThreadID:   threadID,
			ResourceID: resourceID,
		},
		Content: content,
	}
}

// AIV5FromModelMessage converts AIV5 ModelMessage to MastraDBMessage.
// modelMsg is represented as map[string]any since ModelMessage is a complex union type.
func AIV5FromModelMessage(modelMsg map[string]any, messageSource string) *MastraDBMessage {
	role, _ := modelMsg["role"].(string)
	content := modelMsg["content"]

	var contentParts []map[string]any
	if contentStr, ok := content.(string); ok {
		contentParts = []map[string]any{{"type": "text", "text": contentStr}}
	} else if arr, ok := content.([]any); ok {
		for _, item := range arr {
			if m, ok := item.(map[string]any); ok {
				contentParts = append(contentParts, m)
			}
		}
	}

	var mastraDBParts []MastraMessagePart
	var toolInvocations []ToolInvocation
	var reasoningParts []string
	var experimentalAttachments []state.ExperimentalAttachment

	for _, part := range contentParts {
		partType, _ := part["type"].(string)
		switch partType {
		case "text":
			text, _ := part["text"].(string)
			mp := MastraMessagePart{Type: "text", Text: text}
			if po, ok := part["providerOptions"].(map[string]map[string]any); ok {
				mp.ProviderMetadata = state.ProviderMetadata(po)
			}
			mastraDBParts = append(mastraDBParts, mp)

		case "tool-call":
			toolCallID, _ := part["toolCallId"].(string)
			toolName, _ := part["toolName"].(string)
			input := part["input"]
			mp := MastraMessagePart{
				Type: "tool-invocation",
				ToolInvocation: &ToolInvocation{
					ToolCallID: toolCallID,
					ToolName:   utils.SanitizeToolName(toolName),
					Args:       anyToArgs(input),
					State:      "call",
				},
			}
			mastraDBParts = append(mastraDBParts, mp)
			toolInvocations = append(toolInvocations, *mp.ToolInvocation)

		case "tool-result":
			toolCallID, _ := part["toolCallId"].(string)
			toolName, _ := part["toolName"].(string)
			output := part["output"]
			// Try to find matching call
			matchIdx := -1
			for j, inv := range toolInvocations {
				if inv.ToolCallID == toolCallID {
					matchIdx = j
					break
				}
			}
			if matchIdx >= 0 {
				toolInvocations[matchIdx].State = "result"
				toolInvocations[matchIdx].Result = unwrapOutput(output)
				// Update matching V2 part too
				for j := range mastraDBParts {
					if mastraDBParts[j].Type == "tool-invocation" &&
						mastraDBParts[j].ToolInvocation != nil &&
						mastraDBParts[j].ToolInvocation.ToolCallID == toolCallID {
						mastraDBParts[j].ToolInvocation.State = "result"
						mastraDBParts[j].ToolInvocation.Result = unwrapOutput(output)
						break
					}
				}
			} else {
				inv := ToolInvocation{
					State:      "result",
					ToolCallID: toolCallID,
					ToolName:   utils.SanitizeToolName(toolName),
					Args:       map[string]any{},
					Result:     unwrapOutput(output),
				}
				toolInvocations = append(toolInvocations, inv)
				mastraDBParts = append(mastraDBParts, MastraMessagePart{
					Type:           "tool-invocation",
					ToolInvocation: &inv,
				})
			}

		case "reasoning":
			text, _ := part["text"].(string)
			mp := MastraMessagePart{
				Type:      "reasoning",
				Reasoning: "",
				Details:   []state.ReasoningDetail{{Type: "text", Text: text}},
			}
			if po, ok := part["providerOptions"].(map[string]map[string]any); ok {
				mp.ProviderMetadata = state.ProviderMetadata(po)
			}
			mastraDBParts = append(mastraDBParts, mp)
			reasoningParts = append(reasoningParts, text)

		case "image":
			mediaType, _ := part["mediaType"].(string)
			if mediaType == "" {
				mediaType = "image/jpeg"
			}
			imageData := getDataStringFromAIV5DataPart(part)
			mp := MastraMessagePart{Type: "file", Data: imageData, MimeType: mediaType}
			mastraDBParts = append(mastraDBParts, mp)
			experimentalAttachments = append(experimentalAttachments, state.ExperimentalAttachment{
				URL: imageData, ContentType: mediaType,
			})

		case "file":
			mediaType, _ := part["mediaType"].(string)
			if mediaType == "" {
				mediaType = "application/octet-stream"
			}
			fileData := getDataStringFromAIV5DataPart(part)
			mp := MastraMessagePart{Type: "file", Data: fileData, MimeType: mediaType}
			if fn, ok := part["filename"].(string); ok {
				mp.Filename = fn
			}
			mastraDBParts = append(mastraDBParts, mp)
			experimentalAttachments = append(experimentalAttachments, state.ExperimentalAttachment{
				URL: fileData, ContentType: mediaType,
			})
		}
	}

	filteredParts := filterEmptyTextParts(mastraDBParts)

	// Build content string
	var contentStr string
	for _, p := range filteredParts {
		if p.Type == "text" {
			if contentStr != "" {
				contentStr += "\n"
			}
			contentStr += p.Text
		}
	}

	id, _ := modelMsg["id"].(string)
	if id == "" {
		id = fmt.Sprintf("msg-%d-%s", time.Now().UnixMilli(), randomString(9))
	}

	msgRole := role
	if msgRole == "tool" {
		msgRole = "assistant"
	}

	contentV2 := MastraMessageContentV2{
		Format: 2,
		Parts:  filteredParts,
	}
	if len(toolInvocations) > 0 {
		contentV2.ToolInvocations = toolInvocations
	}
	if len(reasoningParts) > 0 {
		contentV2.Reasoning = strings.Join(reasoningParts, "\n")
	}
	if len(experimentalAttachments) > 0 {
		contentV2.ExperimentalAttachments = experimentalAttachments
	}
	if contentStr != "" {
		contentV2.Content = contentStr
	}
	if meta, ok := modelMsg["metadata"].(map[string]any); ok && len(meta) > 0 {
		contentV2.Metadata = meta
	}
	if po, ok := modelMsg["providerOptions"].(map[string]map[string]any); ok {
		contentV2.ProviderMetadata = state.ProviderMetadata(po)
	}

	return &MastraDBMessage{
		MastraMessageShared: state.MastraMessageShared{
			ID:        id,
			Role:      msgRole,
			CreatedAt: time.Now(),
		},
		Content: contentV2,
	}
}

// getDataStringFromAIV5DataPart converts image or file data to a data URI or URL string.
func getDataStringFromAIV5DataPart(part map[string]any) string {
	var data any
	var mimeType string

	if d, ok := part["data"]; ok {
		data = d
		mimeType, _ = part["mediaType"].(string)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	} else if img, ok := part["image"]; ok {
		data = img
		mimeType, _ = part["mediaType"].(string)
		if mimeType == "" {
			mimeType = "image/jpeg"
		}
	} else {
		return ""
	}

	switch v := data.(type) {
	case string:
		if strings.HasPrefix(v, "data:") || strings.HasPrefix(v, "http") {
			return v
		}
		return fmt.Sprintf("data:%s;base64,%s", mimeType, v)
	case []byte:
		b64 := base64.StdEncoding.EncodeToString(v)
		return fmt.Sprintf("data:%s;base64,%s", mimeType, b64)
	default:
		return ""
	}
}

func unwrapOutput(output any) any {
	if m, ok := output.(map[string]any); ok {
		if v, exists := m["value"]; exists {
			return v
		}
	}
	return output
}

func anyToArgs(input any) map[string]any {
	if m, ok := input.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func randomString(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
	}
	return string(b)
}
