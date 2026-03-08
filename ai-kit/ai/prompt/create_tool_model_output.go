// Ported from: packages/ai/src/prompt/create-tool-model-output.ts
package prompt

import "encoding/json"

// JSONValue represents any JSON-serializable value.
// TODO: import from brainlink/experiments/ai-kit/provider once it exists
type JSONValue = interface{}

// ToModelOutputFunc is the function signature for custom tool output conversion.
type ToModelOutputFunc func(args ToModelOutputArgs) (ToolResultOutput, error)

// ToModelOutputArgs holds the arguments for a tool's toModelOutput function.
type ToModelOutputArgs struct {
	ToolCallID string
	Input      interface{}
	Output     interface{}
}

// ToolWithModelOutput represents a tool that may have a custom toModelOutput function.
type ToolWithModelOutput struct {
	ToModelOutput ToModelOutputFunc
}

// CreateToolModelOutput creates the tool result output based on the tool configuration.
func CreateToolModelOutput(
	toolCallID string,
	input interface{},
	output interface{},
	tool *ToolWithModelOutput,
	errorMode string, // "none", "text", or "json"
) (ToolResultOutput, error) {
	if errorMode == "text" {
		return ToolResultOutput{
			Type:  "error-text",
			Value: getErrorMessage(output),
		}, nil
	} else if errorMode == "json" {
		return ToolResultOutput{
			Type:  "error-json",
			Value: toJSONValue(output),
		}, nil
	}

	if tool != nil && tool.ToModelOutput != nil {
		return tool.ToModelOutput(ToModelOutputArgs{
			ToolCallID: toolCallID,
			Input:      input,
			Output:     output,
		})
	}

	if s, ok := output.(string); ok {
		return ToolResultOutput{
			Type:  "text",
			Value: s,
		}, nil
	}

	return ToolResultOutput{
		Type:  "json",
		Value: toJSONValue(output),
	}, nil
}

// getErrorMessage extracts a human-readable message from an unknown error value.
func getErrorMessage(value interface{}) string {
	if value == nil {
		return "unknown error"
	}
	if s, ok := value.(string); ok {
		return s
	}
	if err, ok := value.(error); ok {
		return err.Error()
	}
	// Try JSON stringification
	data, err := json.Marshal(value)
	if err != nil {
		return "unknown error"
	}
	return string(data)
}

// toJSONValue converts an unknown value to a JSONValue, converting undefined/nil to null.
func toJSONValue(value interface{}) JSONValue {
	if value == nil {
		return nil
	}
	return value
}
