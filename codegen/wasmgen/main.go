// wasmgen reads sdk/messages/*.go and generates AssemblyScript typed wrappers
// for the WASM API surface. Each command pair (FooMsg + FooResp) produces a
// request class, response class, and namespace function.
//
// Usage: go run ./codegen/wasmgen -messages ./sdk/messages -out ./kit/runtime/wasm/generated
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MessageType represents a parsed BrainkitMessage type.
type MessageType struct {
	Name   string            // Go type name, e.g. "AiGenerateMsg"
	Topic  string            // BusTopic() return value, e.g. "ai.generate"
	Fields []MessageField    // struct fields
	IsResp bool              // true if this is a response type (has ResultMeta)
}

// MessageField represents a field in a message struct.
type MessageField struct {
	Name     string // Go field name
	JSONName string // json tag name
	GoType   string // Go type as string
}

// DomainGroup holds request/response pairs for a domain.
type DomainGroup struct {
	Domain   string        // e.g. "ai", "tools", "agents"
	Commands []CommandPair // request → response pairs
	Events   []MessageType // fire-and-forget events
}

// CommandPair links a request to its response.
type CommandPair struct {
	Request  MessageType
	Response MessageType
}

func main() {
	messagesDir := flag.String("messages", "./sdk/messages", "path to sdk/messages directory")
	outDir := flag.String("out", "./kit/runtime/wasm/generated", "output directory for generated .ts files")
	flag.Parse()

	types, err := parseMessages(*messagesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	groups := groupByDomain(types)

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir error: %v\n", err)
		os.Exit(1)
	}

	for domain, group := range groups {
		content := generateDomainFile(domain, group)
		path := filepath.Join(*outDir, domain+".ts")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("generated %s (%d commands, %d events)\n", path, len(group.Commands), len(group.Events))
	}

	fmt.Printf("done: %d domains\n", len(groups))
}

// parseMessages reads all .go files in the messages directory and extracts BrainkitMessage types.
func parseMessages(dir string) ([]MessageType, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse dir: %w", err)
	}

	var types []MessageType

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			// First pass: find all types with BusTopic() methods
			topicMap := make(map[string]string) // type name → topic
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || fn.Name.Name != "BusTopic" {
					continue
				}
				// Get receiver type name
				recvType := receiverTypeName(fn.Recv)
				if recvType == "" {
					continue
				}
				// Get return value (string literal)
				topic := extractStringReturn(fn)
				if topic != "" {
					topicMap[recvType] = topic
				}
			}

			// Second pass: extract struct fields for types that have BusTopic
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.TYPE {
					continue
				}
				for _, spec := range genDecl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					topic, hasTopic := topicMap[typeSpec.Name.Name]
					if !hasTopic {
						continue
					}

					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}

					mt := MessageType{
						Name:  typeSpec.Name.Name,
						Topic: topic,
					}

					hasResultMeta := false
					for _, field := range structType.Fields.List {
						// Check for embedded ResultMeta
						if len(field.Names) == 0 {
							if ident, ok := field.Type.(*ast.Ident); ok && ident.Name == "ResultMeta" {
								hasResultMeta = true
							}
							continue
						}

						for _, name := range field.Names {
							if !name.IsExported() {
								continue
							}
							jsonName := strings.ToLower(name.Name[:1]) + name.Name[1:]
							if field.Tag != nil {
								jsonName = extractJSONTag(field.Tag.Value)
								if jsonName == "" || jsonName == "-" {
									jsonName = strings.ToLower(name.Name[:1]) + name.Name[1:]
								}
							}
							mt.Fields = append(mt.Fields, MessageField{
								Name:     name.Name,
								JSONName: jsonName,
								GoType:   typeString(field.Type),
							})
						}
					}

					mt.IsResp = hasResultMeta
					types = append(types, mt)
				}
			}
		}
	}

	return types, nil
}

// groupByDomain groups message types by their topic prefix (e.g., "ai", "tools").
func groupByDomain(types []MessageType) map[string]DomainGroup {
	groups := make(map[string]DomainGroup)

	// Index by topic for pairing
	byTopic := make(map[string]MessageType)
	for _, mt := range types {
		byTopic[mt.Topic] = mt
	}

	// Pair commands: "ai.generate" + "ai.generate.result"
	processed := make(map[string]bool)
	for _, mt := range types {
		if processed[mt.Name] || mt.IsResp {
			continue
		}

		domain := topicDomain(mt.Topic)
		if domain == "" {
			continue
		}

		resultTopic := mt.Topic + ".result"
		resp, hasResp := byTopic[resultTopic]

		group := groups[domain]
		group.Domain = domain

		if hasResp {
			group.Commands = append(group.Commands, CommandPair{
				Request:  mt,
				Response: resp,
			})
			processed[mt.Name] = true
			processed[resp.Name] = true
		} else if !strings.HasSuffix(mt.Topic, ".result") {
			// No response type — treat as event
			group.Events = append(group.Events, mt)
			processed[mt.Name] = true
		}

		groups[domain] = group
	}

	return groups
}

