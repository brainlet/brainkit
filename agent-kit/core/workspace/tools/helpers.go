// Ported from: packages/core/src/workspace/tools/helpers.ts
package tools

import (
	"fmt"
	"strings"
)

// =============================================================================
// Tool Context Types
// =============================================================================

// ToolContext holds the context available to workspace tool execute functions.
// In the TS version this is the tool execution context injected by the agent.
type ToolContext struct {
	// Workspace is the workspace instance.
	Workspace WorkspaceAccessor
	// Filesystem is the workspace filesystem (if configured).
	Filesystem FilesystemAccessor
	// Sandbox is the workspace sandbox (if configured).
	Sandbox SandboxAccessor
	// ToolCallID is the current tool call identifier.
	ToolCallID string
	// Writer is the stream writer for emitting custom events.
	Writer StreamWriter
}

// WorkspaceAccessor provides access to workspace capabilities needed by tools.
type WorkspaceAccessor interface {
	// GetToolsConfig returns the per-tool configuration.
	GetToolsConfig() *WorkspaceToolsConfig
	// LSP returns the LSP manager (if configured).
	LSP() LSPAccessor
	// CanBM25 checks if BM25 keyword search is available.
	CanBM25() bool
	// CanVector checks if vector semantic search is available.
	CanVector() bool
	// CanHybrid checks if hybrid search is available.
	CanHybrid() bool
	// Search searches indexed content.
	Search(query string, opts *SearchOptions) ([]SearchResult, error)
	// Index indexes content for search.
	Index(filePath, content string, opts *IndexOptions) error
	// Filesystem returns the filesystem provider.
	Filesystem() FilesystemAccessor
	// Sandbox returns the sandbox provider.
	Sandbox() SandboxAccessor
}

// FilesystemAccessor is the interface tools use to interact with the filesystem.
type FilesystemAccessor interface {
	ReadOnly() bool
	ReadFile(path string, options *ReadOptions) (interface{}, error)
	WriteFile(path string, content interface{}, options *WriteOptions) error
	DeleteFile(path string, options *RemoveOptions) error
	Mkdir(path string, options *MkdirOptions) error
	Rmdir(path string, options *RemoveOptions) error
	Readdir(path string, options *ListOptions) ([]FileEntry, error)
	Stat(path string) (*FileStat, error)
	Exists(path string) (bool, error)
}

// SandboxAccessor is the interface tools use to interact with the sandbox.
type SandboxAccessor interface {
	ExecuteCommand(command string, args []string, options *ExecuteCommandOptions) (*CommandResult, error)
	Processes() ProcessManager
}

// ProcessManager is the interface for managing background processes.
type ProcessManager interface {
	Spawn(command string, opts *SpawnOptions) (*ProcessHandle, error)
	Get(pid int) (*ProcessHandle, error)
	Kill(pid int) (bool, error)
}

// SpawnOptions configures process spawning.
type SpawnOptions struct {
	Cwd      string
	Timeout  int
	OnStdout func(data string)
	OnStderr func(data string)
}

// ProcessHandle represents a running or completed process.
type ProcessHandle struct {
	PID             int
	Command         string
	ExitCode        *int
	Stdout          string
	Stderr          string
	ExecutionTimeMs int64
	Success         bool
}

// WaitOptions configures process wait behavior.
type WaitOptions struct {
	OnStdout func(data string)
	OnStderr func(data string)
}

// Wait waits for the process to exit.
func (h *ProcessHandle) Wait(opts *WaitOptions) (*ProcessHandle, error) {
	// Stub — actual implementation depends on sandbox provider.
	return h, nil
}

// LSPAccessor is the interface tools use to interact with LSP diagnostics.
type LSPAccessor interface {
	GetDiagnosticsMulti(filePath, content string) ([]LSPDiagnostic, error)
}

// LSPDiagnostic represents an LSP diagnostic message.
type LSPDiagnostic struct {
	Line     int    `json:"line"`
	Char     int    `json:"character"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source,omitempty"`
}

// StreamWriter can emit custom events to the output stream.
type StreamWriter interface {
	Custom(event CustomEvent) error
}

// CustomEvent holds a custom stream event.
type CustomEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ReadOptions configures file read behavior.
type ReadOptions struct {
	Encoding string
}

// WriteOptions configures file write behavior.
type WriteOptions struct {
	Overwrite *bool
}

// RemoveOptions configures file/directory removal behavior.
type RemoveOptions struct {
	Recursive bool
	Force     bool
}

// MkdirOptions configures directory creation behavior.
type MkdirOptions struct {
	Recursive bool
}

