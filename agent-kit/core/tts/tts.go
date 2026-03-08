// Ported from: packages/core/src/tts/index.ts
package tts

import (
	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// BuiltInModelConfig holds provider/model name configuration for a TTS model.
type BuiltInModelConfig struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
	APIKey   string `json:"apiKey,omitempty"`
}

// TTSConfig holds the configuration for constructing a MastraTTS.
type TTSConfig struct {
	Model BuiltInModelConfig
}

// MastraTTS is the abstract base type for text-to-speech implementations.
// Concrete implementations must embed this struct and implement the
// Generate and Stream methods.
type MastraTTS struct {
	*agentkit.MastraBase
	Model BuiltInModelConfig
}

// NewMastraTTS creates a new MastraTTS with the given configuration.
func NewMastraTTS(cfg TTSConfig) *MastraTTS {
	return &MastraTTS{
		MastraBase: agentkit.NewMastraBase(agentkit.MastraBaseOptions{
			Component: logger.RegisteredLogger("TTS"),
		}),
		Model: cfg.Model,
	}
}

// TTSProvider defines the interface that concrete TTS implementations must satisfy.
// This corresponds to the abstract methods on the TypeScript MastraTTS class.
type TTSProvider interface {
	// Generate produces audio from text in a single shot.
	Generate(input GenerateInput) (any, error)

	// Stream produces a streaming audio response from text.
	Stream(input StreamInput) (any, error)
}

// GenerateInput holds the input parameters for the Generate method.
type GenerateInput struct {
	Text string `json:"text"`
}

// StreamInput holds the input parameters for the Stream method.
type StreamInput struct {
	Text string `json:"text"`
}
