// Ported from: packages/core/src/tools/types.ts
package tools

import (
	"context"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/memory"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	"github.com/brainlet/brainkit/agent-kit/core/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/workspace"
)

// ---------------------------------------------------------------------------
// Cross-package interfaces — break circular dependencies
// ---------------------------------------------------------------------------

// AgentRef represents an agent in the Mastra system.
// Defined here (not imported from agent package) to break circular dependency:
// agent imports tools (for ToolAction, etc.), so tools cannot import agent.
// agent.Agent struct satisfies this interface via its ID() and Name() methods.
type AgentRef interface {
	// ID returns the agent's unique identifier.
	ID() string
	// Name returns the agent's display name.
	Name() string
}

// MastraRef represents the top-level Mastra orchestrator.
// Defined here (not imported from core package) to break circular dependency:
// core imports tools (for ToolAction), so tools cannot import core.
// core.Mastra struct satisfies this interface.
// Tools use this to access storage (e.g., mastra.GetStorage() in tool execution)
// and logging (for error reporting within tool handlers).
type MastraRef interface {
	// GetLogger returns the configured logger instance.
	GetLogger() logger.IMastraLogger
	// GetStorage returns the composite storage provider.
	GetStorage() *storage.MastraCompositeStore
}

// MastraUnion represents the union of Mastra | MastraServer | undefined.
// In TypeScript this is a union type that can hold any Mastra-like instance.
// Kept as any because:
// 1. The toolbuilder's wrapMastra function returns any (reflection-based wrapping)
// 2. Tool execution contexts pass it through to user-defined tool handlers
// 3. User code type-asserts to MastraRef or other narrow interfaces as needed
// Both core.Mastra and action.MastraUnion can be stored here.
type MastraUnion = any

// SuspendOptions is wired to the real workflows.SuspendOptions.
// Real type has ResumeLabel []string and Extra map[string]any fields.
type SuspendOptions = workflows.SuspendOptions

// OutputWriter is wired to the real workflows.OutputWriter.
// Real type: func(chunk any) error — same signature as the previous stub.
type OutputWriter = workflows.OutputWriter

// ObservabilityContext is wired to the real observability/types.ObservabilityContext.
// Real type has Tracing, LoggerVNext, Metrics, and TracingCtx fields for
// full observability integration (tracing, logging, metrics).
type ObservabilityContext = obstypes.ObservabilityContext

// Workspace is wired to the real workspace.AnyWorkspace (= *workspace.Workspace).
// Provides filesystem operations, command execution, and sandboxing through
// the workspace provider abstraction.
type Workspace = workspace.AnyWorkspace

// SchemaWithValidation is a stub for ../stream/base/schema.SchemaWithValidation.
// Stub: parallel-stubs architecture — real type has same shape but lives in stream/base.
// Kept local because tools defines its own SafeParser interface for validation;
// importing stream/base would add coupling for an identical struct { Schema any }.
type SchemaWithValidation struct {
	Schema any
}

// MastraMemory is wired to the real memory.MastraMemory interface.
// Real type provides thread-based conversation memory, semantic recall,
// working memory templates, and message processor support.
type MastraMemory = memory.MastraMemory

// FlexibleSchema is a stub for @internal/external-types.FlexibleSchema.
// ai-kit only ported V3 (@ai-sdk/provider-v6). These types remain local stubs.
type FlexibleSchema = any

// Schema is a stub for @internal/external-types.Schema (AI SDK schema).
// ai-kit only ported V3 (@ai-sdk/provider-v6). These types remain local stubs.
type Schema = any

// ToolCallOptions is a stub for @internal/external-types.ToolCallOptions.
// ai-kit only ported V3 (@ai-sdk/provider-v6). These types remain local stubs.
type ToolCallOptions struct {
	ToolCallID string `json:"toolCallId,omitempty"`
	Messages   []any  `json:"messages,omitempty"`
}

// ToolExecutionOptionsAISdk is a stub for @internal/external-types.ToolExecutionOptions.
// Named to avoid collision with the Mastra ToolExecutionContext.
// ai-kit only ported V3 (@ai-sdk/provider-v6). These types remain local stubs.
type ToolExecutionOptionsAISdk = any

// RequestHandlerExtra is a stub for @modelcontextprotocol/sdk RequestHandlerExtra.
// Stub: MCP Go SDK uses different type system; kept as any.
type RequestHandlerExtra = any

// ElicitRequestParams is a stub for @modelcontextprotocol/sdk ElicitRequest['params'].
// Stub: MCP Go SDK uses different type system; kept as any.
type ElicitRequestParams = any

// ElicitResult is a stub for @modelcontextprotocol/sdk ElicitResult.
// Stub: MCP Go SDK uses different type system; kept as any.
type ElicitResult = any

