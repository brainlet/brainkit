// Ported from: packages/core/src/workspace/constants/constants.test.ts
package constants

import (
	"strings"
	"testing"
)

func TestWorkspaceToolsPrefix(t *testing.T) {
	t.Run("prefix is mastra_workspace", func(t *testing.T) {
		if WorkspaceToolsPrefix != "mastra_workspace" {
			t.Errorf("WorkspaceToolsPrefix = %q, want %q", WorkspaceToolsPrefix, "mastra_workspace")
		}
	})
}

func TestFilesystemTools(t *testing.T) {
	t.Run("all filesystem tool names start with prefix", func(t *testing.T) {
		tools := []struct {
			name  string
			value string
		}{
			{"ReadFile", FilesystemTools.ReadFile},
			{"WriteFile", FilesystemTools.WriteFile},
			{"EditFile", FilesystemTools.EditFile},
			{"ListFiles", FilesystemTools.ListFiles},
			{"Delete", FilesystemTools.Delete},
			{"FileStat", FilesystemTools.FileStat},
			{"Mkdir", FilesystemTools.Mkdir},
			{"Grep", FilesystemTools.Grep},
			{"AstEdit", FilesystemTools.AstEdit},
		}
		for _, tc := range tools {
			t.Run(tc.name, func(t *testing.T) {
				if !strings.HasPrefix(tc.value, WorkspaceToolsPrefix+"_") {
					t.Errorf("FilesystemTools.%s = %q, want prefix %q", tc.name, tc.value, WorkspaceToolsPrefix+"_")
				}
			})
		}
	})

	t.Run("ReadFile has correct name", func(t *testing.T) {
		want := "mastra_workspace_read_file"
		if FilesystemTools.ReadFile != want {
			t.Errorf("FilesystemTools.ReadFile = %q, want %q", FilesystemTools.ReadFile, want)
		}
	})

	t.Run("WriteFile has correct name", func(t *testing.T) {
		want := "mastra_workspace_write_file"
		if FilesystemTools.WriteFile != want {
			t.Errorf("FilesystemTools.WriteFile = %q, want %q", FilesystemTools.WriteFile, want)
		}
	})

	t.Run("EditFile has correct name", func(t *testing.T) {
		want := "mastra_workspace_edit_file"
		if FilesystemTools.EditFile != want {
			t.Errorf("FilesystemTools.EditFile = %q, want %q", FilesystemTools.EditFile, want)
		}
	})

	t.Run("ListFiles has correct name", func(t *testing.T) {
		want := "mastra_workspace_list_files"
		if FilesystemTools.ListFiles != want {
			t.Errorf("FilesystemTools.ListFiles = %q, want %q", FilesystemTools.ListFiles, want)
		}
	})
}

func TestSandboxTools(t *testing.T) {
	t.Run("all sandbox tool names start with prefix", func(t *testing.T) {
		tools := []struct {
			name  string
			value string
		}{
			{"ExecuteCommand", SandboxTools.ExecuteCommand},
			{"GetProcessOutput", SandboxTools.GetProcessOutput},
			{"KillProcess", SandboxTools.KillProcess},
		}
		for _, tc := range tools {
			t.Run(tc.name, func(t *testing.T) {
				if !strings.HasPrefix(tc.value, WorkspaceToolsPrefix+"_") {
					t.Errorf("SandboxTools.%s = %q, want prefix %q", tc.name, tc.value, WorkspaceToolsPrefix+"_")
				}
			})
		}
	})

	t.Run("ExecuteCommand has correct name", func(t *testing.T) {
		want := "mastra_workspace_execute_command"
		if SandboxTools.ExecuteCommand != want {
			t.Errorf("SandboxTools.ExecuteCommand = %q, want %q", SandboxTools.ExecuteCommand, want)
		}
	})
}

