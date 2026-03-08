// Ported from: packages/core/src/workspace/tools/get-process-output.ts
package tools

import (
	"fmt"
	"strings"
)

// =============================================================================
// Get Process Output Tool
// =============================================================================

// GetProcessOutputInput holds the input for the get_process_output tool.
type GetProcessOutputInput struct {
	// PID is the process ID returned when the background command was started.
	PID int `json:"pid"`
	// Tail is the number of lines to return (default: DefaultTailLines). Use 0 for no limit.
	Tail *int `json:"tail,omitempty"`
	// Wait blocks until the process exits if true.
	Wait bool `json:"wait,omitempty"`
}

// ExecuteGetProcessOutput executes the get_process_output tool.
func ExecuteGetProcessOutput(input *GetProcessOutputInput, ctx *ToolContext) (string, error) {
	result, err := RequireSandbox(ctx)
	if err != nil {
		return "", err
	}

	ws := result.Workspace
	sb := result.Sandbox

	processes := sb.Processes()
	if processes == nil {
		return "", fmt.Errorf("sandbox does not support processes")
	}

	handle, err := processes.Get(input.PID)
	if err != nil {
		return "", err
	}
	if handle == nil {
		return fmt.Sprintf("No background process found with PID %d.", input.PID), nil
	}

	// Emit command info
	if ctx != nil && ctx.Writer != nil && handle.Command != "" {
		_ = ctx.Writer.Custom(CustomEvent{
			Type: "data-sandbox-command",
			Data: map[string]interface{}{
				"command":    handle.Command,
				"pid":        input.PID,
				"toolCallId": ctx.ToolCallID,
			},
		})
	}

	// If wait requested, block until process exits
	if input.Wait && handle.ExitCode == nil {
		waitResult, err := handle.Wait(nil)
		if err != nil {
			return "", err
		}

		if ctx != nil && ctx.Writer != nil {
			_ = ctx.Writer.Custom(CustomEvent{
				Type: "data-sandbox-exit",
				Data: map[string]interface{}{
					"exitCode":        waitResult.ExitCode,
					"success":         waitResult.Success,
					"executionTimeMs": waitResult.ExecutionTimeMs,
					"toolCallId":      ctx.ToolCallID,
				},
			})
		}
	}

	running := handle.ExitCode == nil

	// Get token limit from config
	var tokenLimit *int
	toolsConfig := ws.GetToolsConfig()
	if toolsConfig != nil {
		tc := toolsConfig.GetToolConfig("mastra_workspace_get_process_output")
		if tc != nil {
			tokenLimit = tc.MaxOutputTokens
		}
	}

	stdout := TruncateOutput(handle.Stdout, input.Tail, tokenLimit, "sandwich")
	stderr := TruncateOutput(handle.Stderr, input.Tail, tokenLimit, "sandwich")

	if stdout == "" && stderr == "" {
		return "(no output yet)", nil
	}

	var parts []string
	if stdout != "" && stderr != "" {
		parts = append(parts, "stdout:", stdout, "", "stderr:", stderr)
	} else if stdout != "" {
		parts = append(parts, stdout)
	} else {
		parts = append(parts, "stderr:", stderr)
	}

	if !running {
		exitCode := 0
		if handle.ExitCode != nil {
			exitCode = *handle.ExitCode
		}
		parts = append(parts, "", fmt.Sprintf("Exit code: %d", exitCode))
	}

	return strings.Join(parts, "\n"), nil
}