// generateDomainFile produces AssemblyScript code for a domain.
func generateDomainFile(domain string, group DomainGroup) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("// AUTO-GENERATED from sdk/messages — do not edit.\n"))
	b.WriteString(fmt.Sprintf("// Domain: %s\n\n", domain))

	// Sort commands by topic for deterministic output
	sort.Slice(group.Commands, func(i, j int) bool {
		return group.Commands[i].Request.Topic < group.Commands[j].Request.Topic
	})

	// Generate request classes
	for _, cmd := range group.Commands {
		b.WriteString(generateRequestClass(cmd.Request))
		b.WriteString("\n")
	}

	// Generate response classes
	for _, cmd := range group.Commands {
		b.WriteString(generateResponseClass(cmd.Response))
		b.WriteString("\n")
	}

	// Generate namespace with functions
	nsName := domain
	// Avoid AS reserved words
	if nsName == "wasm" {
		nsName = "wasm_ops"
	}
	if nsName == "fs" {
		nsName = "fs_ops"
	}

	b.WriteString(fmt.Sprintf("export namespace %s {\n", nsName))
	for _, cmd := range group.Commands {
		funcName := commandFuncName(cmd.Request.Topic, domain)
		reqClass := cmd.Request.Name
		b.WriteString(fmt.Sprintf("    export function %s(msg: %s, callback: string): void {\n", funcName, reqClass))
		b.WriteString(fmt.Sprintf("        _invokeAsync(\"%s\", msg.toJSON(), callback)\n", cmd.Request.Topic))
		b.WriteString("    }\n\n")
	}
	b.WriteString("}\n")

	// Generate event functions if any
	if len(group.Events) > 0 {
		b.WriteString("\n// Events\n")
		for _, evt := range group.Events {
			b.WriteString(generateRequestClass(evt))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func generateRequestClass(mt MessageType) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("export class %s {\n", mt.Name))

	// Fields
	for _, f := range mt.Fields {
		asType := goTypeToAS(f.GoType)
		b.WriteString(fmt.Sprintf("    %s: %s\n", f.JSONName, asType))
	}

	// Constructor
	if len(mt.Fields) > 0 {
		b.WriteString("\n    constructor(")
		for i, f := range mt.Fields {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(fmt.Sprintf("%s: %s", f.JSONName, goTypeToAS(f.GoType)))
		}
		b.WriteString(") {\n")
		for _, f := range mt.Fields {
			b.WriteString(fmt.Sprintf("        this.%s = %s\n", f.JSONName, f.JSONName))
		}
		b.WriteString("    }\n")
	}

	// toJSON
	b.WriteString("\n    toJSON(): string {\n")
	b.WriteString("        let obj = new JSONObject()\n")
	for _, f := range mt.Fields {
		setter := jsonSetter(f)
		b.WriteString(fmt.Sprintf("        %s\n", setter))
	}
	b.WriteString("        return obj.toString()\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return b.String()
}

func generateResponseClass(mt MessageType) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("export class %s {\n", mt.Name))

	// Fields (excluding error which comes from ResultMeta)
	for _, f := range mt.Fields {
		if f.JSONName == "error" {
			continue
		}
		asType := goTypeToAS(f.GoType)
		b.WriteString(fmt.Sprintf("    %s: %s\n", f.JSONName, asType))
	}
	b.WriteString("    error: string\n")

	// Default constructor
	b.WriteString("\n    constructor() {\n")
	for _, f := range mt.Fields {
		if f.JSONName == "error" {
			continue
		}
		b.WriteString(fmt.Sprintf("        this.%s = %s\n", f.JSONName, goTypeDefaultAS(f.GoType)))
	}
	b.WriteString("        this.error = \"\"\n")
	b.WriteString("    }\n")

	// Static parse
	b.WriteString(fmt.Sprintf("\n    static parse(json: string): %s {\n", mt.Name))
	b.WriteString(fmt.Sprintf("        let resp = new %s()\n", mt.Name))
	b.WriteString("        let val = JSONValue.parse(json)\n")
	b.WriteString("        if (val.isObject()) {\n")
	b.WriteString("            let obj = val.asObject()\n")
	for _, f := range mt.Fields {
		if f.JSONName == "error" {
			continue
		}
		parser := jsonParser(f)
		b.WriteString(fmt.Sprintf("            %s\n", parser))
	}
	b.WriteString("            if (obj.has(\"error\")) resp.error = obj.getString(\"error\")\n")
	b.WriteString("        }\n")
	b.WriteString("        return resp\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")

	return b.String()
}

