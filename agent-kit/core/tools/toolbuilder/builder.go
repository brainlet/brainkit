// Ported from: packages/core/src/tools/tool-builder/builder.ts
package toolbuilder

import (
	"encoding/json"
	"fmt"
	"strings"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	tracingtypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	"github.com/brainlet/brainkit/agent-kit/core/tools"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// SchemaCompatLayer is a stub for @mastra/schema-compat compatibility layers.
// Stub: Zod schema-compat is TS-specific; not applicable in Go.
type SchemaCompatLayer interface {
	// ShouldApply returns true if this layer is applicable to the current model.
	ShouldApply() bool
	// ProcessZodType processes a schema through the compatibility layer.
	ProcessZodType(schema any) any
}

// MastraLanguageModel is a stub for the language model type from AI SDK.
// Stub: AI SDK language model type; simplified for tool builder needs.
type MastraLanguageModel struct {
	ModelID                  string `json:"modelId"`
	Provider                 string `json:"provider"`
	SpecificationVersion     string `json:"specificationVersion,omitempty"`
	SupportsStructuredOutputs bool   `json:"supportsStructuredOutputs,omitempty"`
}

// ---------------------------------------------------------------------------
// Validation stub functions
// ---------------------------------------------------------------------------

// ValidationResult holds the outcome of a schema validation.
// Stub: validation logic defined locally — Zod runtime not applicable in Go.
type ValidationResult struct {
	Data  any
	Error *tools.ValidationError
}

// validateToolInput validates input data against a schema.
// Stub: validation logic defined locally — Zod runtime not applicable in Go.
func validateToolInput(schema any, input any, toolID string) ValidationResult {
	// Without Zod schemas in Go, pass through as-is.
	return ValidationResult{Data: input}
}

// validateToolOutput validates output data against a schema.
// Stub: validation logic defined locally — Zod runtime not applicable in Go.
func validateToolOutput(schema any, output any, toolID string, suspendCalled bool) ValidationResult {
	return ValidationResult{Data: output}
}

// validateToolSuspendData validates suspend data against a schema.
// Stub: validation logic defined locally — Zod runtime not applicable in Go.
func validateToolSuspendData(schema any, suspendData any, toolID string) ValidationResult {
	return ValidationResult{Data: suspendData}
}

// applyCompatLayer applies schema compatibility layers.
// Stub: Zod schema-compat is TS-specific; not applicable in Go.
func applyCompatLayer(schema any, compatLayers []SchemaCompatLayer, mode string) any {
	return schema
}

// convertZodSchemaToAISDKSchema converts a Zod schema to AI SDK Schema format.
// Stub: Zod schema-compat is TS-specific; not applicable in Go.
func convertZodSchemaToAISDKSchema(schema any) any {
	return schema
}

// zodToJsonSchema converts a Zod schema to JSON Schema.
// Stub: Zod-to-JSON-Schema conversion is TS-specific; not applicable in Go.
func zodToJsonSchema(schema any) any {
	return schema
}

// isZodObject checks if a value is a Zod object schema.
// Stub: Zod object detection is TS-specific; uses map heuristic in Go.
func isZodObject(schema any) bool {
	if schema == nil {
		return false
	}
	if m, ok := schema.(map[string]any); ok {
		if typeName, exists := m["_typeName"]; exists {
			return typeName == "ZodObject"
		}
	}
	return false
}

// wrapMastra wraps a Mastra instance with tracing context.
// Stub: real observability package has different shape; simplified for builder needs.
func wrapMastra(mastra any, ctx map[string]any) any {
	return mastra
}

// createObservabilityContext creates an ObservabilityContext from a tracing context map.
// Stub: real observability package has different shape; simplified for builder needs.
func createObservabilityContext(ctx map[string]any) *tools.ObservabilityContext {
	return &tools.ObservabilityContext{}
}

// ---------------------------------------------------------------------------
// Types exported from this package
// ---------------------------------------------------------------------------

// LogType categorises tool logging.
type LogType string

const (
	LogTypeTool       LogType = "tool"
	LogTypeToolset    LogType = "toolset"
	LogTypeClientTool LogType = "client-tool"
)

// LogOptions holds parameters for creating log messages.
type LogOptions struct {
	AgentName string  `json:"agentName,omitempty"`
	ToolName  string  `json:"toolName"`
	Type      LogType `json:"type,omitempty"`
}

// LogMessageOptions holds formatted start/error log messages.
type LogMessageOptions struct {
	Start string `json:"start"`
	Error string `json:"error"`
}

// ToolOptions holds the options for building a tool.
// This is the Go equivalent of ToolOptions from utils.ts.
type ToolOptions struct {
	Name           string                               `json:"name"`
	RunID          string                               `json:"runId,omitempty"`
	ThreadID       string                               `json:"threadId,omitempty"`
	ResourceID     string                               `json:"resourceId,omitempty"`
	Logger         logger.IMastraLogger                  `json:"-"`
	Description    string                               `json:"description,omitempty"`
	Mastra         any                                  `json:"-"` // MastraUnion
	RequestContext *requestcontext.RequestContext        `json:"-"`
	TracingContext *tracingtypes.TracingContext           `json:"-"`
	TracingPolicy  *tracingtypes.TracingPolicy            `json:"-"`
	Memory         tools.MastraMemory                    `json:"-"`
	AgentName      string                               `json:"agentName,omitempty"`
	Model          *MastraLanguageModel                  `json:"model,omitempty"`
	OutputWriter   tools.OutputWriter                    `json:"-"`
	RequireApproval bool                                `json:"requireApproval,omitempty"`

	// Workflow-specific properties.
	Workflow   any                `json:"-"`
	WorkflowID string             `json:"workflowId,omitempty"`
	State      any                `json:"-"`
	SetState   func(state any)    `json:"-"`

	// Workspace for tool execution.
	Workspace tools.Workspace `json:"-"`
}

// CoreToolBuilderInput holds the constructor parameters for CoreToolBuilder.
type CoreToolBuilderInput struct {
	OriginalTool            any       `json:"originalTool"`
	Options                 ToolOptions `json:"options"`
	LogType                 LogType   `json:"logType,omitempty"`
	AutoResumeSuspendedTools bool     `json:"autoResumeSuspendedTools,omitempty"`
}

// ---------------------------------------------------------------------------
// CoreToolBuilder
// ---------------------------------------------------------------------------

// CoreToolBuilder converts Mastra tools (ToolAction) and Vercel tools into
// the CoreTool format understood by the AI SDK.
//
// It extends MastraBase for logging and acts as the adapter layer:
//   - Receives: AI SDK calls with MastraToolInvocationOptions
//   - Converts to: ToolExecutionContext for Mastra tool execution
//   - Returns: Results back to AI SDK
type CoreToolBuilder struct {
	*agentkit.MastraBase

	originalTool any
	options      ToolOptions
	logType      LogType
}

// NewCoreToolBuilder creates a new CoreToolBuilder from the given input.
func NewCoreToolBuilder(input CoreToolBuilderInput) *CoreToolBuilder {
	b := &CoreToolBuilder{
		MastraBase:   agentkit.NewMastraBase(agentkit.MastraBaseOptions{Name: "CoreToolBuilder"}),
		originalTool: input.OriginalTool,
		options:      input.Options,
		logType:      input.LogType,
	}

	if !tools.IsVercelTool(b.originalTool) {
		shouldExtendSchema := input.AutoResumeSuspendedTools
		if !shouldExtendSchema {
			if ta, ok := b.originalTool.(*tools.ToolAction); ok {
				shouldExtendSchema = strings.HasPrefix(ta.ID, "agent-") || strings.HasPrefix(ta.ID, "workflow-")
			}
		}

		if shouldExtendSchema {
			// In Go, schema extension for suspend/resume fields would be handled
			// by the schema-compat layer once ported. For now, this is a no-op placeholder.
			// TypeScript adds suspendedToolRunId and resumeData fields to the input schema.
		}
	}

	return b
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// getParameters returns the input parameters schema based on tool type.
func (b *CoreToolBuilder) getParameters() any {
	if tools.IsVercelTool(b.originalTool) {
		m, ok := b.originalTool.(map[string]any)
		if !ok {
			return nil
		}
		// Handle both 'parameters' (v4) and 'inputSchema' (v5) properties.
		if params, exists := m["parameters"]; exists {
			return resolveSchema(params)
		}
		if input, exists := m["inputSchema"]; exists {
			return resolveSchema(input)
		}
		return nil
	}

	// For Mastra tools, use InputSchema.
	if ta, ok := b.originalTool.(*tools.ToolAction); ok {
		if ta.InputSchema != nil {
			return ta.InputSchema
		}
	}
	// Fallback: check map-based tool.
	if m, ok := b.originalTool.(map[string]any); ok {
		if schema, exists := m["inputSchema"]; exists {
			return resolveSchema(schema)
		}
	}
	return nil
}

// getOutputSchema returns the output schema if present.
func (b *CoreToolBuilder) getOutputSchema() any {
	if ta, ok := b.originalTool.(*tools.ToolAction); ok {
		if ta.OutputSchema != nil {
			return ta.OutputSchema
		}
	}
	if m, ok := b.originalTool.(map[string]any); ok {
		if schema, exists := m["outputSchema"]; exists {
			return resolveSchema(schema)
		}
	}
	return nil
}

// getResumeSchema returns the resume schema if present.
func (b *CoreToolBuilder) getResumeSchema() any {
	if ta, ok := b.originalTool.(*tools.ToolAction); ok {
		if ta.ResumeSchema != nil {
			return ta.ResumeSchema
		}
	}
	if m, ok := b.originalTool.(map[string]any); ok {
		if schema, exists := m["resumeSchema"]; exists {
			return resolveSchema(schema)
		}
	}
	return nil
}

// getSuspendSchema returns the suspend schema if present.
func (b *CoreToolBuilder) getSuspendSchema() any {
	if ta, ok := b.originalTool.(*tools.ToolAction); ok {
		if ta.SuspendSchema != nil {
			return ta.SuspendSchema
		}
	}
	if m, ok := b.originalTool.(map[string]any); ok {
		if schema, exists := m["suspendSchema"]; exists {
			return resolveSchema(schema)
		}
	}
	return nil
}

// resolveSchema handles the case where a schema might be a function that returns
// the actual schema (lazy evaluation pattern from TypeScript).
// In Go, we check if the value is a func() any and call it.
func resolveSchema(schema any) any {
	if fn, ok := schema.(func() any); ok {
		return fn()
	}
	return schema
}

// buildProviderTool builds a CoreTool for provider-defined tools (e.g. google.tools.googleSearch).
// Returns nil if the tool is not a provider-defined tool.
func (b *CoreToolBuilder) buildProviderTool(tool any) *tools.CoreTool {
	m, ok := tool.(map[string]any)
	if !ok {
		return nil
	}

	toolType, hasType := m["type"]
	if !hasType {
		return nil
	}
	typeStr, isStr := toolType.(string)
	if !isStr {
		return nil
	}
	if typeStr != "provider-defined" && typeStr != "provider" {
		return nil
	}

	id, hasID := m["id"]
	if !hasID {
		return nil
	}
	idStr, idIsStr := id.(string)
	if !idIsStr || !strings.Contains(idStr, ".") {
		return nil
	}

	// Get schema from provider-defined tool (v4 uses parameters, v5 uses inputSchema).
	var parameters any
	if p, exists := m["parameters"]; exists {
		parameters = resolveSchema(p)
	} else if is, exists := m["inputSchema"]; exists {
		parameters = resolveSchema(is)
	}

	// Get output schema.
	var outputSchema any
	if os, exists := m["outputSchema"]; exists {
		outputSchema = resolveSchema(os)
	}

	// Convert parameters to AI SDK Schema format.
	var processedParameters any
	if parameters != nil {
		if pm, ok := parameters.(map[string]any); ok {
			if _, hasJSON := pm["jsonSchema"]; hasJSON {
				// Already in AI SDK Schema format.
				processedParameters = parameters
			} else {
				processedParameters = convertZodSchemaToAISDKSchema(parameters)
			}
		} else {
			processedParameters = convertZodSchemaToAISDKSchema(parameters)
		}
	}

	// Convert output schema to AI SDK Schema format.
	var processedOutputSchema any
	if outputSchema != nil {
		if pm, ok := outputSchema.(map[string]any); ok {
			if _, hasJSON := pm["jsonSchema"]; hasJSON {
				processedOutputSchema = outputSchema
			} else {
				processedOutputSchema = convertZodSchemaToAISDKSchema(outputSchema)
			}
		} else {
			processedOutputSchema = convertZodSchemaToAISDKSchema(outputSchema)
		}
	}

	// Get args.
	var args map[string]any
	if a, exists := m["args"]; exists {
		if am, ok := a.(map[string]any); ok {
			args = am
		}
	}

	ct := &tools.CoreTool{
		Type:         tools.CoreToolTypeProviderDefined,
		ID:           idStr,
		Description:  getStringField(m, "description"),
		Parameters:   processedParameters,
		OutputSchema: processedOutputSchema,
		Args:         args,
	}

	// Build execute function if the original tool has one.
	if hasExecute(b.originalTool) {
		opts := b.options
		opts.Description = getStringField(m, "description")
		ct.Execute = b.createExecute(b.originalTool, opts, b.logType, nil)
	}

	// Copy toModelOutput if present.
	if ta, ok := b.originalTool.(*tools.ToolAction); ok && ta.ToModelOutput != nil {
		ct.ToModelOutput = ta.ToModelOutput
	}

	// Copy inputExamples if present.
	if ta, ok := b.originalTool.(*tools.ToolAction); ok && ta.InputExamples != nil {
		ct.InputExamples = ta.InputExamples
	}

	return ct
}

// createLogMessageOptions builds log message templates for tool execution.
func (b *CoreToolBuilder) createLogMessageOptions(opts LogOptions) LogMessageOptions {
	if opts.AgentName == "" {
		return LogMessageOptions{
			Start: fmt.Sprintf("Executing tool %s", opts.ToolName),
			Error: "Failed tool execution",
		}
	}

	prefix := fmt.Sprintf("[Agent:%s]", opts.AgentName)
	toolType := "tool"
	if opts.Type == LogTypeToolset {
		toolType = "toolset"
	}

	return LogMessageOptions{
		Start: fmt.Sprintf("%s - Executing %s %s", prefix, toolType, opts.ToolName),
		Error: fmt.Sprintf("%s - Failed %s execution", prefix, toolType),
	}
}

// createExecute builds the execute function for a tool.
// This is the core adapter that bridges AI SDK's (args, options) signature
// to Mastra's (inputData, context) signature.
func (b *CoreToolBuilder) createExecute(
	tool any,
	options ToolOptions,
	logType LogType,
	processedSchema any,
) func(args any, execOptions *tools.MastraToolInvocationOptions) (any, error) {

	logModelObject := map[string]any{}
	if options.Model != nil {
		logModelObject["modelId"] = options.Model.ModelID
		logModelObject["provider"] = options.Model.Provider
		logModelObject["specificationVersion"] = options.Model.SpecificationVersion
	}

	logMsgs := b.createLogMessageOptions(LogOptions{
		AgentName: options.AgentName,
		ToolName:  options.Name,
		Type:      logType,
	})

	// execFunction is the inner execution function that handles tracing and context building.
	execFunction := func(args any, execOptions *tools.MastraToolInvocationOptions) (any, error) {
		// Prefer execution-time tracingContext, fall back to build-time context.
		tracingContext := options.TracingContext
		if execOptions != nil && execOptions.ObservabilityContext != nil {
			// Use execution-time tracing context if available.
			// In the full implementation this would extract TracingContext from ObservabilityContext.
		}

		// Create tool span if we have a current span available.
		var toolSpan tracingtypes.Span
		if tracingContext != nil && tracingContext.CurrentSpan != nil {
			toolSpan = tracingContext.CurrentSpan.CreateChildSpan(tracingtypes.ChildSpanOptions{
				CreateBaseOptions: tracingtypes.CreateBaseOptions{
					Type:       tracingtypes.SpanTypeToolCall,
					Name:       fmt.Sprintf("tool: '%s'", options.Name),
					EntityType: entityTypePtr(tracingtypes.EntityTypeTool),
					EntityID:   options.Name,
					EntityName: options.Name,
					Attributes: map[string]any{
						"toolDescription": options.Description,
						"toolType":        stringOrDefault(string(logType), "tool"),
					},
					TracingPolicy: options.TracingPolicy,
				},
				Input: args,
			})
		}

		var result any
		var suspendData any
		var execErr error

		if tools.IsVercelTool(tool) {
			// Handle Vercel tools (AI SDK tools).
			result, execErr = executeVercelTool(tool, args, execOptions)
			if execErr != nil {
				if toolSpan != nil {
					toolSpan.Error(tracingtypes.ErrorSpanOptions{Error: execErr, Attributes: map[string]any{"success": false}})
				}
				return nil, execErr
			}
		} else {
			// Handle Mastra tools - wrap mastra instance with tracing context.
			wrappedMastra := options.Mastra
			if options.Mastra != nil {
				wrappedMastra = wrapMastra(options.Mastra, map[string]any{"currentSpan": toolSpan})
			}

			resumeSchema := b.getResumeSchema()

			// Build the request context, preferring execution-time over build-time.
			reqCtx := options.RequestContext
			if execOptions != nil && execOptions.RequestContext != nil {
				reqCtx = execOptions.RequestContext
			}
			if reqCtx == nil {
				reqCtx = requestcontext.NewRequestContext()
			}

			// Build the workspace, preferring execution-time over build-time.
			workspace := options.Workspace
			if execOptions != nil && execOptions.Workspace != nil {
				workspace = execOptions.Workspace
			}

			// Build output writer.
			var outputWriter tools.OutputWriter
			if options.OutputWriter != nil {
				outputWriter = options.OutputWriter
			}
			if execOptions != nil && execOptions.OutputWriter != nil {
				outputWriter = execOptions.OutputWriter
			}

			// Build the tool stream writer.
			toolCallID := ""
			var messages []any
			if execOptions != nil {
				// Extract toolCallId and messages from MastraToolInvocationOptions.
				// These would come from the AI SDK's ToolCallOptions embedded in the options.
			}

			writer := tools.NewToolStream(tools.ToolStreamConfig{
				Prefix: "tool",
				CallID: toolCallID,
				Name:   options.Name,
				RunID:  options.RunID,
			}, outputWriter)

			// Build the suspend function.
			suspendFn := func(suspendPayload any, suspendOpts *tools.SuspendOptions) error {
				suspendData = suspendPayload
				if execOptions != nil && execOptions.Suspend != nil {
					// Build new suspend options with resume schema if available.
					_, _ = execOptions.Suspend(suspendPayload, suspendOpts)
				}
				return nil
			}

			// Get resume data from execution options.
			var resumeData any
			if execOptions != nil {
				resumeData = execOptions.ResumeData
			}

			// Build base context.
			baseCtx := &tools.ToolExecutionContext{
				ObservabilityContext: createObservabilityContext(map[string]any{"currentSpan": toolSpan}),
				Mastra:              wrappedMastra,
				RequestContext:      reqCtx,
				Workspace:           workspace,
				Writer:              writer,
			}

			// Determine execution context type.
			isAgentExecution := (toolCallID != "" && len(messages) > 0) ||
				(options.AgentName != "" && options.ThreadID != "" && options.WorkflowID == "")
			isWorkflowExecution := !isAgentExecution && (options.Workflow != nil || options.WorkflowID != "")

			if isAgentExecution {
				baseCtx.Agent = &tools.AgentToolExecutionContext{
					ToolCallID: toolCallID,
					Messages:   messages,
					Suspend:    suspendFn,
					ResumeData: resumeData,
					ThreadID:   options.ThreadID,
					ResourceID: options.ResourceID,
				}
			} else if isWorkflowExecution {
				baseCtx.Workflow = &tools.WorkflowToolExecutionContext{
					RunID:      options.RunID,
					WorkflowID: options.WorkflowID,
					State:      options.State,
					SetState:   options.SetState,
					Suspend:    suspendFn,
					ResumeData: resumeData,
				}
			} else if execOptions != nil && execOptions.MCP != nil {
				baseCtx.MCP = execOptions.MCP
			}

			// Validate resume data if present.
			if resumeData != nil {
				resumeValidation := validateToolInput(resumeSchema, resumeData, options.Name)
				if resumeValidation.Error != nil {
					lgr := options.Logger
					if lgr == nil {
						lgr = b.Logger()
					}
					lgr.Warn(resumeValidation.Error.Message)
					if toolSpan != nil {
						toolSpan.End(&tracingtypes.EndSpanOptions{
							Output:     resumeValidation.Error,
							Attributes: map[string]any{"success": false},
						})
					}
					return resumeValidation.Error, nil
				}
			}

			// Execute the Mastra tool.
			result, execErr = executeMastraTool(tool, args, baseCtx)
			if execErr != nil {
				if toolSpan != nil {
					toolSpan.Error(tracingtypes.ErrorSpanOptions{Error: execErr, Attributes: map[string]any{"success": false}})
				}
				return nil, execErr
			}
		}

		// Validate suspend data if suspend was called.
		if suspendData != nil {
			suspendSchema := b.getSuspendSchema()
			suspendValidation := validateToolSuspendData(suspendSchema, suspendData, options.Name)
			if suspendValidation.Error != nil {
				lgr := options.Logger
				if lgr == nil {
					lgr = b.Logger()
				}
				lgr.Warn(suspendValidation.Error.Message)
				if toolSpan != nil {
					toolSpan.End(&tracingtypes.EndSpanOptions{
						Output:     suspendValidation.Error,
						Attributes: map[string]any{"success": false},
					})
				}
				return suspendValidation.Error, nil
			}
		}

		// Skip validation if suspend was called without a result.
		shouldSkipValidation := result == nil && suspendData != nil
		if shouldSkipValidation {
			if toolSpan != nil {
				toolSpan.End(&tracingtypes.EndSpanOptions{
					Output:     result,
					Attributes: map[string]any{"success": true},
				})
			}
			return result, nil
		}

		// Validate output for Vercel/AI SDK tools.
		// Mastra tools handle their own validation in Tool.Execute().
		if tools.IsVercelTool(tool) {
			outputSchema := b.getOutputSchema()
			outputValidation := validateToolOutput(outputSchema, result, options.Name, false)
			if outputValidation.Error != nil {
				lgr := options.Logger
				if lgr == nil {
					lgr = b.Logger()
				}
				lgr.Warn(outputValidation.Error.Message)
				if toolSpan != nil {
					toolSpan.End(&tracingtypes.EndSpanOptions{
						Output:     outputValidation.Error,
						Attributes: map[string]any{"success": false},
					})
				}
				return outputValidation.Error, nil
			}
			result = outputValidation.Data
		}

		// Return result.
		if toolSpan != nil {
			toolSpan.End(&tracingtypes.EndSpanOptions{
				Output:     result,
				Attributes: map[string]any{"success": true},
			})
		}
		return result, nil
	}

	// The outer function handles logging, input validation, and error wrapping.
	return func(args any, execOptions *tools.MastraToolInvocationOptions) (any, error) {
		lgr := options.Logger
		if lgr == nil {
			lgr = b.Logger()
		}

		lgr.Debug(logMsgs.Start, map[string]any{"model": logModelObject, "args": args})

		// Validate input parameters if schema exists.
		parameters := processedSchema
		if parameters == nil {
			parameters = b.getParameters()
		}
		validation := validateToolInput(parameters, args, options.Name)

		if validation.Error != nil {
			// Check if error is only about suspendedToolRunId (ignore when no resumeData).
			ignoreErr := false
			if strings.Contains(validation.Error.Message, "suspendedToolRunId: Required") {
				if argsMap, ok := args.(map[string]any); ok {
					if _, hasResume := argsMap["resumeData"]; !hasResume {
						ignoreErr = true
					}
				} else {
					ignoreErr = true
				}
			}
			if !ignoreErr {
				lgr.Warn(validation.Error.Message)
				return validation.Error, nil
			}
		}
		// Use validated/transformed data.
		args = validation.Data

		// Execute the tool.
		result, err := execFunction(args, execOptions)
		if err != nil {
			argsJSON, _ := json.Marshal(args)
			modelID := ""
			if options.Model != nil {
				modelID = options.Model.ModelID
			}
			mastraErr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "TOOL_EXECUTION_FAILED",
				Domain:   mastraerror.ErrorDomainTool,
				Category: mastraerror.ErrorCategoryUser,
				Details: map[string]any{
					"errorMessage": fmt.Sprintf("%v", err),
					"argsJson":     string(argsJSON),
					"model":        modelID,
				},
			}, err)
			lgr.TrackException(mastraErr)
			lgr.Error(logMsgs.Error, map[string]any{"model": logModelObject, "error": mastraErr, "args": args})
			return nil, mastraErr
		}

		return result, nil
	}
}

