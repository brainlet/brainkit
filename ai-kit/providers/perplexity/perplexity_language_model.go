// Ported from: packages/perplexity/src/perplexity-language-model.ts
package perplexity

import (
	"encoding/json"
	"regexp"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// PerplexityChatConfig holds configuration for the Perplexity chat model.
type PerplexityChatConfig struct {
	BaseURL    string
	Headers    func() map[string]string
	GenerateID func() string
	Fetch      providerutils.FetchFunction
}

// PerplexityLanguageModel implements languagemodel.LanguageModel for the
// Perplexity chat completions API.
type PerplexityLanguageModel struct {
	modelID PerplexityLanguageModelID
	config  PerplexityChatConfig
}

// NewPerplexityLanguageModel creates a new PerplexityLanguageModel.
func NewPerplexityLanguageModel(
	modelID PerplexityLanguageModelID,
	config PerplexityChatConfig,
) *PerplexityLanguageModel {
	return &PerplexityLanguageModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *PerplexityLanguageModel) SpecificationVersion() string { return "v3" }

// Provider returns "perplexity".
func (m *PerplexityLanguageModel) Provider() string { return "perplexity" }

// ModelID returns the model identifier.
func (m *PerplexityLanguageModel) ModelID() string { return m.modelID }

// SupportedUrls returns an empty map (no URL patterns supported).
func (m *PerplexityLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return map[string][]*regexp.Regexp{}, nil
}

// getArgs prepares the request arguments from CallOptions.
func (m *PerplexityLanguageModel) getArgs(opts languagemodel.CallOptions) (args map[string]any, warnings []shared.Warning, err error) {
	warnings = []shared.Warning{}

	if opts.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}
	if len(opts.StopSequences) > 0 {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "stopSequences"})
	}
	if opts.Seed != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "seed"})
	}

	perplexityMessages, msgErr := ConvertToPerplexityMessages(opts.Prompt)
	if msgErr != nil {
		return nil, nil, msgErr
	}

	args = map[string]any{
		"model":    m.modelID,
		"messages": perplexityMessages,
	}

	// Standardized settings
	if opts.FrequencyPenalty != nil {
		args["frequency_penalty"] = *opts.FrequencyPenalty
	}
	if opts.MaxOutputTokens != nil {
		args["max_tokens"] = *opts.MaxOutputTokens
	}
	if opts.PresencePenalty != nil {
		args["presence_penalty"] = *opts.PresencePenalty
	}
	if opts.Temperature != nil {
		args["temperature"] = *opts.Temperature
	}
	if opts.TopK != nil {
		args["top_k"] = *opts.TopK
	}
	if opts.TopP != nil {
		args["top_p"] = *opts.TopP
	}

	// Response format
	if jsonFormat, ok := opts.ResponseFormat.(languagemodel.ResponseFormatJSON); ok {
		if jsonFormat.Schema != nil {
			args["response_format"] = map[string]any{
				"type": "json_schema",
				"json_schema": map[string]any{
					"schema": jsonFormat.Schema,
				},
			}
		}
	}

	// Provider extensions
	if opts.ProviderOptions != nil {
		if perplexityOpts, ok := opts.ProviderOptions["perplexity"]; ok {
			for k, v := range perplexityOpts {
				args[k] = v
			}
		}
	}

	return args, warnings, nil
}

