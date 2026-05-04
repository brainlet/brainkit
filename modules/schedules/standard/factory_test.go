package standard

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	bkmodule "github.com/brainlet/brainkit/module"
	"gopkg.in/yaml.v3"
)

func TestFactoryBuildWithPathOpensOwnedStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schedules.db")
	mod, err := (Factory{}).Build(bkmodule.BuildContext{
		Decode: yamlDecode(t, "path: "+path+"\n"),
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if mod.ID() != "schedules" {
		t.Fatalf("module ID = %q, want schedules", mod.ID())
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat opened schedule store: %v", err)
	}
	closer, ok := mod.(interface {
		CloseContext(context.Context) error
	})
	if !ok {
		t.Fatalf("%T does not expose CloseContext", mod)
	}
	if err := closer.CloseContext(context.Background()); err != nil {
		t.Fatalf("close owned store: %v", err)
	}
}

func TestFactoryBuildWithoutPathUsesSharedKitStoreAtMount(t *testing.T) {
	mod, err := (Factory{}).Build(bkmodule.BuildContext{
		Decode: yamlDecode(t, "{}\n"),
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if mod.ID() != "schedules" {
		t.Fatalf("module ID = %q, want schedules", mod.ID())
	}
}

func yamlDecode(t *testing.T, text string) func(any) error {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatalf("parse yaml: %v", err)
	}
	return func(v any) error {
		if len(doc.Content) == 0 {
			return nil
		}
		return doc.Content[0].Decode(v)
	}
}
