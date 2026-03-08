// Ported from: packages/core/src/workspace/tools/execute-command.ts
package tools

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// Execute Command Tool
// =============================================================================

// ExecuteCommandInput holds the input for the execute_command tool.
type ExecuteCommandInput struct {
	// Command is the shell command to execute.
	Command string `json:"command"`
	// Timeout is the maximum execution time in seconds.
	Timeout *int `json:"timeout,omitempty"`
	// Cwd is the working directory for the command.
	Cwd string `json:"cwd,omitempty"`
	// Tail limits output to the last N lines (default: DefaultTailLines). Use 0 for no limit.
	Tail *int `json:"tail,omitempty"`
	// Background runs the command in the background (returns PID immediately).
	Background bool `json:"background,omitempty"`
}

// extractTailPipe extracts `| tail -N` or `| tail -n N` from the end of a command.
// LLMs often pipe to tail for long outputs, but this prevents streaming.
// By stripping the tail pipe and applying it programmatically, output streams in real time.
func extractTailPipe(command string) (string, *int) {
	re := regexp.MustCompile(`\|\s*tail\s+(?:-n\s+)?(-?\d+)\s*$`)
	match := re.FindStringSubmatch(command)
	if match != nil {
		lines, err := strconv.Atoi(match[1])
		if err == nil {
			lines = int(math.Abs(float64(lines)))
			if lines > 0 {
				cleaned := re.ReplaceAllString(command, "")
				cleaned = strings.TrimSpace(cleaned)
				return cleaned, &lines
			}
		}
	}
	return command, nil
}

// ExecuteCommand executes the execute_command tool.
func ExecuteCommand(input *ExecuteCommandInput, ctx *ToolContext) (string, error) {
	result, err := RequireSandbox(ctx)
	if err != nil {
		return "", err
	}

	ws := result.Workspace
	sb := result.Sandbox

	command := input.Command
	tail := input.Tail

	// Convert timeout from seconds to milliseconds
	var timeoutMs int
	if input.Timeout != nil {
		timeoutMs = *input.Timeout * 1000
	}

	// Extract tail pipe from command for foreground processes
	if !input.Background {
		cleanedCmd, extractedTail := extractTailPipe(command)
		command = cleanedCmd
		if extractedTail != nil {
			tail = extractedTail
		}
	}

	// Get tool config
	var tokenLimit *int
	tokenFrom := "sandwich"
	toolsConfig := ws.GetToolsConfig()
	if toolsConfig != nil {
		tc := toolsConfig.GetToolConfig("mastra_workspace_execute_command")
		if tc != nil {
			tokenLimit = tc.MaxOutputTokens
		}
	}

	// Background mode: spawn and return immediately
	if input.Background {
		processes := sb.Processes()
		if processes == nil {
			return "", fmt.Errorf("sandbox does not support processes")
		}

		handle, err := processes.Spawn(command, &SpawnOptions{
			Cwd:     input.Cwd,
			Timeout: timeoutMs,
		})
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("Started background process (PID: %d)", handle.PID), nil
	}

	// Foreground mode: execute and wait
	startedAt := time.Now()

	cmdResult, err := sb.ExecuteCommand(command, nil, &ExecuteCommandOptions{
		Timeout: timeoutMs,
		Cwd:     input.Cwd,
	})
	if err != nil {
		elapsed := time.Since(startedAt).Milliseconds()
		_ = elapsed

		// Emit exit event via writer
		if ctx != nil && ctx.Writer != nil {
			_ = ctx.Writer.Custom(CustomEvent{
				Type: "data-sandbox-exit",
				Data: map[string]interface{}{
					"exitCode":        -1,
					"success":         false,
					"executionTimeMs": elapsed,
					"toolCallId":      ctx.ToolCallID,
				},
			})
		}

		errorMessage := err.Error()
		parts := []string{}
		truncatedStdout := TruncateOutput("", tail, tokenLimit, tokenFrom)
		if truncatedStdout != "" {
			parts = append(parts, truncatedStdout)
		}
		parts = append(parts, fmt.Sprintf("Error: %s", errorMessage))
		return strings.Join(parts, "\n"), nil
	}

	// Emit exit event
	if ctx != nil && ctx.Writer != nil {
		_ = ctx.Writer.Custom(CustomEvent{
			Type: "data-sandbox-exit",
			Data: map[string]interface{}{
				"exitCode":        cmdResult.ExitCode,
				"success":         cmdResult.Success,
				"executionTimeMs": cmdResult.ExecutionTimeMs,
				"toolCallId":      ctx.ToolCallID,
			},
		})
	}

	if !cmdResult.Success {
		var parts []string
		stdout := TruncateOutput(cmdResult.Stdout, tail, tokenLimit, tokenFrom)
		stderr := TruncateOutput(cmdResult.Stderr, tail, tokenLimit, tokenFrom)
		if stdout != "" {
			parts = append(parts, stdout)
		}
		if stderr != "" {
			parts = append(parts, stderr)
		}
		parts = append(parts, fmt.Sprintf("Exit code: %d", cmdResult.ExitCode))
		return strings.Join(parts, "\n"), nil
	}

	output := TruncateOutput(cmdResult.Stdout, tail, tokenLimit, tokenFrom)
	if output == "" {
		return "(no output)", nil
	}
	return output, nil
}
