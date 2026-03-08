// Ported from: packages/core/src/stream/aisdk/v4/
// This is a stub file for the AI SDK v4 stream integration.
// Files to port:
//   - input.ts   → AISDKV4InputStream (extends MastraModelInput)
//   - transform.ts → convertFullStreamChunkToMastra (v1 stream part → ChunkType)
//   - usage.ts   → usage accumulation helpers
package v4

// AISDKV4InputStream depends on MastraModelInput and LanguageModelV1StreamPart
// from the AI SDK v1/v4 provider layer, which is not ported (ai-kit targets v6).

// convertFullStreamChunkToMastra maps V1 LanguageModelV1StreamPart events to
// Mastra ChunkType events. Not ported: ai-kit targets AI SDK v6 (see aisdk/v5/).

// V4 usage helpers (promptTokens/completionTokens → inputTokens/outputTokens)
// are implemented in usage.go above (ConvertV4Usage).
