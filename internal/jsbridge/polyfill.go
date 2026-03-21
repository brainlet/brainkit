package jsbridge

import quickjs "github.com/buke/quickjs-go"

// Polyfill registers bridge functions and JS wrappers into a QuickJS context.
type Polyfill interface {
	Name() string
	Setup(ctx *quickjs.Context) error
}

// BridgeAware is optionally implemented by polyfills that need access to the
// Bridge for tracked goroutines (Bridge.Go) and context (Bridge.GoContext).
type BridgeAware interface {
	SetBridge(b *Bridge)
}

// evalJS evaluates JavaScript code in the context, freeing the result.
func evalJS(ctx *quickjs.Context, code string) error {
	val := ctx.Eval(code)
	if val.IsException() {
		return ctx.Exception()
	}
	val.Free()
	return nil
}