// --- Helpers ---

func receiverTypeName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch t := recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

func extractStringReturn(fn *ast.FuncDecl) string {
	if fn.Body == nil || len(fn.Body.List) == 0 {
		return ""
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) == 0 {
		return ""
	}
	lit, ok := ret.Results[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	return strings.Trim(lit.Value, "\"")
}

func extractJSONTag(tag string) string {
	tag = strings.Trim(tag, "`")
	for _, part := range strings.Split(tag, " ") {
		if strings.HasPrefix(part, "json:\"") {
			val := strings.TrimPrefix(part, "json:\"")
			val = strings.TrimSuffix(val, "\"")
			if idx := strings.Index(val, ","); idx >= 0 {
				val = val[:idx]
			}
			return val
		}
	}
	return ""
}

func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	case *ast.InterfaceType:
		return "any"
	default:
		return "any"
	}
}

func topicDomain(topic string) string {
	parts := strings.SplitN(topic, ".", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func commandFuncName(topic, domain string) string {
	// "ai.generate" with domain "ai" → "generate"
	// "ai.generateObject" → "generateObject"
	suffix := strings.TrimPrefix(topic, domain+".")
	// Convert dots to camelCase: "createThread" stays, "foo.bar" becomes "fooBar"
	parts := strings.Split(suffix, ".")
	if len(parts) == 1 {
		return parts[0]
	}
	result := parts[0]
	for _, p := range parts[1:] {
		if len(p) > 0 {
			result += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return result
}

func goTypeToAS(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64":
		return "i32"
	case "float64", "float32":
		return "f64"
	case "bool":
		return "bool"
	case "any":
		return "string" // serialized as JSON string in AS
	case "json.RawMessage":
		return "string" // raw JSON as string
	}
	if strings.HasPrefix(goType, "[]") {
		return "string" // arrays serialized as JSON string
	}
	if strings.HasPrefix(goType, "map[") {
		return "string" // maps serialized as JSON string
	}
	if strings.HasPrefix(goType, "*") {
		return goTypeToAS(goType[1:])
	}
	return "string" // default: serialize as JSON string
}

func goTypeDefaultAS(goType string) string {
	switch goType {
	case "string":
		return "\"\""
	case "int", "int32", "int64":
		return "0"
	case "float64", "float32":
		return "0.0"
	case "bool":
		return "false"
	case "any", "json.RawMessage":
		return "\"\""
	}
	if strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map[") {
		return "\"\""
	}
	if strings.HasPrefix(goType, "*") {
		return goTypeDefaultAS(goType[1:])
	}
	return "\"\""
}

func jsonSetter(f MessageField) string {
	switch f.GoType {
	case "string":
		return fmt.Sprintf("obj.setString(\"%s\", this.%s)", f.JSONName, f.JSONName)
	case "int", "int32", "int64":
		return fmt.Sprintf("obj.setInt(\"%s\", this.%s)", f.JSONName, f.JSONName)
	case "float64", "float32":
		return fmt.Sprintf("obj.setNumber(\"%s\", this.%s)", f.JSONName, f.JSONName)
	case "bool":
		return fmt.Sprintf("obj.setBool(\"%s\", this.%s)", f.JSONName, f.JSONName)
	default:
		// Complex types: parse as JSON and set
		return fmt.Sprintf("if (this.%s.length > 0) obj.set(\"%s\", JSONValue.parse(this.%s))", f.JSONName, f.JSONName, f.JSONName)
	}
}

func jsonParser(f MessageField) string {
	switch f.GoType {
	case "string":
		return fmt.Sprintf("if (obj.has(\"%s\")) resp.%s = obj.getString(\"%s\")", f.JSONName, f.JSONName, f.JSONName)
	case "int", "int32", "int64":
		return fmt.Sprintf("if (obj.has(\"%s\")) resp.%s = obj.getInt(\"%s\")", f.JSONName, f.JSONName, f.JSONName)
	case "float64", "float32":
		return fmt.Sprintf("if (obj.has(\"%s\")) resp.%s = obj.getNumber(\"%s\")", f.JSONName, f.JSONName, f.JSONName)
	case "bool":
		return fmt.Sprintf("if (obj.has(\"%s\")) resp.%s = obj.getBool(\"%s\")", f.JSONName, f.JSONName, f.JSONName)
	default:
		// Complex types: store as JSON string
		return fmt.Sprintf("if (obj.has(\"%s\")) resp.%s = obj.get(\"%s\").toString()", f.JSONName, f.JSONName, f.JSONName)
	}
}
