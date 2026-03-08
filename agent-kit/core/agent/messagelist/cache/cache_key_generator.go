// Ported from: packages/core/src/agent/message-list/cache/CacheKeyGenerator.ts
package cache

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/prompt"
)

// CacheKeyGenerator provides consistent cache key generation for message equality checks.
// This is critical for deduplication, detecting updates, and comparing messages across formats.
type CacheKeyGenerator struct{}

// FromAIV4Parts generates a cache key from AIV4 UIMessage parts.
func FromAIV4Parts(parts []MastraMessagePart) string {
	var key strings.Builder
	for _, part := range parts {
		key.WriteString(part.Type)
		key.WriteString(FromAIV4Part(part))
	}
	return key.String()
}

// FromAIV4Part generates a cache key from a single AIV4 UIMessage part.
func FromAIV4Part(part MastraMessagePart) string {
	var cacheKey strings.Builder

	switch part.Type {
	case "text":
		cacheKey.WriteString(part.Text)
	case "tool-invocation":
		if part.ToolInvocation != nil {
			cacheKey.WriteString(part.ToolInvocation.ToolCallID)
			cacheKey.WriteString(part.ToolInvocation.State)
		}
	case "reasoning":
		cacheKey.WriteString(part.Reasoning)
		detailLen := 0
		for _, detail := range part.Details {
			if detail.Type == "text" {
				detailLen += len(detail.Text) + len(detail.Signature)
			}
		}
		cacheKey.WriteString(fmt.Sprintf("%d", detailLen))

		// Include OpenAI reasoning itemId for proper deduplication
		if part.ProviderMetadata != nil {
			if openai, ok := part.ProviderMetadata["openai"]; ok {
				if itemID, ok := openai["itemId"].(string); ok {
					cacheKey.WriteString("|")
					cacheKey.WriteString(itemID)
				}
			}
		}
	case "file":
		cacheKey.WriteString(part.Data)
		cacheKey.WriteString(part.MimeType)
	}

	return cacheKey.String()
}

// FromDBParts generates a cache key from MastraDB message parts.
func FromDBParts(parts []MastraMessagePart) string {
	var key strings.Builder
	for _, part := range parts {
		key.WriteString(part.Type)
		if strings.HasPrefix(part.Type, "data-") {
			// Stringify data for proper cache key comparison
			data, _ := json.Marshal(part.DataPayload)
			key.Write(data)
		} else {
			key.WriteString(FromAIV4Part(part))
		}
	}
	return key.String()
}

// FromAIV4CoreMessageContent generates a cache key from AIV4 CoreMessage content.
// content can be a string or []CoreMessageContentPart (represented as []map[string]any).
func FromAIV4CoreMessageContent(content any) string {
	if s, ok := content.(string); ok {
		return s
	}

	parts, ok := content.([]map[string]any)
	if !ok {
		return fmt.Sprintf("%v", content)
	}

	var key strings.Builder
	for _, part := range parts {
		partType, _ := part["type"].(string)
		key.WriteString(partType)

		switch partType {
		case "text":
			if text, ok := part["text"].(string); ok {
				key.WriteString(fmt.Sprintf("%d", len(text)))
			}
		case "reasoning":
			if text, ok := part["text"].(string); ok {
				key.WriteString(fmt.Sprintf("%d", len(text)))
			}
		case "tool-call":
			if id, ok := part["toolCallId"].(string); ok {
				key.WriteString(id)
			}
			if name, ok := part["toolName"].(string); ok {
				key.WriteString(name)
			}
		case "tool-result":
			if id, ok := part["toolCallId"].(string); ok {
				key.WriteString(id)
			}
			if name, ok := part["toolName"].(string); ok {
				key.WriteString(name)
			}
		case "file":
			if filename, ok := part["filename"].(string); ok {
				key.WriteString(filename)
			}
			if mimeType, ok := part["mimeType"].(string); ok {
				key.WriteString(mimeType)
			}
		case "image":
			if image, ok := part["image"]; ok {
				cacheKey := prompt.GetImageCacheKey(image)
				key.WriteString(fmt.Sprintf("%v", cacheKey))
			}
			if mimeType, ok := part["mimeType"].(string); ok {
				key.WriteString(mimeType)
			}
		case "redacted-reasoning":
			if data, ok := part["data"].(string); ok {
				key.WriteString(fmt.Sprintf("%d", len(data)))
			}
		}
	}
	return key.String()
}

// FromAIV5Parts generates a cache key from AIV5 UIMessage parts.
func FromAIV5Parts(parts []map[string]any) string {
	var key strings.Builder
	for _, part := range parts {
		partType, _ := part["type"].(string)
		key.WriteString(partType)

		switch {
		case partType == "text":
			if text, ok := part["text"].(string); ok {
				key.WriteString(text)
			}
		case strings.HasPrefix(partType, "tool-") || partType == "dynamic-tool":
			if id, ok := part["toolCallId"].(string); ok {
				key.WriteString(id)
			}
			if st, ok := part["state"].(string); ok {
				key.WriteString(st)
			}
		case partType == "reasoning":
			if text, ok := part["text"].(string); ok {
				key.WriteString(text)
			}
		case partType == "file":
			if u, ok := part["url"].(string); ok {
				key.WriteString(fmt.Sprintf("%d", len(u)))
			}
			if mt, ok := part["mediaType"].(string); ok {
				key.WriteString(mt)
			}
			if fn, ok := part["filename"].(string); ok {
				key.WriteString(fn)
			}
		}
	}
	return key.String()
}

// FromAIV5ModelMessageContent generates a cache key from AIV5 ModelMessage content.
func FromAIV5ModelMessageContent(content any) string {
	if s, ok := content.(string); ok {
		return s
	}

	parts, ok := content.([]map[string]any)
	if !ok {
		return fmt.Sprintf("%v", content)
	}

	var key strings.Builder
	for _, part := range parts {
		partType, _ := part["type"].(string)
		key.WriteString(partType)

		switch partType {
		case "text":
			if text, ok := part["text"].(string); ok {
				key.WriteString(fmt.Sprintf("%d", len(text)))
			}
		case "reasoning":
			if text, ok := part["text"].(string); ok {
				key.WriteString(fmt.Sprintf("%d", len(text)))
			}
		case "tool-call":
			if id, ok := part["toolCallId"].(string); ok {
				key.WriteString(id)
			}
			if name, ok := part["toolName"].(string); ok {
				key.WriteString(name)
			}
		case "tool-result":
			if id, ok := part["toolCallId"].(string); ok {
				key.WriteString(id)
			}
			if name, ok := part["toolName"].(string); ok {
				key.WriteString(name)
			}
		case "file":
			if filename, ok := part["filename"].(string); ok {
				key.WriteString(filename)
			}
			if mt, ok := part["mediaType"].(string); ok {
				key.WriteString(mt)
			}
		case "image":
			if image, ok := part["image"]; ok {
				cacheKey := prompt.GetImageCacheKey(image)
				key.WriteString(fmt.Sprintf("%v", cacheKey))
			}
			if mt, ok := part["mediaType"].(string); ok {
				key.WriteString(mt)
			}
		}
	}
	return key.String()
}
