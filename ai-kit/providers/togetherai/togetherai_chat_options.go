// Ported from: packages/togetherai/src/togetherai-chat-options.ts
package togetherai

// https://docs.together.ai/docs/serverless-models#chat-models

// TogetherAIChatModelID is the type for Together AI chat model identifiers.
// Known model IDs are provided as constants; any string is accepted.
type TogetherAIChatModelID = string

// Known Together AI chat model IDs.
const (
	ChatModelLlama3_3_70BInstructTurbo      TogetherAIChatModelID = "meta-llama/Llama-3.3-70B-Instruct-Turbo"
	ChatModelMetaLlama3_1_8BInstructTurbo   TogetherAIChatModelID = "meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo"
	ChatModelMetaLlama3_1_70BInstructTurbo  TogetherAIChatModelID = "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"
	ChatModelMetaLlama3_1_405BInstructTurbo TogetherAIChatModelID = "meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo"
	ChatModelMetaLlama3_8BInstructTurbo     TogetherAIChatModelID = "meta-llama/Meta-Llama-3-8B-Instruct-Turbo"
	ChatModelMetaLlama3_70BInstructTurbo    TogetherAIChatModelID = "meta-llama/Meta-Llama-3-70B-Instruct-Turbo"
	ChatModelLlama3_2_3BInstructTurbo       TogetherAIChatModelID = "meta-llama/Llama-3.2-3B-Instruct-Turbo"
	ChatModelMetaLlama3_8BInstructLite      TogetherAIChatModelID = "meta-llama/Meta-Llama-3-8B-Instruct-Lite"
	ChatModelMetaLlama3_70BInstructLite     TogetherAIChatModelID = "meta-llama/Meta-Llama-3-70B-Instruct-Lite"
	ChatModelLlama3_8bChatHF                TogetherAIChatModelID = "meta-llama/Llama-3-8b-chat-hf"
	ChatModelLlama3_70bChatHF               TogetherAIChatModelID = "meta-llama/Llama-3-70b-chat-hf"
	ChatModelNvidiaLlama3_1Nemotron70B      TogetherAIChatModelID = "nvidia/Llama-3.1-Nemotron-70B-Instruct-HF"
	ChatModelQwen25Coder32BInstruct         TogetherAIChatModelID = "Qwen/Qwen2.5-Coder-32B-Instruct"
	ChatModelQwQ32BPreview                  TogetherAIChatModelID = "Qwen/QwQ-32B-Preview"
	ChatModelWizardLM2_8x22B               TogetherAIChatModelID = "microsoft/WizardLM-2-8x22B"
	ChatModelGemma2_27bIT                   TogetherAIChatModelID = "google/gemma-2-27b-it"
	ChatModelGemma2_9bIT                    TogetherAIChatModelID = "google/gemma-2-9b-it"
	ChatModelDBRXInstruct                   TogetherAIChatModelID = "databricks/dbrx-instruct"
	ChatModelDeepSeekLLM67BChat             TogetherAIChatModelID = "deepseek-ai/deepseek-llm-67b-chat"
	ChatModelDeepSeekV3                     TogetherAIChatModelID = "deepseek-ai/DeepSeek-V3"
	ChatModelGemma2bIT                      TogetherAIChatModelID = "google/gemma-2b-it"
	ChatModelMythoMaxL2_13b                 TogetherAIChatModelID = "Gryphe/MythoMax-L2-13b"
	ChatModelLlama2_13bChatHF              TogetherAIChatModelID = "meta-llama/Llama-2-13b-chat-hf"
	ChatModelMistral7BInstructV01           TogetherAIChatModelID = "mistralai/Mistral-7B-Instruct-v0.1"
	ChatModelMistral7BInstructV02           TogetherAIChatModelID = "mistralai/Mistral-7B-Instruct-v0.2"
	ChatModelMistral7BInstructV03           TogetherAIChatModelID = "mistralai/Mistral-7B-Instruct-v0.3"
	ChatModelMixtral8x7BInstructV01         TogetherAIChatModelID = "mistralai/Mixtral-8x7B-Instruct-v0.1"
	ChatModelMixtral8x22BInstructV01        TogetherAIChatModelID = "mistralai/Mixtral-8x22B-Instruct-v0.1"
	ChatModelNousHermes2Mixtral8x7BDPO      TogetherAIChatModelID = "NousResearch/Nous-Hermes-2-Mixtral-8x7B-DPO"
	ChatModelQwen25_7BInstructTurbo         TogetherAIChatModelID = "Qwen/Qwen2.5-7B-Instruct-Turbo"
	ChatModelQwen25_72BInstructTurbo        TogetherAIChatModelID = "Qwen/Qwen2.5-72B-Instruct-Turbo"
	ChatModelQwen2_72BInstruct              TogetherAIChatModelID = "Qwen/Qwen2-72B-Instruct"
	ChatModelSolar10_7BInstructV1           TogetherAIChatModelID = "upstage/SOLAR-10.7B-Instruct-v1.0"
)
