// Ported from: packages/anthropic/src/tool/bash_20241022.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// Bash20241022Input is the input schema for the bash_20241022 tool.
type Bash20241022Input struct {
	Command string `json:"command"`
	Restart *bool  `json:"restart,omitempty"`
}

// Bash20241022 is the provider tool factory for the bash_20241022 tool.
var Bash20241022 = providerutils.CreateProviderToolFactory(providerutils.ProviderToolConfig[Bash20241022Input]{
	ID: "anthropic.bash_20241022",
})
