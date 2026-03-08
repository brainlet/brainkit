// Ported from: packages/openai/src/chat/openai-chat-language-model.ts
package openai

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

// OpenAIChatConfig holds the configuration for an OpenAI chat language model.
type OpenAIChatConfig struct {
	// Provider is the provider identifier (e.g. "openai.chat").
	Provider string

	// Headers returns the HTTP headers to include with requests.
	Headers func() map[string]string

	// URL constructs the API URL from model ID and path.
	URL func(options struct {
		ModelID string
		Path    string
	}) string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction
}

// OpenAIChatLanguageModel implements languagemodel.LanguageModel for OpenAI chat APIs.
type OpenAIChatLanguageModel struct {
	modelID string
	config  OpenAIChatConfig
}

// NewOpenAIChatLanguageModel creates a new OpenAIChatLanguageModel.
func NewOpenAIChatLanguageModel(modelID OpenAIChatModelId, config OpenAIChatConfig) *OpenAIChatLanguageModel {
	return &OpenAIChatLanguageModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *OpenAIChatLanguageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *OpenAIChatLanguageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *OpenAIChatLanguageModel) ModelID() string { return m.modelID }

// SupportedUrls returns supported URL patterns.
func (m *OpenAIChatLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return map[string][]*regexp.Regexp{
		"image/*": {regexp.MustCompile(`^https?://.*$`)},
	}, nil
}

