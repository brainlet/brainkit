package harness

// PermissionPolicy determines how a tool category is handled during approval.
type PermissionPolicy string

const (
	PolicyAllow PermissionPolicy = "allow" // auto-approve, no user interaction
	PolicyAsk   PermissionPolicy = "ask"   // pause, emit tool_approval_required, wait
	PolicyDeny  PermissionPolicy = "deny"  // auto-decline
)

// ToolCategory groups tools by risk level for permission resolution.
type ToolCategory string

const (
	CategoryRead    ToolCategory = "read"    // view, search, find_files
	CategoryEdit    ToolCategory = "edit"    // write_file, string_replace
	CategoryExecute ToolCategory = "execute" // execute_command
	CategoryMCP     ToolCategory = "mcp"     // all MCP server tools
)

// DefaultPermissions returns the standard permission set:
// read=allow, edit=ask, execute=ask, mcp=ask.
func DefaultPermissions() map[ToolCategory]PermissionPolicy {
	return map[ToolCategory]PermissionPolicy{
		CategoryRead:    PolicyAllow,
		CategoryEdit:    PolicyAsk,
		CategoryExecute: PolicyAsk,
		CategoryMCP:     PolicyAsk,
	}
}
