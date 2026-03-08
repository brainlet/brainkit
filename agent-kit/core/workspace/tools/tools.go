// Ported from: packages/core/src/workspace/tools/tools.ts
package tools

import (
	"fmt"
	"time"
)

// =============================================================================
// Tool Definition
// =============================================================================

// WorkspaceTool represents a workspace tool with its configuration.
type WorkspaceTool struct {
	// ID is the tool name/identifier.
	ID string
	// Description is the human-readable tool description.
	Description string
	// RequireApproval controls whether the tool requires user approval.
	RequireApproval bool
	// Execute is the tool's execute function.
	Execute func(input interface{}, ctx *ToolContext) (string, error)
}

// =============================================================================
// Tool Config Resolution
// =============================================================================

// ResolvedToolConfig holds the resolved configuration for a specific tool.
type ResolvedToolConfig struct {
	Enabled                bool
	RequireApproval        bool
	RequireReadBeforeWrite bool
	MaxOutputTokens        *int
	Name                   string
}

// ResolveToolConfig resolves the effective configuration for a specific tool.
//
// Resolution order (later overrides earlier):
//  1. Built-in defaults (enabled: true, requireApproval: false)
//  2. Top-level config (tools.Enabled, tools.RequireApproval)
//  3. Per-tool config (tools[toolName].Enabled, tools[toolName].RequireApproval)
func ResolveToolConfig(toolsConfig *WorkspaceToolsConfig, toolName string) ResolvedToolConfig {
	result := ResolvedToolConfig{
		Enabled: true,
		Name:    toolName,
	}

	if toolsConfig == nil {
		return result
	}

	if toolsConfig.Enabled != nil {
		result.Enabled = *toolsConfig.Enabled
	}
	if toolsConfig.RequireApproval != nil {
		result.RequireApproval = *toolsConfig.RequireApproval
	}

	perToolConfig := toolsConfig.GetToolConfig(toolName)
	if perToolConfig == nil {
		return result
	}

	if perToolConfig.Enabled != nil {
		result.Enabled = *perToolConfig.Enabled
	}
	if perToolConfig.RequireApproval != nil {
		result.RequireApproval = *perToolConfig.RequireApproval
	}
	if perToolConfig.RequireReadBeforeWrite != nil {
		result.RequireReadBeforeWrite = *perToolConfig.RequireReadBeforeWrite
	}
	if perToolConfig.MaxOutputTokens != nil {
		result.MaxOutputTokens = perToolConfig.MaxOutputTokens
	}
	if perToolConfig.Name != nil {
		result.Name = *perToolConfig.Name
	}

	return result
}

// =============================================================================
// Read Tracker Integration
// =============================================================================

// FileReadTracker tracks which files have been read, enabling
// requireReadBeforeWrite enforcement for write tools.
type FileReadTracker interface {
	RecordRead(filePath string, modifiedAt time.Time)
	NeedsReRead(filePath string, currentModifiedAt time.Time) ReadCheckResult
	ClearReadRecord(filePath string)
}

// ReadCheckResult holds the result of a NeedsReRead check.
type ReadCheckResult struct {
	NeedsReRead bool
	Reason      string
}

// FileWriteLock serializes write operations to individual file paths.
type FileWriteLock interface {
	WithLock(filePath string, fn func() (interface{}, error)) (interface{}, error)
}

// =============================================================================
// Tool Factory
// =============================================================================

// CreateWorkspaceTools creates workspace tools that will be auto-injected into agents.
func CreateWorkspaceTools(workspace WorkspaceAccessor) (map[string]*WorkspaceTool, error) {
	tools := make(map[string]*WorkspaceTool)
	toolsConfig := workspace.GetToolsConfig()
	fs := workspace.Filesystem()
	sb := workspace.Sandbox()

	isReadOnly := false
	if fs != nil {
		isReadOnly = fs.ReadOnly()
	}

	// Helper: add a tool with config-driven filtering
	addTool := func(name string, tool *WorkspaceTool, requireWrite bool) {
		config := ResolveToolConfig(toolsConfig, name)
		if !config.Enabled {
			return
		}
		if requireWrite && isReadOnly {
			return
		}

		tool.RequireApproval = config.RequireApproval

		// Use custom name if provided
		exposedName := config.Name
		if tools[exposedName] != nil {
			panic(fmt.Sprintf(
				"Duplicate workspace tool name %q: tool %q conflicts with an already-registered tool. "+
					"Check your tools config for duplicate \"name\" values.",
				exposedName, name,
			))
		}
		if exposedName != name {
			tool.ID = exposedName
		}
		tools[exposedName] = tool
	}

	// Filesystem tools
	if fs != nil {
		addTool("mastra_workspace_read_file", &WorkspaceTool{
			ID:          "mastra_workspace_read_file",
			Description: "Read the contents of a file from the workspace filesystem.",
		}, false)

		addTool("mastra_workspace_write_file", &WorkspaceTool{
			ID:          "mastra_workspace_write_file",
			Description: "Write content to a file in the workspace filesystem.",
		}, true)

		addTool("mastra_workspace_edit_file", &WorkspaceTool{
			ID:          "mastra_workspace_edit_file",
			Description: "Edit a file by replacing specific text.",
		}, true)

		addTool("mastra_workspace_list_files", &WorkspaceTool{
			ID:          "mastra_workspace_list_files",
			Description: "List files and directories in the workspace filesystem.",
		}, false)

		addTool("mastra_workspace_delete", &WorkspaceTool{
			ID:          "mastra_workspace_delete",
			Description: "Delete a file or directory from the workspace filesystem.",
		}, true)

		addTool("mastra_workspace_file_stat", &WorkspaceTool{
			ID:          "mastra_workspace_file_stat",
			Description: "Get file or directory metadata from the workspace.",
		}, false)

		addTool("mastra_workspace_mkdir", &WorkspaceTool{
			ID:          "mastra_workspace_mkdir",
			Description: "Create a directory in the workspace filesystem.",
		}, true)

		addTool("mastra_workspace_grep", &WorkspaceTool{
			ID:          "mastra_workspace_grep",
			Description: "Search file contents using a regex pattern.",
		}, false)

		// AST edit tool (only if available)
		if IsASTGrepAvailable() {
			addTool("mastra_workspace_ast_edit", &WorkspaceTool{
				ID:          "mastra_workspace_ast_edit",
				Description: "Edit code using AST-based analysis for intelligent transformations.",
			}, true)
		}
	}

	// Search tools
	if workspace.CanBM25() || workspace.CanVector() {
		addTool("mastra_workspace_search", &WorkspaceTool{
			ID:          "mastra_workspace_search",
			Description: "Search indexed content in the workspace.",
		}, false)

		addTool("mastra_workspace_index", &WorkspaceTool{
			ID:          "mastra_workspace_index",
			Description: "Index content for search.",
		}, true)
	}

	// Sandbox tools
	if sb != nil {
		addTool("mastra_workspace_execute_command", &WorkspaceTool{
			ID:          "mastra_workspace_execute_command",
			Description: "Execute a shell command in the workspace sandbox.",
		}, false)

		// Background process tools (only when process manager is available)
		if sb.Processes() != nil {
			addTool("mastra_workspace_get_process_output", &WorkspaceTool{
				ID:          "mastra_workspace_get_process_output",
				Description: "Get the current output and status of a background process.",
			}, false)

			addTool("mastra_workspace_kill_process", &WorkspaceTool{
				ID:          "mastra_workspace_kill_process",
				Description: "Kill a background process by its PID.",
			}, false)
		}
	}

	return tools, nil
}
