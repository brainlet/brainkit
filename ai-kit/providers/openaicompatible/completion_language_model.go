// Ported from: packages/openai-compatible/src/completion/openai-compatible-completion-language-model.ts
package openaicompatible

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// CompletionConfig holds the configuration for a completion language model.
type CompletionConfig struct {
	// Provider is the provider identifier (e.g. "openai.completion").
	Provider string

	// IncludeUsage controls whether to request usage stats in streaming mode
	// via stream_options.include_usage.
	IncludeUsage *bool

	// Headers returns the HTTP headers to send with each request.
	Headers func() map[string]string

	// URL builds the full API URL from the given path.
	URL func(path string) string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction

	// ErrorStructure is the provider-specific error structure.
	// If nil, DefaultErrorStructure is used.
	ErrorStructure *ProviderErrorStructure[ErrorData]

	// SupportedURLs returns supported URL patterns by media type.
	SupportedURLs func() (map[string][]*regexp.Regexp, error)
}

// CompletionLanguageModel implements languagemodel.LanguageModel for
// OpenAI-compatible completion endpoints.
type CompletionLanguageModel struct {
	modelID              CompletionModelID
	config               CompletionConfig
	failedResponseHandler providerutils.ResponseHandler[error]
	chunkSchema          *providerutils.Schema[completionChunk]
}

// NewCompletionLanguageModel creates a new CompletionLanguageModel.
func NewCompletionLanguageModel(modelID string, config CompletionConfig) *CompletionLanguageModel {
	errorStructure := config.ErrorStructure
	if errorStructure == nil {
		es := DefaultErrorStructure
		errorStructure = &es
	}

	chunkSchema := newCompletionChunkSchema(errorStructure.ErrorSchema)

	failedResponseHandler := providerutils.CreateJsonErrorResponseHandler(
		errorStructure.ErrorSchema,
		errorStructure.ErrorToMessage,
		errorStructure.IsRetryable,
	)

	// Wrap to match ResponseHandler[error]
	wrappedHandler := providerutils.ResponseHandler[error](func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
		result, err := failedResponseHandler(opts)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		return &providerutils.ResponseHandlerResult[error]{
			Value:           result.Value,
			RawValue:        result.RawValue,
			ResponseHeaders: result.ResponseHeaders,
		}, nil
	})

	return &CompletionLanguageModel{
		modelID:               modelID,
		config:                config,
		failedResponseHandler: wrappedHandler,
		chunkSchema:           chunkSchema,
	}
}

// SpecificationVersion returns "v3".
func (m *CompletionLanguageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *CompletionLanguageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *CompletionLanguageModel) ModelID() string { return m.modelID }

func (m *CompletionLanguageModel) providerOptionsName() string {
	return strings.TrimSpace(strings.SplitN(m.config.Provider, ".", 2)[0])
}

// SupportedUrls returns supported URL patterns by media type.
func (m *CompletionLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	if m.config.SupportedURLs != nil {
		return m.config.SupportedURLs()
	}
	return map[string][]*regexp.Regexp{}, nil
}

// completionArgs holds the prepared request arguments and warnings.
type completionArgs struct {
	Args     map[string]interface{}
	Warnings []shared.Warning
}

func (m *CompletionLanguageModel) getArgs(options languagemodel.CallOptions) (*completionArgs, error) {
	var warnings []shared.Warning

	// Parse provider options
	completionOpts, err := providerutils.ParseProviderOptions(
		m.providerOptionsName(),
		providerOptionsToMap(options.ProviderOptions),
		CompletionOptionsSchema,
	)
	if err != nil {
		return nil, err
	}
	if completionOpts == nil {
		completionOpts = &CompletionOptions{}
	}

	if options.TopK != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "topK"})
	}

	if len(options.Tools) > 0 {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "tools"})
	}

	if options.ToolChoice != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "toolChoice"})
	}

	if options.ResponseFormat != nil {
		if _, isText := options.ResponseFormat.(languagemodel.ResponseFormatText); !isText {
			details := "JSON response format is not supported."
			warnings = append(warnings, shared.UnsupportedWarning{
				Feature: "responseFormat",
				Details: &details,
			})
		}
	}

	promptResult, err := ConvertToCompletionPrompt(options.Prompt, "", "")
	if err != nil {
		return nil, err
	}

	// Merge stop sequences from prompt conversion and user-provided ones
	var stop []string
	stop = append(stop, promptResult.StopSequences...)
	stop = append(stop, options.StopSequences...)

	args := map[string]interface{}{
		"model":  m.modelID,
		"prompt": promptResult.Prompt,
	}

	// Model-specific settings
	if completionOpts.Echo != nil {
		args["echo"] = *completionOpts.Echo
	}
	if completionOpts.LogitBias != nil {
		args["logit_bias"] = completionOpts.LogitBias
	}
	if completionOpts.Suffix != nil {
		args["suffix"] = *completionOpts.Suffix
	}
	if completionOpts.User != nil {
		args["user"] = *completionOpts.User
	}

	// Standardized settings
	if options.MaxOutputTokens != nil {
		args["max_tokens"] = *options.MaxOutputTokens
	}
	if options.Temperature != nil {
		args["temperature"] = *options.Temperature
	}
	if options.TopP != nil {
		args["top_p"] = *options.TopP
	}
	if options.FrequencyPenalty != nil {
		args["frequency_penalty"] = *options.FrequencyPenalty
	}
	if options.PresencePenalty != nil {
		args["presence_penalty"] = *options.PresencePenalty
	}
	if options.Seed != nil {
		args["seed"] = *options.Seed
	}

	// Stop sequences
	if len(stop) > 0 {
		args["stop"] = stop
	}

	// Spread provider-specific extra options
	providerName := m.providerOptionsName()
	if options.ProviderOptions != nil {
		if extra, ok := options.ProviderOptions[providerName]; ok {
			for k, v := range extra {
				args[k] = v
			}
		}
	}

	return &completionArgs{
		Args:     args,
		Warnings: warnings,
	}, nil
}

