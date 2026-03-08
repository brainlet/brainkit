// Ported from: packages/core/src/agent/message-list/conversion/step-content.ts
package conversion

import (
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/adapters"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/prompt"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// StepResultContent represents the content of a step result.
// TODO: In TS this is AIV5Type.StepResult<any>['content'] — an array of content parts.
type StepResultContent []map[string]any

// StepContentFn converts a model message to step content.
type StepContentFn func(message map[string]any) StepResultContent

// StepContentExtractor handles extraction of step content from response messages.
type StepContentExtractor struct{}

// ExtractStepContent extracts content for a specific step number from UI messages.
// stepNumber is 1-indexed, or -1 for the last step.
func ExtractStepContent(
	uiMessages []*adapters.AIV5UIMessage,
	stepNumber int,
	stepContentFn StepContentFn,
) StepResultContent {
	// Flatten all parts from UI messages
	var allParts []adapters.AIV5UIPart
	for _, msg := range uiMessages {
		allParts = append(allParts, msg.Parts...)
	}

	// Find step boundaries by looking for step-start markers
	var stepBoundaries []int
	for i, part := range allParts {
		if part.Type == "step-start" {
			stepBoundaries = append(stepBoundaries, i)
		}
	}

	// Handle -1 to get the last step
	if stepNumber == -1 {
		return extractLastStep(allParts, stepBoundaries, stepContentFn)
	}

	// Step 1 is everything before the first step-start
	if stepNumber == 1 {
		return extractFirstStep(allParts, stepBoundaries, stepContentFn)
	}

	// For steps 2+, content is between (stepNumber-1)th and stepNumber-th step-start markers
	return extractMiddleStep(allParts, stepBoundaries, stepNumber, stepContentFn)
}

// extractLastStep extracts the last step content (stepNumber === -1).
func extractLastStep(
	allParts []adapters.AIV5UIPart,
	stepBoundaries []int,
	stepContentFn StepContentFn,
) StepResultContent {
	// For tool-only steps without step-start markers, we need different logic
	var toolParts []adapters.AIV5UIPart
	var toolIndices []int
	for i, p := range allParts {
		if adapters.IsToolUIPart(p) {
			toolParts = append(toolParts, p)
			toolIndices = append(toolIndices, i)
		}
	}
	hasStepStart := len(stepBoundaries) > 0

	if !hasStepStart && len(toolParts) > 0 {
		lastToolIndex := toolIndices[len(toolIndices)-1]
		previousToolIndex := -1
		if len(toolIndices) >= 2 {
			previousToolIndex = toolIndices[len(toolIndices)-2]
		}
		startIndex := previousToolIndex + 1
		stepParts := allParts[startIndex : lastToolIndex+1]
		return convertPartsToContent(stepParts, "last-step", stepContentFn)
	}

	totalSteps := len(stepBoundaries) + 1
	if totalSteps == 1 && !hasStepStart {
		return convertPartsToContent(allParts, "last-step", stepContentFn)
	}

	// Multiple steps - get content after the last step-start marker
	lastStepStart := stepBoundaries[len(stepBoundaries)-1]
	stepParts := allParts[lastStepStart+1:]
	if len(stepParts) == 0 {
		return nil
	}
	return convertPartsToContent(stepParts, "last-step", stepContentFn)
}

// extractFirstStep extracts the first step content (stepNumber === 1).
func extractFirstStep(
	allParts []adapters.AIV5UIPart,
	stepBoundaries []int,
	stepContentFn StepContentFn,
) StepResultContent {
	firstStepStart := len(allParts)
	if len(stepBoundaries) > 0 {
		firstStepStart = stepBoundaries[0]
	}
	if firstStepStart == 0 {
		return nil
	}
	stepParts := allParts[:firstStepStart]
	return convertPartsToContent(stepParts, "step-1", stepContentFn)
}

// extractMiddleStep extracts content for steps 2+ (between step-start markers).
func extractMiddleStep(
	allParts []adapters.AIV5UIPart,
	stepBoundaries []int,
	stepNumber int,
	stepContentFn StepContentFn,
) StepResultContent {
	stepIndex := stepNumber - 2 // -2 because step 2 is at index 0 in boundaries
	if stepIndex < 0 || stepIndex >= len(stepBoundaries) {
		return nil
	}

	startIndex := stepBoundaries[stepIndex] + 1
	endIndex := len(allParts)
	if stepIndex+1 < len(stepBoundaries) {
		endIndex = stepBoundaries[stepIndex+1]
	}

	if startIndex >= endIndex {
		return nil
	}

	stepParts := allParts[startIndex:endIndex]
	return convertPartsToContent(stepParts, "step-"+string(rune('0'+stepNumber)), stepContentFn)
}

// convertPartsToContent converts UI message parts to step content.
func convertPartsToContent(
	parts []adapters.AIV5UIPart,
	stepID string,
	stepContentFn StepContentFn,
) StepResultContent {
	stepUIMessages := []*adapters.AIV5UIMessage{
		{
			ID:    stepID,
			Role:  "assistant",
			Parts: parts,
		},
	}

	sanitized := SanitizeV5UIMessages(stepUIMessages, false)
	// TODO: Call AIV5.convertToModelMessages equivalent.
	// For now, basic conversion:
	var result StepResultContent
	for _, msg := range sanitized {
		modelMsg := map[string]any{
			"role": msg.Role,
		}
		var contentParts []map[string]any
		for _, part := range msg.Parts {
			if part.Type == "text" {
				contentParts = append(contentParts, map[string]any{
					"type": "text",
					"text": part.Text,
				})
			}
		}
		if len(contentParts) > 0 {
			modelMsg["content"] = contentParts
		}
		result = append(result, stepContentFn(modelMsg)...)
	}
	return result
}

// ConvertToStepContent converts a single model message content to step result content.
// Handles tool results, files, images, and other content types.
func ConvertToStepContent(
	message map[string]any,
	dbMessages []*state.MastraDBMessage,
	getLatestMessage func() map[string]any,
) StepResultContent {
	latest := message
	if latest == nil {
		latest = getLatestMessage()
	}
	if latest == nil {
		return nil
	}

	content := latest["content"]

	if contentStr, ok := content.(string); ok {
		return StepResultContent{
			{"type": "text", "text": contentStr},
		}
	}

	contentArr, ok := content.([]map[string]any)
	if !ok {
		if arrAny, ok := content.([]any); ok {
			for _, item := range arrAny {
				if m, ok := item.(map[string]any); ok {
					contentArr = append(contentArr, m)
				}
			}
		}
	}

	var result StepResultContent
	for _, c := range contentArr {
		partType, _ := c["type"].(string)
		switch partType {
		case "tool-result":
			toolCallID, _ := c["toolCallId"].(string)
			toolName, _ := c["toolName"].(string)
			result = append(result, map[string]any{
				"type":       "tool-result",
				"input":      FindToolCallArgs(dbMessages, toolCallID),
				"output":     c["output"],
				"toolCallId": toolCallID,
				"toolName":   toolName,
			})

		case "file":
			data := c["data"]
			mediaType, _ := c["mediaType"].(string)
			fileData := ""
			if dataStr, ok := data.(string); ok {
				parsed := prompt.ParseDataUri(dataStr)
				if parsed.IsDataUri {
					fileData = parsed.Base64Content
				} else {
					fileData = dataStr
				}
			}
			result = append(result, map[string]any{
				"type": "file",
				"file": map[string]any{
					"data":      fileData,
					"mediaType": mediaType,
				},
			})

		case "image":
			image := c["image"]
			mediaType, _ := c["mediaType"].(string)
			if mediaType == "" {
				mediaType = "unknown"
			}
			imageData := ""
			if imageStr, ok := image.(string); ok {
				parsed := prompt.ParseDataUri(imageStr)
				if parsed.IsDataUri {
					imageData = parsed.Base64Content
				} else {
					imageData = imageStr
				}
			}
			result = append(result, map[string]any{
				"type": "file",
				"file": map[string]any{
					"data":      imageData,
					"mediaType": mediaType,
				},
			})

		default:
			// Pass through as-is
			result = append(result, c)
		}
	}

	return result
}
