package brainkit

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/brainlet/brainkit/bus"
)

func newFsTestKit(t *testing.T) *Kit {
	t.Helper()
	workspace := t.TempDir()
	kit, err := New(Config{Namespace: "test", WorkspaceDir: workspace})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { kit.Close() })
	return kit
}

func TestFsHandler_WriteAndRead(t *testing.T) {
	kit := newFsTestKit(t)

	// Write
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "fs.write",
		Payload: json.RawMessage(`{"path":"test.txt","data":"hello world"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var writeResult struct{ OK bool `json:"ok"` }
	json.Unmarshal(resp.Payload, &writeResult)
	if !writeResult.OK {
		t.Fatalf("write: %s", resp.Payload)
	}

	// Read
	resp, err = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "fs.read",
		Payload: json.RawMessage(`{"path":"test.txt"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var readResult struct{ Data string `json:"data"` }
	json.Unmarshal(resp.Payload, &readResult)
	if readResult.Data != "hello world" {
		t.Errorf("read = %q", readResult.Data)
	}
}

func TestFsHandler_List(t *testing.T) {
	kit := newFsTestKit(t)

	// Create some files
	os.WriteFile(filepath.Join(kit.config.WorkspaceDir, "a.txt"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(kit.config.WorkspaceDir, "b.txt"), []byte("b"), 0o644)
	os.Mkdir(filepath.Join(kit.config.WorkspaceDir, "subdir"), 0o755)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "fs.list",
		Payload: json.RawMessage(`{"path":"."}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var listResult struct {
		Files []struct {
			Name  string `json:"name"`
			IsDir bool   `json:"isDir"`
		} `json:"files"`
	}
	json.Unmarshal(resp.Payload, &listResult)
	if len(listResult.Files) != 3 {
		t.Errorf("expected 3 entries, got %d", len(listResult.Files))
	}
}

func TestFsHandler_ListWithPattern(t *testing.T) {
	kit := newFsTestKit(t)

	os.WriteFile(filepath.Join(kit.config.WorkspaceDir, "a.txt"), []byte("a"), 0o644)
	os.WriteFile(filepath.Join(kit.config.WorkspaceDir, "b.go"), []byte("b"), 0o644)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "fs.list",
		Payload: json.RawMessage(`{"path":".","pattern":"*.txt"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var listResult struct {
		Files []struct{ Name string `json:"name"` } `json:"files"`
	}
	json.Unmarshal(resp.Payload, &listResult)
	if len(listResult.Files) != 1 || listResult.Files[0].Name != "a.txt" {
		t.Errorf("expected [a.txt], got %v", listResult.Files)
	}
}

func TestFsHandler_Stat(t *testing.T) {
	kit := newFsTestKit(t)
	os.WriteFile(filepath.Join(kit.config.WorkspaceDir, "stat.txt"), []byte("12345"), 0o644)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "fs.stat",
		Payload: json.RawMessage(`{"path":"stat.txt"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var statResult struct {
		Size  int64 `json:"size"`
		IsDir bool  `json:"isDir"`
	}
	json.Unmarshal(resp.Payload, &statResult)
	if statResult.Size != 5 {
		t.Errorf("size = %d, want 5", statResult.Size)
	}
	if statResult.IsDir {
		t.Error("expected file, got dir")
	}
}

func TestFsHandler_MkdirAndDelete(t *testing.T) {
	kit := newFsTestKit(t)

	// Mkdir
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "fs.mkdir",
		Payload: json.RawMessage(`{"path":"deep/nested/dir"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var mkResult struct{ OK bool `json:"ok"` }
	json.Unmarshal(resp.Payload, &mkResult)
	if !mkResult.OK {
		t.Fatal("mkdir failed")
	}

	// Verify exists
	info, err := os.Stat(filepath.Join(kit.config.WorkspaceDir, "deep", "nested", "dir"))
	if err != nil || !info.IsDir() {
		t.Fatal("directory not created")
	}

	// Write a file then delete
	os.WriteFile(filepath.Join(kit.config.WorkspaceDir, "todelete.txt"), []byte("x"), 0o644)
	resp, err = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "fs.delete",
		Payload: json.RawMessage(`{"path":"todelete.txt"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(kit.config.WorkspaceDir, "todelete.txt")); !os.IsNotExist(err) {
		t.Error("file not deleted")
	}
}

func TestFsHandler_PathEscapePrevention(t *testing.T) {
	kit := newFsTestKit(t)

	// Try to escape workspace
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "fs.read",
		Payload: json.RawMessage(`{"path":"../../etc/passwd"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var errResult struct{ Error string `json:"error"` }
	json.Unmarshal(resp.Payload, &errResult)
	if errResult.Error == "" {
		t.Fatal("expected path escape error")
	}
}
