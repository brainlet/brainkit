// Ported from: packages/cerebras/src/cerebras-chat-options.ts
package cerebras

// CerebrasChatModelID represents Cerebras model identifiers.
// https://inference-docs.cerebras.ai/models/overview
//
// Known values:
//
//	Production:
//	  - "llama3.1-8b"
//	  - "gpt-oss-120b"
//	Preview:
//	  - "qwen-3-235b-a22b-instruct-2507"
//	  - "qwen-3-235b-a22b-thinking-2507"
//	  - "zai-glm-4.6"
//	  - "zai-glm-4.7"
//
// Any string is accepted to allow new models.
type CerebrasChatModelID = string