// ---------------------------------------------------------------------------
// Build methods
// ---------------------------------------------------------------------------

// Build converts the original tool into a CoreTool.
func (b *CoreToolBuilder) Build() tools.CoreTool {
	// Check if this is a provider-defined tool first.
	providerTool := b.buildProviderTool(b.originalTool)
	if providerTool != nil {
		return *providerTool
	}

	model := b.options.Model

	// Build schema compatibility layers.
	var schemaCompatLayers []SchemaCompatLayer
	if model != nil {
		// In the full implementation, these would be real compatibility layers:
		// - OpenAIReasoningSchemaCompatLayer
		// - OpenAISchemaCompatLayer
		// - GoogleSchemaCompatLayer
		// - AnthropicSchemaCompatLayer
		// - DeepSeekSchemaCompatLayer
		// - MetaSchemaCompatLayer
		// Stub: Zod schema-compat is TS-specific; not applicable in Go.
		_ = schemaCompatLayers
	}

	// Apply schema compatibility to get both the transformed schema (for validation)
	// and the AI SDK Schema (for the LLM).
	var processedZodSchema any
	var processedSchema any

	originalSchema := b.getParameters()

	// Find the first applicable compatibility layer.
	var applicableLayer SchemaCompatLayer
	for _, layer := range schemaCompatLayers {
		if layer.ShouldApply() {
			applicableLayer = layer
			break
		}
	}

	if applicableLayer != nil && originalSchema != nil {
		processedZodSchema = applicableLayer.ProcessZodType(originalSchema)
		processedSchema = applyCompatLayer(originalSchema, schemaCompatLayers, "aiSdkSchema")
	} else if originalSchema != nil {
		processedZodSchema = originalSchema
		processedSchema = applyCompatLayer(originalSchema, schemaCompatLayers, "aiSdkSchema")
	}

	// Process output schema (no compat layers since it's never sent to the LLM).
	var processedOutputSchema any
	if outputSchema := b.getOutputSchema(); outputSchema != nil {
		processedOutputSchema = applyCompatLayer(outputSchema, nil, "aiSdkSchema")
	}

	// Map AI SDK's needsApproval to requireApproval.
	requireApproval := b.options.RequireApproval
	if tools.IsVercelTool(b.originalTool) {
		if m, ok := b.originalTool.(map[string]any); ok {
			if na, exists := m["needsApproval"]; exists {
				if boolVal, isBool := na.(bool); isBool {
					requireApproval = boolVal
				}
				// Function-based needsApproval would set requireApproval = true.
				// The function itself would be stored separately for per-call evaluation.
			}
		}
	}

	// Build the execute function.
	var executeFn func(args any, options *tools.MastraToolInvocationOptions) (any, error)
	if hasExecute(b.originalTool) {
		opts := b.options
		opts.Description = getDescription(b.originalTool)
		executeFn = b.createExecute(b.originalTool, opts, b.logType, processedZodSchema)
	}

	// Build the final CoreTool.
	ct := tools.CoreTool{
		Type:            tools.CoreToolTypeFunction,
		Description:     getDescription(b.originalTool),
		Execute:         executeFn,
	}

	// Set parameters (default to empty if nil).
	if processedSchema != nil {
		ct.Parameters = processedSchema
	}

	ct.OutputSchema = processedOutputSchema

	// Set ID if available.
	if ta, ok := b.originalTool.(*tools.ToolAction); ok {
		ct.ID = ta.ID
	} else if m, ok := b.originalTool.(map[string]any); ok {
		if id, exists := m["id"]; exists {
			if idStr, isStr := id.(string); isStr {
				ct.ID = idStr
			}
		}
	}

	// Set requireApproval via a wrapper if needed.
	// In TypeScript this is set directly on the definition; in Go we track it
	// through the options since CoreTool doesn't have a RequireApproval field.
	_ = requireApproval

	// Copy provider options.
	if ta, ok := b.originalTool.(*tools.ToolAction); ok {
		ct.ProviderOptions = ta.ProviderOptions
		ct.MCP = ta.MCP
		ct.ToModelOutput = ta.ToModelOutput
		ct.InputExamples = ta.InputExamples
		ct.OnInputStart = ta.OnInputStart
		ct.OnInputDelta = ta.OnInputDelta
		ct.OnInputAvailable = ta.OnInputAvailable
		ct.OnOutput = ta.OnOutput
	} else if m, ok := b.originalTool.(map[string]any); ok {
		if po, exists := m["providerOptions"]; exists {
			if poMap, ok := po.(map[string]map[string]any); ok {
				ct.ProviderOptions = poMap
			}
		}
		if mcp, exists := m["mcp"]; exists {
			if mcpProps, ok := mcp.(*tools.MCPToolProperties); ok {
				ct.MCP = mcpProps
			}
		}
	}

	return ct
}

