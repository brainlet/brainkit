package aiembed

import (
	_ "embed"

	"fmt"

	"github.com/brainlet/brainkit/jsbridge"
	"github.com/fastschema/qjs"
)

//go:embed ai_sdk_bundle.js
var bundleSource string

// stubs provides global stubs for Web APIs that QuickJS lacks but the AI SDK references.
const stubs = `
if (typeof globalThis.structuredClone === 'undefined') {
  globalThis.structuredClone = function(obj) { return JSON.parse(JSON.stringify(obj)); };
}
`

// LoadBundle evaluates the AI SDK bundle into a jsbridge.Bridge.
// After loading, globalThis.__ai_sdk is available with generateText and createOpenAI.
func LoadBundle(b *jsbridge.Bridge) error {
	// Install stubs for missing Web APIs before the bundle
	stub, err := b.Eval("stubs.js", qjs.Code(stubs))
	if err != nil {
		return fmt.Errorf("ai-embed: load stubs: %w", err)
	}
	stub.Free()

	val, err := b.Eval("ai-sdk-bundle.js", qjs.Code(bundleSource))
	if err != nil {
		return fmt.Errorf("ai-embed: load bundle: %w", err)
	}
	val.Free()
	return nil
}
