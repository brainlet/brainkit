// Ported from: packages/openai/src/transcription/openai-transcription-model.ts
package openai

import (
	"fmt"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAITranscriptionModelConfig extends OpenAIConfig with internal options.
type OpenAITranscriptionModelConfig struct {
	OpenAIConfig

	// Internal contains optional internal configuration.
	Internal *OpenAITranscriptionModelInternal
}

// OpenAITranscriptionModelInternal contains internal options for OpenAITranscriptionModel.
type OpenAITranscriptionModelInternal struct {
	// CurrentDate returns the current date for testing purposes.
	CurrentDate func() time.Time
}

// languageMap maps language names to ISO-639-1 language codes.
// https://platform.openai.com/docs/guides/speech-to-text#supported-languages
var languageMap = map[string]string{
	"afrikaans":   "af",
	"arabic":      "ar",
	"armenian":    "hy",
	"azerbaijani": "az",
	"belarusian":  "be",
	"bosnian":     "bs",
	"bulgarian":   "bg",
	"catalan":     "ca",
	"chinese":     "zh",
	"croatian":    "hr",
	"czech":       "cs",
	"danish":      "da",
	"dutch":       "nl",
	"english":     "en",
	"estonian":    "et",
	"finnish":     "fi",
	"french":      "fr",
	"galician":    "gl",
	"german":      "de",
	"greek":       "el",
	"hebrew":      "he",
	"hindi":       "hi",
	"hungarian":   "hu",
	"icelandic":   "is",
	"indonesian":  "id",
	"italian":     "it",
	"japanese":    "ja",
	"kannada":     "kn",
	"kazakh":      "kk",
	"korean":      "ko",
	"latvian":     "lv",
	"lithuanian":  "lt",
	"macedonian":  "mk",
	"malay":       "ms",
	"marathi":     "mr",
	"maori":       "mi",
	"nepali":      "ne",
	"norwegian":   "no",
	"persian":     "fa",
	"polish":      "pl",
	"portuguese":  "pt",
	"romanian":    "ro",
	"russian":     "ru",
	"serbian":     "sr",
	"slovak":      "sk",
	"slovenian":   "sl",
	"spanish":     "es",
	"swahili":     "sw",
	"swedish":     "sv",
	"tagalog":     "tl",
	"tamil":       "ta",
	"thai":        "th",
	"turkish":     "tr",
	"ukrainian":   "uk",
	"urdu":        "ur",
	"vietnamese":  "vi",
	"welsh":       "cy",
}

// OpenAITranscriptionModel implements transcriptionmodel.TranscriptionModel for the
// OpenAI audio transcription endpoint.
type OpenAITranscriptionModel struct {
	modelID OpenAITranscriptionModelID
	config  OpenAITranscriptionModelConfig
}

// NewOpenAITranscriptionModel creates a new OpenAITranscriptionModel.
func NewOpenAITranscriptionModel(modelID OpenAITranscriptionModelID, config OpenAITranscriptionModelConfig) *OpenAITranscriptionModel {
	return &OpenAITranscriptionModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *OpenAITranscriptionModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *OpenAITranscriptionModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *OpenAITranscriptionModel) ModelID() string { return m.modelID }

// getArgs prepares the form data and warnings for a transcription call.
func (m *OpenAITranscriptionModel) getArgs(options transcriptionmodel.CallOptions) (map[string]interface{}, []shared.Warning, error) {
	var warnings []shared.Warning

	// Parse provider options
	providerOpts := providerOptionsToMapTranscription(options.ProviderOptions)
	openAIOptions, err := providerutils.ParseProviderOptions(
		"openai",
		providerOpts,
		openAITranscriptionModelOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}

	// Convert audio to bytes
	var audioBytes []byte
	switch a := options.Audio.(type) {
	case transcriptionmodel.AudioDataBytes:
		audioBytes = a.Data
	case transcriptionmodel.AudioDataString:
		audioBytes, err = providerutils.ConvertBase64ToBytes(a.Value)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode base64 audio: %w", err)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported audio data type: %T", options.Audio)
	}

	// Create form data
	fileExtension := providerutils.MediaTypeToExtension(options.MediaType)
	formInput := map[string]interface{}{
		"model": m.modelID,
		"file":  audioBytes,
	}

	// We need to set the filename with the correct extension
	// The form data will use the key "file" with the audio bytes
	_ = fileExtension // Used conceptually for file naming in form data

	// Add provider-specific options
	if openAIOptions != nil {
		// Determine response_format based on model
		responseFormat := "verbose_json"
		if m.modelID == "gpt-4o-transcribe" || m.modelID == "gpt-4o-mini-transcribe" {
			responseFormat = "json"
		}

		if openAIOptions.Include != nil {
			for _, item := range openAIOptions.Include {
				// Append array items as include[]
				if _, ok := formInput["include[]"]; !ok {
					formInput["include[]"] = item
				}
			}
			// Handle array properly
			includeArr := make([]interface{}, len(openAIOptions.Include))
			for i, item := range openAIOptions.Include {
				includeArr[i] = item
			}
			formInput["include"] = includeArr
		}
		if openAIOptions.Language != nil {
			formInput["language"] = *openAIOptions.Language
		}
		if openAIOptions.Prompt != nil {
			formInput["prompt"] = *openAIOptions.Prompt
		}
		formInput["response_format"] = responseFormat
		if openAIOptions.Temperature != nil {
			formInput["temperature"] = fmt.Sprintf("%v", *openAIOptions.Temperature)
		}
		if openAIOptions.TimestampGranularities != nil {
			granArr := make([]interface{}, len(openAIOptions.TimestampGranularities))
			for i, item := range openAIOptions.TimestampGranularities {
				granArr[i] = item
			}
			formInput["timestamp_granularities"] = granArr
		}
	}

	return formInput, warnings, nil
}