// DoGenerate performs a non-streaming completion call.
func (m *CompletionLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	prepared, err := m.getArgs(options)
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[completionResponse]{
		URL:     m.config.URL("/completions"),
		Headers: providerutils.CombineHeaders(m.config.Headers(), convertHeaders(options.Headers)),
		Body:    prepared.Args,
		FailedResponseHandler: m.failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(
			completionResponseSchema,
		),
		Ctx:   options.Ctx,
		Fetch: m.config.Fetch,
	})
	if err != nil {
		return languagemodel.GenerateResult{}, err
	}

	response := result.Value
	choice := response.Choices[0]

	var content []languagemodel.Content
	if choice.Text != "" {
		content = append(content, languagemodel.Text{Text: choice.Text})
	}

	meta := GetResponseMetadata(ResponseMetadataInput{
		ID:      response.ID,
		Model:   response.Model,
		Created: response.Created,
	})

	return languagemodel.GenerateResult{
		Content: content,
		Usage:   ConvertCompletionUsage(response.Usage),
		FinishReason: languagemodel.FinishReason{
			Unified: MapFinishReason(&choice.FinishReason),
			Raw:     &choice.FinishReason,
		},
		Request: &languagemodel.GenerateResultRequest{
			Body: prepared.Args,
		},
		Response: &languagemodel.GenerateResultResponse{
			ResponseMetadata: languagemodel.ResponseMetadata{
				ID:        meta.ID,
				ModelID:   meta.ModelID,
				Timestamp: meta.Timestamp,
			},
			Headers: result.ResponseHeaders,
			Body:    result.RawValue,
		},
		Warnings: prepared.Warnings,
	}, nil
}

