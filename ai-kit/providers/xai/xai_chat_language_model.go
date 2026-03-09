// Ported from: packages/xai/src/xai-chat-language-model.ts
package xai

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// XaiChatConfig configures the xAI chat language model.
type XaiChatConfig struct {
	Provider   string
	BaseURL    string
	Headers    func() map[string]string
	GenerateID providerutils.IdGenerator
	Fetch      providerutils.FetchFunction
}

// XaiChatUsage represents usage information from the xAI API.
type XaiChatUsage struct {
	PromptTokens     int                          `json:"prompt_tokens"`
	CompletionTokens int                          `json:"completion_tokens"`
	TotalTokens      int                          `json:"total_tokens"`
	PromptTokensDetails    *XaiPromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *XaiCompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

// XaiPromptTokensDetails contains details about prompt token usage.
type XaiPromptTokensDetails struct {
	TextTokens   *int `json:"text_tokens,omitempty"`
	AudioTokens  *int `json:"audio_tokens,omitempty"`
	ImageTokens  *int `json:"image_tokens,omitempty"`
	CachedTokens *int `json:"cached_tokens,omitempty"`
}

// XaiCompletionTokensDetails contains details about completion token usage.
type XaiCompletionTokensDetails struct {
	ReasoningTokens          *int `json:"reasoning_tokens,omitempty"`
	AudioTokens              *int `json:"audio_tokens,omitempty"`
	AcceptedPredictionTokens *int `json:"accepted_prediction_tokens,omitempty"`
	RejectedPredictionTokens *int `json:"rejected_prediction_tokens,omitempty"`
}

// xaiChatResponseSchema is the schema for xAI chat completion responses.
var xaiChatResponseSchema = &providerutils.Schema[xaiChatResponse]{}

// xaiChatResponse represents the xAI chat completion API response.
type xaiChatResponse struct {
	ID       *string           `json:"id,omitempty"`
	Created  *int64            `json:"created,omitempty"`
	Model    *string           `json:"model,omitempty"`
	Choices  []xaiChatChoice   `json:"choices,omitempty"`
	Object   *string           `json:"object,omitempty"`
	Usage    *XaiChatUsage     `json:"usage,omitempty"`
	Citations []string         `json:"citations,omitempty"`
	Code     *string           `json:"code,omitempty"`
	Error    *string           `json:"error,omitempty"`
}

// xaiChatChoice represents a choice in the response.
type xaiChatChoice struct {
	Message      xaiChatChoiceMessage `json:"message"`
	Index        int                  `json:"index"`
	FinishReason *string              `json:"finish_reason,omitempty"`
}

// xaiChatChoiceMessage represents the message in a choice.
type xaiChatChoiceMessage struct {
	Role             string                  `json:"role"`
	Content          *string                 `json:"content,omitempty"`
	ReasoningContent *string                 `json:"reasoning_content,omitempty"`
	ToolCalls        []xaiChatToolCallEntry  `json:"tool_calls,omitempty"`
}

// xaiChatToolCallEntry represents a tool call in a response message.
type xaiChatToolCallEntry struct {
	ID       string                    `json:"id"`
	Type     string                    `json:"type"`
	Function xaiChatToolCallFunction   `json:"function"`
}

// xaiChatToolCallFunction contains the function details.
type xaiChatToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// xaiChatChunk represents a streaming chunk from the xAI API.
type xaiChatChunk struct {
	ID       *string           `json:"id,omitempty"`
	Created  *int64            `json:"created,omitempty"`
	Model    *string           `json:"model,omitempty"`
	Choices  []xaiChatChunkChoice `json:"choices"`
	Usage    *XaiChatUsage     `json:"usage,omitempty"`
	Citations []string         `json:"citations,omitempty"`
}

// xaiChatChunkChoice represents a choice in a streaming chunk.
type xaiChatChunkChoice struct {
	Delta        xaiChatChunkDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason,omitempty"`
	Index        int               `json:"index"`
}

// xaiChatChunkDelta represents the delta in a streaming chunk.
type xaiChatChunkDelta struct {
	Role             *string                 `json:"role,omitempty"`
	Content          *string                 `json:"content,omitempty"`
	ReasoningContent *string                 `json:"reasoning_content,omitempty"`
	ToolCalls        []xaiChatToolCallEntry  `json:"tool_calls,omitempty"`
}

// xaiStreamError represents an error from the streaming API.
type xaiStreamError struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

// xaiChatChunkSchema is the schema for streaming chunk validation.
var xaiChatChunkSchema = &providerutils.Schema[xaiChatChunk]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[xaiChatChunk], error) {
		b, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		var chunk xaiChatChunk
		if err := json.Unmarshal(b, &chunk); err != nil {
			return nil, err
		}
		return &providerutils.ValidationResult[xaiChatChunk]{
			Success: true,
			Value:   chunk,
		}, nil
	},
}