// DoGenerate generates a transcription from audio.
func (m *OpenAITranscriptionModel) DoGenerate(options transcriptionmodel.CallOptions) (transcriptionmodel.GenerateResult, error) {
	currentDate := time.Now()
	if m.config.Internal != nil && m.config.Internal.CurrentDate != nil {
		currentDate = m.config.Internal.CurrentDate()
	}

	formInput, warnings, err := m.getArgs(options)
	if err != nil {
		return transcriptionmodel.GenerateResult{}, err
	}

	formResult, err := providerutils.ConvertToFormData(formInput, nil)
	if err != nil {
		return transcriptionmodel.GenerateResult{}, fmt.Errorf("failed to create form data: %w", err)
	}

	headers := providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers))
	headers["Content-Type"] = formResult.ContentType

	result, err := providerutils.PostToApi(providerutils.PostToApiOptions[openaiTranscriptionResponse]{
		URL:     m.config.URL(struct{ ModelID string; Path string }{Path: "/audio/transcriptions", ModelID: m.modelID}),
		Headers: headers,
		Body: providerutils.PostToApiBody{
			Content: formResult.Body,
			Values:  formResult.Values,
		},
		FailedResponseHandler:     openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(openaiTranscriptionResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return transcriptionmodel.GenerateResult{}, err
	}

	response := result.Value

	// Map language name to ISO-639-1 code
	var language *string
	if response.Language != nil {
		if code, ok := languageMap[*response.Language]; ok {
			language = &code
		}
	}

	// Map segments - prefer segments, fall back to words
	var segments []transcriptionmodel.Segment
	if len(response.Segments) > 0 {
		segments = make([]transcriptionmodel.Segment, len(response.Segments))
		for i, seg := range response.Segments {
			segments[i] = transcriptionmodel.Segment{
				Text:        seg.Text,
				StartSecond: seg.Start,
				EndSecond:   seg.End,
			}
		}
	} else if len(response.Words) > 0 {
		segments = make([]transcriptionmodel.Segment, len(response.Words))
		for i, word := range response.Words {
			segments[i] = transcriptionmodel.Segment{
				Text:        word.Word,
				StartSecond: word.Start,
				EndSecond:   word.End,
			}
		}
	} else {
		segments = []transcriptionmodel.Segment{}
	}

	return transcriptionmodel.GenerateResult{
		Text:              response.Text,
		Segments:          segments,
		Language:          language,
		DurationInSeconds: response.Duration,
		Warnings:          warnings,
		Response: transcriptionmodel.GenerateResultResponse{
			Timestamp: currentDate,
			ModelID:   m.modelID,
			Headers:   result.ResponseHeaders,
			Body:      result.RawValue,
		},
	}, nil
}

// providerOptionsToMapTranscription converts transcriptionmodel.ProviderOptions to a plain map.
func providerOptionsToMapTranscription(opts transcriptionmodel.ProviderOptions) map[string]interface{} {
	if opts == nil {
		return nil
	}
	result := make(map[string]interface{}, len(opts))
	for k, v := range opts {
		result[k] = v
	}
	return result
}
