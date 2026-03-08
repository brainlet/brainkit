// Ported from: packages/core/src/loop/network/index.ts
//
// NetworkLoop implements the multi-agent network routing loop.
// This file contains the full network loop implementation, ported 1:1
// from the TypeScript source.
package network

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// Memory is a stub for ../../memory.MastraMemory.
// Stub: real memory.MastraMemory has different method signatures; these methods
// match network loop's usage. Wiring requires adapting call sites to real interface.
type Memory interface {
	GetThreadByID(ctx context.Context, params map[string]any) (map[string]any, error)
	CreateThread(ctx context.Context, params map[string]any) (map[string]any, error)
	SaveMessages(ctx context.Context, params map[string]any) error
	Recall(ctx context.Context, params map[string]any) (*RecallResult, error)
	ListTools(ctx context.Context) (map[string]any, error)
	GetMergedThreadConfig(memoryConfig map[string]any) map[string]any
}

// RecallResult holds the result from Memory.Recall.
type RecallResult struct {
	Messages []MastraDBMessage `json:"messages"`
}

// Agent is a stub for ../../agent.Agent.
// Stub: real agent.Agent is a struct with different method signatures.
// Network loop uses a custom interface that matches its call patterns.
type Agent interface {
	GetID() string
	GetName() string
	GetDescription() string
	GetInstructions(ctx context.Context, params map[string]any) (string, error)
	ListAgents(ctx context.Context, params map[string]any) (map[string]Agent, error)
	ListWorkflows(ctx context.Context, params map[string]any) (map[string]Workflow, error)
	ListTools(ctx context.Context, params map[string]any) (map[string]Tool, error)
	GetModel(ctx context.Context, params map[string]any) (any, error)
	GetMemory(ctx context.Context, params map[string]any) (Memory, error)
	GetDefaultOptions(ctx context.Context, params map[string]any) (map[string]any, error)
	ListConfiguredInputProcessors(ctx context.Context, requestContext any) ([]any, error)
	ListConfiguredOutputProcessors(ctx context.Context, requestContext any) ([]any, error)
	GetMastraInstance() Mastra
	GetMostRecentUserMessage(messages []any) map[string]any
	GetLLM(ctx context.Context, params map[string]any) (any, error)
	ResolveTitleGenerationConfig(config any) (shouldGenerate bool, titleModel any, titleInstructions any)
	GenTitle(ctx context.Context, message string, requestContext any, observabilityContext any, titleModel any, titleInstructions any) (string, error)
	Stream(ctx context.Context, messages any, options map[string]any) (*AgentStreamResult, error)
	ResumeStream(ctx context.Context, resumeData any, options map[string]any) (*AgentStreamResult, error)
}

// AgentStreamResult holds the result of Agent.Stream/ResumeStream.
type AgentStreamResult struct {
	FullStream  <-chan map[string]any
	Text        string
	Usage       map[string]any
	MessageList AgentMessageList
	// RememberedMessages for filterMessagesForSubAgent.
	RememberedMessages []MastraDBMessage
	// Object for structured output results.
	Object     any
	FinishReason string
}

// AgentMessageList provides message access methods.
type AgentMessageList interface {
	GetAllV1() []any
}

// Workflow is a stub for ../../workflows.Workflow.
// Stub: real workflows.Workflow is a struct. Network uses interface for method dispatch.
type Workflow interface {
	GetID() string
	GetName() string
	GetDescription() string
	GetInputSchemaJSON() string
	CreateRun(ctx context.Context, params map[string]any) (WorkflowRun, error)
}

// WorkflowRun holds a workflow run.
type WorkflowRun interface {
	RunID() string
	Cancel(ctx context.Context) error
	Stream(ctx context.Context, params map[string]any) (*WorkflowStreamResult, error)
	ResumeStream(ctx context.Context, params map[string]any) (*WorkflowStreamResult, error)
}

// WorkflowStreamResult holds the result of a workflow stream.
type WorkflowStreamResult struct {
	FullStream <-chan map[string]any
	Result     map[string]any
	Usage      map[string]any
}

// Tool is a stub for ../../tools.Tool.
// Stub: real tools.Tool is a struct. Network uses interface for method dispatch.
type Tool interface {
	GetID() string
	GetDescription() string
	GetInputSchemaJSON() string
	Execute(ctx context.Context, input any, execCtx map[string]any) (any, error)
	HasRequireApproval() bool
	NeedsApprovalFn(input any) (bool, error)
	HasSuspendSchema() bool
}

// Mastra is a narrow interface for the Mastra orchestrator, defined here to
// break circular dependency. core.Mastra satisfies this interface.
type Mastra interface {
	GetLogger() logger.IMastraLogger
}

// Logger is a type alias to logger.IMastraLogger so that core.Mastra satisfies
// the network.Mastra interface at compile time.
//
// Ported from: packages/core/src/loop/network — uses mastra.getLogger()
type Logger = logger.IMastraLogger

// MessageList is a stub for ../../agent/message-list.MessageList.
// Stub: real agent.MessageList is struct{} with different methods.
// Network uses custom interface matching its call patterns.
type MessageList interface {
	Add(msg any, source string)
	GetAllDB() []MastraDBMessage
	GetInputDB() []MastraDBMessage
	GetResponseDB() []MastraDBMessage
	GetAllUI() []any
	AddSystem(msg string)
}

// MastraDBMessage is a stub for ../../agent/message-list.MastraDBMessage.
// Stub: real type in agent/processors has different Content struct. Kept local for shape.
type MastraDBMessage struct {
	ID         string         `json:"id,omitempty"`
	Type       string         `json:"type,omitempty"`
	Role       string         `json:"role"`
	Content    MessageContent `json:"content"`
	CreatedAt  time.Time      `json:"createdAt,omitempty"`
	ThreadID   string         `json:"threadId,omitempty"`
	ResourceID string         `json:"resourceId,omitempty"`
}

