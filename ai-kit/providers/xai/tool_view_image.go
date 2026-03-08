// Ported from: packages/xai/src/tool/view-image.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// ViewImageInput is the input for the view image tool (empty).
type ViewImageInput struct{}

// ViewImageOutput is the output of the view image tool.
type ViewImageOutput struct {
	Description string   `json:"description"`
	Objects     []string `json:"objects,omitempty"`
}

// viewImageToolFactory is the factory for the view image tool.
var viewImageToolFactory = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[ViewImageInput, ViewImageOutput]{
		ID:           "xai.view_image",
		InputSchema:  &providerutils.Schema[ViewImageInput]{},
		OutputSchema: &providerutils.Schema[ViewImageOutput]{},
	},
)

// ViewImage creates a view image provider tool.
func ViewImage(opts ...providerutils.ProviderToolOptions[ViewImageInput, ViewImageOutput]) providerutils.ProviderTool[ViewImageInput, ViewImageOutput] {
	var o providerutils.ProviderToolOptions[ViewImageInput, ViewImageOutput]
	if len(opts) > 0 {
		o = opts[0]
	}
	return viewImageToolFactory(o)
}
