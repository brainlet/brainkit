// Ported from: packages/core/src/agent/agent-legacy.ts
package agent

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	mempkg "github.com/brainlet/brainkit/agent-kit/core/memory"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// AgentLegacyCapabilities
// ---------------------------------------------------------------------------

// AgentLegacyCapabilities is the interface for accessing Agent methods needed
// by the legacy handler. This allows the legacy handler to work with Agent
// without directly accessing private members.
type AgentLegacyCapabilities interface {
	// Logger returns the agent's logger instance.
	Logger() AgentLegacyLogger
	// AgentName returns the agent name for logging.
	GetAgentName() string
	// AgentID returns the agent ID.
	GetAgentID() string
	// GetMastraInstance returns the Mastra instance for generating IDs.
	GetMastraInstance() Mastra
	// GetDefaultGenerateOptionsLegacy returns default generate options for legacy mode.
	GetDefaultGenerateOptionsLegacy(reqCtx *requestcontext.RequestContext) (AgentGenerateOptions, error)
	// GetDefaultStreamOptionsLegacy returns default stream options for legacy mode.
	GetDefaultStreamOptionsLegacy(reqCtx *requestcontext.RequestContext) (AgentStreamOptions, error)
	// HasOwnMemory checks if agent has own memory.
	HasOwnMemory() bool
	// GetInstructions returns the agent instructions.
	GetInstructions(ctx context.Context, reqCtx *requestcontext.RequestContext) (any, error)
	// GetLLM returns the LLM instance.
	GetLLM(reqCtx *requestcontext.RequestContext, model DynamicArgument) (MastraLLM, error)
	// GetMemory returns the memory instance.
	GetMemory(reqCtx *requestcontext.RequestContext) (MastraMemory, error)
	// ConvertTools converts tools for LLM usage.
	ConvertTools(ctx context.Context, params ConvertToolsParams) (map[string]CoreTool, error)
	// ConvertInstructionsToString converts instructions to string.
	ConvertInstructionsToString(instructions any) string
	// GetMostRecentUserMessage returns the most recent user message.
	GetMostRecentUserMessage(messages []MastraDBMessage) *MastraDBMessage
	// ResolveTitleGenerationConfig resolves title generation config.
	ResolveTitleGenerationConfig(config any) TitleGenerationResult
	// RunInputProcessors runs the agent's input processors.
	RunInputProcessors(args map[string]any) (InputProcessorsResult, error)
	// RunProcessInputStep runs the processInputStep phase (legacy path compatibility).
	RunProcessInputStep(args map[string]any) (InputProcessorsResult, error)
	// RunOutputProcessors runs the agent's output processors.
	RunOutputProcessors(args map[string]any) (OutputProcessorsResult, error)
	// SaveStepMessages saves step messages for per-step persistence.
	SaveStepMessages(args map[string]any) error
	// GenTitle generates a title for a thread.
	GenTitle(userMessage *MastraDBMessage, reqCtx *requestcontext.RequestContext, titleModel DynamicArgument, titleInstructions DynamicArgument) (string, error)
	// RunScorers runs configured scorers.
	RunScorers(args map[string]any) error
	// ListResolvedOutputProcessors returns resolved output processors.
	ListResolvedOutputProcessors(reqCtx *requestcontext.RequestContext) (any, error)
	// AgentNetworkAppend returns the agent network append flag.
	AgentNetworkAppend() bool
}

// InputProcessorsResult holds the result from running input processors.
type InputProcessorsResult struct {
	Tripwire *TripwireData
}

// OutputProcessorsResult holds the result from running output processors.
type OutputProcessorsResult struct {
	MessageList any
	Tripwire    *TripwireData
}

// AgentLegacyLogger is the minimal logger interface used by the legacy handler.
type AgentLegacyLogger interface {
	Debug(msg string, meta ...any)
	Error(msg string, meta ...any)
	Warn(msg string, meta ...any)
}

// ---------------------------------------------------------------------------
// AgentLegacyHandler
// ---------------------------------------------------------------------------

// AgentLegacyHandler encapsulates all legacy-specific streaming and
// generation logic for v1 models.
type AgentLegacyHandler struct {
	agent *Agent
}

// NewAgentLegacyHandler creates a new legacy handler for the given agent.
func NewAgentLegacyHandler(agent *Agent) *AgentLegacyHandler {
	return &AgentLegacyHandler{agent: agent}
}

// ---------------------------------------------------------------------------
// PrimitiveResult types
// ---------------------------------------------------------------------------

// PrimitiveBeforeResult holds the result of the before() phase of __primitive.
type PrimitiveBeforeResult struct {
	MessageObjects []any
	ConvertedTools map[string]CoreTool
	ThreadExists   bool
	Thread         *StorageThreadType
	MessageList    *MessageList
	AgentSpan      Span
	Tripwire       *TripwireData
}

// PrimitiveAfterResult holds the result of the after() phase of __primitive.
type PrimitiveAfterResult struct {
	ScoringData ScoringData
}

// ScoringData holds input/output data for scoring.
type ScoringData struct {
	Input  any `json:"input"`
	Output any `json:"output"`
}

// PrimitiveCallbacks holds the before/after callbacks returned by __primitive.
type PrimitiveCallbacks struct {
	Before func() (*PrimitiveBeforeResult, error)
	After  func(args PrimitiveAfterArgs) (*PrimitiveAfterResult, error)
}

// PrimitiveAfterArgs holds arguments for the after() callback.
type PrimitiveAfterArgs struct {
	Result           map[string]any
	Thread           *StorageThreadType
	ThreadID         string
	MemoryConfig     any
	OutputText       string
	RunID            string
	MessageList      *MessageList
	ThreadExists     bool
	StructuredOutput bool
	OverrideScorers  any
	AgentSpan        Span
}

