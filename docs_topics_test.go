package brainkit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestBusTopicsCatalogMatchesMessageDefinitions(t *testing.T) {
	actual := collectBusTopics(t)
	docTopics := readBusTopicCatalog(t)

	if len(docTopics) != len(actual) {
		t.Fatalf("docs/bus-topics.md has %d topics, want %d", len(docTopics), len(actual))
	}
	for topic := range actual {
		if _, ok := docTopics[topic]; !ok {
			t.Fatalf("docs/bus-topics.md missing topic %q", topic)
		}
	}
	for topic := range docTopics {
		if _, ok := actual[topic]; !ok {
			t.Fatalf("docs/bus-topics.md contains stale topic %q", topic)
		}
	}
}

func TestDocsDoNotUseKnownRenamedTopicOrRootModuleAliases(t *testing.T) {
	banned := []string{
		"`schedule.create`",
		"`schedule.cancel`",
		"`schedule.list`",
		"`schedule.*`",
		"`kit.teardowned`",
		"brainkit.Module",
		"[]brainkit.Module",
		"[]Module",
	}
	for _, root := range []string{"docs", "modules"} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if root == "modules" && path != "modules" && filepath.Dir(path) != "modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if !(strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".go")) {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(data)
			for _, pattern := range banned {
				if strings.Contains(text, pattern) {
					t.Fatalf("%s contains stale docs/API text %q", path, pattern)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func collectBusTopics(t *testing.T) map[string]struct{} {
	t.Helper()
	out := map[string]struct{}{}
	for _, root := range []string{"sdk", "modules"} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), "_messages.go") {
				return nil
			}
			for _, topic := range parseBusTopics(t, path) {
				out[topic] = struct{}{}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func parseBusTopics(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	constStrings := map[string]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		decl, ok := n.(*ast.GenDecl)
		if !ok || decl.Tok != token.CONST {
			return true
		}
		for _, spec := range decl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if i >= len(vs.Values) {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(lit.Value)
				if err == nil {
					constStrings[name.Name] = value
				}
			}
		}
		return true
	})

	var topics []string
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "BusTopic" || fn.Body == nil || len(fn.Body.List) == 0 {
			return true
		}
		ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			return true
		}
		switch v := ret.Results[0].(type) {
		case *ast.BasicLit:
			if v.Kind == token.STRING {
				if topic, err := strconv.Unquote(v.Value); err == nil {
					topics = append(topics, topic)
				}
			}
		case *ast.Ident:
			if topic := constStrings[v.Name]; topic != "" {
				topics = append(topics, topic)
			}
		}
		return true
	})
	return topics
}

func readBusTopicCatalog(t *testing.T) map[string]struct{} {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("docs", "bus-topics.md"))
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile("^\\| `([^`]+)` \\|")
	out := map[string]struct{}{}
	for _, line := range strings.Split(string(data), "\n") {
		match := re.FindStringSubmatch(line)
		if len(match) == 2 {
			out[match[1]] = struct{}{}
		}
	}
	if len(out) == 0 {
		t.Fatal("docs/bus-topics.md contains no topic rows")
	}
	return out
}
