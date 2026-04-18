//go:build ignore

// Command gen-bus-topics scans sdk/*_messages.go for BusTopic()
// string constants and emits a Markdown table under
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
	sdkDir := filepath.Join(repo, "sdk")
	files, err := filepath.Glob(filepath.Join(sdkDir, "*_messages.go"))
	if err != nil {
		fail(err)
	}

	var topics []entry
	for _, file := range files {
		topics = append(topics, parseFile(file)...)
	}

	sort.Slice(topics, func(i, j int) bool { return topics[i].Topic < topics[j].Topic })

	out := filepath.Join(repo, "docs", "bus-topics.md")
	if err := writeMarkdown(out, topics); err != nil {
		fail(err)
	}
	fmt.Printf("wrote %s (%d topics)\n", out, len(topics))
}

// parseFile walks a _messages.go file and extracts BusTopic →
// (message type, response type). The response is inferred by
// matching on a `<Msg>Resp` struct in the same file; if none is
// found the column renders as "(no reply)".
func parseFile(path string) []entry {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skip %s: %v\n", filepath.Base(path), err)
		return nil
	}

	// Collect all struct names declared in this file; used to
	// decide whether `<Msg>Resp` exists.
	structs := make(map[string]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		if ts, ok := n.(*ast.TypeSpec); ok {
			if _, isStruct := ts.Type.(*ast.StructType); isStruct {
				structs[ts.Name.Name] = true
			}
		}
		return true
	})

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
		topic := extractStringReturn(fn)
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
			File:  filepath.Base(path),
		})
		return true
	})
	return out
}

func extractStringReturn(fn *ast.FuncDecl) string {
	if fn.Body == nil || len(fn.Body.List) == 0 {
		return ""
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return ""
	}
	lit, ok := ret.Results[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	unquoted, err := strconv.Unquote(lit.Value)
	if err != nil {
		return ""
	}
	return unquoted
}

func writeMarkdown(path string, topics []entry) error {
	var b strings.Builder
	b.WriteString("# Bus topic catalog\n\n")
	b.WriteString("Generated from `sdk/*_messages.go` via ")
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
