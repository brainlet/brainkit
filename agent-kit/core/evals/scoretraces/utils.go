// Ported from: packages/core/src/evals/scoreTraces/utils.ts
package scoretraces

import (
	"errors"
	"sort"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/evals"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	obstorage "github.com/brainlet/brainkit/agent-kit/core/storage/domains/observability"
)

// ============================================================================
// Real types imported from storage/domains/observability
// ============================================================================

// SpanRecord is the real span record type from storage/domains/observability.
// Previously this was a local stub; now uses the canonical type directly.
// The real type has additional fields beyond what this package uses
// (UserID, OrganizationID, ResourceID, RunID, SessionID, ThreadID, RequestID,
// Environment, Source, ServiceName, Scope, Tags, CreatedAt, UpdatedAt).
// NOTE: The SpanType field is named "SpanTyp" (type observability.SpanType)
// in the real type to avoid collision with the type name.
type SpanRecord = obstorage.SpanRecord

// TraceRecord is the real trace record type from storage/domains/observability.
// Previously this was a local stub; now uses the canonical type directly.
// It is an alias for obstorage.GetTraceResponse which contains TraceID + []SpanRecord.
type TraceRecord = obstorage.TraceRecord

// ============================================================================
// Span Types (local constants matching obstypes.SpanType values)
// ============================================================================

const (
	spanTypeAgentRun        = string(obstypes.SpanTypeAgentRun)
	spanTypeModelGeneration = string(obstypes.SpanTypeModelGeneration)
	spanTypeToolCall        = string(obstypes.SpanTypeToolCall)
)

// ============================================================================
// Span message types
// ============================================================================

// SpanMessage represents a message within span input/output.
// Corresponds to TS: interface SpanMessage
type SpanMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []SpanMessageContentPart
}

// SpanMessageContentPart represents a part of a message content array.
type SpanMessageContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SpanInputWithMessages represents span input containing a messages array.
// Corresponds to TS: interface SpanInputWithMessages
type SpanInputWithMessages struct {
	Messages []SpanMessage `json:"messages"`
}

// SpanOutputWithText represents span output containing a text field.
// Corresponds to TS: interface SpanOutputWithText
type SpanOutputWithText struct {
	Text string `json:"text"`
}

// ============================================================================
// SpanTree — hierarchical span structure for efficient lookups
// ============================================================================

// SpanTree is a hierarchical span structure for efficient lookups.
// Corresponds to TS: interface SpanTree
type SpanTree struct {
	SpanMap     map[string]*SpanRecord
	ChildrenMap map[string][]*SpanRecord
	RootSpans   []*SpanRecord
}

// BuildSpanTree builds a hierarchical span tree with efficient lookup maps.
// Corresponds to TS: export function buildSpanTree(spans: SpanRecord[]): SpanTree
func BuildSpanTree(spans []SpanRecord) *SpanTree {
	spanMap := make(map[string]*SpanRecord, len(spans))
	childrenMap := make(map[string][]*SpanRecord)
	var rootSpans []*SpanRecord

	// First pass: build span map.
	for i := range spans {
		spanMap[spans[i].SpanID] = &spans[i]
	}

	// Second pass: build parent-child relationships.
	for i := range spans {
		span := &spans[i]
		if span.ParentSpanID == nil {
			rootSpans = append(rootSpans, span)
		} else {
			parentID := *span.ParentSpanID
			childrenMap[parentID] = append(childrenMap[parentID], span)
		}
	}

	// Sort children by startedAt timestamp for temporal ordering.
	for _, children := range childrenMap {
		sort.Slice(children, func(i, j int) bool {
			return children[i].StartedAt.Before(children[j].StartedAt)
		})
	}

	// Sort root spans by startedAt.
	sort.Slice(rootSpans, func(i, j int) bool {
		return rootSpans[i].StartedAt.Before(rootSpans[j].StartedAt)
	})

	return &SpanTree{
		SpanMap:     spanMap,
		ChildrenMap: childrenMap,
		RootSpans:   rootSpans,
	}
}

