package kit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/bus"
)

func (k *Kit) handleWorkflows(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	switch msg.Topic {
	case "workflows.run":
		return k.handleWorkflowRun(ctx, msg)
	case "workflows.resume":
		return k.handleWorkflowResume(ctx, msg)
	case "workflows.cancel":
		return k.handleWorkflowCancel(ctx, msg)
	case "workflows.status":
		return k.handleWorkflowStatus(ctx, msg)
	default:
		return nil, fmt.Errorf("workflows: unknown topic %q", msg.Topic)
	}
}

func (k *Kit) handleWorkflowRun(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__wf_req.js", fmt.Sprintf("globalThis.__wf_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__wf_run.ts", `
		var req = globalThis.__wf_pending_req;
		var wf = globalThis.__kit_workflows && globalThis.__kit_workflows[req.name];
		if (!wf) throw new Error("workflow '" + req.name + "' not found");
		var run = await createWorkflowRun(wf);
		var result = await run.start({ triggerData: req.input });
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("workflows.run: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleWorkflowResume(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__wf_req.js", fmt.Sprintf("globalThis.__wf_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__wf_resume.ts", `
		var req = globalThis.__wf_pending_req;
		var result = await resumeWorkflow(req.runId, req.stepId, req.data);
		return JSON.stringify(result);
	`)
	if err != nil {
		return nil, fmt.Errorf("workflows.resume: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleWorkflowCancel(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__wf_req.js", fmt.Sprintf("globalThis.__wf_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__wf_cancel.ts", `
		var req = globalThis.__wf_pending_req;
		var run = globalThis.__kit_pending_runs && globalThis.__kit_pending_runs[req.runId];
		if (!run) throw new Error("workflow run '" + req.runId + "' not found");
		run.cancel();
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		return nil, fmt.Errorf("workflows.cancel: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}

func (k *Kit) handleWorkflowStatus(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	k.bridge.Eval("__wf_req.js", fmt.Sprintf("globalThis.__wf_pending_req = %s;", string(msg.Payload)))

	resultJSON, err := k.EvalTS(ctx, "__wf_status.ts", `
		var req = globalThis.__wf_pending_req;
		var run = globalThis.__kit_pending_runs && globalThis.__kit_pending_runs[req.runId];
		if (!run) throw new Error("workflow run '" + req.runId + "' not found");
		return JSON.stringify({ status: run.status || "unknown", step: run.currentStep || "" });
	`)
	if err != nil {
		return nil, fmt.Errorf("workflows.status: %w", err)
	}
	return &bus.Message{Payload: json.RawMessage(resultJSON)}, nil
}
