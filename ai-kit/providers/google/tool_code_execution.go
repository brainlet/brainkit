// Ported from: packages/google/src/tool/code-execution.ts
package google

// CodeExecutionToolID is the tool ID for code execution.
const CodeExecutionToolID = "google.code_execution"

// CodeExecutionInput is the input for the code execution tool.
type CodeExecutionInput struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

// CodeExecutionOutput is the output from the code execution tool.
type CodeExecutionOutput struct {
	Outcome string `json:"outcome"`
	Output  string `json:"output"`
}
