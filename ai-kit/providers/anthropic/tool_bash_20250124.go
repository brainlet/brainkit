// Ported from: packages/anthropic/src/tool/bash_20250124.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// Bash20250124Input is the input schema for the bash_20250124 tool.
type Bash20250124Input struct {
	Command string `json:"command"`
	Restart *bool  `json:"restart,omitempty"`
}

// Bash20250124 is the provider tool factory for the bash_20250124 tool.
var Bash20250124 = providerutils.CreateProviderToolFactory(providerutils.ProviderToolConfig[Bash20250124Input]{
	ID: "anthropic.bash_20250124",
})