// DoGenerate implements languagemodel.LanguageModel.DoGenerate.
func (m *PerplexityLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	args, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[perplexityResponse]{
		URL:                       m.config.BaseURL + "/chat/completions",
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      args,
		FailedResponseHandler:     wrapErrorHandler(perplexityErrorResponseSchema, perplexityErrorToMessage),
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(perplexityResponseSchema),
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
	if choice.Message.Content != "" {
		content = append(content, languagemodel.Text{Text: choice.Message.Content})
	}

	// Sources (citations)
	if response.Citations != nil {
		for _, url := range response.Citations {
			content = append(content, languagemodel.SourceURL{
				ID:  m.config.GenerateID(),
				URL: url,
			})
		}
	}

	// Finish reason
	finishReason := languagemodel.FinishReason{
		Unified: MapPerplexityFinishReason(choice.FinishReason),
	}
	if choice.FinishReason != nil {
		finishReason.Raw = choice.FinishReason
	}

	// Provider metadata
	var images []map[string]any
	if response.Images != nil {
		for _, img := range response.Images {
			images = append(images, map[string]any{
				"imageUrl":  img.ImageURL,
				"originUrl": img.OriginURL,
				"height":    img.Height,
				"width":     img.Width,
			})
		}
	}

	perplexityUsageMeta := map[string]any{
		"citationTokens":   nil,
		"numSearchQueries": nil,
	}
	if response.Usage != nil {
		if response.Usage.CitationTokens != nil {
			perplexityUsageMeta["citationTokens"] = *response.Usage.CitationTokens
		}
		if response.Usage.NumSearchQueries != nil {
			perplexityUsageMeta["numSearchQueries"] = *response.Usage.NumSearchQueries
		}
	}

	providerMetadata := shared.ProviderMetadata{
		"perplexity": {
			"images": images,
			"usage":  perplexityUsageMeta,
		},
	}

	// Response metadata
	respMeta := getResponseMetadata(response)

	bodyJSON, _ := json.Marshal(args)

	return languagemodel.GenerateResult{
		Content:          content,
		FinishReason:     finishReason,
		Usage:            ConvertPerplexityUsage(toPerplexityUsage(response.Usage)),
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

// DoStream implements languagemodel.LanguageModel.DoStream.
func (m *PerplexityLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
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
		URL:                       m.config.BaseURL + "/chat/completions",
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     wrapErrorHandler(perplexityErrorResponseSchema, perplexityErrorToMessage),
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(perplexityChunkSchemaInstance),
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

		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
		}
		var usage *PerplexityUsage
		providerMeta := shared.ProviderMetadata{
			"perplexity": {
				"usage": map[string]any{
					"citationTokens":   nil,
					"numSearchQueries": nil,
				},
				"images": nil,
			},
		}
		isFirstChunk := true
		isActive := false

		// Start event
		outputChan <- languagemodel.StreamPartStreamStart{Warnings: warnings}

		for chunk := range inputChan {
			// Emit raw chunk if requested
			if includeRawChunks {
				outputChan <- languagemodel.StreamPartRaw{RawValue: chunk.RawValue}
			}

			if !chunk.Success {
				outputChan <- languagemodel.StreamPartError{Error: chunk.Error}
				continue
			}

			chunkMap, ok := chunk.Value.(map[string]any)
			if !ok {
				continue
			}

			if isFirstChunk {
				respMeta := getResponseMetadataFromMap(chunkMap)
				outputChan <- languagemodel.StreamPartResponseMetadata{
					ResponseMetadata: respMeta,
				}

				// Citations
				if citationsRaw, ok := chunkMap["citations"]; ok && citationsRaw != nil {
					if citations, ok := citationsRaw.([]any); ok {
						for _, cRaw := range citations {
							if urlStr, ok := cRaw.(string); ok {
								outputChan <- languagemodel.SourceURL{
									ID:  m.config.GenerateID(),
									URL: urlStr,
								}
							}
						}
					}
				}

				isFirstChunk = false
			}

			// Usage
			if usageRaw, ok := chunkMap["usage"]; ok && usageRaw != nil {
				usage = parsePerplexityUsage(usageRaw)
				if usage != nil {
					usageMeta := map[string]any{
						"citationTokens":   nil,
						"numSearchQueries": nil,
					}
					if usageMap, ok := usageRaw.(map[string]any); ok {
						if ct, ok := usageMap["citation_tokens"]; ok && ct != nil {
							usageMeta["citationTokens"] = ct
						}
						if nsq, ok := usageMap["num_search_queries"]; ok && nsq != nil {
							usageMeta["numSearchQueries"] = nsq
						}
					}
					providerMeta["perplexity"]["usage"] = usageMeta
				}
			}

			// Images
			if imagesRaw, ok := chunkMap["images"]; ok && imagesRaw != nil {
				if imagesArr, ok := imagesRaw.([]any); ok {
					var images []map[string]any
					for _, imgRaw := range imagesArr {
						if imgMap, ok := imgRaw.(map[string]any); ok {
							images = append(images, map[string]any{
								"imageUrl":  imgMap["image_url"],
								"originUrl": imgMap["origin_url"],
								"height":    imgMap["height"],
								"width":     imgMap["width"],
							})
						}
					}
					providerMeta["perplexity"]["images"] = images
				}
			}

			// Choices
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

			// Finish reason
			if fr, ok := choiceMap["finish_reason"]; ok && fr != nil {
				if frStr, ok := fr.(string); ok {
					finishReason = languagemodel.FinishReason{
						Unified: MapPerplexityFinishReason(&frStr),
						Raw:     &frStr,
					}
				}
			}

			// Delta
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
					if !isActive {
						outputChan <- languagemodel.StreamPartTextStart{ID: "0"}
						isActive = true
					}
					outputChan <- languagemodel.StreamPartTextDelta{
						ID:    "0",
						Delta: contentStr,
					}
				}
			}
		}

		// Flush
		if isActive {
			outputChan <- languagemodel.StreamPartTextEnd{ID: "0"}
		}

		outputChan <- languagemodel.StreamPartFinish{
			FinishReason:     finishReason,
			Usage:            ConvertPerplexityUsage(usage),
			ProviderMetadata: providerMeta,
		}
	}()

	return languagemodel.StreamResult{
		Stream:   outputChan,
		Request:  &languagemodel.StreamResultRequest{Body: body},
		Response: &languagemodel.StreamResultResponse{Headers: result.ResponseHeaders},
	}, nil
}

