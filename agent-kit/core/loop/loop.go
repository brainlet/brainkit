// Ported from: packages/core/src/loop/loop.ts
package loop

import (
	"crypto/rand"
	"fmt"
	"time"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	loggerTypes "github.com/brainlet/brainkit/agent-kit/core/logger"
	workflows "github.com/brainlet/brainkit/agent-kit/core/loop/workflows"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
)

// ---------------------------------------------------------------------------
// Stub types for dependencies not yet ported
// ---------------------------------------------------------------------------

// MastraError is a stub for ../error.MastraBaseError.
// Stub: real error.MastraBaseError has typed enums (ErrorDomain, ErrorCategory),
// additional fields (details, message, cause), and unexported fields.
// This stub uses plain strings for ID/Domain/Category. Shape mismatch.
type MastraError struct {
	ID       string
	Domain   string
	Category string
}

func (e *MastraError) Error() string {
	return fmt.Sprintf("MastraError[%s/%s]: %s", e.Domain, e.Category, e.ID)
}

// ConsoleLogger is a minimal logger that satisfies logger.IMastraLogger.
// Used as fallback when no logger is provided.
type ConsoleLogger struct {
	Level string
}

func (l *ConsoleLogger) Debug(msg string, args ...any)                                    {}
func (l *ConsoleLogger) Info(msg string, args ...any)                                     {}
func (l *ConsoleLogger) Warn(msg string, args ...any)                                     {}
func (l *ConsoleLogger) Error(msg string, args ...any)                                    {}
func (l *ConsoleLogger) TrackException(err *mastraerror.MastraBaseError)                  {}
func (l *ConsoleLogger) GetTransports() map[string]loggerTypes.LoggerTransport            { return nil }
func (l *ConsoleLogger) ListLogs(transportID string, params *loggerTypes.ListLogsParams) (loggerTypes.LogResult, error) {
	return loggerTypes.LogResult{}, nil
}
func (l *ConsoleLogger) ListLogsByRunID(args *loggerTypes.ListLogsByRunIDFullArgs) (loggerTypes.LogResult, error) {
	return loggerTypes.LogResult{}, nil
}

// MastraModelOutput is a stub for ../stream/base/output.MastraModelOutput.
// Stub: real type has sync.Mutex, many private fields (status, streamFinished, etc.),
// and complex constructor/methods. This simplified version only provides
// SerializeState/DeserializeState stubs. Shape mismatch.
type MastraModelOutput struct {
	// Placeholder for model output state.
}

func (m *MastraModelOutput) SerializeState() any        { return nil }
func (m *MastraModelOutput) DeserializeState(state any) {}

// DestructurableOutput is a local type for the loop's return value.
// No direct equivalent exists in stream/base/output. The TS createDestructurableOutput()
// builds a wrapper around MastraModelOutput; this Go version is simplified.
type DestructurableOutput struct {
	ModelOutput *MastraModelOutput
	// Stream is the chunk channel from the workflow loop stream.
	// In the full implementation, MastraModelOutput would consume this stream.
	Stream <-chan map[string]any
}

// noopMessageList is a no-op adapter that satisfies workflows.MessageListInterface
// when the caller's MessageList doesn't implement it directly.
type noopMessageList struct{}

func (n *noopMessageList) Add(msg any, source string)     {}
func (n *noopMessageList) GetAllModelMessages() []any      { return nil }
func (n *noopMessageList) GetInputModelMessages() []any    { return nil }

// ---------------------------------------------------------------------------
// Error constants (mirrors TS ErrorDomain / ErrorCategory)
// ---------------------------------------------------------------------------

const (
	ErrorDomainLLM    = "LLM"
	ErrorCategoryUser = "USER"
)

