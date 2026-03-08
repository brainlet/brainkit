// Ported from: packages/groq/src/groq-tools.ts
package groq

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// GroqTools holds the tools provided by Groq.
var GroqTools = struct {
	BrowserSearch func(opts providerutils.ProviderToolOptions[struct{}, interface{}]) providerutils.ProviderTool[struct{}, interface{}]
}{
	BrowserSearch: BrowserSearch,
}