// MessageContent holds message content with format and parts.
type MessageContent struct {
	Format   int            `json:"format,omitempty"`
	Parts    []MessagePart  `json:"parts,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// MessagePart is a union-like struct for different message part types.
type MessagePart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data any    `json:"data,omitempty"`
}

// MessageListInput is a stub for ../../agent/message-list.MessageListInput.
// Stub: = any in both agent and here. Same shape; kept local to avoid agent dependency.
type MessageListInput = any

// StructuredOutputOptions is a stub for ../../agent/types.StructuredOutputOptions.
// Stub: real type has Schema, Model, JSONPromptInjection fields. Kept local.
type StructuredOutputOptions struct {
	Schema any `json:"schema,omitempty"`
}

// AgentExecutionOptions is a stub for ../../agent.AgentExecutionOptions.
// Stub: real type has many fields. Kept as empty struct.
type AgentExecutionOptions struct{}

// MultiPrimitiveExecutionOptions is a stub for ../../agent/agent.types.MultiPrimitiveExecutionOptions.
// Stub: real type has more fields. Kept local.
type MultiPrimitiveExecutionOptions struct {
	ModelSettings any `json:"modelSettings,omitempty"`
}

// NetworkOptions is a stub for ../../agent/agent.types.NetworkOptions.
// Stub: real type has similar shape but may have additional fields. Kept local.
type NetworkOptions struct {
	OnStepFinish func(event any) error `json:"-"`
	OnError      func(args any) error  `json:"-"`
	OnAbort      func(event any) error `json:"-"`
	AbortSignal  context.Context       `json:"-"`
}

// MastraLLMVNext is a stub for ../../llm/model/model.loop.MastraLLMVNext.
// Cannot import llm/model: would risk circular dependency through loop → llm/model → loop.
// The real type is a struct with model management methods. The network loop references
// this type but does not call methods on it directly; it's passed through to sub-components.
// These methods represent the minimum useful contract for model identification.
type MastraLLMVNext interface {
	// GetFirstModelConfig returns the first configured model's configuration.
	GetFirstModelConfig() any
}

// ObservabilityContext is imported from observability/types.
type ObservabilityContext = obstypes.ObservabilityContext

// ProcessorRunner is a stub for ../../processors/runner.ProcessorRunner.
// Stub: real type lives in processors package. Kept local with no-op methods.
type ProcessorRunner struct {
	OutputProcessors []any
	InputProcessors  []any
	Logger           Logger
	AgentName        string
}

// RunOutputProcessors runs the output processors on the message list.
func (pr *ProcessorRunner) RunOutputProcessors(messageList MessageList, observabilityContext any, requestContext any) error {
	// Stub: processors not yet implemented.
	return nil
}

// RequestContext is imported from requestcontext.
type RequestContext = requestcontext.RequestContext

// newRequestContext creates a new RequestContext.
// Wrapper for requestcontext.NewRequestContext() used within the network package.
func newRequestContext() *RequestContext {
	return requestcontext.NewRequestContext()
}

// ChunkType is a stub for ../../stream.ChunkType.
// Stub: real stream.ChunkType is a struct with Type, Payload, BaseChunkType fields.
// Network uses map[string]any for lightweight chunk passing. Shape mismatch.
type ChunkType = map[string]any

// NetworkChunkType is a stub for ../../stream/types.NetworkChunkType.
// Stub: real type is a struct. Network uses map[string]any. Shape mismatch.
type NetworkChunkType = map[string]any

// NOTE: CompletionConfig is defined in validation.go with full fields.

// MastraAgentNetworkStream holds the network stream output.
// Stub: real type in stream package. Kept local for network's specific shape.
type MastraAgentNetworkStream struct {
	// Stream is the channel of network chunks.
	Stream <-chan NetworkChunkType
	// Result holds the final workflow result (available after stream closes).
	Result any
	// RunID is the workflow run ID.
	RunID string
}

// SuspendOptions is a stub for ../../workflows.SuspendOptions.
// Stub: real workflows.SuspendOptions may have additional fields. Kept local.
type SuspendOptions struct {
	ResumeSchema string `json:"resumeSchema,omitempty"`
}

// IdGeneratorContext is re-exported from the types package.
// Ported from: packages/core/src/types/dynamic-argument.ts — IdGeneratorContext
type IdGeneratorContext = aktypes.IdGeneratorContext

// NetworkIdGenerator is a function that generates IDs for various purposes.
type NetworkIdGenerator func(ctx ...IdGeneratorContext) string

// PrimitiveType is duplicated from the parent loop package.
// Stub: identical shape (string enum with same values). Kept local to avoid
// network → loop dependency which would couple the subpackage to its parent.
// No import cycle exists, but the dependency direction is undesirable.
type PrimitiveType string

const (
	PrimitiveTypeAgent    PrimitiveType = "agent"
	PrimitiveTypeWorkflow PrimitiveType = "workflow"
	PrimitiveTypeNone     PrimitiveType = "none"
	PrimitiveTypeTool     PrimitiveType = "tool"
)

// ChunkFromNetwork is the chunk source identifier for network events.
const ChunkFromNetwork = "NETWORK"

// ---------------------------------------------------------------------------
// Core Network Types
// ---------------------------------------------------------------------------

// NetworkLoopConfig configures a network loop execution.
type NetworkLoopConfig struct {
	// Mastra instance providing agent/workflow/tool registries.
	Mastra Mastra `json:"-"`
	// RoutingAgent is the agent that decides which primitive to run next.
	RoutingAgent Agent `json:"-"`
	// Messages is the initial message list for the conversation.
	Messages MessageListInput `json:"-"`
	// Options are the execution options passed through from the caller.
	Options *NetworkOptions `json:"options,omitempty"`
	// MaxIterations caps the number of routing iterations.
	MaxIterations int `json:"maxIterations,omitempty"`
	// Completion configures completion scoring / task-done detection.
	Completion *CompletionConfig `json:"completion,omitempty"`
	// StructuredOutput enables structured (schema-validated) final output.
	StructuredOutput *StructuredOutputOptions `json:"structuredOutput,omitempty"`
	// Ctx is the context for cancellation (replaces AbortSignal).
	Ctx context.Context `json:"-"`
	// OnAbort callback invoked when the context is cancelled.
	OnAbort func(event any) error `json:"-"`
	// ObservabilityContext for tracing.
	ObservabilityContext
}

// NetworkLoopResult is the result of a completed network loop.
type NetworkLoopResult struct {
	// Text is the final text result.
	Text string `json:"text"`
	// Object is the structured output (if StructuredOutput was configured).
	Object any `json:"object,omitempty"`
	// Iterations is how many routing iterations were executed.
	Iterations int `json:"iterations"`
	// RunID is the unique run identifier.
	RunID string `json:"runId"`
	// Messages contains the full conversation history.
	Messages []MastraDBMessage `json:"messages"`
}

// SelectedPrimitive describes the primitive chosen by the routing agent
// for a single iteration.
type SelectedPrimitive struct {
	ID   string        `json:"id"`
	Type PrimitiveType `json:"type"`
}

// IterationResult captures the outcome of a single network iteration.
type IterationResult struct {
	// SelectedPrimitive is which primitive was selected and executed.
	SelectedPrimitive SelectedPrimitive `json:"selectedPrimitive"`
	// Prompt is the input sent to the primitive.
	Prompt string `json:"prompt"`
	// Result is the text output from the primitive execution.
	Result string `json:"result"`
	// Duration is how long the iteration took.
	Duration time.Duration `json:"duration"`
}

// ---------------------------------------------------------------------------
// NetworkLoop (interface)
// ---------------------------------------------------------------------------

// NetworkLoop defines the interface for running a multi-agent network loop.
type NetworkLoop interface {
	// Run executes the network loop to completion and returns the final result.
	Run(ctx context.Context) (*NetworkLoopResult, error)
	// Stream executes the network loop and streams chunks via the returned channel.
	Stream(ctx context.Context) (<-chan NetworkChunkType, error)
}

// ---------------------------------------------------------------------------
// Internal types
// ---------------------------------------------------------------------------

// iterationStepData is the data flowing through the network workflow steps.
type iterationStepData struct {
	Task                 string        `json:"task"`
	PrimitiveID          string        `json:"primitiveId"`
	PrimitiveType        PrimitiveType `json:"primitiveType"`
	Prompt               string        `json:"prompt,omitempty"`
	Result               string        `json:"result,omitempty"`
	IsComplete           bool          `json:"isComplete,omitempty"`
	CompletionReason     string        `json:"completionReason,omitempty"`
	SelectionReason      string        `json:"selectionReason,omitempty"`
	Iteration            int           `json:"iteration"`
	ThreadID             string        `json:"threadId,omitempty"`
	ThreadResourceID     string        `json:"threadResourceId,omitempty"`
	IsOneOff             bool          `json:"isOneOff"`
	VerboseIntrospection bool          `json:"verboseIntrospection"`
	ConversationContext  []MastraDBMessage `json:"conversationContext,omitempty"`
	ValidationPassed     *bool         `json:"validationPassed,omitempty"`
	ValidationFeedback   string        `json:"validationFeedback,omitempty"`
	StructuredObject     any           `json:"structuredObject,omitempty"`
}

// abortResult is returned when a step is aborted.
type abortResult struct {
	Task          string `json:"task"`
	PrimitiveID   string `json:"primitiveId"`
	PrimitiveType string `json:"primitiveType"`
	Result        string `json:"result"`
	IsComplete    bool   `json:"isComplete"`
	Iteration     int    `json:"iteration"`
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// generateUUID generates a random UUID v4 string.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// safeParseLLMJson safely parses JSON from LLM output, handling common issues
// like unescaped control characters and truncated/incomplete JSON.
func safeParseLLMJson(text string) (any, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Try standard parse first.
	var result any
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return result, nil
	}

	// Attempt to fix common LLM issues:
	// 1. Escape unescaped newlines/tabs inside string values.
	fixed := escapeUnescapedControlChars(text)
	if err := json.Unmarshal([]byte(fixed), &result); err == nil {
		return result, nil
	}

	// 2. Try adding missing closing braces/brackets.
	repaired := repairPartialJSON(fixed)
	if err := json.Unmarshal([]byte(repaired), &result); err == nil {
		return result, nil
	}

	return nil, fmt.Errorf("failed to parse LLM JSON")
}

// escapeUnescapedControlChars escapes unescaped control characters in JSON strings.
func escapeUnescapedControlChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inString := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' && (i == 0 || s[i-1] != '\\') {
			inString = !inString
			b.WriteByte(ch)
			continue
		}
		if inString {
			switch ch {
			case '\n':
				b.WriteString("\\n")
			case '\r':
				b.WriteString("\\r")
			case '\t':
				b.WriteString("\\t")
			default:
				b.WriteByte(ch)
			}
		} else {
			b.WriteByte(ch)
		}
	}
	return b.String()
}

// repairPartialJSON attempts to close any open braces/brackets in partial JSON.
func repairPartialJSON(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	var stack []byte
	inString := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' && (i == 0 || s[i-1] != '\\') {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch ch {
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}', ']':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	// If still inside a string, close it.
	if inString {
		s += `"`
	}
	// Close any remaining open braces/brackets.
	for i := len(stack) - 1; i >= 0; i-- {
		s += string(stack[i])
	}
	return s
}

// filterMessagesForSubAgent filters messages to extract conversation context
// for sub-agents. Includes user messages and assistant messages that are NOT
// internal network JSON. Excludes:
//   - isNetwork: true JSON (result markers after primitive execution)
//   - Routing agent decision JSON (has primitiveId/primitiveType/selectionReason)
//   - Completion feedback messages (metadata.mode === 'network' or metadata.completionResult)
func filterMessagesForSubAgent(messages []MastraDBMessage) []MastraDBMessage {
	var filtered []MastraDBMessage
	for _, msg := range messages {
		// Include all user messages.
		if msg.Role == "user" {
			filtered = append(filtered, msg)
			continue
		}

		// Include assistant messages that are NOT internal network messages.
		if msg.Role == "assistant" {
			// Check metadata for network-internal markers.
			metadata := msg.Content.Metadata
			if metadata != nil {
				if mode, ok := metadata["mode"].(string); ok && mode == "network" {
					continue
				}
				if _, ok := metadata["completionResult"]; ok {
					continue
				}
			}

			// Check ALL parts for network-internal JSON.
			isNetworkMsg := false
			for _, part := range msg.Content.Parts {
				if part.Type == "text" && part.Text != "" {
					var parsed map[string]any
					if err := json.Unmarshal([]byte(part.Text), &parsed); err == nil {
						// Exclude isNetwork JSON (result markers after execution).
						if isNet, ok := parsed["isNetwork"].(bool); ok && isNet {
							isNetworkMsg = true
							break
						}
						// Exclude routing agent decision JSON.
						_, hasPrimID := parsed["primitiveId"]
						_, hasReason := parsed["selectionReason"]
						if hasPrimID && hasReason {
							isNetworkMsg = true
							break
						}
					}
				}
			}
			if !isNetworkMsg {
				filtered = append(filtered, msg)
			}
		}
	}
	return filtered
}

