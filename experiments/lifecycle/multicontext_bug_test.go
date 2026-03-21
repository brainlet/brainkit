//go:build experiment

// Minimal reproduction of the multi-context close crash.
// Narrowing down exactly what causes the JS_FreeRuntime assertion failure.
package lifecycle

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"
)

// Test: can we create and close two plain contexts without crash?
func TestBug_TwoPlainContexts(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctxA := rt.NewContext()
	eval(ctxA, `var x = 1;`)
	ctxA.Close()

	ctxB := rt.NewContext()
	eval(ctxB, `var y = 2;`)
	ctxB.Close()

	t.Log("PASS: two plain contexts, sequential create/close")
}

// Test: two contexts open simultaneously, close in order
func TestBug_TwoContextsSimultaneous(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctxA := rt.NewContext()
	ctxB := rt.NewContext()

	eval(ctxA, `var a = 1;`)
	eval(ctxB, `var b = 2;`)

	ctxB.Close()
	ctxA.Close()

	t.Log("PASS: two simultaneous contexts, closed in reverse order")
}

// Test: two contexts, close one, use the other, then close
func TestBug_CloseOneContinueOther(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctxA := rt.NewContext()
	ctxB := rt.NewContext()

	eval(ctxA, `var a = 1;`)
	eval(ctxB, `var b = 2;`)

	ctxB.Close()

	// Use ctxA after ctxB is closed
	val := evalStr(ctxA, `"still alive"`)
	if val != "still alive" {
		t.Fatal("ctxA should work after ctxB close")
	}

	ctxA.Close()

	t.Log("PASS: close one context, use the other, close it")
}

// Test: does registering a Go function cause the crash?
func TestBug_GoFunctionRegistration(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctxA := rt.NewContext()
	ctxB := rt.NewContext()

	// Register Go functions on both
	fnA := ctxA.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.String("from-a")
	})
	ctxA.Globals().Set("goFn", fnA)

	fnB := ctxB.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		return ctx.String("from-b")
	})
	ctxB.Globals().Set("goFn", fnB)

	// Use them
	valA := evalStr(ctxA, `goFn()`)
	valB := evalStr(ctxB, `goFn()`)
	if valA != "from-a" || valB != "from-b" {
		t.Fatalf("unexpected: a=%s b=%s", valA, valB)
	}

	// Close B, use A
	ctxB.Close()

	valA = evalStr(ctxA, `goFn()`)
	if valA != "from-a" {
		t.Fatalf("A should still work: %s", valA)
	}

	ctxA.Close()

	t.Log("PASS: Go functions registered on separate contexts, close one, other works")
}

// Test: does the number of registered functions matter?
func TestBug_ManyGoFunctions(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctxA := rt.NewContext()
	ctxB := rt.NewContext()

	// Register 10 functions on each
	for i := range 10 {
		name := "fn" + string(rune('0'+i))
		fnA := ctxA.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			return ctx.String("a")
		})
		ctxA.Globals().Set(name, fnA)

		fnB := ctxB.Function(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			return ctx.String("b")
		})
		ctxB.Globals().Set(name, fnB)
	}

	ctxB.Close()
	rt.RunGC()

	val := evalStr(ctxA, `fn0()`)
	if val != "a" {
		t.Fatal("A should still work")
	}

	ctxA.Close()

	t.Log("PASS: many Go functions, close one context, other works")
}

// Test: does eval AFTER close of another context crash?
func TestBug_EvalAfterOtherClose(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	ctxA := rt.NewContext()
	ctxB := rt.NewContext()

	// Complex eval on both
	eval(ctxA, `
		var data = { items: [] };
		for (var i = 0; i < 100; i++) data.items.push("item-" + i);
	`)
	eval(ctxB, `
		var data = { items: [] };
		for (var i = 0; i < 100; i++) data.items.push("item-" + i);
	`)

	ctxB.Close()
	rt.RunGC()

	// Eval on A after B is closed
	val := evalInt(ctxA, `data.items.length`)
	if val != 100 {
		t.Fatalf("expected 100, got %d", val)
	}

	ctxA.Close()

	t.Log("PASS: complex eval, close B, eval on A, close A")
}