// PrimitiveOptions holds options for __primitive.
type PrimitiveOptions struct {
	Instructions   any
	Messages       MessageListInput
	Context        []any
	Thread         *StorageThreadType
	MemoryConfig   any
	ResourceID     string
	RunID          string
	Toolsets       ToolsetsInput
	ClientTools    ToolsInput
	RequestContext *requestcontext.RequestContext
	WritableStream any
	MethodType     string // "generate" or "stream"
	TracingOptions any
}

// ---------------------------------------------------------------------------
// __primitive
// ---------------------------------------------------------------------------

// primitive prepares message list and tools before LLM execution and handles
// memory persistence after. This is the legacy version that only works with v1 models.
//
// Ported from TS: private __primitive({...})
func (h *AgentLegacyHandler) primitive(opts PrimitiveOptions) *PrimitiveCallbacks {
	agentName := h.agent.AgentName
	agentID := h.agent.ID
	logger := h.agent.Logger()
	resourceID := opts.ResourceID

	return &PrimitiveCallbacks{
		// before() phase: prepare tools, messages, memory
		Before: func() (*PrimitiveBeforeResult, error) {
			if logger != nil {
				logger.Debug(fmt.Sprintf("[Agents:%s] - Starting generation", agentName),
					"runId", opts.RunID)
			}

			// Create agent span for observability
			// TODO: implement once observability is ported:
			//   agentSpan = getOrCreateSpan({
			//       type: SpanType.AGENT_RUN,
			//       name: fmt.Sprintf("agent run: '%s'", agentID),
			//       entityType: EntityType.AGENT,
			//       entityId: agentID,
			//       entityName: agentName,
			//       input: map[string]any{"messages": opts.Messages},
			//       attributes: map[string]any{...},
			//       metadata: map[string]any{...},
			//   })
			var agentSpan Span
			_ = agentID

			// Get memory
			var reqCtx *requestcontext.RequestContext
			if opts.RequestContext != nil {
				reqCtx = opts.RequestContext
			}
			memory, err := h.agent.GetMemory(reqCtx)
			if err != nil {
				return nil, err
			}

			// Log tool enhancements
			var toolEnhancements string
			if len(opts.Toolsets) > 0 {
				toolEnhancements += fmt.Sprintf("toolsets present (%d tools)", len(opts.Toolsets))
			}
			if memory != nil && resourceID != "" {
				if toolEnhancements != "" {
					toolEnhancements += ", "
				}
				toolEnhancements += "memory and resourceId available"
			}
			if logger != nil {
				logger.Debug(fmt.Sprintf("[Agent:%s] - Enhancing tools: %s", agentName, toolEnhancements),
					"runId", opts.RunID)
			}

			var threadID string
			if opts.Thread != nil {
				threadID = opts.Thread.ID
			}

			// Determine method type for tools
			var agentMethodType AgentMethodType
			if opts.MethodType == "generate" {
				agentMethodType = "generateLegacy"
			} else {
				agentMethodType = "streamLegacy"
			}

			// Convert tools
			var mc *MemoryConfig
			if opts.MemoryConfig != nil {
				if cfg, ok := opts.MemoryConfig.(*MemoryConfig); ok {
					mc = cfg
				}
			}
			convertedTools, err := h.agent.ConvertTools(context.Background(), ConvertToolsParams{
				Toolsets:       opts.Toolsets,
				ClientTools:    opts.ClientTools,
				ThreadID:       threadID,
				ResourceID:     resourceID,
				RunID:          opts.RunID,
				RequestContext: reqCtx,
				MethodType:     agentMethodType,
				MemoryConfig:   mc,
			})
			if err != nil {
				return nil, err
			}

			// Create message list
			// TODO: use actual MessageList constructor once ported:
			//   messageList = NewMessageList(MessageListOptions{
			//       ThreadID: threadID,
			//       ResourceID: resourceID,
			//       GenerateMessageID: mastra.GenerateID,
			//       AgentNetworkAppend: capabilities.AgentNetworkAppend(),
			//   })
			//   .AddSystem(instructions || getInstructions())
			//   .Add(context, "context")
			var messageList *MessageList

			// Path 1: No memory or no thread/resource
			if memory == nil || (threadID == "" && resourceID == "") {
				// Add user messages
				// TODO: messageList.Add(opts.Messages, "user")

				// Run input processors
				// TODO: runInputProcessors({requestContext, messageList, ...})
				var tripwire *TripwireData

				// Run processInputStep for step 0 (legacy path compatibility)
				// TODO: runProcessInputStep({requestContext, messageList, stepNumber: 0})

				return &PrimitiveBeforeResult{
					MessageObjects: nil, // TODO: messageList.Get.All.Prompt()
					ConvertedTools: convertedTools,
					ThreadExists:   false,
					Thread:         nil,
					MessageList:    messageList,
					AgentSpan:      agentSpan,
					Tripwire:       tripwire,
				}, nil
			}

			// Path 2: Memory is configured - validate required IDs
			if threadID == "" || resourceID == "" {
				errMsg := fmt.Sprintf(
					`A resourceId and a threadId must be provided when using Memory. Saw threadId "%s" and resourceId "%s"`,
					threadID, resourceID,
				)
				if logger != nil {
					logger.Error(errMsg)
				}
				return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "AGENT_MEMORY_MISSING_RESOURCE_ID",
					Domain:   mastraerror.ErrorDomainAgent,
					Category: mastraerror.ErrorCategoryUser,
					Details: map[string]any{
						"agentName":  agentName,
						"threadId":   threadID,
						"resourceId": resourceID,
					},
					Text: errMsg,
				})
			}

			// Log memory persistence info
			if logger != nil {
				logger.Debug(
					fmt.Sprintf("[Agent:%s] - Memory persistence enabled: store=memory, resourceId=%s",
						agentName, resourceID),
					"runId", opts.RunID,
					"resourceId", resourceID,
					"threadId", threadID,
				)
			}

			// Thread management:
			// 1. Try to get existing thread by ID
			// 2. If exists, check if metadata needs updating
			// 3. If not exists, create new thread with saveThread: true
			var threadObject *StorageThreadType
			var existingThread bool

			existing, getErr := memory.GetThreadById(context.Background(), threadID)
			if getErr != nil {
				// Non-fatal: thread may not exist yet.
				if logger != nil {
					logger.Debug(fmt.Sprintf("[Agent:%s] - Thread %s not found: %v", agentName, threadID, getErr))
				}
			}
			if existing != nil {
				existingThread = true
				threadObject = existing
				// In TS: if thread.metadata && !deepEqual(existing.metadata, thread.metadata) {
				//     threadObject = memory.saveThread(existing with updated metadata)
				// }
				// TODO: wire metadata update once memory.SaveThread is available.
			} else {
				// Thread doesn't exist - will be created in the After phase
				// or via savePerStep in prepareLLMOptions.
				threadObject = opts.Thread
			}

			// Set memory context in RequestContext for processors to access.
			if reqCtx != nil {
				reqCtx.Set("MastraMemory", map[string]any{
					"thread":       threadObject,
					"resourceId":   resourceID,
					"memoryConfig": opts.MemoryConfig,
				})
			}

			// Add user messages
			// TODO: messageList.Add(opts.Messages, "user")

			// Run input processors
			// TODO: result, err := capabilities.RunInputProcessors({requestContext, messageList, ...})
			var tripwire *TripwireData

			// Run processInputStep for step 0 (legacy path compatibility)
			// TODO: inputStepResult := capabilities.RunProcessInputStep({requestContext, messageList, stepNumber: 0})
			// if inputStepResult.Tripwire != nil { return early with tripwire }

			return &PrimitiveBeforeResult{
				ConvertedTools: convertedTools,
				Thread:         threadObject,
				MessageList:    messageList,
				MessageObjects: nil, // TODO: messageList.Get.All.Prompt()
				AgentSpan:      agentSpan,
				Tripwire:       tripwire,
				ThreadExists:   existingThread,
			}, nil
		},

		// after() phase: persist memory, generate title, run scorers
		After: func(args PrimitiveAfterArgs) (*PrimitiveAfterResult, error) {
			if logger != nil {
				logger.Debug(fmt.Sprintf("[Agent:%s] - Post processing LLM response", agentName),
					"runId", args.RunID,
					"threadId", args.ThreadID,
				)
			}

			// Build response message list for checking working memory usage
			// TODO: implement once MessageList is ported:
			//   responseMessageList = NewMessageList({threadId, resourceId, generateMessageId, agentNetworkAppend})
			//     .Add(result.response.messages, "response")
			//     .Get.All.Core()

			// Check if working memory was used (requires inspecting tool calls)
			// TODO: check for "updateWorkingMemory" tool call in responses

			// Get latest memory and thread
			var reqCtx *requestcontext.RequestContext
			if opts.RequestContext != nil {
				reqCtx = opts.RequestContext
			}
			memory, err := h.agent.GetMemory(reqCtx)
			if err != nil {
				return nil, err
			}

			thread := args.Thread
			threadID := args.ThreadID
			messageList := args.MessageList

			if memory != nil && resourceID != "" && thread != nil {
				// Add LLM response messages to the list
				// TODO: implement once MessageList is ported:
				//   responseMessages := result["response"].(map[string]any)["messages"]
				//   if responseMessages == nil && result["object"] != nil {
				//       responseMessages = []map[string]any{{
				//           "role": "assistant",
				//           "content": []map[string]any{{"type": "text", "text": outputText}},
				//       }}
				//   }
				//   messageList.Add(responseMessages, "response")

				// Create thread if it doesn't exist yet
				if !args.ThreadExists {
					var mc *MemoryConfig
					if args.MemoryConfig != nil {
						if cfg, ok := args.MemoryConfig.(*MemoryConfig); ok {
							mc = cfg
						}
					}
					saveThread := true
					_, createErr := memory.CreateThread(context.Background(), mempkg.CreateThreadOpts{
						ThreadID:     threadID,
						ResourceID:   resourceID,
						Title:        thread.Title,
						Metadata:     thread.Metadata,
						MemoryConfig: mc,
						SaveThread:   &saveThread,
					})
					if createErr != nil && logger != nil {
						logger.Error(fmt.Sprintf("[Agent:%s] - Error creating thread in after phase: %v", agentName, createErr))
					}
				}

				// Title generation (parallel with message saving)
				// TODO: implement once memory config and title generation are ported:
				//   config := memory.GetMergedThreadConfig(memoryConfig)
				//   userMessage := capabilities.GetMostRecentUserMessage(messageList.Get.All.UI())
				//   {shouldGenerate, model, instructions} := capabilities.ResolveTitleGenerationConfig(config.GenerateTitle)
				//   if shouldGenerate && thread.Title == "" && userMessage != nil {
				//       title := capabilities.GenTitle(userMessage, requestContext, observabilityContext, model, instructions)
				//       if title != "" { memory.CreateThread({threadId, resourceId, memoryConfig, title, metadata}) }
				//   }
			} else {
				// No memory - still add response messages to list for scoring
				// TODO: add response messages to messageList
			}

			// Run scorers
			// TODO: capabilities.RunScorers({messageList, runId, requestContext, structuredOutput, overrideScorers, threadId, resourceId, observabilityContext})
			_ = memory
			_ = threadID
			_ = messageList

			// Build scoring data
			scoringData := ScoringData{
				// TODO: populate with actual message list data once ported:
				//   Input: map[string]any{
				//       "inputMessages": messageList.GetPersisted.Input.UI(),
				//       "rememberedMessages": messageList.GetPersisted.Remembered.UI(),
				//       "systemMessages": messageList.GetSystemMessages(),
				//       "taggedSystemMessages": messageList.GetPersisted.TaggedSystemMessages,
				//   },
				//   Output: messageList.GetPersisted.Response.UI(),
			}

			// End agent span
			// TODO: agentSpan.End({output: {text, object, files}})

			return &PrimitiveAfterResult{
				ScoringData: scoringData,
			}, nil
		},
	}
}