// getArgs prepares the request arguments from CallOptions.
func (m *OpenAIChatLanguageModel) getArgs(opts languagemodel.CallOptions) (args map[string]any, warnings []shared.Warning, err error) {
	warnings = []shared.Warning{}

	// Convert ProviderOptions to map[string]interface{} for ParseProviderOptions
	providerOptsMap := providerOptionsToMap(opts.ProviderOptions)

	// Parse provider options
	openaiOptions, err := providerutils.ParseProviderOptions(
		"openai",
		providerOptsMap,
		openaiLanguageModelChatOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}
	if openaiOptions == nil {
		openaiOptions = &OpenAILanguageModelChatOptions{}
	}

	modelCapabilities := GetOpenAILanguageModelCapabilities(m.modelID)

	isReasoningModel := modelCapabilities.IsReasoningModel
	if openaiOptions.ForceReasoning != nil && *openaiOptions.ForceReasoning {
		isReasoningModel = true
	}

	if opts.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}

	// Convert messages
	systemMessageMode := modelCapabilities.SystemMessageMode
	if openaiOptions.SystemMessageMode != nil {
		systemMessageMode = *openaiOptions.SystemMessageMode
	} else if isReasoningModel {
		systemMessageMode = "developer"
	}

	result := ConvertToOpenAIChatMessages(opts.Prompt, systemMessageMode)
	warnings = append(warnings, result.Warnings...)

	strictJsonSchema := true
	if openaiOptions.StrictJSONSchema != nil {
		strictJsonSchema = *openaiOptions.StrictJSONSchema
	}

	// Build base args
	args = map[string]any{
		"model": m.modelID,
	}

	// Model specific settings
	if openaiOptions.LogitBias != nil {
		args["logit_bias"] = openaiOptions.LogitBias
	}

	// Handle logprobs (can be bool or number)
	if openaiOptions.Logprobs != nil {
		switch lp := openaiOptions.Logprobs.(type) {
		case bool:
			if lp {
				args["logprobs"] = true
				args["top_logprobs"] = float64(0)
			}
		case float64:
			args["logprobs"] = true
			args["top_logprobs"] = lp
		}
	}

	if openaiOptions.User != nil {
		args["user"] = *openaiOptions.User
	}
	if openaiOptions.ParallelToolCalls != nil {
		args["parallel_tool_calls"] = *openaiOptions.ParallelToolCalls
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
		if jsonFormat.Schema != nil {
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
	if openaiOptions.TextVerbosity != nil {
		args["verbosity"] = *openaiOptions.TextVerbosity
	}

	// OpenAI specific settings
	if openaiOptions.MaxCompletionTokens != nil {
		args["max_completion_tokens"] = *openaiOptions.MaxCompletionTokens
	}
	if openaiOptions.Store != nil {
		args["store"] = *openaiOptions.Store
	}
	if openaiOptions.Metadata != nil {
		args["metadata"] = openaiOptions.Metadata
	}
	if openaiOptions.Prediction != nil {
		args["prediction"] = openaiOptions.Prediction
	}
	if openaiOptions.ReasoningEffort != nil {
		args["reasoning_effort"] = *openaiOptions.ReasoningEffort
	}
	if openaiOptions.ServiceTier != nil {
		args["service_tier"] = *openaiOptions.ServiceTier
	}
	if openaiOptions.PromptCacheKey != nil {
		args["prompt_cache_key"] = *openaiOptions.PromptCacheKey
	}
	if openaiOptions.PromptCacheRetention != nil {
		args["prompt_cache_retention"] = *openaiOptions.PromptCacheRetention
	}
	if openaiOptions.SafetyIdentifier != nil {
		args["safety_identifier"] = *openaiOptions.SafetyIdentifier
	}

	// Messages
	args["messages"] = result.Messages

	// Remove unsupported settings for reasoning models
	if isReasoningModel {
		reasoningEffortIsNone := openaiOptions.ReasoningEffort != nil && *openaiOptions.ReasoningEffort == "none"

		if !reasoningEffortIsNone || !modelCapabilities.SupportsNonReasoningParameters {
			if _, ok := args["temperature"]; ok {
				delete(args, "temperature")
				detail := "temperature is not supported for reasoning models"
				warnings = append(warnings, shared.UnsupportedWarning{Feature: "temperature", Details: &detail})
			}
			if _, ok := args["top_p"]; ok {
				delete(args, "top_p")
				detail := "topP is not supported for reasoning models"
				warnings = append(warnings, shared.UnsupportedWarning{Feature: "topP", Details: &detail})
			}
			if _, ok := args["logprobs"]; ok {
				delete(args, "logprobs")
				warnings = append(warnings, shared.OtherWarning{Message: "logprobs is not supported for reasoning models"})
			}
		}

		if _, ok := args["frequency_penalty"]; ok {
			delete(args, "frequency_penalty")
			detail := "frequencyPenalty is not supported for reasoning models"
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "frequencyPenalty", Details: &detail})
		}
		if _, ok := args["presence_penalty"]; ok {
			delete(args, "presence_penalty")
			detail := "presencePenalty is not supported for reasoning models"
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "presencePenalty", Details: &detail})
		}
		if _, ok := args["logit_bias"]; ok {
			delete(args, "logit_bias")
			warnings = append(warnings, shared.OtherWarning{Message: "logitBias is not supported for reasoning models"})
		}
		if _, ok := args["top_logprobs"]; ok {
			delete(args, "top_logprobs")
			warnings = append(warnings, shared.OtherWarning{Message: "topLogprobs is not supported for reasoning models"})
		}

		// Reasoning models use max_completion_tokens instead of max_tokens
		if maxTokens, ok := args["max_tokens"]; ok {
			if _, hasMaxCompletion := args["max_completion_tokens"]; !hasMaxCompletion {
				args["max_completion_tokens"] = maxTokens
			}
			delete(args, "max_tokens")
		}
	} else if strings.HasPrefix(m.modelID, "gpt-4o-search-preview") ||
		strings.HasPrefix(m.modelID, "gpt-4o-mini-search-preview") {
		if _, ok := args["temperature"]; ok {
			delete(args, "temperature")
			detail := "temperature is not supported for the search preview models and has been removed."
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "temperature", Details: &detail})
		}
	}

	// Validate flex processing support
	if openaiOptions.ServiceTier != nil && *openaiOptions.ServiceTier == "flex" &&
		!modelCapabilities.SupportsFlexProcessing {
		detail := "flex processing is only available for o3, o4-mini, and gpt-5 models"
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "serviceTier", Details: &detail})
		delete(args, "service_tier")
	}

	// Validate priority processing support
	if openaiOptions.ServiceTier != nil && *openaiOptions.ServiceTier == "priority" &&
		!modelCapabilities.SupportsPriorityProcessing {
		detail := "priority processing is only available for supported models (gpt-4, gpt-5, gpt-5-mini, o3, o4-mini) and requires Enterprise access. gpt-5-nano is not supported"
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "serviceTier", Details: &detail})
		delete(args, "service_tier")
	}

	// Prepare tools
	toolResult := PrepareChatTools(opts.Tools, opts.ToolChoice)

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
func (m *OpenAIChatLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	args, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[openaiChatResponse]{
		URL: m.config.URL(struct {
			ModelID string
			Path    string
		}{ModelID: m.modelID, Path: "/chat/completions"}),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers)),
		Body:                      args,
		FailedResponseHandler:     openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(openaiChatResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := result.Value
	if len(response.Choices) == 0 {
		return languagemodel.GenerateResult{}, fmt.Errorf("openai chat: no choices in response")
	}
	choice := response.Choices[0]
	content := []languagemodel.Content{}

	// Text content
	if choice.Message.Content != nil && len(*choice.Message.Content) > 0 {
		content = append(content, languagemodel.Text{Text: *choice.Message.Content})
	}

	// Tool calls
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

	// Annotations/citations
	for _, annotation := range choice.Message.Annotations {
		title := annotation.URLCitation.Title
		content = append(content, languagemodel.SourceURL{
			ID:    providerutils.GenerateId(),
			URL:   annotation.URLCitation.URL,
			Title: &title,
		})
	}

	// Provider metadata
	providerMetadata := shared.ProviderMetadata{
		"openai": map[string]any{},
	}

	if response.Usage != nil && response.Usage.CompletionTokensDetails != nil {
		details := response.Usage.CompletionTokensDetails
		if details.AcceptedPredictionTokens != nil {
			providerMetadata["openai"]["acceptedPredictionTokens"] = *details.AcceptedPredictionTokens
		}
		if details.RejectedPredictionTokens != nil {
			providerMetadata["openai"]["rejectedPredictionTokens"] = *details.RejectedPredictionTokens
		}
	}

	if choice.Logprobs != nil && choice.Logprobs.Content != nil {
		providerMetadata["openai"]["logprobs"] = choice.Logprobs.Content
	}

	// Finish reason
	finishReason := languagemodel.FinishReason{
		Unified: MapOpenAIChatFinishReason(choice.FinishReason),
	}
	if choice.FinishReason != nil {
		finishReason.Raw = choice.FinishReason
	}

	// Response metadata
	respMeta := getChatResponseMetadata(response.ID, response.Model, response.Created)

	return languagemodel.GenerateResult{
		Content:          content,
		FinishReason:     finishReason,
		Usage:            ConvertOpenAIChatUsage(response.Usage),
		ProviderMetadata: providerMetadata,
		Request:          &languagemodel.GenerateResultRequest{Body: args},
		Response: &languagemodel.GenerateResultResponse{
			ResponseMetadata: respMeta,
			Headers:          result.ResponseHeaders,
			Body:             result.RawValue,
		},
		Warnings: warnings,
	}, nil
}

