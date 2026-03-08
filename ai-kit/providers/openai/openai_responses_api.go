// Ported from: packages/openai/src/responses/openai-responses-api.ts
package openai

// OpenAIResponsesInput is the input to the OpenAI Responses API.
type OpenAIResponsesInput = []OpenAIResponsesInputItem

// OpenAIResponsesInputItem is a sealed interface for input items.
type OpenAIResponsesInputItem interface {
	openaiResponsesInputItem()
}

// --- Input item types ---

// OpenAIResponsesSystemMessage is a system or developer message.
type OpenAIResponsesSystemMessage struct {
	Role    string `json:"role"` // "system" or "developer"
	Content string `json:"content"`
}

func (OpenAIResponsesSystemMessage) openaiResponsesInputItem() {}

// OpenAIResponsesUserMessage is a user message with multiple content parts.
type OpenAIResponsesUserMessage struct {
	Role    string `json:"role"` // "user"
	Content []any  `json:"content"`
}

func (OpenAIResponsesUserMessage) openaiResponsesInputItem() {}

// OpenAIResponsesAssistantMessage is an assistant message.
type OpenAIResponsesAssistantMessage struct {
	Role    string `json:"role"` // "assistant"
	Content []any  `json:"content"`
	ID      string `json:"id,omitempty"`
	Phase   string `json:"phase,omitempty"`
}

func (OpenAIResponsesAssistantMessage) openaiResponsesInputItem() {}

// OpenAIResponsesFunctionCall is a function call input item.
type OpenAIResponsesFunctionCall struct {
	Type      string `json:"type"` // "function_call"
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	ID        string `json:"id,omitempty"`
}

func (OpenAIResponsesFunctionCall) openaiResponsesInputItem() {}

// OpenAIResponsesFunctionCallOutput is a function call output input item.
type OpenAIResponsesFunctionCallOutput struct {
	Type   string `json:"type"` // "function_call_output"
	CallID string `json:"call_id"`
	Output any    `json:"output"` // string or []any
}

func (OpenAIResponsesFunctionCallOutput) openaiResponsesInputItem() {}

// OpenAIResponsesCustomToolCall is a custom tool call input item.
type OpenAIResponsesCustomToolCall struct {
	Type   string `json:"type"` // "custom_tool_call"
	ID     string `json:"id,omitempty"`
	CallID string `json:"call_id"`
	Name   string `json:"name"`
	Input  string `json:"input"`
}

func (OpenAIResponsesCustomToolCall) openaiResponsesInputItem() {}

// OpenAIResponsesCustomToolCallOutput is a custom tool call output input item.
type OpenAIResponsesCustomToolCallOutput struct {
	Type   string `json:"type"` // "custom_tool_call_output"
	CallID string `json:"call_id"`
	Output any    `json:"output"` // string or []any
}

func (OpenAIResponsesCustomToolCallOutput) openaiResponsesInputItem() {}

// OpenAIResponsesMcpApprovalResponse is an MCP approval response input item.
type OpenAIResponsesMcpApprovalResponse struct {
	Type              string `json:"type"` // "mcp_approval_response"
	ApprovalRequestID string `json:"approval_request_id"`
	Approve           bool   `json:"approve"`
}

func (OpenAIResponsesMcpApprovalResponse) openaiResponsesInputItem() {}

// OpenAIResponsesComputerCall is a computer call input item.
type OpenAIResponsesComputerCall struct {
	Type   string `json:"type"` // "computer_call"
	ID     string `json:"id"`
	Status string `json:"status,omitempty"`
}

func (OpenAIResponsesComputerCall) openaiResponsesInputItem() {}

// OpenAIResponsesLocalShellCall is a local shell call input item.
type OpenAIResponsesLocalShellCall struct {
	Type   string                              `json:"type"` // "local_shell_call"
	ID     string                              `json:"id"`
	CallID string                              `json:"call_id"`
	Action OpenAIResponsesLocalShellCallAction `json:"action"`
}

func (OpenAIResponsesLocalShellCall) openaiResponsesInputItem() {}

