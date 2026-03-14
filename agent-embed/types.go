package agentembed

import "encoding/json"

// ProviderConfig configures an LLM provider.
type ProviderConfig struct {
	APIKey  string
	BaseURL string // optional override
	Headers map[string]string
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
	ReasoningTokens  int `json:"reasoningTokens,omitempty"`
}

// FinishReason indicates why generation stopped.
type FinishReason string

const (
	FinishStop          FinishReason = "stop"
	FinishLength        FinishReason = "length"
	FinishContentFilter FinishReason = "content-filter"
	FinishToolCalls     FinishReason = "tool-calls"
	FinishError         FinishReason = "error"
	FinishSuspended     FinishReason = "suspended"
	FinishOther         FinishReason = "other"
)

// ResponseMeta contains metadata about the LLM response.
type ResponseMeta struct {
	ID        string `json:"id"`
	ModelID   string `json:"modelId"`
	Timestamp string `json:"timestamp"` // ISO 8601
}

// Message represents a conversation message.
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant", "tool"
	Content any    `json:"content"` // string or []ContentPart
}

// SystemMessage creates a system message.
func SystemMessage(content string) Message {
	return Message{Role: "system", Content: content}
}

// UserMessage creates a user message.
func UserMessage(content string) Message {
	return Message{Role: "user", Content: content}
}

// AssistantMessage creates an assistant message.
func AssistantMessage(content string) Message {
	return Message{Role: "assistant", Content: content}
}

// Tool defines a function that the agent can call.
type Tool struct {
	Description string
	Parameters  json.RawMessage // JSON Schema
	Execute     func(ctx ToolContext, args json.RawMessage) (any, error)
}

// ToolContext is passed to tool Execute functions.
type ToolContext struct {
	// Future: AbortSignal, RequestContext, etc.
}

// ToolChoice controls how the model selects tools.
type ToolChoice struct {
	Mode     string `json:"mode"` // "auto", "none", "required", "tool"
	ToolName string `json:"toolName,omitempty"`
}

// ToolCall represents a tool invocation by the model.
type ToolCall struct {
	ToolCallID string          `json:"toolCallId"`
	ToolName   string          `json:"toolName"`
	Args       json.RawMessage `json:"args"`
}

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	ToolCallID string `json:"toolCallId"`
	ToolName   string `json:"toolName"`
	Args       any    `json:"args"`
	Result     any    `json:"result"`
}

// StepResult contains the result of a single LLM call step.
type StepResult struct {
	Text         string       `json:"text"`
	Reasoning    string       `json:"reasoning,omitempty"`
	ToolCalls    []ToolCall   `json:"toolCalls,omitempty"`
	ToolResults  []ToolResult `json:"toolResults,omitempty"`
	FinishReason FinishReason `json:"finishReason"`
	Usage        Usage        `json:"usage"`
	StepType     string       `json:"stepType"` // "initial", "tool-result", "continue"
	IsContinued  bool         `json:"isContinued"`
}

// GenerateResult is returned by Agent.Generate.
type GenerateResult struct {
	Text            string          `json:"text"`
	Reasoning       string          `json:"reasoning,omitempty"`
	Object          json.RawMessage `json:"object,omitempty"` // structured output
	ToolCalls       []ToolCall      `json:"toolCalls,omitempty"`
	ToolResults     []ToolResult    `json:"toolResults,omitempty"`
	FinishReason    FinishReason    `json:"finishReason"`
	Usage           Usage           `json:"usage"`
	Steps           []StepResult    `json:"steps,omitempty"`
	Response        ResponseMeta    `json:"response"`
	SuspendPayload  json.RawMessage `json:"suspendPayload,omitempty"`
	RunID           string          `json:"runId,omitempty"`
	ProviderMeta    json.RawMessage `json:"providerMetadata,omitempty"`
	TripWire        json.RawMessage `json:"tripwire,omitempty"`
	ScoringData     json.RawMessage `json:"scoringData,omitempty"`
}

// StreamResult is returned by Agent.Stream after streaming completes.
// Same shape as GenerateResult — streaming is a transport concern.
type StreamResult = GenerateResult

// Pointer helpers for optional fields.
func Float64(v float64) *float64 { return &v }
func Int(v int) *int             { return &v }