// ListOptions configures directory listing behavior.
type ListOptions struct {
	Recursive bool
	MaxDepth  int
}

// CopyOptions configures file copy behavior.
type CopyOptions struct {
	Overwrite bool
}

// FileStat represents file/directory metadata.
type FileStat struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Type       string `json:"type"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
	MimeType   string `json:"mimeType,omitempty"`
}

// FileEntry represents a directory entry.
type FileEntry struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsSymlink bool   `json:"isSymlink,omitempty"`
}

// ExecuteCommandOptions configures command execution.
type ExecuteCommandOptions struct {
	Cwd      string
	Timeout  int
	OnStdout func(data string)
	OnStderr func(data string)
}

// CommandResult holds the result of a command execution.
type CommandResult struct {
	ExitCode        int
	Stdout          string
	Stderr          string
	Success         bool
	ExecutionTimeMs int64
}

// SearchOptions configures search behavior.
type SearchOptions struct {
	TopK     int    `json:"topK,omitempty"`
	Mode     string `json:"mode,omitempty"` // "bm25", "vector", "hybrid"
	MinScore *float64 `json:"minScore,omitempty"`
}

// SearchResult holds a search result.
type SearchResult struct {
	ID        string     `json:"id"`
	Content   string     `json:"content"`
	LineRange *LineRange `json:"lineRange,omitempty"`
}

// LineRange represents a line range.
type LineRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// IndexOptions configures content indexing.
type IndexOptions struct {
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// =============================================================================
// Helper Functions
// =============================================================================

// RequireWorkspace extracts the workspace from the tool context.
// Returns an error if the workspace is not available.
func RequireWorkspace(ctx *ToolContext) (WorkspaceAccessor, error) {
	if ctx == nil || ctx.Workspace == nil {
		return nil, fmt.Errorf("workspace not available: ensure the agent has a workspace configured")
	}
	return ctx.Workspace, nil
}

// RequireFilesystemResult holds the result of RequireFilesystem.
type RequireFilesystemResult struct {
	Workspace  WorkspaceAccessor
	Filesystem FilesystemAccessor
}

// RequireFilesystem extracts both workspace and filesystem from the tool context.
// Returns an error if either is not available.
func RequireFilesystem(ctx *ToolContext) (*RequireFilesystemResult, error) {
	ws, err := RequireWorkspace(ctx)
	if err != nil {
		return nil, err
	}

	fs := ws.Filesystem()
	if fs == nil {
		return nil, fmt.Errorf("workspace does not have a filesystem configured")
	}

	return &RequireFilesystemResult{
		Workspace:  ws,
		Filesystem: fs,
	}, nil
}

// RequireSandboxResult holds the result of RequireSandbox.
type RequireSandboxResult struct {
	Workspace WorkspaceAccessor
	Sandbox   SandboxAccessor
}

// RequireSandbox extracts both workspace and sandbox from the tool context.
// Returns an error if either is not available.
func RequireSandbox(ctx *ToolContext) (*RequireSandboxResult, error) {
	ws, err := RequireWorkspace(ctx)
	if err != nil {
		return nil, err
	}

	sb := ws.Sandbox()
	if sb == nil {
		return nil, fmt.Errorf("workspace does not have a sandbox configured")
	}

	return &RequireSandboxResult{
		Workspace: ws,
		Sandbox:   sb,
	}, nil
}

// EmitWorkspaceMetadata emits workspace metadata for a tool execution.
func EmitWorkspaceMetadata(ctx *ToolContext, toolName string) error {
	if ctx == nil || ctx.Writer == nil {
		return nil
	}
	return ctx.Writer.Custom(CustomEvent{
		Type: "data-workspace-metadata",
		Data: map[string]interface{}{
			"toolName": toolName,
		},
	})
}

// GetEditDiagnosticsText returns LSP diagnostics text for an edited file.
// Returns an empty string if LSP is not configured or there are no diagnostics.
func GetEditDiagnosticsText(ws WorkspaceAccessor, filePath, content string) string {
	lsp := ws.LSP()
	if lsp == nil {
		return ""
	}

	diagnostics, err := lsp.GetDiagnosticsMulti(filePath, content)
	if err != nil || len(diagnostics) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("\n\nDiagnostics (%d):", len(diagnostics)))
	for _, d := range diagnostics {
		source := ""
		if d.Source != "" {
			source = " [" + d.Source + "]"
		}
		lines = append(lines, fmt.Sprintf("  Line %d:%d %s: %s%s",
			d.Line, d.Char, strings.ToUpper(d.Severity), d.Message, source))
	}
	return strings.Join(lines, "\n")
}
