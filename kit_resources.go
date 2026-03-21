package brainkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

// EvalTS runs .ts-style code with brainlet imports destructured.
func (k *Kit) EvalTS(ctx context.Context, filename, code string) (string, error) {
	wrapped := fmt.Sprintf(`(async () => {
		globalThis.__kit_current_source = %q;
		const { agent, createTool, createSubagent, createWorkflow, createStep, createMemory, z, ai, wasm, tools, tool, bus, agents, mcp, sandbox, output, Memory, InMemoryStore, LibSQLStore, UpstashStore, PostgresStore, MongoDBStore, LibSQLVector, PgVector, MongoDBVector, generateText, streamText, generateObject, streamObject, createWorkflowRun, resumeWorkflow, createScorer, runEvals, scorers, processors, RequestContext, MDocument, GraphRAG, createVectorQueryTool, createDocumentChunkerTool, createGraphRAGTool, rerank, rerankWithScorer, Workspace, LocalFilesystem, LocalSandbox, createHarness } = globalThis.__kit;
		%s
	})()`, filename, code)

	// If the Bridge is currently in an eval/await loop (e.g., we're being called
	// from a Go tool callback during agent.generate/stream), use EvalOnJSThread.
	// This handles two cases:
	//   1. Direct tool callback (same goroutine) → calls ctx.Eval directly
	//   2. Bus handler (different goroutine) → schedules via ctx.Schedule + channel
	// Both avoid the mutex deadlock.
	if k.bridge.IsEvalBusy() {
		return k.bridge.EvalOnJSThread(filename, wrapped)
	}

	return k.agents.Eval(ctx, filename, wrapped)
}

// EvalModule runs code as an ES module with import { ... } from "kit".
func (k *Kit) EvalModule(ctx context.Context, filename, code string) (string, error) {
	k.bridge.Eval("__clear_result.js", `delete globalThis.__module_result`)

	val, err := k.bridge.EvalAsyncModule(filename, code)
	if err != nil {
		return "", fmt.Errorf("brainkit: eval module: %w", err)
	}
	if val != nil {
		val.Free()
	}

	result, err := k.bridge.Eval("__get_result.js",
		`typeof globalThis.__module_result !== 'undefined' ? String(globalThis.__module_result) : ""`)
	if err != nil {
		return "", err
	}
	defer result.Free()
	return result.String(), nil
}

// RegisterTool is a convenience method for registering typed Go tools.
// The JSON Schema is generated automatically from T's struct tags.
//
// Example:
//
//	type AddInput struct {
//	    A float64 `json:"a" desc:"First number"`
//	    B float64 `json:"b" desc:"Second number"`
//	}
//	kit.RegisterTool("brainlet/math@1.0.0/add", registry.TypedTool[AddInput]{
//	    Description: "Adds two numbers",
//	    Execute: func(ctx context.Context, input AddInput) (any, error) {
//	        return map[string]any{"result": input.A + input.B}, nil
//	    },
//	})
func RegisterTool[T any](k *Kit, name string, tool registry.TypedTool[T]) error {
	return registry.Register(k.Tools, name, tool)
}

// ResumeWorkflow resumes a suspended workflow run from the Go side.
// runId: the workflow run's ID
// stepId: which step to resume (empty string for auto-detect)
// resumeDataJSON: JSON-encoded resume data to pass to the step
func (k *Kit) ResumeWorkflow(ctx context.Context, runId, stepId, resumeDataJSON string) (string, error) {
	stepArg := "undefined"
	if stepId != "" {
		stepArg = fmt.Sprintf("%q", stepId)
	}

	code := fmt.Sprintf(`(async () => {
		var result = await globalThis.__kit.resumeWorkflow(%q, %s, %s);
		globalThis.__module_result = JSON.stringify(result);
	})()`, runId, stepArg, resumeDataJSON)

	val, err := k.bridge.EvalAsync("__resume_workflow.js", code)
	if err != nil {
		return "", fmt.Errorf("resume workflow %s: %w", runId, err)
	}
	if val != nil {
		val.Free()
	}

	result, err := k.bridge.Eval("__get_resume_result.js", `typeof globalThis.__module_result !== 'undefined' ? String(globalThis.__module_result) : ""`)
	if err != nil {
		return "", err
	}
	defer result.Free()
	return result.String(), nil
}

// ListResources returns all tracked resources, optionally filtered by type.
// Types: "agent", "tool", "workflow", "wasm", "memory", "harness"
func (k *Kit) ListResources(resourceType ...string) ([]ResourceInfo, error) {
	filter := ""
	if len(resourceType) > 0 {
		filter = resourceType[0]
	}
	code := fmt.Sprintf(`return JSON.stringify(globalThis.__kit_registry.list(%q))`, filter)
	result, err := k.EvalTS(context.Background(), "__list_resources.ts", code)
	if err != nil {
		return nil, err
	}
	var resources []ResourceInfo
	if err := json.Unmarshal([]byte(result), &resources); err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	return resources, nil
}

