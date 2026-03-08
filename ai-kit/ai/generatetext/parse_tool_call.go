// Ported from: packages/ai/src/generate-text/parse-tool-call.ts
package generatetext

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseToolCallOptions contains the parameters for parsing a tool call.
type ParseToolCallOptions struct {
	ToolCall       LanguageModelV4ToolCall
	Tools          ToolSet
	RepairToolCall ToolCallRepairFunction
	System         interface{} // string | SystemModelMessage | []SystemModelMessage | nil
	Messages       []ModelMessage
}

// ParseToolCall parses a raw tool call from the model into a typed ToolCall.
func ParseToolCall(opts ParseToolCallOptions) (ToolCall, error) {
	tc, err := doParseToolCallOuter(opts)
	if err != nil {
		// On error, try to produce an invalid tool call with parsed input
		parsedInput := tryParseJSON(opts.ToolCall.Input)

		return ToolCall{
			Type:             "tool-call",
			ToolCallID:       opts.ToolCall.ToolCallID,
			ToolName:         opts.ToolCall.ToolName,
			Input:            parsedInput,
			Dynamic:          true,
			Invalid:          true,
			Error:            err,
			Title:            getToolTitle(opts.Tools, opts.ToolCall.ToolName),
			ProviderExecuted: opts.ToolCall.ProviderExecuted,
			ProviderMetadata: opts.ToolCall.ProviderMetadata,
		}, nil
	}
	return tc, nil
}

func doParseToolCallOuter(opts ParseToolCallOptions) (ToolCall, error) {
	if opts.Tools == nil {
		// Provider-executed dynamic tools are not part of our list of tools
		if opts.ToolCall.ProviderExecuted && opts.ToolCall.Dynamic {
			return parseProviderExecutedDynamicToolCall(opts.ToolCall)
		}
		return ToolCall{}, &NoSuchToolError{ToolName: opts.ToolCall.ToolName}
	}

	tc, err := doParseToolCall(opts.ToolCall, opts.Tools)
	if err != nil {
		// Try repair if applicable
		if opts.RepairToolCall == nil {
			return ToolCall{}, err
		}
		if !IsNoSuchToolError(err) && !IsInvalidToolInputError(err) {
			return ToolCall{}, err
		}

		repairedToolCall, repairErr := opts.RepairToolCall(ToolCallRepairOptions{
			System:   opts.System,
			Messages: opts.Messages,
			ToolCall: opts.ToolCall,
			Tools:    opts.Tools,
			InputSchema: func(toolName string) (JSONSchema7, error) {
				tool, ok := opts.Tools[toolName]
				if !ok {
					return nil, fmt.Errorf("tool %q not found", toolName)
				}
				schema, ok := tool.InputSchema.(map[string]interface{})
				if !ok {
					return map[string]interface{}{}, nil
				}
				return schema, nil
			},
			Error: err,
		})
		if repairErr != nil {
			return ToolCall{}, &ToolCallRepairError{
				Cause:         repairErr,
				OriginalError: err,
			}
		}
		if repairedToolCall == nil {
			return ToolCall{}, err
		}
		return doParseToolCall(*repairedToolCall, opts.Tools)
	}
	return tc, nil
}

func parseProviderExecutedDynamicToolCall(toolCall LanguageModelV4ToolCall) (ToolCall, error) {
	input := strings.TrimSpace(toolCall.Input)
	var parsed interface{}
	if input == "" {
		parsed = map[string]interface{}{}
	} else {
		if err := json.Unmarshal([]byte(input), &parsed); err != nil {
			return ToolCall{}, &InvalidToolInputError{
				ToolName:  toolCall.ToolName,
				ToolInput: toolCall.Input,
				Cause:     err,
			}
		}
	}

	return ToolCall{
		Type:             "tool-call",
		ToolCallID:       toolCall.ToolCallID,
		ToolName:         toolCall.ToolName,
		Input:            parsed,
		ProviderExecuted: true,
		Dynamic:          true,
		ProviderMetadata: toolCall.ProviderMetadata,
	}, nil
}

func doParseToolCall(toolCall LanguageModelV4ToolCall, tools ToolSet) (ToolCall, error) {
	toolName := toolCall.ToolName

	tool, ok := tools[toolName]
	if !ok {
		// Provider-executed dynamic tools are not part of our list of tools
		if toolCall.ProviderExecuted && toolCall.Dynamic {
			return parseProviderExecutedDynamicToolCall(toolCall)
		}
		availableTools := make([]string, 0, len(tools))
		for k := range tools {
			availableTools = append(availableTools, k)
		}
		return ToolCall{}, &NoSuchToolError{
			ToolName:       toolCall.ToolName,
			AvailableTools: availableTools,
		}
	}

	// Parse input: when empty, try empty object
	input := strings.TrimSpace(toolCall.Input)
	var parsed interface{}
	if input == "" {
		parsed = map[string]interface{}{}
	} else {
		if err := json.Unmarshal([]byte(input), &parsed); err != nil {
			return ToolCall{}, &InvalidToolInputError{
				ToolName:  toolName,
				ToolInput: toolCall.Input,
				Cause:     err,
			}
		}
	}

	// TODO: validate parsed input against tool.InputSchema

	isDynamic := tool.Type == "dynamic"

	return ToolCall{
		Type:             "tool-call",
		ToolCallID:       toolCall.ToolCallID,
		ToolName:         toolName,
		Input:            parsed,
		ProviderExecuted: toolCall.ProviderExecuted,
		ProviderMetadata: toolCall.ProviderMetadata,
		Dynamic:          isDynamic,
		Title:            tool.Title,
	}, nil
}

func tryParseJSON(text string) interface{} {
	var result interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return text
	}
	return result
}

func getToolTitle(tools ToolSet, toolName string) string {
	if tools == nil {
		return ""
	}
	if tool, ok := tools[toolName]; ok {
		return tool.Title
	}
	return ""
}

// --- Error types ---

// NoSuchToolError indicates the model called a tool that does not exist.
type NoSuchToolError struct {
	ToolName       string
	AvailableTools []string
}

func (e *NoSuchToolError) Error() string {
	return fmt.Sprintf("no such tool: %s", e.ToolName)
}

// IsNoSuchToolError checks whether the given error is a NoSuchToolError.
func IsNoSuchToolError(err error) bool {
	_, ok := err.(*NoSuchToolError)
	return ok
}

// InvalidToolInputError indicates the tool call input could not be parsed.
type InvalidToolInputError struct {
	ToolName  string
	ToolInput string
	Cause     error
}

func (e *InvalidToolInputError) Error() string {
	return fmt.Sprintf("invalid tool input for %s: %v", e.ToolName, e.Cause)
}

func (e *InvalidToolInputError) Unwrap() error { return e.Cause }

// IsInvalidToolInputError checks whether the given error is an InvalidToolInputError.
func IsInvalidToolInputError(err error) bool {
	_, ok := err.(*InvalidToolInputError)
	return ok
}

// ToolCallRepairError indicates a tool call repair attempt failed.
type ToolCallRepairError struct {
	Cause         error
	OriginalError error
}

func (e *ToolCallRepairError) Error() string {
	return fmt.Sprintf("tool call repair failed: %v (original: %v)", e.Cause, e.OriginalError)
}

func (e *ToolCallRepairError) Unwrap() error { return e.Cause }
