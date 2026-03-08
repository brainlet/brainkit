// Ported from: packages/deepseek/src/chat/deepseek-chat-language-model.ts
package deepseek

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ChatConfig holds the configuration for a DeepSeek chat language model.
type ChatConfig struct {
	// Provider is the provider identifier (e.g. "deepseek.chat").
	Provider string

	// URL constructs the API URL from a model ID and path.
	URL func(modelID string, path string) string

	// Headers returns the HTTP headers to include with requests.
	Headers func() map[string]string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction
}

// ChatLanguageModel implements languagemodel.LanguageModel for DeepSeek APIs.
type ChatLanguageModel struct {
	modelID               string
	config                ChatConfig
	failedResponseHandler providerutils.ResponseHandler[error]
}

// NewChatLanguageModel creates a new DeepSeek ChatLanguageModel.
func NewChatLanguageModel(modelID string, config ChatConfig) *ChatLanguageModel {
	m := &ChatLanguageModel{
		modelID: modelID,
		config:  config,
	}

	// Initialize error handling
	typedHandler := providerutils.CreateJsonErrorResponseHandler(
		deepSeekErrorSchema,
		func(e DeepSeekErrorData) string {
			return e.Error.Message
		},
		nil,
	)
	m.failedResponseHandler = func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
		res, err := typedHandler(opts)
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, nil
		}
		return &providerutils.ResponseHandlerResult[error]{
			Value:           res.Value,
			RawValue:        res.RawValue,
			ResponseHeaders: res.ResponseHeaders,
		}, nil
	}

	return m
}

// SpecificationVersion returns "v3".
func (m *ChatLanguageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *ChatLanguageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *ChatLanguageModel) ModelID() string { return m.modelID }

// SupportedUrls returns supported URL patterns (empty for DeepSeek).
func (m *ChatLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return map[string][]*regexp.Regexp{}, nil
}

// providerOptionsName extracts the provider name from the provider string
// (e.g. "deepseek.chat" -> "deepseek").
func (m *ChatLanguageModel) providerOptionsName() string {
	parts := strings.SplitN(m.config.Provider, ".", 2)
	return strings.TrimSpace(parts[0])
}

// getArgs prepares the request arguments from CallOptions.
func (m *ChatLanguageModel) getArgs(opts languagemodel.CallOptions) (args map[string]any, warnings []shared.Warning, err error) {
	warnings = []shared.Warning{}

	// Parse DeepSeek-specific provider options
	providerOptsMap := toInterfaceMap(opts.ProviderOptions)
	deepseekOptions, err := providerutils.ParseProviderOptions(
		m.providerOptionsName(),
		providerOptsMap,
		deepSeekLanguageModelOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}
	if deepseekOptions == nil {
		deepseekOptions = &DeepSeekLanguageModelOptions{}
	}

	// Convert prompt to DeepSeek messages
	result := ConvertToDeepSeekChatMessages(opts.Prompt, opts.ResponseFormat)

	warnings = append(warnings, result.Warnings...)

	if opts.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}

	if opts.Seed != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "seed"})
	}

	// Prepare tools
	toolResult := PrepareTools(opts.Tools, opts.ToolChoice)

	// Build request body
	args = map[string]any{
		"model":    m.modelID,
		"messages": result.Messages,
	}

	if opts.MaxOutputTokens != nil {
		args["max_tokens"] = *opts.MaxOutputTokens
	}
	if opts.Temperature != nil {
		args["temperature"] = *opts.Temperature
	}
	if opts.TopP != nil {
		args["top_p"] = *opts.TopP
	}
	if opts.FrequencyPenalty != nil {
		args["frequency_penalty"] = *opts.FrequencyPenalty
	}
	if opts.PresencePenalty != nil {
		args["presence_penalty"] = *opts.PresencePenalty
	}

	// Response format
	if _, ok := opts.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		args["response_format"] = map[string]any{
			"type": "json_object",
		}
	}

	if len(opts.StopSequences) > 0 {
		args["stop"] = opts.StopSequences
	}

	// Tools
	if toolResult.Tools != nil {
		args["tools"] = toolResult.Tools
	}
	if toolResult.ToolChoice != nil {
		args["tool_choice"] = toolResult.ToolChoice
	}

	// Thinking mode
	if deepseekOptions.Thinking != nil && deepseekOptions.Thinking.Type != nil {
		args["thinking"] = map[string]any{
			"type": *deepseekOptions.Thinking.Type,
		}
	}

	warnings = append(warnings, toolResult.ToolWarnings...)

	return args, warnings, nil
}

