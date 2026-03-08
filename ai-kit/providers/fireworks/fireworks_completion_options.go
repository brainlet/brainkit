// Ported from: packages/fireworks/src/fireworks-completion-options.ts
package fireworks

// FireworksCompletionModelID represents a Fireworks completion model identifier.
// Below is just a subset of the available models.
type FireworksCompletionModelID = string

const (
	FireworksCompletionModelLlama3_8bInstruct FireworksCompletionModelID = "accounts/fireworks/models/llama-v3-8b-instruct"
	FireworksCompletionModelLlama2_34bCode    FireworksCompletionModelID = "accounts/fireworks/models/llama-v2-34b-code"
)
