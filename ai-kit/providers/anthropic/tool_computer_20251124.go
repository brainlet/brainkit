// Ported from: packages/anthropic/src/tool/computer_20251124.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// Computer20251124Input is the input schema for the computer_20251124 tool.
type Computer20251124Input struct {
	Action          string   `json:"action"`
	Coordinate      *[2]int  `json:"coordinate,omitempty"`
	Duration        *float64 `json:"duration,omitempty"`
	Region          *[4]int  `json:"region,omitempty"`
	ScrollAmount    *float64 `json:"scroll_amount,omitempty"`
	ScrollDirection *string  `json:"scroll_direction,omitempty"`
	StartCoordinate *[2]int  `json:"start_coordinate,omitempty"`
	Text            *string  `json:"text,omitempty"`
}

// Computer20251124 is the provider tool factory for the computer_20251124 tool.
var Computer20251124 = providerutils.CreateProviderToolFactory(providerutils.ProviderToolConfig[Computer20251124Input]{
	ID: "anthropic.computer_20251124",
})
