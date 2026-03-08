// Ported from: packages/groq/src/groq-transcription-model.ts
package groq

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GroqTranscriptionModelConfig extends GroqConfig with internal options.
type GroqTranscriptionModelConfig struct {
	GroqConfig

	// Internal options for testing.
	Internal *GroqTranscriptionModelInternal
}

// GroqTranscriptionModelInternal holds internal options.
type GroqTranscriptionModelInternal struct {
	CurrentDate func() time.Time
}

// GroqTranscriptionModel implements transcriptionmodel.TranscriptionModel for Groq APIs.
type GroqTranscriptionModel struct {
	modelID string
	config  GroqTranscriptionModelConfig
}

// NewGroqTranscriptionModel creates a new GroqTranscriptionModel.
func NewGroqTranscriptionModel(modelID GroqTranscriptionModelId, config GroqTranscriptionModelConfig) *GroqTranscriptionModel {
	return &GroqTranscriptionModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *GroqTranscriptionModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *GroqTranscriptionModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *GroqTranscriptionModel) ModelID() string { return m.modelID }

// getArgs prepares the request arguments from transcription CallOptions.
func (m *GroqTranscriptionModel) getArgs(options transcriptionmodel.CallOptions) (formData map[string]interface{}, warnings []shared.Warning, err error) {
	warnings = []shared.Warning{}

	// Convert ProviderOptions to map[string]interface{} for ParseProviderOptions
	providerOptsMap := make(map[string]interface{})
	for k, v := range options.ProviderOptions {
		providerOptsMap[k] = v
	}

	groqOptions, err := providerutils.ParseProviderOptions(
		"groq",
		providerOptsMap,
		GroqTranscriptionModelOptionsSchema,
	)
	if err != nil {
		return nil, nil, err
	}

	// Convert audio data to bytes
	var audioBytes []byte
	switch a := options.Audio.(type) {
	case transcriptionmodel.AudioDataBytes:
		audioBytes = a.Data
	case transcriptionmodel.AudioDataString:
		decoded, decErr := base64.StdEncoding.DecodeString(a.Value)
		if decErr != nil {
			return nil, nil, fmt.Errorf("failed to decode base64 audio: %w", decErr)
		}
		audioBytes = decoded
	default:
		return nil, nil, fmt.Errorf("unsupported audio data type: %T", options.Audio)
	}

	fileExtension := providerutils.MediaTypeToExtension(options.MediaType)

	formData = map[string]interface{}{
		"model":    m.modelID,
		"file":     audioBytes,
		"filename": fmt.Sprintf("audio.%s", fileExtension),
	}

	// Add provider-specific options
	if groqOptions != nil {
		if groqOptions.Language != nil {
			formData["language"] = *groqOptions.Language
		}
		if groqOptions.Prompt != nil {
			formData["prompt"] = *groqOptions.Prompt
		}
		if groqOptions.ResponseFormat != nil {
			formData["response_format"] = *groqOptions.ResponseFormat
		}
		if groqOptions.Temperature != nil {
			formData["temperature"] = fmt.Sprintf("%v", *groqOptions.Temperature)
		}
		if groqOptions.TimestampGranularities != nil {
			for _, item := range groqOptions.TimestampGranularities {
				formData["timestamp_granularities[]"] = item
			}
		}
	}

	return formData, warnings, nil
}

// DoGenerate implements transcriptionmodel.TranscriptionModel.DoGenerate.
func (m *GroqTranscriptionModel) DoGenerate(options transcriptionmodel.CallOptions) (transcriptionmodel.GenerateResult, error) {
	currentDate := time.Now()
	if m.config.Internal != nil && m.config.Internal.CurrentDate != nil {
		currentDate = m.config.Internal.CurrentDate()
	}

	formDataMap, warnings, err := m.getArgs(options)
	if err != nil {
		return transcriptionmodel.GenerateResult{}, err
	}

	formResult, err := providerutils.ConvertToFormData(formDataMap, nil)
	if err != nil {
		return transcriptionmodel.GenerateResult{}, fmt.Errorf("failed to create form data: %w", err)
	}

	headers := providerutils.CombineHeaders(m.config.Headers(), convertOptionalHeaders(options.Headers))
	headers["Content-Type"] = formResult.ContentType

	result, err := providerutils.PostToApi(providerutils.PostToApiOptions[groqTranscriptionResponse]{
		URL:     m.config.URL(m.modelID, "/audio/transcriptions"),
		Headers: headers,
		Body: providerutils.PostToApiBody{
			Content: formResult.Body,
			Values:  formResult.Values,
		},
		FailedResponseHandler:     groqFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(groqTranscriptionResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return transcriptionmodel.GenerateResult{}, err
	}

	response := result.Value

	segments := []transcriptionmodel.Segment{}
	if response.Segments != nil {
		for _, seg := range response.Segments {
			segments = append(segments, transcriptionmodel.Segment{
				Text:        seg.Text,
				StartSecond: seg.Start,
				EndSecond:   seg.End,
			})
		}
	}

	return transcriptionmodel.GenerateResult{
		Text:              response.Text,
		Segments:          segments,
		Language:          response.Language,
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

// --- Response schema ---

// groqTranscriptionResponse represents the Groq transcription API response.
type groqTranscriptionResponse struct {
	Text   string `json:"text"`
	XGroq  struct {
		ID string `json:"id"`
	} `json:"x_groq"`
	Task     *string                         `json:"task,omitempty"`
	Language *string                         `json:"language,omitempty"`
	Duration *float64                        `json:"duration,omitempty"`
	Segments []groqTranscriptionSegment      `json:"segments,omitempty"`
}

// groqTranscriptionSegment represents a segment in the transcription response.
type groqTranscriptionSegment struct {
	ID               int       `json:"id"`
	Seek             int       `json:"seek"`
	Start            float64   `json:"start"`
	End              float64   `json:"end"`
	Text             string    `json:"text"`
	Tokens           []int     `json:"tokens"`
	Temperature      float64   `json:"temperature"`
	AvgLogprob       float64   `json:"avg_logprob"`
	CompressionRatio float64   `json:"compression_ratio"`
	NoSpeechProb     float64   `json:"no_speech_prob"`
}

// groqTranscriptionResponseSchema is the schema for transcription responses.
var groqTranscriptionResponseSchema = &providerutils.Schema[groqTranscriptionResponse]{}

// Verify GroqTranscriptionModel implements the TranscriptionModel interface.
var _ transcriptionmodel.TranscriptionModel = (*GroqTranscriptionModel)(nil)
