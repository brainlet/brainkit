// Ported from: packages/core/src/workspace/sandbox/native-sandbox/wrapper.ts
package nativesandbox

// =============================================================================
// Command Wrapper
// =============================================================================

// WrappedCommand holds a wrapped command and its arguments.
type WrappedCommand struct {
	Command string
	Args    []string
}

// WrapCommandOptions holds options for wrapping a command.
type WrapCommandOptions struct {
	// Backend is the isolation backend to use.
	Backend IsolationBackend
	// WorkspacePath is the workspace directory path.
	WorkspacePath string
	// SeatbeltProfile is pre-generated seatbelt profile content (optional).
	SeatbeltProfile string
	// Config is the native sandbox configuration.
	Config NativeSandboxConfig
}

// WrapCommand wraps a command with the appropriate sandbox backend.
func WrapCommand(command string, options WrapCommandOptions) (WrappedCommand, error) {
	switch options.Backend {
	case IsolationSeatbelt:
		profile := options.SeatbeltProfile
		if profile == "" {
			var err error
			profile, err = GenerateSeatbeltProfile(options.WorkspacePath, options.Config)
			if err != nil {
				return WrappedCommand{}, err
			}
		}
		return BuildSeatbeltCommand(command, profile), nil

	case IsolationBwrap:
		return BuildBwrapCommand(command, options.WorkspacePath, options.Config), nil

	default: // "none"
		return WrappedCommand{Command: command, Args: nil}, nil
	}
}
