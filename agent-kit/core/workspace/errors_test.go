// Ported from: packages/core/src/workspace/errors.test.ts
package workspace

import (
	"strings"
	"testing"
)

func TestWorkspaceError(t *testing.T) {
	t.Run("NewWorkspaceError creates error with message and code", func(t *testing.T) {
		err := NewWorkspaceError("test message", "TEST_CODE")
		if err.Message != "test message" {
			t.Errorf("Message = %q, want %q", err.Message, "test message")
		}
		if err.Code != "TEST_CODE" {
			t.Errorf("Code = %q, want %q", err.Code, "TEST_CODE")
		}
		if err.WorkspaceID != "" {
			t.Errorf("WorkspaceID = %q, want empty", err.WorkspaceID)
		}
	})

	t.Run("NewWorkspaceError with workspace ID", func(t *testing.T) {
		err := NewWorkspaceError("msg", "CODE", "ws-123")
		if err.WorkspaceID != "ws-123" {
			t.Errorf("WorkspaceID = %q, want %q", err.WorkspaceID, "ws-123")
		}
	})

	t.Run("Error returns message", func(t *testing.T) {
		err := NewWorkspaceError("something failed", "FAIL")
		if err.Error() != "something failed" {
			t.Errorf("Error() = %q, want %q", err.Error(), "something failed")
		}
	})
}

func TestWorkspaceNotAvailableError(t *testing.T) {
	t.Run("has correct message and code", func(t *testing.T) {
		err := NewWorkspaceNotAvailableError()
		if err.Code != "NO_WORKSPACE" {
			t.Errorf("Code = %q, want %q", err.Code, "NO_WORKSPACE")
		}
		if !strings.Contains(err.Message, "not available") {
			t.Errorf("Message should contain 'not available', got %q", err.Message)
		}
	})
}

func TestFilesystemNotAvailableError(t *testing.T) {
	t.Run("has correct message and code", func(t *testing.T) {
		err := NewFilesystemNotAvailableError()
		if err.Code != "NO_FILESYSTEM" {
			t.Errorf("Code = %q, want %q", err.Code, "NO_FILESYSTEM")
		}
		if !strings.Contains(err.Message, "filesystem") {
			t.Errorf("Message should contain 'filesystem', got %q", err.Message)
		}
	})
}

func TestSandboxNotAvailableError(t *testing.T) {
	t.Run("uses default message when none provided", func(t *testing.T) {
		err := NewSandboxNotAvailableError()
		if err.Code != "NO_SANDBOX" {
			t.Errorf("Code = %q, want %q", err.Code, "NO_SANDBOX")
		}
		if !strings.Contains(err.Message, "sandbox") {
			t.Errorf("Message should contain 'sandbox', got %q", err.Message)
		}
	})

	t.Run("uses custom message when provided", func(t *testing.T) {
		err := NewSandboxNotAvailableError("custom sandbox error")
		if err.Message != "custom sandbox error" {
			t.Errorf("Message = %q, want %q", err.Message, "custom sandbox error")
		}
		if err.Code != "NO_SANDBOX" {
			t.Errorf("Code = %q, want %q", err.Code, "NO_SANDBOX")
		}
	})

	t.Run("ignores empty string message", func(t *testing.T) {
		err := NewSandboxNotAvailableError("")
		if !strings.Contains(err.Message, "sandbox") {
			t.Errorf("empty message should use default, got %q", err.Message)
		}
	})
}

func TestSandboxFeatureNotSupportedError(t *testing.T) {
	t.Run("includes feature name in message", func(t *testing.T) {
		err := NewSandboxFeatureNotSupportedError(SandboxFeatureExecuteCommand)
		if err.Code != "FEATURE_NOT_SUPPORTED" {
			t.Errorf("Code = %q, want %q", err.Code, "FEATURE_NOT_SUPPORTED")
		}
		if err.Feature != SandboxFeatureExecuteCommand {
			t.Errorf("Feature = %q, want %q", err.Feature, SandboxFeatureExecuteCommand)
		}
		if !strings.Contains(err.Message, string(SandboxFeatureExecuteCommand)) {
			t.Errorf("Message should contain feature name, got %q", err.Message)
		}
	})

	t.Run("SandboxFeature constants are correct", func(t *testing.T) {
		if SandboxFeatureExecuteCommand != "executeCommand" {
			t.Errorf("SandboxFeatureExecuteCommand = %q, want %q", SandboxFeatureExecuteCommand, "executeCommand")
		}
		if SandboxFeatureInstallPackage != "installPackage" {
			t.Errorf("SandboxFeatureInstallPackage = %q, want %q", SandboxFeatureInstallPackage, "installPackage")
		}
		if SandboxFeatureProcesses != "processes" {
			t.Errorf("SandboxFeatureProcesses = %q, want %q", SandboxFeatureProcesses, "processes")
		}
	})
}

func TestSearchNotAvailableError(t *testing.T) {
	t.Run("has correct message and code", func(t *testing.T) {
		err := NewSearchNotAvailableError()
		if err.Code != "NO_SEARCH" {
			t.Errorf("Code = %q, want %q", err.Code, "NO_SEARCH")
		}
		if !strings.Contains(err.Message, "search") {
			t.Errorf("Message should contain 'search', got %q", err.Message)
		}
	})
}

func TestWorkspaceNotReadyError(t *testing.T) {
	t.Run("includes workspace ID and status", func(t *testing.T) {
		err := NewWorkspaceNotReadyError("ws-abc", WorkspaceStatusInitializing)
		if err.Code != "NOT_READY" {
			t.Errorf("Code = %q, want %q", err.Code, "NOT_READY")
		}
		if err.WorkspaceID != "ws-abc" {
			t.Errorf("WorkspaceID = %q, want %q", err.WorkspaceID, "ws-abc")
		}
		if !strings.Contains(err.Message, string(WorkspaceStatusInitializing)) {
			t.Errorf("Message should contain status, got %q", err.Message)
		}
	})
}