// ---------------------------------------------------------------------------
// PrepareLLMOptionsResult
// ---------------------------------------------------------------------------

// PrepareLLMOptionsResult holds the assembled options for LLM calls.
type PrepareLLMOptionsResult struct {
	LLM    MastraLLM
	Before func() (*LegacyBeforeResult, error)
	After  func(args LegacyAfterArgs) (*PrimitiveAfterResult, error)
}

// LegacyBeforeResult extends PrimitiveBeforeResult with resolved LLM options.
type LegacyBeforeResult struct {
	// From PrimitiveBeforeResult
	MessageObjects []any
	ConvertedTools map[string]CoreTool
	ThreadExists   bool
	Thread         *StorageThreadType
	MessageList    *MessageList
	AgentSpan      Span
	Tripwire       *TripwireData

	// Additional resolved options
	Tools          map[string]CoreTool
	RunID          string
	Temperature    *float64
	ToolChoice     any
	ThreadID       string
	ResourceID     string
	RequestContext *requestcontext.RequestContext
	OnStepFinish   func(props any) error
	Output         any
	ExperimentalOutput any
	MaxSteps       *int
}

// LegacyAfterArgs holds arguments for the legacy after callback.
type LegacyAfterArgs struct {
	Result           any
	OutputText       string
	StructuredOutput bool
	AgentSpan        Span
	OverrideScorers  any
}

