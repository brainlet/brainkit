// Ported from: packages/core/src/llm/model/aisdk/generate-to-stream.ts
package aisdk

import (
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Stream event types
// ---------------------------------------------------------------------------

// StreamEvent represents a single event emitted by CreateStreamFromGenerateResult.
// In TypeScript this is a plain object enqueued onto a ReadableStream controller.
// In Go we represent it as a struct that can be sent over a channel.
type StreamEvent struct {
	Type string `json:"type"`

	// Common fields
	ID               string `json:"id,omitempty"`
	Warnings         []any  `json:"warnings,omitempty"`
	ModelID          string `json:"modelId,omitempty"`
	Timestamp        any    `json:"timestamp,omitempty"`
	ToolName         string `json:"toolName,omitempty"`
	Delta            any    `json:"delta,omitempty"`
	ProviderMetadata any    `json:"providerMetadata,omitempty"`
	FinishReason     any    `json:"finishReason,omitempty"`
	Usage            any    `json:"usage,omitempty"`

	// Source fields
	SourceType string `json:"sourceType,omitempty"`
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
	MediaType  string `json:"mediaType,omitempty"`
	Filename   string `json:"filename,omitempty"`
	Data       any    `json:"data,omitempty"`
}

// ---------------------------------------------------------------------------
// GenerateResultContent represents a content item from a generate result.
// ---------------------------------------------------------------------------

// GenerateResultContent represents a content item from a generate result.
type GenerateResultContent struct {
	Type string `json:"type"`
	// Additional fields vary by content type; accessed by key.
	Extra map[string]any `json:"-"`
}

// GenerateResultResponse holds optional response metadata from a generate result.
type GenerateResultResponse struct {
	ID        string `json:"id,omitempty"`
	ModelID   string `json:"modelId,omitempty"`
	Timestamp any    `json:"timestamp,omitempty"`
}

// GenerateResult holds the result of a doGenerate call.
type GenerateResult struct {
	Warnings         []any                   `json:"warnings"`
	Response         *GenerateResultResponse `json:"response,omitempty"`
	Content          []map[string]any        `json:"content"`
	FinishReason     any                     `json:"finishReason"`
	Usage            any                     `json:"usage"`
	ProviderMetadata any                     `json:"providerMetadata,omitempty"`
}

// ---------------------------------------------------------------------------
// CreateStreamFromGenerateResult
// ---------------------------------------------------------------------------

// CreateStreamFromGenerateResult converts a doGenerate result to a channel of
// StreamEvents, matching the TypeScript ReadableStream pattern.
// This is shared between V5 and V6 model wrappers since the content/result
// structure is compatible.
func CreateStreamFromGenerateResult(result *GenerateResult) <-chan StreamEvent {
	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)

		// stream-start
		ch <- StreamEvent{
			Type:     "stream-start",
			Warnings: result.Warnings,
		}

		// response-metadata
		var respID, respModelID string
		var respTimestamp any
		if result.Response != nil {
			respID = result.Response.ID
			respModelID = result.Response.ModelID
			respTimestamp = result.Response.Timestamp
		}
		ch <- StreamEvent{
			Type:      "response-metadata",
			ID:        respID,
			ModelID:   respModelID,
			Timestamp: respTimestamp,
		}

		// Process each content message
		for _, message := range result.Content {
			msgType, _ := message["type"].(string)

			switch msgType {
			case "tool-call":
				toolCallID, _ := message["toolCallId"].(string)
				toolName, _ := message["toolName"].(string)
				input := message["input"]

				ch <- StreamEvent{
					Type:     "tool-input-start",
					ID:       toolCallID,
					ToolName: toolName,
				}
				ch <- StreamEvent{
					Type:  "tool-input-delta",
					ID:    toolCallID,
					Delta: input,
				}
				ch <- StreamEvent{
					Type: "tool-input-end",
					ID:   toolCallID,
				}
				// Emit the full tool-call event
				ch <- StreamEvent{
					Type:     "tool-call",
					ID:       toolCallID,
					ToolName: toolName,
					Delta:    input,
				}

			case "tool-result":
				ch <- StreamEvent{
					Type:  "tool-result",
					ID:    getStr(message, "id"),
					Delta: message,
				}

			case "text":
				text, _ := message["text"].(string)
				providerMeta := message["providerMetadata"]
				id := "msg_" + uuid.New().String()

				ch <- StreamEvent{
					Type:             "text-start",
					ID:               id,
					ProviderMetadata: providerMeta,
				}
				ch <- StreamEvent{
					Type:  "text-delta",
					ID:    id,
					Delta: text,
				}
				ch <- StreamEvent{
					Type: "text-end",
					ID:   id,
				}

			case "reasoning":
				text, _ := message["text"].(string)
				providerMeta := message["providerMetadata"]
				id := "reasoning_" + uuid.New().String()

				ch <- StreamEvent{
					Type:             "reasoning-start",
					ID:               id,
					ProviderMetadata: providerMeta,
				}
				ch <- StreamEvent{
					Type:             "reasoning-delta",
					ID:               id,
					Delta:            text,
					ProviderMetadata: providerMeta,
				}
				ch <- StreamEvent{
					Type:             "reasoning-end",
					ID:               id,
					ProviderMetadata: providerMeta,
				}

			case "file":
				ch <- StreamEvent{
					Type:      "file",
					MediaType: getStr(message, "mediaType"),
					Data:      message["data"],
				}

			case "source":
				sourceType, _ := message["sourceType"].(string)
				srcID := getStr(message, "id")
				providerMeta := message["providerMetadata"]

				if sourceType == "url" {
					ch <- StreamEvent{
						Type:             "source",
						ID:               srcID,
						SourceType:       "url",
						URL:              getStr(message, "url"),
						Title:            getStr(message, "title"),
						ProviderMetadata: providerMeta,
					}
				} else {
					ch <- StreamEvent{
						Type:             "source",
						ID:               srcID,
						SourceType:       "document",
						MediaType:        getStr(message, "mediaType"),
						Filename:         getStr(message, "filename"),
						Title:            getStr(message, "title"),
						ProviderMetadata: providerMeta,
					}
				}
			}
		}

		// finish
		ch <- StreamEvent{
			Type:             "finish",
			FinishReason:     result.FinishReason,
			Usage:            result.Usage,
			ProviderMetadata: result.ProviderMetadata,
		}
	}()

	return ch
}

// getStr safely extracts a string from a map.
func getStr(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}