// xaiStreamErrorSchema is the schema for stream error validation.
var xaiStreamErrorSchema = &providerutils.Schema[xaiStreamError]{}

// XaiChatLanguageModel implements the LanguageModel interface for xAI chat models.
type XaiChatLanguageModel struct {
	specificationVersion string
	modelId              XaiChatModelId
	config               XaiChatConfig
}

// NewXaiChatLanguageModel creates a new xAI chat language model.
func NewXaiChatLanguageModel(modelId XaiChatModelId, config XaiChatConfig) *XaiChatLanguageModel {
	return &XaiChatLanguageModel{
		specificationVersion: "v3",
		modelId:              modelId,
		config:               config,
	}
}

// SpecificationVersion returns the language model interface version.
func (m *XaiChatLanguageModel) SpecificationVersion() string {
	return m.specificationVersion
}

// Provider returns the provider ID.
func (m *XaiChatLanguageModel) Provider() string {
	return m.config.Provider
}

// ModelID returns the model ID.
func (m *XaiChatLanguageModel) ModelID() string {
	return m.modelId
}

// SupportedUrls returns the supported URL patterns.
func (m *XaiChatLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return map[string][]*regexp.Regexp{
		"image/*": {regexp.MustCompile(`^https?://.*$`)},
	}, nil
}