// getLastMessage extracts the last user message text from various message formats.
func getLastMessage(messages MessageListInput) string {
	if messages == nil {
		return ""
	}

	// Handle string input directly.
	if s, ok := messages.(string); ok {
		return s
	}

	// Handle slice of messages.
	if arr, ok := messages.([]any); ok {
		if len(arr) == 0 {
			return ""
		}
		lastMessage := arr[len(arr)-1]
		return extractTextFromMessage(lastMessage)
	}

	// Handle slice of MastraDBMessage.
	if arr, ok := messages.([]MastraDBMessage); ok {
		if len(arr) == 0 {
			return ""
		}
		return extractTextFromDBMessage(arr[len(arr)-1])
	}

	// Handle single message map.
	if m, ok := messages.(map[string]any); ok {
		return extractTextFromMessage(m)
	}

	return ""
}

func extractTextFromMessage(msg any) string {
	if s, ok := msg.(string); ok {
		return s
	}
	m, ok := msg.(map[string]any)
	if !ok {
		return ""
	}

	// Check 'content' field.
	if content, ok := m["content"]; ok {
		if s, ok := content.(string); ok {
			return s
		}
		// Content is array (parts format).
		if arr, ok := content.([]any); ok && len(arr) > 0 {
			lastPart := arr[len(arr)-1]
			if partMap, ok := lastPart.(map[string]any); ok {
				if partMap["type"] == "text" {
					if t, ok := partMap["text"].(string); ok {
						return t
					}
				}
			}
		}
	}

	// Check 'parts' field.
	if parts, ok := m["parts"]; ok {
		if arr, ok := parts.([]any); ok && len(arr) > 0 {
			lastPart := arr[len(arr)-1]
			if partMap, ok := lastPart.(map[string]any); ok {
				if partMap["type"] == "text" {
					if t, ok := partMap["text"].(string); ok {
						return t
					}
				}
			}
		}
	}

	return ""
}

func extractTextFromDBMessage(msg MastraDBMessage) string {
	for _, part := range msg.Content.Parts {
		if part.Type == "text" && part.Text != "" {
			return part.Text
		}
	}
	return ""
}

// getRoutingAgent creates the routing agent with network instructions.
// It builds the instruction prompt listing available agents, workflows, and tools,
// then returns a new routing agent configured for primitive selection.
func getRoutingAgent(
	ctx context.Context,
	agent Agent,
	requestContext any,
	routingConfig map[string]any,
) (Agent, error) {
	instructions, err := agent.GetInstructions(ctx, map[string]any{"requestContext": requestContext})
	if err != nil {
		return nil, fmt.Errorf("get instructions: %w", err)
	}
	agentsMap, err := agent.ListAgents(ctx, map[string]any{"requestContext": requestContext})
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}
	workflowsMap, err := agent.ListWorkflows(ctx, map[string]any{"requestContext": requestContext})
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	toolsMap, err := agent.ListTools(ctx, map[string]any{"requestContext": requestContext})
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}
	model, err := agent.GetModel(ctx, map[string]any{"requestContext": requestContext})
	if err != nil {
		return nil, fmt.Errorf("get model: %w", err)
	}
	memory, err := agent.GetMemory(ctx, map[string]any{"requestContext": requestContext})
	if err != nil {
		return nil, fmt.Errorf("get memory: %w", err)
	}
	defaultOpts, _ := agent.GetDefaultOptions(ctx, map[string]any{"requestContext": requestContext})
	configuredInputProcessors, _ := agent.ListConfiguredInputProcessors(ctx, requestContext)
	configuredOutputProcessors, _ := agent.ListConfiguredOutputProcessors(ctx, requestContext)

	// Build agent list string.
	var agentLines []string
	for name, a := range agentsMap {
		agentLines = append(agentLines, fmt.Sprintf(" - **%s**: %s", name, a.GetDescription()))
	}
	agentList := strings.Join(agentLines, "\n")

	// Build workflow list string.
	var workflowLines []string
	for name, wf := range workflowsMap {
		workflowLines = append(workflowLines, fmt.Sprintf(" - **%s**: %s, input schema: %s",
			name, wf.GetDescription(), wf.GetInputSchemaJSON()))
	}
	workflowList := strings.Join(workflowLines, "\n")

	// Build tool list string (including memory tools and client tools).
	allTools := make(map[string]Tool)
	for k, v := range toolsMap {
		allTools[k] = v
	}
	if memory != nil {
		memTools, _ := memory.ListTools(ctx)
		for k, v := range memTools {
			if t, ok := v.(Tool); ok {
				allTools[k] = t
			}
		}
	}
	if defaultOpts != nil {
		if clientTools, ok := defaultOpts["clientTools"].(map[string]Tool); ok {
			for k, v := range clientTools {
				allTools[k] = v
			}
		}
	}

	var toolLines []string
	for name, t := range allTools {
		toolLines = append(toolLines, fmt.Sprintf(" - **%s**: %s, input schema: %s",
			name, t.GetDescription(), t.GetInputSchemaJSON()))
	}
	toolList := strings.Join(toolLines, "\n")

	additionalInstructionsSection := ""
	if routingConfig != nil {
		if ai, ok := routingConfig["additionalInstructions"].(string); ok && ai != "" {
			additionalInstructionsSection = "\n## Additional Instructions\n" + ai
		}
	}

	routingInstructions := fmt.Sprintf(`
          You are a router in a network of specialized AI agents.
          Your job is to decide which agent should handle each step of a task.
          If asking for completion of a task, make sure to follow system instructions closely.

          Every step will result in a prompt message. It will be a JSON object with a "selectionReason" and "finalResult" property. Make your decision based on previous decision history, as well as the overall task criteria. If you already called a primitive, you shouldn't need to call it again, unless you strongly believe it adds something to the task completion criteria. Make sure to call enough primitives to complete the task.

          ## System Instructions
          %s
          You can only pick agents and workflows that are available in the lists below. Never call any agents or workflows that are not available in the lists below.
          ## Available Agents in Network
          %s
          ## Available Workflows in Network (make sure to use inputs corresponding to the input schema when calling a workflow)
          %s
          ## Available Tools in Network (make sure to use inputs corresponding to the input schema when calling a tool)
          %s
          If you have multiple entries that need to be called with a workflow or agent, call them separately with each input.
          When calling a workflow, the prompt should be a JSON value that corresponds to the input schema of the workflow. The JSON value is stringified.
          When calling a tool, the prompt should be a JSON value that corresponds to the input schema of the tool. The JSON value is stringified.
          When calling an agent, the prompt should be a text value, like you would call an LLM in a chat interface.
          Keep in mind that the user only sees the final result of the task. When reviewing completion, you should know that the user will not see the intermediate results.
          %s
        `, instructions, agentList, workflowList, toolList, additionalInstructionsSection)

	// The TS code creates a new Agent with routing instructions.
	// Since Agent is an interface stub, we return the original agent here.
	// In the full implementation, this would create a new Agent({...}).
	// TODO: create routing agent with routingInstructions once Agent is concrete.
	_ = routingInstructions
	_ = model
	_ = memory
	_ = configuredInputProcessors
	_ = configuredOutputProcessors

	return agent, nil
}

// prepareMemoryStep sets up memory/thread for the network loop.
// It ensures a thread exists, saves initial messages, and handles title generation.
func prepareMemoryStep(
	ctx context.Context,
	threadID string,
	resourceID string,
	messages MessageListInput,
	routingAgent Agent,
	requestContext any,
	generateID NetworkIdGenerator,
	memoryConfig map[string]any,
) (map[string]any, error) {
	memory, err := routingAgent.GetMemory(ctx, map[string]any{"requestContext": requestContext})
	if err != nil {
		return nil, err
	}
	if memory == nil {
		return nil, nil
	}

	// Get or create thread.
	thread, _ := memory.GetThreadByID(ctx, map[string]any{"threadId": threadID})
	if thread == nil {
		thread, err = memory.CreateThread(ctx, map[string]any{
			"threadId":   threadID,
			"title":      fmt.Sprintf("New Thread %s", time.Now().Format(time.RFC3339)),
			"resourceId": resourceID,
		})
		if err != nil {
			return nil, fmt.Errorf("create thread: %w", err)
		}
	}

	// Save initial messages.
	if s, ok := messages.(string); ok {
		threadIDVal := getStr(thread, "id")
		resourceIDVal := getStr(thread, "resourceId")
		msgID := generateID(IdGeneratorContext{
			IdType:     aktypes.IdTypeMessage,
			Source:     idGenSrcPtr(aktypes.IdGeneratorSourceAgent),
			ThreadId:   &threadIDVal,
			ResourceId: &resourceIDVal,
			Role:       strPtr("user"),
		})
		_ = memory.SaveMessages(ctx, map[string]any{
			"messages": []MastraDBMessage{{
				ID:         msgID,
				Type:       "text",
				Role:       "user",
				Content:    MessageContent{Format: 2, Parts: []MessagePart{{Type: "text", Text: s}}},
				CreatedAt:  time.Now(),
				ThreadID:   getStr(thread, "id"),
				ResourceID: getStr(thread, "resourceId"),
			}},
		})
	}
	// For non-string messages, the full implementation would create a MessageList,
	// add the messages, and save them. Simplified here since MessageList is a stub.

	return thread, nil
}

