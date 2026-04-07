// compat.go — Backward compatibility aliases.
// These exist so existing code (tests, CLI, testutil) continues to compile
// during the migration to Kit. New code should use brainkit.New() + sdk.Publish.
package brainkit

import (
	"github.com/brainlet/brainkit/internal/engine"
	"github.com/brainlet/brainkit/internal/types"
)

// Kernel is the internal runtime type. Use Kit for new code.
type Kernel = engine.Kernel

// Node is the transport-connected runtime type. Use Kit for new code.
type Node = engine.Node

// KernelConfig is the internal config type. Use Config for new code.
type KernelConfig = types.KernelConfig

// NodeConfig is the internal node config type. Use Config for new code.
type NodeConfig = types.NodeConfig

// MessagingConfig is the internal messaging config. Transport fields are on Config directly.
type MessagingConfig = types.MessagingConfig

// NewKernel creates a standalone runtime. Use New() for new code.
func NewKernel(cfg KernelConfig) (*Kernel, error) {
	return engine.NewKernel(cfg)
}

// NewNode creates a transport-connected runtime. Use New() for new code.
func NewNode(cfg NodeConfig) (*Node, error) {
	return engine.NewNode(cfg)
}

// AIProviderRegistration is the internal provider type. Use ProviderConfig constructors.
type AIProviderRegistration = types.AIProviderRegistration

// AIProviderType is the internal provider type constant.
type AIProviderType = types.AIProviderType

const (
	AIProviderOpenAI      = types.AIProviderOpenAI
	AIProviderAnthropic   = types.AIProviderAnthropic
	AIProviderGoogle      = types.AIProviderGoogle
	AIProviderMistral     = types.AIProviderMistral
	AIProviderCohere      = types.AIProviderCohere
	AIProviderGroq        = types.AIProviderGroq
	AIProviderPerplexity  = types.AIProviderPerplexity
	AIProviderDeepSeek    = types.AIProviderDeepSeek
	AIProviderFireworks   = types.AIProviderFireworks
	AIProviderTogetherAI  = types.AIProviderTogetherAI
	AIProviderXAI         = types.AIProviderXAI
	AIProviderAzure       = types.AIProviderAzure
	AIProviderBedrock     = types.AIProviderBedrock
	AIProviderVertex      = types.AIProviderVertex
	AIProviderHuggingFace = types.AIProviderHuggingFace
	AIProviderCerebras    = types.AIProviderCerebras
)

// Provider config structs — use OpenAI(key), Anthropic(key) constructors instead.
type OpenAIProviderConfig = types.OpenAIProviderConfig
type AnthropicProviderConfig = types.AnthropicProviderConfig
type GoogleProviderConfig = types.GoogleProviderConfig
type MistralProviderConfig = types.MistralProviderConfig
type CohereProviderConfig = types.CohereProviderConfig
type GroqProviderConfig = types.GroqProviderConfig
type PerplexityProviderConfig = types.PerplexityProviderConfig
type DeepSeekProviderConfig = types.DeepSeekProviderConfig
type FireworksProviderConfig = types.FireworksProviderConfig
type TogetherAIProviderConfig = types.TogetherAIProviderConfig
type XAIProviderConfig = types.XAIProviderConfig
type CerebrasProviderConfig = types.CerebrasProviderConfig
