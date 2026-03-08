// Ported from: packages/xai/src/tool/code-execution.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// CodeExecutionInput is the input for the code execution tool (empty).
type CodeExecutionInput struct{}

// CodeExecutionOutput is the output of the code execution tool.
type CodeExecutionOutput struct {
	Output string  `json:"output"`
	Error  *string `json:"error,omitempty"`
}

// codeExecutionToolFactory is the factory for the code execution tool.
var codeExecutionToolFactory = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[CodeExecutionInput, CodeExecutionOutput]{
		ID:           "xai.code_execution",
		InputSchema:  &providerutils.Schema[CodeExecutionInput]{},
		OutputSchema: &providerutils.Schema[CodeExecutionOutput]{},
	},
)

// CodeExecution creates a code execution provider tool.
func CodeExecution(opts ...providerutils.ProviderToolOptions[CodeExecutionInput, CodeExecutionOutput]) providerutils.ProviderTool[CodeExecutionInput, CodeExecutionOutput] {
	var o providerutils.ProviderToolOptions[CodeExecutionInput, CodeExecutionOutput]
	if len(opts) > 0 {
		o = opts[0]
	}
	return codeExecutionToolFactory(o)
}
