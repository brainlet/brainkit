package workflow

import (
	"fmt"
	"strings"
)

// GenerateASDeclarations generates AssemblyScript type declaration files
// from registered host functions. These .d.ts files give workflow authors
// (and LLMs) typed APIs for all available host functions.
func GenerateASDeclarations(registry *HostFunctionRegistry) map[string]string {
	files := make(map[string]string)

	// Built-in brainkit declarations (always present)
	files["brainkit.d.ts"] = generateBuiltinDeclarations()

	// Plugin-provided declarations (one file per module)
	for _, module := range registry.ListModules() {
		if module == "brainkit" || module == "ai" || module == "env" || module == "host" {
			continue // skip built-in modules
		}
		funcs := registry.ListFunctions(module)
		if len(funcs) == 0 {
			continue
		}
		files[module+".d.ts"] = generateModuleDeclarations(module, funcs)
	}

	// AI module (built-in)
	files["ai.d.ts"] = generateAIDeclarations()

	return files
}

func generateBuiltinDeclarations() string {
	var b strings.Builder
	b.WriteString("// Generated: brainkit built-in host functions\n")
	b.WriteString("// Do not edit — regenerated when plugins change\n\n")

	decls := []struct{ name, sig string }{
		{"step", "(name: string): void"},
		{"sleep", "(seconds: i64): void"},
		{"waitForEvent", "(topic: string, timeoutSeconds: i64): string"},
		{"complete", "(result: string): void"},
		{"fail", "(error: string): void"},
		{"log", "(message: string, level: i32): void"},
		{"get_state", "(key: string): string"},
		{"set_state", "(key: string, value: string): void"},
	}

	b.WriteString("declare module \"brainkit\" {\n")
	for _, d := range decls {
		fmt.Fprintf(&b, "  export function %s%s;\n", d.name, d.sig)
	}
	b.WriteString("}\n")
	return b.String()
}

func generateAIDeclarations() string {
	var b strings.Builder
	b.WriteString("// Generated: AI host functions (built-in)\n\n")
	b.WriteString("declare module \"ai\" {\n")
	b.WriteString("  export function generate(prompt: string): string;\n")
	b.WriteString("}\n")
	return b.String()
}

func generateModuleDeclarations(module string, funcs []*HostFunctionDef) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// Generated: %s host functions (from plugin: %s)\n\n",
		module, funcs[0].PluginName)

	fmt.Fprintf(&b, "declare module \"%s\" {\n", module)
	for _, f := range funcs {
		params := make([]string, len(f.Params))
		for i, p := range f.Params {
			params[i] = fmt.Sprintf("%s: %s", p.Name, asType(p.Type))
		}
		ret := asType(f.Returns)
		if f.Description != "" {
			fmt.Fprintf(&b, "  /** %s */\n", f.Description)
		}
		fmt.Fprintf(&b, "  export function %s(%s): %s;\n", f.Name, strings.Join(params, ", "), ret)
	}
	b.WriteString("}\n")
	return b.String()
}

// asType maps host function parameter types to AssemblyScript types.
func asType(t string) string {
	switch t {
	case "string":
		return "string"
	case "i32":
		return "i32"
	case "i64":
		return "i64"
	case "f64":
		return "f64"
	case "void", "":
		return "void"
	default:
		return "string" // default to string for unknown types
	}
}
