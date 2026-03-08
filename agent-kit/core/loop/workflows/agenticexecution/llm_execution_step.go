// Ported from: packages/core/src/loop/workflows/agentic-execution/llm-execution-step.ts
package agenticexecution

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported (llm-execution-specific)
// ---------------------------------------------------------------------------

// MastraModelOutput is a stub for ../../../stream/base/output.MastraModelOutput.
// Stub: real type has sync.Mutex, many private fields (status, streamFinished, etc.),
// and different method signatures. This simplified version only exposes the immediate
// getters needed by processOutputStream. Shape mismatch.
type MastraModelOutput struct {
	Tripwire bool
	// Internal state for immediate getters.
	immediateUsage        map[string]any
	immediateText         string
	immediateToolCalls    []map[string]any
	immediateFinishReason string
	immediateWarnings     []any
	immediateObject       any
}

// GetBaseStream returns the base stream for processing.
func (m *MastraModelOutput) GetBaseStream() <-chan map[string]any {
	ch := make(chan map[string]any)
	close(ch)
	return ch
}

// GetImmediateUsage returns immediate usage data.
func (m *MastraModelOutput) GetImmediateUsage() map[string]any { return m.immediateUsage }

// GetImmediateText returns immediate text data.
func (m *MastraModelOutput) GetImmediateText() string { return m.immediateText }

// GetImmediateToolCalls returns immediate tool call data.
func (m *MastraModelOutput) GetImmediateToolCalls() []map[string]any { return m.immediateToolCalls }

// GetImmediateFinishReason returns the immediate finish reason.
func (m *MastraModelOutput) GetImmediateFinishReason() string { return m.immediateFinishReason }

// GetImmediateWarnings returns immediate warnings.
func (m *MastraModelOutput) GetImmediateWarnings() []any { return m.immediateWarnings }

// GetImmediateObject returns the immediate object output.
func (m *MastraModelOutput) GetImmediateObject() any { return m.immediateObject }

// AgenticRunState is a stub reference to ../run-state.AgenticRunState.
// Stub: real type in loop/workflows has sync.RWMutex + typed State struct;
// this uses map[string]any. Can't import parent package (agenticexecution → workflows
// would create cycle since workflows imports agenticexecution). Shape mismatch.
type AgenticRunState struct {
	state map[string]any
}

// NewAgenticRunState creates a new AgenticRunState.
func NewAgenticRunState(internal any, model MastraLanguageModel) *AgenticRunState {
	return &AgenticRunState{
		state: map[string]any{
			"isReasoning":         false,
			"isStreaming":          false,
			"hasToolCallStreaming": false,
			"hasErrored":          false,
			"reasoningDeltas":     []string{},
			"textDeltas":          []string{},
		},
	}
}

// SetState merges partial state.
func (rs *AgenticRunState) SetState(partial map[string]any) {
	for k, v := range partial {
		rs.state[k] = v
	}
}

// GetState returns the current state snapshot.
func (rs *AgenticRunState) GetState() map[string]any {
	return rs.state
}

// MastraLanguageModel is a stub for ../../../llm/model/shared_types.MastraLanguageModel.
// Stub: real interface has methods ModelID(), Provider(), SpecificationVersion() (no Get prefix).
// This stub uses GetModelID(), GetProvider(), GetSpecificationVersion(). Method name mismatch.
type MastraLanguageModel interface {
	GetModelID() string
	GetSpecificationVersion() string
	GetProvider() string
}

// ModelManagerModelConfig is a stub for ../../../stream/types.ModelManagerModelConfig.
// Stub: real type has flat Model field (MastraLanguageModel interface) + different extras.
// This stub has Model as local MastraLanguageModel + MaxRetries/ID/Headers. Shape mismatch.
type ModelManagerModelConfig struct {
	Model      MastraLanguageModel `json:"model"`
	MaxRetries int                 `json:"maxRetries"`
	ID         string              `json:"id"`
	Headers    map[string]string   `json:"headers,omitempty"`
}

// TripWire is a stub for ../../../agent/trip_wire.TripWire.
// Stub: real type has Reason (not Message) field, Options as value type (not pointer),
// and ProcessorID field. Importing agent could create cycle risk. Shape mismatch.
type TripWire struct {
	Message     string
	ProcessorID string
	Options     *TripWireOptions
}

func (t *TripWire) Error() string { return t.Message }

// TripWireOptions holds tripwire options.
type TripWireOptions struct {
	Retry    bool           `json:"retry,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// APICallError is a stub for @internal/ai-sdk-v5.APICallError.
// ai-kit has errors.APICallError but the V5 shape differs; local stub kept.
type APICallError struct {
	Message string
}

func (e *APICallError) Error() string { return e.Message }

// IsAPICallError checks if an error is an APICallError.
func IsAPICallError(err error) bool {
	_, ok := err.(*APICallError)
	return ok
}

// IsAbortError is a stub for @ai-sdk/provider-utils-v5.isAbortError.
// ai-kit only ported the @ai-sdk/provider layer; provider-utils remain local stubs.
func IsAbortError(err error) bool {
	return false
}

// IMastraLogger is a stub for ../../../logger.IMastraLogger.
// Stub: real interface has 8 methods (Debug, Info, Warn, Error, TrackException,
// GetTransports, ListLogs, ListLogsByRunID). This is a 4-method subset used locally.
// Could use real import since this subset is compatible for consumers, but ConsoleLogger
// below would also need to implement the full interface. Kept as subset for simplicity.
type IMastraLogger interface {
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

// ConsoleLogger is a stub for ../../../logger.ConsoleLogger.
// Stub: real logger.IMastraLogger has 8 methods; this implements only 4.
// Kept local to match the reduced IMastraLogger interface above.
type ConsoleLogger struct {
	Level string
}

func (l *ConsoleLogger) Error(msg string, args ...any) {}
func (l *ConsoleLogger) Info(msg string, args ...any)  {}
func (l *ConsoleLogger) Debug(msg string, args ...any) {}
func (l *ConsoleLogger) Warn(msg string, args ...any)  {}

// PrepareStepProcessor is a stub for ../../../processors/processors/prepare_step.PrepareStepProcessor.
// Stub: real type embeds processors.BaseProcessor + has typed prepareStep field.
// This stub has only PrepareStep any. Shape mismatch.
type PrepareStepProcessor struct {
	PrepareStep any
}

// MessageListFull is the full MessageList interface needed by llm-execution-step.
// Stub: real agent.MessageList is a struct with methods. Importing agent from this
// deep subpackage would create coupling risk. Kept as interface matching call sites.
type MessageListFull interface {
	GetAllSystemMessages() []any
	ReplaceAllSystemMessages(msgs []any)
	AddSystem(content string, tag string)
	Add(msg any, source string)
	RemoveByIds(ids []string)
	// Access sub-interfaces for different views.
	GetAll() MessageListView
	GetInput() MessageListView
	GetResponse() MessageListView
}

// MessageListView provides access to messages in different formats.
// Stub: real type lives in agent package. Importing agent from this deep subpackage
// would create coupling risk. Kept as interface matching call sites.
type MessageListView interface {
	DB() []map[string]any
	AIV5Model() []any
	AIV5ModelContent(stepNumber int) []any
	AIV5LLMPrompt(args map[string]any) ([]any, error)
}

// StreamTransport is a stub for ../../../stream/types.StreamTransport.
// Stub: real stream.StreamTransport is a struct with Type, CloseFunc, CloseOnFinish fields.
// This stub uses any for flexibility. Shape mismatch.
type StreamTransport = any

// StreamTransportRef is a stub for ../../../stream/types.StreamTransportRef.
// Stub: real type has Current *StreamTransport (pointer to struct). This stub has
// Current as any (StreamTransport = any). Shape mismatch.
type StreamTransportRef struct {
	Current StreamTransport
}

// TextStartPayload holds the payload for text-start chunks.
// Stub: real stream.TextStartPayload has ID string + ProviderMetadata as typed struct.
// This stub omits ID and uses map[string]any for ProviderMetadata. Shape mismatch.
type TextStartPayload struct {
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`
}

