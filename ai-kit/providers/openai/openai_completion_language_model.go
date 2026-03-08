// Ported from: packages/openai/src/completion/openai-completion-language-model.ts
package openai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAICompletionConfig holds the configuration for an OpenAI completion language model.
type OpenAICompletionConfig struct {
	// Provider is the provider identifier (e.g. "openai.completion").
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

// OpenAICompletionLanguageModel implements languagemodel.LanguageModel for OpenAI completion APIs.
type OpenAICompletionLanguageModel struct {
	modelID string
	config  OpenAICompletionConfig
}

// NewOpenAICompletionLanguageModel creates a new OpenAICompletionLanguageModel.
func NewOpenAICompletionLanguageModel(modelID OpenAICompletionModelId, config OpenAICompletionConfig) *OpenAICompletionLanguageModel {
	return &OpenAICompletionLanguageModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *OpenAICompletionLanguageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *OpenAICompletionLanguageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *OpenAICompletionLanguageModel) ModelID() string { return m.modelID }

// SupportedUrls returns supported URL patterns. No URLs are supported for completion models.
func (m *OpenAICompletionLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return map[string][]*regexp.Regexp{}, nil
}

// providerOptionsName extracts the provider name from the provider string.
func (m *OpenAICompletionLanguageModel) providerOptionsName() string {
	parts := strings.SplitN(m.config.Provider, ".", 2)
	return strings.TrimSpace(parts[0])
}

// getArgs prepares the request arguments from CallOptions.
func (m *OpenAICompletionLanguageModel) getArgs(opts languagemodel.CallOptions) (args map[string]any, warnings []shared.Warning, err error) {
	warnings = []shared.Warning{}

	// Convert ProviderOptions to map[string]interface{}
	providerOptsMap := providerOptionsToMap(opts.ProviderOptions)

	// Parse provider options - check "openai" key
	openaiOpts, err := providerutils.ParseProviderOptions(
		"openai",
		providerOptsMap,
		openaiLanguageModelCompletionOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}

	// Also parse under the provider-specific name
	providerSpecificOpts, err := providerutils.ParseProviderOptions(
		m.providerOptionsName(),
		providerOptsMap,
		openaiLanguageModelCompletionOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}

	// Merge options: openai < providerSpecific
	mergedOptions := &OpenAILanguageModelCompletionOptions{}
	if openaiOpts != nil {
		mergeCompletionOptions(mergedOptions, openaiOpts)
	}
	if providerSpecificOpts != nil {
		mergeCompletionOptions(mergedOptions, providerSpecificOpts)
	}

	if opts.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}

	if len(opts.Tools) > 0 {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "tools"})
	}

	if opts.ToolChoice != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "toolChoice"})
	}

	if opts.ResponseFormat != nil {
		if _, ok := opts.ResponseFormat.(languagemodel.ResponseFormatText); !ok {
			detail := "JSON response format is not supported."
			warnings = append(warnings, shared.UnsupportedWarning{Feature: "responseFormat", Details: &detail})
		}
	}

	promptResult := ConvertToOpenAICompletionPrompt(opts.Prompt, "", "")

	stop := append([]string{}, promptResult.StopSequences...)
	stop = append(stop, opts.StopSequences...)

	// Build args
	args = map[string]any{
		"model":  m.modelID,
		"prompt": promptResult.Prompt,
	}

	// Model specific settings
	if mergedOptions.Echo != nil {
		args["echo"] = *mergedOptions.Echo
	}
	if mergedOptions.LogitBias != nil {
		args["logit_bias"] = mergedOptions.LogitBias
	}

	// Handle logprobs (can be bool or number)
	if mergedOptions.Logprobs != nil {
		switch lp := mergedOptions.Logprobs.(type) {
		case bool:
			if lp {
				args["logprobs"] = float64(0)
			}
		case float64:
			args["logprobs"] = lp
		}
	}

	if mergedOptions.Suffix != nil {
		args["suffix"] = *mergedOptions.Suffix
	}
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
	if opts.Seed != nil {
		args["seed"] = *opts.Seed
	}

	if len(stop) > 0 {
		args["stop"] = stop
	}

	return args, warnings, nil
}