// getArgs builds the request arguments from call options.
func (m *XaiChatLanguageModel) getArgs(options languagemodel.CallOptions) (map[string]interface{}, []shared.Warning, error) {
	var warnings []shared.Warning

	// Parse xAI-specific provider options
	xaiOpts, err := providerutils.ParseProviderOptions("xai", providerOptionsToMap(options.ProviderOptions), xaiLanguageModelChatOptionsSchema)
	if err != nil {
		return nil, nil, err
	}
	if xaiOpts == nil {
		xaiOpts = &XaiLanguageModelChatOptions{}
	}

	// Check for unsupported parameters
	if options.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}
	if options.FrequencyPenalty != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "frequencyPenalty"})
	}
	if options.PresencePenalty != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "presencePenalty"})
	}
	if len(options.StopSequences) > 0 {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "stopSequences"})
	}

	// Convert messages
	result := convertToXaiChatMessages(options.Prompt)
	warnings = append(warnings, result.Warnings...)

	// Prepare tools
	toolResult := prepareTools(options.Tools, options.ToolChoice)
	warnings = append(warnings, toolResult.Warnings...)

	// Build base args
	baseArgs := map[string]interface{}{
		"model":    m.modelId,
		"messages": result.Messages,
	}

	// Logprobs
	if (xaiOpts.Logprobs != nil && *xaiOpts.Logprobs) || xaiOpts.TopLogprobs != nil {
		baseArgs["logprobs"] = true
	}
	if xaiOpts.TopLogprobs != nil {
		baseArgs["top_logprobs"] = *xaiOpts.TopLogprobs
	}

	// Standard generation settings
	if options.MaxOutputTokens != nil {
		baseArgs["max_completion_tokens"] = *options.MaxOutputTokens
	}
	if options.Temperature != nil {
		baseArgs["temperature"] = *options.Temperature
	}
	if options.TopP != nil {
		baseArgs["top_p"] = *options.TopP
	}
	if options.Seed != nil {
		baseArgs["seed"] = *options.Seed
	}

	// Reasoning effort
	if xaiOpts.ReasoningEffort != nil {
		baseArgs["reasoning_effort"] = *xaiOpts.ReasoningEffort
	}

	// Parallel function calling
	if xaiOpts.ParallelFunctionCalling != nil {
		baseArgs["parallel_function_calling"] = *xaiOpts.ParallelFunctionCalling
	}

	// Response format
	if options.ResponseFormat != nil {
		if jsonFmt, ok := options.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
			if jsonFmt.Schema != nil {
				name := "response"
				if jsonFmt.Name != nil {
					name = *jsonFmt.Name
				}
				baseArgs["response_format"] = map[string]interface{}{
					"type": "json_schema",
					"json_schema": map[string]interface{}{
						"name":   name,
						"schema": jsonFmt.Schema,
						"strict": true,
					},
				}
			} else {
				baseArgs["response_format"] = map[string]interface{}{
					"type": "json_object",
				}
			}
		}
	}

	// Search parameters
	if xaiOpts.SearchParameters != nil {
		sp := xaiOpts.SearchParameters
		searchParams := map[string]interface{}{
			"mode": sp.Mode,
		}
		if sp.ReturnCitations != nil {
			searchParams["return_citations"] = *sp.ReturnCitations
		}
		if sp.FromDate != nil {
			searchParams["from_date"] = *sp.FromDate
		}
		if sp.ToDate != nil {
			searchParams["to_date"] = *sp.ToDate
		}
		if sp.MaxSearchResults != nil {
			searchParams["max_search_results"] = *sp.MaxSearchResults
		}
		if len(sp.Sources) > 0 {
			var sources []interface{}
			for _, source := range sp.Sources {
				s := map[string]interface{}{
					"type": source.Type,
				}
				switch source.Type {
				case "web":
					if source.Country != nil {
						s["country"] = *source.Country
					}
					if len(source.ExcludedWebsites) > 0 {
						s["excluded_websites"] = source.ExcludedWebsites
					}
					if len(source.AllowedWebsites) > 0 {
						s["allowed_websites"] = source.AllowedWebsites
					}
					if source.SafeSearch != nil {
						s["safe_search"] = *source.SafeSearch
					}
				case "x":
					if len(source.ExcludedXHandles) > 0 {
						s["excluded_x_handles"] = source.ExcludedXHandles
					}
					// Use IncludedXHandles, fall back to XHandles (deprecated)
					included := source.IncludedXHandles
					if included == nil {
						included = source.XHandles
					}
					if len(included) > 0 {
						s["included_x_handles"] = included
					}
					if source.PostFavoriteCount != nil {
						s["post_favorite_count"] = *source.PostFavoriteCount
					}
					if source.PostViewCount != nil {
						s["post_view_count"] = *source.PostViewCount
					}
				case "news":
					if source.Country != nil {
						s["country"] = *source.Country
					}
					if len(source.ExcludedWebsites) > 0 {
						s["excluded_websites"] = source.ExcludedWebsites
					}
					if source.SafeSearch != nil {
						s["safe_search"] = *source.SafeSearch
					}
				case "rss":
					if len(source.Links) > 0 {
						s["links"] = source.Links
					}
				}
				sources = append(sources, s)
			}
			searchParams["sources"] = sources
		}
		baseArgs["search_parameters"] = searchParams
	}

	// Tools
	if toolResult.Tools != nil {
		baseArgs["tools"] = toolResult.Tools
	}
	if toolResult.ToolChoice != nil {
		baseArgs["tool_choice"] = toolResult.ToolChoice
	}

	return baseArgs, warnings, nil
}

