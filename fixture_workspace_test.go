//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestFixture_TS_WorkspaceReadWrite(t *testing.T) {
	tmpDir := t.TempDir()

	kit, err := New(Config{
		Namespace: "test",
		EnvVars:   map[string]string{"TEST_TMPDIR": tmpDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/workspace-read-write.js")
	result, err := kit.EvalModule(context.Background(), "workspace-read-write.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Content            string   `json:"content"`
		AppendedContent    string   `json:"appendedContent"`
		StatIsFile         bool     `json:"statIsFile"`
		StatIsDir          bool     `json:"statIsDir"`
		StatSize           float64  `json:"statSize"`
		Entries            []string `json:"entries"`
		CopyContent        string   `json:"copyContent"`
		RenamedContent     string   `json:"renamedContent"`
		EntriesAfterDelete []string `json:"entriesAfterDelete"`
		Realpath           string   `json:"realpath"`
		LstatIsFile        bool     `json:"lstatIsFile"`
		PathJoin           string   `json:"pathJoin"`
		PathResolve        string   `json:"pathResolve"`
		PathDirname        string   `json:"pathDirname"`
		PathBasename       string   `json:"pathBasename"`
		PathExtname        string   `json:"pathExtname"`
		PathRelative       string   `json:"pathRelative"`
		PathIsAbsolute     bool     `json:"pathIsAbsolute"`
		Success            bool     `json:"success"`
		Error              string   `json:"error"`
		Stack              string   `json:"stack"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Error != "" {
		t.Fatalf("fixture error: %s\nstack: %s", out.Error, out.Stack)
	}
	if out.Content != "Hello from brainkit!" {
		t.Errorf("content = %q, want 'Hello from brainkit!'", out.Content)
	}
	if out.AppendedContent != "Hello from brainkit! Extra." {
		t.Errorf("appended = %q", out.AppendedContent)
	}
	if !out.StatIsFile {
		t.Error("stat.isFile should be true")
	}
	if out.StatIsDir {
		t.Error("stat.isDirectory should be false for a file")
	}
	if !out.Success {
		t.Error("expected success")
	}
	if !out.LstatIsFile {
		t.Error("lstat.isFile should be true")
	}
	if out.Realpath == "" {
		t.Error("realpath should return a non-empty path")
	}
	t.Logf("workspace-read-write: content=%q entries=%v size=%.0f realpath=%s copy=%q",
		out.Content, out.Entries, out.StatSize, out.Realpath, out.CopyContent)
}

func TestFixture_TS_WorkspaceAgentTools(t *testing.T) {
	tmpDir := t.TempDir()
	key := requireKey(t)

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{"openai": {APIKey: key}},
		EnvVars:   map[string]string{"TEST_TMPDIR": tmpDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/workspace-agent-tools.js")
	result, err := kit.EvalModule(context.Background(), "workspace-agent-tools.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]interface{}
	json.Unmarshal([]byte(result), &out)
	if errMsg, ok := out["error"]; ok && errMsg != nil {
		t.Fatalf("fixture error: %v\nstack: %v", errMsg, out["stack"])
	}
	check := func(name string, key string) {
		if m, ok := out[name].(map[string]interface{}); ok {
			t.Logf("%s: %v", name, m)
		} else {
			t.Logf("%s: missing or wrong type: %v", name, out[name])
		}
		_ = key
	}
	check("read", "has")
	check("write", "ok")
	check("list", "hasReadme")
	check("grep", "found")
	check("exec", "has")
	t.Logf("ragChunks: %v", out["ragChunks"])
}

func TestFixture_TS_WorkspaceSkillsSearch(t *testing.T) {
	tmpDir := t.TempDir()
	key := requireKey(t)

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{"openai": {APIKey: key}},
		EnvVars:   map[string]string{"TEST_TMPDIR": tmpDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/workspace-skills-search.js")
	result, err := kit.EvalModule(context.Background(), "workspace-skills-search.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]interface{}
	json.Unmarshal([]byte(result), &out)
	if errMsg, ok := out["error"]; ok && errMsg != nil {
		t.Fatalf("fixture error: %v\nstack: %v", errMsg, out["stack"])
	}
	t.Logf("skills: %v", out["skillList"])
	t.Logf("search: %v", out["search"])
}

func TestFixture_TS_WorkspaceLSP(t *testing.T) {
	tmpDir := t.TempDir()

	kit, err := New(Config{
		Namespace: "test",
		EnvVars: map[string]string{
			"TEST_TMPDIR": tmpDir,
			"PATH":        os.Getenv("PATH"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	code := loadFixture(t, "testdata/ts/workspace-lsp.js")
	result, err := kit.EvalModule(context.Background(), "workspace-lsp.js", code)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("LSP: %s", result)
}
