// Ported from: packages/core/src/workspace/tools/kill-process.ts
package tools

import (
	"fmt"
	"strings"
)

// =============================================================================
// Kill Process Tool
// =============================================================================

// KillTailLines is the number of tail lines to show after killing a process.
const KillTailLines = 50

// KillProcessInput holds the input for the kill_process tool.
type KillProcessInput struct {
	// PID is the process ID of the background process to kill.
	PID int `json:"pid"`
}

// ExecuteKillProcess executes the kill_process tool.
func ExecuteKillProcess(input *KillProcessInput, ctx *ToolContext) (string, error) {
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

	// Snapshot output before kill
	handle, err := processes.Get(input.PID)
	if err != nil {
		return "", err
	}

	// Emit command info
	if handle != nil && handle.Command != "" && ctx != nil && ctx.Writer != nil {
		_ = ctx.Writer.Custom(CustomEvent{
			Type: "data-sandbox-command",
			Data: map[string]interface{}{
				"command":    handle.Command,
				"pid":        input.PID,
				"toolCallId": ctx.ToolCallID,
			},
		})
	}

	killed, err := processes.Kill(input.PID)
	if err != nil {
		return "", err
	}

	if !killed {
		if ctx != nil && ctx.Writer != nil {
			exitCode := -1
			if handle != nil && handle.ExitCode != nil {
				exitCode = *handle.ExitCode
			}
			_ = ctx.Writer.Custom(CustomEvent{
				Type: "data-sandbox-exit",
				Data: map[string]interface{}{
					"exitCode":   exitCode,
					"success":    false,
					"killed":     false,
					"toolCallId": ctx.ToolCallID,
				},
			})
		}
		return fmt.Sprintf("Process %d was not found or had already exited.", input.PID), nil
	}

	if ctx != nil && ctx.Writer != nil {
		exitCode := 137
		if handle != nil && handle.ExitCode != nil {
			exitCode = *handle.ExitCode
		}
		_ = ctx.Writer.Custom(CustomEvent{
			Type: "data-sandbox-exit",
			Data: map[string]interface{}{
				"exitCode":   exitCode,
				"success":    false,
				"killed":     true,
				"toolCallId": ctx.ToolCallID,
			},
		})
	}

	parts := []string{fmt.Sprintf("Process %d has been killed.", input.PID)}

	if handle != nil {
		// Get token limit from config
		var tokenLimit *int
		toolsConfig := ws.GetToolsConfig()
		if toolsConfig != nil {
			tc := toolsConfig.GetToolConfig("mastra_workspace_kill_process")
			if tc != nil {
				tokenLimit = tc.MaxOutputTokens
			}
		}

		tailLines := KillTailLines
		if handle.Stdout != "" {
			stdout := TruncateOutput(handle.Stdout, &tailLines, tokenLimit, "sandwich")
			if stdout != "" {
				parts = append(parts, "", "--- stdout (last output) ---", stdout)
			}
		}
		if handle.Stderr != "" {
			stderr := TruncateOutput(handle.Stderr, &tailLines, tokenLimit, "sandwich")
			if stderr != "" {
				parts = append(parts, "", "--- stderr (last output) ---", stderr)
			}
		}
	}

	return strings.Join(parts, "\n"), nil
}
