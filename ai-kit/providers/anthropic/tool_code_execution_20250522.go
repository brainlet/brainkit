// Ported from: packages/anthropic/src/tool/code-execution_20250522.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// CodeExecution20250522Input is the input schema for the code_execution_20250522 tool.
type CodeExecution20250522Input struct {
	Code string `json:"code"`
}

// CodeExecution20250522Output is the output schema for the code_execution_20250522 tool.
type CodeExecution20250522Output struct {
	Type       string `json:"type"` // "code_execution_result"
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ReturnCode int    `json:"return_code"`
	Content    []struct {
		Type   string `json:"type"` // "code_execution_output"
		FileID string `json:"file_id"`
	} `json:"content"`
}

// CodeExecution20250522 is the provider tool factory for the code_execution_20250522 tool.
var CodeExecution20250522 = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[CodeExecution20250522Input, CodeExecution20250522Output]{
		ID: "anthropic.code_execution_20250522",
	},
)
