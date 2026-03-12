package jsbridge

import "github.com/fastschema/qjs"

// Polyfill registers bridge functions and JS wrappers into a QuickJS context.
type Polyfill interface {
	Name() string
	Setup(ctx *qjs.Context) error
}

// evalJS evaluates JavaScript code in the context, freeing the result.
func evalJS(ctx *qjs.Context, code string) error {
	val, err := ctx.Eval("polyfill.js", qjs.Code(code))
	if err != nil {
		return err
	}
	val.Free()
	return nil
}