// DoStream implements languagemodel.LanguageModel.DoStream for streaming.
func (m *OpenAIChatLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
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
		URL: m.config.URL(struct {
			ModelID string
			Path    string
		}{ModelID: m.modelID, Path: "/chat/completions"}),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(openaiChatChunkSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	inputChan := result.Value
	outputChan := make(chan languagemodel.StreamPart)
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
		var usage *OpenAIChatUsage
		metadataExtracted := false
		isActiveText := false

		providerMeta := shared.ProviderMetadata{
			"openai": map[string]any{},
		}

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
				outputChan <- languagemodel.StreamPartError{Error: errVal}
				continue
			}

			// Extract and emit response metadata once
			if !metadataExtracted {
				meta := chatResponseMetadataFromMap(chunkMap)
				if meta.ID != nil || meta.ModelID != nil || meta.Timestamp != nil {
					metadataExtracted = true
					outputChan <- languagemodel.StreamPartResponseMetadata{
						ResponseMetadata: meta,
					}
				}
			}

			// Handle usage
			if usageRaw, ok := chunkMap["usage"]; ok && usageRaw != nil {
				usage = parseChatUsage(usageRaw)

				if usage != nil && usage.CompletionTokensDetails != nil {
					if usage.CompletionTokensDetails.AcceptedPredictionTokens != nil {
						providerMeta["openai"]["acceptedPredictionTokens"] = *usage.CompletionTokensDetails.AcceptedPredictionTokens
					}
					if usage.CompletionTokensDetails.RejectedPredictionTokens != nil {
						providerMeta["openai"]["rejectedPredictionTokens"] = *usage.CompletionTokensDetails.RejectedPredictionTokens
					}
				}
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
					Unified: MapOpenAIChatFinishReason(&frStr),
					Raw:     &frStr,
				}
			}

			// Check logprobs
			if logprobsRaw, ok := choiceMap["logprobs"]; ok && logprobsRaw != nil {
				if logprobsMap, ok := logprobsRaw.(map[string]any); ok {
					if contentRaw, ok := logprobsMap["content"]; ok && contentRaw != nil {
						providerMeta["openai"]["logprobs"] = contentRaw
					}
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

			// Text content
			if contentVal, ok := delta["content"]; ok && contentVal != nil {
				if contentStr, ok := contentVal.(string); ok {
					if !isActiveText {
						outputChan <- languagemodel.StreamPartTextStart{ID: "0"}
						isActiveText = true
					}
					outputChan <- languagemodel.StreamPartTextDelta{
						ID:    "0",
						Delta: contentStr,
					}
				}
			}

			// Tool calls
			if toolCallsRaw, ok := delta["tool_calls"]; ok && toolCallsRaw != nil {
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
						// Type check
						if tcType, ok := tcDelta["type"]; ok && tcType != nil {
							if typeStr, ok := tcType.(string); ok && typeStr != "function" {
								outputChan <- languagemodel.StreamPartError{
									Error: errors.NewInvalidResponseDataError(tcDelta, "Expected 'function' type."),
								}
								continue
							}
						}

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

			// Annotations/citations
			if annotationsRaw, ok := delta["annotations"]; ok && annotationsRaw != nil {
				if annotations, ok := annotationsRaw.([]any); ok {
					for _, annRaw := range annotations {
						if ann, ok := annRaw.(map[string]any); ok {
							if urlCitation, ok := ann["url_citation"].(map[string]any); ok {
								url, _ := urlCitation["url"].(string)
								title, _ := urlCitation["title"].(string)
								outputChan <- languagemodel.SourceURL{
									ID:    providerutils.GenerateId(),
									URL:   url,
									Title: &title,
								}
							}
						}
					}
				}
			}
		}

		// Flush: end active text block
		if isActiveText {
			outputChan <- languagemodel.StreamPartTextEnd{ID: "0"}
		}

		// Flush: finish event
		outputChan <- languagemodel.StreamPartFinish{
			FinishReason:     finishReason,
			Usage:            ConvertOpenAIChatUsage(usage),
			ProviderMetadata: providerMeta,
		}
	}()

	return languagemodel.StreamResult{
		Stream:   outputChan,
		Request:  &languagemodel.StreamResultRequest{Body: body},
		Response: &languagemodel.StreamResultResponse{Headers: result.ResponseHeaders},
	}, nil
}

// --- Helper functions ---

// chatResponseMetadataFromMap extracts response metadata from a streaming chunk map.
func chatResponseMetadataFromMap(m map[string]any) languagemodel.ResponseMetadata {
	rm := languagemodel.ResponseMetadata{}
	if id, ok := m["id"].(string); ok {
		rm.ID = &id
	}
	if model, ok := m["model"].(string); ok {
		rm.ModelID = &model
	}
	if created, ok := m["created"].(float64); ok {
		t := getChatResponseMetadata(nil, nil, &created)
		rm.Timestamp = t.Timestamp
	}
	return rm
}

// parseChatUsage attempts to parse a raw value into OpenAIChatUsage.
func parseChatUsage(raw any) *OpenAIChatUsage {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var usage OpenAIChatUsage
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil
	}
	return &usage
}

// Verify OpenAIChatLanguageModel implements the LanguageModel interface.
var _ languagemodel.LanguageModel = (*OpenAIChatLanguageModel)(nil)