// ---------------------------------------------------------------------------
// generateUUID generates a random UUID v4 string.
// ---------------------------------------------------------------------------
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// ---------------------------------------------------------------------------
// generateID generates a short alphanumeric ID (stub for ai-sdk generateId).
// ---------------------------------------------------------------------------
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// ---------------------------------------------------------------------------
// Loop is the main entry point that creates and returns a model output.
//
// It validates models, initialises internal state, constructs a LoopRun,
// invokes workflowLoopStream, and wraps the resulting stream in a
// MastraModelOutput.
//
// In TypeScript this is the exported `loop()` function.
// ---------------------------------------------------------------------------
func Loop(opts LoopOptions) (*DestructurableOutput, error) {
	// --- Logger fallback ---
	loggerToUse := opts.Logger
	if loggerToUse == nil {
		loggerToUse = &ConsoleLogger{Level: "debug"}
	}

	// --- Validate models ---
	if len(opts.Models) == 0 {
		err := &MastraError{
			ID:       "LOOP_MODELS_EMPTY",
			Domain:   ErrorDomainLLM,
			Category: ErrorCategoryUser,
		}
		return nil, err
	}

	firstModel := opts.Models[0]

	// --- Resolve runId ---
	runID := opts.RunID
	if runID == "" {
		if opts.IDGenerator != nil {
			agentSource := aktypes.IdGeneratorSourceAgent
			agentID := opts.AgentID
			threadID := internalThreadID(opts.Internal)
			resourceID := internalResourceID(opts.Internal)
			runID = opts.IDGenerator(&IdGeneratorContext{
				IdType:     aktypes.IdTypeRun,
				Source:     &agentSource,
				EntityId:   &agentID,
				ThreadId:   &threadID,
				ResourceId: &resourceID,
			})
		}
		if runID == "" {
			runID = generateUUID()
		}
	}

	// --- Build StreamInternal with defaults ---
	internal := &StreamInternal{
		Now:         func() time.Time { return time.Now() },
		GenerateID:  generateID,
		CurrentDate: func() time.Time { return time.Now() },
	}
	if opts.Internal != nil {
		if opts.Internal.Now != nil {
			internal.Now = opts.Internal.Now
		}
		if opts.Internal.GenerateID != nil {
			internal.GenerateID = opts.Internal.GenerateID
		}
		if opts.Internal.CurrentDate != nil {
			internal.CurrentDate = opts.Internal.CurrentDate
		}
		internal.SaveQueueManager = opts.Internal.SaveQueueManager
		internal.MemoryConfig = opts.Internal.MemoryConfig
		internal.ThreadID = opts.Internal.ThreadID
		internal.ResourceID = opts.Internal.ResourceID
		internal.Memory = opts.Internal.Memory
		internal.ThreadExists = opts.Internal.ThreadExists
		if opts.Internal.TransportRef.Current != nil {
			internal.TransportRef = opts.Internal.TransportRef
		}
	}

	// --- Start timestamp ---
	startTimestamp := internal.Now().UnixMilli()

	// --- Message ID ---
	messageID := ""
	if opts.ExperimentalGenerateMessageID != nil {
		messageID = opts.ExperimentalGenerateMessageID()
	}
	if messageID == "" {
		messageID = internal.GenerateID()
	}

	// --- Model output state (serialisation) ---
	var modelOutput *MastraModelOutput
	serializeStreamState := func() any {
		if modelOutput != nil {
			return modelOutput.SerializeState()
		}
		return nil
	}
	deserializeStreamState := func(state any) {
		if modelOutput != nil {
			modelOutput.DeserializeState(state)
		}
	}

	// --- Processor states ---
	processorStates := opts.ProcessorStates
	if processorStates == nil {
		processorStates = make(map[string]ProcessorState)
	}

	// --- Build LoopRun ---
	loopRun := &LoopRun{
		LoopOptions: LoopOptions{
			Mastra:                        opts.Mastra,
			ResumeContext:                 opts.ResumeContext,
			Models:                        opts.Models,
			Logger:                        loggerToUse,
			MessageList:                   opts.MessageList,
			IncludeRawChunks:              opts.IncludeRawChunks,
			Tools:                         opts.Tools,
			ModelSettings:                 opts.ModelSettings,
			OutputProcessors:              opts.OutputProcessors,
			AgentID:                       opts.AgentID,
			RequireToolApproval:           opts.RequireToolApproval,
			ToolCallConcurrency:           opts.ToolCallConcurrency,
			ToolCallID:                    opts.ToolCallID,
			Mode:                          opts.Mode,
			IDGenerator:                   opts.IDGenerator,
			ToolCallStreaming:             opts.ToolCallStreaming,
			ToolChoice:                    opts.ToolChoice,
			ActiveTools:                   opts.ActiveTools,
			Options:                       opts.Options,
			ProviderOptions:               opts.ProviderOptions,
			InputProcessors:               opts.InputProcessors,
			ExperimentalGenerateMessageID: opts.ExperimentalGenerateMessageID,
			StopWhen:                      opts.StopWhen,
			MaxSteps:                      opts.MaxSteps,
			StructuredOutput:              opts.StructuredOutput,
			ReturnScorerData:              opts.ReturnScorerData,
			DownloadRetries:               opts.DownloadRetries,
			DownloadConcurrency:           opts.DownloadConcurrency,
			ModelSpanTracker:              opts.ModelSpanTracker,
			AutoResumeSuspendedTools:      opts.AutoResumeSuspendedTools,
			AgentName:                     opts.AgentName,
			RequestContext:                opts.RequestContext,
			MaxProcessorRetries:           opts.MaxProcessorRetries,
			IsTaskComplete:                opts.IsTaskComplete,
			OnIterationComplete:           opts.OnIterationComplete,
			Workspace:                     opts.Workspace,
			ProcessorStates:               processorStates,
			ObservabilityContext:           opts.ObservabilityContext,
		},
		MessageID:      messageID,
		RunID:          runID,
		StartTimestamp: startTimestamp,
		Internal:       internal,
		StreamState: StreamState{
			Serialize:   serializeStreamState,
			Deserialize: deserializeStreamState,
		},
		MethodType: opts.MethodType,
	}

	// --- Resume: extract initial stream state from snapshot ---
	var initialStreamState any
	if opts.ResumeContext != nil {
		snapshot, ok := opts.ResumeContext.Snapshot.(map[string]any)
		if ok {
			if ctx, ok := snapshot["context"].(map[string]any); ok {
				for _, step := range ctx {
					stepMap, ok := step.(map[string]any)
					if !ok {
						continue
					}
					if status, ok := stepMap["status"].(string); ok && status == "suspended" {
						if sp, ok := stepMap["suspendPayload"].(map[string]any); ok {
							if ss, ok := sp["__streamState"]; ok {
								initialStreamState = ss
								break
							}
						}
					}
				}
			}
		}
	}

	// --- Adapt loop.LoopRun to workflows.LoopRun ---
	// The workflows package has its own LoopRun type (stub). We bridge
	// between the two by mapping fields.
	var wfInternal *workflows.StreamInternal
	if loopRun.Internal != nil {
		wfInternal = &workflows.StreamInternal{
			ThreadID:   loopRun.Internal.ThreadID,
			ResourceID: loopRun.Internal.ResourceID,
		}
		if loopRun.Internal.GenerateID != nil {
			wfInternal.GenerateID = loopRun.Internal.GenerateID
		}
	}

	// Adapt MessageList (empty interface) to workflows.MessageListInterface.
	// If the caller provides an object that satisfies the interface, use it;
	// otherwise wrap with a no-op adapter.
	var wfMessageList workflows.MessageListInterface
	if ml, ok := loopRun.MessageList.(workflows.MessageListInterface); ok {
		wfMessageList = ml
	} else {
		wfMessageList = &noopMessageList{}
	}

	// Adapt StopWhen ([]StopCondition / []any) to []any for workflows.
	var wfStopWhen []any
	for _, sw := range loopRun.StopWhen {
		wfStopWhen = append(wfStopWhen, sw)
	}

	// Adapt OnIterationComplete to the workflows signature.
	var wfOnIterComplete func(args any) error
	if loopRun.OnIterationComplete != nil {
		wfOnIterComplete = loopRun.OnIterationComplete
	}

	// Adapt ProcessorStates map.
	var wfProcessorStates map[string]any
	if loopRun.ProcessorStates != nil {
		wfProcessorStates = make(map[string]any)
		for k, v := range loopRun.ProcessorStates {
			wfProcessorStates[k] = v
		}
	}

	// Build workflow models slice.
	var wfModels []any
	for _, m := range loopRun.Models {
		wfModels = append(wfModels, m)
	}

	// Build the ResumeContext for workflows if present.
	var wfResumeCtx *workflows.ResumeContext
	if loopRun.ResumeContext != nil {
		wfResumeCtx = &workflows.ResumeContext{
			ResumeData: loopRun.ResumeContext.ResumeData,
			Snapshot:   loopRun.ResumeContext.Snapshot,
		}
	}

	// Build the LoopConfig for workflows if present.
	var wfOptions *workflows.LoopConfig
	if loopRun.Options != nil {
		wfOptions = &workflows.LoopConfig{}
		if loopRun.Options.OnChunk != nil {
			wfOptions.OnChunk = func(chunk any) error {
				if ct, ok := chunk.(ChunkType); ok {
					return loopRun.Options.OnChunk(ct)
				}
				return nil
			}
		}
		if loopRun.Options.OnError != nil {
			wfOptions.OnError = func(args any) error {
				if e, ok := args.(error); ok {
					return loopRun.Options.OnError(e)
				}
				return nil
			}
		}
		if loopRun.Options.OnFinish != nil {
			wfOptions.OnFinish = func(result any) error {
				return loopRun.Options.OnFinish(result)
			}
		}
		if loopRun.Options.OnStepFinish != nil {
			wfOptions.OnStepFinish = func(result any) error {
				return loopRun.Options.OnStepFinish(result)
			}
		}
	}

	wfLoopRun := workflows.LoopRun{
		ResumeContext:       wfResumeCtx,
		RequireToolApproval: loopRun.RequireToolApproval,
		Models:              wfModels,
		ToolChoice:          loopRun.ToolChoice,
		ModelSettings:       loopRun.ModelSettings,
		Internal:            wfInternal,
		MessageID:           loopRun.MessageID,
		RunID:               loopRun.RunID,
		MessageList:         wfMessageList,
		StartTimestamp:      loopRun.StartTimestamp,
		AgentID:             loopRun.AgentID,
		ToolCallID:          loopRun.ToolCallID,
		ToolCallConcurrency: loopRun.ToolCallConcurrency,
		Mastra:              loopRun.Mastra,
		Options:             wfOptions,
		ModelSpanTracker:    loopRun.ModelSpanTracker,
		Tools:               loopRun.Tools,
		Logger:              loopRun.Logger,
		AgentName:           loopRun.AgentName,
		MaxSteps:            loopRun.MaxSteps,
		StopWhen:            wfStopWhen,
		OnIterationComplete: wfOnIterComplete,
		IsTaskComplete:      loopRun.IsTaskComplete,
		OutputProcessors:    toAnySlice(loopRun.OutputProcessors),
		InputProcessors:     toAnySlice(loopRun.InputProcessors),
		ProcessorStates:     wfProcessorStates,
		ExperimentalGenerateMessageID: loopRun.ExperimentalGenerateMessageID,
	}

	// --- Create workflow loop stream ---
	streamResult := workflows.WorkflowLoopStream(wfLoopRun)
	_ = initialStreamState

	// --- Apply chunk tracing transform ---
	// TODO: wrap stream with modelSpanTracker.WrapStream() once ported.
	_ = streamResult

	// --- Build observability context ---
	// TODO: call createObservabilityContext() once ported.

	// --- Build MastraModelOutput ---
	modelOutput = &MastraModelOutput{}
	_ = firstModel // used for model metadata in MastraModelOutput

	// --- Return destructurable output ---
	// TODO: call createDestructurableOutput(modelOutput) once ported.
	// The streamResult.Stream channel provides the chunk stream that
	// MastraModelOutput would consume.
	return &DestructurableOutput{
		ModelOutput: modelOutput,
		Stream:      streamResult.Stream,
	}, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func internalThreadID(s *StreamInternal) string {
	if s != nil {
		return s.ThreadID
	}
	return ""
}

func internalResourceID(s *StreamInternal) string {
	if s != nil {
		return s.ResourceID
	}
	return ""
}

// toAnySlice converts a typed slice to []any for cross-package bridging.
func toAnySlice[T any](in []T) []any {
	if in == nil {
		return nil
	}
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}