// DoGenerate implements languagemodel.LanguageModel.DoGenerate for non-streaming.
func (m *ChatLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	args, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	bodyJSON, err := json.Marshal(args)
	if err != nil {
		return languagemodel.GenerateResult{}, fmt.Errorf("failed to marshal request body: %w", err)
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[deepSeekChatCompletionResponse]{
		URL:                       m.config.URL(m.modelID, "/chat/completions"),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      args,
		FailedResponseHandler:     m.failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(deepSeekChatCompletionResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	responseBody := result.Value
	choice := responseBody.Choices[0]
	content := []languagemodel.Content{}

	// Reasoning content (before text)
	if choice.Message.ReasoningContent != nil && len(*choice.Message.ReasoningContent) > 0 {
		content = append(content, languagemodel.Reasoning{Text: *choice.Message.ReasoningContent})
	}

	// Tool calls
	if choice.Message.ToolCalls != nil {
		for _, toolCall := range choice.Message.ToolCalls {
			id := ""
			if toolCall.ID != nil {
				id = *toolCall.ID
			} else {
				id = providerutils.GenerateId()
			}
			content = append(content, languagemodel.ToolCall{
				ToolCallID: id,
				ToolName:   toolCall.Function.Name,
				Input:      toolCall.Function.Arguments,
			})
		}
	}

	// Text content
	if choice.Message.Content != nil && len(*choice.Message.Content) > 0 {
		content = append(content, languagemodel.Text{Text: *choice.Message.Content})
	}

	// Finish reason
	finishReason := languagemodel.FinishReason{
		Unified: MapDeepSeekFinishReason(choice.FinishReason),
	}
	if choice.FinishReason != nil {
		finishReason.Raw = choice.FinishReason
	}

	// Provider metadata
	providerMetadata := shared.ProviderMetadata{
		m.providerOptionsName(): map[string]any{},
	}
	if responseBody.Usage != nil {
		if responseBody.Usage.PromptCacheHitTokens != nil {
			providerMetadata[m.providerOptionsName()]["promptCacheHitTokens"] = *responseBody.Usage.PromptCacheHitTokens
		}
		if responseBody.Usage.PromptCacheMissTokens != nil {
			providerMetadata[m.providerOptionsName()]["promptCacheMissTokens"] = *responseBody.Usage.PromptCacheMissTokens
		}
	}

	// Response metadata
	respMeta := getResponseMetadataFromResponse(responseBody)

	return languagemodel.GenerateResult{
		Content:          content,
		FinishReason:     finishReason,
		Usage:            ConvertDeepSeekUsage(responseBody.Usage),
		ProviderMetadata: providerMetadata,
		Request:          &languagemodel.GenerateResultRequest{Body: string(bodyJSON)},
		Response: &languagemodel.GenerateResultResponse{
			ResponseMetadata: respMeta,
			Headers:          result.ResponseHeaders,
			Body:             result.RawValue,
		},
		Warnings: warnings,
	}, nil
}

// DoStream implements languagemodel.LanguageModel.DoStream for streaming.
func (m *ChatLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	args, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	body := make(map[string]any)
	for k, v := range args {
		body[k] = v
	}
	body["stream"] = true
	body["stream_options"] = map[string]any{
		"include_usage": true,
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[any]]{
		URL:                       m.config.URL(m.modelID, "/chat/completions"),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     m.failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(deepSeekChatChunkSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	inputChan := result.Value
	outputChan := make(chan languagemodel.StreamPart)

	providerOptionsName := m.providerOptionsName()
	includeRawChunks := options.IncludeRawChunks != nil && *options.IncludeRawChunks

	go func() {
		defer close(outputChan)

		type toolCallState struct {
			id           string
			functionName string
			functionArgs string
			hasFinished  bool
		}

		var toolCalls []*toolCallState
		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
		}
		var usage *DeepSeekChatTokenUsage
		isFirstChunk := true
		isActiveReasoning := false
		isActiveText := false

		// Start event
		outputChan <- languagemodel.StreamPartStreamStart{Warnings: warnings}

		for chunk := range inputChan {
			// Emit raw chunk if requested
			if includeRawChunks {
				outputChan <- languagemodel.StreamPartRaw{RawValue: chunk.RawValue}
			}

			// Handle failed chunk parsing/validation
			if !chunk.Success {
				finishReason = languagemodel.FinishReason{Unified: languagemodel.FinishReasonError}
				outputChan <- languagemodel.StreamPartError{Error: chunk.Error}
				continue
			}

			// Try to extract the chunk value as a map
			chunkMap, ok := chunk.Value.(map[string]any)
			if !ok {
				continue
			}

			// Check for error chunks
			if errVal, hasError := chunkMap["error"]; hasError {
				finishReason = languagemodel.FinishReason{Unified: languagemodel.FinishReasonError}
				errMsg := ""
				if errObj, ok := errVal.(map[string]any); ok {
					if msg, ok := errObj["message"].(string); ok {
						errMsg = msg
					}
				}
				outputChan <- languagemodel.StreamPartError{Error: errMsg}
				continue
			}

			if isFirstChunk {
				isFirstChunk = false
				respMeta := getResponseMetadataFromMap(chunkMap)
				outputChan <- languagemodel.StreamPartResponseMetadata{
					ResponseMetadata: respMeta,
				}
			}

			// Handle usage
			if usageRaw, ok := chunkMap["usage"]; ok && usageRaw != nil {
				usage = parseDeepSeekTokenUsage(usageRaw)
			}

			// Handle choices
			choicesRaw, ok := chunkMap["choices"]
			if !ok {
				continue
			}
			choices, ok := choicesRaw.([]any)
			if !ok || len(choices) == 0 {
				continue
			}
			choiceMap, ok := choices[0].(map[string]any)
			if !ok {
				continue
			}

			// Check finish_reason
			if fr, ok := choiceMap["finish_reason"]; ok && fr != nil {
				frStr := fmt.Sprintf("%v", fr)
				finishReason = languagemodel.FinishReason{
					Unified: MapDeepSeekFinishReason(&frStr),
					Raw:     &frStr,
				}
			}

			// Check delta
			deltaRaw, ok := choiceMap["delta"]
			if !ok || deltaRaw == nil {
				continue
			}
			delta, ok := deltaRaw.(map[string]any)
			if !ok {
				continue
			}

			// Reasoning content (before text deltas)
			if rc, ok := delta["reasoning_content"]; ok && rc != nil {
				if reasoningContent, ok := rc.(string); ok && reasoningContent != "" {
					if !isActiveReasoning {
						outputChan <- languagemodel.StreamPartReasoningStart{ID: "reasoning-0"}
						isActiveReasoning = true
					}
					outputChan <- languagemodel.StreamPartReasoningDelta{
						ID:    "reasoning-0",
						Delta: reasoningContent,
					}
				}
			}

			// Text content
			if contentVal, ok := delta["content"]; ok && contentVal != nil {
				if contentStr, ok := contentVal.(string); ok && contentStr != "" {
					if !isActiveText {
						outputChan <- languagemodel.StreamPartTextStart{ID: "txt-0"}
						isActiveText = true
					}

					// End reasoning when text starts
					if isActiveReasoning {
						outputChan <- languagemodel.StreamPartReasoningEnd{ID: "reasoning-0"}
						isActiveReasoning = false
					}

					outputChan <- languagemodel.StreamPartTextDelta{
						ID:    "txt-0",
						Delta: contentStr,
					}
				}
			}

			// Tool calls
			if toolCallsRaw, ok := delta["tool_calls"]; ok && toolCallsRaw != nil {
				// End reasoning when tool calls start
				if isActiveReasoning {
					outputChan <- languagemodel.StreamPartReasoningEnd{ID: "reasoning-0"}
					isActiveReasoning = false
				}

				toolCallDeltas, ok := toolCallsRaw.([]any)
				if !ok {
					continue
				}

				for _, tcRaw := range toolCallDeltas {
					tcDelta, ok := tcRaw.(map[string]any)
					if !ok {
						continue
					}

					index := len(toolCalls)
					if idxVal, ok := tcDelta["index"]; ok && idxVal != nil {
						if idxFloat, ok := idxVal.(float64); ok {
							index = int(idxFloat)
						}
					}

					// New tool call
					if index >= len(toolCalls) {
						// Must have id
						tcID, _ := tcDelta["id"].(string)
						if tcID == "" {
							outputChan <- languagemodel.StreamPartError{
								Error: errors.NewInvalidResponseDataError(tcDelta, "Expected 'id' to be a string."),
							}
							continue
						}

						// Must have function name
						funcName := ""
						if fnObj, ok := tcDelta["function"].(map[string]any); ok {
							if name, ok := fnObj["name"].(string); ok {
								funcName = name
							}
						}
						if funcName == "" {
							outputChan <- languagemodel.StreamPartError{
								Error: errors.NewInvalidResponseDataError(tcDelta, "Expected 'function.name' to be a string."),
							}
							continue
						}

						outputChan <- languagemodel.StreamPartToolInputStart{
							ID:       tcID,
							ToolName: funcName,
						}

						funcArgs := ""
						if fnObj, ok := tcDelta["function"].(map[string]any); ok {
							if a, ok := fnObj["arguments"].(string); ok {
								funcArgs = a
							}
						}

						// Extend toolCalls slice to fit the index
						for len(toolCalls) <= index {
							toolCalls = append(toolCalls, nil)
						}

						toolCalls[index] = &toolCallState{
							id:           tcID,
							functionName: funcName,
							functionArgs: funcArgs,
							hasFinished:  false,
						}

						tc := toolCalls[index]

						// Send delta if the argument text has already started
						if len(tc.functionArgs) > 0 {
							outputChan <- languagemodel.StreamPartToolInputDelta{
								ID:    tc.id,
								Delta: tc.functionArgs,
							}
						}

						// Check if tool call is complete (some providers send full in one chunk)
						if providerutils.IsParsableJson(tc.functionArgs) {
							outputChan <- languagemodel.StreamPartToolInputEnd{ID: tc.id}

							toolCallID := tc.id
							if toolCallID == "" {
								toolCallID = providerutils.GenerateId()
							}

							outputChan <- languagemodel.ToolCall{
								ToolCallID: toolCallID,
								ToolName:   tc.functionName,
								Input:      tc.functionArgs,
							}
							tc.hasFinished = true
						}

						continue
					}

					// Existing tool call, merge if not finished
					tc := toolCalls[index]
					if tc == nil || tc.hasFinished {
						continue
					}

					argsDelta := ""
					if fnObj, ok := tcDelta["function"].(map[string]any); ok {
						if a, ok := fnObj["arguments"].(string); ok {
							tc.functionArgs += a
							argsDelta = a
						}
					}

					// Send delta
					outputChan <- languagemodel.StreamPartToolInputDelta{
						ID:    tc.id,
						Delta: argsDelta,
					}

					// Check if tool call is complete
					if tc.functionName != "" && providerutils.IsParsableJson(tc.functionArgs) {
						outputChan <- languagemodel.StreamPartToolInputEnd{ID: tc.id}

						toolCallID := tc.id
						if toolCallID == "" {
							toolCallID = providerutils.GenerateId()
						}

						outputChan <- languagemodel.ToolCall{
							ToolCallID: toolCallID,
							ToolName:   tc.functionName,
							Input:      tc.functionArgs,
						}
						tc.hasFinished = true
					}
				}
			}
		}

		// Flush: end active blocks
		if isActiveReasoning {
			outputChan <- languagemodel.StreamPartReasoningEnd{ID: "reasoning-0"}
		}
		if isActiveText {
			outputChan <- languagemodel.StreamPartTextEnd{ID: "txt-0"}
		}

		// Flush: send incomplete tool calls
		for _, tc := range toolCalls {
			if tc != nil && !tc.hasFinished {
				outputChan <- languagemodel.StreamPartToolInputEnd{ID: tc.id}

				toolCallID := tc.id
				if toolCallID == "" {
					toolCallID = providerutils.GenerateId()
				}

				outputChan <- languagemodel.ToolCall{
					ToolCallID: toolCallID,
					ToolName:   tc.functionName,
					Input:      tc.functionArgs,
				}
			}
		}

		// Provider metadata
		providerMeta := shared.ProviderMetadata{
			providerOptionsName: map[string]any{},
		}
		if usage != nil {
			if usage.PromptCacheHitTokens != nil {
				providerMeta[providerOptionsName]["promptCacheHitTokens"] = *usage.PromptCacheHitTokens
			}
			if usage.PromptCacheMissTokens != nil {
				providerMeta[providerOptionsName]["promptCacheMissTokens"] = *usage.PromptCacheMissTokens
			}
		}

		// Finish event
		outputChan <- languagemodel.StreamPartFinish{
			FinishReason:     finishReason,
			Usage:            ConvertDeepSeekUsage(usage),
			ProviderMetadata: providerMeta,
		}
	}()

	return languagemodel.StreamResult{
		Stream:   outputChan,
		Request:  &languagemodel.StreamResultRequest{Body: body},
		Response: &languagemodel.StreamResultResponse{Headers: result.ResponseHeaders},
	}, nil
}

// toInterfaceMap converts shared.ProviderOptions (map[string]map[string]any) to
// map[string]interface{} for use with providerutils.ParseProviderOptions.
func toInterfaceMap(opts shared.ProviderOptions) map[string]interface{} {
	if opts == nil {
		return nil
	}
	result := make(map[string]interface{}, len(opts))
	for k, v := range opts {
		result[k] = v
	}
	return result
}

// convertOptionalHeaders converts map[string]*string to map[string]string,
// skipping nil values.
func convertOptionalHeaders(headers map[string]*string) map[string]string {
	if headers == nil {
		return nil
	}
	result := make(map[string]string)
	for k, v := range headers {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}

// parseDeepSeekTokenUsage attempts to parse a raw value into DeepSeekChatTokenUsage.
func parseDeepSeekTokenUsage(raw any) *DeepSeekChatTokenUsage {
	usageMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	usage := &DeepSeekChatTokenUsage{}

	if v, ok := usageMap["prompt_tokens"].(float64); ok {
		i := int(v)
		usage.PromptTokens = &i
	}
	if v, ok := usageMap["completion_tokens"].(float64); ok {
		i := int(v)
		usage.CompletionTokens = &i
	}
	if v, ok := usageMap["prompt_cache_hit_tokens"].(float64); ok {
		i := int(v)
		usage.PromptCacheHitTokens = &i
	}
	if v, ok := usageMap["prompt_cache_miss_tokens"].(float64); ok {
		i := int(v)
		usage.PromptCacheMissTokens = &i
	}
	if v, ok := usageMap["total_tokens"].(float64); ok {
		_ = v // total_tokens is available but not stored separately
	}

	if details, ok := usageMap["completion_tokens_details"].(map[string]any); ok {
		usage.CompletionTokensDetails = &struct {
			ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
		}{}
		if v, ok := details["reasoning_tokens"].(float64); ok {
			i := int(v)
			usage.CompletionTokensDetails.ReasoningTokens = &i
		}
	}

	return usage
}

// deepSeekLanguageModelOptionsSchema is the schema for parsing DeepSeek provider options.
var deepSeekLanguageModelOptionsSchema = &providerutils.Schema[DeepSeekLanguageModelOptions]{}

// Verify ChatLanguageModel implements the LanguageModel interface.
var _ languagemodel.LanguageModel = (*ChatLanguageModel)(nil)
