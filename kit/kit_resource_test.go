package kit

import (
	"context"
	"testing"
)

func TestResourceRegistry_AgentRegistration(t *testing.T) {
	kit := newTestKit(t)

	_, err := kit.EvalTS(context.Background(), "my-agents.ts", `
		const a = agent({
			model: "openai/gpt-4o-mini",
			name: "coder",
			instructions: "You code.",
		});
	`)
	if err != nil {
		t.Fatalf("Create agent: %v", err)
	}

	resources, err := kit.ListResources("agent")
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(resources) == 0 {
		t.Fatal("expected at least 1 agent resource")
	}

	found := false
	for _, r := range resources {
		if r.Name == "coder" {
			found = true
			if r.Source != "my-agents.ts" {
				t.Errorf("source = %q, want my-agents.ts", r.Source)
			}
			if r.Type != "agent" {
				t.Errorf("type = %q, want agent", r.Type)
			}
		}
	}
	if !found {
		t.Errorf("agent 'coder' not found in resources: %+v", resources)
	}
}

func TestResourceRegistry_ToolRegistration(t *testing.T) {
	kit := newTestKit(t)

	_, err := kit.EvalTS(context.Background(), "my-tools.ts", `
		const t1 = createTool({
			id: "calculator",
			description: "Adds numbers",
			inputSchema: z.object({ a: z.number(), b: z.number() }),
			execute: async ({ a, b }) => ({ result: a + b }),
		});
	`)
	if err != nil {
		t.Fatalf("Create tool: %v", err)
	}

	resources, err := kit.ListResources("tool")
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}

	found := false
	for _, r := range resources {
		if r.ID == "calculator" {
			found = true
			if r.Source != "my-tools.ts" {
				t.Errorf("source = %q, want my-tools.ts", r.Source)
			}
		}
	}
	if !found {
		t.Errorf("tool 'calculator' not found: %+v", resources)
	}
}

func TestResourceRegistry_SourceTracking(t *testing.T) {
	kit := newTestKit(t)

	kit.EvalTS(context.Background(), "file-a.ts", `
		agent({ model: "openai/gpt-4o-mini", name: "agentA", instructions: "A" });
		createTool({ id: "toolA", description: "A", inputSchema: z.object({}), execute: async () => ({}) });
	`)

	kit.EvalTS(context.Background(), "file-b.ts", `
		agent({ model: "openai/gpt-4o-mini", name: "agentB", instructions: "B" });
	`)

	fromA, err := kit.ResourcesFrom("file-a.ts")
	if err != nil {
		t.Fatalf("ResourcesFrom: %v", err)
	}
	if len(fromA) != 2 {
		t.Errorf("file-a.ts: expected 2 resources, got %d: %+v", len(fromA), fromA)
	}

	fromB, err := kit.ResourcesFrom("file-b.ts")
	if err != nil {
		t.Fatalf("ResourcesFrom: %v", err)
	}
	if len(fromB) != 1 {
		t.Errorf("file-b.ts: expected 1 resource, got %d: %+v", len(fromB), fromB)
	}
}

func TestResourceRegistry_Idempotency(t *testing.T) {
	kit := newTestKit(t)

	kit.EvalTS(context.Background(), "setup.ts", `
		agent({ model: "openai/gpt-4o-mini", name: "worker", instructions: "V1" });
	`)
	kit.EvalTS(context.Background(), "setup.ts", `
		agent({ model: "openai/gpt-4o-mini", name: "worker", instructions: "V2" });
	`)

	resources, _ := kit.ListResources("agent")
	count := 0
	for _, r := range resources {
		if r.Name == "worker" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 'worker' agent (idempotent), got %d", count)
	}
}

func TestResourceRegistry_TeardownFile(t *testing.T) {
	kit := newTestKit(t)

	kit.EvalTS(context.Background(), "ephemeral.ts", `
		agent({ model: "openai/gpt-4o-mini", name: "temp1", instructions: "temp" });
		agent({ model: "openai/gpt-4o-mini", name: "temp2", instructions: "temp" });
	`)
	kit.EvalTS(context.Background(), "permanent.ts", `
		agent({ model: "openai/gpt-4o-mini", name: "keeper", instructions: "keep" });
	`)

	all, _ := kit.ListResources("agent")
	if len(all) < 3 {
		t.Fatalf("expected 3 agents, got %d", len(all))
	}

	removed, err := kit.TeardownFile("ephemeral.ts")
	if err != nil {
		t.Fatalf("TeardownFile: %v", err)
	}
	if removed != 2 {
		t.Errorf("removed = %d, want 2", removed)
	}

	remaining, _ := kit.ListResources("agent")
	for _, r := range remaining {
		if r.Name == "temp1" || r.Name == "temp2" {
			t.Errorf("torn-down agent %q still in registry", r.Name)
		}
	}

	found := false
	for _, r := range remaining {
		if r.Name == "keeper" {
			found = true
		}
	}
	if !found {
		t.Error("keeper agent should still exist after teardown of ephemeral.ts")
	}
}

func TestResourceRegistry_ListAll(t *testing.T) {
	kit := newTestKit(t)

	kit.EvalTS(context.Background(), "mixed.ts", `
		agent({ model: "openai/gpt-4o-mini", name: "myAgent", instructions: "hi" });
		createTool({ id: "myTool", description: "tool", inputSchema: z.object({}), execute: async () => ({}) });
	`)

	all, err := kit.ListResources()
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 resources, got %d: %+v", len(all), all)
	}

	types := map[string]bool{}
	for _, r := range all {
		types[r.Type] = true
	}
	if !types["agent"] {
		t.Error("expected agent type in list")
	}
	if !types["tool"] {
		t.Error("expected tool type in list")
	}
}

func TestResourceRegistry_RemoveResource(t *testing.T) {
	kit := newTestKit(t)

	kit.EvalTS(context.Background(), "setup.ts", `
		agent({ model: "openai/gpt-4o-mini", name: "removable", instructions: "temp" });
	`)

	resources, _ := kit.ListResources("agent")
	found := false
	for _, r := range resources {
		if r.Name == "removable" {
			found = true
		}
	}
	if !found {
		t.Fatal("agent should exist before removal")
	}

	if err := kit.RemoveResource("agent", "removable"); err != nil {
		t.Fatalf("RemoveResource: %v", err)
	}

	resources2, _ := kit.ListResources("agent")
	for _, r := range resources2 {
		if r.Name == "removable" {
			t.Error("agent should be gone after removal")
		}
	}
}
