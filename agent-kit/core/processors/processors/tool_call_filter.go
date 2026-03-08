// Ported from: packages/core/src/processors/processors/tool-call-filter.ts
package concreteprocessors

import (
	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// V2ToolInvocationPart
// ---------------------------------------------------------------------------

// v2ToolInvocation is a local type for tool invocation data in MastraDBMessage format 2.
type v2ToolInvocation struct {
	ToolName   string `json:"toolName"`
	ToolCallID string `json:"toolCallId"`
	Args       any    `json:"args,omitempty"`
	Result     any    `json:"result,omitempty"`
	State      string `json:"state"` // "call" | "result"
}

// ---------------------------------------------------------------------------
// ToolCallFilter
// ---------------------------------------------------------------------------

// ToolCallFilter filters out tool calls and results from messages.
// By default (with no arguments), excludes all tool calls and their results.
// Can be configured to exclude only specific tools by name.
type ToolCallFilter struct {
	processors.BaseProcessor
	exclude    []string
	excludeAll bool
}

// ToolCallFilterOptions holds configuration for ToolCallFilter.
type ToolCallFilterOptions struct {
	// Exclude is a list of specific tool names to exclude.
	// If nil or not provided, all tool calls are excluded.
	Exclude []string
}

// NewToolCallFilter creates a new ToolCallFilter.
func NewToolCallFilter(opts *ToolCallFilterOptions) *ToolCallFilter {
	f := &ToolCallFilter{
		BaseProcessor: processors.NewBaseProcessor("tool-call-filter", "ToolCallFilter"),
	}
	if opts == nil || opts.Exclude == nil {
		f.excludeAll = true
	} else {
		f.exclude = opts.Exclude
	}
	return f
}

// hasToolInvocations checks if a message has tool invocation parts.
func hasToolInvocations(message processors.MastraDBMessage) bool {
	if len(message.Content.Parts) == 0 {
		return false
	}
	for _, part := range message.Content.Parts {
		if part.Type == "tool-invocation" {
			return true
		}
	}
	return false
}

// isExcluded checks if a tool name is in the exclude list.
func (f *ToolCallFilter) isExcluded(toolName string) bool {
	for _, name := range f.exclude {
		if name == toolName {
			return true
		}
	}
	return false
}

// ProcessInput filters tool calls from messages.
func (f *ToolCallFilter) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	messages := args.Messages

	// Case 1: Exclude all tool calls and tool results.
	if f.excludeAll {
		var result []processors.MastraDBMessage
		for _, message := range messages {
			if !hasToolInvocations(message) {
				result = append(result, message)
				continue
			}

			// Filter out tool invocation parts.
			var nonToolParts []processors.MessagePart
			for _, part := range message.Content.Parts {
				if part.Type != "tool-invocation" {
					nonToolParts = append(nonToolParts, part)
				}
			}

			// If no parts remain after filtering, skip the message.
			if len(nonToolParts) == 0 {
				continue
			}

			// Return message with filtered parts.
			msg := message
			msg.Content.Parts = nonToolParts
			result = append(result, msg)
		}
		return result, nil, nil, nil
	}

	// Case 2: Exclude specific tools by name.
	if len(f.exclude) > 0 {
		// First pass: identify excluded tool call IDs.
		excludedToolCallIDs := make(map[string]bool)
		for _, message := range messages {
			for _, part := range message.Content.Parts {
				if part.Type == "tool-invocation" && part.ToolInvocationData != nil {
					if f.isExcluded(part.ToolInvocationData.ToolName) {
						excludedToolCallIDs[part.ToolInvocationData.ToolCallID] = true
					}
				}
			}
		}

		// Second pass: filter out excluded tool invocation parts.
		var filteredMessages []processors.MastraDBMessage
		for _, message := range messages {
			if !hasToolInvocations(message) {
				filteredMessages = append(filteredMessages, message)
				continue
			}

			var filteredParts []processors.MessagePart
			for _, part := range message.Content.Parts {
				if part.Type != "tool-invocation" {
					filteredParts = append(filteredParts, part)
					continue
				}

				if part.ToolInvocationData == nil {
					filteredParts = append(filteredParts, part)
					continue
				}

				inv := part.ToolInvocationData

				// Exclude if it's a call for an excluded tool.
				if inv.State == "call" && f.isExcluded(inv.ToolName) {
					continue
				}

				// Exclude if it's a result for an excluded tool call.
				if inv.State == "result" && excludedToolCallIDs[inv.ToolCallID] {
					continue
				}

				// Also exclude results by tool name if no corresponding call exists.
				if inv.State == "result" && f.isExcluded(inv.ToolName) {
					continue
				}

				filteredParts = append(filteredParts, part)
			}

			// If no parts remain, skip the message.
			if len(filteredParts) == 0 {
				continue
			}

			msg := message
			msg.Content.Parts = filteredParts
			filteredMessages = append(filteredMessages, msg)
		}
		return filteredMessages, nil, nil, nil
	}

	// Case 3: Empty exclude list, return original messages.
	return messages, nil, nil, nil
}

// ProcessInputStep is not implemented for this processor.
func (f *ToolCallFilter) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ProcessOutputStream is not implemented for this processor.
func (f *ToolCallFilter) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputResult is not implemented for this processor.
func (f *ToolCallFilter) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (f *ToolCallFilter) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}
