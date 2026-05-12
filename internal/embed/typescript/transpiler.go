// Package typescript provides Kit-level TypeScript transpilation.
//
// Wraps vendor_typescript.Transpile with Brainkit-specific defaults. The JS
// runtime uses this for direct .ts source deployments before evaluation.
package typescript

import (
	ts "github.com/brainlet/brainkit/vendor_typescript"
)

// TranspileTS converts TypeScript source to JavaScript.
//
// Uses ESNext target + ESNext modules with no downleveling. Type annotations,
// interfaces, type aliases, and generics are erased; imports, exports,
// async/await, and runtime code are preserved for the caller to handle.
func TranspileTS(source string, fileName ...string) (string, error) {
	name := "input.ts"
	if len(fileName) > 0 && fileName[0] != "" {
		name = fileName[0]
	}
	return ts.Transpile(source, ts.TranspileOptions{
		FileName: name,
	})
}
