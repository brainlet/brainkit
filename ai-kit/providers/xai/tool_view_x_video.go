// Ported from: packages/xai/src/tool/view-x-video.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// ViewXVideoInput is the input for the view X video tool (empty).
type ViewXVideoInput struct{}

// ViewXVideoOutput is the output of the view X video tool.
type ViewXVideoOutput struct {
	Transcript  *string  `json:"transcript,omitempty"`
	Description string   `json:"description"`
	Duration    *float64 `json:"duration,omitempty"`
}

// viewXVideoToolFactory is the factory for the view X video tool.
var viewXVideoToolFactory = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[ViewXVideoInput, ViewXVideoOutput]{
		ID:           "xai.view_x_video",
		InputSchema:  &providerutils.Schema[ViewXVideoInput]{},
		OutputSchema: &providerutils.Schema[ViewXVideoOutput]{},
	},
)

// ViewXVideo creates a view X video provider tool.
func ViewXVideo(opts ...providerutils.ProviderToolOptions[ViewXVideoInput, ViewXVideoOutput]) providerutils.ProviderTool[ViewXVideoInput, ViewXVideoOutput] {
	var o providerutils.ProviderToolOptions[ViewXVideoInput, ViewXVideoOutput]
	if len(opts) > 0 {
		o = opts[0]
	}
	return viewXVideoToolFactory(o)
}