// getChildrenOfType extracts children spans of a specific type.
// Corresponds to TS: function getChildrenOfType(...)
func getChildrenOfType(spanTree *SpanTree, parentSpanID string, spanType string) []*SpanRecord {
	children := spanTree.ChildrenMap[parentSpanID]
	var result []*SpanRecord
	for _, child := range children {
		// SpanTyp is obstorage.SpanType (named string); cast to string for comparison
		// with the local spanType* constants.
		if string(child.SpanTyp) == spanType {
			result = append(result, child)
		}
	}
	return result
}

// ============================================================================
// Type guards for span data
// ============================================================================

// isSpanMessage checks if a value looks like a SpanMessage.
// Corresponds to TS: function isSpanMessage(value: unknown): value is SpanMessage
func isSpanMessage(value any) (*SpanMessage, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	role, hasRole := m["role"].(string)
	_, hasContent := m["content"]
	if !hasRole || !hasContent {
		return nil, false
	}
	return &SpanMessage{
		Role:    role,
		Content: m["content"],
	}, true
}

// hasMessagesArray checks if a value has a "messages" array field.
// Corresponds to TS: function hasMessagesArray(value: unknown): value is SpanInputWithMessages
func hasMessagesArray(value any) ([]any, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	msgs, ok := m["messages"].([]any)
	return msgs, ok
}

// hasTextProperty checks if a value has a "text" string field.
// Corresponds to TS: function hasTextProperty(value: unknown): value is SpanOutputWithText
func hasTextProperty(value any) (string, bool) {
	m, ok := value.(map[string]any)
	if !ok {
		return "", false
	}
	text, ok := m["text"].(string)
	return text, ok
}

// ============================================================================
// Message normalization and creation
// ============================================================================

// normalizeMessageContent normalizes message content to a string.
// For arrays with multiple text parts, returns only the last text part (AI SDK convention).
// Corresponds to TS: function normalizeMessageContent(content: ...)
func normalizeMessageContent(content any) string {
	if s, ok := content.(string); ok {
		return s
	}
	if arr, ok := content.([]any); ok {
		var lastText string
		for _, part := range arr {
			if m, ok := part.(map[string]any); ok {
				if t, ok := m["type"].(string); ok && t == "text" {
					if text, ok := m["text"].(string); ok {
						lastText = text
					}
				}
			}
		}
		return lastText
	}
	return ""
}

// createMastraDBMessage creates a MastraDBMessage from span message data.
// Corresponds to TS: function createMastraDBMessage(...)
func createMastraDBMessage(role string, content any, createdAt time.Time, id string) evals.MastraDBMessage {
	contentText := normalizeMessageContent(content)

	return evals.MastraDBMessage{
		ID:   id,
		Role: role,
		Content: evals.MastraDBMessageContent{
			Format:  2,
			Parts:   []any{map[string]any{"type": "text", "text": contentText}},
			Content: contentText,
		},
		CreatedAt: createdAt,
	}
}

// ============================================================================
// Input/output extraction from spans
// ============================================================================

// extractInputMessages extracts input messages from an agent run span.
// Corresponds to TS: function extractInputMessages(agentSpan: SpanRecord): MastraDBMessage[]
func extractInputMessages(agentSpan *SpanRecord) []evals.MastraDBMessage {
	input := agentSpan.Input

	// Handle string input.
	if s, ok := input.(string); ok {
		return []evals.MastraDBMessage{
			createMastraDBMessage("user", s, agentSpan.StartedAt, ""),
		}
	}

	// Handle array input.
	if arr, ok := input.([]any); ok {
		var messages []evals.MastraDBMessage
		for _, item := range arr {
			if msg, ok := isSpanMessage(item); ok {
				messages = append(messages, createMastraDBMessage(msg.Role, msg.Content, agentSpan.StartedAt, ""))
			}
		}
		return messages
	}

	// Handle input with messages array.
	if msgs, ok := hasMessagesArray(input); ok {
		var messages []evals.MastraDBMessage
		for _, item := range msgs {
			if msg, ok := isSpanMessage(item); ok {
				messages = append(messages, createMastraDBMessage(msg.Role, msg.Content, agentSpan.StartedAt, ""))
			}
		}
		return messages
	}

	return nil
}

