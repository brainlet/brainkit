// Ported from: packages/openai/src/speech/openai-speech-model.ts
package openai

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAISpeechModelConfig extends OpenAIConfig with internal options.
type OpenAISpeechModelConfig struct {
	OpenAIConfig

	// Internal contains optional internal configuration.
	Internal *OpenAISpeechModelInternal
}

// OpenAISpeechModelInternal contains internal options for OpenAISpeechModel.
type OpenAISpeechModelInternal struct {
	// CurrentDate returns the current date for testing purposes.
	CurrentDate func() time.Time
}

// OpenAISpeechModel implements speechmodel.SpeechModel for the OpenAI
// text-to-speech endpoint.
type OpenAISpeechModel struct {
	modelID OpenAISpeechModelID
	config  OpenAISpeechModelConfig
}

// NewOpenAISpeechModel creates a new OpenAISpeechModel.
func NewOpenAISpeechModel(modelID OpenAISpeechModelID, config OpenAISpeechModelConfig) *OpenAISpeechModel {
	return &OpenAISpeechModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *OpenAISpeechModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *OpenAISpeechModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *OpenAISpeechModel) ModelID() string { return m.modelID }

// supportedOutputFormats lists the audio formats supported by OpenAI speech.
var supportedOutputFormats = map[string]bool{
	"mp3":  true,
	"opus": true,
	"aac":  true,
	"flac": true,
	"wav":  true,
	"pcm":  true,
}

// getArgs prepares the request body and warnings for a speech generation call.
func (m *OpenAISpeechModel) getArgs(options speechmodel.CallOptions) (map[string]interface{}, []shared.Warning, error) {
	var warnings []shared.Warning

	// Parse provider options
	providerOpts := providerOptionsToMapSpeech(options.ProviderOptions)
	_, err := providerutils.ParseProviderOptions(
		"openai",
		providerOpts,
		openaiSpeechModelOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}

	// Determine voice
	voice := "alloy"
	if options.Voice != nil {
		voice = *options.Voice
	}

	// Determine output format
	outputFormat := "mp3"
	if options.OutputFormat != nil {
		if supportedOutputFormats[*options.OutputFormat] {
			outputFormat = *options.OutputFormat
		} else {
			details := fmt.Sprintf("Unsupported output format: %s. Using mp3 instead.", *options.OutputFormat)
			warnings = append(warnings, shared.UnsupportedWarning{
				Feature: "outputFormat",
				Details: &details,
			})
		}
	}

	// Create request body
	requestBody := map[string]interface{}{
		"model":           m.modelID,
		"input":           options.Text,
		"voice":           voice,
		"response_format": outputFormat,
	}

	if options.Speed != nil {
		requestBody["speed"] = *options.Speed
	}

	if options.Instructions != nil {
		requestBody["instructions"] = *options.Instructions
	}

	// Language is not supported by OpenAI speech models
	if options.Language != nil {
		details := fmt.Sprintf(
			"OpenAI speech models do not support language selection. Language parameter %q was ignored.",
			*options.Language,
		)
		warnings = append(warnings, shared.UnsupportedWarning{
			Feature: "language",
			Details: &details,
		})
	}

	return requestBody, warnings, nil
}

// DoGenerate generates speech audio from text.
func (m *OpenAISpeechModel) DoGenerate(options speechmodel.CallOptions) (speechmodel.GenerateResult, error) {
	currentDate := time.Now()
	if m.config.Internal != nil && m.config.Internal.CurrentDate != nil {
		currentDate = m.config.Internal.CurrentDate()
	}

	requestBody, warnings, err := m.getArgs(options)
	if err != nil {
		return speechmodel.GenerateResult{}, err
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[[]byte]{
		URL:                       m.config.URL(struct{ ModelID string; Path string }{Path: "/audio/speech", ModelID: m.modelID}),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers)),
		Body:                      requestBody,
		FailedResponseHandler:     openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateBinaryResponseHandler(),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return speechmodel.GenerateResult{}, err
	}

	bodyJSON, _ := json.Marshal(requestBody)
	bodyStr := string(bodyJSON)

	return speechmodel.GenerateResult{
		Audio:    speechmodel.AudioDataBytes{Data: result.Value},
		Warnings: warnings,
		Request: &speechmodel.GenerateResultRequest{
			Body: bodyStr,
		},
		Response: speechmodel.GenerateResultResponse{
			Timestamp: currentDate,
			ModelID:   m.modelID,
			Headers:   result.ResponseHeaders,
			Body:      result.RawValue,
		},
	}, nil
}

// providerOptionsToMapSpeech converts speechmodel.ProviderOptions to a plain map.
func providerOptionsToMapSpeech(opts speechmodel.ProviderOptions) map[string]interface{} {
	if opts == nil {
		return nil
	}
	result := make(map[string]interface{}, len(opts))
	for k, v := range opts {
		result[k] = v
	}
	return result
}
