// Ported from: packages/core/src/agent/message-list/conversion/utils.ts
package conversion

import (
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/cache"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/detection"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// coreContentToString converts CoreMessage content to a plain string.
// Extracts text from text parts and concatenates them.
func CoreContentToString(content any) string {
	if s, ok := content.(string); ok {
		return s
	}

	parts, ok := content.([]any)
	if !ok {
		return ""
	}

	result := ""
	for _, partRaw := range parts {
		part, ok := partRaw.(map[string]any)
		if !ok {
			continue
		}
		if partType, _ := part["type"].(string); partType == "text" {
			if text, ok := part["text"].(string); ok {
				result += text
			}
		}
	}
	return result
}

// MessagesAreEqual compares two messages for equality based on their content.
// Uses cache keys for efficient comparison across different message formats.
func MessagesAreEqual(one map[string]any, two map[string]any) bool {
	oneUIV4 := detection.IsAIV4UIMessage(one)
	twoUIV4 := detection.IsAIV4UIMessage(two)
	if oneUIV4 && !twoUIV4 {
		return false
	}
	if oneUIV4 && twoUIV4 {
		oneParts := extractMastraMessageParts(one)
		twoParts := extractMastraMessageParts(two)
		return cache.FromAIV4Parts(oneParts) == cache.FromAIV4Parts(twoParts)
	}

	oneCMV4 := detection.IsAIV4CoreMessage(one)
	twoCMV4 := detection.IsAIV4CoreMessage(two)
	if oneCMV4 && !twoCMV4 {
		return false
	}
	if oneCMV4 && twoCMV4 {
		return cache.FromAIV4CoreMessageContent(one["content"]) == cache.FromAIV4CoreMessageContent(two["content"])
	}

	oneMM1 := detection.IsMastraMessageV1(one)
	twoMM1 := detection.IsMastraMessageV1(two)
	if oneMM1 && !twoMM1 {
		return false
	}
	if oneMM1 && twoMM1 {
		oneID, _ := one["id"].(string)
		twoID, _ := two["id"].(string)
		return oneID == twoID &&
			cache.FromAIV4CoreMessageContent(one["content"]) == cache.FromAIV4CoreMessageContent(two["content"])
	}

	oneMM2 := detection.IsMastraDBMessage(one)
	twoMM2 := detection.IsMastraDBMessage(two)
	if oneMM2 && !twoMM2 {
		return false
	}
	if oneMM2 && twoMM2 {
		oneID, _ := one["id"].(string)
		twoID, _ := two["id"].(string)
		oneParts := extractDBParts(one)
		twoParts := extractDBParts(two)
		return oneID == twoID &&
			cache.FromDBParts(oneParts) == cache.FromDBParts(twoParts)
	}

	oneUIV5 := detection.IsAIV5UIMessage(one)
	twoUIV5 := detection.IsAIV5UIMessage(two)
	if oneUIV5 && !twoUIV5 {
		return false
	}
	if oneUIV5 && twoUIV5 {
		oneV5Parts := extractMapParts(one)
		twoV5Parts := extractMapParts(two)
		return cache.FromAIV5Parts(oneV5Parts) == cache.FromAIV5Parts(twoV5Parts)
	}

	oneCMV5 := detection.IsAIV5CoreMessage(one)
	twoCMV5 := detection.IsAIV5CoreMessage(two)
	if oneCMV5 && !twoCMV5 {
		return false
	}
	if oneCMV5 && twoCMV5 {
		return cache.FromAIV5ModelMessageContent(one["content"]) == cache.FromAIV5ModelMessageContent(two["content"])
	}

	// default to it did change. we'll likely never reach this codepath
	return true
}

// MessagesAreEqualDB compares two MastraDBMessages for equality.
func MessagesAreEqualDB(one *state.MastraDBMessage, two *state.MastraDBMessage) bool {
	if one == nil || two == nil {
		return one == two
	}
	return one.ID == two.ID &&
		cache.FromDBParts(one.Content.Parts) == cache.FromDBParts(two.Content.Parts)
}

// extractMastraMessageParts extracts MastraMessagePart slice from a message map.
func extractMastraMessageParts(msg map[string]any) []state.MastraMessagePart {
	partsRaw, ok := msg["parts"]
	if !ok {
		return nil
	}
	// In Go, parts stored on typed structs are already []state.MastraMessagePart
	if parts, ok := partsRaw.([]state.MastraMessagePart); ok {
		return parts
	}
	return nil
}

// extractDBParts extracts MastraDBMessage parts from a message map.
func extractDBParts(msg map[string]any) []state.MastraMessagePart {
	content, ok := msg["content"]
	if !ok {
		return nil
	}
	contentMap, ok := content.(map[string]any)
	if !ok {
		return nil
	}
	partsRaw, ok := contentMap["parts"]
	if !ok {
		return nil
	}
	if parts, ok := partsRaw.([]state.MastraMessagePart); ok {
		return parts
	}
	return nil
}

// extractMapParts extracts parts as []map[string]any from a message map.
func extractMapParts(msg map[string]any) []map[string]any {
	partsRaw, ok := msg["parts"]
	if !ok {
		return nil
	}
	partsArr, ok := partsRaw.([]any)
	if !ok {
		return nil
	}
	var result []map[string]any
	for _, p := range partsArr {
		if pm, ok := p.(map[string]any); ok {
			result = append(result, pm)
		}
	}
	return result
}
