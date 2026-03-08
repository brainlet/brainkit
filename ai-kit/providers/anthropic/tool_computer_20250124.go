// Ported from: packages/anthropic/src/tool/computer_20250124.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// Computer20250124Input is the input schema for the computer_20250124 tool.
type Computer20250124Input struct {
	Action          string   `json:"action"`
	Coordinate      *[2]int  `json:"coordinate,omitempty"`
	Duration        *float64 `json:"duration,omitempty"`
	ScrollAmount    *float64 `json:"scroll_amount,omitempty"`
	ScrollDirection *string  `json:"scroll_direction,omitempty"`
	StartCoordinate *[2]int  `json:"start_coordinate,omitempty"`
	Text            *string  `json:"text,omitempty"`
}

// Computer20250124 is the provider tool factory for the computer_20250124 tool.
var Computer20250124 = providerutils.CreateProviderToolFactory(providerutils.ProviderToolConfig[Computer20250124Input]{
	ID: "anthropic.computer_20250124",
})