// prepareLLMOptions prepares options and handlers for LLM text/object generation or streaming.
// This is the legacy version that only works with v1 models.
//
// Ported from TS: private async prepareLLMOptions<...>(messages, options, methodType)
func (h *AgentLegacyHandler) prepareLLMOptions(
	messages MessageListInput,
	options map[string]any,
	methodType string,
) (*PrepareLLMOptionsResult, error) {
	agentName := h.agent.AgentName
	logger := h.agent.Logger()
	reqCtx := extractRequestContext(options)

	// Reserved keys from requestContext take precedence for security.
	// This allows middleware to securely set resourceId/threadId based on authenticated user.
	var resourceIDFromContext string
	var threadIDFromContext string
	if reqCtx != nil {
		if v := reqCtx.Get(requestcontext.MastraResourceIDKey); v != nil {
			if s, ok := v.(string); ok {
				resourceIDFromContext = s
			}
		}
		if v := reqCtx.Get(requestcontext.MastraThreadIDKey); v != nil {
			if s, ok := v.(string); ok {
				threadIDFromContext = s
			}
		}
	}

	// Resolve thread from args
	var threadFromArgs *StorageThreadType
	if threadIDFromContext != "" {
		threadFromArgs = &StorageThreadType{ID: threadIDFromContext}
	} else {
		// Build ResolveThreadArgs from options map
		var resolveArgs ResolveThreadArgs
		if memOpt, ok := options["memory"].(*AgentMemoryOption); ok {
			resolveArgs.Memory = memOpt
		}
		if tid, ok := options["threadId"].(string); ok {
			resolveArgs.ThreadID = tid
		}
		resolved := ResolveThreadIdFromArgs(resolveArgs)
		if resolved != nil {
			threadFromArgs = &resolved.StorageThreadType
		}
	}

	// Resolve resourceId
	resourceID := resourceIDFromContext
	if resourceID == "" {
		if memOpts, ok := options["memory"].(map[string]any); ok {
			if r, ok := memOpts["resource"].(string); ok {
				resourceID = r
			}
		}
	}
	if resourceID == "" {
		if r, ok := options["resourceId"].(string); ok {
			resourceID = r
		}
	}

	// Resolve memory config
	var memoryConfig any
	if memOpts, ok := options["memory"].(map[string]any); ok {
		memoryConfig = memOpts["options"]
	}
	if memoryConfig == nil {
		memoryConfig = options["memoryOptions"]
	}

	// Warn if memory args provided but no memory configured
	if resourceID != "" && threadFromArgs != nil && !h.agent.HasOwnMemory() {
		if logger != nil {
			logger.Warn(fmt.Sprintf(
				"[Agent:%s] - No memory is configured but resourceId and threadId were passed in args. This will not work.",
				agentName,
			))
		}
	}

	// Generate run ID
	runID := ""
	if r, ok := options["runId"].(string); ok && r != "" {
		runID = r
	}
	if runID == "" {
		if h.agent.mastra != nil {
			// Try to call GenerateID if Mastra implements it
			if gen, ok := h.agent.mastra.(interface{ GenerateID() string }); ok {
				runID = gen.GenerateID()
			}
		}
	}
	if runID == "" {
		runID = generateUUID()
	}

	// Resolve instructions
	instructions, err := h.agent.GetInstructions(context.Background(), reqCtx)
	if err != nil {
		return nil, err
	}
	if instr, ok := options["instructions"]; ok && instr != nil {
		instructions = instr
	}

	// Get LLM
	llm, err := h.agent.GetLLM(reqCtx, nil)
	if err != nil {
		return nil, err
	}

	// Get memory
	memory, err := h.agent.GetMemory(reqCtx)
	if err != nil {
		return nil, err
	}

	// Extract option fields
	contextMsgs, _ := options["context"].([]any)
	toolsets, _ := options["toolsets"].(ToolsetsInput)
	clientTools, _ := options["clientTools"].(ToolsInput)
	temperature, _ := options["temperature"].(*float64)
	toolChoice := options["toolChoice"]
	if toolChoice == nil {
		toolChoice = "auto"
	}
	savePerStep, _ := options["savePerStep"].(bool)
	onStepFinish := options["onStepFinish"]
	onFinish := options["onFinish"]
	output := options["output"]
	experimentalOutput := options["experimental_output"]
	maxSteps, _ := options["maxSteps"].(*int)
	tracingOptions := options["tracingOptions"]
	scorers := options["scorers"]

	// Build __primitive
	primitiveCallbacks := h.primitive(PrimitiveOptions{
		Messages:       messages,
		Instructions:   instructions,
		Context:        contextMsgs,
		Thread:         threadFromArgs,
		MemoryConfig:   memoryConfig,
		ResourceID:     resourceID,
		RunID:          runID,
		Toolsets:        toolsets,
		ClientTools:     clientTools,
		RequestContext:  reqCtx,
		WritableStream:  options["writableStream"],
		MethodType:      methodType,
		TracingOptions:  tracingOptions,
	})

	// Closed-over state shared between before and after
	var messageList *MessageList
	var thread *StorageThreadType
	var threadExists bool
	var threadCreatedByStep bool

	return &PrepareLLMOptionsResult{
		LLM: llm,

		Before: func() (*LegacyBeforeResult, error) {
			beforeResult, err := primitiveCallbacks.Before()
			if err != nil {
				return nil, err
			}

			threadExists = beforeResult.ThreadExists
			threadCreatedByStep = false
			messageList = beforeResult.MessageList
			thread = beforeResult.Thread

			var threadID string
			if thread != nil {
				threadID = thread.ID
			}

			result := &LegacyBeforeResult{
				MessageObjects:     beforeResult.MessageObjects,
				ConvertedTools:     beforeResult.ConvertedTools,
				ThreadExists:       beforeResult.ThreadExists,
				Thread:             beforeResult.Thread,
				MessageList:        beforeResult.MessageList,
				AgentSpan:          beforeResult.AgentSpan,
				Tripwire:           beforeResult.Tripwire,
				Tools:              beforeResult.ConvertedTools,
				RunID:              runID,
				Temperature:        temperature,
				ToolChoice:         toolChoice,
				ThreadID:           threadID,
				ResourceID:         resourceID,
				RequestContext:     reqCtx,
				Output:             output,
				ExperimentalOutput: experimentalOutput,
				MaxSteps:           maxSteps,
			}

			// Wire onStepFinish with per-step memory persistence
			result.OnStepFinish = func(props any) error {
				if savePerStep {
					if !threadExists && !threadCreatedByStep && memory != nil && thread != nil {
						var mc *MemoryConfig
						if memoryConfig != nil {
							if cfg, ok := memoryConfig.(*MemoryConfig); ok {
								mc = cfg
							}
						}
						saveThread := true
						_, createErr := memory.CreateThread(context.Background(), mempkg.CreateThreadOpts{
							ThreadID:     thread.ID,
							ResourceID:   resourceID,
							Title:        thread.Title,
							Metadata:     thread.Metadata,
							MemoryConfig: mc,
							SaveThread:   &saveThread,
						})
						if createErr != nil && logger != nil {
							logger.Error(fmt.Sprintf("[Agent:%s] - Error creating thread: %v", agentName, createErr))
						}
						threadCreatedByStep = true
					}

					// TODO: capabilities.SaveStepMessages({result: props, messageList, runId})
					// Requires MessageList to be fully ported.
				}

				if onStepFinish != nil {
					if fn, ok := onStepFinish.(func(any) error); ok {
						return fn(props)
					}
				}
				return nil
			}

			return result, nil
		},

		After: func(args LegacyAfterArgs) (*PrimitiveAfterResult, error) {
			afterResult, err := primitiveCallbacks.After(PrimitiveAfterArgs{
				Result:           args.Result.(map[string]any),
				OutputText:       args.OutputText,
				Thread:           thread,
				ThreadID:         func() string { if thread != nil { return thread.ID }; return "" }(),
				MemoryConfig:     memoryConfig,
				RunID:            runID,
				MessageList:      messageList,
				StructuredOutput: args.StructuredOutput,
				ThreadExists:     threadExists,
				AgentSpan:        args.AgentSpan,
				OverrideScorers:  args.OverrideScorers,
			})
			if err != nil {
				return nil, err
			}
			_ = onFinish
			_ = scorers
			return afterResult, nil
		},
	}, nil
}