// ValidationError represents a tool input/output validation failure.
// Stub: parallel-stubs architecture — also defined in tools/validation.go with same shape.
type ValidationError struct {
	Error            bool   `json:"error"`
	Message          string `json:"message"`
	ValidationErrors any    `json:"validationErrors,omitempty"`
}

// DataChunkType is a stub for ../stream/types.DataChunkType.
// Stub: parallel-stubs architecture — real type in stream has same shape.
type DataChunkType struct {
	Type string `json:"type"` // must match `data-*` pattern
	Data any    `json:"data"`
	ID   string `json:"id,omitempty"`
}

// ---------------------------------------------------------------------------
// Ported types
// ---------------------------------------------------------------------------

// VercelTool is an alias for AI SDK Tool.
// In Go we represent it as any since the full AI SDK type is not ported.
type VercelTool = any

// VercelToolV5 is an alias for AI SDK ToolV5.
type VercelToolV5 = any

// ToolInvocationOptions represents the union of ToolExecutionOptions | ToolCallOptions
// from the AI SDK. In Go this is represented as any since both are external types.
type ToolInvocationOptions = any

// ---------------------------------------------------------------------------
// Execution Context Types
// ---------------------------------------------------------------------------

// AgentToolExecutionContext holds properties specific to agent-initiated tool execution.
type AgentToolExecutionContext struct {
	// Always present when called from agent context.
	ToolCallID string `json:"toolCallId"`
	Messages   []any  `json:"messages"`

	// ThreadID is optional - memory identifier.
	ThreadID string `json:"threadId,omitempty"`
	// ResourceID is optional - memory identifier.
	ResourceID string `json:"resourceId,omitempty"`

	// ResumeData is optional - only present if tool was previously suspended.
	ResumeData any `json:"resumeData,omitempty"`

	// WritableStream is an optional writer passed from AI SDK (without Mastra metadata wrapping).
	WritableStream any `json:"writableStream,omitempty"`

	// Suspend suspends the tool execution with optional payload and options.
	Suspend func(suspendPayload any, suspendOptions *SuspendOptions) error `json:"-"`
}

// WorkflowToolExecutionContext holds properties specific to workflow-initiated tool execution.
type WorkflowToolExecutionContext struct {
	// Always present when called from workflow context.
	RunID      string `json:"runId"`
	WorkflowID string `json:"workflowId"`
	State      any    `json:"state,omitempty"`

	// SetState updates the workflow state.
	SetState func(state any) `json:"-"`

	// ResumeData is optional - only present if workflow step was previously suspended.
	ResumeData any `json:"resumeData,omitempty"`

	// Suspend suspends the workflow step execution.
	Suspend func(suspendPayload any, suspendOptions *SuspendOptions) error `json:"-"`
}

// Elicitation provides interactive user input during MCP tool execution.
type Elicitation struct {
	// SendRequest sends an elicitation request to the MCP client.
	SendRequest func(request ElicitRequestParams) (ElicitResult, error) `json:"-"`
}

// MCPToolExecutionContext holds properties specific to MCP-initiated tool execution.
type MCPToolExecutionContext struct {
	// Extra is the MCP protocol context passed by the server.
	Extra RequestHandlerExtra `json:"extra,omitempty"`
	// Elicitation provides interactive user input during tool execution.
	Elicitation Elicitation `json:"-"`
}

// MastraToolInvocationOptions extends ToolInvocationOptions with Mastra-specific properties
// for suspend/resume functionality, stream writing, and tracing context.
//
// This is used by CoreTool/InternalCoreTool for AI SDK compatibility.
// Mastra v1.0 tools (ToolAction) use ToolExecutionContext instead.
//
// CoreToolBuilder acts as the adapter layer:
//   - Receives: AI SDK calls with MastraToolInvocationOptions
//   - Converts to: ToolExecutionContext for Mastra tool execution
//   - Returns: Results back to AI SDK
type MastraToolInvocationOptions struct {
	// Embedded observability context (partial).
	*ObservabilityContext

	// Suspend suspends tool execution with a payload.
	Suspend func(suspendPayload any, suspendOptions *SuspendOptions) (any, error) `json:"-"`
	// ResumeData is present if the tool was previously suspended.
	ResumeData any `json:"resumeData,omitempty"`
	// OutputWriter writes structured event chunks.
	OutputWriter OutputWriter `json:"-"`

	// MCP is the optional MCP-specific context passed when tool is executed in MCP server.
	MCP *MCPToolExecutionContext `json:"mcp,omitempty"`

	// Workspace for tool execution. When provided at execution time, this overrides
	// any workspace configured at tool build time.
	Workspace Workspace `json:"workspace,omitempty"`

	// RequestContext for tool execution. When provided at execution time, this overrides
	// any requestContext configured at tool build time.
	RequestContext *requestcontext.RequestContext `json:"requestContext,omitempty"`
}

