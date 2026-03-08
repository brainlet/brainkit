// Ported from: packages/core/src/stream/aisdk/v5/input.ts
package v5

import (
	"regexp"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/stream"
	"github.com/brainlet/brainkit/agent-kit/core/stream/base"
)

// IdGenerator is a function that generates unique IDs.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type IdGenerator func() string

// ---------------------------------------------------------------------------
// isNumericId
// ---------------------------------------------------------------------------

// numericIDRegex matches simple numeric strings (e.g., "0", "1", "2").
var numericIDRegex = regexp.MustCompile(`^\d+$`)

// isNumericID checks if an ID is a simple numeric string.
// Anthropic and Google providers use these indices which reset per LLM call,
// while OpenAI uses UUIDs that are already unique.
func isNumericID(id string) bool {
	return numericIDRegex.MatchString(id)
}

// ---------------------------------------------------------------------------
// defaultGenerateID
// ---------------------------------------------------------------------------

// defaultCounter provides a simple incrementing ID generator for fallback.
var defaultCounter int

// defaultGenerateID is a simple fallback ID generator.
// Simple fallback; callers can inject a custom IdGenerator for real UUID/nanoid.
func defaultGenerateID() string {
	defaultCounter++
	return "id-" + string(rune('0'+defaultCounter%10))
}

// ---------------------------------------------------------------------------
// AISDKV5InputStream
// ---------------------------------------------------------------------------

// AISDKV5InputStream is the AI SDK v5 input stream transformer.
// It reads LanguageModelV2StreamPart chunks from the provider stream,
// converts them to Mastra ChunkType chunks, and enqueues them to the output.
//
// In TS this extends MastraModelInput (which extends MastraBase).
// In Go it implements the base.MastraModelInput interface.
//
// Key logic:
//   - Numeric ID detection for Anthropic/Google providers
//   - Unique ID generation for tool calls to avoid collisions across steps
type AISDKV5InputStream struct {
	Component  logger.RegisteredLogger
	Name       string
	generateID IdGenerator
}

// AISDKV5InputStreamOptions configures an AISDKV5InputStream.
type AISDKV5InputStreamOptions struct {
	Component  logger.RegisteredLogger
	Name       string
	GenerateID IdGenerator
}

// NewAISDKV5InputStream creates a new AISDKV5InputStream.
func NewAISDKV5InputStream(opts AISDKV5InputStreamOptions) *AISDKV5InputStream {
	genID := opts.GenerateID
	if genID == nil {
		genID = defaultGenerateID
	}
	return &AISDKV5InputStream{
		Component:  opts.Component,
		Name:       opts.Name,
		generateID: genID,
	}
}

// Transform reads from the raw provider stream and converts each
// LanguageModelV2StreamPart to a Mastra ChunkType, enqueuing it
// to the controller channel.
//
// This mirrors the TS async transform() method which iterates the
// ReadableStream<LanguageModelV2StreamPart> and calls
// convertFullStreamChunkToMastra for each chunk.
//
// It maps numeric IDs to unique IDs for uniqueness across steps.
// Workaround for @ai-sdk/anthropic and @ai-sdk/google duplicate IDs bug:
// These providers use numeric indices ("0", "1", etc.) that reset per LLM call.
// See: https://github.com/mastra-ai/mastra/issues/9909
func (s *AISDKV5InputStream) Transform(params base.TransformParams) error {
	// Map numeric IDs to unique IDs for uniqueness across steps.
	idMap := make(map[string]string)

	for chunk := range params.Stream {
		rawChunk := StreamPartFromRaw(chunk)

		// Clear ID map on new step so each step gets fresh UUIDs
		if rawChunk.Type == "stream-start" {
			idMap = make(map[string]string)
		}

		transformedChunk := ConvertFullStreamChunkToMastra(rawChunk, TransformContext{RunID: params.RunID})

		if transformedChunk != nil {
			// Replace numeric IDs with unique IDs for text chunks
			if (transformedChunk.Type == "text-start" ||
				transformedChunk.Type == "text-delta" ||
				transformedChunk.Type == "text-end") &&
				hasPayloadID(transformedChunk) {

				payloadID := getPayloadID(transformedChunk)
				if isNumericID(payloadID) {
					if _, exists := idMap[payloadID]; !exists {
						idMap[payloadID] = s.generateID()
					}
					setPayloadID(transformedChunk, idMap[payloadID])
				}
			}

			base.SafeEnqueue(params.Controller, *transformedChunk)
		}
	}

	return nil
}

// Initialize wires up the stream creation pipeline for the AISDKV5InputStream.
// This is a convenience method that delegates to base.Initialize.
func (s *AISDKV5InputStream) Initialize(params InitializeParams) <-chan stream.ChunkType {
	return base.Initialize(s, params.RunID, params.CreateStream, params.OnResult)
}

// InitializeParams are the parameters for AISDKV5InputStream.Initialize.
type InitializeParams struct {
	RunID        string
	CreateStream stream.CreateStream
	OnResult     stream.OnResult
}

// ---------------------------------------------------------------------------
// Payload ID helpers
// ---------------------------------------------------------------------------

func hasPayloadID(chunk *stream.ChunkType) bool {
	if chunk.Payload == nil {
		return false
	}
	if p, ok := chunk.Payload.(map[string]any); ok {
		_, ok := p["id"].(string)
		return ok
	}
	return false
}

func getPayloadID(chunk *stream.ChunkType) string {
	if p, ok := chunk.Payload.(map[string]any); ok {
		if id, ok := p["id"].(string); ok {
			return id
		}
	}
	return ""
}

func setPayloadID(chunk *stream.ChunkType, id string) {
	if p, ok := chunk.Payload.(map[string]any); ok {
		p["id"] = id
	}
}