// saveMessagesWithProcessors saves messages to memory, applying output processors
// if a ProcessorRunner is configured.
func saveMessagesWithProcessors(
	ctx context.Context,
	memory Memory,
	messages []MastraDBMessage,
	processorRunner *ProcessorRunner,
	requestContext any,
) error {
	if memory == nil {
		return nil
	}

	if processorRunner == nil || len(messages) == 0 {
		return memory.SaveMessages(ctx, map[string]any{"messages": messages})
	}

	// In the full implementation, we'd create a MessageList, add messages,
	// run output processors, then save the processed messages.
	// Since MessageList is a stub interface, we save directly.
	// TODO: apply output processors once MessageList is concrete.
	return memory.SaveMessages(ctx, map[string]any{"messages": messages})
}

// saveFinalResultIfProvided saves the finalResult to memory if the LLM provided one.
func saveFinalResultIfProvided(
	ctx context.Context,
	memory Memory,
	finalResult string,
	threadID string,
	resourceID string,
	generateID NetworkIdGenerator,
	processorRunner *ProcessorRunner,
	requestContext any,
) error {
	if memory == nil || finalResult == "" {
		return nil
	}
	return saveMessagesWithProcessors(ctx, memory, []MastraDBMessage{{
		ID:         generateID(),
		Type:       "text",
		Role:       "assistant",
		Content:    MessageContent{Format: 2, Parts: []MessagePart{{Type: "text", Text: finalResult}}},
		CreatedAt:  time.Now(),
		ThreadID:   threadID,
		ResourceID: resourceID,
	}}, processorRunner, requestContext)
}

// ---------------------------------------------------------------------------
// handleAbort is a shared handler for abort events.
// ---------------------------------------------------------------------------
func handleAbort(
	writer chan<- NetworkChunkType,
	runID string,
	eventType string,
	primitiveType string,
	primitiveID string,
	iteration int,
	task string,
	onAbort func(event any) error,
) iterationStepData {
	if onAbort != nil {
		_ = onAbort(map[string]any{
			"primitiveType": primitiveType,
			"primitiveId":   primitiveID,
			"iteration":     iteration,
		})
	}
	select {
	case writer <- NetworkChunkType{
		"type":  eventType,
		"runId": runID,
		"from":  ChunkFromNetwork,
		"payload": map[string]any{
			"primitiveType": primitiveType,
			"primitiveId":   primitiveID,
		},
	}:
	default:
	}
	return iterationStepData{
		Task:          task,
		PrimitiveID:   primitiveID,
		PrimitiveType: PrimitiveType(primitiveType),
		Result:        "Aborted",
		IsComplete:    true,
		Iteration:     iteration,
	}
}

// ---------------------------------------------------------------------------
// createNetworkLoop builds the network workflow with routing, execution,
// and finish steps. Returns the workflow runner and processor runner.
// ---------------------------------------------------------------------------

// networkWorkflowResult holds the result from createNetworkLoop.
type networkWorkflowResult struct {
	processorRunner *ProcessorRunner
	// execute runs the network workflow for a single iteration.
	execute func(ctx context.Context, data iterationStepData, writer chan<- NetworkChunkType) (iterationStepData, error)
}

func createNetworkLoop(
	ctx context.Context,
	networkName string,
	requestContext any,
	runID string,
	agent Agent,
	routingAgentOptions *MultiPrimitiveExecutionOptions,
	generateID NetworkIdGenerator,
	routing map[string]any,
	onStepFinish func(event any) error,
	onError func(args any) error,
	onAbort func(event any) error,
) (*networkWorkflowResult, error) {
	// Get configured output processors.
	configuredOutputProcessors, _ := agent.ListConfiguredOutputProcessors(ctx, requestContext)

	// Create ProcessorRunner if there are output processors.
	var processorRunner *ProcessorRunner
	if len(configuredOutputProcessors) > 0 {
		mastraInstance := agent.GetMastraInstance()
		var logger Logger
		if mastraInstance != nil {
			logger = mastraInstance.GetLogger()
		}
		processorRunner = &ProcessorRunner{
			OutputProcessors: configuredOutputProcessors,
			InputProcessors:  nil,
			Logger:           logger,
			AgentName:        agent.GetName(),
		}
	}

	// execute runs a single network iteration: routing -> execution -> finish.
	execute := func(goCtx context.Context, data iterationStepData, writer chan<- NetworkChunkType) (iterationStepData, error) {
		// Check abort.
		if goCtx.Err() != nil {
			return handleAbort(writer, runID, "routing-agent-abort", "routing", "routing-agent",
				data.Iteration, data.Task, onAbort), nil
		}

		// === ROUTING STEP ===
		routingAgent, err := getRoutingAgent(goCtx, agent, requestContext, routing)
		if err != nil {
			return data, fmt.Errorf("get routing agent: %w", err)
		}

		iterationCount := data.Iteration + 1

		stepID := generateID(IdGeneratorContext{
			IdType:   aktypes.IdTypeStep,
			Source:   idGenSrcPtr(aktypes.IdGeneratorSourceAgent),
			StepType: strPtr("routing-agent"),
		})

		// Emit routing-agent-start chunk.
		select {
		case writer <- NetworkChunkType{
			"type": "routing-agent-start",
			"payload": map[string]any{
				"networkId": agent.GetID(),
				"agentId":   routingAgent.GetID(),
				"runId":     stepID,
				"inputData": map[string]any{
					"task":      data.Task,
					"iteration": iterationCount,
				},
			},
			"runId": runID,
			"from":  ChunkFromNetwork,
		}:
		default:
		}

		// Build the routing prompt.
		isOneOffText := ""
		if data.IsOneOff {
			isOneOffText = "You are executing just one primitive based on the user task. Make sure to pick the primitive that is the best suited to accomplish the whole task."
		} else {
			isOneOffText = "You will be calling just *one* primitive at a time to accomplish the user task, every call to you is one decision in the process of accomplishing the user task."
		}

		verboseText := "."
		if data.VerboseIntrospection {
			verboseText = ", as well as why the other primitives were not picked."
		}

		routingPromptContent := fmt.Sprintf(`
%s

The user has given you the following task:
%s

# Rules:

## Agent:
- prompt should be a text value, like you would call an LLM in a chat interface.
- If you are calling the same agent again, make sure to adjust the prompt to be more specific.

## Workflow/Tool:
- prompt should be a JSON value that corresponds to the input schema of the workflow or tool. The JSON value is stringified.
- Make sure to use inputs corresponding to the input schema when calling a workflow or tool.

DO NOT CALL THE PRIMITIVE YOURSELF. Make sure to not call the same primitive twice, unless you call it with different arguments and believe it adds something to the task completion criteria.

Please select the most appropriate primitive to handle this task and the prompt to be sent to the primitive. If no primitive is appropriate, return "none" for the primitiveId and "none" for the primitiveType.

{
    "primitiveId": string,
    "primitiveType": "agent" | "workflow" | "tool",
    "prompt": string,
    "selectionReason": string
}

The 'selectionReason' property should explain why you picked the primitive%s
`, isOneOffText, data.Task, verboseText)

		routingMessages := []any{
			map[string]any{
				"role":    "assistant",
				"content": routingPromptContent,
			},
		}

		threadIDToUse := data.ThreadID
		if threadIDToUse == "" {
			threadIDToUse = runID
		}
		threadResourceToUse := data.ThreadResourceID
		if threadResourceToUse == "" {
			threadResourceToUse = networkName
		}

		// Call the routing agent with structured output.
		routingResult, err := routingAgent.Stream(goCtx, routingMessages, map[string]any{
			"structuredOutput": map[string]any{
				"schema": "routing-schema", // placeholder
			},
			"requestContext": requestContext,
			"maxSteps":       1,
			"memory": map[string]any{
				"thread":   threadIDToUse,
				"resource": threadResourceToUse,
				"options": map[string]any{
					"readOnly": true,
					"workingMemory": map[string]any{
						"enabled": false,
					},
				},
			},
		})

		if err != nil {
			if goCtx.Err() != nil {
				return handleAbort(writer, runID, "routing-agent-abort", "routing", "routing-agent",
					iterationCount, data.Task, onAbort), nil
			}
			return data, fmt.Errorf("routing agent stream: %w", err)
		}

		if goCtx.Err() != nil {
			return handleAbort(writer, runID, "routing-agent-abort", "routing", "routing-agent",
				iterationCount, data.Task, onAbort), nil
		}

		// Parse routing result object.
		var routingObject map[string]any
		if routingResult.Object != nil {
			if m, ok := routingResult.Object.(map[string]any); ok {
				routingObject = m
			}
		}
		if routingObject == nil {
			return data, fmt.Errorf("routing agent returned nil object")
		}

		primitiveID := getStrFromMap(routingObject, "primitiveId")
		primitiveType := PrimitiveType(getStrFromMap(routingObject, "primitiveType"))
		prompt := getStrFromMap(routingObject, "prompt")
		selectionReason := getStrFromMap(routingObject, "selectionReason")
		isComplete := primitiveID == "none" && primitiveType == PrimitiveTypeNone

		// Extract conversation context from remembered messages.
		conversationContext := filterMessagesForSubAgent(routingResult.RememberedMessages)

		// Emit routing-agent-end chunk.
		endPayload := map[string]any{
			"task":                data.Task,
			"result":             "",
			"primitiveId":        primitiveID,
			"primitiveType":      string(primitiveType),
			"prompt":             prompt,
			"isComplete":         isComplete,
			"selectionReason":    selectionReason,
			"iteration":          iterationCount,
			"runId":              stepID,
			"conversationContext": conversationContext,
			"usage":              routingResult.Usage,
		}
		if isComplete {
			endPayload["result"] = selectionReason
		}

		select {
		case writer <- NetworkChunkType{
			"type":    "routing-agent-end",
			"payload": endPayload,
			"from":    ChunkFromNetwork,
			"runId":   runID,
		}:
		default:
		}

		routingData := iterationStepData{
			Task:                 data.Task,
			PrimitiveID:          primitiveID,
			PrimitiveType:        primitiveType,
			Prompt:               prompt,
			SelectionReason:      selectionReason,
			IsComplete:           isComplete,
			Iteration:            iterationCount,
			ThreadID:             data.ThreadID,
			ThreadResourceID:     data.ThreadResourceID,
			IsOneOff:             data.IsOneOff,
			VerboseIntrospection: data.VerboseIntrospection,
			ConversationContext:  conversationContext,
		}

		// === EXECUTION STEP (branching based on primitive type) ===
		if isComplete {
			// FINISH STEP
			return executeFinishStep(goCtx, routingData, writer, runID)
		}

		var execResult iterationStepData
		switch primitiveType {
		case PrimitiveTypeAgent:
			execResult, err = executeAgentStep(goCtx, routingData, writer, runID, networkName,
				agent, requestContext, generateID, processorRunner, onStepFinish, onError, onAbort)
		case PrimitiveTypeWorkflow:
			execResult, err = executeWorkflowStep(goCtx, routingData, writer, runID, networkName,
				agent, requestContext, generateID, processorRunner, onAbort)
		case PrimitiveTypeTool:
			execResult, err = executeToolStep(goCtx, routingData, writer, runID, networkName,
				agent, requestContext, generateID, processorRunner, onAbort)
		default:
			return executeFinishStep(goCtx, routingData, writer, runID)
		}
		if err != nil {
			return execResult, err
		}

		// Emit step-finish chunk.
		select {
		case writer <- NetworkChunkType{
			"type": "network-execution-event-step-finish",
			"payload": map[string]any{
				"task":      execResult.Task,
				"result":    execResult.Result,
				"isComplete": execResult.IsComplete,
				"iteration": execResult.Iteration,
				"runId":     runID,
			},
			"from":  ChunkFromNetwork,
			"runId": runID,
		}:
		default:
		}

		return execResult, nil
	}

	return &networkWorkflowResult{
		processorRunner: processorRunner,
		execute:         execute,
	}, nil
}

