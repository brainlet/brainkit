// Ported from: packages/core/src/loop/network/run-command-tool.ts
//
// Node.js-specific tool for running shell commands.
// This file is separated from validation.go to avoid bundling process
// dependencies into non-server builds.
//
// Security WARNING: This tool executes shell commands and can be dangerous.
//   - NEVER use with untrusted input or in multi-tenant environments
//   - Always configure AllowedCommands to restrict executable commands
//   - Always set AllowedBasePaths to restrict working directories
//   - Consider running in a sandboxed environment (container, VM)
//   - Review all commands that agents may construct before deployment
package network

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Security constants
// ---------------------------------------------------------------------------

// dangerousPatterns are characters that could enable shell injection attacks.
// These are rejected when found in command input.
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`[;&|` + "`" + `$(){}[\]<>]`), // Shell metacharacters
	regexp.MustCompile(`\n|\r`),                        // Newlines (command chaining)
	regexp.MustCompile(`\\[^ ]`),                        // Backslashes followed by non-space (escaped spaces are OK)
}

// blockedCommands are commands that are inherently dangerous and blocked by default.
var blockedCommands = map[string]bool{
	"rm": true, "rmdir": true, "del": true, "format": true, "mkfs": true,
	"dd": true, "shutdown": true, "reboot": true, "halt": true, "poweroff": true,
	"init": true, "kill": true, "killall": true, "pkill": true,
	"chmod": true, "chown": true, "chgrp": true,
	"sudo": true, "su": true, "passwd": true,
	"useradd": true, "userdel": true, "usermod": true, "groupadd": true,
	"visudo": true, "crontab": true, "systemctl": true, "service": true,
	"curl": true, "wget": true, "nc": true, "netcat": true,
	"ssh": true, "scp": true, "ftp": true, "telnet": true,
	"eval": true, "source": true, "exec": true,
}

// ---------------------------------------------------------------------------
// RunCommandToolOptions
// ---------------------------------------------------------------------------

// RunCommandToolOptions configures the run-command tool.
type RunCommandToolOptions struct {
	// AllowedCommands is an allowlist of command prefixes that are permitted.
	// If empty, all non-blocked commands are allowed (less secure).
	// Example: []string{"git", "npm", "node", "ls", "cat", "echo"}
	AllowedCommands []string

	// AllowedBasePaths are base paths where command execution is permitted.
	// The cwd parameter must resolve to a path under one of these directories.
	// If empty, any cwd is allowed (less secure).
	// Example: []string{"/home/user/projects", "/tmp/workspace"}
	AllowedBasePaths []string

	// AdditionalBlockedCommands are additional commands to block beyond the
	// default blocklist.
	AdditionalBlockedCommands []string

	// MaxTimeout is the maximum execution time in milliseconds.
	// Default: 30000 (30 seconds).
	MaxTimeout int

	// MaxBuffer is the maximum buffer size for stdout/stderr in bytes.
	// Default: 1048576 (1MB).
	MaxBuffer int

	// AllowUnsafeCharacters controls whether potentially dangerous shell
	// metacharacters are allowed. Setting this to true is NOT recommended.
	// Default: false.
	AllowUnsafeCharacters bool
}

// ---------------------------------------------------------------------------
// RunCommandResult
// ---------------------------------------------------------------------------

// RunCommandResult is the output of executing a command via the tool.
type RunCommandResult struct {
	Success  bool   `json:"success"`
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Message  string `json:"message,omitempty"`
}

// ---------------------------------------------------------------------------
// RunCommandInput
// ---------------------------------------------------------------------------

// RunCommandInput is the input schema for the run-command tool.
type RunCommandInput struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
	Cwd     string `json:"cwd,omitempty"`
}

// ---------------------------------------------------------------------------
// Helper functions
// ---------------------------------------------------------------------------

