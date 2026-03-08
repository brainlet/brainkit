// Ported from: packages/ai/src/generate-text/response-message.ts
package generatetext

// ResponseMessage is a message generated during the generation process.
// It can be either an assistant message or a tool message.
// In Go, we use ModelMessage directly since both are the same underlying type.
type ResponseMessage = ModelMessage