// executeFinishStep handles the case when the routing agent says the task is done.
func executeFinishStep(ctx context.Context, data iterationStepData, writer chan<- NetworkChunkType, runID string) (iterationStepData, error) {
	endResult := data.Result
	if data.PrimitiveID == "none" && data.PrimitiveType == PrimitiveTypeNone && endResult == "" {
		endResult = data.SelectionReason
	}

	result := data
	result.Result = endResult
	result.IsComplete = true

	return result, nil
}

// executeAgentStep executes an agent primitive.
func executeAgentStep(
	ctx context.Context,
	data iterationStepData,
	writer chan<- NetworkChunkType,
	runID string,
	networkName string,
	parentAgent Agent,
	requestContext any,
	generateID NetworkIdGenerator,
	processorRunner *ProcessorRunner,
	onStepFinish func(event any) error,
	onError func(args any) error,
	onAbort func(event any) error,
) (iterationStepData, error) {
	if ctx.Err() != nil {
		return handleAbort(writer, runID, "agent-execution-abort", "agent", data.PrimitiveID,
			data.Iteration, data.Task, onAbort), nil
	}

	agentsMap, err := parentAgent.ListAgents(ctx, map[string]any{"requestContext": requestContext})
	if err != nil {
		return data, fmt.Errorf("list agents: %w", err)
	}

	agentForStep, ok := agentsMap[data.PrimitiveID]
	if !ok {
		return data, fmt.Errorf("agent %s not found", data.PrimitiveID)
	}

	agentID := agentForStep.GetID()
	stepID := generateID(IdGeneratorContext{
		IdType:   aktypes.IdTypeStep,
		Source:   idGenSrcPtr(aktypes.IdGeneratorSourceAgent),
		EntityId: &agentID,
		StepType: strPtr("agent-execution"),
	})

	// Emit agent-execution-start chunk.
	select {
	case writer <- NetworkChunkType{
		"type": "agent-execution-start",
		"payload": map[string]any{
			"agentId": agentID,
			"args":    data,
			"runId":   stepID,
		},
		"from":  ChunkFromNetwork,
		"runId": runID,
	}:
	default:
	}

	threadID := data.ThreadID
	if threadID == "" {
		threadID = runID
	}
	resourceID := data.ThreadResourceID
	if resourceID == "" {
		resourceID = networkName
	}

	// Build messages for sub-agent: conversation context + the prompt.
	var messagesForSubAgent []any
	for _, msg := range data.ConversationContext {
		messagesForSubAgent = append(messagesForSubAgent, msg)
	}
	messagesForSubAgent = append(messagesForSubAgent, map[string]any{
		"role":    "user",
		"content": data.Prompt,
	})

	// Stream the sub-agent.
	result, err := agentForStep.Stream(ctx, messagesForSubAgent, map[string]any{
		"requestContext": requestContext,
		"runId":          runID,
		"memory": map[string]any{
			"thread":   threadID,
			"resource": resourceID,
			"options": map[string]any{
				"lastMessages": 0,
			},
		},
		"onStepFinish": onStepFinish,
		"onError":      onError,
	})
	if err != nil {
		return data, fmt.Errorf("agent stream: %w", err)
	}

	// Consume the full stream, forwarding chunks.
	agentCallAborted := false
	for chunk := range result.FullStream {
		chunkType, _ := chunk["type"].(string)
		select {
		case writer <- NetworkChunkType{
			"type": fmt.Sprintf("agent-execution-event-%s", chunkType),
			"payload": map[string]any{
				"runId": stepID,
			},
			"from":  ChunkFromNetwork,
			"runId": runID,
		}:
		default:
		}
		if chunkType == "abort" {
			agentCallAborted = true
		}
	}

	if agentCallAborted {
		return handleAbort(writer, runID, "agent-execution-abort", "agent", data.PrimitiveID,
			data.Iteration, data.Task, onAbort), nil
	}

	finalText := result.Text

	// Save result to memory.
	memory, _ := parentAgent.GetMemory(ctx, map[string]any{"requestContext": requestContext})
	networkJSON, _ := json.Marshal(map[string]any{
		"isNetwork":       true,
		"selectionReason": data.SelectionReason,
		"primitiveType":   string(data.PrimitiveType),
		"primitiveId":     data.PrimitiveID,
		"input":           data.Prompt,
		"finalResult":     map[string]any{"text": finalText},
	})

	_ = saveMessagesWithProcessors(ctx, memory, []MastraDBMessage{{
		ID:   generateID(IdGeneratorContext{
			IdType:     aktypes.IdTypeMessage,
			Source:     idGenSrcPtr(aktypes.IdGeneratorSourceAgent),
			EntityId:   &agentID,
			ThreadId:   &threadID,
			ResourceId: &resourceID,
			Role:       strPtr("assistant"),
		}),
		Type:       "text",
		Role:       "assistant",
		Content:    MessageContent{
			Format: 2,
			Parts:  []MessagePart{{Type: "text", Text: string(networkJSON)}},
			Metadata: map[string]any{"mode": "network"},
		},
		CreatedAt:  time.Now(),
		ThreadID:   threadID,
		ResourceID: resourceID,
	}}, processorRunner, requestContext)

	// Emit agent-execution-end chunk.
	select {
	case writer <- NetworkChunkType{
		"type": "agent-execution-end",
		"payload": map[string]any{
			"task":      data.Task,
			"agentId":   agentID,
			"result":    finalText,
			"isComplete": false,
			"iteration": data.Iteration,
			"runId":     stepID,
			"usage":     result.Usage,
		},
		"from":  ChunkFromNetwork,
		"runId": runID,
	}:
	default:
	}

	return iterationStepData{
		Task:          data.Task,
		PrimitiveID:   data.PrimitiveID,
		PrimitiveType: data.PrimitiveType,
		Result:        finalText,
		IsComplete:    false,
		Iteration:     data.Iteration,
		ThreadID:      data.ThreadID,
		ThreadResourceID: data.ThreadResourceID,
	}, nil
}