// isPathAllowed validates that a path is under one of the allowed base paths.
func isPathAllowed(targetPath string, allowedBasePaths []string) bool {
	if len(allowedBasePaths) == 0 {
		return true
	}
	normalizedTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}
	normalizedTarget = filepath.Clean(normalizedTarget)
	for _, basePath := range allowedBasePaths {
		normalizedBase, err := filepath.Abs(basePath)
		if err != nil {
			continue
		}
		normalizedBase = filepath.Clean(normalizedBase)
		if normalizedTarget == normalizedBase || strings.HasPrefix(normalizedTarget, normalizedBase+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// extractBaseCommand extracts the base command from a command string.
func extractBaseCommand(command string) string {
	trimmed := strings.TrimSpace(command)
	firstSpace := strings.Index(trimmed, " ")
	baseCmd := trimmed
	if firstSpace != -1 {
		baseCmd = trimmed[:firstSpace]
	}
	// Handle paths like /usr/bin/git -> git
	lastSlash := strings.LastIndex(baseCmd, "/")
	if lastSlash != -1 {
		baseCmd = baseCmd[lastSlash+1:]
	}
	return baseCmd
}

// ---------------------------------------------------------------------------
// CreateRunCommandTool
// ---------------------------------------------------------------------------

// RunCommandTool is a tool that executes shell commands with security restrictions.
type RunCommandTool struct {
	ID          string
	Description string
	Options     RunCommandToolOptions
	blocked     map[string]bool
}

// CreateRunCommandTool creates a tool that lets agents run shell commands
// with security restrictions.
//
// Security WARNING: This tool executes shell commands. Even with restrictions,
// it should NEVER be used with untrusted input. Always:
//   - Configure AllowedCommands to restrict which commands can run
//   - Configure AllowedBasePaths to restrict working directories
//   - Review agent prompts to understand what commands may be generated
//   - Consider additional sandboxing (containers, VMs) for production use
func CreateRunCommandTool(options RunCommandToolOptions) *RunCommandTool {
	if options.MaxTimeout == 0 {
		options.MaxTimeout = 30000
	}
	if options.MaxBuffer == 0 {
		options.MaxBuffer = 1024 * 1024 // 1MB
	}

	blocked := make(map[string]bool)
	for k, v := range blockedCommands {
		blocked[k] = v
	}
	for _, cmd := range options.AdditionalBlockedCommands {
		blocked[strings.ToLower(cmd)] = true
	}

	return &RunCommandTool{
		ID:          "run-command",
		Description: "Execute a shell command and return the result. Only permitted commands in allowed directories can be executed.",
		Options:     options,
		blocked:     blocked,
	}
}

// Execute runs the shell command with validation and security checks.
func (t *RunCommandTool) Execute(ctx context.Context, input RunCommandInput) RunCommandResult {
	// Validate: reject dangerous characters
	if !t.Options.AllowUnsafeCharacters {
		for _, pattern := range dangerousPatterns {
			if pattern.MatchString(input.Command) {
				return RunCommandResult{
					Success:  false,
					ExitCode: 1,
					Message:  fmt.Sprintf("Command rejected: contains potentially unsafe characters. Pattern: %s", pattern.String()),
				}
			}
		}
	}

	// Validate: extract and check base command
	baseCommand := strings.ToLower(extractBaseCommand(input.Command))

	// Check blocked commands
	if t.blocked[baseCommand] {
		return RunCommandResult{
			Success:  false,
			ExitCode: 1,
			Message:  fmt.Sprintf("Command rejected: '%s' is not permitted for security reasons", baseCommand),
		}
	}

	// Check allowlist if configured
	if len(t.Options.AllowedCommands) > 0 {
		allowed := false
		for _, cmd := range t.Options.AllowedCommands {
			if baseCommand == strings.ToLower(cmd) {
				allowed = true
				break
			}
		}
		if !allowed {
			return RunCommandResult{
				Success:  false,
				ExitCode: 1,
				Message:  fmt.Sprintf("Command rejected: '%s' is not in the allowed commands list", baseCommand),
			}
		}
	}

	// Validate: check cwd against allowed base paths
	if input.Cwd != "" && !isPathAllowed(input.Cwd, t.Options.AllowedBasePaths) {
		return RunCommandResult{
			Success:  false,
			ExitCode: 1,
			Message:  fmt.Sprintf("Command rejected: working directory '%s' is not within allowed paths", input.Cwd),
		}
	}

	// Apply timeout cap
	timeout := input.Timeout
	if timeout <= 0 || timeout > t.Options.MaxTimeout {
		timeout = t.Options.MaxTimeout
	}

	// Execute command
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, "sh", "-c", input.Command)
	if input.Cwd != "" {
		cmd.Dir = input.Cwd
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		exitCode := 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		outStr := string(output)
		return RunCommandResult{
			Success:  false,
			ExitCode: exitCode,
			Stdout:   truncateLast(outStr, 2000),
			Stderr:   truncateLast(err.Error(), 2000),
			Message:  err.Error(),
		}
	}

	outStr := string(output)
	return RunCommandResult{
		Success:  true,
		ExitCode: 0,
		Stdout:   truncateLast(outStr, 3000),
	}
}

// truncateLast returns the last n bytes of s.
func truncateLast(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