// --- Response types ---

type perplexityImage struct {
	ImageURL  string `json:"image_url"`
	OriginURL string `json:"origin_url"`
	Height    int    `json:"height"`
	Width     int    `json:"width"`
}

type perplexityResponseUsage struct {
	PromptTokens     *int `json:"prompt_tokens,omitempty"`
	CompletionTokens *int `json:"completion_tokens,omitempty"`
	TotalTokens      *int `json:"total_tokens,omitempty"`
	CitationTokens   *int `json:"citation_tokens,omitempty"`
	NumSearchQueries *int `json:"num_search_queries,omitempty"`
	ReasoningTokens  *int `json:"reasoning_tokens,omitempty"`
}

type perplexityResponse struct {
	ID        string `json:"id"`
	Created   int64  `json:"created"`
	Model     string `json:"model"`
	Choices   []perplexityChoice `json:"choices"`
	Citations []string           `json:"citations,omitempty"`
	Images    []perplexityImage  `json:"images,omitempty"`
	Usage     *perplexityResponseUsage `json:"usage,omitempty"`
}

type perplexityChoice struct {
	Message      perplexityChoiceMessage `json:"message"`
	FinishReason *string                `json:"finish_reason,omitempty"`
}

type perplexityChoiceMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// --- Error types ---

type perplexityErrorResponse struct {
	Error perplexityErrorDetail `json:"error"`
}

type perplexityErrorDetail struct {
	Code    int     `json:"code"`
	Message *string `json:"message,omitempty"`
	Type    *string `json:"type,omitempty"`
}

// --- Schemas ---

var perplexityResponseSchema = &providerutils.Schema[perplexityResponse]{}

var perplexityChunkSchemaInstance = &providerutils.Schema[any]{}

var perplexityErrorResponseSchema = &providerutils.Schema[perplexityErrorResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[perplexityErrorResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[perplexityErrorResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var errResp perplexityErrorResponse
		if err := json.Unmarshal(data, &errResp); err != nil {
			return &providerutils.ValidationResult[perplexityErrorResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[perplexityErrorResponse]{
			Success: true,
			Value:   errResp,
		}, nil
	},
}

func perplexityErrorToMessage(data perplexityErrorResponse) string {
	if data.Error.Message != nil {
		return *data.Error.Message
	}
	if data.Error.Type != nil {
		return *data.Error.Type
	}
	return "unknown error"
}

// --- Helpers ---

func getResponseMetadata(resp perplexityResponse) languagemodel.ResponseMetadata {
	t := time.Unix(resp.Created, 0)
	id := resp.ID
	model := resp.Model
	return languagemodel.ResponseMetadata{
		ID:        &id,
		ModelID:   &model,
		Timestamp: &t,
	}
}

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

func parsePerplexityUsage(raw any) *PerplexityUsage {
	usageMap, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	usage := &PerplexityUsage{}
	if v, ok := usageMap["prompt_tokens"].(float64); ok {
		i := int(v)
		usage.PromptTokens = &i
	}
	if v, ok := usageMap["completion_tokens"].(float64); ok {
		i := int(v)
		usage.CompletionTokens = &i
	}
	if v, ok := usageMap["reasoning_tokens"].(float64); ok {
		i := int(v)
		usage.ReasoningTokens = &i
	}
	return usage
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

// toPerplexityUsage converts perplexityResponseUsage to PerplexityUsage.
func toPerplexityUsage(u *perplexityResponseUsage) *PerplexityUsage {
	if u == nil {
		return nil
	}
	return &PerplexityUsage{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		ReasoningTokens:  u.ReasoningTokens,
	}
}

// wrapErrorHandler wraps a typed error handler to satisfy ResponseHandler[error].
func wrapErrorHandler[T any](
	errorSchema *providerutils.Schema[T],
	errorToMessage func(T) string,
) providerutils.ResponseHandler[error] {
	typedHandler := providerutils.CreateJsonErrorResponseHandler(errorSchema, errorToMessage, nil)
	return func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
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
}

// Verify PerplexityLanguageModel implements the LanguageModel interface.
var _ languagemodel.LanguageModel = (*PerplexityLanguageModel)(nil)