// ---------------------------------------------------------------------------
// GenerateLegacy
// ---------------------------------------------------------------------------

// GenerateLegacy is the legacy implementation of generate using AI SDK v4 models.
// Use this method if you need to continue using AI SDK v4 models.
//
// Ported from TS: async generateLegacy<OUTPUT, EXPERIMENTAL_OUTPUT>(messages, generateOptions)
func (h *AgentLegacyHandler) GenerateLegacy(
	ctx context.Context,
	messages MessageListInput,
	generateOptions AgentGenerateOptions,
) (any, error) {
	// Check for unsupported structuredOutput in legacy mode.
	// In TS: if ('structuredOutput' in generateOptions && generateOptions.structuredOutput) { throw ... }
	// AgentGenerateOptions doesn't have StructuredOutput in Go, so this is handled by the
	// AgentExecutionOptions level. No-op here.

	// Get default options and merge
	defaultOpts, err := h.agent.GetDefaultGenerateOptionsLegacy(generateOptions.RequestContext)
	if err != nil {
		return nil, err
	}
	mergedOpts := mergeGenerateOptions(defaultOpts, generateOptions)

	// Convert to map for prepareLLMOptions
	optionsMap := generateOptionsToMap(mergedOpts)

	// Prepare LLM options
	prepared, err := h.prepareLLMOptions(messages, optionsMap, "generate")
	if err != nil {
		return nil, err
	}

	// Verify model specification version
	llm := prepared.LLM
	model := llm.GetModel()
	specVersion := model.SpecificationVersion()
	if specVersion != "v1" {
		logger := h.agent.Logger()
		if logger != nil {
			logger.Error(fmt.Sprintf(
				`Models with specificationVersion "%s" are not supported for generateLegacy. Please use generate() instead.`,
				specVersion,
			))
		}
		details := map[string]any{
			"specificationVersion": specVersion,
		}
		if m, ok := model.(interface{ ModelID() string }); ok {
			details["modelId"] = m.ModelID()
		}
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_GENERATE_V2_MODEL_NOT_SUPPORTED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Details:  details,
			Text: fmt.Sprintf(
				`Models with specificationVersion "%s" are not supported for generateLegacy(). Please use generate() instead.`,
				specVersion,
			),
		})
	}

	// Run before() phase
	beforeResult, err := prepared.Before()
	if err != nil {
		return nil, err
	}

	// Check for tripwire and return early if triggered
	if beforeResult.Tripwire != nil {
		tripwireResult := map[string]any{
			"text":     "",
			"object":   nil,
			"usage":    map[string]any{"totalTokens": 0, "promptTokens": 0, "completionTokens": 0},
			"finishReason": "other",
			"response": map[string]any{
				"id":        generateUUID(),
				"modelId":   "tripwire",
				"messages":  []any{},
			},
			"responseMessages":              []any{},
			"toolCalls":                     []any{},
			"toolResults":                   []any{},
			"tripwire":                      beforeResult.Tripwire,
		}
		return tripwireResult, nil
	}

	// Call LLM
	// TODO: implement LLM call once MastraLLMV1 is ported:
	//   if output == nil || experimentalOutput != nil {
	//       result = llm.__text({...llmOptions, experimental_output, ...observabilityContext})
	//       messageList.Add({role: "assistant", content: [{type: "text", text: result.text}]}, "response")
	//       outputProcessorResult = capabilities.__runOutputProcessors({requestContext, messageList, ...})
	//       if outputProcessorResult.Tripwire != nil { return tripwire result }
	//       newText = outputProcessorResult.MessageList.Get.Response.DB()...
	//       result.text = newText
	//       afterResult = after({result, outputText: newText, agentSpan, overrideScorers})
	//       if returnScorerData { result.scoringData = afterResult.scoringData }
	//       return result
	//   } else {
	//       result = llm.__textObject({...llmOptions, structuredOutput: output})
	//       outputText = JSON.stringify(result.object)
	//       messageList.Add({role: "assistant", content: [{type: "text", text: outputText}]}, "response")
	//       outputProcessorResult = capabilities.__runOutputProcessors({requestContext, messageList})
	//       afterResult = after({result, outputText: newText, structuredOutput: true, agentSpan, overrideScorers})
	//       return result
	//   }

	_ = beforeResult

	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "AGENT_GENERATE_LEGACY_LLM_CALL_NOT_IMPLEMENTED",
		Domain:   mastraerror.ErrorDomainAgent,
		Category: mastraerror.ErrorCategorySystem,
		Text:     "GenerateLegacy LLM call is not fully implemented. Port MastraLLMV1 (__text, __textObject) first.",
	})
}

