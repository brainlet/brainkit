package brainkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
)

// ── Workflow Bus Command Handlers ──
// All delegate to Mastra's Workflow APIs via JS eval through __kit_registry.
// No in-memory run tracking — Mastra manages runs internally.

func handleWorkflowStart(ctx context.Context, kernel *Kernel, req messages.WorkflowStartMsg) (*messages.WorkflowStartResp, error) {
	inputJSON := "null"
	if len(req.InputData) > 0 {
		inputJSON = string(req.InputData)
	}
	script := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.get("workflow", %q);
		if (!entry || !entry.ref) throw new BrainkitError("workflow not found: " + %q, "NOT_FOUND");
		var wf = entry.ref;
		var run = await wf.createRun();
		var result = await run.start({ inputData: JSON.parse(%q) });
		return JSON.stringify({
			runId: run.runId || "",
			status: result.status || "unknown",
			steps: result.steps || null,
		});
	`, req.Name, req.Name, inputJSON)
	resultJSON, err := kernel.EvalTS(ctx, "__workflow_start.ts", script)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowStartResp
	if err := json.Unmarshal([]byte(resultJSON), &resp); err != nil {
		return nil, fmt.Errorf("workflow.start: parse result: %w", err)
	}
	return &resp, nil
}

func handleWorkflowStartAsync(ctx context.Context, kernel *Kernel, req messages.WorkflowStartAsyncMsg) (*messages.WorkflowStartAsyncResp, error) {
	inputJSON := "null"
	if len(req.InputData) > 0 {
		inputJSON = string(req.InputData)
	}
	script := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.get("workflow", %q);
		if (!entry || !entry.ref) throw new BrainkitError("workflow not found: " + %q, "NOT_FOUND");
		var wf = entry.ref;
		var run = await wf.createRun();
		var runId = run.runId || "";
		run.start({ inputData: JSON.parse(%q) }).then(function(result) {
			__go_brainkit_bus_emit("workflow.completed." + runId, JSON.stringify({
				runId: runId, name: %q, status: result.status || "unknown", steps: result.steps || null,
			}));
		}).catch(function(err) {
			__go_brainkit_bus_emit("workflow.completed." + runId, JSON.stringify({
				runId: runId, name: %q, status: "failed", error: err.message || String(err),
			}));
		});
		return JSON.stringify({ runId: runId });
	`, req.Name, req.Name, inputJSON, req.Name, req.Name)
	resultJSON, err := kernel.EvalTS(ctx, "__workflow_start_async.ts", script)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowStartAsyncResp
	if err := json.Unmarshal([]byte(resultJSON), &resp); err != nil {
		return nil, fmt.Errorf("workflow.startAsync: parse result: %w", err)
	}
	return &resp, nil
}

func handleWorkflowStatus(ctx context.Context, kernel *Kernel, req messages.WorkflowStatusMsg) (*messages.WorkflowStatusResp, error) {
	script := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.get("workflow", %q);
		if (!entry || !entry.ref) throw new BrainkitError("workflow not found: " + %q, "NOT_FOUND");
		var wf = entry.ref;
		var runState = await wf.getWorkflowRunById(%q);
		if (!runState) throw new BrainkitError("workflow run not found: " + %q, "NOT_FOUND");
		return JSON.stringify({
			runId: %q,
			status: runState.status || "unknown",
			steps: runState.steps || null,
		});
	`, req.Name, req.Name, req.RunID, req.RunID, req.RunID)
	resultJSON, err := kernel.EvalTS(ctx, "__workflow_status.ts", script)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowStatusResp
	if err := json.Unmarshal([]byte(resultJSON), &resp); err != nil {
		return nil, fmt.Errorf("workflow.status: parse result: %w", err)
	}
	return &resp, nil
}

func handleWorkflowResume(ctx context.Context, kernel *Kernel, req messages.WorkflowResumeMsg) (*messages.WorkflowResumeResp, error) {
	resumeJSON := "null"
	if len(req.ResumeData) > 0 {
		resumeJSON = string(req.ResumeData)
	}
	stepArg := "undefined"
	if req.Step != "" {
		stepArg = fmt.Sprintf("%q", req.Step)
	}
	script := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.get("workflow", %q);
		if (!entry || !entry.ref) throw new BrainkitError("workflow not found: " + %q, "NOT_FOUND");
		var wf = entry.ref;
		var run = await wf.createRun({ runId: %q });
		var opts = { resumeData: JSON.parse(%q) };
		var stepVal = %s;
		if (stepVal !== undefined) opts.step = stepVal;
		var result = await run.resume(opts);
		return JSON.stringify({
			status: result.status || "unknown",
			steps: result.steps || null,
		});
	`, req.Name, req.Name, req.RunID, resumeJSON, stepArg)
	resultJSON, err := kernel.EvalTS(ctx, "__workflow_resume.ts", script)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowResumeResp
	if err := json.Unmarshal([]byte(resultJSON), &resp); err != nil {
		return nil, fmt.Errorf("workflow.resume: parse result: %w", err)
	}
	return &resp, nil
}

