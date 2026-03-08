// Ported from: packages/provider-utils/src/provider-tool-factory.ts
package providerutils

// ProviderToolConfig contains the configuration for creating a provider tool factory.
type ProviderToolConfig[INPUT any] struct {
	// ID is the tool ID. Must follow the format "provider.toolName".
	ID string
	// InputSchema is the schema for validating tool input.
	InputSchema *Schema[INPUT]
}

// ProviderToolOptions contains the user-configurable options for a provider tool.
type ProviderToolOptions[INPUT any, OUTPUT any] struct {
	// Execute is the function that runs the tool.
	Execute func(input INPUT, opts ToolExecutionOptions) (OUTPUT, error)
	// OutputSchema is the optional schema for validating tool output.
	OutputSchema *Schema[OUTPUT]
	// NeedsApproval indicates whether the tool needs approval before execution.
	NeedsApproval interface{}
	// Args are additional provider-specific arguments.
	Args map[string]interface{}
}

// ProviderTool represents a fully configured provider tool.
type ProviderTool[INPUT any, OUTPUT any] struct {
	// Type is always "provider" for provider tools.
	Type string
	// ID is the tool ID.
	ID string
	// InputSchema is the input validation schema.
	InputSchema *Schema[INPUT]
	// OutputSchema is the optional output validation schema.
	OutputSchema *Schema[OUTPUT]
	// Execute is the function that runs the tool.
	Execute func(input INPUT, opts ToolExecutionOptions) (OUTPUT, error)
	// NeedsApproval indicates whether the tool needs approval.
	NeedsApproval interface{}
	// Args are additional provider-specific arguments.
	Args map[string]interface{}
	// SupportsDeferredResults indicates whether this tool supports deferred results.
	SupportsDeferredResults bool
}

// CreateProviderToolFactory creates a factory function for provider tools.
func CreateProviderToolFactory[INPUT any](config ProviderToolConfig[INPUT]) func(opts ProviderToolOptions[INPUT, interface{}]) ProviderTool[INPUT, interface{}] {
	return func(opts ProviderToolOptions[INPUT, interface{}]) ProviderTool[INPUT, interface{}] {
		return ProviderTool[INPUT, interface{}]{
			Type:          "provider",
			ID:            config.ID,
			InputSchema:   config.InputSchema,
			OutputSchema:  opts.OutputSchema,
			Execute:       opts.Execute,
			NeedsApproval: opts.NeedsApproval,
			Args:          opts.Args,
		}
	}
}

// ProviderToolWithOutputSchemaConfig contains the configuration for a provider tool
// factory that also specifies an output schema.
type ProviderToolWithOutputSchemaConfig[INPUT any, OUTPUT any] struct {
	// ID is the tool ID.
	ID string
	// InputSchema is the schema for validating tool input.
	InputSchema *Schema[INPUT]
	// OutputSchema is the schema for validating tool output.
	OutputSchema *Schema[OUTPUT]
	// SupportsDeferredResults indicates whether this tool supports deferred results.
	SupportsDeferredResults bool
}

// CreateProviderToolFactoryWithOutputSchema creates a factory for provider tools
// that includes a predefined output schema.
func CreateProviderToolFactoryWithOutputSchema[INPUT any, OUTPUT any](
	config ProviderToolWithOutputSchemaConfig[INPUT, OUTPUT],
) func(opts ProviderToolOptions[INPUT, OUTPUT]) ProviderTool[INPUT, OUTPUT] {
	return func(opts ProviderToolOptions[INPUT, OUTPUT]) ProviderTool[INPUT, OUTPUT] {
		return ProviderTool[INPUT, OUTPUT]{
			Type:                    "provider",
			ID:                      config.ID,
			InputSchema:             config.InputSchema,
			OutputSchema:            config.OutputSchema,
			Execute:                 opts.Execute,
			NeedsApproval:           opts.NeedsApproval,
			Args:                    opts.Args,
			SupportsDeferredResults: config.SupportsDeferredResults,
		}
	}
}
