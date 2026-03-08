// Ported from: packages/groq/src/groq-chat-language-model.ts
package groq

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GroqChatConfig holds the configuration for a Groq chat language model.
type GroqChatConfig struct {
	// Provider is the provider identifier (e.g. "groq.chat").
	Provider string

	// Headers returns the HTTP headers to include with requests.
	Headers func() map[string]string

	// URL constructs the API URL from modelId and path.
	URL func(modelId string, path string) string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction
}

// GroqChatLanguageModel implements languagemodel.LanguageModel for Groq APIs.
type GroqChatLanguageModel struct {
	modelID string
	config  GroqChatConfig
}

// NewGroqChatLanguageModel creates a new GroqChatLanguageModel.
func NewGroqChatLanguageModel(modelID GroqChatModelId, config GroqChatConfig) *GroqChatLanguageModel {
	return &GroqChatLanguageModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *GroqChatLanguageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *GroqChatLanguageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *GroqChatLanguageModel) ModelID() string { return m.modelID }

// SupportedUrls returns supported URL patterns.
func (m *GroqChatLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return map[string][]*regexp.Regexp{
		"image/*": {regexp.MustCompile(`^https?://.*$`)},
	}, nil
}

// getArgs prepares the request arguments from CallOptions.
func (m *GroqChatLanguageModel) getArgs(opts languagemodel.CallOptions) (args map[string]any, warnings []shared.Warning, err error) {
	warnings = []shared.Warning{}

	// Convert ProviderOptions to map[string]interface{} for ParseProviderOptions
	providerOptsMap := toInterfaceMap(opts.ProviderOptions)

	groqOptions, err := providerutils.ParseProviderOptions(
		"groq",
		providerOptsMap,
		GroqLanguageModelOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}

	structuredOutputs := true
	if groqOptions != nil && groqOptions.StructuredOutputs != nil {
		structuredOutputs = *groqOptions.StructuredOutputs
	}

	strictJsonSchema := true
	if groqOptions != nil && groqOptions.StrictJSONSchema != nil {
		strictJsonSchema = *groqOptions.StrictJSONSchema
	}

	if opts.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}

	if jsonFormat, ok := opts.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		if jsonFormat.Schema != nil && !structuredOutputs {
			detail := "JSON response format schema is only supported with structuredOutputs"
			warnings = append(warnings, shared.UnsupportedWarning{
				Feature: "responseFormat",
				Details: &detail,
			})
		}
	}

	// Prepare tools
	toolResult := PrepareTools(opts.Tools, opts.ToolChoice, m.modelID)

	// Build request body
	args = map[string]any{
		"model": m.modelID,
	}

	// Model specific settings
	if groqOptions != nil && groqOptions.User != nil {
		args["user"] = *groqOptions.User
	}
	if groqOptions != nil && groqOptions.ParallelToolCalls != nil {
		args["parallel_tool_calls"] = *groqOptions.ParallelToolCalls
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
	if len(opts.StopSequences) > 0 {
		args["stop"] = opts.StopSequences
	}
	if opts.Seed != nil {
		args["seed"] = *opts.Seed
	}

	// Response format
	if jsonFormat, ok := opts.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		if structuredOutputs && jsonFormat.Schema != nil {
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

	// Provider options
	if groqOptions != nil && groqOptions.ReasoningFormat != nil {
		args["reasoning_format"] = *groqOptions.ReasoningFormat
	}
	if groqOptions != nil && groqOptions.ReasoningEffort != nil {
		args["reasoning_effort"] = *groqOptions.ReasoningEffort
	}
	if groqOptions != nil && groqOptions.ServiceTier != nil {
		args["service_tier"] = *groqOptions.ServiceTier
	}

	// Messages
	args["messages"] = ConvertToGroqChatMessages(opts.Prompt)

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
func (m *GroqChatLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	args, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	bodyJSON, err := json.Marshal(args)
	if err != nil {
		return languagemodel.GenerateResult{}, fmt.Errorf("failed to marshal request body: %w", err)
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[groqChatCompletionResponse]{
		URL:                       m.config.URL(m.modelID, "/chat/completions"),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      args,
		FailedResponseHandler:     groqFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(groqChatCompletionResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := result.Value
	choice := response.Choices[0]
	content := []languagemodel.Content{}

	// Text content
	if choice.Message.Content != nil && len(*choice.Message.Content) > 0 {
		content = append(content, languagemodel.Text{Text: *choice.Message.Content})
	}

	// Reasoning
	if choice.Message.Reasoning != nil && len(*choice.Message.Reasoning) > 0 {
		content = append(content, languagemodel.Reasoning{Text: *choice.Message.Reasoning})
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

	// Finish reason
	finishReason := languagemodel.FinishReason{
		Unified: MapGroqFinishReason(choice.FinishReason),
	}
	if choice.FinishReason != nil {
		finishReason.Raw = choice.FinishReason
	}

	// Response metadata
	respMeta := getResponseMetadata(response.ID, response.Model, response.Created)

	return languagemodel.GenerateResult{
		Content:      content,
		FinishReason: finishReason,
		Usage:        ConvertGroqUsage(response.Usage),
		Request:      &languagemodel.GenerateResultRequest{Body: string(bodyJSON)},
		Response: &languagemodel.GenerateResultResponse{
			ResponseMetadata: respMeta,
			Headers:          result.ResponseHeaders,
			Body:             result.RawValue,
		},
		Warnings: warnings,
	}, nil
}

// DoStream implements languagemodel.LanguageModel.DoStream for streaming.
func (m *GroqChatLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	args, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	body := make(map[string]any)
	for k, v := range args {
		body[k] = v
	}
	body["stream"] = true

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[any]]{
		URL:                       m.config.URL(m.modelID, "/chat/completions"),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     groqFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(groqChatChunkSchema),
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
		var usage *GroqTokenUsage
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
				outputChan <- languagemodel.StreamPartError{Error: errVal}
				continue
			}

			if isFirstChunk {
				isFirstChunk = false
				respMeta := getResponseMetadataFromMap(chunkMap)
				outputChan <- languagemodel.StreamPartResponseMetadata{
					ResponseMetadata: respMeta,
				}
			}

			// Handle x_groq usage
			if xGroq, ok := chunkMap["x_groq"].(map[string]any); ok && xGroq != nil {
				if usageRaw, ok := xGroq["usage"]; ok && usageRaw != nil {
					usage = parseGroqTokenUsage(usageRaw)
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
					Unified: MapGroqFinishReason(&frStr),
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

			// Reasoning content
			if reasoning, ok := delta["reasoning"].(string); ok && reasoning != "" {
				if !isActiveReasoning {
					outputChan <- languagemodel.StreamPartReasoningStart{ID: "reasoning-0"}
					isActiveReasoning = true
				}
				outputChan <- languagemodel.StreamPartReasoningDelta{
					ID:    "reasoning-0",
					Delta: reasoning,
				}
			}

			// Text content
			if contentVal, ok := delta["content"].(string); ok && contentVal != "" {
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
					Delta: contentVal,
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
						tcType, _ := tcDelta["type"].(string)
						if tcType != "" && tcType != "function" {
							outputChan <- languagemodel.StreamPartError{
								Error: errors.NewInvalidResponseDataError(tcDelta, "Expected 'function' type."),
							}
							continue
						}

						tcID, _ := tcDelta["id"].(string)
						if tcID == "" {
							outputChan <- languagemodel.StreamPartError{
								Error: errors.NewInvalidResponseDataError(tcDelta, "Expected 'id' to be a string."),
							}
							continue
						}

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

		// Finish event
		outputChan <- languagemodel.StreamPartFinish{
			FinishReason: finishReason,
			Usage:        ConvertGroqUsage(usage),
		}
	}()

	return languagemodel.StreamResult{
		Stream:   outputChan,
		Request:  &languagemodel.StreamResultRequest{Body: body},
		Response: &languagemodel.StreamResultResponse{Headers: result.ResponseHeaders},
	}, nil
}

// --- Response schemas (Go structs replacing Zod schemas) ---

// groqChatCompletionResponse represents the non-streaming API response.
type groqChatCompletionResponse struct {
	ID      *string         `json:"id,omitempty"`
	Created *float64        `json:"created,omitempty"`
	Model   *string         `json:"model,omitempty"`
	Choices []groqChatChoice `json:"choices"`
	Usage   *GroqTokenUsage `json:"usage,omitempty"`
}

// groqChatChoice represents a choice in the chat completion response.
type groqChatChoice struct {
	Message      groqChatMessage `json:"message"`
	Index        int             `json:"index"`
	FinishReason *string         `json:"finish_reason,omitempty"`
}

// groqChatMessage represents a message in a chat completion choice.
type groqChatMessage struct {
	Content   *string               `json:"content,omitempty"`
	Reasoning *string               `json:"reasoning,omitempty"`
	ToolCalls []groqChatToolCall    `json:"tool_calls,omitempty"`
}

// groqChatToolCall represents a tool call in the response.
type groqChatToolCall struct {
	ID       *string                    `json:"id,omitempty"`
	Type     string                     `json:"type"`
	Function groqChatToolCallFunction   `json:"function"`
}

// groqChatToolCallFunction represents the function part of a tool call.
type groqChatToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// groqChatCompletionResponseSchema is the schema for non-streaming responses.
var groqChatCompletionResponseSchema = &providerutils.Schema[groqChatCompletionResponse]{}

// groqChatChunkSchema is the schema for streaming chunks.
// Parses into map[string]any since the chunk can be either a regular chunk or an error.
var groqChatChunkSchema = &providerutils.Schema[any]{}

// parseGroqTokenUsage attempts to parse a raw value into GroqTokenUsage.
func parseGroqTokenUsage(raw any) *GroqTokenUsage {
	usageMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	usage := &GroqTokenUsage{}

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
			ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
		}{}
		if v, ok := details["reasoning_tokens"].(float64); ok {
			i := int(v)
			usage.CompletionTokensDetails.ReasoningTokens = &i
		}
	}

	return usage
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

// Verify GroqChatLanguageModel implements the LanguageModel interface.
var _ languagemodel.LanguageModel = (*GroqChatLanguageModel)(nil)
