// Ported from: packages/core/src/agent/message-list/merge/MessageMerger.ts
package merge

import (
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/cache"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// MessageMerger handles complex logic for merging assistant messages.
// When streaming responses from LLMs, multiple messages may need merging:
// tool calls updated with results, text parts appended, step-start markers inserted.
type MessageMerger struct{}

// IsSealed checks if a message is sealed (should not be merged into).
// Messages are sealed after observation to preserve observation markers.
func IsSealed(message *state.MastraDBMessage) bool {
	if message.Content.Metadata == nil {
		return false
	}
	mastra, ok := message.Content.Metadata["mastra"]
	if !ok {
		return false
	}
	mastraMap, ok := mastra.(map[string]any)
	if !ok {
		return false
	}
	sealed, ok := mastraMap["sealed"].(bool)
	return ok && sealed
}

// ShouldMerge checks if an incoming message should be merged with the latest message.
func ShouldMerge(
	latestMessage *state.MastraDBMessage,
	incomingMessage *state.MastraDBMessage,
	messageSource string,
	isLatestFromMemory bool,
	agentNetworkAppend bool,
) bool {
	if latestMessage == nil {
		return false
	}

	if IsSealed(latestMessage) {
		return false
	}

	// Don't merge completion result messages
	if latestMessage.Content.Metadata != nil {
		if _, ok := latestMessage.Content.Metadata["completionResult"]; ok {
			return false
		}
		if _, ok := latestMessage.Content.Metadata["isTaskCompleteResult"]; ok {
			return false
		}
	}
	if incomingMessage.Content.Metadata != nil {
		if _, ok := incomingMessage.Content.Metadata["completionResult"]; ok {
			return false
		}
		if _, ok := incomingMessage.Content.Metadata["isTaskCompleteResult"]; ok {
			return false
		}
	}

	// Basic merge conditions
	shouldAppend := latestMessage.Role == "assistant" &&
		incomingMessage.Role == "assistant" &&
		latestMessage.ThreadID == incomingMessage.ThreadID &&
		messageSource != "memory"

	// Agent network append flag handling
	appendNetwork := true
	if agentNetworkAppend {
		appendNetwork = !isLatestFromMemory
	}

	return shouldAppend && appendNetwork
}

// Merge merges an incoming assistant message into the latest assistant message.
func Merge(latestMessage *state.MastraDBMessage, incomingMessage *state.MastraDBMessage) {
	// Update timestamp
	if !incomingMessage.CreatedAt.IsZero() {
		latestMessage.CreatedAt = incomingMessage.CreatedAt
	}

	// Build anchor map and parts to add
	toolResultAnchorMap := make(map[int]int) // incoming index -> latest index
	partsToAdd := make(map[int]state.MastraMessagePart)

	for i, part := range incomingMessage.Content.Parts {
		if part.Type == "tool-invocation" && part.ToolInvocation != nil {
			// Find corresponding call in latest message
			existingIdx := -1
			for j := len(latestMessage.Content.Parts) - 1; j >= 0; j-- {
				p := latestMessage.Content.Parts[j]
				if p.Type == "tool-invocation" && p.ToolInvocation != nil &&
					p.ToolInvocation.ToolCallID == part.ToolInvocation.ToolCallID {
					existingIdx = j
					break
				}
			}

			if existingIdx >= 0 {
				existingPart := &latestMessage.Content.Parts[existingIdx]
				if part.ToolInvocation.State == "result" {
					// Update existing tool-call with result
					existingPart.ToolInvocation.State = "result"
					existingPart.ToolInvocation.Result = part.ToolInvocation.Result
					existingPart.ToolInvocation.Step = part.ToolInvocation.Step
					// Merge args
					if existingPart.ToolInvocation.Args == nil {
						existingPart.ToolInvocation.Args = make(map[string]any)
					}
					for k, v := range part.ToolInvocation.Args {
						existingPart.ToolInvocation.Args[k] = v
					}

					// Preserve providerMetadata from result
					if part.ProviderMetadata != nil {
						if existingPart.ProviderMetadata == nil {
							existingPart.ProviderMetadata = make(state.ProviderMetadata)
						}
						for k, v := range part.ProviderMetadata {
							existingPart.ProviderMetadata[k] = v
						}
					}

					// Update toolInvocations array
					if latestMessage.Content.ToolInvocations == nil {
						latestMessage.Content.ToolInvocations = []state.ToolInvocation{}
					}
					found := false
					for j, ti := range latestMessage.Content.ToolInvocations {
						if ti.ToolCallID == existingPart.ToolInvocation.ToolCallID {
							latestMessage.Content.ToolInvocations[j] = *existingPart.ToolInvocation
							found = true
							break
						}
					}
					if !found {
						latestMessage.Content.ToolInvocations = append(latestMessage.Content.ToolInvocations, *existingPart.ToolInvocation)
					}
				}
				toolResultAnchorMap[i] = existingIdx
			} else {
				partsToAdd[i] = part
			}
		} else {
			partsToAdd[i] = part
		}
	}

	addPartsToMessage(latestMessage, incomingMessage, toolResultAnchorMap, partsToAdd)

	if latestMessage.CreatedAt.Before(incomingMessage.CreatedAt) {
		latestMessage.CreatedAt = incomingMessage.CreatedAt
	}
	if latestMessage.Content.Content == "" && incomingMessage.Content.Content != "" {
		latestMessage.Content.Content = incomingMessage.Content.Content
	}
	if latestMessage.Content.Content != "" && incomingMessage.Content.Content != "" &&
		latestMessage.Content.Content != incomingMessage.Content.Content {
		latestMessage.Content.Content = incomingMessage.Content.Content
	}
}

func addPartsToMessage(
	latestMessage *state.MastraDBMessage,
	incomingMessage *state.MastraDBMessage,
	anchorMap map[int]int,
	partsToAdd map[int]state.MastraMessagePart,
) {
	for i := 0; i < len(incomingMessage.Content.Parts); i++ {
		part := incomingMessage.Content.Parts[i]
		key := cache.FromDBParts([]state.MastraMessagePart{part})
		if _, isPart := partsToAdd[i]; !isPart || key == "" {
			continue
		}

		if len(anchorMap) > 0 {
			if _, isAnchor := anchorMap[i]; isAnchor {
				continue
			}

			// Find left anchor
			leftAnchorV2 := -1
			for idx := range anchorMap {
				if idx < i && idx > leftAnchorV2 {
					leftAnchorV2 = idx
				}
			}

			// Find right anchor
			rightAnchorV2 := -1
			for idx := range anchorMap {
				if idx > i && (rightAnchorV2 == -1 || idx < rightAnchorV2) {
					rightAnchorV2 = idx
				}
			}

			leftAnchorLatest := 0
			if leftAnchorV2 != -1 {
				leftAnchorLatest = anchorMap[leftAnchorV2]
			}

			offset := i
			if leftAnchorV2 != -1 {
				offset = i - leftAnchorV2
			}

			insertAt := leftAnchorLatest + offset
			rightAnchorLatest := len(latestMessage.Content.Parts)
			if rightAnchorV2 != -1 {
				rightAnchorLatest = anchorMap[rightAnchorV2]
			}

			if insertAt >= 0 && insertAt <= rightAnchorLatest {
				// Check for duplicates in the range
				hasDup := false
				for _, p := range latestMessage.Content.Parts[insertAt:rightAnchorLatest] {
					if cache.FromDBParts([]state.MastraMessagePart{p}) == key {
						hasDup = true
						break
					}
				}
				if !hasDup {
					pushNewPart(latestMessage, incomingMessage, part, &insertAt)
					// Shift anchors
					for v2Idx, latestIdx := range anchorMap {
						if latestIdx >= insertAt {
							anchorMap[v2Idx] = latestIdx + 1
						}
					}
				}
			}
		} else {
			pushNewPart(latestMessage, incomingMessage, part, nil)
		}
	}
}

func pushNewPart(
	latestMessage *state.MastraDBMessage,
	newMessage *state.MastraDBMessage,
	part state.MastraMessagePart,
	insertAt *int,
) {
	partKey := cache.FromDBParts([]state.MastraMessagePart{part})

	latestCount := 0
	for _, p := range latestMessage.Content.Parts {
		if cache.FromDBParts([]state.MastraMessagePart{p}) == partKey {
			latestCount++
		}
	}

	newCount := 0
	for _, p := range newMessage.Content.Parts {
		if cache.FromDBParts([]state.MastraMessagePart{p}) == partKey {
			newCount++
		}
	}

	if latestCount < newCount {
		// Check if step-start is needed
		partIndex := -1
		for idx, p := range newMessage.Content.Parts {
			if cache.FromDBParts([]state.MastraMessagePart{p}) == partKey {
				partIndex = idx
				break
			}
		}

		hasStepStartBefore := partIndex > 0 && newMessage.Content.Parts[partIndex-1].Type == "step-start"

		needsStepStart := latestMessage.Role == "assistant" &&
			part.Type == "text" &&
			!hasStepStartBefore &&
			len(latestMessage.Content.Parts) > 0 &&
			latestMessage.Content.Parts[len(latestMessage.Content.Parts)-1].Type == "tool-invocation"

		stepStartPart := state.MastraMessagePart{Type: "step-start"}

		if insertAt != nil {
			at := *insertAt
			if needsStepStart {
				latestMessage.Content.Parts = insertPartAt(latestMessage.Content.Parts, at, stepStartPart)
				latestMessage.Content.Parts = insertPartAt(latestMessage.Content.Parts, at+1, part)
			} else {
				latestMessage.Content.Parts = insertPartAt(latestMessage.Content.Parts, at, part)
			}
		} else {
			if needsStepStart {
				latestMessage.Content.Parts = append(latestMessage.Content.Parts, stepStartPart)
			}
			latestMessage.Content.Parts = append(latestMessage.Content.Parts, part)
		}
	}
}

func insertPartAt(parts []state.MastraMessagePart, index int, part state.MastraMessagePart) []state.MastraMessagePart {
	if index >= len(parts) {
		return append(parts, part)
	}
	parts = append(parts, state.MastraMessagePart{})
	copy(parts[index+1:], parts[index:])
	parts[index] = part
	return parts
}