func handleWorkflowCancel(ctx context.Context, kernel *Kernel, req messages.WorkflowCancelMsg) (*messages.WorkflowCancelResp, error) {
	script := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.get("workflow", %q);
		if (!entry || !entry.ref) throw new BrainkitError("workflow not found: " + %q, "NOT_FOUND");
		var wf = entry.ref;
		var run = await wf.createRun({ runId: %q });
		await run.cancel();
		return JSON.stringify({ cancelled: true });
	`, req.Name, req.Name, req.RunID)
	resultJSON, err := kernel.EvalTS(ctx, "__workflow_cancel.ts", script)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowCancelResp
	if err := json.Unmarshal([]byte(resultJSON), &resp); err != nil {
		return nil, fmt.Errorf("workflow.cancel: parse result: %w", err)
	}
	return &resp, nil
}

func handleWorkflowList(ctx context.Context, kernel *Kernel, _ messages.WorkflowListMsg) (*messages.WorkflowListResp, error) {
	script := `
		var entries = globalThis.__kit_registry.list("workflow");
		var result = [];
		for (var i = 0; i < entries.length; i++) {
			var e = entries[i];
			var ref = globalThis.__kit_registry.get("workflow", e.name);
			result.push({
				name: e.name,
				source: e.source || "",
				hasInput: !!(ref && ref.ref && ref.ref.inputSchema),
				hasOutput: !!(ref && ref.ref && ref.ref.outputSchema),
			});
		}
		return JSON.stringify({ workflows: result });
	`
	resultJSON, err := kernel.EvalTS(ctx, "__workflow_list.ts", script)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowListResp
	if err := json.Unmarshal([]byte(resultJSON), &resp); err != nil {
		return nil, fmt.Errorf("workflow.list: parse result: %w", err)
	}
	return &resp, nil
}

func handleWorkflowRuns(ctx context.Context, kernel *Kernel, req messages.WorkflowRunsMsg) (*messages.WorkflowRunsResp, error) {
	statusFilter := "undefined"
	if req.Status != "" {
		statusFilter = fmt.Sprintf("%q", req.Status)
	}
	script := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.get("workflow", %q);
		if (!entry || !entry.ref) throw new BrainkitError("workflow not found: " + %q, "NOT_FOUND");
		var wf = entry.ref;
		var statusVal = %s;
		var opts = {};
		if (statusVal !== undefined) opts.status = statusVal;
		var result = await wf.listWorkflowRuns(opts);
		return JSON.stringify({
			runs: result.runs || [],
			total: result.total || 0,
		});
	`, req.Name, req.Name, statusFilter)
	resultJSON, err := kernel.EvalTS(ctx, "__workflow_list_runs.ts", script)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowRunsResp
	if err := json.Unmarshal([]byte(resultJSON), &resp); err != nil {
		return nil, fmt.Errorf("workflow.runs: parse result: %w", err)
	}
	return &resp, nil
}

func handleWorkflowRestart(ctx context.Context, kernel *Kernel, req messages.WorkflowRestartMsg) (*messages.WorkflowRestartResp, error) {
	script := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.get("workflow", %q);
		if (!entry || !entry.ref) throw new BrainkitError("workflow not found: " + %q, "NOT_FOUND");
		var wf = entry.ref;
		var run = await wf.createRun({ runId: %q });
		var result = await run.restart();
		return JSON.stringify({
			status: result.status || "unknown",
			steps: result.steps || null,
		});
	`, req.Name, req.Name, req.RunID)
	resultJSON, err := kernel.EvalTS(ctx, "__workflow_restart.ts", script)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowRestartResp
	if err := json.Unmarshal([]byte(resultJSON), &resp); err != nil {
		return nil, fmt.Errorf("workflow.restart: parse result: %w", err)
	}
	return &resp, nil
}