// OpenAIResponsesLocalShellCallAction is the action for a local shell call.
type OpenAIResponsesLocalShellCallAction struct {
	Type             string            `json:"type"` // "exec"
	Command          []string          `json:"command"`
	TimeoutMs        *int              `json:"timeout_ms,omitempty"`
	User             string            `json:"user,omitempty"`
	WorkingDirectory string            `json:"working_directory,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
}

// OpenAIResponsesLocalShellCallOutput is a local shell call output input item.
type OpenAIResponsesLocalShellCallOutput struct {
	Type   string `json:"type"` // "local_shell_call_output"
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

func (OpenAIResponsesLocalShellCallOutput) openaiResponsesInputItem() {}

// OpenAIResponsesShellCall is a shell call input item.
type OpenAIResponsesShellCall struct {
	Type   string                         `json:"type"` // "shell_call"
	ID     string                         `json:"id"`
	CallID string                         `json:"call_id"`
	Status string                         `json:"status"`
	Action OpenAIResponsesShellCallAction `json:"action"`
}

func (OpenAIResponsesShellCall) openaiResponsesInputItem() {}

// OpenAIResponsesShellCallAction is the action for a shell call.
type OpenAIResponsesShellCallAction struct {
	Commands        []string `json:"commands"`
	TimeoutMs       *int     `json:"timeout_ms,omitempty"`
	MaxOutputLength *int     `json:"max_output_length,omitempty"`
}

// OpenAIResponsesShellCallOutput is a shell call output input item.
type OpenAIResponsesShellCallOutput struct {
	Type            string                              `json:"type"` // "shell_call_output"
	ID              string                              `json:"id,omitempty"`
	CallID          string                              `json:"call_id"`
	Status          string                              `json:"status,omitempty"`
	MaxOutputLength *int                                `json:"max_output_length,omitempty"`
	Output          []OpenAIResponsesShellOutputEntry   `json:"output"`
}

func (OpenAIResponsesShellCallOutput) openaiResponsesInputItem() {}

// OpenAIResponsesShellOutputEntry is an entry in the shell call output.
type OpenAIResponsesShellOutputEntry struct {
	Stdout  string                           `json:"stdout"`
	Stderr  string                           `json:"stderr"`
	Outcome OpenAIResponsesShellCallOutcome  `json:"outcome"`
}

// OpenAIResponsesShellCallOutcome is the outcome of a shell call.
type OpenAIResponsesShellCallOutcome struct {
	Type     string `json:"type"` // "timeout" or "exit"
	ExitCode *int   `json:"exit_code,omitempty"`
}

// OpenAIResponsesApplyPatchCall is an apply_patch call input item.
type OpenAIResponsesApplyPatchCall struct {
	Type      string `json:"type"` // "apply_patch_call"
	ID        string `json:"id,omitempty"`
	CallID    string `json:"call_id"`
	Status    string `json:"status"`
	Operation any    `json:"operation"` // discriminated union by "type"
}

func (OpenAIResponsesApplyPatchCall) openaiResponsesInputItem() {}

// OpenAIResponsesApplyPatchCallOutput is an apply_patch call output input item.
type OpenAIResponsesApplyPatchCallOutput struct {
	Type   string `json:"type"` // "apply_patch_call_output"
	CallID string `json:"call_id"`
	Status string `json:"status"`
	Output string `json:"output,omitempty"`
}

func (OpenAIResponsesApplyPatchCallOutput) openaiResponsesInputItem() {}

// OpenAIResponsesReasoning is a reasoning input item.
type OpenAIResponsesReasoning struct {
	Type             string                            `json:"type"` // "reasoning"
	ID               string                            `json:"id,omitempty"`
	EncryptedContent *string                           `json:"encrypted_content,omitempty"`
	Summary          []OpenAIResponsesReasoningSummary `json:"summary"`
}

func (OpenAIResponsesReasoning) openaiResponsesInputItem() {}

// OpenAIResponsesReasoningSummary is a summary entry in a reasoning item.
type OpenAIResponsesReasoningSummary struct {
	Type string `json:"type"` // "summary_text"
	Text string `json:"text"`
}

// OpenAIResponsesItemReference is an item reference input item.
type OpenAIResponsesItemReference struct {
	Type string `json:"type"` // "item_reference"
	ID   string `json:"id"`
}

func (OpenAIResponsesItemReference) openaiResponsesInputItem() {}

// --- Tool types ---

// OpenAIResponsesTool represents a tool definition for the Responses API.
// This is represented as a map for flexibility since it's a discriminated union.
type OpenAIResponsesTool = map[string]any

// --- Logprobs ---

// OpenAIResponsesLogprob represents logprob information for a token.
type OpenAIResponsesLogprob struct {
	Token       string                          `json:"token"`
	Logprob     float64                         `json:"logprob"`
	TopLogprobs []OpenAIResponsesTopLogprobItem `json:"top_logprobs"`
}

// OpenAIResponsesTopLogprobItem represents a top logprob item.
type OpenAIResponsesTopLogprobItem struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
}

// --- Chunk types ---

// OpenAIResponsesIncludeValue is a valid value for the "include" parameter.
type OpenAIResponsesIncludeValue = string

// --- Streaming chunk types ---

// OpenAIResponsesChunk represents a parsed streaming chunk from the Responses API.
// Because Go doesn't have discriminated unions, we use a single struct with a Type
// field and optional fields for each chunk type.
type OpenAIResponsesChunk struct {
	Type string `json:"type"`

	// For response.output_text.delta
	ItemID  string                    `json:"item_id,omitempty"`
	Delta   string                    `json:"delta,omitempty"`
	Logprobs []OpenAIResponsesLogprob `json:"logprobs,omitempty"`

	// For response.completed / response.incomplete
	Response *OpenAIResponsesChunkResponse `json:"response,omitempty"`

	// For response.output_item.added / response.output_item.done
	OutputIndex int                           `json:"output_index,omitempty"`
	Item        map[string]any                `json:"item,omitempty"`

	// For response.output_text.annotation.added
	Annotation map[string]any `json:"annotation,omitempty"`

	// For response.reasoning_summary_text.delta / response.reasoning_summary_part.added/done
	SummaryIndex int `json:"summary_index,omitempty"`

	// For response.apply_patch_call_operation_diff.delta
	Obfuscation *string `json:"obfuscation,omitempty"`

	// For response.apply_patch_call_operation_diff.done
	Diff string `json:"diff,omitempty"`

	// For response.image_generation_call.partial_image
	PartialImageB64 string `json:"partial_image_b64,omitempty"`

	// For response.code_interpreter_call_code.done
	Code string `json:"code,omitempty"`

	// For error chunk
	SequenceNumber int            `json:"sequence_number,omitempty"`
	Error          map[string]any `json:"error,omitempty"`

	// Raw JSON for unknown chunks
	RawJSON map[string]any `json:"-"`
}

// OpenAIResponsesChunkResponse contains response data within a streaming chunk.
type OpenAIResponsesChunkResponse struct {
	ID                string                                `json:"id,omitempty"`
	CreatedAt         int64                                 `json:"created_at,omitempty"`
	Model             string                                `json:"model,omitempty"`
	ServiceTier       *string                               `json:"service_tier,omitempty"`
	IncompleteDetails *OpenAIResponsesIncompleteDetails     `json:"incomplete_details,omitempty"`
	Usage             *OpenAIResponsesUsage                 `json:"usage,omitempty"`
}

// OpenAIResponsesIncompleteDetails contains details about why a response is incomplete.
type OpenAIResponsesIncompleteDetails struct {
	Reason string `json:"reason"`
}

// --- Non-streaming response ---

// OpenAIResponsesResponse represents a non-streaming response from the Responses API.
type OpenAIResponsesResponse struct {
	ID                string                            `json:"id,omitempty"`
	CreatedAt         *int64                            `json:"created_at,omitempty"`
	Error             *OpenAIResponsesResponseError     `json:"error,omitempty"`
	Model             string                            `json:"model,omitempty"`
	Output            []map[string]any                  `json:"output,omitempty"`
	ServiceTier       *string                           `json:"service_tier,omitempty"`
	IncompleteDetails *OpenAIResponsesIncompleteDetails `json:"incomplete_details,omitempty"`
	Usage             *OpenAIResponsesUsage             `json:"usage,omitempty"`
}

// OpenAIResponsesResponseError represents an error in the response.
type OpenAIResponsesResponseError struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param,omitempty"`
	Code    string  `json:"code"`
}

// --- Filter types ---

// OpenAIResponsesFileSearchToolComparisonFilter is a filter for file search.
type OpenAIResponsesFileSearchToolComparisonFilter struct {
	Key   string `json:"key"`
	Type  string `json:"type"` // "eq", "ne", "gt", "gte", "lt", "lte", "in", "nin"
	Value any    `json:"value"` // string, number, bool, or []string
}

// OpenAIResponsesFileSearchToolCompoundFilter is a compound filter for file search.
type OpenAIResponsesFileSearchToolCompoundFilter struct {
	Type    string `json:"type"` // "and" or "or"
	Filters []any  `json:"filters"` // ComparisonFilter or CompoundFilter
}