// ---------------------------------------------------------------------------
// StreamLegacy
// ---------------------------------------------------------------------------

// StreamLegacy is the legacy implementation of stream using AI SDK v4 models.
// Use this method if you need to continue using AI SDK v4 models.
//
// Ported from TS: async streamLegacy<OUTPUT, EXPERIMENTAL_OUTPUT>(messages, streamOptions)
func (h *AgentLegacyHandler) StreamLegacy(
	ctx context.Context,
	messages MessageListInput,
	streamOptions AgentStreamOptions,
) (any, error) {
	// Get default options and merge
	defaultOpts, err := h.agent.GetDefaultStreamOptionsLegacy(streamOptions.RequestContext)
	if err != nil {
		return nil, err
	}
	mergedOpts := mergeStreamOptions(defaultOpts, streamOptions)

	// Convert to map for prepareLLMOptions
	optionsMap := streamOptionsToMap(mergedOpts)

	// Prepare LLM options
	prepared, err := h.prepareLLMOptions(messages, optionsMap, "stream")
	if err != nil {
		return nil, err
	}

	// Verify model specification version
	llm := prepared.LLM
	model := llm.GetModel()
	specVersion := model.SpecificationVersion()
	if specVersion != "v1" {
		logger := h.agent.Logger()
		if logger != nil {
			logger.Error(fmt.Sprintf(
				`Models with specificationVersion "%s" are not supported for streamLegacy. Please use stream() instead.`,
				specVersion,
			))
		}
		streamDetails := map[string]any{
			"specificationVersion": specVersion,
		}
		if m, ok := model.(interface{ ModelID() string }); ok {
			streamDetails["modelId"] = m.ModelID()
		}
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "AGENT_STREAM_V2_MODEL_NOT_SUPPORTED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Details:  streamDetails,
			Text: fmt.Sprintf(
				`Models with specificationVersion "%s" are not supported for streamLegacy(). Please use stream() instead.`,
				specVersion,
			),
		})
	}

	// Run before() phase
	beforeResult, err := prepared.Before()
	if err != nil {
		return nil, err
	}

	// Check for tripwire and return early if triggered
	if beforeResult.Tripwire != nil {
		// Return empty stream-like result
		emptyResult := map[string]any{
			"text":                "",
			"usage":              map[string]any{"totalTokens": 0, "promptTokens": 0, "completionTokens": 0},
			"finishReason":       "other",
			"tripwire":           beforeResult.Tripwire,
			"response": map[string]any{
				"id":       generateUUID(),
				"modelId":  "tripwire",
				"messages": []any{},
			},
			"toolCalls":   []any{},
			"toolResults": []any{},
		}
		return emptyResult, nil
	}

	// Call LLM stream
	// TODO: implement LLM stream call once MastraLLMV1 is ported:
	//   if output == nil || experimentalOutput != nil {
	//       streamResult = llm.__stream({...llmOptions, experimental_output, requestContext,
	//           outputProcessors: capabilities.ListResolvedOutputProcessors(requestContext),
	//           onFinish: func(result) {
	//               messageList.Add(result.response.messages, "response")
	//               capabilities.__runOutputProcessors({requestContext, messageList})
	//               after({result, outputText: result.text, agentSpan, overrideScorers})
	//               onFinish({...result, runId})
	//           },
	//           runId,
	//       })
	//       return streamResult
	//   } else {
	//       streamObjectResult = llm.__streamObject({...llmOptions, requestContext, structuredOutput: output,
	//           onFinish: func(result) {
	//               if result.object { messageList.Add(responseMessages, "response") }
	//               capabilities.__runOutputProcessors({requestContext, messageList})
	//               after({result, outputText: JSON.stringify(result.object), structuredOutput: true, agentSpan, overrideScorers})
	//               onFinish({...result, runId})
	//           },
	//           runId,
	//       })
	//       return streamObjectResult
	//   }

	_ = beforeResult

	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "AGENT_STREAM_LEGACY_LLM_CALL_NOT_IMPLEMENTED",
		Domain:   mastraerror.ErrorDomainAgent,
		Category: mastraerror.ErrorCategorySystem,
		Text:     "StreamLegacy LLM call is not fully implemented. Port MastraLLMV1 (__stream, __streamObject) first.",
	})
}

