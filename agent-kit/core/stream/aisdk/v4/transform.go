// Ported from: packages/core/src/stream/aisdk/v4/transform.ts
package v4

import (
	"encoding/json"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

// ---------------------------------------------------------------------------
// Stub types for AI SDK v4 stream parts not yet ported
// ---------------------------------------------------------------------------

// LanguageModelV1StreamPart mirrors @internal/ai-sdk-v4 LanguageModelV1StreamPart.
// This is a discriminated union in TS; in Go we use a struct with a Type field
// and callers switch on Type to interpret the other fields.
// ai-kit only ported V3. V4 types remain local stubs.
type LanguageModelV1StreamPart struct {
	Type string `json:"type"`

	// step-start fields
	MessageID string         `json:"messageId,omitempty"`
	Request   *RequestData   `json:"request,omitempty"`
	Warnings  []any          `json:"warnings,omitempty"`

	// tool-call fields
	ToolCallID string `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	Args       any    `json:"args,omitempty"`

	// tool-result fields
	Result any `json:"result,omitempty"`

	// text-delta fields
	ID        string `json:"id,omitempty"`
	TextDelta string `json:"textDelta,omitempty"`

	// step-finish / finish fields
	FinishReason     string                              `json:"finishReason,omitempty"`
	Usage            *LanguageModelUsageV4               `json:"usage,omitempty"`
	TotalUsage       *LanguageModelUsageV4               `json:"totalUsage,omitempty"`
	Response         *stream.LanguageModelV2ResponseMetadata `json:"response,omitempty"`
	ProviderMetadata stream.ProviderMetadata             `json:"providerMetadata,omitempty"`
	IsContinued      *bool                               `json:"isContinued,omitempty"`
	Logprobs         *stream.LanguageModelV1LogProbs     `json:"logprobs,omitempty"`
	Messages         *MessagesData                       `json:"messages,omitempty"`

	// tripwire fields
	Reason      string `json:"reason,omitempty"`
	Retry       *bool  `json:"retry,omitempty"`
	Metadata    any    `json:"metadata,omitempty"`
	ProcessorID string `json:"processorId,omitempty"`
}

// RequestData mirrors the request object on step-start stream parts.
type RequestData struct {
	Body string `json:"body,omitempty"`
}

// MessagesData mirrors the messages object on step-finish / finish stream parts.
type MessagesData struct {
	All     []stream.ModelMessage `json:"all"`
	User    []stream.ModelMessage `json:"user"`
	NonUser []stream.ModelMessage `json:"nonUser"`
}

// ---------------------------------------------------------------------------
// ConvertFullStreamChunkToMastra
// ---------------------------------------------------------------------------

// ConvertFullStreamChunkToMastra converts an AI SDK v4 LanguageModelV1StreamPart
// to a Mastra ChunkType. Returns nil if the chunk type is not recognized.
//
// This is the v4-specific transform that maps provider stream events
// into Mastra's unified chunk format.
func ConvertFullStreamChunkToMastra(value LanguageModelV1StreamPart, ctx TransformContext) *stream.ChunkType {
	switch value.Type {
	case "step-start":
		return convertStepStart(value, ctx)
	case "tool-call":
		return convertToolCall(value, ctx)
	case "tool-result":
		return convertToolResult(value, ctx)
	case "text-delta":
		return convertTextDelta(value, ctx)
	case "step-finish":
		return convertStepFinish(value, ctx)
	case "finish":
		return convertFinish(value, ctx)
	case "tripwire":
		return convertTripwire(value, ctx)
	default:
		return nil
	}
}

// TransformContext carries context for the transform operation.
type TransformContext struct {
	RunID string
}

// ---------------------------------------------------------------------------
// Individual chunk converters
// ---------------------------------------------------------------------------

func convertStepStart(value LanguageModelV1StreamPart, ctx TransformContext) *stream.ChunkType {
	// Parse request body from JSON string to map, defaulting to empty object.
	var requestBody map[string]any
	if value.Request != nil && value.Request.Body != "" {
		if err := json.Unmarshal([]byte(value.Request.Body), &requestBody); err != nil {
			requestBody = map[string]any{}
		}
	} else {
		requestBody = map[string]any{}
	}

	return &stream.ChunkType{
		BaseChunkType: stream.BaseChunkType{
			RunID: ctx.RunID,
		},
		Type: "step-start",
		Payload: stream.StepStartPayload{
			MessageID: value.MessageID,
			Request:   map[string]any{"body": requestBody},
			Warnings:  toCallWarnings(value.Warnings),
		},
	}
}

func convertToolCall(value LanguageModelV1StreamPart, ctx TransformContext) *stream.ChunkType {
	return &stream.ChunkType{
		BaseChunkType: stream.BaseChunkType{
			RunID: ctx.RunID,
			From:  stream.ChunkFromAgent,
		},
		Type: "tool-call",
		Payload: stream.ToolCallPayload{
			ToolCallID: value.ToolCallID,
			Args:       value.Args,
			ToolName:   value.ToolName,
		},
	}
}

func convertToolResult(value LanguageModelV1StreamPart, ctx TransformContext) *stream.ChunkType {
	return &stream.ChunkType{
		BaseChunkType: stream.BaseChunkType{
			RunID: ctx.RunID,
			From:  stream.ChunkFromAgent,
		},
		Type: "tool-result",
		Payload: stream.ToolResultPayload{
			ToolCallID: value.ToolCallID,
			ToolName:   value.ToolName,
			Result:     value.Result,
		},
	}
}

func convertTextDelta(value LanguageModelV1StreamPart, ctx TransformContext) *stream.ChunkType {
	return &stream.ChunkType{
		BaseChunkType: stream.BaseChunkType{
			RunID: ctx.RunID,
			From:  stream.ChunkFromAgent,
		},
		Type: "text-delta",
		Payload: stream.TextDeltaPayload{
			ID:   value.ID,
			Text: value.TextDelta,
		},
	}
}

func convertStepFinish(value LanguageModelV1StreamPart, ctx TransformContext) *stream.ChunkType {
	var messages *stream.StepFinishPayloadMessages
	if value.Messages != nil {
		messages = &stream.StepFinishPayloadMessages{
			All:     value.Messages.All,
			User:    value.Messages.User,
			NonUser: toResponseMessages(value.Messages.NonUser),
		}
	}

	return &stream.ChunkType{
		BaseChunkType: stream.BaseChunkType{
			RunID: ctx.RunID,
			From:  stream.ChunkFromAgent,
		},
		Type: "step-finish",
		Payload: stream.StepFinishPayload{
			ID:               value.ID,
			Response:         value.Response,
			MessageID:        value.MessageID,
			ProviderMetadata: value.ProviderMetadata,
			StepResult: stream.StepFinishPayloadStepResult{
				Reason:      stream.LanguageModelV2FinishReason(value.FinishReason),
				Warnings:    toCallWarnings(value.Warnings),
				IsContinued: value.IsContinued,
				Logprobs:    value.Logprobs,
			},
			Output: stream.StepFinishPayloadOutput{
				Usage: toUsage(value.Usage),
			},
			Metadata: stream.StepFinishPayloadMetadata{
				Request:          toRequestMetadata(value.Request),
				ProviderMetadata: value.ProviderMetadata,
			},
			Messages: messages,
		},
	}
}

func convertFinish(value LanguageModelV1StreamPart, ctx TransformContext) *stream.ChunkType {
	// In the TS, messages is optional-chained with fallback to empty arrays.
	var messages *stream.StepFinishPayloadMessages
	if value.Messages != nil {
		messages = &stream.StepFinishPayloadMessages{
			All:     value.Messages.All,
			User:    value.Messages.User,
			NonUser: toResponseMessages(value.Messages.NonUser),
		}
	} else {
		messages = &stream.StepFinishPayloadMessages{
			All:     []stream.ModelMessage{},
			User:    []stream.ModelMessage{},
			NonUser: []stream.AIV5ResponseMessage{},
		}
	}

	// The TS finish chunk uses FinishPayload (not StepFinishPayload).
	// It includes totalUsage and the same stepResult/output/metadata/messages structure.
	return &stream.ChunkType{
		BaseChunkType: stream.BaseChunkType{
			RunID: ctx.RunID,
			From:  stream.ChunkFromAgent,
		},
		Type: "finish",
		Payload: FinishChunkPayload{
			ID:               value.ID,
			Usage:            toUsage(value.Usage),
			TotalUsage:       toUsage(value.TotalUsage),
			ProviderMetadata: value.ProviderMetadata,
			StepResult: stream.FinishPayloadStepResult{
				Reason:      value.FinishReason,
				Warnings:    toCallWarnings(value.Warnings),
				IsContinued: value.IsContinued,
				Logprobs:    value.Logprobs,
			},
			Output: stream.FinishPayloadOutput{
				Usage: toUsage(value.Usage),
			},
			Metadata: stream.FinishPayloadMetadata{
				Request:          toRequestMetadata(value.Request),
				ProviderMetadata: value.ProviderMetadata,
			},
			Messages: stream.FinishPayloadMessages{
				All:     messages.All,
				User:    messages.User,
				NonUser: messages.NonUser,
			},
		},
	}
}

func convertTripwire(value LanguageModelV1StreamPart, ctx TransformContext) *stream.ChunkType {
	return &stream.ChunkType{
		BaseChunkType: stream.BaseChunkType{
			RunID: ctx.RunID,
			From:  stream.ChunkFromAgent,
		},
		Type: "tripwire",
		Payload: stream.TripwirePayload{
			Reason:      value.Reason,
			Retry:       value.Retry,
			Metadata:    value.Metadata,
			ProcessorID: value.ProcessorID,
		},
	}
}

// ---------------------------------------------------------------------------
// FinishChunkPayload — the payload shape emitted by the "finish" chunk.
// This extends FinishPayload with additional fields like ID, Usage, TotalUsage.
// ---------------------------------------------------------------------------

// FinishChunkPayload is the payload for a "finish" chunk, which includes
// top-level id/usage/totalUsage in addition to the nested stepResult/output/metadata/messages.
type FinishChunkPayload struct {
	ID               string                        `json:"id,omitempty"`
	Usage            stream.LanguageModelUsage     `json:"usage"`
	TotalUsage       stream.LanguageModelUsage     `json:"totalUsage"`
	ProviderMetadata stream.ProviderMetadata       `json:"providerMetadata,omitempty"`
	StepResult       stream.FinishPayloadStepResult  `json:"stepResult"`
	Output           stream.FinishPayloadOutput      `json:"output"`
	Metadata         stream.FinishPayloadMetadata    `json:"metadata"`
	Messages         stream.FinishPayloadMessages    `json:"messages"`
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// toUsage converts a *LanguageModelUsageV4 to stream.LanguageModelUsage.
func toUsage(usage *LanguageModelUsageV4) stream.LanguageModelUsage {
	if usage == nil {
		return stream.LanguageModelUsage{}
	}
	totalTokens := 0
	if usage.TotalTokens != nil {
		totalTokens = *usage.TotalTokens
	}
	return stream.LanguageModelUsage{
		InputTokens:  usage.PromptTokens,
		OutputTokens: usage.CompletionTokens,
		TotalTokens:  totalTokens,
	}
}

// toCallWarnings converts []any warnings to []stream.LanguageModelV2CallWarning.
// The TS source passes warnings through as-is; here we do a best-effort conversion.
func toCallWarnings(warnings []any) []stream.LanguageModelV2CallWarning {
	if warnings == nil {
		return nil
	}
	result := make([]stream.LanguageModelV2CallWarning, 0, len(warnings))
	for _, w := range warnings {
		switch v := w.(type) {
		case stream.LanguageModelV2CallWarning:
			result = append(result, v)
		case map[string]any:
			cw := stream.LanguageModelV2CallWarning{}
			if t, ok := v["type"].(string); ok {
				cw.Type = t
			}
			if s, ok := v["setting"].(string); ok {
				cw.Setting = s
			}
			if d, ok := v["details"].(string); ok {
				cw.Details = d
			}
			if m, ok := v["message"].(string); ok {
				cw.Message = m
			}
			result = append(result, cw)
		}
	}
	return result
}

// toRequestMetadata converts a *RequestData to *stream.LanguageModelRequestMetadata.
func toRequestMetadata(req *RequestData) *stream.LanguageModelRequestMetadata {
	if req == nil {
		return nil
	}
	return &stream.LanguageModelRequestMetadata{
		Body: map[string]any{"body": req.Body},
	}
}

// toResponseMessages converts []stream.ModelMessage to []stream.AIV5ResponseMessage.
// In the TS source, nonUser messages are typed as AIV5ResponseMessage (which is map[string]any).
// ModelMessage is also map[string]any in Go, so this is a direct cast.
func toResponseMessages(msgs []stream.ModelMessage) []stream.AIV5ResponseMessage {
	if msgs == nil {
		return nil
	}
	result := make([]stream.AIV5ResponseMessage, len(msgs))
	for i, m := range msgs {
		result[i] = stream.AIV5ResponseMessage(m)
	}
	return result
}