// DoStream performs a streaming completion call.
func (m *CompletionLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	prepared, err := m.getArgs(options)
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	body := make(map[string]interface{})
	for k, v := range prepared.Args {
		body[k] = v
	}
	body["stream"] = true

	// Only include stream_options when in strict compatibility mode
	if m.config.IncludeUsage != nil && *m.config.IncludeUsage {
		body["stream_options"] = map[string]interface{}{
			"include_usage": true,
		}
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[<-chan providerutils.ParseResult[completionChunk]]{
		URL:     m.config.URL("/completions"),
		Headers: providerutils.CombineHeaders(m.config.Headers(), convertHeaders(options.Headers)),
		Body:    body,
		FailedResponseHandler: m.failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateEventSourceResponseHandler(
			m.chunkSchema,
		),
		Ctx:   options.Ctx,
		Fetch: m.config.Fetch,
	})
	if err != nil {
		return languagemodel.StreamResult{}, err
	}

	includeRawChunks := options.IncludeRawChunks != nil && *options.IncludeRawChunks
	warnings := prepared.Warnings

	outCh := make(chan languagemodel.StreamPart)
	go func() {
		defer close(outCh)

		finishReason := languagemodel.FinishReason{
			Unified: languagemodel.FinishReasonOther,
			Raw:     nil,
		}
		var usage *CompletionUsageRaw
		isFirstChunk := true

		// Send stream-start
		outCh <- languagemodel.StreamPartStreamStart{Warnings: warnings}

		for chunk := range result.Value {
			if includeRawChunks {
				outCh <- languagemodel.StreamPartRaw{RawValue: chunk.RawValue}
			}

			// Handle failed chunk parsing/validation
			if !chunk.Success {
				finishReason = languagemodel.FinishReason{
					Unified: languagemodel.FinishReasonError,
					Raw:     nil,
				}
				outCh <- languagemodel.StreamPartError{Error: chunk.Error}
				continue
			}

			value := chunk.Value

			// Handle error chunks
			if value.ErrorMessage != nil {
				finishReason = languagemodel.FinishReason{
					Unified: languagemodel.FinishReasonError,
					Raw:     nil,
				}
				outCh <- languagemodel.StreamPartError{Error: *value.ErrorMessage}
				continue
			}

			if isFirstChunk {
				isFirstChunk = false

				meta := GetResponseMetadata(ResponseMetadataInput{
					ID:      value.ID,
					Model:   value.Model,
					Created: value.Created,
				})

				outCh <- languagemodel.StreamPartResponseMetadata{
					ResponseMetadata: languagemodel.ResponseMetadata{
						ID:        meta.ID,
						ModelID:   meta.ModelID,
						Timestamp: meta.Timestamp,
					},
				}

				outCh <- languagemodel.StreamPartTextStart{ID: "0"}
			}

			if value.Usage != nil {
				usage = value.Usage
			}

			if len(value.Choices) > 0 {
				choice := value.Choices[0]

				if choice.FinishReason != nil {
					finishReason = languagemodel.FinishReason{
						Unified: MapFinishReason(choice.FinishReason),
						Raw:     choice.FinishReason,
					}
				}

				if choice.Text != nil {
					outCh <- languagemodel.StreamPartTextDelta{
						ID:    "0",
						Delta: *choice.Text,
					}
				}
			}
		}

		if !isFirstChunk {
			outCh <- languagemodel.StreamPartTextEnd{ID: "0"}
		}

		outCh <- languagemodel.StreamPartFinish{
			FinishReason: finishReason,
			Usage:        ConvertCompletionUsage(usage),
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

// --- Response schemas ---

// completionResponse is the parsed response from a non-streaming completion call.
type completionResponse struct {
	ID      *string             `json:"id,omitempty"`
	Created *int64              `json:"created,omitempty"`
	Model   *string             `json:"model,omitempty"`
	Choices []completionChoice  `json:"choices"`
	Usage   *CompletionUsageRaw `json:"usage,omitempty"`
}

type completionChoice struct {
	Text         string `json:"text"`
	FinishReason string `json:"finish_reason"`
}

var completionResponseSchema = &providerutils.Schema[completionResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[completionResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[completionResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var resp completionResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return &providerutils.ValidationResult[completionResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[completionResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}

// completionChunk is a union type covering both valid stream chunks and error chunks.
// In TS this uses z.union([baseSchema, errorSchema]); in Go we parse both and check.
type completionChunk struct {
	// Standard chunk fields
	ID      *string                  `json:"id,omitempty"`
	Created *int64                   `json:"created,omitempty"`
	Model   *string                  `json:"model,omitempty"`
	Choices []completionStreamChoice `json:"choices,omitempty"`
	Usage   *CompletionUsageRaw      `json:"usage,omitempty"`

	// Error chunk field -- set when the response is an error object
	ErrorMessage *string `json:"-"`
}

type completionStreamChoice struct {
	Text         *string `json:"text,omitempty"`
	FinishReason *string `json:"finish_reason,omitempty"`
	Index        int     `json:"index"`
}

// newCompletionChunkSchema creates a schema that can parse both valid completion
// stream chunks and error responses (union type in TS).
func newCompletionChunkSchema(errorSchema *providerutils.Schema[ErrorData]) *providerutils.Schema[completionChunk] {
	return &providerutils.Schema[completionChunk]{
		Validate: func(value interface{}) (*providerutils.ValidationResult[completionChunk], error) {
			data, err := json.Marshal(value)
			if err != nil {
				return &providerutils.ValidationResult[completionChunk]{
					Success: false,
					Error:   err,
				}, nil
			}

			// First try parsing as error
			if errorSchema != nil {
				errorResult, _ := errorSchema.Validate(value)
				if errorResult != nil && errorResult.Success {
					errMsg := errorResult.Value.Error.Message
					return &providerutils.ValidationResult[completionChunk]{
						Success: true,
						Value: completionChunk{
							ErrorMessage: &errMsg,
						},
					}, nil
				}
			}

			// Parse as standard chunk
			var chunk completionChunk
			if err := json.Unmarshal(data, &chunk); err != nil {
				return &providerutils.ValidationResult[completionChunk]{
					Success: false,
					Error:   err,
				}, nil
			}

			return &providerutils.ValidationResult[completionChunk]{
				Success: true,
				Value:   chunk,
			}, nil
		},
	}
}

// convertHeaders converts map[string]*string to map[string]string,
// dropping nil values.
func convertHeaders(headers map[string]*string) map[string]string {
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

// providerOptionsToMap converts shared.ProviderOptions (map[string]jsonvalue.JSONObject)
// to map[string]interface{} for use with providerutils.ParseProviderOptions.
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
