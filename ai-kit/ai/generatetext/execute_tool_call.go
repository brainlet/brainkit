// Ported from: packages/ai/src/generate-text/execute-tool-call.ts
package generatetext

import (
	"time"
)

// ExecuteToolCallOptions contains the parameters for executing a tool call.
type ExecuteToolCallOptions struct {
	ToolCall               ToolCall
	Tools                  ToolSet
	Messages               []ModelMessage
	AbortSignal            <-chan struct{}
	ExperimentalContext    interface{}
	StepNumber             *int
	Model                  *ModelInfo
	OnPreliminaryToolResult func(result ToolResult)
	OnToolCallStart        []func(event OnToolCallStartEvent)
	OnToolCallFinish       []func(event OnToolCallFinishEvent)
}

// ExecuteToolCall executes a single tool call and manages its lifecycle callbacks.
//
// Returns the tool output (result or error), or nil if the tool has no execute function.
func ExecuteToolCall(opts ExecuteToolCallOptions) (ToolOutput, error) {
	toolName := opts.ToolCall.ToolName
	toolCallID := opts.ToolCall.ToolCallID
	input := opts.ToolCall.Input

	tool, ok := opts.Tools[toolName]
	if !ok || tool.Execute == nil {
		return nil, nil
	}

	baseEvent := OnToolCallStartEvent{
		StepNumber:          opts.StepNumber,
		Model:               opts.Model,
		ToolCall:            opts.ToolCall,
		Messages:            opts.Messages,
		AbortSignal:         opts.AbortSignal,
		ExperimentalContext: opts.ExperimentalContext,
	}

	// Notify onToolCallStart
	for _, cb := range opts.OnToolCallStart {
		if cb != nil {
			cb(baseEvent)
		}
	}

	startTime := time.Now()

	output, execErr := tool.Execute(input, ToolExecuteOptions{
		ToolCallID:          toolCallID,
		Messages:            opts.Messages,
		AbortSignal:         opts.AbortSignal,
		ExperimentalContext: opts.ExperimentalContext,
	})

	durationMs := float64(time.Since(startTime).Milliseconds())

	if execErr != nil {
		// Notify onToolCallFinish with error
		finishEvent := OnToolCallFinishEvent{
			StepNumber:          opts.StepNumber,
			Model:               opts.Model,
			ToolCall:            opts.ToolCall,
			Messages:            opts.Messages,
			AbortSignal:         opts.AbortSignal,
			DurationMs:          durationMs,
			ExperimentalContext: opts.ExperimentalContext,
			Success:             false,
			Error:               execErr,
		}
		for _, cb := range opts.OnToolCallFinish {
			if cb != nil {
				cb(finishEvent)
			}
		}

		toolError := &ToolError{
			Type:       "tool-error",
			ToolCallID: toolCallID,
			ToolName:   toolName,
			Input:      input,
			Error:      execErr,
			Dynamic:    tool.Type == "dynamic",
		}
		if opts.ToolCall.ProviderMetadata != nil {
			toolError.ProviderMetadata = opts.ToolCall.ProviderMetadata
		}
		return toolError, nil
	}

	// Notify onToolCallFinish with success
	finishEvent := OnToolCallFinishEvent{
		StepNumber:          opts.StepNumber,
		Model:               opts.Model,
		ToolCall:            opts.ToolCall,
		Messages:            opts.Messages,
		AbortSignal:         opts.AbortSignal,
		DurationMs:          durationMs,
		ExperimentalContext: opts.ExperimentalContext,
		Success:             true,
		Output:              output,
	}
	for _, cb := range opts.OnToolCallFinish {
		if cb != nil {
			cb(finishEvent)
		}
	}

	toolResult := &ToolResult{
		Type:       "tool-result",
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Input:      input,
		Output:     output,
		Dynamic:    tool.Type == "dynamic",
	}
	if opts.ToolCall.ProviderMetadata != nil {
		toolResult.ProviderMetadata = opts.ToolCall.ProviderMetadata
	}
	return toolResult, nil
}
