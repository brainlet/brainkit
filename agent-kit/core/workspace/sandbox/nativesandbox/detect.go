// Ported from: packages/core/src/workspace/sandbox/native-sandbox/detect.ts
package nativesandbox

import (
	"fmt"
	"os/exec"
	"runtime"
)

// =============================================================================
// Platform Detection
// =============================================================================

// commandExists checks if a command exists on the system.
func commandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// IsSeatbeltAvailable checks if seatbelt (sandbox-exec) is available.
// This is built-in on macOS.
func IsSeatbeltAvailable() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	return commandExists("sandbox-exec")
}

// IsBwrapAvailable checks if bubblewrap (bwrap) is available.
// This must be installed on Linux systems.
func IsBwrapAvailable() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	return commandExists("bwrap")
}

// DetectIsolation detects the best available isolation backend for the current platform.
func DetectIsolation() SandboxDetectionResult {
	switch runtime.GOOS {
	case "darwin":
		available := IsSeatbeltAvailable()
		message := "macOS seatbelt (sandbox-exec) is available"
		if !available {
			message = "macOS seatbelt (sandbox-exec) not found - this is unexpected on macOS"
		}
		return SandboxDetectionResult{
			Backend:   IsolationSeatbelt,
			Available: available,
			Message:   message,
		}

	case "linux":
		available := IsBwrapAvailable()
		message := "Linux bubblewrap (bwrap) is available"
		if !available {
			message = "Linux bubblewrap (bwrap) not found. Install with: apt install bubblewrap (Debian/Ubuntu) or dnf install bubblewrap (Fedora)"
		}
		return SandboxDetectionResult{
			Backend:   IsolationBwrap,
			Available: available,
			Message:   message,
		}

	default:
		return SandboxDetectionResult{
			Backend:   IsolationNone,
			Available: false,
			Message:   fmt.Sprintf("Native sandboxing is not supported on %s. Commands will run without isolation.", runtime.GOOS),
		}
	}
}

// IsIsolationAvailable checks if a specific isolation backend is available.
func IsIsolationAvailable(backend IsolationBackend) bool {
	switch backend {
	case IsolationSeatbelt:
		return IsSeatbeltAvailable()
	case IsolationBwrap:
		return IsBwrapAvailable()
	case IsolationNone:
		return true
	default:
		return false
	}
}

// GetRecommendedIsolation returns the recommended isolation backend for this platform.
// Returns IsolationNone if no sandboxing is available.
func GetRecommendedIsolation() IsolationBackend {
	result := DetectIsolation()
	if result.Available {
		return result.Backend
	}
	return IsolationNone
}
