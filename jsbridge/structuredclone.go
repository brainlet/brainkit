package jsbridge

import "github.com/fastschema/qjs"

// StructuredClonePolyfill provides globalThis.structuredClone.
type StructuredClonePolyfill struct{}

// StructuredClone creates a structuredClone polyfill.
func StructuredClone() *StructuredClonePolyfill { return &StructuredClonePolyfill{} }

func (p *StructuredClonePolyfill) Name() string { return "structuredClone" }

func (p *StructuredClonePolyfill) Setup(ctx *qjs.Context) error {
	return evalJS(ctx, `
if (typeof globalThis.structuredClone === 'undefined') {
  globalThis.structuredClone = function(value) {
    return JSON.parse(JSON.stringify(value));
  };
}
`)
}