// ---------------------------------------------------------------------------
// Option merging helpers
// ---------------------------------------------------------------------------

// mergeGenerateOptions merges default generate options with provided overrides.
// Fields in override take precedence over defaults.
func mergeGenerateOptions(defaults, override AgentGenerateOptions) AgentGenerateOptions {
	merged := defaults

	if override.Instructions != nil {
		merged.Instructions = override.Instructions
	}
	if override.Toolsets != nil {
		merged.Toolsets = override.Toolsets
	}
	if override.ClientTools != nil {
		merged.ClientTools = override.ClientTools
	}
	if override.Context != nil {
		merged.Context = override.Context
	}
	if override.Memory != nil {
		merged.Memory = override.Memory
	}
	if override.RunID != "" {
		merged.RunID = override.RunID
	}
	if override.OnStepFinish != nil {
		merged.OnStepFinish = override.OnStepFinish
	}
	if override.MaxSteps != nil {
		merged.MaxSteps = override.MaxSteps
	}
	if override.Output != nil {
		merged.Output = override.Output
	}
	if override.ExperimentalOutput != nil {
		merged.ExperimentalOutput = override.ExperimentalOutput
	}
	if override.ToolChoice != nil {
		merged.ToolChoice = override.ToolChoice
	}
	if override.RequestContext != nil {
		merged.RequestContext = override.RequestContext
	}
	if override.Scorers != nil {
		merged.Scorers = override.Scorers
	}
	if override.ReturnScorerData {
		merged.ReturnScorerData = override.ReturnScorerData
	}
	if override.SavePerStep {
		merged.SavePerStep = override.SavePerStep
	}
	if override.InputProcessors != nil {
		merged.InputProcessors = override.InputProcessors
	}
	if override.OutputProcessors != nil {
		merged.OutputProcessors = override.OutputProcessors
	}
	if override.MaxProcessorRetries != nil {
		merged.MaxProcessorRetries = override.MaxProcessorRetries
	}
	if override.TracingOptions != nil {
		merged.TracingOptions = override.TracingOptions
	}
	if override.ProviderOptions != nil {
		merged.ProviderOptions = override.ProviderOptions
	}
	if override.ResourceID != nil {
		merged.ResourceID = override.ResourceID
	}
	if override.ThreadID != nil {
		merged.ThreadID = override.ThreadID
	}

	return merged
}

// mergeStreamOptions merges default stream options with provided overrides.
// Fields in override take precedence over defaults.
func mergeStreamOptions(defaults, override AgentStreamOptions) AgentStreamOptions {
	merged := defaults

	if override.Instructions != nil {
		merged.Instructions = override.Instructions
	}
	if override.Toolsets != nil {
		merged.Toolsets = override.Toolsets
	}
	if override.ClientTools != nil {
		merged.ClientTools = override.ClientTools
	}
	if override.Context != nil {
		merged.Context = override.Context
	}
	if override.Memory != nil {
		merged.Memory = override.Memory
	}
	if override.RunID != "" {
		merged.RunID = override.RunID
	}
	if override.OnFinish != nil {
		merged.OnFinish = override.OnFinish
	}
	if override.OnStepFinish != nil {
		merged.OnStepFinish = override.OnStepFinish
	}
	if override.MaxSteps != nil {
		merged.MaxSteps = override.MaxSteps
	}
	if override.Output != nil {
		merged.Output = override.Output
	}
	if override.Temperature != nil {
		merged.Temperature = override.Temperature
	}
	if override.ToolChoice != nil {
		merged.ToolChoice = override.ToolChoice
	}
	if override.ExperimentalOutput != nil {
		merged.ExperimentalOutput = override.ExperimentalOutput
	}
	if override.RequestContext != nil {
		merged.RequestContext = override.RequestContext
	}
	if override.SavePerStep {
		merged.SavePerStep = override.SavePerStep
	}
	if override.InputProcessors != nil {
		merged.InputProcessors = override.InputProcessors
	}
	if override.TracingOptions != nil {
		merged.TracingOptions = override.TracingOptions
	}
	if override.Scorers != nil {
		merged.Scorers = override.Scorers
	}
	if override.ProviderOptions != nil {
		merged.ProviderOptions = override.ProviderOptions
	}
	if override.ResourceID != nil {
		merged.ResourceID = override.ResourceID
	}
	if override.ThreadID != nil {
		merged.ThreadID = override.ThreadID
	}

	return merged
}

