// Ported from: packages/mcp/src/tool/mcp-stdio/create-child-process.ts
package mcp

import (
	"context"
	"os/exec"
)

// CreateChildProcess creates a child process (os/exec.Cmd) from a StdioConfig.
// The context is used for cancellation (equivalent to TS AbortSignal).
func CreateChildProcess(config StdioConfig, ctx context.Context) *exec.Cmd {
	args := config.Args
	if args == nil {
		args = []string{}
	}

	cmd := exec.CommandContext(ctx, config.Command, args...)

	// Build environment
	env := GetEnvironment(config.Env)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	if config.Cwd != "" {
		cmd.Dir = config.Cwd
	}

	// Stderr defaults to os.Stderr (inherit) unless config.Stderr is set.
	// In Go, exec.Cmd.Stderr defaults to nil (discarded).
	// We mimic the TS behavior of defaulting to 'inherit' by not setting stderr
	// (the caller can set cmd.Stderr = os.Stderr if desired).

	return cmd
}