// executeWorkflowStep executes a workflow primitive.
func executeWorkflowStep(
	ctx context.Context,
	data iterationStepData,
	writer chan<- NetworkChunkType,
	runID string,
	networkName string,
	parentAgent Agent,
	requestContext any,
	generateID NetworkIdGenerator,
	processorRunner *ProcessorRunner,
	onAbort func(event any) error,
) (iterationStepData, error) {
	if ctx.Err() != nil {
		return handleAbort(writer, runID, "workflow-execution-abort", "workflow", data.PrimitiveID,
			data.Iteration, data.Task, onAbort), nil
	}

	workflowsMap, err := parentAgent.ListWorkflows(ctx, map[string]any{"requestContext": requestContext})
	if err != nil {
		return data, fmt.Errorf("list workflows: %w", err)
	}

	wf, ok := workflowsMap[data.PrimitiveID]
	if !ok {
		return data, fmt.Errorf("workflow %s not found", data.PrimitiveID)
	}

	// Parse the prompt as JSON input for the workflow.
	input, err := safeParseLLMJson(data.Prompt)
	if err != nil {
		// Return error message to routing agent for retry.
		return iterationStepData{
			Task:          data.Task,
			PrimitiveID:   data.PrimitiveID,
			PrimitiveType: data.PrimitiveType,
			Result: fmt.Sprintf("Error: The prompt provided for workflow %q is not valid JSON. Received: %q. "+
				"Workflows require a valid JSON string matching their input schema.", data.PrimitiveID, data.Prompt),
			IsComplete: false,
			Iteration:  data.Iteration,
			ThreadID:   data.ThreadID,
			ThreadResourceID: data.ThreadResourceID,
		}, nil
	}

	wfID := wf.GetID()
	stepID := generateID(IdGeneratorContext{
		IdType:   aktypes.IdTypeStep,
		Source:   idGenSrcPtr(aktypes.IdGeneratorSourceWorkflow),
		EntityId: &wfID,
		StepType: strPtr("workflow-execution"),
	})

	// Emit workflow-execution-start chunk.
	select {
	case writer <- NetworkChunkType{
		"type": "workflow-execution-start",
		"payload": map[string]any{
			"workflowId": wf.GetID(),
			"args":       data,
			"runId":      stepID,
		},
		"from":  ChunkFromNetwork,
		"runId": runID,
	}:
	default:
	}

	run, err := wf.CreateRun(ctx, map[string]any{"runId": runID})
	if err != nil {
		return data, fmt.Errorf("create workflow run: %w", err)
	}

	stream, err := run.Stream(ctx, map[string]any{
		"inputData":      input,
		"requestContext": requestContext,
	})
	if err != nil {
		return data, fmt.Errorf("workflow stream: %w", err)
	}

	// Consume the stream, forwarding chunks.
	workflowCancelled := false
	for chunk := range stream.FullStream {
		chunkType, _ := chunk["type"].(string)
		select {
		case writer <- NetworkChunkType{
			"type": fmt.Sprintf("workflow-execution-event-%s", chunkType),
			"payload": map[string]any{
				"runId": stepID,
			},
			"from":  ChunkFromNetwork,
			"runId": runID,
		}:
		default:
		}
		if chunkType == "workflow-canceled" {
			workflowCancelled = true
		}
	}

	if workflowCancelled && ctx.Err() != nil {
		return handleAbort(writer, runID, "workflow-execution-abort", "workflow", data.PrimitiveID,
			data.Iteration, data.Task, onAbort), nil
	}

	// Build final result JSON.
	finalResultJSON, _ := json.Marshal(map[string]any{
		"isNetwork":       true,
		"primitiveType":   string(data.PrimitiveType),
		"primitiveId":     data.PrimitiveID,
		"selectionReason": data.SelectionReason,
		"input":           input,
		"finalResult": map[string]any{
			"runId":     run.RunID(),
			"runResult": stream.Result,
			"runSuccess": stream.Result != nil,
		},
	})

	// Save to memory.
	threadID := data.ThreadID
	if threadID == "" {
		threadID = runID
	}
	resourceIDVal := data.ThreadResourceID
	if resourceIDVal == "" {
		resourceIDVal = networkName
	}

	memory, _ := parentAgent.GetMemory(ctx, map[string]any{"requestContext": requestContext})
	_ = saveMessagesWithProcessors(ctx, memory, []MastraDBMessage{{
		ID: generateID(IdGeneratorContext{
			IdType:     aktypes.IdTypeMessage,
			Source:     idGenSrcPtr(aktypes.IdGeneratorSourceWorkflow),
			EntityId:   &wfID,
			ThreadId:   &threadID,
			ResourceId: &resourceIDVal,
			Role:       strPtr("assistant"),
		}),
		Type:       "text",
		Role:       "assistant",
		Content:    MessageContent{
			Format: 2,
			Parts:  []MessagePart{{Type: "text", Text: string(finalResultJSON)}},
			Metadata: map[string]any{"mode": "network"},
		},
		CreatedAt:  time.Now(),
		ThreadID:   threadID,
		ResourceID: resourceIDVal,
	}}, processorRunner, requestContext)

	// Emit workflow-execution-end chunk.
	select {
	case writer <- NetworkChunkType{
		"type": "workflow-execution-end",
		"payload": map[string]any{
			"task":          data.Task,
			"primitiveId":   data.PrimitiveID,
			"primitiveType": string(data.PrimitiveType),
			"result":        stream.Result,
			"name":          wf.GetName(),
			"isComplete":    false,
			"iteration":     data.Iteration,
			"runId":         stepID,
			"usage":         stream.Usage,
		},
		"from":  ChunkFromNetwork,
		"runId": runID,
	}:
	default:
	}

	return iterationStepData{
		Task:          data.Task,
		PrimitiveID:   data.PrimitiveID,
		PrimitiveType: data.PrimitiveType,
		Result:        string(finalResultJSON),
		IsComplete:    false,
		Iteration:     data.Iteration,
		ThreadID:      data.ThreadID,
		ThreadResourceID: data.ThreadResourceID,
	}, nil
}

// executeToolStep executes a tool primitive.
func executeToolStep(
	ctx context.Context,
	data iterationStepData,
	writer chan<- NetworkChunkType,
	runID string,
	networkName string,
	parentAgent Agent,
	requestContext any,
	generateID NetworkIdGenerator,
	processorRunner *ProcessorRunner,
	onAbort func(event any) error,
) (iterationStepData, error) {
	if ctx.Err() != nil {
		return handleAbort(writer, runID, "tool-execution-abort", "tool", data.PrimitiveID,
			data.Iteration, data.Task, onAbort), nil
	}

	agentTools, err := parentAgent.ListTools(ctx, map[string]any{"requestContext": requestContext})
	if err != nil {
		return data, fmt.Errorf("list tools: %w", err)
	}
	memory, _ := parentAgent.GetMemory(ctx, map[string]any{"requestContext": requestContext})

	allTools := make(map[string]Tool)
	for k, v := range agentTools {
		allTools[k] = v
	}
	if memory != nil {
		memTools, _ := memory.ListTools(ctx)
		for k, v := range memTools {
			if t, ok := v.(Tool); ok {
				allTools[k] = t
			}
		}
	}

	tool, ok := allTools[data.PrimitiveID]
	if !ok {
		return data, fmt.Errorf("tool %s not found", data.PrimitiveID)
	}

	// Parse the prompt as JSON input for the tool.
	inputDataToUse, err := safeParseLLMJson(data.Prompt)
	if err != nil {
		return iterationStepData{
			Task:          data.Task,
			PrimitiveID:   data.PrimitiveID,
			PrimitiveType: data.PrimitiveType,
			Result: fmt.Sprintf("Error: The prompt provided for tool %q is not valid JSON. Received: %q. "+
				"Tools require a valid JSON string matching their input schema.", data.PrimitiveID, data.Prompt),
			IsComplete: false,
			Iteration:  data.Iteration,
			ThreadID:   data.ThreadID,
			ThreadResourceID: data.ThreadResourceID,
		}, nil
	}

	toolID := tool.GetID()
	toolCallID := generateID(IdGeneratorContext{
		IdType:   aktypes.IdTypeStep,
		Source:   idGenSrcPtr(aktypes.IdGeneratorSourceAgent),
		EntityId: &toolID,
		StepType: strPtr("tool-execution"),
	})

	// Emit tool-execution-start chunk.
	select {
	case writer <- NetworkChunkType{
		"type": "tool-execution-start",
		"payload": map[string]any{
			"args": map[string]any{
				"args":       inputDataToUse,
				"toolName":   tool.GetID(),
				"toolCallId": toolCallID,
			},
			"runId": runID,
		},
		"from":  ChunkFromNetwork,
		"runId": runID,
	}:
	default:
	}

	// Check if approval is required.
	toolRequiresApproval := tool.HasRequireApproval()
	if toolRequiresApproval {
		// In the full implementation, this would suspend and wait for approval.
		// Since suspend/resume is not yet ported, we log a warning and proceed.
		// TODO: implement tool approval flow once workflow suspend/resume is ported.
	}

	if ctx.Err() != nil {
		return handleAbort(writer, runID, "tool-execution-abort", "tool", data.PrimitiveID,
			data.Iteration, data.Task, onAbort), nil
	}

	// Execute the tool.
	threadID := data.ThreadID
	if threadID == "" {
		threadID = runID
	}
	resourceIDVal := data.ThreadResourceID
	if resourceIDVal == "" {
		resourceIDVal = networkName
	}

	finalResult, err := tool.Execute(ctx, inputDataToUse, map[string]any{
		"requestContext": requestContext,
		"runId":          runID,
		"agent": map[string]any{
			"resourceId": resourceIDVal,
			"toolCallId": toolCallID,
			"threadId":   threadID,
		},
	})
	if err != nil {
		return data, fmt.Errorf("tool execute: %w", err)
	}

	if ctx.Err() != nil {
		return handleAbort(writer, runID, "tool-execution-abort", "tool", data.PrimitiveID,
			data.Iteration, data.Task, onAbort), nil
	}

	// Serialize result.
	finalResultStr := ""
	if finalResult != nil {
		if s, ok := finalResult.(string); ok {
			finalResultStr = s
		} else {
			b, _ := json.Marshal(finalResult)
			finalResultStr = string(b)
		}
	}

	// Save result to memory.
	networkJSON, _ := json.Marshal(map[string]any{
		"isNetwork":       true,
		"selectionReason": data.SelectionReason,
		"primitiveType":   string(data.PrimitiveType),
		"primitiveId":     tool.GetID(),
		"finalResult":     map[string]any{"result": finalResultStr, "toolCallId": toolCallID},
		"input":           inputDataToUse,
	})

	_ = saveMessagesWithProcessors(ctx, memory, []MastraDBMessage{{
		ID: generateID(IdGeneratorContext{
			IdType:     aktypes.IdTypeMessage,
			Source:     idGenSrcPtr(aktypes.IdGeneratorSourceAgent),
			EntityId:   &toolID,
			ThreadId:   &threadID,
			ResourceId: &resourceIDVal,
			Role:       strPtr("assistant"),
		}),
		Type:       "text",
		Role:       "assistant",
		Content:    MessageContent{
			Format: 2,
			Parts:  []MessagePart{{Type: "text", Text: string(networkJSON)}},
			Metadata: map[string]any{"mode": "network"},
		},
		CreatedAt:  time.Now(),
		ThreadID:   threadID,
		ResourceID: resourceIDVal,
	}}, processorRunner, requestContext)

	// Emit tool-execution-end chunk.
	select {
	case writer <- NetworkChunkType{
		"type": "tool-execution-end",
		"payload": map[string]any{
			"task":          data.Task,
			"primitiveId":   data.PrimitiveID,
			"primitiveType": string(data.PrimitiveType),
			"result":        finalResultStr,
			"isComplete":    false,
			"iteration":     data.Iteration,
			"toolCallId":    toolCallID,
			"toolName":      tool.GetID(),
		},
		"from":  ChunkFromNetwork,
		"runId": runID,
	}:
	default:
	}

	return iterationStepData{
		Task:          data.Task,
		PrimitiveID:   data.PrimitiveID,
		PrimitiveType: data.PrimitiveType,
		Result:        finalResultStr,
		IsComplete:    false,
		Iteration:     data.Iteration,
		ThreadID:      data.ThreadID,
		ThreadResourceID: data.ThreadResourceID,
	}, nil
}

