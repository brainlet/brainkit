//go:build experiment

package lifecycle

import (
	"testing"

	quickjs "github.com/buke/quickjs-go"
)

// Reproduces the exact pattern from IsolatedTeardown
func TestBug2_ExactIsolatedTeardownPattern(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()

	reg := NewGoRegistry()

	ctxA := rt.NewContext()
	registerBridges(ctxA, reg, "a.ts")
	eval(ctxA, `agent({ name: "a1" }); agent({ name: "a2" });`)

	ctxB := rt.NewContext()
	registerBridges(ctxB, reg, "b.ts")
	eval(ctxB, `agent({ name: "b1" }); createTool({ id: "b-tool" });`)

	ctxC := rt.NewContext()
	registerBridges(ctxC, reg, "c.ts")
	eval(ctxC, `agent({ name: "c1" }); subscribe("events.*");`)

	if reg.AgentCount() != 4 {
		t.Fatalf("expected 4 agents, got %d", reg.AgentCount())
	}

	// Close B
	ctxB.Close()
	reg.TeardownSource("b.ts")

	// Use A (this is where it crashes in the full test)
	val := evalStr(ctxA, `"a-alive"`)
	if val != "a-alive" {
		t.Fatal("A should work")
	}

	// Close C and A
	ctxC.Close()
	reg.TeardownSource("c.ts")
	ctxA.Close()
	reg.TeardownSource("a.ts")

	t.Log("PASS: exact isolated teardown pattern")
}

// Simpler: just 2 contexts with bridges, close one, eval on other
func TestBug2_TwoBridgesCloseOneEvalOther(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	reg := NewGoRegistry()

	ctxA := rt.NewContext()
	registerBridges(ctxA, reg, "a.ts")
	eval(ctxA, `agent({ name: "a1" });`)

	ctxB := rt.NewContext()
	registerBridges(ctxB, reg, "b.ts")
	eval(ctxB, `agent({ name: "b1" });`)

	ctxB.Close()

	val := evalStr(ctxA, `"still-here"`)
	if val != "still-here" {
		t.Fatal("A should work")
	}

	ctxA.Close()
	t.Log("PASS: two bridges, close one, eval other")
}

// Does it crash with just eval (no bridge call) after close?
func TestBug2_PlainEvalAfterBridgeClose(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	reg := NewGoRegistry()

	ctxA := rt.NewContext()
	registerBridges(ctxA, reg, "a.ts")

	ctxB := rt.NewContext()
	registerBridges(ctxB, reg, "b.ts")

	ctxB.Close()

	// Plain eval, no bridge function call
	val := evalStr(ctxA, `1 + 1 + ""`)
	if val != "2" {
		t.Fatal("plain eval should work")
	}

	ctxA.Close()
	t.Log("PASS: plain eval after bridge context close")
}

// Does calling a bridge function after other context close crash?
func TestBug2_BridgeCallAfterOtherClose(t *testing.T) {
	rt := quickjs.NewRuntime()
	defer rt.Close()
	reg := NewGoRegistry()

	ctxA := rt.NewContext()
	registerBridges(ctxA, reg, "a.ts")

	ctxB := rt.NewContext()
	registerBridges(ctxB, reg, "b.ts")

	ctxB.Close()

	// Call bridge function on A after B is closed
	eval(ctxA, `agent({ name: "after-close" });`)

	if reg.AgentCount() != 1 {
		t.Fatal("expected 1 agent")
	}

	ctxA.Close()
	t.Log("PASS: bridge call on A after B close")
}