// ---------------------------------------------------------------------------
// Conversion helpers
// ---------------------------------------------------------------------------

// generateOptionsToMap converts AgentGenerateOptions to a map for prepareLLMOptions.
func generateOptionsToMap(opts AgentGenerateOptions) map[string]any {
	m := make(map[string]any)
	if opts.Instructions != nil {
		m["instructions"] = opts.Instructions
	}
	if opts.Toolsets != nil {
		m["toolsets"] = opts.Toolsets
	}
	if opts.ClientTools != nil {
		m["clientTools"] = opts.ClientTools
	}
	if opts.Context != nil {
		m["context"] = opts.Context
	}
	if opts.Memory != nil {
		m["memory"] = opts.Memory
	}
	if opts.RunID != "" {
		m["runId"] = opts.RunID
	}
	if opts.OnStepFinish != nil {
		m["onStepFinish"] = opts.OnStepFinish
	}
	if opts.MaxSteps != nil {
		m["maxSteps"] = opts.MaxSteps
	}
	if opts.Output != nil {
		m["output"] = opts.Output
	}
	if opts.ExperimentalOutput != nil {
		m["experimental_output"] = opts.ExperimentalOutput
	}
	if opts.ToolChoice != nil {
		m["toolChoice"] = opts.ToolChoice
	}
	if opts.RequestContext != nil {
		m["requestContext"] = opts.RequestContext
	}
	if opts.Scorers != nil {
		m["scorers"] = opts.Scorers
	}
	m["returnScorerData"] = opts.ReturnScorerData
	m["savePerStep"] = opts.SavePerStep
	if opts.InputProcessors != nil {
		m["inputProcessors"] = opts.InputProcessors
	}
	if opts.OutputProcessors != nil {
		m["outputProcessors"] = opts.OutputProcessors
	}
	if opts.MaxProcessorRetries != nil {
		m["maxProcessorRetries"] = opts.MaxProcessorRetries
	}
	if opts.TracingOptions != nil {
		m["tracingOptions"] = opts.TracingOptions
	}
	if opts.ProviderOptions != nil {
		m["providerOptions"] = opts.ProviderOptions
	}
	if opts.ResourceID != nil {
		m["resourceId"] = opts.ResourceID
	}
	if opts.ThreadID != nil {
		m["threadId"] = opts.ThreadID
	}
	return m
}

// streamOptionsToMap converts AgentStreamOptions to a map for prepareLLMOptions.
func streamOptionsToMap(opts AgentStreamOptions) map[string]any {
	m := make(map[string]any)
	if opts.Instructions != nil {
		m["instructions"] = opts.Instructions
	}
	if opts.Toolsets != nil {
		m["toolsets"] = opts.Toolsets
	}
	if opts.ClientTools != nil {
		m["clientTools"] = opts.ClientTools
	}
	if opts.Context != nil {
		m["context"] = opts.Context
	}
	if opts.Memory != nil {
		m["memory"] = opts.Memory
	}
	if opts.RunID != "" {
		m["runId"] = opts.RunID
	}
	if opts.OnFinish != nil {
		m["onFinish"] = opts.OnFinish
	}
	if opts.OnStepFinish != nil {
		m["onStepFinish"] = opts.OnStepFinish
	}
	if opts.MaxSteps != nil {
		m["maxSteps"] = opts.MaxSteps
	}
	if opts.Output != nil {
		m["output"] = opts.Output
	}
	if opts.Temperature != nil {
		m["temperature"] = opts.Temperature
	}
	if opts.ToolChoice != nil {
		m["toolChoice"] = opts.ToolChoice
	}
	if opts.ExperimentalOutput != nil {
		m["experimental_output"] = opts.ExperimentalOutput
	}
	if opts.RequestContext != nil {
		m["requestContext"] = opts.RequestContext
	}
	m["savePerStep"] = opts.SavePerStep
	if opts.InputProcessors != nil {
		m["inputProcessors"] = opts.InputProcessors
	}
	if opts.TracingOptions != nil {
		m["tracingOptions"] = opts.TracingOptions
	}
	if opts.Scorers != nil {
		m["scorers"] = opts.Scorers
	}
	if opts.ProviderOptions != nil {
		m["providerOptions"] = opts.ProviderOptions
	}
	if opts.ResourceID != nil {
		m["resourceId"] = opts.ResourceID
	}
	if opts.ThreadID != nil {
		m["threadId"] = opts.ThreadID
	}
	return m
}

// extractRequestContext extracts RequestContext from an options map.
func extractRequestContext(options map[string]any) *requestcontext.RequestContext {
	if rc, ok := options["requestContext"].(*requestcontext.RequestContext); ok {
		return rc
	}
	return nil
}

// generateUUID generates a random UUID string.
func generateUUID() string {
	return uuid.New().String()
}
