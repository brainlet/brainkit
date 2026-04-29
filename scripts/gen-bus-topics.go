//go:build ignore

// Command gen-bus-topics scans sdk/**/*_messages.go and
// modules/**/*_messages.go for BusTopic() string constants and emits a
// Markdown table under
// docs/bus-topics.md. Run from the repo root:
//
//	go run scripts/gen-bus-topics.go
//
// The tool parses the source files with go/parser — it does not
// execute any code from the SDK, so it stays fast and side-effect
// free.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type entry struct {
	Msg   string
	Topic string
	Resp  string
	File  string
}

func main() {
	repo, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	var topics []entry
	files, err := messageFiles(repo)
	if err != nil {
		fail(err)
	}
	for _, file := range files {
		topics = append(topics, parseFile(repo, file)...)
	}

	sort.Slice(topics, func(i, j int) bool { return topics[i].Topic < topics[j].Topic })

	out := filepath.Join(repo, "docs", "bus-topics.md")
	if err := writeMarkdown(out, topics); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %s (%d topics)\n", out, len(topics))
}

func messageFiles(repo string) ([]string, error) {
	var files []string
	sdkDir := filepath.Join(repo, "sdk")
	if err := filepath.WalkDir(sdkDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_messages.go") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	modulesDir := filepath.Join(repo, "modules")
	if err := filepath.WalkDir(modulesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_messages.go") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// parseFile walks a _messages.go file and extracts BusTopic →
// (message type, response type). The response is inferred by
// matching on a `<Msg>Resp` struct in the same file; if none is
// found the column renders as "(no reply)".
func parseFile(repo, path string) []entry {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skip %s: %v\n", filepath.Base(path), err)
		return nil
	}

	// Collect all struct names declared in this file; used to
	// decide whether `<Msg>Resp` exists.
	structs := make(map[string]bool)
	constStrings := make(map[string]string)
	ast.Inspect(f, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if _, isStruct := ts.Type.(*ast.StructType); isStruct {
				structs[ts.Name.Name] = true
			}
		}
		if decl, ok := n.(*ast.GenDecl); ok && decl.Tok == token.CONST {
			for _, spec := range decl.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						if unquoted, err := strconv.Unquote(lit.Value); err == nil {
							constStrings[name.Name] = unquoted
						}
					}
				}
			}
		}
		return true
	})

	source, err := filepath.Rel(repo, path)
	if err != nil {
		source = filepath.Base(path)
	}
	source = filepath.ToSlash(source)

	var out []entry
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "BusTopic" || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return true
		}
		recv, ok := fn.Recv.List[0].Type.(*ast.Ident)
		if !ok {
			return true
		}
		topic := extractStringReturn(fn, constStrings)
		if topic == "" {
			return true
		}

		resp := strings.TrimSuffix(recv.Name, "Msg") + "Resp"
		if !structs[resp] {
			resp = "(no reply)"
		}
		out = append(out, entry{
			Msg:   recv.Name,
			Topic: topic,
			Resp:  resp,
			File:  source,
		})
		return true
	})
	return out
}

func extractStringReturn(fn *ast.FuncDecl, constStrings map[string]string) string {
	if fn.Body == nil || len(fn.Body.List) == 0 {
		return ""
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return ""
	}
	switch v := ret.Results[0].(type) {
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return ""
		}
		unquoted, err := strconv.Unquote(v.Value)
		if err != nil {
			return ""
		}
		return unquoted
	case *ast.Ident:
		return constStrings[v.Name]
	default:
		return ""
	}
}

func writeMarkdown(path string, topics []entry) error {
	var b strings.Builder
	b.WriteString("# Bus topic catalog\n\n")
	b.WriteString("Generated from `sdk/**/*_messages.go` and `modules/**/*_messages.go` via ")
	b.WriteString("`go run scripts/gen-bus-topics.go`. Do not edit by hand.\n\n")
	b.WriteString("| Topic | Request | Response | Source |\n")
	b.WriteString("|-------|---------|----------|--------|\n")
	for _, e := range topics {
		fmt.Fprintf(&b, "| `%s` | `%s` | `%s` | `%s` |\n",
			e.Topic, e.Msg, e.Resp, e.File)
	}
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
