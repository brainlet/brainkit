// Ported from: packages/provider/src/language-model/v3/language-model-v3-provider-tool.ts
package languagemodel

// ProviderTool is the configuration of a provider tool.
//
// Provider tools are tools that are specific to a certain provider.
// The input and output schemas are defined by the provider, and
// some of the tools are also executed on the provider systems.
type ProviderTool struct {
	// ID of the tool. Should follow the format "<provider-id>.<unique-tool-name>".
	ID string

	// Name of the tool. Unique within this model call.
	Name string

	// Args are the arguments for configuring the tool.
	// Must match the expected arguments defined by the provider for this tool.
	Args map[string]any
}

func (ProviderTool) toolType() string { return "provider" }
