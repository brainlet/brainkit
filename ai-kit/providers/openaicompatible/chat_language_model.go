// Ported from: packages/openai-compatible/src/chat/openai-compatible-chat-language-model.ts
package openaicompatible

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ChatConfig holds the configuration for an OpenAI-compatible chat language model.
type ChatConfig struct {
	// Provider is the provider identifier (e.g. "openai.chat").
	Provider string

	// URL constructs the API URL from a path (e.g. "/chat/completions").
	URL func(path string) string

	// Headers returns the HTTP headers to include with requests.
	Headers func() map[string]string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction

	// IncludeUsage indicates whether to include stream_options with include_usage.
	IncludeUsage *bool

	// SupportsStructuredOutputs indicates whether the model supports structured outputs.
	SupportsStructuredOutputs *bool

	// TransformRequestBody optionally transforms the request body before sending.
	TransformRequestBody func(map[string]any) map[string]any

	// MetadataExtractor is an optional extractor for provider-specific metadata.
	MetadataExtractor MetadataExtractor

	// ErrorStructure is the error handling structure. If nil, defaults are used.
	ErrorStructure *ProviderErrorStructure[ErrorData]

	// SupportedURLs returns the supported URL patterns by media type.
	SupportedURLs func() (map[string][]*regexp.Regexp, error)
}

// ChatLanguageModel implements languagemodel.LanguageModel for OpenAI-compatible APIs.
type ChatLanguageModel struct {
	modelID               string
	config                ChatConfig
	failedResponseHandler providerutils.ResponseHandler[error]
	chunkSchema           *providerutils.Schema[any]
	supportsStructured    bool
}

