// Ported from: packages/openai/src/tool/local-shell.ts
package openai

// LocalShellAction represents the exec action for the local_shell tool.
type LocalShellAction struct {
	// Type is always "exec".
	Type string `json:"type"`

	// Command is the command to run.
	Command []string `json:"command"`

	// TimeoutMs is an optional timeout in milliseconds for the command.
	TimeoutMs *int `json:"timeoutMs,omitempty"`

	// User is an optional user to run the command as.
	User string `json:"user,omitempty"`

	// WorkingDirectory is an optional working directory to run the command in.
	WorkingDirectory string `json:"workingDirectory,omitempty"`

	// Env is environment variables to set for the command.
	Env map[string]string `json:"env,omitempty"`
}

// LocalShellInput is the input schema for the local_shell tool.
type LocalShellInput struct {
	// Action is the shell execution action.
	Action LocalShellAction `json:"action"`
}

// LocalShellOutput is the output schema for the local_shell tool.
type LocalShellOutput struct {
	// Output is the output of the local shell tool call.
	Output string `json:"output"`
}

// LocalShellToolID is the provider tool ID for local_shell.
const LocalShellToolID = "openai.local_shell"

// NewLocalShellTool creates a provider tool configuration for the local_shell tool.
func NewLocalShellTool() map[string]interface{} {
	return map[string]interface{}{
		"type": "provider",
		"id":   LocalShellToolID,
	}
}
