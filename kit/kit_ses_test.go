package kit

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSES_AvailableInKit(t *testing.T) {
	kit := newTestKitNoKey(t)
	defer kit.Close()

	result, err := kit.EvalTS(context.Background(), "__ses_check.ts", `
		return JSON.stringify({
			hasCompartment: typeof globalThis.Compartment === "function",
			hasHarden: typeof globalThis.harden === "function",
		});
	`)
	if err != nil {
		t.Fatal(err)
	}

	var check struct {
		HasCompartment bool `json:"hasCompartment"`
		HasHarden      bool `json:"hasHarden"`
	}
	json.Unmarshal([]byte(result), &check)

	if !check.HasCompartment {
		t.Fatal("Compartment not available after lockdown")
	}
	if !check.HasHarden {
		t.Fatal("harden not available after lockdown")
	}
	t.Log("PASS: SES Compartment and harden available in Kit")
}

func TestSES_EndowmentsFactory(t *testing.T) {
	kit := newTestKitNoKey(t)
	defer kit.Close()

	result, err := kit.EvalTS(context.Background(), "__endow_check.ts", `
		var e = globalThis.__kitEndowments("test-source");
		return JSON.stringify({
			hasAgent: typeof e.agent === "function",
			hasCreateTool: typeof e.createTool === "function",
			hasAI: typeof e.ai === "object",
			hasTools: typeof e.tools === "object",
			hasBus: typeof e.bus === "object",
			hasZ: typeof e.z === "object",
			hasConsole: typeof e.console === "object",
			hasJSON: typeof e.JSON === "object",
			hasMcp: typeof e.mcp === "object",
		});
	`)
	if err != nil {
		t.Fatal(err)
	}

	var check map[string]bool
	json.Unmarshal([]byte(result), &check)

	for key, val := range check {
		if !val {
			t.Errorf("%s missing from endowments", key)
		}
	}
	t.Logf("endowments: %s", result)
}

func TestSES_EndowmentsHardened(t *testing.T) {
	kit := newTestKitNoKey(t)
	defer kit.Close()

	result, err := kit.EvalTS(context.Background(), "__endow_harden.ts", `
		var e = globalThis.__kitEndowments("test-source");
		// In non-strict mode, assignment to frozen object silently fails (no throw)
		e.newProp = "attempt";
		var mutated = e.newProp === "attempt";
		// Also try overwriting an existing property
		var origAgent = e.agent;
		e.agent = null;
		var agentChanged = e.agent !== origAgent;
		return JSON.stringify({ newPropAdded: mutated, agentOverwritten: agentChanged });
	`)
	if err != nil {
		t.Fatal(err)
	}

	var check struct {
		NewPropAdded    bool `json:"newPropAdded"`
		AgentOverwritten bool `json:"agentOverwritten"`
	}
	json.Unmarshal([]byte(result), &check)

	if check.NewPropAdded {
		t.Fatal("new property should not be added to hardened object")
	}
	if check.AgentOverwritten {
		t.Fatal("existing property should not be overwritten on hardened object")
	}
	t.Logf("harden: %s", result)
}

func TestSES_CompartmentWithEndowments(t *testing.T) {
	kit := newTestKitNoKey(t)
	defer kit.Close()

	result, err := kit.EvalTS(context.Background(), "__compartment_endow.ts", `
		var endowments = globalThis.__kitEndowments("test-deploy");
		var c = new globalThis.Compartment({ __options__: true, globals: endowments });
		var val = c.evaluate('typeof agent === "function" && typeof ai === "object" && typeof z === "object"');
		return JSON.stringify({ allAvailable: val });
	`)
	if err != nil {
		t.Fatal(err)
	}

	var check struct{ AllAvailable bool `json:"allAvailable"` }
	json.Unmarshal([]byte(result), &check)
	if !check.AllAvailable {
		t.Fatalf("endowed APIs not available in compartment: %s", result)
	}
	t.Log("PASS: all Kit APIs available inside Compartment via endowments")
}

func TestSES_CompartmentWorksInKit(t *testing.T) {
	kit := newTestKitNoKey(t)
	defer kit.Close()

	result, err := kit.EvalTS(context.Background(), "__ses_compartment.ts", `
		var c = new globalThis.Compartment({ __options__: true, globals: { x: 42 } });
		var val = c.evaluate("x + 1");
		return JSON.stringify({ result: val });
	`)
	if err != nil {
		t.Fatal(err)
	}

	var check struct{ Result int `json:"result"` }
	json.Unmarshal([]byte(result), &check)

	if check.Result != 43 {
		t.Fatalf("expected 43, got %d (raw: %s)", check.Result, result)
	}
	t.Log("PASS: SES Compartment evaluates correctly inside Kit")
}
