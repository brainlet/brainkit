package jsbridge

import quickjs "github.com/buke/quickjs-go"

// StructuredClonePolyfill provides globalThis.structuredClone.
type StructuredClonePolyfill struct{}

// StructuredClone creates a structuredClone polyfill.
func StructuredClone() *StructuredClonePolyfill { return &StructuredClonePolyfill{} }

func (p *StructuredClonePolyfill) Name() string { return "structuredClone" }

func (p *StructuredClonePolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, `
if (typeof globalThis.structuredClone === 'undefined') {
  globalThis.structuredClone = function(value) {
    return JSON.parse(JSON.stringify(value));
  };
}
`)
}
