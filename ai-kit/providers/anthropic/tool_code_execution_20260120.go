// Ported from: packages/anthropic/src/tool/code-execution_20260120.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// CodeExecution20260120Input is the input schema for the code_execution_20260120 tool.
// This is a discriminated union; the Type field indicates the variant.
type CodeExecution20260120Input struct {
	Type     string  `json:"type"`
	Code     *string `json:"code,omitempty"`
	Command  *string `json:"command,omitempty"`
	Path     *string `json:"path,omitempty"`
	FileText *string `json:"file_text,omitempty"`
	OldStr   *string `json:"old_str,omitempty"`
	NewStr   *string `json:"new_str,omitempty"`
}

// CodeExecution20260120Output is the output schema for the code_execution_20260120 tool.
// This is a discriminated union; the Type field indicates the variant.
type CodeExecution20260120Output struct {
	Type             string   `json:"type"`
	Stdout           *string  `json:"stdout,omitempty"`
	EncryptedStdout  *string  `json:"encrypted_stdout,omitempty"`
	Stderr           *string  `json:"stderr,omitempty"`
	ReturnCode       *int     `json:"return_code,omitempty"`
	Content          any      `json:"content,omitempty"`
	ErrorCode        *string  `json:"error_code,omitempty"`
	FileType         *string  `json:"file_type,omitempty"`
	NumLines         *int     `json:"num_lines,omitempty"`
	StartLine        *int     `json:"start_line,omitempty"`
	TotalLines       *int     `json:"total_lines,omitempty"`
	IsFileUpdate     *bool    `json:"is_file_update,omitempty"`
	Lines            []string `json:"lines,omitempty"`
	NewLines         *int     `json:"new_lines,omitempty"`
	NewStart         *int     `json:"new_start,omitempty"`
	OldLines         *int     `json:"old_lines,omitempty"`
	OldStart         *int     `json:"old_start,omitempty"`
}

// CodeExecution20260120 is the provider tool factory for the code_execution_20260120 tool.
var CodeExecution20260120 = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[CodeExecution20260120Input, CodeExecution20260120Output]{
		ID:                      "anthropic.code_execution_20260120",
		SupportsDeferredResults: true,
	},
)
