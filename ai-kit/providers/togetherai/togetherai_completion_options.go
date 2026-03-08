// Ported from: packages/togetherai/src/togetherai-completion-options.ts
package togetherai

// https://docs.together.ai/docs/serverless-models#language-models

// TogetherAICompletionModelID is the type for Together AI completion model identifiers.
// Known model IDs are provided as constants; any string is accepted.
type TogetherAICompletionModelID = string

// Known Together AI completion model IDs.
const (
	CompletionModelLlama2_70bHF             TogetherAICompletionModelID = "meta-llama/Llama-2-70b-hf"
	CompletionModelMistral7BV01             TogetherAICompletionModelID = "mistralai/Mistral-7B-v0.1"
	CompletionModelMixtral8x7BV01           TogetherAICompletionModelID = "mistralai/Mixtral-8x7B-v0.1"
	CompletionModelLlamaGuard7b             TogetherAICompletionModelID = "Meta-Llama/Llama-Guard-7b"
	CompletionModelCodeLlama34bInstructHF   TogetherAICompletionModelID = "codellama/CodeLlama-34b-Instruct-hf"
	CompletionModelQwen25Coder32BInstruct   TogetherAICompletionModelID = "Qwen/Qwen2.5-Coder-32B-Instruct"
)