// BuildV5 converts the original tool into a VercelToolV5-compatible format.
// It wraps Build() and adds V5-specific properties.
func (b *CoreToolBuilder) BuildV5() map[string]any {
	builtTool := b.Build()

	if builtTool.Parameters == nil {
		panic("Tool parameters are required")
	}

	base := map[string]any{
		"type":        builtTool.Type,
		"description": builtTool.Description,
		"parameters":  builtTool.Parameters,
		"inputSchema": builtTool.Parameters,
	}

	if builtTool.ID != "" {
		base["id"] = builtTool.ID
	}
	if builtTool.OutputSchema != nil {
		base["outputSchema"] = builtTool.OutputSchema
	}
	if builtTool.Execute != nil {
		base["execute"] = builtTool.Execute
	}

	// Copy lifecycle callbacks from original tool.
	if ta, ok := b.originalTool.(*tools.ToolAction); ok {
		if ta.OnInputStart != nil {
			base["onInputStart"] = ta.OnInputStart
		}
		if ta.OnInputDelta != nil {
			base["onInputDelta"] = ta.OnInputDelta
		}
		if ta.OnInputAvailable != nil {
			base["onInputAvailable"] = ta.OnInputAvailable
		}
		if ta.OnOutput != nil {
			base["onOutput"] = ta.OnOutput
		}
	}

	// For provider-defined tools, exclude execute and add name as per V5 spec.
	if builtTool.Type == tools.CoreToolTypeProviderDefined {
		delete(base, "execute")
		delete(base, "parameters")
		name := builtTool.ID
		if idx := strings.Index(builtTool.ID, "."); idx >= 0 {
			name = builtTool.ID[idx+1:]
		}
		base["name"] = name
		base["args"] = builtTool.Args
	}

	return base
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

// getDescription extracts the description from a tool (ToolAction or map).
func getDescription(tool any) string {
	if ta, ok := tool.(*tools.ToolAction); ok {
		return ta.Description
	}
	if m, ok := tool.(map[string]any); ok {
		return getStringField(m, "description")
	}
	return ""
}

// getStringField safely extracts a string field from a map.
func getStringField(m map[string]any, key string) string {
	if v, exists := m[key]; exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// hasExecute checks whether a tool has an execute function.
func hasExecute(tool any) bool {
	if ta, ok := tool.(*tools.ToolAction); ok {
		return ta.Execute != nil
	}
	if m, ok := tool.(map[string]any); ok {
		if exec, exists := m["execute"]; exists && exec != nil {
			return true
		}
	}
	return false
}

// executeVercelTool executes a Vercel/AI SDK tool.
func executeVercelTool(tool any, args any, execOptions *tools.MastraToolInvocationOptions) (any, error) {
	m, ok := tool.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("vercel tool is not a map")
	}
	exec, exists := m["execute"]
	if !exists || exec == nil {
		return nil, nil
	}
	if fn, ok := exec.(func(any, any) (any, error)); ok {
		return fn(args, execOptions)
	}
	return nil, fmt.Errorf("vercel tool execute is not callable")
}

// executeMastraTool executes a Mastra ToolAction.
func executeMastraTool(tool any, args any, ctx *tools.ToolExecutionContext) (any, error) {
	if ta, ok := tool.(*tools.ToolAction); ok {
		if ta.Execute == nil {
			return nil, nil
		}
		return ta.Execute(args, ctx)
	}
	// Fallback for map-based tools.
	if m, ok := tool.(map[string]any); ok {
		if exec, exists := m["execute"]; exists && exec != nil {
			if fn, ok := exec.(func(any, *tools.ToolExecutionContext) (any, error)); ok {
				return fn(args, ctx)
			}
		}
	}
	return nil, fmt.Errorf("tool does not have a callable execute function")
}

// stringOrDefault returns s if non-empty, otherwise the default value.
func stringOrDefault(s, def string) string {
	if s != "" {
		return s
	}
	return def
}

// entityTypePtr returns a pointer to an EntityType value.
func entityTypePtr(et tracingtypes.EntityType) *tracingtypes.EntityType {
	return &et
}