// extractSystemMessages extracts system messages from an LLM span.
// Corresponds to TS: function extractSystemMessages(llmSpan: SpanRecord): Array<{role: 'system', content: string}>
func extractSystemMessages(llmSpan *SpanRecord) []evals.CoreMessage {
	msgs, ok := hasMessagesArray(llmSpan.Input)
	if !ok {
		return nil
	}

	var result []evals.CoreMessage
	for _, item := range msgs {
		if msg, ok := isSpanMessage(item); ok && msg.Role == "system" {
			result = append(result, evals.CoreMessage{
				Role:    "system",
				Content: normalizeMessageContent(msg.Content),
			})
		}
	}
	return result
}

// extractRememberedMessages extracts conversation history from an LLM span,
// excluding system messages and the current input message.
// Corresponds to TS: function extractRememberedMessages(llmSpan: SpanRecord, currentInputContent: string): MastraDBMessage[]
func extractRememberedMessages(llmSpan *SpanRecord, currentInputContent string) []evals.MastraDBMessage {
	msgs, ok := hasMessagesArray(llmSpan.Input)
	if !ok {
		return nil
	}

	var result []evals.MastraDBMessage
	for _, item := range msgs {
		msg, ok := isSpanMessage(item)
		if !ok {
			continue
		}
		if msg.Role == "system" {
			continue
		}
		content := normalizeMessageContent(msg.Content)
		if content == currentInputContent {
			continue
		}
		result = append(result, createMastraDBMessage(msg.Role, msg.Content, llmSpan.StartedAt, ""))
	}
	return result
}

// reconstructToolInvocations reconstructs tool invocations from tool call spans.
// Corresponds to TS: function reconstructToolInvocations(spanTree: SpanTree, parentSpanId: string)
func reconstructToolInvocations(spanTree *SpanTree, parentSpanID string) []map[string]any {
	toolSpans := getChildrenOfType(spanTree, parentSpanID, spanTypeToolCall)

	var invocations []map[string]any
	for _, toolSpan := range toolSpans {
		toolName := "unknown"
		if toolSpan.EntityName != nil {
			toolName = *toolSpan.EntityName
		} else if toolSpan.EntityID != nil {
			toolName = *toolSpan.EntityID
		}

		toolID := ""
		if toolSpan.EntityID != nil {
			toolID = *toolSpan.EntityID
		}

		input := toolSpan.Input
		if input == nil {
			input = map[string]any{}
		}
		output := toolSpan.Output
		if output == nil {
			output = map[string]any{}
		}

		invocations = append(invocations, map[string]any{
			"toolCallId": toolSpan.SpanID,
			"toolName":   toolName,
			"toolId":     toolID,
			"args":       input,
			"result":     output,
			"state":      "result",
		})
	}
	return invocations
}

// ============================================================================
// Trace validation
// ============================================================================

// ValidateTrace validates the trace structure and returns descriptive errors.
// Corresponds to TS: export function validateTrace(trace: TraceRecord): void
func ValidateTrace(trace *TraceRecord) error {
	if trace == nil {
		return errors.New("trace is null or undefined")
	}
	if trace.Spans == nil {
		return errors.New("trace must have a spans array")
	}
	if len(trace.Spans) == 0 {
		return errors.New("trace has no spans")
	}

	// Check for circular references in parent-child relationships.
	spanIDs := make(map[string]bool, len(trace.Spans))
	for _, span := range trace.Spans {
		spanIDs[span.SpanID] = true
	}
	for _, span := range trace.Spans {
		if span.ParentSpanID != nil && !spanIDs[*span.ParentSpanID] {
			return errors.New("span " + span.SpanID + " references non-existent parent " + *span.ParentSpanID)
		}
	}

	return nil
}

// ============================================================================
// Primary LLM span finder
// ============================================================================

// findPrimaryLLMSpan finds the most recent model span that contains conversation history.
// Corresponds to TS: function findPrimaryLLMSpan(spanTree: SpanTree, rootAgentSpan: SpanRecord): SpanRecord
func findPrimaryLLMSpan(spanTree *SpanTree, rootAgentSpan *SpanRecord) (*SpanRecord, error) {
	directLLMSpans := getChildrenOfType(spanTree, rootAgentSpan.SpanID, spanTypeModelGeneration)
	if len(directLLMSpans) > 0 {
		return directLLMSpans[0], nil
	}
	return nil, errors.New("no model generation span found in trace")
}

// ============================================================================
// Trace preparation
// ============================================================================

