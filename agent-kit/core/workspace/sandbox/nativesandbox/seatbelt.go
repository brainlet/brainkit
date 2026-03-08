// Ported from: packages/core/src/workspace/sandbox/native-sandbox/seatbelt.ts
package nativesandbox

import (
	"fmt"
	"strings"
)

// =============================================================================
// Seatbelt (macOS sandbox-exec)
// =============================================================================

// machServices are Mach services needed for basic operation.
var machServices = []string{
	"com.apple.distributed_notifications@Uv3",
	"com.apple.logd",
	"com.apple.system.logger",
	"com.apple.system.notification_center",
	"com.apple.system.opendirectoryd.libinfo",
	"com.apple.system.opendirectoryd.membership",
	"com.apple.bsd.dirhelper",
	"com.apple.securityd.xpc",
	"com.apple.SecurityServer",
	"com.apple.trustd.agent",
}

// escapePath escapes a path for use in an SBPL profile.
// Uses Go's %q format for proper escaping.
func escapePath(pathStr string) string {
	return fmt.Sprintf("%q", pathStr)
}

// GenerateSeatbeltProfile generates a seatbelt profile for the given configuration.
//
// The profile:
//   - Allows all file reads (can't restrict with subpath on macOS)
//   - Restricts file writes to workspace and temp directories
//   - Blocks network unless explicitly allowed
func GenerateSeatbeltProfile(workspacePath string, config NativeSandboxConfig) (string, error) {
	// Fail-closed: seatbelt cannot restrict process-exec, so reject unsupported config
	if config.AllowSystemBinaries != nil && !*config.AllowSystemBinaries {
		return "", fmt.Errorf(
			"allowSystemBinaries: false is not supported by seatbelt (macOS). " +
				"Use bubblewrap on Linux or remove this restriction.",
		)
	}

	var lines []string

	// Version and default deny
	lines = append(lines, "(version 1)")
	lines = append(lines, `(deny default (with message "mastra-sandbox"))`)
	lines = append(lines, "")

	// Process permissions
	lines = append(lines, "; Process permissions")
	lines = append(lines, "(allow process-exec)")
	lines = append(lines, "(allow process-fork)")
	lines = append(lines, "(allow process-info* (target same-sandbox))")
	lines = append(lines, "(allow signal (target same-sandbox))")
	lines = append(lines, "")

	// Mach IPC
	lines = append(lines, "; Mach IPC")
	lines = append(lines, "(allow mach-lookup")
	for _, service := range machServices {
		lines = append(lines, fmt.Sprintf(`  (global-name "%s")`, service))
	}
	lines = append(lines, ")")
	lines = append(lines, "")

	// IPC
	lines = append(lines, "; IPC")
	lines = append(lines, "(allow ipc-posix-shm)")
	lines = append(lines, "(allow ipc-posix-sem)")
	lines = append(lines, "")

	// User preferences
	lines = append(lines, "; User preferences")
	lines = append(lines, "(allow user-preference-read)")
	lines = append(lines, "")

	// sysctl
	lines = append(lines, "; sysctl")
	lines = append(lines, "(allow sysctl-read)")
	lines = append(lines, "")

	// Device files
	lines = append(lines, "; Device files")
	lines = append(lines, `(allow file-ioctl (literal "/dev/null"))`)
	lines = append(lines, `(allow file-ioctl (literal "/dev/zero"))`)
	lines = append(lines, `(allow file-ioctl (literal "/dev/random"))`)
	lines = append(lines, `(allow file-ioctl (literal "/dev/urandom"))`)
	lines = append(lines, `(allow file-ioctl (literal "/dev/tty"))`)
	lines = append(lines, "")

	// File read access - allow all reads (macOS limitation: can't use subpath without this)
	lines = append(lines, "; File read access (allow all - macOS sandbox limitation)")
	lines = append(lines, "(allow file-read*)")

	// Add custom read-only paths
	for _, p := range config.ReadOnlyPaths {
		lines = append(lines, fmt.Sprintf("(allow file-read* (subpath %s))", escapePath(p)))
	}
	lines = append(lines, "")

	// File write access - restrict to workspace and temp
	lines = append(lines, "; File write access (restricted to workspace and temp)")

	// Workspace
	lines = append(lines, fmt.Sprintf("(allow file-write* (subpath %s))", escapePath(workspacePath)))

	// Temp directories (needed for many operations)
	lines = append(lines, `(allow file-write* (subpath "/private/tmp"))`)
	lines = append(lines, `(allow file-write* (subpath "/var/folders"))`)
	lines = append(lines, `(allow file-write* (subpath "/private/var/folders"))`)

	// Custom read-write paths
	for _, p := range config.ReadWritePaths {
		lines = append(lines, fmt.Sprintf("(allow file-write* (subpath %s))", escapePath(p)))
	}
	lines = append(lines, "")

	// Network
	lines = append(lines, "; Network")
	if config.AllowNetwork {
		lines = append(lines, "(allow network*)")
	} else {
		lines = append(lines, `(deny network* (with message "mastra-sandbox-network"))`)
	}

	return strings.Join(lines, "\n"), nil
}

// BuildSeatbeltCommand builds the command arguments for sandbox-exec.
//
// Uses -p (inline profile) instead of -f (file) because
// -f doesn't work reliably with path filters on modern macOS.
func BuildSeatbeltCommand(command, profile string) WrappedCommand {
	return WrappedCommand{
		Command: "sandbox-exec",
		Args:    []string{"-p", profile, "sh", "-c", command},
	}
}
