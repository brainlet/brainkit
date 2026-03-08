// Ported from: packages/anthropic/src/tool/computer_20241022.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// Computer20241022Input is the input schema for the computer_20241022 tool.
type Computer20241022Input struct {
	Action     string  `json:"action"`
	Coordinate []int   `json:"coordinate,omitempty"`
	Text       *string `json:"text,omitempty"`
}

// Computer20241022 is the provider tool factory for the computer_20241022 tool.
var Computer20241022 = providerutils.CreateProviderToolFactory(providerutils.ProviderToolConfig[Computer20241022Input]{
	ID: "anthropic.computer_20241022",
})
