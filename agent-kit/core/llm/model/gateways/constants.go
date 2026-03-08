// Ported from: packages/core/src/llm/model/gateways/constants.ts
package gateways

// ProvidersWithInstalledPackages lists providers that use their corresponding
// AI SDK package instead of using openai-compat endpoints.
var ProvidersWithInstalledPackages = []string{
	"anthropic",
	"cerebras",
	"deepinfra",
	"deepseek",
	"google",
	"groq",
	"mistral",
	"openai",
	"openrouter",
	"perplexity",
	"togetherai",
	"vercel",
	"xai",
}

// ExcludedProviders lists providers that don't show up in model router.
// For now that's just copilot which requires a special oauth flow.
var ExcludedProviders = []string{
	"github-copilot",
}

// isProviderWithInstalledPackage checks whether the given provider ID is in
// the installed packages list.
func isProviderWithInstalledPackage(providerID string) bool {
	for _, p := range ProvidersWithInstalledPackages {
		if p == providerID {
			return true
		}
	}
	return false
}

// isExcludedProvider checks whether the given provider ID is excluded.
func isExcludedProvider(providerID string) bool {
	for _, p := range ExcludedProviders {
		if p == providerID {
			return true
		}
	}
	return false
}
