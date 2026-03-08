// Ported from: packages/core/src/stream/aisdk/v5/
// This is a stub file for the AI SDK v5 stream integration.
// Files to port:
//   - input.ts       → AISDKV5InputStream (extends MastraModelInput)
//   - transform.ts   → convertFullStreamChunkToMastra, convertMastraChunkToAISDKv5
//   - execute.ts     → execution helpers
//   - file.ts        → DefaultGeneratedFile, DefaultGeneratedFileWithType
//   - output-helpers.ts → output format helpers
//
// Subdirectory compat/ files to port:
//   - consume-stream.ts → consumeStream (with logger support)
//   - content.ts        → content conversion helpers
//   - delayed-promise.ts → DelayedPromise (already ported in stream/run_output.go)
//   - media.ts          → media handling helpers
//   - prepare-tools.ts  → tool preparation helpers
//   - ui-message.ts     → UI message conversion (convertFullStreamChunkToUIMessageStream)
//   - validation.ts     → validation helpers
package v5

// AISDKV5InputStream is partially ported in input.go. Remaining parts
// (full V2 stream part conversion) depend on MastraModelInput base class
// and LanguageModelV2StreamPart provider types from AI SDK v5.

// convertFullStreamChunkToMastra is partially ported in transform.go.
// Maps V2 LanguageModelV2StreamPart events to Mastra ChunkType events.

// convertMastraChunkToAISDKv5 (reverse conversion) is not yet ported.
// Used by MCP and A2A stream adapters for outbound stream conversion.

// DefaultGeneratedFile and DefaultGeneratedFileWithType handle
// file streaming responses. Not yet ported (low priority — file streaming
// is rare in agent-kit use cases).

// compat/ utilities (content conversion, UI message streams,
// tool preparation, validation) are partially ported in compat/ subdirectory.
