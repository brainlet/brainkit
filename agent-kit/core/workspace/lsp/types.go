// Ported from: packages/core/src/workspace/lsp/types.ts
package lsp

// =============================================================================
// Configuration
// =============================================================================

// LSPConfig holds configuration for LSP diagnostics in a workspace.
type LSPConfig struct {
	// Root is the project root directory (absolute path).
	// Used as rootUri for LSP servers and cwd for spawning.
	Root string
	// DiagnosticTimeout is the timeout in ms for waiting for diagnostics
	// after an edit (default: 5000).
	DiagnosticTimeout int
	// InitTimeout is the timeout in ms for LSP server initialization
	// (default: 15000).
	InitTimeout int
	// DisableServers is a list of server IDs to disable.
	DisableServers []string
	// BinaryOverrides is an explicit command override for a specific server,
	// bypassing all automatic lookup.
	BinaryOverrides map[string]string
	// SearchPaths are extra directories to search for language server binaries
	// and Node.js modules.
	SearchPaths []string
	// PackageRunner is a last-resort fallback runner command.
	PackageRunner string
}

// =============================================================================
// Diagnostics
// =============================================================================

// DiagnosticSeverity represents the severity of a diagnostic.
type DiagnosticSeverity string

const (
	DiagnosticSeverityError   DiagnosticSeverity = "error"
	DiagnosticSeverityWarning DiagnosticSeverity = "warning"
	DiagnosticSeverityInfo    DiagnosticSeverity = "info"
	DiagnosticSeverityHint    DiagnosticSeverity = "hint"
)

// LSPDiagnostic is a diagnostic message from an LSP server.
type LSPDiagnostic struct {
	// Severity is the diagnostic severity.
	Severity DiagnosticSeverity
	// Message is the diagnostic message.
	Message string
	// Line is the 1-indexed line number.
	Line int
	// Character is the 1-indexed character offset.
	Character int
	// Source is the source of the diagnostic (e.g., "typescript", "eslint").
	Source string
}

// =============================================================================
// Server Definitions
// =============================================================================

// LSPServerDef defines a built-in LSP server.
type LSPServerDef struct {
	// ID is the server identifier.
	ID string
	// Name is the display name.
	Name string
	// LanguageIDs are the LSP language identifiers this server handles.
	LanguageIDs []string
	// Markers are file/directory markers that identify the project root.
	Markers []string
	// Command returns the command string to spawn the server.
	// Returns empty string if the server is not available.
	Command func(root string) string
	// Initialization returns initialization options for the server.
	Initialization func(root string) map[string]interface{}
}

// MapSeverity maps LSP DiagnosticSeverity (numeric) to our string severity.
func MapSeverity(severity int) DiagnosticSeverity {
	switch severity {
	case 1:
		return DiagnosticSeverityError
	case 2:
		return DiagnosticSeverityWarning
	case 3:
		return DiagnosticSeverityInfo
	case 4:
		return DiagnosticSeverityHint
	default:
		return DiagnosticSeverityWarning
	}
}