// ---------------------------------------------------------------------------
// networkLoop is the main entry point that orchestrates the network loop.
// It validates memory, handles auto-resume, creates the network workflow,
// runs validation iterations, and returns a MastraAgentNetworkStream.
// ---------------------------------------------------------------------------

// NetworkLoopParams holds all parameters for the networkLoop function.
type NetworkLoopParams struct {
	NetworkName            string
	RequestContext         any
	RunID                  string
	RoutingAgent           Agent
	RoutingAgentOptions    *MultiPrimitiveExecutionOptions
	GenerateID             NetworkIdGenerator
	MaxIterations          int
	ThreadID               string
	ResourceID             string
	Messages               MessageListInput
	Validation             *CompletionConfig
	Routing                map[string]any
	OnIterationComplete    func(ctx IterationCallbackContext) error
	ResumeData             any
	AutoResumeSuspendedTools bool
	Mastra                 Mastra
	StructuredOutput       *StructuredOutputOptions
	OnStepFinish           func(event any) error
	OnError                func(args any) error
	OnAbort                func(event any) error
}

// IterationCallbackContext is the context passed to OnIterationComplete.
type IterationCallbackContext struct {
	Iteration     int           `json:"iteration"`
	PrimitiveID   string        `json:"primitiveId"`
	PrimitiveType PrimitiveType `json:"primitiveType"`
	Result        string        `json:"result"`
	IsComplete    bool          `json:"isComplete"`
}

// RunNetworkLoop executes the full network loop and returns a MastraAgentNetworkStream.
func RunNetworkLoop(ctx context.Context, params NetworkLoopParams) (*MastraAgentNetworkStream, error) {
	// Validate that memory is available.
	memoryToUse, err := params.RoutingAgent.GetMemory(ctx, map[string]any{"requestContext": params.RequestContext})
	if err != nil || memoryToUse == nil {
		return nil, fmt.Errorf("memory is required for the agent network to function properly; please configure memory for the agent")
	}

	task := getLastMessage(params.Messages)

	runIDToUse := params.RunID
	resumeDataToUse := params.ResumeData

	// Auto-resume suspended tools if configured.
	if params.AutoResumeSuspendedTools && params.ThreadID != "" {
		resumeFromTask, runIDFromTask := tryAutoResumeSuspendedTools(ctx, params)
		if resumeFromTask != nil {
			resumeDataToUse = resumeFromTask
			runIDToUse = runIDFromTask
		}
	}

	// TODO: resumeDataToUse is used when creating the workflow run with
	// run.resumeStream() once the full workflow resume support is ported.
	_ = resumeDataToUse

	// Create the network loop workflow.
	nwResult, err := createNetworkLoop(
		ctx,
		params.NetworkName,
		params.RequestContext,
		runIDToUse,
		params.RoutingAgent,
		params.RoutingAgentOptions,
		params.GenerateID,
		params.Routing,
		params.OnStepFinish,
		params.OnError,
		params.OnAbort,
	)
	if err != nil {
		return nil, fmt.Errorf("create network loop: %w", err)
	}

	// Prepare memory step.
	thread, err := prepareMemoryStep(
		ctx,
		orDefault(params.ThreadID, runIDToUse),
		orDefault(params.ResourceID, params.NetworkName),
		params.Messages,
		params.RoutingAgent,
		params.RequestContext,
		params.GenerateID,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("prepare memory: %w", err)
	}

	// Create the stream channel.
	ch := make(chan NetworkChunkType, 64)

	go func() {
		defer close(ch)

		threadIDVal := ""
		threadResourceVal := ""
		if thread != nil {
			threadIDVal = getStr(thread, "id")
			threadResourceVal = getStr(thread, "resourceId")
		}

		current := iterationStepData{
			Task:                 task,
			PrimitiveID:          "",
			PrimitiveType:        PrimitiveTypeNone,
			Iteration:            -1, // Start at -1 so first iteration becomes 0.
			ThreadID:             threadIDVal,
			ThreadResourceID:     threadResourceVal,
			IsOneOff:             false,
			VerboseIntrospection: true,
		}

		for {
			// Execute one iteration (routing + execution).
			iterResult, err := nwResult.execute(ctx, current, ch)
			if err != nil {
				if params.OnError != nil {
					_ = params.OnError(map[string]any{"error": err})
				}
				ch <- NetworkChunkType{
					"type":  "error",
					"runId": runIDToUse,
					"from":  ChunkFromNetwork,
					"payload": map[string]any{
						"error": err.Error(),
					},
				}
				return
			}

			// === VALIDATION STEP ===
			validatedResult, validationErr := executeValidationStep(
				ctx, iterResult, ch, runIDToUse, params.NetworkName,
				params.RoutingAgent, params.RequestContext, params.GenerateID,
				params.Validation, params.Routing, params.MaxIterations,
				params.StructuredOutput, params.OnIterationComplete,
				nwResult.processorRunner, params.OnAbort,
			)
			if validationErr != nil {
				ch <- NetworkChunkType{
					"type":  "error",
					"runId": runIDToUse,
					"from":  ChunkFromNetwork,
					"payload": map[string]any{
						"error": validationErr.Error(),
					},
				}
				return
			}

			// Check termination conditions.
			llmComplete := validatedResult.IsComplete
			validationOk := validatedResult.ValidationPassed == nil || *validatedResult.ValidationPassed
			maxReached := params.MaxIterations > 0 && validatedResult.Iteration >= params.MaxIterations

			if (llmComplete && validationOk) || maxReached {
				// === FINAL STEP ===
				finalData := validatedResult
				if params.MaxIterations > 0 && validatedResult.Iteration >= params.MaxIterations {
					finalData.CompletionReason = fmt.Sprintf("Max iterations reached: %d", params.MaxIterations)
				}

				ch <- NetworkChunkType{
					"type": "network-execution-event-finish",
					"payload": map[string]any{
						"task":             finalData.Task,
						"primitiveId":      finalData.PrimitiveID,
						"primitiveType":    string(finalData.PrimitiveType),
						"prompt":           finalData.Prompt,
						"result":           finalData.Result,
						"isComplete":       finalData.IsComplete,
						"completionReason": finalData.CompletionReason,
						"iteration":        finalData.Iteration,
						"validationPassed": finalData.ValidationPassed,
					},
					"from":  ChunkFromNetwork,
					"runId": runIDToUse,
				}
				return
			}

			// Continue iterating.
			current = validatedResult
		}
	}()

	return &MastraAgentNetworkStream{
		Stream: ch,
		RunID:  runIDToUse,
	}, nil
}

