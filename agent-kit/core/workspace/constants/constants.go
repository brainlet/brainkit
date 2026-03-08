// Ported from: packages/core/src/workspace/constants/constants.ts
package constants

// WorkspaceToolsPrefix is the prefix for all workspace tool names.
const WorkspaceToolsPrefix = "mastra_workspace"

// WorkspaceToolName represents any workspace tool name.
type WorkspaceToolName = string

// Filesystem tool name constants.
var FilesystemTools = struct {
	ReadFile string
	WriteFile string
	EditFile string
	ListFiles string
	Delete   string
	FileStat string
	Mkdir    string
	Grep     string
	AstEdit  string
}{
	ReadFile:  WorkspaceToolsPrefix + "_read_file",
	WriteFile: WorkspaceToolsPrefix + "_write_file",
	EditFile:  WorkspaceToolsPrefix + "_edit_file",
	ListFiles: WorkspaceToolsPrefix + "_list_files",
	Delete:    WorkspaceToolsPrefix + "_delete",
	FileStat:  WorkspaceToolsPrefix + "_file_stat",
	Mkdir:     WorkspaceToolsPrefix + "_mkdir",
	Grep:      WorkspaceToolsPrefix + "_grep",
	AstEdit:   WorkspaceToolsPrefix + "_ast_edit",
}

// Sandbox tool name constants.
var SandboxTools = struct {
	ExecuteCommand   string
	GetProcessOutput string
	KillProcess      string
}{
	ExecuteCommand:   WorkspaceToolsPrefix + "_execute_command",
	GetProcessOutput: WorkspaceToolsPrefix + "_get_process_output",
	KillProcess:      WorkspaceToolsPrefix + "_kill_process",
}

// Search tool name constants.
var SearchTools = struct {
	Search string
	Index  string
}{
	Search: WorkspaceToolsPrefix + "_search",
	Index:  WorkspaceToolsPrefix + "_index",
}

// WorkspaceTools groups all workspace tool name constants.
var WorkspaceTools = struct {
	Filesystem struct {
		ReadFile  string
		WriteFile string
		EditFile  string
		ListFiles string
		Delete    string
		FileStat  string
		Mkdir     string
		Grep      string
		AstEdit   string
	}
	Sandbox struct {
		ExecuteCommand   string
		GetProcessOutput string
		KillProcess      string
	}
	Search struct {
		Search string
		Index  string
	}
}{
	Filesystem: struct {
		ReadFile  string
		WriteFile string
		EditFile  string
		ListFiles string
		Delete    string
		FileStat  string
		Mkdir     string
		Grep      string
		AstEdit   string
	}{
		ReadFile:  FilesystemTools.ReadFile,
		WriteFile: FilesystemTools.WriteFile,
		EditFile:  FilesystemTools.EditFile,
		ListFiles: FilesystemTools.ListFiles,
		Delete:    FilesystemTools.Delete,
		FileStat:  FilesystemTools.FileStat,
		Mkdir:     FilesystemTools.Mkdir,
		Grep:      FilesystemTools.Grep,
		AstEdit:   FilesystemTools.AstEdit,
	},
	Sandbox: struct {
		ExecuteCommand   string
		GetProcessOutput string
		KillProcess      string
	}{
		ExecuteCommand:   SandboxTools.ExecuteCommand,
		GetProcessOutput: SandboxTools.GetProcessOutput,
		KillProcess:      SandboxTools.KillProcess,
	},
	Search: struct {
		Search string
		Index  string
	}{
		Search: SearchTools.Search,
		Index:  SearchTools.Index,
	},
}
