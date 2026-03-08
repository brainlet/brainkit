// Ported from: packages/groq/src/tool/browser-search.ts
package groq

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// BrowserSearch is the browser search tool factory for Groq models.
//
// Provides interactive browser search capabilities that go beyond traditional web search
// by navigating websites interactively and providing more detailed results.
//
// Currently supported on:
//   - openai/gpt-oss-20b
//   - openai/gpt-oss-120b
//
// See: https://console.groq.com/docs/browser-search
var BrowserSearch = providerutils.CreateProviderToolFactory(providerutils.ProviderToolConfig[struct{}]{
	ID:          "groq.browser_search",
	InputSchema: &providerutils.Schema[struct{}]{},
})
