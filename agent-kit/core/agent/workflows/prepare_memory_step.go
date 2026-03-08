// Ported from: packages/core/src/agent/workflows/prepare-stream/prepare-memory-step.ts
package workflows

import "fmt"

// ---------------------------------------------------------------------------
// Additional stub types
// ---------------------------------------------------------------------------

// SystemMessage is a stub for ../../../llm.SystemMessage.
// Both llm.SystemMessage and this stub are = any. No value in wiring import.
type SystemMessage = any

// MemoryConfig is a stub for ../../../memory/types.MemoryConfig.
// Real memory.MemoryConfig is a struct with ReadOnly, LastMessages, SemanticRecall,
// etc. Kept as = any because workflow code passes it opaquely (never accesses fields).
type MemoryConfig = any

// PrepareMemoryStepOptions holds options for creating a prepare-memory step.
type PrepareMemoryStepOptions struct {
	Capabilities   AgentCapabilities
	Options        InnerAgentExecutionOptions
	ThreadFromArgs *StorageThreadType
	ResourceID     string
	RunID          string
	RequestContext RequestContext
	MethodType     AgentMethodType
	Instructions   SystemMessage
	MemoryConfig   MemoryConfig
	Memory         MastraMemory
}

// addSystemMessage adds system message(s) to a MessageList.
// Handles string, CoreSystemMessage, SystemModelMessage, and arrays of these message formats.
// Used for both agent instructions and user-provided system messages.
// TODO: call actual messageList.AddSystem() once MessageList is ported.
func addSystemMessage(messageList MessageList, content SystemMessage, tag string) {
	if content == nil {
		return
	}
	// In TS, this checks Array.isArray and loops or calls messageList.addSystem().
	// TODO: implement once MessageList type is fully ported:
	// if arr, ok := content.([]any); ok {
	//     for _, msg := range arr {
	//         messageList.AddSystem(msg, tag)
	//     }
	// } else {
	//     messageList.AddSystem(content, tag)
	// }
	_ = messageList
	_ = tag
}

// CreatePrepareMemoryStep creates the memory preparation step for the agent workflow.
// This step initializes the MessageList, loads thread history from memory (if configured),
// adds system instructions and user messages, and runs input processors.
//
// Ported from TS: createPrepareMemoryStep()
func CreatePrepareMemoryStep(opts PrepareMemoryStepOptions) func() (*PrepareMemoryStepOutput, error) {
	return func() (*PrepareMemoryStepOutput, error) {
		thread := opts.ThreadFromArgs

		// Create a new MessageList for this execution.
		// TODO: use actual MessageList constructor once ported:
		//   messageList = NewMessageList(MessageListOptions{
		//       ThreadID: thread.ID,
		//       ResourceID: opts.ResourceID,
		//       GenerateMessageID: opts.Capabilities.GenerateMessageID,
		//       Logger: opts.Capabilities.Logger,
		//       AgentNetworkAppend: opts.Capabilities.AgentNetworkAppend,
		//   })
		var messageList MessageList = nil

		// Create processorStates map - persists across loop iterations within this agent turn.
		// Shared by all processor methods (input and output) for state sharing.
		processorStates := make(map[string]ProcessorState)

		// Add instructions as system message(s)
		addSystemMessage(messageList, opts.Instructions, "")

		// Add context messages
		// TODO: messageList.Add(opts.Options.Context, "context")

		// Add user-provided system message if present
		// TODO: addSystemMessage(messageList, opts.Options.System, "user-provided")

		// Check: no memory or no thread/resource
		if opts.Memory == nil || (thread == nil && opts.ResourceID == "") {
			// No memory configured - just add input messages and run processors.
			// TODO: messageList.Add(opts.Options.Messages, "input")

			// Run input processors
			var tripwire *TripwireData
			if opts.Capabilities.RunInputProcessors != nil {
				result, err := opts.Capabilities.RunInputProcessors(map[string]any{
					"requestContext":         opts.RequestContext,
					"messageList":            messageList,
					"inputProcessorOverrides": opts.Options.InputProcessors,
					"processorStates":        processorStates,
				})
				if err != nil {
					return nil, err
				}
				tripwire = result.Tripwire
			}

			return &PrepareMemoryStepOutput{
				ThreadExists:    false,
				Thread:          nil,
				MessageList:     messageList,
				ProcessorStates: processorStates,
				Tripwire:        tripwire,
			}, nil
		}

		// Memory is configured - validate required IDs
		if thread == nil || opts.ResourceID == "" {
			// In TS this throws MastraError with AGENT_MEMORY_MISSING_RESOURCE_ID
			var threadID string
			if thread != nil {
				threadID = thread.ID
			}
			errMsg := fmt.Sprintf(
				`A resourceId and a threadId must be provided when using Memory. Saw threadId "%s" and resourceId "%s"`,
				threadID, opts.ResourceID,
			)
			if logger, ok := opts.Capabilities.Logger.(interface {
				Error(msg string, fields ...any)
			}); ok {
				logger.Error(errMsg)
			}
			return nil, fmt.Errorf("AGENT_MEMORY_MISSING_RESOURCE_ID: %s", errMsg)
		}

		// Log memory persistence info
		if logger, ok := opts.Capabilities.Logger.(interface {
			Debug(msg string, fields ...any)
		}); ok {
			logger.Debug(
				fmt.Sprintf("[Agent:%s] - Memory persistence enabled: store=memory, resourceId=%s",
					opts.Capabilities.AgentName, opts.ResourceID),
				"runId", opts.RunID,
				"resourceId", opts.ResourceID,
				"threadId", thread.ID,
			)
		}

		// Thread management:
		// 1. Try to get existing thread by ID
		// 2. If exists, check if metadata needs updating and save if so
		// 3. If not exists, create new thread with saveThread: true
		//
		// TODO: implement once MastraMemory interface is ported:
		//   existingThread, err := memory.GetThreadById(thread.ID)
		//   if existingThread != nil {
		//       if thread.Metadata != nil && !deepEqual(existingThread.Metadata, thread.Metadata) {
		//           threadObject, _ = memory.SaveThread(existingThread with updated metadata, memoryConfig)
		//       } else {
		//           threadObject = existingThread
		//       }
		//   } else {
		//       threadObject, _ = memory.CreateThread(thread.ID, thread.Metadata, thread.Title, memoryConfig, resourceId, saveThread=true)
		//   }
		var threadObject *StorageThreadType
		var existingThread bool

		// Stub: assume thread is used as-is until memory package is ported
		threadObject = thread

		// Set memory context in RequestContext for processors to access
		// TODO: requestContext.Set("MastraMemory", map[string]any{
		//     "thread": threadObject,
		//     "resourceId": opts.ResourceID,
		//     "memoryConfig": opts.MemoryConfig,
		// })

		// Add user messages - memory processors will handle history/semantic recall/working memory
		// TODO: messageList.Add(opts.Options.Messages, "input")

		// Run input processors
		var tripwire *TripwireData
		if opts.Capabilities.RunInputProcessors != nil {
			result, err := opts.Capabilities.RunInputProcessors(map[string]any{
				"requestContext":          opts.RequestContext,
				"messageList":             messageList,
				"inputProcessorOverrides": opts.Options.InputProcessors,
				"processorStates":         processorStates,
			})
			if err != nil {
				return nil, err
			}
			tripwire = result.Tripwire
		}

		return &PrepareMemoryStepOutput{
			Thread:          threadObject,
			MessageList:     messageList,
			ProcessorStates: processorStates,
			Tripwire:        tripwire,
			ThreadExists:    existingThread,
		}, nil
	}
}