func TestWorkspaceReadOnlyError(t *testing.T) {
	t.Run("includes operation in message", func(t *testing.T) {
		err := NewWorkspaceReadOnlyError("write")
		if err.Code != "READ_ONLY" {
			t.Errorf("Code = %q, want %q", err.Code, "READ_ONLY")
		}
		if err.Operation != "write" {
			t.Errorf("Operation = %q, want %q", err.Operation, "write")
		}
		if !strings.Contains(err.Message, "write") {
			t.Errorf("Message should contain operation, got %q", err.Message)
		}
	})
}

func TestFilesystemError(t *testing.T) {
	t.Run("NewFilesystemError creates error with all fields", func(t *testing.T) {
		err := NewFilesystemError("not found", "ENOENT", "/foo/bar")
		if err.Message != "not found" {
			t.Errorf("Message = %q, want %q", err.Message, "not found")
		}
		if err.Code != "ENOENT" {
			t.Errorf("Code = %q, want %q", err.Code, "ENOENT")
		}
		if err.Path != "/foo/bar" {
			t.Errorf("Path = %q, want %q", err.Path, "/foo/bar")
		}
	})

	t.Run("Error returns message", func(t *testing.T) {
		err := NewFilesystemError("test error", "TEST", "/path")
		if err.Error() != "test error" {
			t.Errorf("Error() = %q, want %q", err.Error(), "test error")
		}
	})
}

func TestFileNotFoundError(t *testing.T) {
	t.Run("includes path in message", func(t *testing.T) {
		err := NewFileNotFoundError("/foo/bar.txt")
		if err.Code != "ENOENT" {
			t.Errorf("Code = %q, want %q", err.Code, "ENOENT")
		}
		if err.Path != "/foo/bar.txt" {
			t.Errorf("Path = %q, want %q", err.Path, "/foo/bar.txt")
		}
		if !strings.Contains(err.Message, "/foo/bar.txt") {
			t.Errorf("Message should contain path, got %q", err.Message)
		}
	})
}

func TestDirectoryNotFoundError(t *testing.T) {
	t.Run("includes path in message", func(t *testing.T) {
		err := NewDirectoryNotFoundError("/some/dir")
		if err.Code != "ENOENT" {
			t.Errorf("Code = %q, want %q", err.Code, "ENOENT")
		}
		if !strings.Contains(err.Message, "/some/dir") {
			t.Errorf("Message should contain path, got %q", err.Message)
		}
	})
}

func TestFileExistsError(t *testing.T) {
	t.Run("includes path in message", func(t *testing.T) {
		err := NewFileExistsError("/foo/exists.txt")
		if err.Code != "EEXIST" {
			t.Errorf("Code = %q, want %q", err.Code, "EEXIST")
		}
		if !strings.Contains(err.Message, "/foo/exists.txt") {
			t.Errorf("Message should contain path, got %q", err.Message)
		}
	})
}

func TestIsDirectoryError(t *testing.T) {
	t.Run("has EISDIR code", func(t *testing.T) {
		err := NewIsDirectoryError("/some/dir")
		if err.Code != "EISDIR" {
			t.Errorf("Code = %q, want %q", err.Code, "EISDIR")
		}
	})
}

func TestNotDirectoryError(t *testing.T) {
	t.Run("has ENOTDIR code", func(t *testing.T) {
		err := NewNotDirectoryError("/some/file")
		if err.Code != "ENOTDIR" {
			t.Errorf("Code = %q, want %q", err.Code, "ENOTDIR")
		}
	})
}

func TestDirectoryNotEmptyError(t *testing.T) {
	t.Run("has ENOTEMPTY code", func(t *testing.T) {
		err := NewDirectoryNotEmptyError("/notempty")
		if err.Code != "ENOTEMPTY" {
			t.Errorf("Code = %q, want %q", err.Code, "ENOTEMPTY")
		}
	})
}

func TestPermissionError(t *testing.T) {
	t.Run("includes path and operation", func(t *testing.T) {
		err := NewPermissionError("/secret", "read")
		if err.Code != "EACCES" {
			t.Errorf("Code = %q, want %q", err.Code, "EACCES")
		}
		if err.Operation != "read" {
			t.Errorf("Operation = %q, want %q", err.Operation, "read")
		}
		if !strings.Contains(err.Message, "read") || !strings.Contains(err.Message, "/secret") {
			t.Errorf("Message should contain operation and path, got %q", err.Message)
		}
	})
}

func TestFileReadRequiredError(t *testing.T) {
	t.Run("stores path and reason", func(t *testing.T) {
		err := NewFileReadRequiredError("/file.txt", "must read before edit")
		if err.Code != "EREAD_REQUIRED" {
			t.Errorf("Code = %q, want %q", err.Code, "EREAD_REQUIRED")
		}
		if err.Path != "/file.txt" {
			t.Errorf("Path = %q, want %q", err.Path, "/file.txt")
		}
		if err.Message != "must read before edit" {
			t.Errorf("Message = %q, want %q", err.Message, "must read before edit")
		}
	})
}

func TestFilesystemNotReadyError(t *testing.T) {
	t.Run("includes ID in message", func(t *testing.T) {
		err := NewFilesystemNotReadyError("local-fs")
		if err.Code != "ENOTREADY" {
			t.Errorf("Code = %q, want %q", err.Code, "ENOTREADY")
		}
		if !strings.Contains(err.Message, "local-fs") {
			t.Errorf("Message should contain ID, got %q", err.Message)
		}
	})
}
