// Ported from: packages/ai/src/types/speech-model.ts
package aitypes

// SpeechModel is the speech model used by the AI SDK.
//
// In TypeScript this is a union: string | SpeechModelV4 | SpeechModelV3 | SpeechModelV2.
// In Go, we represent this as an interface that can hold either a string model ID
// or a model interface implementation.
type SpeechModel = any