// prepareTraceForTransformation validates and builds the span tree.
// Corresponds to TS: function prepareTraceForTransformation(trace: TraceRecord)
func prepareTraceForTransformation(trace *TraceRecord) (*SpanTree, *SpanRecord, error) {
	if err := ValidateTrace(trace); err != nil {
		return nil, nil, err
	}

	spanTree := BuildSpanTree(trace.Spans)

	// Find the root agent run span.
	// SpanTyp is obstorage.SpanType (named string); cast to string for comparison.
	var rootAgentSpan *SpanRecord
	for _, span := range spanTree.RootSpans {
		if string(span.SpanTyp) == spanTypeAgentRun {
			rootAgentSpan = span
			break
		}
	}

	if rootAgentSpan == nil {
		return nil, nil, errors.New("no root agent_run span found in trace")
	}

	return spanTree, rootAgentSpan, nil
}

// ============================================================================
// TransformTraceToScorerInputAndOutput
// ============================================================================

// TransformTraceToScorerInputAndOutput transforms a trace into scorer input/output format.
// Corresponds to TS: export function transformTraceToScorerInputAndOutput(trace: TraceRecord)
func TransformTraceToScorerInputAndOutput(trace *TraceRecord) (*evals.ScorerRunInputForAgent, evals.ScorerRunOutputForAgent, error) {
	spanTree, rootAgentSpan, err := prepareTraceForTransformation(trace)
	if err != nil {
		return nil, nil, err
	}

	if rootAgentSpan.Output == nil {
		return nil, nil, errors.New("root agent span has no output")
	}

	// Build input.
	primaryLLMSpan, err := findPrimaryLLMSpan(spanTree, rootAgentSpan)
	if err != nil {
		return nil, nil, err
	}

	inputMessages := extractInputMessages(rootAgentSpan)
	systemMessages := extractSystemMessages(primaryLLMSpan)

	// Extract remembered messages from LLM span (excluding current input).
	currentInputContent := ""
	if len(inputMessages) > 0 {
		currentInputContent = inputMessages[0].Content.Content
	}
	rememberedMessages := extractRememberedMessages(primaryLLMSpan, currentInputContent)

	// Convert CoreMessage to CoreSystemMessage for tagged system messages.
	var coreSystemMessages []evals.CoreMessage
	for _, sm := range systemMessages {
		coreSystemMessages = append(coreSystemMessages, evals.CoreMessage{
			Role:    sm.Role,
			Content: sm.Content,
		})
	}

	input := &evals.ScorerRunInputForAgent{
		InputMessages:        inputMessages,
		RememberedMessages:   rememberedMessages,
		SystemMessages:       coreSystemMessages,
		TaggedSystemMessages: map[string][]evals.CoreSystemMessage{}, // TODO: Support tagged system messages
	}

	// Build output.
	toolInvocations := reconstructToolInvocations(spanTree, rootAgentSpan.SpanID)
	responseText := ""
	if text, ok := hasTextProperty(rootAgentSpan.Output); ok {
		responseText = text
	}

	// Build parts array: tool invocations first, then text.
	var parts []any
	for _, toolInvocation := range toolInvocations {
		parts = append(parts, map[string]any{
			"type":           "tool-invocation",
			"toolInvocation": toolInvocation,
		})
	}
	if len(responseText) > 0 {
		parts = append(parts, map[string]any{
			"type": "text",
			"text": responseText,
		})
	}

	endedAt := rootAgentSpan.StartedAt
	if rootAgentSpan.EndedAt != nil {
		endedAt = *rootAgentSpan.EndedAt
	}

	responseMessage := evals.MastraDBMessage{
		ID:   "",
		Role: "assistant",
		Content: evals.MastraDBMessageContent{
			Format:          2,
			Parts:           parts,
			Content:         responseText,
			ToolInvocations: toolInvocationsToAnySlice(toolInvocations),
		},
		CreatedAt: endedAt,
	}

	output := evals.ScorerRunOutputForAgent{responseMessage}

	return input, output, nil
}

// toolInvocationsToAnySlice converts tool invocations to []any for the MastraDBMessageContent.
func toolInvocationsToAnySlice(invocations []map[string]any) []any {
	if len(invocations) == 0 {
		return nil
	}
	result := make([]any, len(invocations))
	for i, inv := range invocations {
		result[i] = inv
	}
	return result
}