func TestSearchTools(t *testing.T) {
	t.Run("all search tool names start with prefix", func(t *testing.T) {
		tools := []struct {
			name  string
			value string
		}{
			{"Search", SearchTools.Search},
			{"Index", SearchTools.Index},
		}
		for _, tc := range tools {
			t.Run(tc.name, func(t *testing.T) {
				if !strings.HasPrefix(tc.value, WorkspaceToolsPrefix+"_") {
					t.Errorf("SearchTools.%s = %q, want prefix %q", tc.name, tc.value, WorkspaceToolsPrefix+"_")
				}
			})
		}
	})

	t.Run("Search has correct name", func(t *testing.T) {
		want := "mastra_workspace_search"
		if SearchTools.Search != want {
			t.Errorf("SearchTools.Search = %q, want %q", SearchTools.Search, want)
		}
	})

	t.Run("Index has correct name", func(t *testing.T) {
		want := "mastra_workspace_index"
		if SearchTools.Index != want {
			t.Errorf("SearchTools.Index = %q, want %q", SearchTools.Index, want)
		}
	})
}

func TestWorkspaceToolsComposite(t *testing.T) {
	t.Run("Filesystem tools match individual FilesystemTools", func(t *testing.T) {
		if WorkspaceTools.Filesystem.ReadFile != FilesystemTools.ReadFile {
			t.Errorf("WorkspaceTools.Filesystem.ReadFile = %q, want %q", WorkspaceTools.Filesystem.ReadFile, FilesystemTools.ReadFile)
		}
		if WorkspaceTools.Filesystem.WriteFile != FilesystemTools.WriteFile {
			t.Errorf("WorkspaceTools.Filesystem.WriteFile = %q, want %q", WorkspaceTools.Filesystem.WriteFile, FilesystemTools.WriteFile)
		}
		if WorkspaceTools.Filesystem.EditFile != FilesystemTools.EditFile {
			t.Errorf("WorkspaceTools.Filesystem.EditFile = %q, want %q", WorkspaceTools.Filesystem.EditFile, FilesystemTools.EditFile)
		}
		if WorkspaceTools.Filesystem.Grep != FilesystemTools.Grep {
			t.Errorf("WorkspaceTools.Filesystem.Grep = %q, want %q", WorkspaceTools.Filesystem.Grep, FilesystemTools.Grep)
		}
	})

	t.Run("Sandbox tools match individual SandboxTools", func(t *testing.T) {
		if WorkspaceTools.Sandbox.ExecuteCommand != SandboxTools.ExecuteCommand {
			t.Errorf("WorkspaceTools.Sandbox.ExecuteCommand = %q, want %q", WorkspaceTools.Sandbox.ExecuteCommand, SandboxTools.ExecuteCommand)
		}
		if WorkspaceTools.Sandbox.GetProcessOutput != SandboxTools.GetProcessOutput {
			t.Errorf("WorkspaceTools.Sandbox.GetProcessOutput = %q, want %q", WorkspaceTools.Sandbox.GetProcessOutput, SandboxTools.GetProcessOutput)
		}
		if WorkspaceTools.Sandbox.KillProcess != SandboxTools.KillProcess {
			t.Errorf("WorkspaceTools.Sandbox.KillProcess = %q, want %q", WorkspaceTools.Sandbox.KillProcess, SandboxTools.KillProcess)
		}
	})

	t.Run("Search tools match individual SearchTools", func(t *testing.T) {
		if WorkspaceTools.Search.Search != SearchTools.Search {
			t.Errorf("WorkspaceTools.Search.Search = %q, want %q", WorkspaceTools.Search.Search, SearchTools.Search)
		}
		if WorkspaceTools.Search.Index != SearchTools.Index {
			t.Errorf("WorkspaceTools.Search.Index = %q, want %q", WorkspaceTools.Search.Index, SearchTools.Index)
		}
	})
}

func TestAllToolNamesAreUnique(t *testing.T) {
	allNames := []string{
		FilesystemTools.ReadFile,
		FilesystemTools.WriteFile,
		FilesystemTools.EditFile,
		FilesystemTools.ListFiles,
		FilesystemTools.Delete,
		FilesystemTools.FileStat,
		FilesystemTools.Mkdir,
		FilesystemTools.Grep,
		FilesystemTools.AstEdit,
		SandboxTools.ExecuteCommand,
		SandboxTools.GetProcessOutput,
		SandboxTools.KillProcess,
		SearchTools.Search,
		SearchTools.Index,
	}

	seen := make(map[string]bool, len(allNames))
	for _, name := range allNames {
		if seen[name] {
			t.Errorf("duplicate tool name: %q", name)
		}
		seen[name] = true
	}
}
