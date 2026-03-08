// Ported from: packages/core/src/workspace/tools/types.ts
package tools

// =============================================================================
// Tool Configuration Types
// =============================================================================

// WorkspaceToolConfig configures a single workspace tool.
type WorkspaceToolConfig struct {
	// Enabled controls whether the tool is available (default: true).
	Enabled *bool `json:"enabled,omitempty"`
	// RequireApproval controls whether the tool requires user approval (default: false).
	RequireApproval *bool `json:"requireApproval,omitempty"`
	// RequireReadBeforeWrite controls whether write tools require a read first.
	RequireReadBeforeWrite *bool `json:"requireReadBeforeWrite,omitempty"`
	// MaxOutputTokens limits the output sent to the model.
	MaxOutputTokens *int `json:"maxOutputTokens,omitempty"`
	// Name overrides the tool name exposed to the model.
	Name *string `json:"name,omitempty"`
	// BackgroundProcesses configures background process behavior (execute_command only).
	BackgroundProcesses *BackgroundProcessConfig `json:"backgroundProcesses,omitempty"`
}

// BackgroundProcessConfig configures background process behavior for the execute_command tool.
type BackgroundProcessConfig struct {
	// AbortSignal controls process cancellation. nil = use context signal, false = disabled.
	AbortSignal interface{} `json:"abortSignal,omitempty"`
	// OnStdout callback for background process stdout data.
	OnStdout func(data string, meta BackgroundProcessMeta) `json:"-"`
	// OnStderr callback for background process stderr data.
	OnStderr func(data string, meta BackgroundProcessMeta) `json:"-"`
	// OnExit callback when background process exits.
	OnExit func(meta BackgroundProcessExitMeta) `json:"-"`
}

// BackgroundProcessMeta holds metadata for background process callbacks.
type BackgroundProcessMeta struct {
	// PID is the process ID.
	PID int `json:"pid"`
	// ToolCallID is the tool call that spawned the process.
	ToolCallID string `json:"toolCallId,omitempty"`
}

// BackgroundProcessExitMeta holds metadata for background process exit callbacks.
type BackgroundProcessExitMeta struct {
	// PID is the process ID.
	PID int `json:"pid"`
	// ExitCode is the process exit code.
	ExitCode int `json:"exitCode"`
	// Stdout is the full stdout output.
	Stdout string `json:"stdout"`
	// Stderr is the full stderr output.
	Stderr string `json:"stderr"`
	// ToolCallID is the tool call that spawned the process.
	ToolCallID string `json:"toolCallId,omitempty"`
}

// WorkspaceToolsConfig holds per-tool configuration for all workspace tools.
// Keys are tool name constants (e.g., constants.WorkspaceTools.Filesystem.ReadFile).
type WorkspaceToolsConfig struct {
	// Enabled controls whether all tools are available (default: true).
	// Per-tool Enabled overrides this.
	Enabled *bool `json:"enabled,omitempty"`
	// RequireApproval controls whether all tools require approval (default: false).
	// Per-tool RequireApproval overrides this.
	RequireApproval *bool `json:"requireApproval,omitempty"`
	// PerTool holds per-tool configuration keyed by tool name.
	PerTool map[string]*WorkspaceToolConfig `json:"perTool,omitempty"`
}

// GetToolConfig returns the configuration for a specific tool, or nil if not set.
func (c *WorkspaceToolsConfig) GetToolConfig(toolName string) *WorkspaceToolConfig {
	if c == nil || c.PerTool == nil {
		return nil
	}
	return c.PerTool[toolName]
}