// ---------------------------------------------------------------------------
// MCP Tool Types
// ---------------------------------------------------------------------------

// MCPToolType categorises a tool registered with the MCP server.
// Used to categorise tools in the MCP Server playground.
type MCPToolType string

const (
	MCPToolTypeAgent    MCPToolType = "agent"
	MCPToolTypeWorkflow MCPToolType = "workflow"
)

// ToolAnnotations describes tool behavior and UI presentation per the MCP protocol.
// See: https://spec.modelcontextprotocol.io/specification/2025-03-26/server/tools/#tool-annotations
type ToolAnnotations struct {
	// Title is a human-readable title for the tool.
	Title string `json:"title,omitempty"`
	// ReadOnlyHint indicates the tool does not modify its environment (default: false).
	ReadOnlyHint *bool `json:"readOnlyHint,omitempty"`
	// DestructiveHint indicates the tool may perform destructive updates (default: true).
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	// IdempotentHint indicates calling the tool repeatedly with the same arguments
	// will have no additional effect (default: false).
	IdempotentHint *bool `json:"idempotentHint,omitempty"`
	// OpenWorldHint indicates the tool may interact with external entities (default: true).
	OpenWorldHint *bool `json:"openWorldHint,omitempty"`
}

// MCPToolProperties holds MCP-specific properties for tools.
type MCPToolProperties struct {
	// ToolType categorises the tool in the MCP Server playground.
	ToolType MCPToolType `json:"toolType,omitempty"`
	// Annotations describes tool behavior and UI presentation.
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
	// Meta holds arbitrary metadata passed through to MCP clients.
	Meta map[string]any `json:"_meta,omitempty"`
}

// ---------------------------------------------------------------------------
// CoreTool
// ---------------------------------------------------------------------------

// CoreToolType distinguishes between function tools and provider-defined tools.
type CoreToolType string

const (
	CoreToolTypeFunction        CoreToolType = "function"
	CoreToolTypeProviderDefined CoreToolType = "provider-defined"
)

