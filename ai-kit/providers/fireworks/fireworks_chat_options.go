// Ported from: packages/fireworks/src/fireworks-chat-options.ts
package fireworks

// FireworksChatModelID represents a Fireworks chat model identifier.
// https://docs.fireworks.ai/docs/serverless-models#chat-models
// Below is just a subset of the available models.
type FireworksChatModelID = string

const (
	FireworksChatModelDeepseekV3                FireworksChatModelID = "accounts/fireworks/models/deepseek-v3"
	FireworksChatModelLlama3p3_70bInstruct      FireworksChatModelID = "accounts/fireworks/models/llama-v3p3-70b-instruct"
	FireworksChatModelLlama3p2_3bInstruct       FireworksChatModelID = "accounts/fireworks/models/llama-v3p2-3b-instruct"
	FireworksChatModelLlama3p1_405bInstruct     FireworksChatModelID = "accounts/fireworks/models/llama-v3p1-405b-instruct"
	FireworksChatModelLlama3p1_8bInstruct       FireworksChatModelID = "accounts/fireworks/models/llama-v3p1-8b-instruct"
	FireworksChatModelMixtral8x7bInstruct       FireworksChatModelID = "accounts/fireworks/models/mixtral-8x7b-instruct"
	FireworksChatModelMixtral8x22bInstruct      FireworksChatModelID = "accounts/fireworks/models/mixtral-8x22b-instruct"
	FireworksChatModelMixtral8x7bInstructHf     FireworksChatModelID = "accounts/fireworks/models/mixtral-8x7b-instruct-hf"
	FireworksChatModelQwen2p5Coder32bInstruct   FireworksChatModelID = "accounts/fireworks/models/qwen2p5-coder-32b-instruct"
	FireworksChatModelQwen2p5_72bInstruct       FireworksChatModelID = "accounts/fireworks/models/qwen2p5-72b-instruct"
	FireworksChatModelQwenQwq32bPreview         FireworksChatModelID = "accounts/fireworks/models/qwen-qwq-32b-preview"
	FireworksChatModelQwen2Vl72bInstruct        FireworksChatModelID = "accounts/fireworks/models/qwen2-vl-72b-instruct"
	FireworksChatModelLlama3p2_11bVisionInstruct FireworksChatModelID = "accounts/fireworks/models/llama-v3p2-11b-vision-instruct"
	FireworksChatModelQwq32b                    FireworksChatModelID = "accounts/fireworks/models/qwq-32b"
	FireworksChatModelYiLarge                   FireworksChatModelID = "accounts/fireworks/models/yi-large"
	FireworksChatModelKimiK2Instruct            FireworksChatModelID = "accounts/fireworks/models/kimi-k2-instruct"
	FireworksChatModelKimiK2Thinking            FireworksChatModelID = "accounts/fireworks/models/kimi-k2-thinking"
	FireworksChatModelKimiK2p5                  FireworksChatModelID = "accounts/fireworks/models/kimi-k2p5"
	FireworksChatModelMinimaxM2                 FireworksChatModelID = "accounts/fireworks/models/minimax-m2"
)

// FireworksThinking holds the thinking/reasoning configuration for Fireworks models.
type FireworksThinking struct {
	// Type enables or disables thinking. Optional; valid values: "enabled", "disabled".
	Type *string `json:"type,omitempty"`

	// BudgetTokens is the budget for thinking tokens. Must be >= 1024. Optional.
	BudgetTokens *int `json:"budgetTokens,omitempty"`
}

// FireworksLanguageModelOptions are the provider-specific options for Fireworks chat models.
type FireworksLanguageModelOptions struct {
	// Thinking configures thinking/reasoning behavior.
	Thinking *FireworksThinking `json:"thinking,omitempty"`

	// ReasoningHistory controls how reasoning history is handled.
	// Valid values: "disabled", "interleaved", "preserved".
	ReasoningHistory *string `json:"reasoningHistory,omitempty"`
}
