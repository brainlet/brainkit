package kit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
)

// WorkflowsDomain handles workflow execution.
type WorkflowsDomain struct {
	kit *Kernel
}

func newWorkflowsDomain(k *Kernel) *WorkflowsDomain {
	return &WorkflowsDomain{kit: k}
}

func (d *WorkflowsDomain) Run(ctx context.Context, req messages.WorkflowRunMsg) (*messages.WorkflowRunResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__wf_run.ts", `
		var req = globalThis.__pending_req;
		var wf = globalThis.__kit_workflows && globalThis.__kit_workflows[req.name];
		if (!wf) throw new Error("workflow '" + req.name + "' not found");
		var run = await createWorkflowRun(wf);
		var result = await run.start({ inputData: req.input });
		result.runId = run.runId;
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("workflows.run: %w", err)
	}
	return &messages.WorkflowRunResp{Result: raw}, nil
}

func (d *WorkflowsDomain) Resume(ctx context.Context, req messages.WorkflowResumeMsg) (*messages.WorkflowResumeResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__wf_resume.ts", `
		var req = globalThis.__pending_req;
		var result = await resumeWorkflow(req.runId, req.stepId, req.data);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("workflows.resume: %w", err)
	}
	return &messages.WorkflowResumeResp{Result: raw}, nil
}

func (d *WorkflowsDomain) Cancel(ctx context.Context, req messages.WorkflowCancelMsg) (*messages.WorkflowCancelResp, error) {
	_, err := d.kit.evalDomain(ctx, req, "__wf_cancel.ts", `
		var req = globalThis.__pending_req;
		var run = globalThis.__kit_pending_runs && globalThis.__kit_pending_runs[req.runId];
		if (!run) throw new Error("workflow run '" + req.runId + "' not found");
		run.cancel();
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("workflows.cancel: %w", err)
	}
	return &messages.WorkflowCancelResp{OK: true}, nil
}

func (d *WorkflowsDomain) Status(ctx context.Context, req messages.WorkflowStatusMsg) (*messages.WorkflowStatusResp, error) {
	raw, err := d.kit.evalDomain(ctx, req, "__wf_status.ts", `
		var req = globalThis.__pending_req;
		var run = globalThis.__kit_pending_runs && globalThis.__kit_pending_runs[req.runId];
		if (!run) throw new Error("workflow run '" + req.runId + "' not found");
		return JSON.stringify({ status: run.status || "unknown", step: run.currentStep || "" });
	`)
	if err != nil {
		return nil, fmt.Errorf("workflows.status: %w", err)
	}
	var resp messages.WorkflowStatusResp
	json.Unmarshal(raw, &resp)
	return &resp, nil
}