// CoreTool is the AI SDK-compatible tool format used when passing tools to the AI SDK.
// It matches the AI SDK's Tool interface.
//
// CoreToolBuilder converts Mastra tools (ToolAction) to this format and handles the
// signature transformation from Mastra's (inputData, context) to AI SDK format.
//
// Key differences from ToolAction:
//   - Uses Parameters instead of InputSchema (AI SDK naming)
//   - Execute signature: (params, options) (AI SDK format)
//   - Supports FlexibleSchema | Schema for broader AI SDK compatibility
type CoreTool struct {
	// Type is "function" (default) or "provider-defined".
	Type CoreToolType `json:"type,omitempty"`
	// ID is the tool identifier. For provider-defined tools, format is "provider.tool_name".
	ID string `json:"id,omitempty"`
	// Description is a human-readable description of the tool.
	Description string `json:"description,omitempty"`
	// Parameters defines the input schema (FlexibleSchema | Schema).
	Parameters any `json:"parameters"`
	// OutputSchema defines the output schema.
	OutputSchema any `json:"outputSchema,omitempty"`
	// ProviderOptions holds provider-specific options passed to the model.
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`
	// MCP holds optional MCP-specific properties.
	MCP *MCPToolProperties `json:"mcp,omitempty"`
	// InputExamples shows valid argument examples for model providers.
	InputExamples []InputExample `json:"inputExamples,omitempty"`
	// Args holds arguments for provider-defined tools.
	Args map[string]any `json:"args,omitempty"`

	// Execute runs the tool with the given params and options.
	Execute func(params any, options *MastraToolInvocationOptions) (any, error) `json:"-"`
	// ToModelOutput transforms tool output before returning to the model.
	ToModelOutput func(output any) any `json:"-"`

	// Lifecycle callbacks (AI SDK hooks).
	OnInputStart     func(options ToolCallOptions)                                                `json:"-"`
	OnInputDelta     func(options InputDeltaOptions)                                              `json:"-"`
	OnInputAvailable func(options InputAvailableOptions)                                          `json:"-"`
	OnOutput         func(options OutputOptions)                                                  `json:"-"`
}

// InputExample represents an example of valid tool input.
type InputExample struct {
	Input map[string]any `json:"input"`
}

// InputDeltaOptions extends ToolCallOptions with an input text delta.
type InputDeltaOptions struct {
	ToolCallOptions
	InputTextDelta string `json:"inputTextDelta"`
}

// InputAvailableOptions extends ToolCallOptions with the full input.
type InputAvailableOptions struct {
	ToolCallOptions
	Input any `json:"input"`
}

// OutputOptions holds the output and tool name, without messages from ToolCallOptions.
type OutputOptions struct {
	ToolCallID string `json:"toolCallId,omitempty"`
	Output     any    `json:"output"`
	ToolName   string `json:"toolName"`
}

// ---------------------------------------------------------------------------
// InternalCoreTool
// ---------------------------------------------------------------------------

// InternalCoreTool is identical to CoreTool but with stricter typing.
// Used internally where the schema has already been converted to AI SDK Schema format.
//
// The only difference: Parameters must be Schema (not FlexibleSchema | Schema).
// In Go, this distinction is documented but not enforced at the type level
// since both are represented as any.
type InternalCoreTool = CoreTool

// ---------------------------------------------------------------------------
// ToolExecutionContext
// ---------------------------------------------------------------------------

// ToolExecutionContext is the unified tool execution context that works for all scenarios
// (agent, workflow, MCP, direct execution).
type ToolExecutionContext struct {
	// Embedded observability context (partial).
	*ObservabilityContext

	// Common properties (available in all contexts).
	Mastra         MastraUnion                   `json:"-"`
	RequestContext *requestcontext.RequestContext `json:"-"`
	Ctx            context.Context               `json:"-"` // Go equivalent of AbortSignal

	// Workspace available for tool execution. When provided, tools can access
	// filesystem operations and command execution through the workspace.
	Workspace Workspace `json:"-"`

	// Writer is created by Mastra for ALL contexts (agent, workflow, direct execution).
	// Wraps chunks with metadata (toolCallId, toolName, runId) before passing to underlying stream.
	Writer *ToolStream `json:"-"`

	// Context-specific nested properties.

	// Agent holds agent-specific properties (present when called from agent context).
	Agent *AgentToolExecutionContext `json:"-"`
	// Workflow holds workflow-specific properties (present when called from workflow context).
	Workflow *WorkflowToolExecutionContext `json:"-"`
	// MCP holds MCP-specific properties (present when called from MCP context).
	MCP *MCPToolExecutionContext `json:"-"`
}

// ---------------------------------------------------------------------------
// ToolAction
// ---------------------------------------------------------------------------

// ToolAction defines a Mastra tool with full type information, schemas, and execution logic.
//
// This is the primary tool definition type in Mastra v1.0.
// TypeScript generics (TSchemaIn, TSchemaOut, TSuspend, TResume, TContext, TId, TRequestContext)
// are collapsed to any in Go.
type ToolAction struct {
	// ID uniquely identifies the tool.
	ID string `json:"id"`
	// Description is a human-readable description of the tool.
	Description string `json:"description"`

	// InputSchema validates the tool's input data.
	InputSchema *SchemaWithValidation `json:"inputSchema,omitempty"`
	// OutputSchema validates the tool's output data.
	OutputSchema *SchemaWithValidation `json:"outputSchema,omitempty"`
	// SuspendSchema validates suspend payloads.
	SuspendSchema *SchemaWithValidation `json:"suspendSchema,omitempty"`
	// ResumeSchema validates resume payloads.
	ResumeSchema *SchemaWithValidation `json:"resumeSchema,omitempty"`
	// RequestContextSchema validates request context values before tool execution.
	// When provided, the request context will be validated against this schema.
	// If validation fails, a validation error is returned instead of executing the tool.
	RequestContextSchema *SchemaWithValidation `json:"requestContextSchema,omitempty"`

	// MCP holds optional MCP-specific properties.
	MCP *MCPToolProperties `json:"mcp,omitempty"`

	// ProviderOptions holds provider-specific options passed to the model.
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`

	// InputExamples shows valid argument examples for model providers.
	InputExamples []InputExample `json:"inputExamples,omitempty"`

	// RequireApproval indicates whether the tool requires user approval before execution.
	RequireApproval bool `json:"requireApproval,omitempty"`

	// Mastra is the Mastra instance associated with this tool.
	Mastra MastraRef `json:"-"`

	// Execute runs the tool.
	// First parameter: raw input data (validated against InputSchema).
	// Second parameter: unified execution context with all metadata.
	// Returns: the expected output OR a ValidationError if input validation fails.
	Execute func(inputData any, ctx *ToolExecutionContext) (any, error) `json:"-"`

	// ToModelOutput transforms tool output before returning to the model.
	ToModelOutput func(output any) any `json:"-"`

	// Lifecycle callbacks (AI SDK hooks).
	OnInputStart     func(options ToolCallOptions)         `json:"-"`
	OnInputDelta     func(options InputDeltaOptions)       `json:"-"`
	OnInputAvailable func(options InputAvailableOptions)   `json:"-"`
	OnOutput         func(options OutputOptions)           `json:"-"`
}