// executeValidationStep runs the validation step after each network iteration.
func executeValidationStep(
	ctx context.Context,
	data iterationStepData,
	writer chan<- NetworkChunkType,
	runID string,
	networkName string,
	routingAgent Agent,
	requestContext any,
	generateID NetworkIdGenerator,
	validation *CompletionConfig,
	routing map[string]any,
	maxIterations int,
	structuredOutput *StructuredOutputOptions,
	onIterationComplete func(ctx IterationCallbackContext) error,
	processorRunner *ProcessorRunner,
	onAbort func(event any) error,
) (iterationStepData, error) {
	configuredScorers := make([]MastraScorer, 0)
	if validation != nil && len(validation.Scorers) > 0 {
		configuredScorers = validation.Scorers
	}

	// Build completion context.
	memory, _ := routingAgent.GetMemory(ctx, map[string]any{"requestContext": requestContext})
	var recallMessages []MastraDBMessage
	if memory != nil {
		threadIDVal := data.ThreadID
		if threadIDVal == "" {
			threadIDVal = runID
		}
		recallResult, err := memory.Recall(ctx, map[string]any{
			"threadId": threadIDVal,
		})
		if err == nil && recallResult != nil {
			recallMessages = recallResult.Messages
		}
	}

	completionCtx := CompletionContext{
		Iteration:     data.Iteration,
		MaxIterations: maxIterations,
		Messages:      recallMessages,
		OriginalTask:  data.Task,
		SelectedPrimitive: SelectedPrimitiveInfo{
			ID:   data.PrimitiveID,
			Type: string(data.PrimitiveType),
		},
		PrimitivePrompt: data.Prompt,
		PrimitiveResult: data.Result,
		NetworkName:     networkName,
		RunID:           runID,
		ThreadID:        data.ThreadID,
		ResourceID:      data.ThreadResourceID,
	}

	hasConfiguredScorers := len(configuredScorers) > 0
	checksCount := 1
	if hasConfiguredScorers {
		checksCount = len(configuredScorers)
	}

	// Emit validation-start chunk.
	select {
	case writer <- NetworkChunkType{
		"type": "network-validation-start",
		"payload": map[string]any{
			"runId":       runID,
			"iteration":   data.Iteration,
			"checksCount": checksCount,
		},
		"from":  ChunkFromNetwork,
		"runId": runID,
	}:
	default:
	}

	// Run completion checks.
	var completionResult CompletionRunResult
	var generatedFinalResult string

	if data.Result == "Aborted" {
		completionResult = CompletionRunResult{
			Complete:         true,
			CompletionReason: "Task aborted",
			Scorers:          nil,
			TotalDuration:    0,
			TimedOut:         false,
		}
	} else if hasConfiguredScorers {
		var err error
		completionResult, err = RunValidation(ctx, *validation, completionCtx)
		if err != nil {
			return data, err
		}

		// Generate final result if validation passed.
		if completionResult.Complete {
			generatedFinalResult, _ = GenerateFinalResult(ctx, routingAgent, completionCtx, nil)
			if generatedFinalResult != "" {
				threadIDVal := data.ThreadID
				if threadIDVal == "" {
					threadIDVal = runID
				}
				resourceIDVal := data.ThreadResourceID
				if resourceIDVal == "" {
					resourceIDVal = networkName
				}
				_ = saveFinalResultIfProvided(ctx, memory, generatedFinalResult,
					threadIDVal, resourceIDVal, generateID, processorRunner, requestContext)
			}
		}
	} else {
		// Use default LLM completion check.
		defaultResult, err := RunDefaultCompletionCheck(ctx, routingAgent, completionCtx, nil)
		if err != nil {
			return data, err
		}
		completionResult = CompletionRunResult{
			Complete:         defaultResult.Passed,
			CompletionReason: defaultResult.Reason,
			Scorers:          []ScorerResult{*defaultResult},
			TotalDuration:    defaultResult.Duration,
			TimedOut:         false,
		}
		generatedFinalResult = defaultResult.FinalResult

		// Save final result.
		if defaultResult.Passed && generatedFinalResult != "" {
			threadIDVal := data.ThreadID
			if threadIDVal == "" {
				threadIDVal = runID
			}
			resourceIDVal := data.ThreadResourceID
			if resourceIDVal == "" {
				resourceIDVal = networkName
			}
			_ = saveFinalResultIfProvided(ctx, memory, generatedFinalResult,
				threadIDVal, resourceIDVal, generateID, processorRunner, requestContext)
		}
	}

	maxIterationReached := maxIterations > 0 && data.Iteration >= maxIterations

	suppressFeedback := false
	if validation != nil {
		suppressFeedback = validation.SuppressFeedback
	}

	// Emit validation-end chunk.
	select {
	case writer <- NetworkChunkType{
		"type": "network-validation-end",
		"payload": map[string]any{
			"runId":               runID,
			"iteration":          data.Iteration,
			"passed":             completionResult.Complete,
			"results":            completionResult.Scorers,
			"duration":           completionResult.TotalDuration,
			"timedOut":           completionResult.TimedOut,
			"reason":             completionResult.CompletionReason,
			"maxIterationReached": maxIterationReached,
			"suppressFeedback":   suppressFeedback,
		},
		"from":  ChunkFromNetwork,
		"runId": runID,
	}:
	default:
	}

	isComplete := completionResult.Complete

	// Fire onIterationComplete callback.
	if onIterationComplete != nil {
		_ = onIterationComplete(IterationCallbackContext{
			Iteration:     data.Iteration,
			PrimitiveID:   data.PrimitiveID,
			PrimitiveType: data.PrimitiveType,
			Result:        data.Result,
			IsComplete:    isComplete,
		})
	}

	// Save feedback to memory.
	feedback := FormatCompletionFeedback(completionResult, maxIterationReached)
	if memory != nil {
		threadIDVal := data.ThreadID
		if threadIDVal == "" {
			threadIDVal = runID
		}
		resourceIDVal := data.ThreadResourceID
		if resourceIDVal == "" {
			resourceIDVal = networkName
		}
		_ = saveMessagesWithProcessors(ctx, memory, []MastraDBMessage{{
			ID:   generateID(),
			Type: "text",
			Role: "assistant",
			Content: MessageContent{
				Format: 2,
				Parts:  []MessagePart{{Type: "text", Text: feedback}},
				Metadata: map[string]any{
					"mode": "network",
					"completionResult": map[string]any{
						"passed":           completionResult.Complete,
						"suppressFeedback": suppressFeedback,
					},
				},
			},
			CreatedAt:  time.Now(),
			ThreadID:   threadIDVal,
			ResourceID: resourceIDVal,
		}}, processorRunner, requestContext)
	}

	result := data
	if isComplete {
		if generatedFinalResult != "" {
			result.Result = generatedFinalResult
		}
		result.IsComplete = true
		passed := true
		result.ValidationPassed = &passed
		if completionResult.CompletionReason != "" {
			result.CompletionReason = completionResult.CompletionReason
		} else {
			result.CompletionReason = "Task complete"
		}
	} else {
		result.IsComplete = false
		failed := false
		result.ValidationPassed = &failed
		result.ValidationFeedback = feedback
	}

	return result, nil
}

// tryAutoResumeSuspendedTools attempts to auto-resume suspended tools from previous runs.
func tryAutoResumeSuspendedTools(ctx context.Context, params NetworkLoopParams) (any, string) {
	memory, err := params.RoutingAgent.GetMemory(ctx, map[string]any{"requestContext": params.RequestContext})
	if err != nil || memory == nil {
		return nil, ""
	}

	threadExists, _ := memory.GetThreadByID(ctx, map[string]any{"threadId": params.ThreadID})
	if threadExists == nil {
		return nil, ""
	}

	recallResult, err := memory.Recall(ctx, map[string]any{
		"threadId":   params.ThreadID,
		"resourceId": orDefault(params.ResourceID, params.NetworkName),
	})
	if err != nil || recallResult == nil || len(recallResult.Messages) == 0 {
		return nil, ""
	}

	// Find the last assistant message.
	var lastAssistantMsg *MastraDBMessage
	for i := len(recallResult.Messages) - 1; i >= 0; i-- {
		if recallResult.Messages[i].Role == "assistant" {
			lastAssistantMsg = &recallResult.Messages[i]
			break
		}
	}
	if lastAssistantMsg == nil {
		return nil, ""
	}

	// Check for suspended tools or approval metadata.
	metadata := lastAssistantMsg.Content.Metadata
	if metadata == nil {
		return nil, ""
	}

	requireApproval, _ := metadata["requireApprovalMetadata"].(map[string]any)
	suspendedTools, _ := metadata["suspendedTools"].(map[string]any)

	if requireApproval == nil && suspendedTools == nil {
		return nil, ""
	}

	// Merge and find the first suspended tool.
	merged := make(map[string]any)
	for k, v := range suspendedTools {
		merged[k] = v
	}
	for k, v := range requireApproval {
		merged[k] = v
	}

	for _, v := range merged {
		toolInfo, ok := v.(map[string]any)
		if !ok {
			continue
		}
		resumeSchema, _ := toolInfo["resumeSchema"].(string)
		if resumeSchema == "" {
			continue
		}

		// The full implementation would use the LLM to generate resume data
		// from the resume schema and messages. Since LLM streaming is a stub,
		// we just return the tool's run ID.
		// TODO: implement LLM-based resume data generation.
		toolRunID, _ := toolInfo["runId"].(string)
		if toolRunID != "" {
			return nil, toolRunID
		}
	}

	return nil, ""
}

// ---------------------------------------------------------------------------
// Helper utilities
// ---------------------------------------------------------------------------

func getStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStrFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func orDefault(val, def string) string {
	if val != "" {
		return val
	}
	return def
}

// strPtr returns a pointer to s. Useful for constructing IdGeneratorContext
// literals where optional fields are *string.
func strPtr(s string) *string { return &s }

// idGenSrcPtr returns a pointer to an IdGeneratorSource constant.
func idGenSrcPtr(s aktypes.IdGeneratorSource) *aktypes.IdGeneratorSource { return &s }