// DoGenerate implements languagemodel.LanguageModel.DoGenerate for non-streaming.
func (m *OpenAICompletionLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	args, warnings, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[openaiCompletionResponse]{
		URL: m.config.URL(struct {
			ModelID string
			Path    string
		}{ModelID: m.modelID, Path: "/completions"}),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers)),
		Body:                      args,
		FailedResponseHandler:     openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(openaiCompletionResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := result.Value
	choice := response.Choices[0]

	// Provider metadata
	providerMetadata := shared.ProviderMetadata{
		"openai": map[string]any{},
	}
	if choice.Logprobs != nil {
		providerMetadata["openai"]["logprobs"] = choice.Logprobs
	}

	// Finish reason
	finishReason := languagemodel.FinishReason{
		Unified: MapOpenAICompletionFinishReason(&choice.FinishReason),
		Raw:     &choice.FinishReason,
	}

	// Response metadata
	respMeta := getCompletionResponseMetadata(response.ID, response.Model, response.Created)

	return languagemodel.GenerateResult{
		Content:          []languagemodel.Content{languagemodel.Text{Text: choice.Text}},
		Usage:            ConvertOpenAICompletionUsage(response.Usage),
		FinishReason:     finishReason,
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
func (m *OpenAICompletionLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
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
		}{ModelID: m.modelID, Path: "/completions"}),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(openaiCompletionChunkSchema),
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
		providerMeta := shared.ProviderMetadata{
			"openai": map[string]any{},
		}
		var usage *OpenAICompletionUsage
		isFirstChunk := true

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

				meta := getCompletionResponseMetadataFromMap(chunkMap)
				outputChan <- languagemodel.StreamPartResponseMetadata{
					ResponseMetadata: meta,
				}

				outputChan <- languagemodel.StreamPartTextStart{ID: "0"}
			}

			// Handle usage
			if usageRaw, ok := chunkMap["usage"]; ok && usageRaw != nil {
				usage = parseCompletionUsage(usageRaw)
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
					Unified: MapOpenAICompletionFinishReason(&frStr),
					Raw:     &frStr,
				}
			}

			// Check logprobs
			if logprobsRaw, ok := choiceMap["logprobs"]; ok && logprobsRaw != nil {
				providerMeta["openai"]["logprobs"] = logprobsRaw
			}

			// Check text
			if textRaw, ok := choiceMap["text"]; ok && textRaw != nil {
				if textStr, ok := textRaw.(string); ok && len(textStr) > 0 {
					outputChan <- languagemodel.StreamPartTextDelta{
						ID:    "0",
						Delta: textStr,
					}
				}
			}
		}

		// Flush: end text block if started
		if !isFirstChunk {
			outputChan <- languagemodel.StreamPartTextEnd{ID: "0"}
		}

		// Flush: finish event
		outputChan <- languagemodel.StreamPartFinish{
			FinishReason:     finishReason,
			ProviderMetadata: providerMeta,
			Usage:            ConvertOpenAICompletionUsage(usage),
		}
	}()

	return languagemodel.StreamResult{
		Stream:   outputChan,
		Request:  &languagemodel.StreamResultRequest{Body: body},
		Response: &languagemodel.StreamResultResponse{Headers: result.ResponseHeaders},
	}, nil
}

// --- Helper functions ---

// mergeCompletionOptions merges non-nil fields from src into dst.
func mergeCompletionOptions(dst, src *OpenAILanguageModelCompletionOptions) {
	if src.Echo != nil {
		dst.Echo = src.Echo
	}
	if src.LogitBias != nil {
		dst.LogitBias = src.LogitBias
	}
	if src.Suffix != nil {
		dst.Suffix = src.Suffix
	}
	if src.User != nil {
		dst.User = src.User
	}
	if src.Logprobs != nil {
		dst.Logprobs = src.Logprobs
	}
}

// getCompletionResponseMetadataFromMap extracts response metadata from a streaming chunk map.
func getCompletionResponseMetadataFromMap(m map[string]any) languagemodel.ResponseMetadata {
	rm := languagemodel.ResponseMetadata{}
	if id, ok := m["id"].(string); ok {
		rm.ID = &id
	}
	if model, ok := m["model"].(string); ok {
		rm.ModelID = &model
	}
	if created, ok := m["created"].(float64); ok {
		t := getCompletionResponseMetadata(nil, nil, &created)
		rm.Timestamp = t.Timestamp
	}
	return rm
}

// parseCompletionUsage attempts to parse a raw value into OpenAICompletionUsage.
func parseCompletionUsage(raw any) *OpenAICompletionUsage {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var usage OpenAICompletionUsage
	if err := json.Unmarshal(data, &usage); err != nil {
		return nil
	}
	return &usage
}

// Verify OpenAICompletionLanguageModel implements the LanguageModel interface.
var _ languagemodel.LanguageModel = (*OpenAICompletionLanguageModel)(nil)