// DefaultStepResult wraps step result data.
// Stub: real type in stream/aisdk/v5 has typed fields (ContentPart, LanguageModelUsage,
// *StepTripwireData). This stub uses []any and map[string]any. Shape mismatch.
type DefaultStepResult struct {
	Warnings         []any          `json:"warnings,omitempty"`
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`
	FinishReason     string         `json:"finishReason,omitempty"`
	Content          []any          `json:"content,omitempty"`
	Response         map[string]any `json:"response,omitempty"`
	Request          any            `json:"request,omitempty"`
	Usage            map[string]any `json:"usage,omitempty"`
	Tripwire         map[string]any `json:"tripwire,omitempty"`
}

// ChunkFrom enumerates the source of chunks.
const ChunkFromAGENT = "agent"

// SafeEnqueue is a stub for ../../../stream/base.SafeEnqueue.
// Stub: real function takes chan<- stream.ChunkType and stream.ChunkType.
// This stub takes (any, map[string]any). Signature mismatch.
func SafeEnqueue(controller any, chunk map[string]any) {
	if ch, ok := controller.(chan map[string]any); ok {
		select {
		case ch <- chunk:
		default:
		}
	}
}

// GetErrorFromUnknown is a stub for ../../../error.GetErrorFromUnknown.
// Stub: real function returns *SerializableError with variadic *GetErrorOptions.
// This stub returns error with map[string]any opts. Signature mismatch.
func GetErrorFromUnknown(err any, opts map[string]any) error {
	if e, ok := err.(error); ok {
		return e
	}
	fallback, _ := opts["fallbackMessage"].(string)
	if fallback == "" {
		fallback = "unknown error"
	}
	return fmt.Errorf("%s", fallback)
}

// CreateObservabilityContext is a stub for ../../../observability.CreateObservabilityContext.
// Stub: real function takes *obstypes.TracingContext and returns obstypes.ObservabilityContext.
// This stub takes any and returns map[string]any. Signature mismatch.
func CreateObservabilityContext(tracingContext any) map[string]any {
	return nil
}

// ExecuteWithContextSync is a stub for ../../../observability.ExecuteWithContextSync.
// Stub: real function takes ExecuteWithContextSyncParams struct with Span + Fn fields.
// This stub takes map[string]any. Signature mismatch.
func ExecuteWithContextSync(opts map[string]any) any {
	if fn, ok := opts["fn"].(func() any); ok {
		return fn()
	}
	return nil
}

// ---------------------------------------------------------------------------
// ProcessOutputStream
// ---------------------------------------------------------------------------

// ProcessOutputStreamOptions holds the parameters for processOutputStream.
type ProcessOutputStreamOptions struct {
	Tools             ToolSet
	MessageID         string
	IncludeRawChunks  bool
	MessageList       MessageListFull
	OutputStreamModel *MastraModelOutput
	RunState          *AgenticRunState
	Options           *LoopConfigLocal
	Controller        any // channel-based controller
	ResponseFromModel ResponseFromModel
	Logger            IMastraLogger
	TransportRef      *StreamTransportRef
	TransportResolver func() StreamTransport
}

// LoopConfigLocal is a local stub for LoopConfig to avoid circular imports.
// Stub: can't import parent loop package (would create cycle: agenticexecution → workflows → loop
// while loop → workflows → agenticexecution). Callback signatures also differ
// (map[string]any vs typed params). Shape mismatch + cycle risk.
type LoopConfigLocal struct {
	OnChunk      func(chunk map[string]any) error
	OnError      func(args map[string]any) error
	OnFinish     func(result any) error
	OnStepFinish func(result any) error
	OnAbort      func(event map[string]any) error
	AbortSignal  *AbortSignal
	PrepareStep  any
}

// AbortSignal is a stub for abort signal handling.
// Stub: Go idiom is context.Context cancellation. This struct-based approach is kept
// for 1:1 TS port fidelity. Should be replaced with context.Context when wiring.
type AbortSignal struct {
	Aborted bool
}

// ResponseFromModel holds the response metadata from the model.
type ResponseFromModel struct {
	Warnings    any `json:"warnings"`
	Request     any `json:"request"`
	RawResponse any `json:"rawResponse"`
}

// ProcessOutputStream processes chunks from the output stream, dispatching
// them to the controller and updating the run state. It handles:
//   - response-metadata: Update response metadata in run state
//   - text-start/delta/end: Track text deltas and flush to message list
//   - reasoning-start/delta/end: Track reasoning and flush to messages
//   - tool-call-input-streaming-start/delta: Handle tool call streaming
//   - file: Add file parts to messages
//   - source: Add source parts to messages
//   - finish: Update step result in run state
//   - error: Handle errors and update state
//   - object/object-result: Pass through directly
func ProcessOutputStream(opts ProcessOutputStreamOptions) error {
	transportSet := false

	for chunk := range opts.OutputStreamModel.GetBaseStream() {
		// Stop processing if abort signal fired.
		if opts.Options != nil && opts.Options.AbortSignal != nil && opts.Options.AbortSignal.Aborted {
			break
		}

		if chunk == nil {
			continue
		}

		// Resolve transport on first chunk if needed.
		if !transportSet && opts.TransportRef != nil && opts.TransportResolver != nil {
			transport := opts.TransportResolver()
			if transport != nil {
				opts.TransportRef.Current = transport
				transportSet = true
			}
		}

		chunkType, _ := chunk["type"].(string)

		// Pass through object chunks directly.
		if chunkType == "object" || chunkType == "object-result" {
			SafeEnqueue(opts.Controller, chunk)
			continue
		}

		// Flush text deltas when a non-text, non-response-metadata chunk arrives.
		if chunkType != "text-delta" &&
			chunkType != "tool-call" &&
			chunkType != "response-metadata" {
			isStreaming, _ := opts.RunState.state["isStreaming"].(bool)
			if isStreaming {
				textDeltas, _ := opts.RunState.state["textDeltas"].([]string)
				if len(textDeltas) > 0 {
					payload, _ := chunk["payload"].(map[string]any)
					var providerMetadata map[string]any
					if payload != nil {
						pm, _ := payload["providerMetadata"].(map[string]any)
						providerMetadata = pm
					}
					if providerMetadata == nil {
						providerMetadata, _ = opts.RunState.state["providerOptions"].(map[string]any)
					}

					msg := map[string]any{
						"id":   opts.MessageID,
						"role": "assistant",
						"content": map[string]any{
							"format": 2,
							"parts": []map[string]any{
								{
									"type": "text",
									"text": strings.Join(textDeltas, ""),
								},
							},
						},
						"createdAt": time.Now(),
					}
					if providerMetadata != nil {
						parts := msg["content"].(map[string]any)["parts"].([]map[string]any)
						parts[0]["providerMetadata"] = providerMetadata
					}
					opts.MessageList.Add(msg, "response")
				}

				opts.RunState.SetState(map[string]any{
					"isStreaming": false,
					"textDeltas": []string{},
				})
			}
		}

		// Reset reasoning state for unexpected chunk types.
		isReasoning, _ := opts.RunState.state["isReasoning"].(bool)
		if chunkType != "reasoning-start" &&
			chunkType != "reasoning-delta" &&
			chunkType != "reasoning-end" &&
			chunkType != "redacted-reasoning" &&
			chunkType != "reasoning-signature" &&
			chunkType != "response-metadata" &&
			chunkType != "text-start" &&
			isReasoning {
			opts.RunState.SetState(map[string]any{
				"isReasoning":     false,
				"reasoningDeltas": []string{},
			})
		}

		switch chunkType {
		case "response-metadata":
			payload, _ := chunk["payload"].(map[string]any)
			if payload != nil {
				opts.RunState.SetState(map[string]any{
					"responseMetadata": map[string]any{
						"id":        payload["id"],
						"timestamp": payload["timestamp"],
						"modelId":   payload["modelId"],
						"headers":   payload["headers"],
					},
				})
			}

		case "text-start":
			payload, _ := chunk["payload"].(map[string]any)
			if payload != nil {
				if pm, ok := payload["providerMetadata"].(map[string]any); ok && pm != nil {
					opts.RunState.SetState(map[string]any{
						"providerOptions": pm,
					})
				}
			}
			SafeEnqueue(opts.Controller, chunk)

		case "text-delta":
			textDeltas, _ := opts.RunState.state["textDeltas"].([]string)
			payload, _ := chunk["payload"].(map[string]any)
			if payload != nil {
				if text, ok := payload["text"].(string); ok {
					textDeltas = append(textDeltas, text)
				}
			}
			opts.RunState.SetState(map[string]any{
				"textDeltas": textDeltas,
				"isStreaming": true,
			})
			SafeEnqueue(opts.Controller, chunk)

		case "text-end":
			opts.RunState.SetState(map[string]any{
				"providerOptions": nil,
			})
			SafeEnqueue(opts.Controller, chunk)

		case "tool-call-input-streaming-start":
			payload, _ := chunk["payload"].(map[string]any)
			if payload != nil {
				toolName, _ := payload["toolName"].(string)
				tool := findTool(opts.Tools, toolName)
				if tool != nil {
					if toolMap, ok := tool.(map[string]any); ok {
						if onInputStart, ok := toolMap["onInputStart"].(func(args map[string]any) error); ok {
							if err := onInputStart(map[string]any{
								"toolCallId": payload["toolCallId"],
							}); err != nil && opts.Logger != nil {
								opts.Logger.Error("Error calling onInputStart", err)
							}
						}
					}
				}
			}
			SafeEnqueue(opts.Controller, chunk)

		case "tool-call-delta":
			payload, _ := chunk["payload"].(map[string]any)
			if payload != nil {
				toolName, _ := payload["toolName"].(string)
				tool := findTool(opts.Tools, toolName)
				if tool != nil {
					if toolMap, ok := tool.(map[string]any); ok {
						if onInputDelta, ok := toolMap["onInputDelta"].(func(args map[string]any) error); ok {
							if err := onInputDelta(map[string]any{
								"inputTextDelta": payload["argsTextDelta"],
								"toolCallId":     payload["toolCallId"],
							}); err != nil && opts.Logger != nil {
								opts.Logger.Error("Error calling onInputDelta", err)
							}
						}
					}
				}
			}
			SafeEnqueue(opts.Controller, chunk)

		case "reasoning-start":
			payload, _ := chunk["payload"].(map[string]any)
			var pm map[string]any
			if payload != nil {
				pm, _ = payload["providerMetadata"].(map[string]any)
			}
			existingPO, _ := opts.RunState.state["providerOptions"].(map[string]any)
			if pm == nil {
				pm = existingPO
			}
			opts.RunState.SetState(map[string]any{
				"isReasoning":     true,
				"reasoningDeltas": []string{},
				"providerOptions": pm,
			})

			// Check for redacted reasoning data.
			hasRedacted := false
			if pm != nil {
				for _, v := range pm {
					if vm, ok := v.(map[string]any); ok {
						if _, ok := vm["redactedData"]; ok {
							hasRedacted = true
							break
						}
					}
				}
			}
			if hasRedacted {
				msg := map[string]any{
					"id":   opts.MessageID,
					"role": "assistant",
					"content": map[string]any{
						"format": 2,
						"parts": []map[string]any{
							{
								"type":      "reasoning",
								"reasoning": "",
								"details":   []map[string]any{{"type": "redacted", "data": ""}},
							},
						},
					},
					"createdAt": time.Now(),
				}
				if pm != nil {
					parts := msg["content"].(map[string]any)["parts"].([]map[string]any)
					parts[0]["providerMetadata"] = pm
				}
				opts.MessageList.Add(msg, "response")
			}
			SafeEnqueue(opts.Controller, chunk)

		case "reasoning-delta":
			reasoningDeltas, _ := opts.RunState.state["reasoningDeltas"].([]string)
			payload, _ := chunk["payload"].(map[string]any)
			if payload != nil {
				if text, ok := payload["text"].(string); ok {
					reasoningDeltas = append(reasoningDeltas, text)
				}
			}
			var pm map[string]any
			if payload != nil {
				pm, _ = payload["providerMetadata"].(map[string]any)
			}
			existingPO, _ := opts.RunState.state["providerOptions"].(map[string]any)
			if pm == nil {
				pm = existingPO
			}
			opts.RunState.SetState(map[string]any{
				"isReasoning":     true,
				"reasoningDeltas": reasoningDeltas,
				"providerOptions": pm,
			})
			SafeEnqueue(opts.Controller, chunk)

		case "reasoning-end":
			reasoningDeltas, _ := opts.RunState.state["reasoningDeltas"].([]string)
			existingPO, _ := opts.RunState.state["providerOptions"].(map[string]any)
			payload, _ := chunk["payload"].(map[string]any)
			var pm map[string]any
			if payload != nil {
				pm, _ = payload["providerMetadata"].(map[string]any)
			}
			if pm == nil {
				pm = existingPO
			}

			msg := map[string]any{
				"id":   opts.MessageID,
				"role": "assistant",
				"content": map[string]any{
					"format": 2,
					"parts": []map[string]any{
						{
							"type":      "reasoning",
							"reasoning": "",
							"details":   []map[string]any{{"type": "text", "text": strings.Join(reasoningDeltas, "")}},
						},
					},
				},
				"createdAt": time.Now(),
			}
			if pm != nil {
				parts := msg["content"].(map[string]any)["parts"].([]map[string]any)
				parts[0]["providerMetadata"] = pm
			}
			opts.MessageList.Add(msg, "response")

			opts.RunState.SetState(map[string]any{
				"isReasoning":     false,
				"reasoningDeltas": []string{},
				"providerOptions": nil,
			})
			SafeEnqueue(opts.Controller, chunk)

		case "file":
			payload, _ := chunk["payload"].(map[string]any)
			if payload != nil {
				msg := map[string]any{
					"id":   opts.MessageID,
					"role": "assistant",
					"content": map[string]any{
						"format": 2,
						"parts": []map[string]any{
							{
								"type":     "file",
								"data":     payload["data"],
								"mimeType": payload["mimeType"],
							},
						},
					},
					"createdAt": time.Now(),
				}
				opts.MessageList.Add(msg, "response")
			}
			SafeEnqueue(opts.Controller, chunk)

		case "source":
			payload, _ := chunk["payload"].(map[string]any)
			if payload != nil {
				url, _ := payload["url"].(string)
				msg := map[string]any{
					"id":   opts.MessageID,
					"role": "assistant",
					"content": map[string]any{
						"format": 2,
						"parts": []map[string]any{
							{
								"type": "source",
								"source": map[string]any{
									"sourceType":       "url",
									"id":               payload["id"],
									"url":              url,
									"title":            payload["title"],
									"providerMetadata": payload["providerMetadata"],
								},
							},
						},
					},
					"createdAt": time.Now(),
				}
				opts.MessageList.Add(msg, "response")
			}
			SafeEnqueue(opts.Controller, chunk)

		case "finish":
			payload, _ := chunk["payload"].(map[string]any)
			if payload != nil {
				metadata, _ := payload["metadata"].(map[string]any)
				stepResult, _ := payload["stepResult"].(map[string]any)
				reason, _ := payload["reason"].(string)

				var rawHeaders map[string]string
				if rawResp, ok := opts.ResponseFromModel.RawResponse.(map[string]any); ok {
					if h, ok := rawResp["headers"].(map[string]string); ok {
						rawHeaders = h
					}
				}

				isContinued := true
				if stepResult != nil {
					if sr, ok := stepResult["reason"].(string); ok {
						if sr == "stop" || sr == "error" {
							isContinued = false
						}
					}
				}

				opts.RunState.SetState(map[string]any{
					"providerOptions": func() any {
						if metadata != nil {
							return metadata["providerMetadata"]
						}
						return nil
					}(),
					"stepResult": map[string]any{
						"reason":     reason,
						"logprobs":   payload["logprobs"],
						"warnings":   opts.ResponseFromModel.Warnings,
						"totalUsage": payload["totalUsage"],
						"headers":    rawHeaders,
						"messageId":  opts.MessageID,
						"isContinued": isContinued,
						"request":    opts.ResponseFromModel.Request,
					},
				})
			}

		case "error":
			payload, _ := chunk["payload"].(map[string]any)
			errVal, _ := payload["error"]
			if IsAbortError(GetErrorFromUnknown(errVal, nil)) {
				if opts.Options != nil && opts.Options.AbortSignal != nil && opts.Options.AbortSignal.Aborted {
					break
				}
			}

			opts.RunState.SetState(map[string]any{
				"hasErrored": true,
			})
			opts.RunState.SetState(map[string]any{
				"stepResult": map[string]any{
					"isContinued": false,
					"reason":      "error",
				},
			})

			processedErr := GetErrorFromUnknown(errVal, map[string]any{
				"fallbackMessage": "Unknown error in agent stream",
			})
			errorChunk := map[string]any{
				"type": "error",
			}
			for k, v := range chunk {
				if k != "type" {
					errorChunk[k] = v
				}
			}
			errorChunk["payload"] = map[string]any{"error": processedErr}
			SafeEnqueue(opts.Controller, errorChunk)

			if opts.Options != nil && opts.Options.OnError != nil {
				_ = opts.Options.OnError(map[string]any{"error": processedErr})
			}

		default:
			SafeEnqueue(opts.Controller, chunk)
		}

		// Call onChunk for relevant chunk types.
		chunkTypesForCallback := map[string]bool{
			"text-delta":                       true,
			"reasoning-delta":                  true,
			"source":                           true,
			"tool-call":                        true,
			"tool-call-input-streaming-start":  true,
			"tool-call-delta":                  true,
			"raw":                              true,
		}
		if chunkTypesForCallback[chunkType] {
			if chunkType == "raw" && !opts.IncludeRawChunks {
				continue
			}
			if opts.Options != nil && opts.Options.OnChunk != nil {
				_ = opts.Options.OnChunk(chunk)
			}
		}

		hasErrored, _ := opts.RunState.state["hasErrored"].(bool)
		if hasErrored {
			break
		}
	}

	return nil
}

// findTool looks up a tool by name, falling back to searching by id field.
func findTool(tools ToolSet, name string) any {
	if tools == nil || name == "" {
		return nil
	}
	if t, ok := tools[name]; ok {
		return t
	}
	for _, t := range tools {
		if tm, ok := t.(map[string]any); ok {
			if id, ok := tm["id"].(string); ok && id == name {
				return t
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// ExecuteStreamWithFallbackModels
// ---------------------------------------------------------------------------

// ExecuteStreamWithFallbackModels tries each model in sequence with retries
// and exponential backoff. Returns the first successful result or an error
// if all models are exhausted.
//
// Key behaviors:
//   - TripWire errors are re-thrown immediately (intentional aborts)
//   - Exponential backoff: 1s, 2s, 4s, 8s, max 10s
//   - If all models and retries are exhausted, returns an error with the
//     last error as cause
func ExecuteStreamWithFallbackModels[T any](
	models []ModelManagerModelConfig,
	logger IMastraLogger,
	callback func(model ModelManagerModelConfig, isLastModel bool) (T, error),
) (T, error) {
	var finalResult T
	var lastError error
	done := false

	for i, modelConfig := range models {
		if done {
			break
		}

		maxRetries := modelConfig.MaxRetries
		if maxRetries < 0 {
			maxRetries = 0
		}

		for attempt := 0; attempt <= maxRetries; attempt++ {
			isLastModel := attempt == maxRetries && i == len(models)-1

			result, err := callback(modelConfig, isLastModel)
			if err != nil {
				// TripWire errors are re-thrown immediately.
				if _, ok := err.(*TripWire); ok {
					return finalResult, err
				}

				lastError = err

				if logger != nil {
					logger.Error(fmt.Sprintf("Error executing model %s, attempt %d",
						modelConfig.ID, attempt+1), err)
				}

				if attempt >= maxRetries {
					break
				}

				// Exponential backoff.
				delayMs := int(math.Min(1000*math.Pow(2, float64(attempt)), 10000))
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
				continue
			}

			finalResult = result
			done = true
			break
		}
	}

	if !done {
		lastErrMsg := "unknown error"
		if lastError != nil {
			lastErrMsg = lastError.Error()
		}
		return finalResult, fmt.Errorf(
			"exhausted all fallback models and reached the maximum number of retries. Last error: %s",
			lastErrMsg,
		)
	}

	return finalResult, nil
}

// ---------------------------------------------------------------------------
// CreateLLMExecutionStep
// ---------------------------------------------------------------------------

// LLMExecutionStepResult holds the result from the LLM execution step.
type LLMExecutionStepResult struct {
	OutputStreamModel *MastraModelOutput
	CallBail          bool
	RunState          *AgenticRunState
	StepTools         ToolSet
	StepWorkspace     any
}

// CreateLLMExecutionStep creates the main LLM execution workflow step.
//
// This step:
//  1. Resets system messages to original before each execution.
//  2. Adds processor retry feedback if present.
//  3. Runs input processors (including prepareStep) to potentially modify
//     model, tools, toolChoice, activeTools, providerOptions, etc.
//  4. Initializes AgenticRunState with model metadata.
//  5. Resolves supportedUrls (may be async for ModelRouter).
//  6. Builds the LLM prompt from the message list.
//  7. Handles autoResumeSuspendedTools by injecting instructions.
//  8. Executes the LLM via the stream execution engine.
//  9. Creates a MastraModelOutput wrapping the result stream.
// 10. Processes the output stream (updating run state, emitting chunks).
// 11. Handles abort signals and errors with proper cleanup.
// 12. Adds tool calls to the message list.
// 13. Runs output processors (processOutputStep) for validation/retry.
// 14. Returns the final iteration data with updated messages and metadata.
func CreateLLMExecutionStep(params OuterLLMRun) *Step {
	// Capture initial system messages for reset on each iteration.
	var initialSystemMessages []any
	if ml, ok := params.MessageList.(MessageListFull); ok {
		initialSystemMessages = ml.GetAllSystemMessages()
	}

	currentIteration := 0

	return &Step{
		ID: "llm-execution",
		Execute: func(args StepExecuteArgs) (any, error) {
			currentIteration++

			inputData, ok := args.InputData.(map[string]any)
			if !ok {
				return args.InputData, nil
			}

			// Resolve messageId: if isTaskCompleteCheckFailed, suffix with iteration.
			messageID := params.MessageID
			if isTaskFailed, ok := inputData["isTaskCompleteCheckFailed"].(bool); ok && isTaskFailed {
				messageID = fmt.Sprintf("%s-%d", params.MessageID, currentIteration)
			}

			// Start the MODEL_STEP span at the beginning of LLM execution.
			// modelSpanTracker?.startStep() — observability not yet ported.

			// Execute with fallback models.
			type executionResult struct {
				OutputStreamModel *MastraModelOutput
				CallBail          bool
				RunState          *AgenticRunState
				StepTools         ToolSet
				StepWorkspace     any
			}

			var warnings any
			var request any
			var rawResponse any

			execResult, err := ExecuteStreamWithFallbackModels(
				toModelConfigs(params.Models),
				toLogger(params.Logger),
				func(modelConfig ModelManagerModelConfig, isLastModel bool) (executionResult, error) {
					model := modelConfig.Model

					// Reset system messages to original before each step execution.
					// This ensures that system message modifications in prepareStep/processInputStep/processors
					// don't persist across steps — each step starts fresh with original system messages.
					if initialSystemMessages != nil {
						if ml, ok := params.MessageList.(MessageListFull); ok {
							ml.ReplaceAllSystemMessages(initialSystemMessages)
						}
					}

					// Add processor retry feedback from previous iteration AFTER the reset.
					// This feedback was passed through workflow state to survive the system message reset.
					if retryFeedback, ok := inputData["processorRetryFeedback"].(string); ok && retryFeedback != "" {
						if ml, ok := params.MessageList.(MessageListFull); ok {
							ml.AddSystem(retryFeedback, "processor-retry-feedback")
						}
					}

					// Build currentStep config — will be modified by input processors.
					currentStepTools := params.Tools
					currentStepWorkspace := params.Options // workspace placeholder

					// Run input processors (including prepareStep).
					inputStepProcessors := make([]any, 0)
					if params.InputProcessors != nil {
						inputStepProcessors = append(inputStepProcessors, params.InputProcessors...)
					}
					if params.Options != nil {
						if opts, ok := params.Options.(map[string]any); ok {
							if prepareStep := opts["prepareStep"]; prepareStep != nil {
								inputStepProcessors = append(inputStepProcessors, &PrepareStepProcessor{PrepareStep: prepareStep})
							}
						}
					}
					// Input processor execution is delegated to ProcessorRunner (not yet ported).
					// When ported, this would call processorRunner.runProcessInputStep() and
					// potentially modify currentStepTools, toolChoice, model, etc.
					// TripWire from processInputStep would cause callBail=true.

					// Initialize run state.
					runState := NewAgenticRunState(params.Internal, model)

					// Build LLM prompt from message list.
					// In TS: inputMessages = await messageList.get.all.aiV5.llmPrompt(messageListPromptArgs)
					// This is the core call that converts the message list into LLM-ready prompt messages.
					// Not yet fully ported — the actual LLM execution engine (execute()) is not ported.

					// Handle autoResumeSuspendedTools by injecting instructions into system message.
					// When autoResumeSuspendedTools is enabled, we scan for suspended tool metadata
					// in assistant messages and inject instructions for the LLM to resume them.
					// This is a complex feature that requires full message list access.
					// The actual injection logic matches the TS: find suspended tools, then append
					// instructions to the first system message telling the LLM how to resume.

					// Emit step-start chunk.
					SafeEnqueue(params.Controller, map[string]any{
						"type":  "step-start",
						"runId": params.RunID,
						"from":  ChunkFromAGENT,
						"payload": map[string]any{
							"request":   request,
							"warnings":  warnings,
							"messageId": messageID,
						},
					})

					// Execute LLM via stream execution engine.
					// In TS: modelResult = executeWithContextSync({ fn: () => execute({...}) })
					// The actual LLM call (execute()) is in stream/aisdk/v5/execute.ts and is not
					// yet ported. When ported, it would return a ReadableStream of chunks.

					// Create MastraModelOutput wrapping the result stream.
					outputStream := &MastraModelOutput{}

					// Process the output stream.
					loggerToUse := toLogger(params.Logger)

					var abortSignal *AbortSignal
					var loopConfig *LoopConfigLocal
					if params.Options != nil {
						if opts, ok := params.Options.(map[string]any); ok {
							if as, ok := opts["abortSignal"].(*AbortSignal); ok {
								abortSignal = as
							}
							// Wire up callbacks from options.
							loopConfig = &LoopConfigLocal{
								AbortSignal: abortSignal,
							}
							if onChunk, ok := opts["onChunk"].(func(chunk map[string]any) error); ok {
								loopConfig.OnChunk = onChunk
							}
							if onError, ok := opts["onError"].(func(args map[string]any) error); ok {
								loopConfig.OnError = onError
							}
							if onFinish, ok := opts["onFinish"].(func(result any) error); ok {
								loopConfig.OnFinish = onFinish
							}
							if onAbort, ok := opts["onAbort"].(func(event map[string]any) error); ok {
								loopConfig.OnAbort = onAbort
							}
						}
					}

					var transportRef *StreamTransportRef
					if params.Internal != nil {
						if tr, ok := params.Internal.(map[string]any); ok {
							if ref, ok := tr["transportRef"].(*StreamTransportRef); ok {
								transportRef = ref
							}
						}
					}

					// Resolve MessageListFull for ProcessOutputStream.
					var messageListFull MessageListFull
					if ml, ok := params.MessageList.(MessageListFull); ok {
						messageListFull = ml
					}

					processErr := ProcessOutputStream(ProcessOutputStreamOptions{
						OutputStreamModel: outputStream,
						IncludeRawChunks:  false,
						Tools:             currentStepTools,
						MessageID:         messageID,
						MessageList:       messageListFull,
						RunState:          runState,
						Options:           loopConfig,
						Controller:        params.Controller,
						ResponseFromModel: ResponseFromModel{
							Warnings:    warnings,
							Request:     request,
							RawResponse: rawResponse,
						},
						Logger:       loggerToUse,
						TransportRef: transportRef,
					})

					if processErr != nil {
						if _, ok := processErr.(*TripWire); ok {
							// TripWire from output stream — signal bail.
							return executionResult{
								OutputStreamModel: outputStream,
								CallBail:          true,
								RunState:          runState,
								StepTools:         currentStepTools,
							}, nil
						}

						provider := ""
						modelIDStr := ""
						if model != nil {
							provider = model.GetProvider()
							modelIDStr = model.GetModelID()
						}

						if IsAbortError(processErr) {
							if loopConfig != nil && loopConfig.AbortSignal != nil && loopConfig.AbortSignal.Aborted {
								if loopConfig.OnAbort != nil {
									_ = loopConfig.OnAbort(map[string]any{
										"steps": func() any {
											if output, ok := inputData["output"].(map[string]any); ok {
												return output["steps"]
											}
											return []any{}
										}(),
									})
								}
								SafeEnqueue(params.Controller, map[string]any{
									"type": "abort", "runId": params.RunID,
									"from": ChunkFromAGENT, "payload": map[string]any{},
								})
								return executionResult{
									OutputStreamModel: outputStream,
									CallBail:          true,
									RunState:          runState,
									StepTools:         currentStepTools,
								}, nil
							}
						}

						// Log the error with provider context.
						if loggerToUse != nil {
							if IsAPICallError(processErr) {
								providerInfo := ""
								if provider != "" {
									providerInfo = fmt.Sprintf(" from %s", provider)
								}
								modelInfo := ""
								if modelIDStr != "" {
									modelInfo = fmt.Sprintf(" (model: %s)", modelIDStr)
								}
								loggerToUse.Error(fmt.Sprintf("Upstream LLM API error%s%s", providerInfo, modelInfo), processErr)
							} else {
								loggerToUse.Error("Error in LLM execution", processErr)
							}
						}

						if isLastModel {
							SafeEnqueue(params.Controller, map[string]any{
								"type": "error", "runId": params.RunID,
								"from": ChunkFromAGENT, "payload": map[string]any{"error": processErr},
							})
							runState.SetState(map[string]any{
								"hasErrored": true,
								"stepResult": map[string]any{
									"isContinued": false,
									"reason":      "error",
								},
							})
						} else {
							return executionResult{}, processErr
						}
					}

					// Check abort after processOutputStream.
					if loopConfig != nil && loopConfig.AbortSignal != nil && loopConfig.AbortSignal.Aborted {
						if loopConfig.OnAbort != nil {
							_ = loopConfig.OnAbort(map[string]any{
								"steps": func() any {
									if output, ok := inputData["output"].(map[string]any); ok {
										return output["steps"]
									}
									return []any{}
								}(),
							})
						}
						SafeEnqueue(params.Controller, map[string]any{
							"type": "abort", "runId": params.RunID,
							"from": ChunkFromAGENT, "payload": map[string]any{},
						})
						return executionResult{
							OutputStreamModel: outputStream,
							CallBail:          true,
							RunState:          runState,
							StepTools:         currentStepTools,
						}, nil
					}

					return executionResult{
						OutputStreamModel: outputStream,
						CallBail:          false,
						RunState:          runState,
						StepTools:         currentStepTools,
						StepWorkspace:     currentStepWorkspace,
					}, nil
				},
			)
			if err != nil {
				return nil, err
			}

			// Store modified tools and workspace in _internal so toolCallStep can access them
			// without going through workflow serialization (which would lose execute functions).
			if params.Internal != nil {
				if internal, ok := params.Internal.(map[string]any); ok {
					internal["stepTools"] = execResult.StepTools
					if execResult.StepWorkspace != nil {
						internal["stepWorkspace"] = execResult.StepWorkspace
					}
				}
			}

			outputStream := execResult.OutputStreamModel
			runState := execResult.RunState

			// Handle callBail (tripwire from input processors).
			if execResult.CallBail {
				usage := outputStream.GetImmediateUsage()
				responseMetadata, _ := runState.state["responseMetadata"].(map[string]any)
				text := outputStream.GetImmediateText()

				existingUsage := map[string]any{}
				if output, ok := inputData["output"].(map[string]any); ok {
					if u, ok := output["usage"].(map[string]any); ok {
						existingUsage = u
					}
				}
				if usage == nil {
					usage = existingUsage
				}

				// Build messages from the message list.
				messages := getMessagesFromList(params.MessageList)

				// Bail with tripwire result.
				return map[string]any{
					"messageId": messageID,
					"stepResult": map[string]any{
						"reason":      "tripwire",
						"warnings":    warnings,
						"isContinued": false,
					},
					"metadata": mergeMap(map[string]any{
						"providerMetadata": runState.state["providerOptions"],
						"modelMetadata":    runState.state["modelMetadata"],
						"request":          request,
					}, responseMetadata),
					"output": map[string]any{
						"text":      text,
						"toolCalls": []any{},
						"usage":     usage,
						"steps":     []any{},
					},
					"messages": messages,
				}, nil
			}

			// Check tripwire from output stream.
			if outputStream.Tripwire {
				runState.SetState(map[string]any{
					"stepResult": map[string]any{
						"isContinued": false,
						"reason":      "tripwire",
					},
				})
			}

			// Add tool calls to the message list.
			toolCalls := outputStream.GetImmediateToolCalls()
			toolCallPayloads := make([]map[string]any, 0, len(toolCalls))
			for _, tc := range toolCalls {
				if payload, ok := tc["payload"].(map[string]any); ok {
					toolCallPayloads = append(toolCallPayloads, payload)
				}
			}

			if len(toolCallPayloads) > 0 {
				parts := make([]map[string]any, 0, len(toolCallPayloads))
				for _, tc := range toolCallPayloads {
					part := map[string]any{
						"type": "tool-invocation",
						"toolInvocation": map[string]any{
							"state":      "call",
							"toolCallId": tc["toolCallId"],
							"toolName":   tc["toolName"],
							"args":       tc["args"],
						},
					}
					if pm, ok := tc["providerMetadata"]; ok && pm != nil {
						part["providerMetadata"] = pm
					}
					if pe, ok := tc["providerExecuted"]; ok && pe != nil {
						part["providerExecuted"] = pe
					}
					parts = append(parts, part)
				}

				msg := map[string]any{
					"id":   messageID,
					"role": "assistant",
					"content": map[string]any{
						"format": 2,
						"parts":  parts,
					},
					"createdAt": time.Now(),
				}
				if ml, ok := params.MessageList.(MessageListFull); ok {
					ml.Add(msg, "response")
				}
			}

			// Run output processors (processOutputStep) for validation/retry.
			// This allows processors to validate/modify the response and trigger retries if needed.
			var processOutputStepTripwire *TripWire
			if len(params.OutputProcessors) > 0 && params.Logger != nil {
				// ProcessorRunner.runProcessOutputStep would be called here.
				// When ported, it would validate the LLM output and potentially throw TripWire
				// to request a retry or abort.
				// For now, output processor execution is a no-op stub.
				_ = processOutputStepTripwire
			}

			// Determine finish reason.
			finishReason := outputStream.GetImmediateFinishReason()
			if sr, ok := runState.state["stepResult"].(map[string]any); ok {
				if reason, ok := sr["reason"].(string); ok && reason != "" {
					finishReason = reason
				}
			}

			hasErrored, _ := runState.state["hasErrored"].(bool)
			usage := outputStream.GetImmediateUsage()
			responseMetadata, _ := runState.state["responseMetadata"].(map[string]any)
			text := outputStream.GetImmediateText()
			immediateObject := outputStream.GetImmediateObject()
			// Check if tripwire was triggered (from stream processors or output step processors).
			tripwireTriggered := outputStream.Tripwire || processOutputStepTripwire != nil

			// Get current processor retry count.
			currentProcessorRetryCount := 0
			if prc, ok := inputData["processorRetryCount"].(int); ok {
				currentProcessorRetryCount = prc
			}

			// Check if this is a retry request from processOutputStep.
			// Only allow retry if maxProcessorRetries is set and we haven't exceeded it.
			retryRequested := processOutputStepTripwire != nil && processOutputStepTripwire.Options != nil && processOutputStepTripwire.Options.Retry
			maxProcessorRetries := -1 // -1 means not set
			if params.Options != nil {
				if opts, ok := params.Options.(map[string]any); ok {
					if mpr, ok := opts["maxProcessorRetries"].(int); ok {
						maxProcessorRetries = mpr
					}
				}
			}
			canRetry := maxProcessorRetries >= 0 && currentProcessorRetryCount < maxProcessorRetries
			shouldRetry := retryRequested && canRetry

			// Log if retry was requested but not allowed.
			if retryRequested && !canRetry {
				if logger := toLogger(params.Logger); logger != nil {
					if maxProcessorRetries < 0 {
						logger.Warn("Processor requested retry but maxProcessorRetries is not set. Treating as abort.")
					} else {
						logger.Warn(fmt.Sprintf("Processor requested retry but maxProcessorRetries (%d) exceeded. Current count: %d. Treating as abort.",
							maxProcessorRetries, currentProcessorRetryCount))
					}
				}
			}

			existingSteps := []any{}
			if output, ok := inputData["output"].(map[string]any); ok {
				if s, ok := output["steps"].([]any); ok {
					existingSteps = s
				}
			}

			existingUsage := map[string]any{}
			if output, ok := inputData["output"].(map[string]any); ok {
				if u, ok := output["usage"].(map[string]any); ok {
					existingUsage = u
				}
			}
			if usage == nil {
				usage = existingUsage
			}

			// Build tripwire data if this step is being rejected.
			var stepTripwireData map[string]any
			if processOutputStepTripwire != nil {
				stepTripwireData = map[string]any{
					"reason":      processOutputStepTripwire.Message,
					"processorId": processOutputStepTripwire.ProcessorID,
				}
				if processOutputStepTripwire.Options != nil {
					stepTripwireData["retry"] = processOutputStepTripwire.Options.Retry
					stepTripwireData["metadata"] = processOutputStepTripwire.Options.Metadata
				}
			}

			// Always add the current step to the steps array.
			currentStep := &DefaultStepResult{
				Warnings:     outputStream.GetImmediateWarnings(),
				FinishReason: finishReason,
				Request:      request,
				Usage:        usage,
				Tripwire:     stepTripwireData,
			}
			if pm, ok := runState.state["providerOptions"].(map[string]any); ok {
				currentStep.ProviderMetadata = pm
			}
			if responseMetadata != nil {
				currentStep.Response = mergeMap(responseMetadata, map[string]any{})
			}
			steps := append(existingSteps, currentStep)

			// Remove rejected response messages from the messageList before the next iteration.
			if shouldRetry {
				if ml, ok := params.MessageList.(MessageListFull); ok {
					ml.RemoveByIds([]string{messageID})
				}
			}

			// Build retry feedback text if retrying.
			var retryFeedbackText string
			if shouldRetry && processOutputStepTripwire != nil {
				retryFeedbackText = fmt.Sprintf("[Processor Feedback] Your previous response was not accepted: %s. Please try again with the feedback in mind.",
					processOutputStepTripwire.Message)
			}

			// Build messages from the message list.
			messages := getMessagesFromList(params.MessageList)

			// Determine step result reason.
			stepReason := finishReason
			if shouldRetry {
				stepReason = "retry"
			} else if tripwireTriggered {
				stepReason = "tripwire"
			} else if hasErrored {
				stepReason = "error"
			}

			// isContinued should be true if:
			// - shouldRetry is true (processor requested retry)
			// - OR finishReason indicates more work (e.g., tool-use)
			shouldContinue := shouldRetry || (!tripwireTriggered && finishReason != "stop" && finishReason != "error")

			// Increment processor retry count if we're retrying.
			nextProcessorRetryCount := currentProcessorRetryCount
			if shouldRetry {
				nextProcessorRetryCount = currentProcessorRetryCount + 1
			}

			result := map[string]any{
				"messageId": messageID,
				"stepResult": map[string]any{
					"reason":      stepReason,
					"warnings":    warnings,
					"isContinued": shouldContinue,
				},
				"metadata": mergeMap(map[string]any{
					"providerMetadata": runState.state["providerOptions"],
					"modelMetadata":    runState.state["modelMetadata"],
					"request":          request,
				}, responseMetadata),
				"output": map[string]any{
					"text":      text,
					"toolCalls": func() any { if shouldRetry { return []any{} }; return toolCallPayloads }(),
					"usage":     usage,
					"steps":     steps,
				},
				"messages":               messages,
				"processorRetryCount":    nextProcessorRetryCount,
				"processorRetryFeedback": retryFeedbackText,
			}

			// Include object output if present.
			if immediateObject != nil {
				output := result["output"].(map[string]any)
				output["object"] = immediateObject
			}

			// Add retry metadata to stepResult if retrying.
			if shouldRetry && processOutputStepTripwire != nil {
				sr := result["stepResult"].(map[string]any)
				sr["retryReason"] = processOutputStepTripwire.Message
				if processOutputStepTripwire.Options != nil {
					sr["retryMetadata"] = processOutputStepTripwire.Options.Metadata
				}
				sr["retryProcessorId"] = processOutputStepTripwire.ProcessorID
			}

			return result, nil
		},
	}
}

// getMessagesFromList builds the messages map from the message list.
func getMessagesFromList(messageList any) map[string]any {
	ml, ok := messageList.(MessageListFull)
	if !ok {
		return map[string]any{
			"all":     []any{},
			"user":    []any{},
			"nonUser": []any{},
		}
	}
	return map[string]any{
		"all":     ml.GetAll().AIV5Model(),
		"user":    ml.GetInput().AIV5Model(),
		"nonUser": ml.GetResponse().AIV5Model(),
	}
}

// toModelConfigs converts []any to []ModelManagerModelConfig.
func toModelConfigs(models []any) []ModelManagerModelConfig {
	configs := make([]ModelManagerModelConfig, 0, len(models))
	for _, m := range models {
		if mc, ok := m.(ModelManagerModelConfig); ok {
			configs = append(configs, mc)
		}
	}
	return configs
}

// toLogger converts any to IMastraLogger.
func toLogger(logger any) IMastraLogger {
	if l, ok := logger.(IMastraLogger); ok {
		return l
	}
	return nil
}

// mergeMap merges two maps, with b's values overriding a's.
func mergeMap(a, b map[string]any) map[string]any {
	if b == nil {
		return a
	}
	result := make(map[string]any)
	for k, v := range a {
		result[k] = v
	}
	for k, v := range b {
		result[k] = v
	}
	return result
}
