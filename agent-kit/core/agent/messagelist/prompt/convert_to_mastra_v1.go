// Ported from: packages/core/src/agent/message-list/prompt/convert-to-mastra-v1.ts
package prompt

import (
	"regexp"
	"strings"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

var splitSuffixPattern = regexp.MustCompile(`__split-\d+$`)

// pushOrCombineState holds state for the push-or-combine function.
type pushOrCombineState struct {
	messages     []state.MastraMessageV1
	idUsageCount map[string]int
}

func newPushOrCombineState() *pushOrCombineState {
	return &pushOrCombineState{
		messages:     nil,
		idUsageCount: make(map[string]int),
	}
}

func (s *pushOrCombineState) push(msg state.MastraMessageV1) {
	if len(s.messages) > 0 {
		prev := &s.messages[len(s.messages)-1]
		// If same role and both have array content, combine (except tool-call assistant messages)
		prevArr, prevIsArr := prev.Content.([]map[string]any)
		msgArr, msgIsArr := msg.Content.([]map[string]any)
		if msg.Role == prev.Role && prevIsArr && msgIsArr {
			// Don't append tool calls to previous assistant message
			if msg.Role == "assistant" && len(msgArr) > 0 {
				lastPart := msgArr[len(msgArr)-1]
				if t, ok := lastPart["type"].(string); ok && t == "tool-call" {
					goto pushNew
				}
			}
			prev.Content = append(prevArr, msgArr...)
			return
		}
	}

pushNew:
	baseID := msg.ID
	if splitSuffixPattern.MatchString(baseID) {
		s.messages = append(s.messages, msg)
		return
	}

	currentCount := s.idUsageCount[baseID]
	if currentCount > 0 {
		msg.ID = baseID + "__split-" + itoa(currentCount)
	}
	s.idUsageCount[baseID] = currentCount + 1
	s.messages = append(s.messages, msg)
}

func itoa(n int) string {
	return strings.TrimLeft(strings.Replace(strings.Replace(strings.Replace(strings.Replace(strings.Replace(strings.Replace(strings.Replace(strings.Replace(strings.Replace(strings.Replace("0000000000", "0", string(rune('0'+n%10)), 1), "0", "", -1), "", "", 0), "", "", 0), "", "", 0), "", "", 0), "", "", 0), "", "", 0), "", "", 0), "", "", 0), "0")
}

// ConvertToV1Messages converts an array of MastraDBMessage to MastraMessageV1 format.
// This is a faithful port of the complex TS conversion logic.
func ConvertToV1Messages(messages []state.MastraDBMessage) []state.MastraMessageV1 {
	s := newPushOrCombineState()

	for i, message := range messages {
		isLastMessage := i == len(messages)-1
		content := message.Content
		role := message.Role

		fields := struct {
			ID         string
			CreatedAt  interface{}
			ResourceID string
			ThreadID   string
		}{
			ID:         message.ID,
			CreatedAt:  message.CreatedAt,
			ResourceID: message.ResourceID,
			ThreadID:   message.ThreadID,
		}

		// Separate file parts from other parts
		var experimentalAttachments []Attachment
		for _, att := range content.ExperimentalAttachments {
			experimentalAttachments = append(experimentalAttachments, Attachment{
				URL:         att.URL,
				ContentType: att.ContentType,
			})
		}

		var parts []state.MastraMessagePart
		for _, part := range content.Parts {
			if part.Type == "file" {
				experimentalAttachments = append(experimentalAttachments, Attachment{
					URL:         part.Data,
					ContentType: part.MimeType,
				})
			} else {
				parts = append(parts, part)
			}
		}

		switch role {
		case "user":
			if len(parts) == 0 {
				// No parts, use content string
				contentStr := content.Content
				if contentStr == "" {
					contentStr = ""
				}

				var userContent any
				if len(experimentalAttachments) > 0 {
					attParts, _ := AttachmentsToParts(experimentalAttachments)
					arr := []map[string]any{{"type": "text", "text": contentStr}}
					for _, ap := range attParts {
						arr = append(arr, map[string]any{
							"type":     ap.Type,
							"data":     ap.Data,
							"mimeType": ap.MimeType,
						})
					}
					userContent = arr
				} else {
					userContent = contentStr
				}

				s.push(state.MastraMessageV1{
					ID:         fields.ID,
					Role:       "user",
					CreatedAt:  message.CreatedAt,
					ResourceID: fields.ResourceID,
					ThreadID:   fields.ThreadID,
					Type:       "text",
					Content:    userContent,
				})
			} else {
				// Has parts
				var textParts []map[string]any
				for _, part := range content.Parts {
					if part.Type == "text" {
						textParts = append(textParts, map[string]any{
							"type": "text",
							"text": part.Text,
						})
					}
				}

				var userContent any
				if len(experimentalAttachments) > 0 {
					attParts, _ := AttachmentsToParts(experimentalAttachments)
					var arr []map[string]any
					arr = append(arr, textParts...)
					for _, ap := range attParts {
						arr = append(arr, map[string]any{
							"type":     ap.Type,
							"data":     ap.Data,
							"mimeType": ap.MimeType,
						})
					}
					userContent = arr
				} else {
					if len(textParts) == 1 && content.Content != "" {
						userContent = content.Content
					} else {
						userContent = textParts
					}
				}

				s.push(state.MastraMessageV1{
					ID:         fields.ID,
					Role:       "user",
					CreatedAt:  message.CreatedAt,
					ResourceID: fields.ResourceID,
					ThreadID:   fields.ThreadID,
					Type:       "text",
					Content:    userContent,
				})
			}

		case "assistant":
			if len(content.Parts) > 0 {
				// Process parts in blocks
				currentStep := 0
				blockHasToolInvocations := false
				var block []state.MastraMessagePart

				processBlock := func() {
					var contentArr []map[string]any
					for _, part := range block {
						switch part.Type {
						case "file", "text":
							contentArr = append(contentArr, map[string]any{
								"type": part.Type,
								"text": part.Text,
							})
						case "reasoning":
							for _, detail := range part.Details {
								switch detail.Type {
								case "text":
									contentArr = append(contentArr, map[string]any{
										"type":      "reasoning",
										"text":      detail.Text,
										"signature": detail.Signature,
									})
								case "redacted":
									contentArr = append(contentArr, map[string]any{
										"type": "redacted-reasoning",
										"data": detail.Data,
									})
								}
							}
						case "tool-invocation":
							if part.ToolInvocation != nil && part.ToolInvocation.ToolName != "updateWorkingMemory" {
								contentArr = append(contentArr, map[string]any{
									"type":       "tool-call",
									"toolCallId": part.ToolInvocation.ToolCallID,
									"toolName":   part.ToolInvocation.ToolName,
									"args":       part.ToolInvocation.Args,
								})
							}
						}
					}

					msgType := "text"
					for _, c := range contentArr {
						if t, ok := c["type"].(string); ok && t == "tool-call" {
							msgType = "tool-call"
							break
						}
					}

					var finalContent any = contentArr
					if len(contentArr) == 1 && contentArr[0]["type"] == "text" {
						if text, ok := contentArr[0]["text"].(string); ok {
							finalContent = text
						}
					}

					s.push(state.MastraMessageV1{
						ID:         fields.ID,
						Role:       "assistant",
						CreatedAt:  message.CreatedAt,
						ResourceID: fields.ResourceID,
						ThreadID:   fields.ThreadID,
						Type:       msgType,
						Content:    finalContent,
					})

					// Check for tool invocations with results
					var invocationsWithResults []state.ToolInvocation
					for _, part := range block {
						if part.Type == "tool-invocation" && part.ToolInvocation != nil {
							if part.ToolInvocation.ToolName != "updateWorkingMemory" &&
								part.ToolInvocation.State == "result" {
								invocationsWithResults = append(invocationsWithResults, *part.ToolInvocation)
							}
						}
					}

					if len(invocationsWithResults) > 0 {
						var toolResults []map[string]any
						for _, inv := range invocationsWithResults {
							toolResults = append(toolResults, map[string]any{
								"type":       "tool-result",
								"toolCallId": inv.ToolCallID,
								"toolName":   inv.ToolName,
								"result":     inv.Result,
							})
						}
						s.push(state.MastraMessageV1{
							ID:         fields.ID,
							Role:       "tool",
							CreatedAt:  message.CreatedAt,
							ResourceID: fields.ResourceID,
							ThreadID:   fields.ThreadID,
							Type:       "tool-result",
							Content:    toolResults,
						})
					}

					block = nil
					blockHasToolInvocations = false
					currentStep++
				}

				for _, part := range content.Parts {
					switch part.Type {
					case "text":
						if blockHasToolInvocations {
							processBlock()
						}
						block = append(block, part)
					case "file", "reasoning":
						block = append(block, part)
					case "tool-invocation":
						hasNonToolContent := false
						for _, p := range block {
							if p.Type == "text" || p.Type == "file" || p.Type == "reasoning" {
								hasNonToolContent = true
								break
							}
						}
						step := 0
						if part.ToolInvocation != nil && part.ToolInvocation.Step != nil {
							step = *part.ToolInvocation.Step
						}
						if hasNonToolContent || step != currentStep {
							processBlock()
						}
						block = append(block, part)
						blockHasToolInvocations = true
					}
				}
				processBlock()

			} else {
				// No parts, use toolInvocations or content
				toolInvocations := content.ToolInvocations
				if len(toolInvocations) == 0 {
					contentStr := content.Content
					s.push(state.MastraMessageV1{
						ID:         fields.ID,
						Role:       "assistant",
						CreatedAt:  message.CreatedAt,
						ResourceID: fields.ResourceID,
						ThreadID:   fields.ThreadID,
						Type:       "text",
						Content:    contentStr,
					})
				} else {
					maxStep := 0
					for _, ti := range toolInvocations {
						step := 0
						if ti.Step != nil {
							step = *ti.Step
						}
						if step > maxStep {
							maxStep = step
						}
					}

					for step := 0; step <= maxStep; step++ {
						var stepInvocations []state.ToolInvocation
						for _, ti := range toolInvocations {
							tiStep := 0
							if ti.Step != nil {
								tiStep = *ti.Step
							}
							if tiStep == step && ti.ToolName != "updateWorkingMemory" {
								stepInvocations = append(stepInvocations, ti)
							}
						}
						if len(stepInvocations) == 0 {
							continue
						}

						var assistantContent []map[string]any
						if isLastMessage && content.Content != "" && step == 0 {
							assistantContent = append(assistantContent, map[string]any{
								"type": "text",
								"text": content.Content,
							})
						}
						for _, inv := range stepInvocations {
							assistantContent = append(assistantContent, map[string]any{
								"type":       "tool-call",
								"toolCallId": inv.ToolCallID,
								"toolName":   inv.ToolName,
								"args":       inv.Args,
							})
						}

						s.push(state.MastraMessageV1{
							ID:         fields.ID,
							Role:       "assistant",
							CreatedAt:  message.CreatedAt,
							ResourceID: fields.ResourceID,
							ThreadID:   fields.ThreadID,
							Type:       "tool-call",
							Content:    assistantContent,
						})

						var invocationsWithResults []state.ToolInvocation
						for _, ti := range stepInvocations {
							if ti.State == "result" {
								invocationsWithResults = append(invocationsWithResults, ti)
							}
						}

						if len(invocationsWithResults) > 0 {
							var toolResults []map[string]any
							for _, inv := range invocationsWithResults {
								toolResults = append(toolResults, map[string]any{
									"type":       "tool-result",
									"toolCallId": inv.ToolCallID,
									"toolName":   inv.ToolName,
									"result":     inv.Result,
								})
							}
							s.push(state.MastraMessageV1{
								ID:         fields.ID,
								Role:       "tool",
								CreatedAt:  message.CreatedAt,
								ResourceID: fields.ResourceID,
								ThreadID:   fields.ThreadID,
								Type:       "tool-result",
								Content:    toolResults,
							})
						}
					}

					if content.Content != "" && !isLastMessage {
						s.push(state.MastraMessageV1{
							ID:         fields.ID,
							Role:       "assistant",
							CreatedAt:  message.CreatedAt,
							ResourceID: fields.ResourceID,
							ThreadID:   fields.ThreadID,
							Type:       "text",
							Content:    content.Content,
						})
					}
				}
			}
		}
	}

	return s.messages
}
