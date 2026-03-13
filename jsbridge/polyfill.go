package jsbridge

import quickjs "github.com/buke/quickjs-go"

// Polyfill registers bridge functions and JS wrappers into a QuickJS context.
type Polyfill interface {
	Name() string
	Setup(ctx *quickjs.Context) error
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
