// Ported from: packages/core/src/loop/server.ts
//
// Server-only exports for the loop module.
// These exports contain OS/process dependencies and should not be imported
// in environments that do not support subprocess execution.
//
// Security WARNING: CreateRunCommandTool executes shell commands and poses
// significant security risks if misused. NEVER use with untrusted input.
// Always configure:
//   - AllowedCommands: Restrict which commands can be executed
//   - AllowedBasePaths: Restrict working directories
//   - Consider additional sandboxing (containers, VMs) for production
//
// See RunCommandToolOptions in the network sub-package for configuration details.
package loop

// Re-exports from the network sub-package are available via direct import:
//
//   import "github.com/brainlet/brainkit/agent-kit/core/loop/network"
//
//   tool := network.CreateRunCommandTool(network.RunCommandToolOptions{...})
//
// This file exists for documentation parity with the TypeScript server.ts
// which re-exports { createRunCommandTool, RunCommandToolOptions } from
// './network/run-command-tool'.
