package standard

import (
	"strings"
	"testing"

	bkmodule "github.com/brainlet/brainkit/module"
	"gopkg.in/yaml.v3"
)

func TestFactoryRejectsPostgresWithoutPostgresBackendImport(t *testing.T) {
	_, err := (Factory{}).Build(bkmodule.BuildContext{
		Decode: yamlDecode(t, "type: postgres\nconnection_string: postgres://example\n"),
	})
	if err == nil {
		t.Fatal("Build with postgres type succeeded; want backend import hint")
	}
	if !strings.Contains(err.Error(), "modules/audit/standard/postgres") {
		t.Fatalf("Build error = %v, want modules/audit/standard/postgres hint", err)
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
