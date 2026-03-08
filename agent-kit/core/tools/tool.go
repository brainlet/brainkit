// Ported from: packages/core/src/tools/tool.ts
package tools

import (
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// MASTRA_TOOL_MARKER
// ---------------------------------------------------------------------------

// MastraToolMarkerKey is the sentinel key used to identify Mastra tools even
// when type assertions fail. This mirrors the TypeScript
// Symbol.for('mastra.core.tool.Tool') pattern.
//
// NOTE: The constant MastraToolMarker is already declared in toolchecks.go.
// This file uses it by reference.

// ---------------------------------------------------------------------------
// Tool
// ---------------------------------------------------------------------------

// Tool is a type-safe tool that agents and workflows can call to perform
// specific actions. It wraps a ToolAction with automatic input/output
// validation.
//
// In TypeScript this is a generic class with 7 type parameters
// (TSchemaIn, TSchemaOut, TSuspendSchema, TResumeSchema, TContext, TId, TRequestContext).
// In Go, all schema/data types are collapsed to any.
type Tool struct {
	// ID uniquely identifies the tool.
	ID string `json:"id"`

	// Description is a human-readable description of what the tool does.
	Description string `json:"description"`

	// InputSchema validates the tool's input data.
	InputSchema *SchemaWithValidation `json:"inputSchema,omitempty"`

	// OutputSchema validates the tool's output data.
	OutputSchema *SchemaWithValidation `json:"outputSchema,omitempty"`

	// SuspendSchema validates suspend operation data.
	SuspendSchema *SchemaWithValidation `json:"suspendSchema,omitempty"`

	// ResumeSchema validates resume operation data.
	ResumeSchema *SchemaWithValidation `json:"resumeSchema,omitempty"`

	// RequestContextSchema validates request context values before tool execution.
	// When provided, the request context will be validated against this schema.
	RequestContextSchema *SchemaWithValidation `json:"requestContextSchema,omitempty"`

	// Execute runs the tool.
	// First parameter: raw input data (will be validated against InputSchema).
	// Second parameter: unified execution context with all metadata.
	// Returns: the expected output OR a *ValidationError if validation fails.
	Execute func(inputData any, ctx *ToolExecutionContext) (any, error) `json:"-"`

	// Mastra is the parent Mastra instance for accessing shared resources.
	Mastra MastraRef `json:"-"`

	// RequireApproval indicates whether the tool requires explicit user
	// approval before execution.
	RequireApproval bool `json:"requireApproval,omitempty"`

	// ProviderOptions holds provider-specific options passed to the model
	// when this tool is used. Keys are provider names (e.g., "anthropic",
	// "openai"), values are provider-specific configs.
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`

	// ToModelOutput transforms the tool's raw output before sending it to
	// the model. The raw result is still available for application logic;
	// only the model sees the transformed version.
	ToModelOutput func(output any) any `json:"-"`

	// InputExamples shows valid argument examples for model providers.
	InputExamples []InputExample `json:"inputExamples,omitempty"`

	// MCP holds optional MCP-specific properties including annotations
	// and metadata. Only relevant when the tool is used in an MCP context.
	MCP *MCPToolProperties `json:"mcp,omitempty"`

	// Lifecycle callbacks (AI SDK hooks).
	OnInputStart     func(options ToolCallOptions)       `json:"-"`
	OnInputDelta     func(options InputDeltaOptions)     `json:"-"`
	OnInputAvailable func(options InputAvailableOptions) `json:"-"`
	OnOutput         func(options OutputOptions)         `json:"-"`

	// isMastraTool is an internal flag that marks this as a Mastra tool,
	// equivalent to the MASTRA_TOOL_MARKER Symbol in TypeScript.
	isMastraTool bool
}

// IsMastraTool implements MastraToolChecker, allowing this Tool to be
// identified via the interface-based marker check in IsMastraTool().
func (t *Tool) IsMastraTool() bool {
	return t.isMastraTool
}

// Ensure Tool implements MastraToolChecker at compile time.
var _ MastraToolChecker = (*Tool)(nil)

// NewTool creates a new Tool instance from a ToolAction, wrapping the execute
// function with automatic input, output, suspend, resume, and request context
// validation.
//
// This is the Go equivalent of the TypeScript Tool constructor.
func NewTool(opts ToolAction) *Tool {
	t := &Tool{
		isMastraTool:         true,
		ID:                   opts.ID,
		Description:          opts.Description,
		InputSchema:          opts.InputSchema,
		OutputSchema:         opts.OutputSchema,
		SuspendSchema:        opts.SuspendSchema,
		ResumeSchema:         opts.ResumeSchema,
		RequestContextSchema: opts.RequestContextSchema,
		Mastra:               opts.Mastra,
		RequireApproval:      opts.RequireApproval,
		ProviderOptions:      opts.ProviderOptions,
		ToModelOutput:        opts.ToModelOutput,
		InputExamples:        opts.InputExamples,
		MCP:                  opts.MCP,
		OnInputStart:         opts.OnInputStart,
		OnInputDelta:         opts.OnInputDelta,
		OnInputAvailable:     opts.OnInputAvailable,
		OnOutput:             opts.OnOutput,
	}

	if opts.Execute != nil {
		originalExecute := opts.Execute

		t.Execute = func(inputData any, ctx *ToolExecutionContext) (any, error) {
			// Validate input if schema exists.
			data, inputErr := ValidateToolInput(t.InputSchema, inputData, t.ID)
			if inputErr != nil {
				return inputErr, nil
			}

			// Validate request context if schema exists.
			var rc *requestcontext.RequestContext
			if ctx != nil {
				rc = ctx.RequestContext
			}
			_, rcErr := ValidateRequestContext(t.RequestContextSchema, rc, t.ID)
			if rcErr != nil {
				return rcErr, nil
			}

			// Track suspend data for post-execution validation.
			var suspendData any
			var suspendCalled bool

			// Build the organized context, wrapping suspend functions
			// to capture suspend data for validation.
			organizedCtx := organizeContext(ctx, &suspendData, &suspendCalled)

			// Validate resume data if present.
			resumeData := getResumeData(organizedCtx)
			if resumeData != nil {
				_, resumeErr := ValidateToolInput(t.ResumeSchema, resumeData, t.ID)
				if resumeErr != nil {
					return resumeErr, nil
				}
			}

			// Call the original execute with validated input and organized context.
			output, execErr := originalExecute(data, organizedCtx)
			if execErr != nil {
				return nil, execErr
			}

			// Validate suspend data if suspend was called.
			if suspendCalled {
				_, suspendErr := ValidateToolSuspendData(t.SuspendSchema, suspendData, t.ID)
				if suspendErr != nil {
					return suspendErr, nil
				}
			}

			// Validate output if schema exists.
			skipOutputValidation := output == nil && suspendCalled
			validatedOutput, outputErr := ValidateToolOutput(t.OutputSchema, output, t.ID, skipOutputValidation)
			if outputErr != nil {
				return outputErr, nil
			}

			return validatedOutput, nil
		}
	}

	return t
}

// CreateTool creates a type-safe tool with automatic input validation.
// This is the Go equivalent of the TypeScript createTool() factory function.
func CreateTool(opts ToolAction) *Tool {
	return NewTool(opts)
}

// ---------------------------------------------------------------------------
// Context organization helpers
// ---------------------------------------------------------------------------

// organizeContext builds the organized execution context, wrapping suspend
// functions to capture suspend data for validation. This mirrors the
// TypeScript context reorganization logic in the Tool constructor.
func organizeContext(ctx *ToolExecutionContext, suspendData *any, suspendCalled *bool) *ToolExecutionContext {
	if ctx == nil {
		// No context provided - create a minimal context with requestContext.
		return &ToolExecutionContext{
			RequestContext: requestcontext.NewRequestContext(),
		}
	}

	// Ensure requestContext is always present.
	organized := *ctx // shallow copy
	if organized.RequestContext == nil {
		organized.RequestContext = requestcontext.NewRequestContext()
	}

	// Wrap agent suspend to capture suspend data.
	if organized.Agent != nil && organized.Agent.Suspend != nil {
		originalSuspend := organized.Agent.Suspend
		agentCopy := *organized.Agent
		agentCopy.Suspend = func(payload any, opts *SuspendOptions) error {
			*suspendData = payload
			*suspendCalled = true
			return originalSuspend(payload, opts)
		}
		organized.Agent = &agentCopy
	}

	// Wrap workflow suspend to capture suspend data.
	if organized.Workflow != nil && organized.Workflow.Suspend != nil {
		originalSuspend := organized.Workflow.Suspend
		wfCopy := *organized.Workflow
		wfCopy.Suspend = func(payload any, opts *SuspendOptions) error {
			*suspendData = payload
			*suspendCalled = true
			return originalSuspend(payload, opts)
		}
		organized.Workflow = &wfCopy
	}

	return &organized
}

// getResumeData extracts resume data from the context, checking agent,
// workflow, and top-level context in order.
func getResumeData(ctx *ToolExecutionContext) any {
	if ctx == nil {
		return nil
	}
	if ctx.Agent != nil && ctx.Agent.ResumeData != nil {
		return ctx.Agent.ResumeData
	}
	if ctx.Workflow != nil && ctx.Workflow.ResumeData != nil {
		return ctx.Workflow.ResumeData
	}
	return nil
}
