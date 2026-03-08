// Ported from: packages/ai/src/types/transcription-model.ts
package aitypes

// TranscriptionModel is the transcription model used by the AI SDK.
//
// In TypeScript this is a union: string | TranscriptionModelV4 | TranscriptionModelV3 | TranscriptionModelV2.
// In Go, we represent this as an interface that can hold either a string model ID
// or a model interface implementation.
type TranscriptionModel = any
