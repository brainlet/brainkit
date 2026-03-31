// Package typescript provides Kit-level TypeScript transpilation.
//
// Wraps vendor_typescript.Transpile() with brainkit-specific defaults.
// Used by brainkit.Deploy when source is .ts and by the fixture test runner.
package typescript

import (
	ts "github.com/brainlet/brainkit/vendor_typescript"
)

// TranspileTS converts TypeScript source to JavaScript.
// Uses ESNext target + ESNext modules — no downleveling.
// Strips: type annotations, interfaces, type aliases, generics.
// Preserves: imports, exports, async/await, all runtime code.
func TranspileTS(source string, fileName ...string) (string, error) {
	name := "input.ts"
	if len(fileName) > 0 && fileName[0] != "" {
		name = fileName[0]
	}
	return ts.Transpile(source, ts.TranspileOptions{
		FileName: name,
	})
}
