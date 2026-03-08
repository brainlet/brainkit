// Ported from: packages/core/src/loop/workflows/stream.ts
package workflows

import (
	"github.com/brainlet/brainkit/agent-kit/core/loop/workflows/agenticexecution"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	wftypes "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// ToolSet is re-declared in schema.go (same package). Not imported here.
// Stub: ai-kit only ported V3 (@ai-sdk/provider-v6). V5 ToolSet remains local.

// MastraDBMessage is a stub for ../../../agent/message-list.MastraDBMessage.
// Stub: real agent.MastraDBMessage has different content structure and additional fields.
// Importing agent could create coupling risk. Kept local with simplified shape.
type MastraDBMessage struct {
	ID         string         `json:"id"`
	Role       string         `json:"role"`
	Content    MessageContent `json:"content"`
	CreatedAt  any            `json:"createdAt"`
	ThreadID   string         `json:"threadId,omitempty"`
	ResourceID string         `json:"resourceId,omitempty"`
	Type       string         `json:"type,omitempty"`
}

// MessageContent holds message content with format and parts.
type MessageContent struct {
	Format   int            `json:"format,omitempty"`
	Parts    []MessagePart  `json:"parts,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// MessagePart is a union-like struct for different message part types.
type MessagePart struct {
	Type             string         `json:"type"`
	Text             string         `json:"text,omitempty"`
	Data             any            `json:"data,omitempty"`
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`
}

// MessageListInterface is a stub for the MessageList interface used by stream.
// Stub: real agent.MessageList is a struct with many methods. Importing agent from
// loop/workflows could create coupling risk. Kept as 3-method interface subset.
type MessageListInterface interface {
	Add(msg any, source string)
	GetAllModelMessages() []any
	GetInputModelMessages() []any
}

// ChunkType is a stub for ../../stream/types.ChunkType.
// Stub: real stream.ChunkType is a struct with Type, Payload, BaseChunkType fields.
// This stub uses map[string]any for flexibility. Shape mismatch.
type ChunkType = map[string]any

// ChunkFrom enumerates the source of chunks.
const ChunkFromAGENT = "agent"

// RequestContext is imported from requestcontext.
type RequestContext = requestcontext.RequestContext

// OutputWriter is imported from workflows.
type OutputWriter = wftypes.OutputWriter

// SafeEnqueue is a stub for ../../stream/base.SafeEnqueue.
// Stub: real function takes chan<- stream.ChunkType and stream.ChunkType (typed struct).
// This stub takes (any, any) with map[string]any channels. Signature mismatch.
func SafeEnqueue(controller any, chunk any) {
	if ch, ok := controller.(chan ChunkType); ok {
		select {
		case ch <- chunk.(ChunkType):
		default:
		}
	}
}

// SafeClose is a stub for ../../stream/base.SafeClose.
// Stub: real function takes chan<- stream.ChunkType. This stub takes any. Signature mismatch.
func SafeClose(controller any) {
	if ch, ok := controller.(chan ChunkType); ok {
		// Don't close here — the goroutine owns the channel via defer close.
		_ = ch
	}
}

// GetErrorFromUnknown is a stub for ../../error.GetErrorFromUnknown.
// Stub: real function returns *error.SerializableError with variadic *GetErrorOptions.
// This stub returns error with map[string]any opts. Signature mismatch.
func GetErrorFromUnknown(err any, opts map[string]any) error {
	if e, ok := err.(error); ok {
		return e
	}
	msg := "unknown error"
	if opts != nil {
		if fm, ok := opts["fallbackMessage"].(string); ok {
			msg = fm
		}
	}
	return &streamError{msg: msg}
}

type streamError struct {
	msg string
}

func (e *streamError) Error() string {
	return e.msg
}

// CreateObservabilityContext is a stub for ../../observability.CreateObservabilityContext.
// Stub: real function takes *obstypes.TracingContext and returns obstypes.ObservabilityContext.
// This stub takes any and returns map[string]any. Signature mismatch.
func CreateObservabilityContext(tracingContext any) map[string]any {
	return nil
}

// ---------------------------------------------------------------------------
// LoopRun stub (imported from loop/types in TS)
// ---------------------------------------------------------------------------

// LoopRun is a stub for ../types.LoopRun.
// Stub: can't import parent loop package (loop imports loop/workflows — would create cycle).
// Uses simplified field types (any instead of typed structs). Cycle risk + shape mismatch.
type LoopRun struct {
	ResumeContext        *ResumeContext       `json:"resumeContext,omitempty"`
	RequireToolApproval  bool                 `json:"requireToolApproval,omitempty"`
	Models               []any                `json:"models"`
	ToolChoice           any                  `json:"toolChoice,omitempty"`
	ModelSettings        any                  `json:"modelSettings,omitempty"`
	Internal             *StreamInternal      `json:"_internal,omitempty"`
	MessageID            string               `json:"messageId"`
	RunID                string               `json:"runId"`
	MessageList          MessageListInterface `json:"-"`
	StartTimestamp       int64                `json:"startTimestamp"`
	StreamState          *StreamState         `json:"-"`
	AgentID              string               `json:"agentId"`
	ToolCallID           string               `json:"toolCallId,omitempty"`
	ToolCallConcurrency  int                  `json:"toolCallConcurrency,omitempty"`
	Mastra               any                  `json:"-"`
	Options              *LoopConfig          `json:"-"`
	RequestContext       *RequestContext      `json:"-"`
	ModelSpanTracker     any                  `json:"-"`
	Controller           any                  `json:"-"`
	OutputWriter         OutputWriter         `json:"-"`
	// Additional fields from rest spread
	Tools                ToolSet              `json:"-"`
	Logger               any                  `json:"-"`
	AgentName            string               `json:"agentName,omitempty"`
	MaxSteps             int                  `json:"maxSteps,omitempty"`
	StopWhen             []any                `json:"-"`
	OnIterationComplete  func(args any) error `json:"-"`
	IsTaskComplete       any                  `json:"isTaskComplete,omitempty"`
	OutputProcessors     []any                `json:"-"`
	InputProcessors      []any                `json:"-"`
	ProcessorStates      map[string]any       `json:"-"`
	ExperimentalGenerateMessageID func() string `json:"-"`
}

// ResumeContext holds data needed to resume a suspended loop.
type ResumeContext struct {
	ResumeData any `json:"resumeData"`
	Snapshot   any `json:"snapshot"`
}

// StreamState provides serialization/deserialization for model output state.
type StreamState struct {
	Serialize   func() any      `json:"-"`
	Deserialize func(state any) `json:"-"`
}

// LoopConfig holds per-invocation callbacks and settings.
type LoopConfig struct {
	OnChunk      func(chunk any) error `json:"-"`
	OnError      func(args any) error  `json:"-"`
	OnFinish     func(result any) error `json:"-"`
	OnStepFinish func(result any) error `json:"-"`
	OnAbort      func(event any) error  `json:"-"`
	AbortSignal  any                    `json:"-"`
	PrepareStep  any                    `json:"-"`
}

// StreamInternal is defined in run_state.go within this package.
// It has ThreadID, ResourceID, GenerateID, CurrentDate fields.

// ---------------------------------------------------------------------------
// WorkflowLoopStream
// ---------------------------------------------------------------------------

// WorkflowLoopStreamResult holds the result of workflowLoopStream.
type WorkflowLoopStreamResult struct {
	// In TypeScript, this returns a ReadableStream<ChunkType>.
	// In Go, this would be a channel or reader interface.
	Stream <-chan ChunkType
}

// WorkflowLoopStream creates a readable stream that orchestrates the agentic
// loop workflow. It:
//  1. Sets up an output writer that handles data-* chunks (persisting them).
//  2. Creates the agentic loop workflow via createAgenticLoopWorkflow.
//  3. Registers the Mastra instance if present.
//  4. Builds initial iteration data from the message list.
//  5. Emits a 'start' chunk (unless resuming).
//  6. Creates and starts (or resumes) the workflow run.
//  7. On completion, emits 'finish' chunk; on failure, emits 'error' chunk.
//  8. Cleans up the workflow run.
//
// This is the main entry point that bridges the loop() function to the
// workflow-based execution engine.
func WorkflowLoopStream(params LoopRun) *WorkflowLoopStreamResult {
	ch := make(chan ChunkType, 64)

	go func() {
		defer close(ch)

		// Set up the output writer that handles data-* chunks and enqueues
		// all chunks to the stream channel.
		outputWriter := func(chunk ChunkType) error {
			// Handle data-* chunks — persist them to storage.
			chunkType, _ := chunk["type"].(string)
			if len(chunkType) > 5 && chunkType[:5] == "data-" && params.MessageID != "" {
				dataPart := MessagePart{
					Type: chunkType,
					Data: chunk["data"],
				}
				msg := MastraDBMessage{
					ID:   params.MessageID,
					Role: "assistant",
					Content: MessageContent{
						Format: 2,
						Parts:  []MessagePart{dataPart},
					},
				}
				if params.Internal != nil {
					msg.ThreadID = params.Internal.ThreadID
					msg.ResourceID = params.Internal.ResourceID
				}
				params.MessageList.Add(msg, "response")
			}
			// Enqueue to the stream channel (acts as safeEnqueue(controller, chunk)).
			select {
			case ch <- chunk:
			default:
			}
			return nil
		}

		// Build the OuterLLMRun for the agenticexecution package.
		// This maps the LoopRun fields to the agenticexecution.OuterLLMRun shape.
		var internalAny any
		if params.Internal != nil {
			internalAny = map[string]any{
				"threadId":   params.Internal.ThreadID,
				"resourceId": params.Internal.ResourceID,
			}
		}

		outerRun := agenticexecution.OuterLLMRun{
			Models:              params.Models,
			Internal:            internalAny,
			MessageID:           params.MessageID,
			RunID:               params.RunID,
			MessageList:         params.MessageList,
			Controller:          ch,
			OutputWriter:        outputWriter,
			StreamState:         params.StreamState,
			Tools:               params.Tools,
			ToolChoice:          params.ToolChoice,
			ModelSettings:       params.ModelSettings,
			Options:             params.Options,
			Logger:              params.Logger,
			AgentID:             params.AgentID,
			AgentName:           params.AgentName,
			MaxSteps:            params.MaxSteps,
			RequireToolApproval: params.RequireToolApproval,
			ToolCallConcurrency: params.ToolCallConcurrency,
			IsTaskComplete:      params.IsTaskComplete,
			OutputProcessors:    params.OutputProcessors,
			InputProcessors:     params.InputProcessors,
			ModelSpanTracker:    params.ModelSpanTracker,
			RequestContext:      params.RequestContext,
			ProcessorStates:     params.ProcessorStates,
			Mastra:              params.Mastra,
			ExperimentalGenerateMessageID: params.ExperimentalGenerateMessageID,
		}

		// Build stopWhen conditions — convert []any to []func(map[string]any)bool.
		var stopWhenFuncs []func(args map[string]any) bool
		for _, sw := range params.StopWhen {
			if fn, ok := sw.(func(args map[string]any) bool); ok {
				stopWhenFuncs = append(stopWhenFuncs, fn)
			}
		}

		// Build the onIterationComplete handler — adapt from func(any)error to
		// the agenticexecution.OnIterationCompleteHandler signature.
		var onIterComplete agenticexecution.OnIterationCompleteHandler
		if params.OnIterationComplete != nil {
			onIterComplete = func(ctx agenticexecution.IterationContext) (*agenticexecution.IterationResult, error) {
				// Pass the IterationContext as-is; the callback can type-assert.
				err := params.OnIterationComplete(ctx)
				return nil, err
			}
		}

		// Create the agentic loop workflow.
		agenticLoopWorkflow := agenticexecution.CreateAgenticLoopWorkflow(agenticexecution.AgenticLoopParams{
			OuterLLMRun:         outerRun,
			OnIterationComplete: onIterComplete,
			StopWhen:            stopWhenFuncs,
		})

		// Register Mastra instance if present.
		if params.Mastra != nil {
			agenticLoopWorkflow.RegisterMastra(params.Mastra)
		}

		// Build initial iteration data from the message list.
		var allMessages, userMessages []any
		if params.MessageList != nil {
			allMessages = params.MessageList.GetAllModelMessages()
			userMessages = params.MessageList.GetInputModelMessages()
		}

		initialData := map[string]any{
			"messageId": params.MessageID,
			"messages": map[string]any{
				"all":     allMessages,
				"user":    userMessages,
				"nonUser": []any{},
			},
			"output": map[string]any{
				"steps": []any{},
				"usage": map[string]any{"inputTokens": 0, "outputTokens": 0, "totalTokens": 0},
			},
			"metadata": map[string]any{},
			"stepResult": map[string]any{
				"reason":      "undefined",
				"warnings":    []any{},
				"isContinued": true,
				"totalUsage":  map[string]any{"inputTokens": 0, "outputTokens": 0, "totalTokens": 0},
			},
		}

		// Set up request context with tool approval flag if needed.
		requestContext := params.RequestContext
		if requestContext == nil {
			requestContext = requestcontext.NewRequestContext()
		}
		if params.RequireToolApproval {
			requestContext.Set("__mastra_requireToolApproval", true)
		}

		// Emit start chunk unless resuming.
		if params.ResumeContext == nil {
			ch <- ChunkType{
				"type":  "start",
				"runId": params.RunID,
				"from":  ChunkFromAGENT,
				"payload": map[string]any{
					"id":        params.AgentID,
					"messageId": params.MessageID,
				},
			}
		}

		// Execute the workflow.
		// In TS this uses createRun() + run.start/resume with observability
		// context. In Go we call Execute directly since the workflow engine
		// is simplified.
		executionResult, executionErr := agenticLoopWorkflow.Execute(initialData)

		if executionErr != nil {
			// Execution failed — emit error chunk.
			wrappedErr := GetErrorFromUnknown(executionErr, map[string]any{
				"fallbackMessage": "Unknown error in agent workflow stream",
			})

			ch <- ChunkType{
				"type":  "error",
				"runId": params.RunID,
				"from":  ChunkFromAGENT,
				"payload": map[string]any{
					"error": wrappedErr,
				},
			}

			// Call onError callback if configured.
			if params.Options != nil && params.Options.OnError != nil {
				_ = params.Options.OnError(map[string]any{"error": wrappedErr})
			}

			// Clean up the workflow run.
			_ = agenticLoopWorkflow.DeleteWorkflowRunByID(params.RunID)
			return
		}

		// Clean up the workflow run.
		_ = agenticLoopWorkflow.DeleteWorkflowRunByID(params.RunID)

		// Emit finish chunk.
		// The execution result contains the final iteration data with
		// stepResult indicating the reason for completion.
		finishPayload := map[string]any{}
		if resultMap, ok := executionResult.(map[string]any); ok {
			finishPayload = resultMap
		}

		// Ensure stepResult is present in the payload.
		if _, ok := finishPayload["stepResult"]; !ok {
			finishPayload["stepResult"] = map[string]any{
				"reason": "stop",
			}
		}

		ch <- ChunkType{
			"type":    "finish",
			"runId":   params.RunID,
			"from":    ChunkFromAGENT,
			"payload": finishPayload,
		}
	}()

	return &WorkflowLoopStreamResult{Stream: ch}
}
