package packagesource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInlineDeployPayload(t *testing.T) {
	pkg := Inline("demo", "index.ts", `output("ok");`)
	payload, err := pkg.DeployPayload()
	if err != nil {
		t.Fatalf("DeployPayload: %v", err)
	}
	if payload.Path != "" {
		t.Fatalf("Path = %q, want empty", payload.Path)
	}
	if got := payload.Files["index.ts"]; got != `output("ok");` {
		t.Fatalf("Files[index.ts] = %q", got)
	}
	var manifest map[string]string
	if err := json.Unmarshal(payload.Manifest, &manifest); err != nil {
		t.Fatalf("manifest json: %v", err)
	}
	if manifest["name"] != "demo" || manifest["entry"] != "index.ts" {
		t.Fatalf("manifest = %#v", manifest)
	}
}

func TestFromDirDeployPayloadUsesPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"name":"demo","version":"1.0.0","entry":"index.ts"}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	pkg, err := FromDir(dir)
	if err != nil {
		t.Fatalf("FromDir: %v", err)
	}
	payload, err := pkg.DeployPayload()
	if err != nil {
		t.Fatalf("DeployPayload: %v", err)
	}
	if payload.Path != dir || len(payload.Manifest) != 0 || len(payload.Files) != 0 {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestFromFileDeployPayloadUsesPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "worker.ts")
	if err := os.WriteFile(path, []byte(`output("ok");`), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	pkg, err := FromFile(path)
	if err != nil {
		t.Fatalf("FromFile: %v", err)
	}
	payload, err := pkg.DeployPayload()
	if err != nil {
		t.Fatalf("DeployPayload: %v", err)
	}
	if payload.Path != path {
		t.Fatalf("Path = %q, want %q", payload.Path, path)
	}
}
