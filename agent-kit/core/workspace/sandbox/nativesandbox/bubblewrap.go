// Ported from: packages/core/src/workspace/sandbox/native-sandbox/bubblewrap.ts
package nativesandbox

import (
	"os"
	"path/filepath"
	"strings"
)

// =============================================================================
// Bubblewrap (Linux bwrap)
// =============================================================================

// defaultReadonlyBinds are system paths to mount read-only by default.
// These are needed for basic command execution.
var defaultReadonlyBinds = []string{
	"/usr",
	"/lib",
	"/lib64",
	"/bin",
	"/sbin",
	"/etc/alternatives",
	"/etc/ssl",
	"/etc/ca-certificates",
	"/etc/resolv.conf",
	"/etc/hosts",
	"/etc/passwd",
	"/etc/group",
	"/etc/nsswitch.conf",
	"/etc/ld.so.cache",
	"/etc/localtime",
}

// BuildBwrapCommand builds the bwrap command arguments for the given configuration.
func BuildBwrapCommand(command, workspacePath string, config NativeSandboxConfig) WrappedCommand {
	// If custom bwrap args are provided, use them directly
	if len(config.BwrapArgs) > 0 {
		args := make([]string, len(config.BwrapArgs))
		copy(args, config.BwrapArgs)
		args = append(args, "--", "sh", "-c", command)
		return WrappedCommand{
			Command: "bwrap",
			Args:    args,
		}
	}

	var bwrapArgs []string

	// Create new namespaces for isolation
	bwrapArgs = append(bwrapArgs, "--unshare-pid")  // PID namespace
	bwrapArgs = append(bwrapArgs, "--unshare-ipc")  // IPC namespace
	bwrapArgs = append(bwrapArgs, "--unshare-uts")  // UTS namespace

	// Network isolation (unless explicitly allowed)
	if !config.AllowNetwork {
		bwrapArgs = append(bwrapArgs, "--unshare-net")
	}

	// Mount a new /proc for the PID namespace
	bwrapArgs = append(bwrapArgs, "--proc", "/proc")

	// Mount a tmpfs at /tmp
	bwrapArgs = append(bwrapArgs, "--tmpfs", "/tmp")

	// Mount system paths read-only
	for _, p := range defaultReadonlyBinds {
		// Use --ro-bind-try to skip paths that don't exist
		bwrapArgs = append(bwrapArgs, "--ro-bind-try", p, p)
	}

	// Mount custom read-only paths
	for _, p := range config.ReadOnlyPaths {
		bwrapArgs = append(bwrapArgs, "--ro-bind", p, p)
	}

	// Allow system binaries by default
	if config.AllowSystemBinaries == nil || *config.AllowSystemBinaries {
		// Include the Go binary location
		execPath, _ := os.Executable()
		if execPath != "" {
			nodeDir := filepath.Dir(execPath)
			// Mount the binary directory if it's not already covered
			covered := false
			for _, p := range defaultReadonlyBinds {
				if strings.HasPrefix(nodeDir, p) {
					covered = true
					break
				}
			}
			if !covered {
				bwrapArgs = append(bwrapArgs, "--ro-bind", nodeDir, nodeDir)
			}
		}

		// Also mount common runtime locations
		bwrapArgs = append(bwrapArgs, "--ro-bind-try", "/opt", "/opt")
		bwrapArgs = append(bwrapArgs, "--ro-bind-try", "/snap", "/snap")
	}

	// Mount workspace read-write
	bwrapArgs = append(bwrapArgs, "--bind", workspacePath, workspacePath)

	// Mount custom read-write paths
	for _, p := range config.ReadWritePaths {
		bwrapArgs = append(bwrapArgs, "--bind", p, p)
	}

	// Set the working directory
	bwrapArgs = append(bwrapArgs, "--chdir", workspacePath)

	// Die with parent (clean up if the parent process dies)
	bwrapArgs = append(bwrapArgs, "--die-with-parent")

	// Add the command separator and run via sh -c
	bwrapArgs = append(bwrapArgs, "--", "sh", "-c", command)

	return WrappedCommand{
		Command: "bwrap",
		Args:    bwrapArgs,
	}
}