// ResourcesFrom returns all resources created by a specific .ts file.
func (k *Kit) ResourcesFrom(filename string) ([]ResourceInfo, error) {
	code := fmt.Sprintf(`return JSON.stringify(globalThis.__kit_registry.listBySource(%q))`, filename)
	result, err := k.EvalTS(context.Background(), "__resources_from.ts", code)
	if err != nil {
		return nil, err
	}
	var resources []ResourceInfo
	if err := json.Unmarshal([]byte(result), &resources); err != nil {
		return nil, fmt.Errorf("resources from: %w", err)
	}
	return resources, nil
}

// TeardownFile removes all resources created by a specific .ts file.
// Returns the number of resources removed.
func (k *Kit) TeardownFile(filename string) (int, error) {
	code := fmt.Sprintf(`
		var resources = globalThis.__kit_registry.listBySource(%q);
		var count = 0;
		// Teardown in reverse order (LIFO — last created, first destroyed)
		for (var i = resources.length - 1; i >= 0; i--) {
			globalThis.__kit_registry.unregister(resources[i].type, resources[i].id);
			count++;
		}
		return JSON.stringify(count);
	`, filename)
	result, err := k.EvalTS(context.Background(), "__teardown_file.ts", code)
	if err != nil {
		return 0, err
	}
	var count int
	if err := json.Unmarshal([]byte(result), &count); err != nil {
		return 0, nil
	}
	return count, nil
}

// RemoveResource removes a specific resource by type and ID.
func (k *Kit) RemoveResource(resourceType, id string) error {
	code := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.unregister(%q, %q);
		return JSON.stringify(entry !== null);
	`, resourceType, id)
	_, err := k.EvalTS(context.Background(), "__remove_resource.ts", code)
	return err
}

// ---------------------------------------------------------------------------
// WASM Convenience Methods
// ---------------------------------------------------------------------------

// ListWASMModules returns metadata for all compiled WASM modules.
func (k *Kit) ListWASMModules() ([]WASMModuleInfo, error) {
	return k.wasm.ListModules(), nil
}

// GetWASMModule returns metadata for a specific module by name.
func (k *Kit) GetWASMModule(name string) (*WASMModuleInfo, error) {
	info := k.wasm.GetModule(name)
	return info, nil
}

// RemoveWASMModule unloads a compiled module by name.
// Fails if a shard is deployed from this module (undeploy first).
func (k *Kit) RemoveWASMModule(name string) error {
	resp, err := bus.AskSync(k.Bus, context.Background(), bus.Message{
		Topic:    "wasm.remove",
		CallerID: k.callerID,
		Payload:  json.RawMessage(fmt.Sprintf(`{"name":%q}`, name)),
	})
	if err != nil {
		return err
	}
	// Check for error in response payload (bus wraps handler errors as {"error":"..."})
	var result struct {
		Error   string `json:"error"`
		Removed bool   `json:"removed"`
	}
	json.Unmarshal(resp.Payload, &result)
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	if !result.Removed {
		return fmt.Errorf("wasm module %q not found", name)
	}
	return nil
}

// DeployWASM activates a compiled shard — calls init(), registers event handlers.
func (k *Kit) DeployWASM(name string) (*ShardDescriptor, error) {
	resp, err := bus.AskSync(k.Bus, context.Background(), bus.Message{
		Topic:    "wasm.deploy",
		CallerID: k.callerID,
		Payload:  json.RawMessage(fmt.Sprintf(`{"name":%q}`, name)),
	})
	if err != nil {
		return nil, err
	}
	var desc ShardDescriptor
	json.Unmarshal(resp.Payload, &desc)
	return &desc, nil
}

// UndeployWASM removes all event subscriptions for a deployed shard.
func (k *Kit) UndeployWASM(name string) error {
	_, err := bus.AskSync(k.Bus, context.Background(), bus.Message{
		Topic:    "wasm.undeploy",
		CallerID: k.callerID,
		Payload:  json.RawMessage(fmt.Sprintf(`{"name":%q}`, name)),
	})
	return err
}

// DescribeWASM returns the shard's registrations (mode, handlers, state key).
func (k *Kit) DescribeWASM(name string) (*ShardDescriptor, error) {
	resp, err := bus.AskSync(k.Bus, context.Background(), bus.Message{
		Topic:    "wasm.describe",
		CallerID: k.callerID,
		Payload:  json.RawMessage(fmt.Sprintf(`{"name":%q}`, name)),
	})
	if err != nil {
		return nil, err
	}
	var desc ShardDescriptor
	json.Unmarshal(resp.Payload, &desc)
	if desc.Module == "" {
		return nil, nil
	}
	return &desc, nil
}

// ListDeployedWASM returns all active shard descriptors.
func (k *Kit) ListDeployedWASM() []ShardDescriptor {
	return k.wasm.ListDeployedShards()
}

// InjectWASMEvent manually triggers a shard handler (for testing and SDK use).
func (k *Kit) InjectWASMEvent(shardName, topic string, payload json.RawMessage) (*WASMEventResult, error) {
	return k.wasm.InjectEvent(context.Background(), shardName, topic, payload)
}