// NewChatLanguageModel creates a new ChatLanguageModel.
func NewChatLanguageModel(modelID string, config ChatConfig) *ChatLanguageModel {
	m := &ChatLanguageModel{
		modelID: modelID,
		config:  config,
	}

	if config.SupportsStructuredOutputs != nil {
		m.supportsStructured = *config.SupportsStructuredOutputs
	}

	// Initialize error handling
	errorStructure := config.ErrorStructure
	if errorStructure == nil {
		errorStructure = &DefaultErrorStructure
	}

	m.chunkSchema = createChatChunkSchema(errorStructure.ErrorSchema)

	// Wrap the typed error handler to satisfy ResponseHandler[error]
	typedHandler := providerutils.CreateJsonErrorResponseHandler(
		errorStructure.ErrorSchema,
		errorStructure.ErrorToMessage,
		errorStructure.IsRetryable,
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

// SupportedUrls returns supported URL patterns.
func (m *ChatLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	if m.config.SupportedURLs != nil {
		return m.config.SupportedURLs()
	}
	return map[string][]*regexp.Regexp{}, nil
}

// providerOptionsName extracts the provider name from the provider string
// (e.g. "openai.chat" -> "openai").
func (m *ChatLanguageModel) providerOptionsName() string {
	parts := strings.SplitN(m.config.Provider, ".", 2)
	return strings.TrimSpace(parts[0])
}

// transformRequestBody applies the optional body transform.
func (m *ChatLanguageModel) transformRequestBody(body map[string]any) map[string]any {
	if m.config.TransformRequestBody != nil {
		return m.config.TransformRequestBody(body)
	}
	return body
}

// isChatOptionKey returns true if the key is a known ChatOptions field.
func isChatOptionKey(key string) bool {
	switch key {
	case "user", "reasoningEffort", "textVerbosity", "strictJsonSchema":
		return true
	default:
		return false
	}
}

// mergeChatOptions merges non-nil fields from src into dst.
func mergeChatOptions(dst, src *ChatOptions) {
	if src.User != nil {
		dst.User = src.User
	}
	if src.ReasoningEffort != nil {
		dst.ReasoningEffort = src.ReasoningEffort
	}
	if src.TextVerbosity != nil {
		dst.TextVerbosity = src.TextVerbosity
	}
	if src.StrictJSONSchema != nil {
		dst.StrictJSONSchema = src.StrictJSONSchema
	}
}

// getArgs prepares the request arguments from CallOptions.
func (m *ChatLanguageModel) getArgs(opts languagemodel.CallOptions) (args map[string]any, warnings []shared.Warning, err error) {
	warnings = []shared.Warning{}

	// Convert ProviderOptions to map[string]interface{} for ParseProviderOptions
	providerOptsMap := toInterfaceMap(opts.ProviderOptions)

	// Parse provider options - check for deprecated 'openai-compatible' key
	deprecatedOptions, err := providerutils.ParseProviderOptions(
		"openai-compatible",
		providerOptsMap,
		ChatOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}

	if deprecatedOptions != nil {
		warnings = append(warnings, shared.OtherWarning{
			Message: "The 'openai-compatible' key in providerOptions is deprecated. Use 'openaiCompatible' instead.",
		})
	}

	// Parse openaiCompatible options
	compatOpts, err := providerutils.ParseProviderOptions(
		"openaiCompatible",
		providerOptsMap,
		ChatOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}

	// Parse provider-specific options
	providerSpecificOpts, err := providerutils.ParseProviderOptions(
		m.providerOptionsName(),
		providerOptsMap,
		ChatOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}

	// Merge options: deprecated < openaiCompatible < providerSpecific
	mergedOptions := &ChatOptions{}
	if deprecatedOptions != nil {
		mergeChatOptions(mergedOptions, deprecatedOptions)
	}
	if compatOpts != nil {
		mergeChatOptions(mergedOptions, compatOpts)
	}
	if providerSpecificOpts != nil {
		mergeChatOptions(mergedOptions, providerSpecificOpts)
	}

	strictJsonSchema := true
	if mergedOptions.StrictJSONSchema != nil {
		strictJsonSchema = *mergedOptions.StrictJSONSchema
	}

	if opts.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}

	if jsonFormat, ok := opts.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		if jsonFormat.Schema != nil && !m.supportsStructured {
			detail := "JSON response format schema is only supported with structuredOutputs"
			warnings = append(warnings, shared.UnsupportedWarning{
				Feature: "responseFormat",
				Details: &detail,
			})
		}
	}

	// Prepare tools
	toolResult := PrepareTools(opts.Tools, opts.ToolChoice)

	// Build request body
	args = map[string]any{
		"model": m.modelID,
	}

	// Model specific settings
	if mergedOptions.User != nil {
		args["user"] = *mergedOptions.User
	}

	// Standardized settings
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
	if jsonFormat, ok := opts.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		if m.supportsStructured && jsonFormat.Schema != nil {
			name := "response"
			if jsonFormat.Name != nil {
				name = *jsonFormat.Name
			}
			jsonSchemaObj := map[string]any{
				"schema": jsonFormat.Schema,
				"strict": strictJsonSchema,
				"name":   name,
			}
			if jsonFormat.Description != nil {
				jsonSchemaObj["description"] = *jsonFormat.Description
			}
			args["response_format"] = map[string]any{
				"type":        "json_schema",
				"json_schema": jsonSchemaObj,
			}
		} else {
			args["response_format"] = map[string]any{
				"type": "json_object",
			}
		}
	}

	if len(opts.StopSequences) > 0 {
		args["stop"] = opts.StopSequences
	}
	if opts.Seed != nil {
		args["seed"] = *opts.Seed
	}

	// Pass through extra provider-specific options not in the ChatOptions schema
	if opts.ProviderOptions != nil {
		providerOptsName := m.providerOptionsName()
		if providerSpecific, ok := opts.ProviderOptions[providerOptsName]; ok {
			for key, val := range providerSpecific {
				if !isChatOptionKey(key) {
					args[key] = val
				}
			}
		}
	}

	if mergedOptions.ReasoningEffort != nil {
		args["reasoning_effort"] = *mergedOptions.ReasoningEffort
	}
	if mergedOptions.TextVerbosity != nil {
		args["verbosity"] = *mergedOptions.TextVerbosity
	}

	// Messages
	args["messages"] = ConvertToChatMessages(opts.Prompt)

	// Tools
	if toolResult.Tools != nil {
		args["tools"] = toolResult.Tools
	}
	if toolResult.ToolChoice != nil {
		args["tool_choice"] = toolResult.ToolChoice
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

	transformedBody := m.transformRequestBody(args)
	bodyJSON, err := json.Marshal(transformedBody)
	if err != nil {
		return languagemodel.GenerateResult{}, fmt.Errorf("failed to marshal request body: %w", err)
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[chatCompletionResponse]{
		URL:                       m.config.URL("/chat/completions"),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      transformedBody,
		FailedResponseHandler:     m.failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(chatCompletionResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	responseBody := result.Value
	choice := responseBody.Choices[0]
	content := []languagemodel.Content{}

	// Text content
	if choice.Message.Content != nil && len(*choice.Message.Content) > 0 {
		content = append(content, languagemodel.Text{Text: *choice.Message.Content})
	}

	// Reasoning content
	reasoning := ""
	if choice.Message.ReasoningContent != nil {
		reasoning = *choice.Message.ReasoningContent
	} else if choice.Message.Reasoning != nil {
		reasoning = *choice.Message.Reasoning
	}
	if reasoning != "" {
		content = append(content, languagemodel.Reasoning{Text: reasoning})
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

			tc := languagemodel.ToolCall{
				ToolCallID: id,
				ToolName:   toolCall.Function.Name,
				Input:      toolCall.Function.Arguments,
			}

			// Handle thought signature
			if toolCall.ExtraContent != nil && toolCall.ExtraContent.Google != nil &&
				toolCall.ExtraContent.Google.ThoughtSignature != nil {
				tc.ProviderMetadata = shared.ProviderMetadata{
					m.providerOptionsName(): map[string]any{
						"thoughtSignature": *toolCall.ExtraContent.Google.ThoughtSignature,
					},
				}
			}

			content = append(content, tc)
		}
	}

	// Provider metadata
	providerMetadata := shared.ProviderMetadata{
		m.providerOptionsName(): map[string]any{},
	}

	// Extract metadata from response
	if m.config.MetadataExtractor != nil {
		extracted, extractErr := m.config.MetadataExtractor.ExtractMetadata(result.RawValue)
		if extractErr == nil && extracted != nil {
			for k, v := range *extracted {
				providerMetadata[k] = v
			}
		}
	}

	// Completion token details
	if responseBody.Usage != nil && responseBody.Usage.CompletionTokensDetails != nil {
		details := responseBody.Usage.CompletionTokensDetails
		providerName := m.providerOptionsName()
		if details.AcceptedPredictionTokens != nil {
			providerMetadata[providerName]["acceptedPredictionTokens"] = *details.AcceptedPredictionTokens
		}
		if details.RejectedPredictionTokens != nil {
			providerMetadata[providerName]["rejectedPredictionTokens"] = *details.RejectedPredictionTokens
		}
	}

	// Finish reason
	finishReason := languagemodel.FinishReason{
		Unified: MapFinishReason(choice.FinishReason),
	}
	if choice.FinishReason != nil {
		finishReason.Raw = choice.FinishReason
	}

	// Response metadata
	respMeta := getResponseMetadataFromResponse(responseBody)

	return languagemodel.GenerateResult{
		Content:          content,
		FinishReason:     finishReason,
		Usage:            ConvertChatUsage(responseBody.Usage),
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

	// Only include stream_options when in strict compatibility mode
	if m.config.IncludeUsage != nil && *m.config.IncludeUsage {
		body["stream_options"] = map[string]any{
			"include_usage": true,
		}
	}

	transformedBody := m.transformRequestBody(body)

	var metadataExtractor StreamExtractor
	if m.config.MetadataExtractor != nil {
		metadataExtractor = m.config.MetadataExtractor.CreateStreamExtractor()
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[any]]{
		URL:                       m.config.URL("/chat/completions"),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      transformedBody,
		FailedResponseHandler:     m.failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(m.chunkSchema),
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
			id               string
			functionName     string
			functionArgs     string
			hasFinished      bool
			thoughtSignature *string
		}

		var toolCalls []*toolCallState
		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
		}
		var usage *OpenAICompatibleTokenUsage
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

			if metadataExtractor != nil {
				metadataExtractor.ProcessChunk(chunk.RawValue)
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
				usage = parseTokenUsage(usageRaw)
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
					Unified: MapFinishReason(&frStr),
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
			reasoningContent := ""
			if rc, ok := delta["reasoning_content"]; ok && rc != nil {
				if s, ok := rc.(string); ok {
					reasoningContent = s
				}
			}
			if reasoningContent == "" {
				if rc, ok := delta["reasoning"]; ok && rc != nil {
					if s, ok := rc.(string); ok {
						reasoningContent = s
					}
				}
			}

			if reasoningContent != "" {
				if !isActiveReasoning {
					outputChan <- languagemodel.StreamPartReasoningStart{ID: "reasoning-0"}
					isActiveReasoning = true
				}
				outputChan <- languagemodel.StreamPartReasoningDelta{
					ID:    "reasoning-0",
					Delta: reasoningContent,
				}
			}

			// Text content
			if contentVal, ok := delta["content"]; ok && contentVal != nil {
				if contentStr, ok := contentVal.(string); ok && contentStr != "" {
					// End active reasoning block before text starts
					if isActiveReasoning {
						outputChan <- languagemodel.StreamPartReasoningEnd{ID: "reasoning-0"}
						isActiveReasoning = false
					}

					if !isActiveText {
						outputChan <- languagemodel.StreamPartTextStart{ID: "txt-0"}
						isActiveText = true
					}

					outputChan <- languagemodel.StreamPartTextDelta{
						ID:    "txt-0",
						Delta: contentStr,
					}
				}
			}

			// Tool calls
			if toolCallsRaw, ok := delta["tool_calls"]; ok && toolCallsRaw != nil {
				// End active reasoning block before tool calls start
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

						// Extract thought signature
						var thoughtSig *string
						if ec, ok := tcDelta["extra_content"].(map[string]any); ok {
							if google, ok := ec["google"].(map[string]any); ok {
								if sig, ok := google["thought_signature"].(string); ok {
									thoughtSig = &sig
								}
							}
						}

						// Extend toolCalls slice to fit the index
						for len(toolCalls) <= index {
							toolCalls = append(toolCalls, nil)
						}

						toolCalls[index] = &toolCallState{
							id:               tcID,
							functionName:     funcName,
							functionArgs:     funcArgs,
							hasFinished:      false,
							thoughtSignature: thoughtSig,
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

							tcContent := languagemodel.ToolCall{
								ToolCallID: toolCallID,
								ToolName:   tc.functionName,
								Input:      tc.functionArgs,
							}
							if tc.thoughtSignature != nil {
								tcContent.ProviderMetadata = shared.ProviderMetadata{
									providerOptionsName: map[string]any{
										"thoughtSignature": *tc.thoughtSignature,
									},
								}
							}
							outputChan <- tcContent
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

						tcContent := languagemodel.ToolCall{
							ToolCallID: toolCallID,
							ToolName:   tc.functionName,
							Input:      tc.functionArgs,
						}
						if tc.thoughtSignature != nil {
							tcContent.ProviderMetadata = shared.ProviderMetadata{
								providerOptionsName: map[string]any{
									"thoughtSignature": *tc.thoughtSignature,
								},
							}
						}
						outputChan <- tcContent
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

				tcContent := languagemodel.ToolCall{
					ToolCallID: toolCallID,
					ToolName:   tc.functionName,
					Input:      tc.functionArgs,
				}
				if tc.thoughtSignature != nil {
					tcContent.ProviderMetadata = shared.ProviderMetadata{
						providerOptionsName: map[string]any{
							"thoughtSignature": *tc.thoughtSignature,
						},
					}
				}
				outputChan <- tcContent
			}
		}

		// Provider metadata
		providerMeta := shared.ProviderMetadata{
			providerOptionsName: map[string]any{},
		}
		if metadataExtractor != nil {
			extracted := metadataExtractor.BuildMetadata()
			if extracted != nil {
				for k, v := range *extracted {
					providerMeta[k] = v
				}
			}
		}
		if usage != nil && usage.CompletionTokensDetails != nil {
			if usage.CompletionTokensDetails.AcceptedPredictionTokens != nil {
				providerMeta[providerOptionsName]["acceptedPredictionTokens"] = *usage.CompletionTokensDetails.AcceptedPredictionTokens
			}
			if usage.CompletionTokensDetails.RejectedPredictionTokens != nil {
				providerMeta[providerOptionsName]["rejectedPredictionTokens"] = *usage.CompletionTokensDetails.RejectedPredictionTokens
			}
		}

		// Finish event
		outputChan <- languagemodel.StreamPartFinish{
			FinishReason:     finishReason,
			Usage:            ConvertChatUsage(usage),
			ProviderMetadata: providerMeta,
		}
	}()

	return languagemodel.StreamResult{
		Stream:   outputChan,
		Request:  &languagemodel.StreamResultRequest{Body: transformedBody},
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

// --- Response schemas (Go structs replacing Zod schemas) ---

// chatCompletionResponse represents the non-streaming API response.
type chatCompletionResponse struct {
	ID      *string                     `json:"id,omitempty"`
	Created *float64                    `json:"created,omitempty"`
	Model   *string                     `json:"model,omitempty"`
	Choices []chatCompletionChoice      `json:"choices"`
	Usage   *OpenAICompatibleTokenUsage `json:"usage,omitempty"`
}

// chatCompletionChoice represents a choice in the chat completion response.
type chatCompletionChoice struct {
	Message      chatCompletionMessage `json:"message"`
	FinishReason *string               `json:"finish_reason,omitempty"`
}

// chatCompletionMessage represents a message in a chat completion choice.
type chatCompletionMessage struct {
	Role             *string                          `json:"role,omitempty"`
	Content          *string                          `json:"content,omitempty"`
	ReasoningContent *string                          `json:"reasoning_content,omitempty"`
	Reasoning        *string                          `json:"reasoning,omitempty"`
	ToolCalls        []chatCompletionMessageToolCall   `json:"tool_calls,omitempty"`
}

// chatCompletionMessageToolCall represents a tool call in a message.
type chatCompletionMessageToolCall struct {
	ID           *string                                `json:"id,omitempty"`
	Function     chatCompletionMessageToolCallFunction  `json:"function"`
	ExtraContent *chatCompletionExtraContent             `json:"extra_content,omitempty"`
}

// chatCompletionMessageToolCallFunction represents the function part of a tool call.
type chatCompletionMessageToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// chatCompletionExtraContent supports provider-specific extensions like Google thought signatures.
type chatCompletionExtraContent struct {
	Google *chatCompletionExtraContentGoogle `json:"google,omitempty"`
}

// chatCompletionExtraContentGoogle holds Google-specific extra content.
type chatCompletionExtraContentGoogle struct {
	ThoughtSignature *string `json:"thought_signature,omitempty"`
}

// chatCompletionResponseSchema is the schema for non-streaming responses.
// Go structs with json tags replace the Zod schema.
var chatCompletionResponseSchema = &providerutils.Schema[chatCompletionResponse]{}

// createChatChunkSchema creates a schema for streaming chunks.
// In TS this uses z.union([chunkBaseSchema, errorSchema]).
// In Go, we parse into map[string]any since the chunk can be either a regular chunk or an error.
func createChatChunkSchema(_ *providerutils.Schema[ErrorData]) *providerutils.Schema[any] {
	// Returns a permissive schema that parses into map[string]any.
	// The transform logic in DoStream handles both regular chunks and error chunks.
	return &providerutils.Schema[any]{}
}

// getResponseMetadataFromResponse converts a chatCompletionResponse to languagemodel.ResponseMetadata.
func getResponseMetadataFromResponse(resp chatCompletionResponse) languagemodel.ResponseMetadata {
	var ts *time.Time
	if resp.Created != nil {
		t := time.Unix(int64(*resp.Created), 0)
		ts = &t
	}
	return languagemodel.ResponseMetadata{
		ID:        resp.ID,
		Timestamp: ts,
		ModelID:   resp.Model,
	}
}

// getResponseMetadataFromMap converts a streaming chunk map to languagemodel.ResponseMetadata.
func getResponseMetadataFromMap(m map[string]any) languagemodel.ResponseMetadata {
	rm := languagemodel.ResponseMetadata{}
	if id, ok := m["id"].(string); ok {
		rm.ID = &id
	}
	if model, ok := m["model"].(string); ok {
		rm.ModelID = &model
	}
	if created, ok := m["created"].(float64); ok {
		t := time.Unix(int64(created), 0)
		rm.Timestamp = &t
	}
	return rm
}

// parseTokenUsage attempts to parse a raw value into OpenAICompatibleTokenUsage.
func parseTokenUsage(raw any) *OpenAICompatibleTokenUsage {
	usageMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	usage := &OpenAICompatibleTokenUsage{}

	if v, ok := usageMap["prompt_tokens"].(float64); ok {
		i := int(v)
		usage.PromptTokens = &i
	}
	if v, ok := usageMap["completion_tokens"].(float64); ok {
		i := int(v)
		usage.CompletionTokens = &i
	}
	if v, ok := usageMap["total_tokens"].(float64); ok {
		i := int(v)
		usage.TotalTokens = &i
	}

	if details, ok := usageMap["prompt_tokens_details"].(map[string]any); ok {
		usage.PromptTokensDetails = &struct {
			CachedTokens *int `json:"cached_tokens,omitempty"`
		}{}
		if v, ok := details["cached_tokens"].(float64); ok {
			i := int(v)
			usage.PromptTokensDetails.CachedTokens = &i
		}
	}

	if details, ok := usageMap["completion_tokens_details"].(map[string]any); ok {
		usage.CompletionTokensDetails = &struct {
			ReasoningTokens          *int `json:"reasoning_tokens,omitempty"`
			AcceptedPredictionTokens *int `json:"accepted_prediction_tokens,omitempty"`
			RejectedPredictionTokens *int `json:"rejected_prediction_tokens,omitempty"`
		}{}
		if v, ok := details["reasoning_tokens"].(float64); ok {
			i := int(v)
			usage.CompletionTokensDetails.ReasoningTokens = &i
		}
		if v, ok := details["accepted_prediction_tokens"].(float64); ok {
			i := int(v)
			usage.CompletionTokensDetails.AcceptedPredictionTokens = &i
		}
		if v, ok := details["rejected_prediction_tokens"].(float64); ok {
			i := int(v)
			usage.CompletionTokensDetails.RejectedPredictionTokens = &i
		}
	}

	return usage
}

// Verify ChatLanguageModel implements the LanguageModel interface.
var _ languagemodel.LanguageModel = (*ChatLanguageModel)(nil)
