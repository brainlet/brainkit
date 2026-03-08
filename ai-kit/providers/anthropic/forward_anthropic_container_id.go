// Ported from: packages/anthropic/src/forward-anthropic-container-id-from-last-step.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"

// StepWithProviderMetadata represents a step that may have provider metadata.
type StepWithProviderMetadata struct {
	ProviderMetadata map[string]jsonvalue.JSONObject
}

// ForwardAnthropicContainerIdFromLastStep sets the Anthropic container ID in the
// provider options based on any previous step's provider metadata.
//
// Searches backwards through steps to find the most recent container ID.
// You can use this function in prepareStep to forward the container ID between steps.
func ForwardAnthropicContainerIdFromLastStep(steps []StepWithProviderMetadata) map[string]jsonvalue.JSONObject {
	// Search backwards through steps to find the most recent container ID
	for i := len(steps) - 1; i >= 0; i-- {
		anthropicMeta, ok := steps[i].ProviderMetadata["anthropic"]
		if !ok || anthropicMeta == nil {
			continue
		}

		containerRaw, ok := anthropicMeta["container"]
		if !ok || containerRaw == nil {
			continue
		}

		containerMap, ok := containerRaw.(map[string]any)
		if !ok {
			continue
		}

		containerID, ok := containerMap["id"].(string)
		if !ok || containerID == "" {
			continue
		}

		return map[string]jsonvalue.JSONObject{
			"anthropic": {
				"container": map[string]any{"id": containerID},
			},
		}
	}

	return nil
}
