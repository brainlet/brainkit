// Ported from: packages/provider-utils/src/create-tool-name-mapping.ts
package providerutils

// ToolNameMapping provides bidirectional mapping between custom tool names
// and provider tool names.
type ToolNameMapping struct {
	// ToProviderToolName maps a custom tool name to the provider's tool name.
	// If the custom tool name does not have a mapping, returns the input name.
	ToProviderToolName func(customToolName string) string
	// ToCustomToolName maps a provider tool name to the custom tool name.
	// If the provider tool name does not have a mapping, returns the input name.
	ToCustomToolName func(providerToolName string) string
}

// ProviderToolDefinition represents a tool definition (function or provider type).
type ProviderToolDefinition struct {
	// Type is the tool type: "function", "provider", or "dynamic".
	Type string
	// Name is the custom name of the tool.
	Name string
	// ID is the provider tool ID (for provider tools). Must follow "provider.toolName" format.
	ID string
}

// CreateToolNameMappingOptions are the options for CreateToolNameMapping.
type CreateToolNameMappingOptions struct {
	// Tools that were passed to the language model.
	Tools []ProviderToolDefinition
	// ProviderToolNames maps the provider tool IDs to the provider tool names.
	ProviderToolNames map[string]string
	// ResolveProviderToolName is an optional resolver for provider tool names
	// that cannot be represented as static id -> name mappings.
	ResolveProviderToolName func(tool ProviderToolDefinition) *string
}

// CreateToolNameMapping creates a bidirectional mapping between custom tool names
// and provider tool names based on the provided tools and name mappings.
func CreateToolNameMapping(opts CreateToolNameMappingOptions) ToolNameMapping {
	customToProvider := make(map[string]string)
	providerToCustom := make(map[string]string)

	for _, tool := range opts.Tools {
		if tool.Type != "provider" {
			continue
		}

		var providerToolName *string

		// Try the resolver first
		if opts.ResolveProviderToolName != nil {
			providerToolName = opts.ResolveProviderToolName(tool)
		}

		// Fall back to static mapping
		if providerToolName == nil {
			if name, ok := opts.ProviderToolNames[tool.ID]; ok {
				providerToolName = &name
			}
		}

		if providerToolName == nil {
			continue
		}

		customToProvider[tool.Name] = *providerToolName
		providerToCustom[*providerToolName] = tool.Name
	}

	return ToolNameMapping{
		ToProviderToolName: func(customToolName string) string {
			if name, ok := customToProvider[customToolName]; ok {
				return name
			}
			return customToolName
		},
		ToCustomToolName: func(providerToolName string) string {
			if name, ok := providerToCustom[providerToolName]; ok {
				return name
			}
			return providerToolName
		},
	}
}
