// Package integration provides OpenAPI toolset support.
//
// Ported from: packages/core/src/integration/openapi-toolset.ts
package integration

import (
	"fmt"

	"github.com/brainlet/brainkit/agent-kit/core/tools"
)

// ToolDocumentation holds documentation metadata for a tool.
//
// Ported from: packages/core/src/integration/openapi-toolset.ts — toolDocumentations return type
type ToolDocumentation struct {
	Comment string
	Doc     string
}

// OpenAPIToolset is the abstract base for OpenAPI-based tool providers.
// Concrete implementations must provide Name, Tools, ToolSchemas,
// ToolDocumentations, and BaseClient.
//
// Ported from: packages/core/src/integration/openapi-toolset.ts — OpenAPIToolset
type OpenAPIToolset struct {
	// AuthType is the authentication type for the toolset.
	// Defaults to "API_KEY".
	//
	// Ported from: packages/core/src/integration/openapi-toolset.ts — authType
	AuthType string

	// GetName returns the name of the toolset.
	// Must be provided by the concrete implementation.
	//
	// Ported from: packages/core/src/integration/openapi-toolset.ts — abstract readonly name
	GetName func() string

	// GetTools returns all tools provided by the toolset.
	// Must be provided by the concrete implementation.
	//
	// Ported from: packages/core/src/integration/openapi-toolset.ts — abstract readonly tools
	GetTools func() map[string]*tools.ToolAction

	// GetToolSchemas returns the JSON schemas for each tool input, keyed by tool name.
	// Override to provide schemas. Defaults to empty map.
	//
	// Ported from: packages/core/src/integration/openapi-toolset.ts — get toolSchemas()
	GetToolSchemas func() map[string]map[string]any

	// GetToolDocumentations returns documentation for each tool, keyed by tool name.
	// Override to provide documentation. Defaults to empty map.
	//
	// Ported from: packages/core/src/integration/openapi-toolset.ts — get toolDocumentations()
	GetToolDocumentations func() map[string]ToolDocumentation

	// GetBaseClient returns the base API client with callable methods.
	// Override to provide the client. Defaults to nil.
	//
	// Ported from: packages/core/src/integration/openapi-toolset.ts — get baseClient()
	GetBaseClient func() map[string]func(input any) (any, error)
}

// NewOpenAPIToolset creates a new OpenAPIToolset with default values.
//
// Ported from: packages/core/src/integration/openapi-toolset.ts — constructor()
func NewOpenAPIToolset() *OpenAPIToolset {
	return &OpenAPIToolset{
		AuthType: "API_KEY",
	}
}

// GetAPIClient returns the API client for the toolset.
// Base implementation returns an error — subclasses must override.
//
// Ported from: packages/core/src/integration/openapi-toolset.ts — getApiClient()
func (o *OpenAPIToolset) GetAPIClient() (any, error) {
	return nil, fmt.Errorf("API not implemented")
}

// GenerateIntegrationTools creates tools from the base client methods, schemas,
// and documentation. Each client method becomes a tool that delegates to the
// API client.
//
// Ported from: packages/core/src/integration/openapi-toolset.ts — _generateIntegrationTools()
func (o *OpenAPIToolset) GenerateIntegrationTools() map[string]*tools.ToolAction {
	clientMethods := make(map[string]func(input any) (any, error))
	if o.GetBaseClient != nil {
		clientMethods = o.GetBaseClient()
	}

	schemas := make(map[string]map[string]any)
	if o.GetToolSchemas != nil {
		schemas = o.GetToolSchemas()
	}

	documentations := make(map[string]ToolDocumentation)
	if o.GetToolDocumentations != nil {
		documentations = o.GetToolDocumentations()
	}

	result := make(map[string]*tools.ToolAction, len(clientMethods))

	for key := range clientMethods {
		// Capture loop variable for closure
		methodKey := key

		comment := ""
		if doc, ok := documentations[methodKey]; ok {
			comment = doc.Comment
		}
		if comment == "" {
			comment = fmt.Sprintf("Execute %s", methodKey)
		}

		// Get the input schema for this tool, defaulting to empty object schema
		var inputSchema *tools.SchemaWithValidation
		if s, ok := schemas[methodKey]; ok {
			inputSchema = &tools.SchemaWithValidation{Schema: s}
		} else {
			inputSchema = &tools.SchemaWithValidation{
				Schema: map[string]any{"type": "object", "properties": map[string]any{}},
			}
		}

		tool := tools.CreateTool(tools.ToolAction{
			ID:          methodKey,
			InputSchema: inputSchema,
			Description: comment,
			Execute: func(inputData any, ctx *tools.ToolExecutionContext) (any, error) {
				// Get a fresh client for each invocation
				client := clientMethods
				if o.GetBaseClient != nil {
					client = o.GetBaseClient()
				}
				method, ok := client[methodKey]
				if !ok {
					return nil, fmt.Errorf("method %q not found on client", methodKey)
				}
				return method(inputData)
			},
		})

		result[methodKey] = &tools.ToolAction{
			ID:          tool.ID,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
	}

	return result
}