// DoGenerate generates a language model output (non-streaming).
func (m *XaiChatLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	body, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	baseURL := m.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.x.ai/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[xaiChatResponse]{
		URL:                       url,
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), headersToStringMap(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     xaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(xaiChatResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := result.Value

	// Check for API-level error in 200 response
	if response.Error != nil {
		isRetryable := false
		if response.Code != nil && *response.Code == "The service is currently unavailable" {
			isRetryable = true
		}
		rawBody, _ := json.Marshal(result.RawValue)
		return languagemodel.GenerateResult{}, providerutils.NewAPICallError(providerutils.APICallErrorOptions{
			Message:           *response.Error,
			URL:               url,
			RequestBodyValues: body,
			StatusCode:        200,
			ResponseHeaders:   result.ResponseHeaders,
			ResponseBody:      string(rawBody),
			IsRetryable:       isRetryable,
		})
	}

	if len(response.Choices) == 0 {
		return languagemodel.GenerateResult{}, fmt.Errorf("no choices in xAI response")
	}

	choice := response.Choices[0]
	var content []languagemodel.Content

	// Extract text content
	if choice.Message.Content != nil && len(*choice.Message.Content) > 0 {
		text := *choice.Message.Content

		// Skip if this content duplicates the last assistant message
		msgs, ok := body["messages"].([]interface{})
		if ok && len(msgs) > 0 {
			if lastMsg, ok := msgs[len(msgs)-1].(map[string]interface{}); ok {
				if lastMsg["role"] == "assistant" {
					if lastContent, ok := lastMsg["content"].(string); ok && text == lastContent {
						text = ""
					}
				}
			}
		}

		if len(text) > 0 {
			content = append(content, languagemodel.Text{Text: text})
		}
	}

	// Extract reasoning content
	if choice.Message.ReasoningContent != nil && len(*choice.Message.ReasoningContent) > 0 {
		content = append(content, languagemodel.Reasoning{
			Text: *choice.Message.ReasoningContent,
		})
	}

	// Extract tool calls
	for _, tc := range choice.Message.ToolCalls {
		content = append(content, languagemodel.ToolCall{
			ToolCallID: tc.ID,
			ToolName:   tc.Function.Name,
			Input:      tc.Function.Arguments,
		})
	}

	// Extract citations
	if len(response.Citations) > 0 {
		for _, citationURL := range response.Citations {
			content = append(content, languagemodel.SourceURL{
				ID:  m.config.GenerateID(),
				URL: citationURL,
			})
		}
	}

	// Build usage
	usage := zeroUsage()
	if response.Usage != nil {
		usage = convertXaiChatUsage(*response.Usage)
	}

	metadata := getResponseMetadata(getResponseMetadataInput{
		ID:      response.ID,
		Model:   response.Model,
		Created: response.Created,
	})

	return languagemodel.GenerateResult{
		Content: content,
		FinishReason: languagemodel.FinishReason{
			Unified: mapXaiFinishReason(choice.FinishReason),
			Raw:     choice.FinishReason,
		},
		Usage: usage,
		Request: &languagemodel.GenerateResultRequest{
			Body: body,
		},
		Response: &languagemodel.GenerateResultResponse{
			ResponseMetadata: metadata,
			Headers:          result.ResponseHeaders,
			Body:             result.RawValue,
		},
		Warnings: warnings,
	}, nil
}

// DoStream generates a language model output (streaming).
func (m *XaiChatLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	args, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	body := make(map[string]interface{})
	for k, v := range args {
		body[k] = v
	}
	body["stream"] = true
	body["stream_options"] = map[string]interface{}{
		"include_usage": true,
	}

	baseURL := m.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.x.ai/v1"
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[xaiChatChunk]]{
		URL:                   url,
		Headers:               providerutils.CombineHeaders(m.config.Headers(), headersToStringMap(options.Headers)),
		Body:                  body,
		FailedResponseHandler: xaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(xaiChatChunkSchema),
		Ctx:                   options.Ctx,
		Fetch:                 m.config.Fetch,
	})
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	incomingCh := result.Value

	outCh := make(chan languagemodel.StreamPart)

	go func() {
		defer close(outCh)

		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
		}
		var usage *languagemodel.Usage
		isFirstChunk := true
		contentBlocks := make(map[string]struct{ blockType string; ended bool })
		lastReasoningDeltas := make(map[string]string)
		activeReasoningBlockId := ""

		// Send stream-start
		outCh <- languagemodel.StreamPartStreamStart{Warnings: warnings}

		for chunk := range incomingCh {
			// Emit raw chunk if requested
			if options.IncludeRawChunks != nil && *options.IncludeRawChunks {
				outCh <- languagemodel.StreamPartRaw{RawValue: chunk.RawValue}
			}

			if !chunk.Success {
				outCh <- languagemodel.StreamPartError{Error: chunk.Error}
				continue
			}

			value := chunk.Value

			// Emit response metadata on first chunk
			if isFirstChunk {
				metadata := getResponseMetadata(getResponseMetadataInput{
					ID:      value.ID,
					Model:   value.Model,
					Created: value.Created,
				})
				outCh <- languagemodel.StreamPartResponseMetadata{
					ResponseMetadata: metadata,
				}
				isFirstChunk = false
			}

			// Emit citations
			if len(value.Citations) > 0 {
				for _, citationURL := range value.Citations {
					outCh <- languagemodel.SourceURL{
						ID:  m.config.GenerateID(),
						URL: citationURL,
					}
				}
			}

			// Update usage
			if value.Usage != nil {
				u := convertXaiChatUsage(*value.Usage)
				usage = &u
			}

			if len(value.Choices) == 0 {
				continue
			}

			choice := value.Choices[0]

			// Update finish reason
			if choice.FinishReason != nil {
				finishReason = languagemodel.FinishReason{
					Unified: mapXaiFinishReason(choice.FinishReason),
					Raw:     choice.FinishReason,
				}
			}

			delta := choice.Delta
			choiceIndex := choice.Index

			// Process text content
			if delta.Content != nil && len(*delta.Content) > 0 {
				textContent := *delta.Content

				// End active reasoning block when text content arrives
				if activeReasoningBlockId != "" {
					if block, ok := contentBlocks[activeReasoningBlockId]; ok && !block.ended {
						outCh <- languagemodel.StreamPartReasoningEnd{ID: activeReasoningBlockId}
						block.ended = true
						contentBlocks[activeReasoningBlockId] = block
						activeReasoningBlockId = ""
					}
				}

				// Skip if this content duplicates the last assistant message
				msgs, ok := body["messages"].([]interface{})
				if ok && len(msgs) > 0 {
					if lastMsg, ok := msgs[len(msgs)-1].(map[string]interface{}); ok {
						if lastMsg["role"] == "assistant" {
							if lastContent, ok := lastMsg["content"].(string); ok && textContent == lastContent {
								continue
							}
						}
					}
				}

				fallbackID := fmt.Sprintf("%d", choiceIndex)
				blockId := fmt.Sprintf("text-%v", coalesce(value.ID, &fallbackID))

				if _, exists := contentBlocks[blockId]; !exists {
					contentBlocks[blockId] = struct{ blockType string; ended bool }{"text", false}
					outCh <- languagemodel.StreamPartTextStart{ID: blockId}
				}

				outCh <- languagemodel.StreamPartTextDelta{ID: blockId, Delta: textContent}
			}

			// Process reasoning content
			if delta.ReasoningContent != nil && len(*delta.ReasoningContent) > 0 {
				reasoningFallbackID := fmt.Sprintf("%d", choiceIndex)
				blockId := fmt.Sprintf("reasoning-%v", coalesce(value.ID, &reasoningFallbackID))

				// Skip if this reasoning content duplicates the last delta
				if lastDelta, ok := lastReasoningDeltas[blockId]; ok && lastDelta == *delta.ReasoningContent {
					continue
				}
				lastReasoningDeltas[blockId] = *delta.ReasoningContent

				if _, exists := contentBlocks[blockId]; !exists {
					contentBlocks[blockId] = struct{ blockType string; ended bool }{"reasoning", false}
					activeReasoningBlockId = blockId
					outCh <- languagemodel.StreamPartReasoningStart{ID: blockId}
				}

				outCh <- languagemodel.StreamPartReasoningDelta{ID: blockId, Delta: *delta.ReasoningContent}
			}

			// Process tool calls
			if len(delta.ToolCalls) > 0 {
				// End active reasoning block before tool calls start
				if activeReasoningBlockId != "" {
					if block, ok := contentBlocks[activeReasoningBlockId]; ok && !block.ended {
						outCh <- languagemodel.StreamPartReasoningEnd{ID: activeReasoningBlockId}
						block.ended = true
						contentBlocks[activeReasoningBlockId] = block
						activeReasoningBlockId = ""
					}
				}

				for _, tc := range delta.ToolCalls {
					toolCallId := tc.ID

					outCh <- languagemodel.StreamPartToolInputStart{
						ID:       toolCallId,
						ToolName: tc.Function.Name,
					}

					outCh <- languagemodel.StreamPartToolInputDelta{
						ID:    toolCallId,
						Delta: tc.Function.Arguments,
					}

					outCh <- languagemodel.StreamPartToolInputEnd{
						ID: toolCallId,
					}

					outCh <- languagemodel.ToolCall{
						ToolCallID: toolCallId,
						ToolName:   tc.Function.Name,
						Input:      tc.Function.Arguments,
					}
				}
			}
		}

		// End any blocks that haven't been ended yet
		for blockId, block := range contentBlocks {
			if !block.ended {
				if block.blockType == "text" {
					outCh <- languagemodel.StreamPartTextEnd{ID: blockId}
				} else {
					outCh <- languagemodel.StreamPartReasoningEnd{ID: blockId}
				}
			}
		}

		// Emit finish
		finalUsage := zeroUsage()
		if usage != nil {
			finalUsage = *usage
		}

		outCh <- languagemodel.StreamPartFinish{
			FinishReason: finishReason,
			Usage:        finalUsage,
		}
	}()

	return languagemodel.StreamResult{
		Stream: outCh,
		Request: &languagemodel.StreamResultRequest{
			Body: body,
		},
		Response: &languagemodel.StreamResultResponse{
			Headers: result.ResponseHeaders,
		},
	}, nil
}

// zeroUsage returns a zero-valued Usage.
func zeroUsage() languagemodel.Usage {
	zero := 0
	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:      &zero,
			NoCache:    &zero,
			CacheRead:  &zero,
			CacheWrite: &zero,
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &zero,
			Text:      &zero,
			Reasoning: &zero,
		},
	}
}

// headersToStringMap converts a map[string]*string to map[string]string,
// skipping nil values.
func headersToStringMap(headers map[string]*string) map[string]string {
	if headers == nil {
		return nil
	}
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}

// coalesce returns the first non-nil string pointer's value, or the fallback.
func coalesce(ptrs ...*string) string {
	for _, p := range ptrs {
		if p != nil {
			return *p
		}
	}
	return ""
}

// providerOptionsToMap converts shared.ProviderOptions to map[string]interface{}
// for use with providerutils.ParseProviderOptions.
func providerOptionsToMap(opts shared.ProviderOptions) map[string]interface{} {
	if opts == nil {
		return nil
	}
	result := make(map[string]interface{}, len(opts))
	for k, v := range opts {
		result[k] = v
	}
	return result
}

// coalesceStr returns the first non-empty string.
func coalesceStr(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
