// Ported from: packages/ai/src/prompt/call-settings.ts
package prompt

// TimeoutConfiguration represents timeout configuration for API calls.
// It can be a simple number (milliseconds) or a structured configuration.
type TimeoutConfiguration struct {
	// TotalMs is the total timeout in milliseconds (also used for simple number config).
	TotalMs *int
	// StepMs is the timeout for each step in milliseconds.
	StepMs *int
	// ChunkMs is the timeout between stream chunks (streaming only).
	ChunkMs *int
}

// NewSimpleTimeout creates a TimeoutConfiguration from a simple millisecond value.
func NewSimpleTimeout(ms int) *TimeoutConfiguration {
	return &TimeoutConfiguration{TotalMs: &ms}
}

// GetTotalTimeoutMs extracts the total timeout value in milliseconds from a TimeoutConfiguration.
func GetTotalTimeoutMs(timeout *TimeoutConfiguration) *int {
	if timeout == nil {
		return nil
	}
	return timeout.TotalMs
}

// GetStepTimeoutMs extracts the step timeout value in milliseconds from a TimeoutConfiguration.
func GetStepTimeoutMs(timeout *TimeoutConfiguration) *int {
	if timeout == nil {
		return nil
	}
	return timeout.StepMs
}

// GetChunkTimeoutMs extracts the chunk timeout value in milliseconds from a TimeoutConfiguration.
// This timeout is for streaming only - it aborts if no new chunk is received within the specified duration.
func GetChunkTimeoutMs(timeout *TimeoutConfiguration) *int {
	if timeout == nil {
		return nil
	}
	return timeout.ChunkMs
}

// CallSettings contains settings for AI model calls.
type CallSettings struct {
	// MaxOutputTokens is the maximum number of tokens to generate.
	MaxOutputTokens *int

	// Temperature setting. The range depends on the provider and model.
	// It is recommended to set either Temperature or TopP, but not both.
	Temperature *float64

	// TopP is nucleus sampling. This is a number between 0 and 1.
	// It is recommended to set either Temperature or TopP, but not both.
	TopP *float64

	// TopK samples from the top K options for each subsequent token.
	TopK *int

	// PresencePenalty affects the likelihood of the model to repeat information
	// that is already in the prompt. Range: -1 to 1.
	PresencePenalty *float64

	// FrequencyPenalty affects the likelihood of the model to repeatedly use
	// the same words or phrases. Range: -1 to 1.
	FrequencyPenalty *float64

	// StopSequences causes the model to stop generating when one is produced.
	StopSequences []string

	// Seed for random sampling. If set and supported, calls generate deterministic results.
	Seed *int

	// MaxRetries is the maximum number of retries. Set to 0 to disable retries.
	// Default: 2.
	MaxRetries *int

	// Timeout configuration for the call.
	Timeout *TimeoutConfiguration

	// Headers are additional HTTP headers to be sent with the request.
	Headers map[string]string
}

// CallSettingsForTelemetry is a subset of CallSettings used for telemetry attributes.
// It excludes AbortSignal and Headers (which are handled separately).
type CallSettingsForTelemetry struct {
	MaxOutputTokens  *int
	TopP             *float64
	TopK             *int
	PresencePenalty  *float64
	FrequencyPenalty *float64
	StopSequences    []string
	Seed             *int
	MaxRetries       *int
	Timeout          *TimeoutConfiguration
}
