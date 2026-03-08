// Ported from: packages/ai/src/telemetry/get-base-telemetry-attributes.ts
package telemetry

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/ai/prompt"
)

// ModelInfo holds the model identification used for telemetry.
type ModelInfo struct {
	ModelID  string
	Provider string
}

// GetBaseTelemetryAttributes builds the base telemetry attributes from model, settings, telemetry config, and headers.
func GetBaseTelemetryAttributes(
	model ModelInfo,
	settings prompt.CallSettingsForTelemetry,
	telemetry *TelemetrySettings,
	headers map[string]string,
) Attributes {
	attrs := Attributes{
		"ai.model.provider": model.Provider,
		"ai.model.id":       model.ModelID,
	}

	// settings
	if settings.MaxOutputTokens != nil {
		attrs["ai.settings.maxOutputTokens"] = *settings.MaxOutputTokens
	}
	if settings.TopP != nil {
		attrs["ai.settings.topP"] = *settings.TopP
	}
	if settings.TopK != nil {
		attrs["ai.settings.topK"] = *settings.TopK
	}
	if settings.PresencePenalty != nil {
		attrs["ai.settings.presencePenalty"] = *settings.PresencePenalty
	}
	if settings.FrequencyPenalty != nil {
		attrs["ai.settings.frequencyPenalty"] = *settings.FrequencyPenalty
	}
	if settings.Seed != nil {
		attrs["ai.settings.seed"] = *settings.Seed
	}
	if settings.MaxRetries != nil {
		attrs["ai.settings.maxRetries"] = *settings.MaxRetries
	}
	if settings.StopSequences != nil {
		attrs["ai.settings.stopSequences"] = fmt.Sprintf("%v", settings.StopSequences)
	}
	if settings.Timeout != nil {
		totalMs := prompt.GetTotalTimeoutMs(settings.Timeout)
		if totalMs != nil {
			attrs["ai.settings.timeout"] = *totalMs
		}
	}

	// add metadata as attributes
	if telemetry != nil && telemetry.Metadata != nil {
		for key, value := range telemetry.Metadata {
			attrs[fmt.Sprintf("ai.telemetry.metadata.%s", key)] = value
		}
	}

	// request headers
	for key, value := range headers {
		attrs[fmt.Sprintf("ai.request.headers.%s", key)] = value
	}

	return attrs
}
