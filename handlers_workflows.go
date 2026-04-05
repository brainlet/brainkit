package brainkit

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/sdk/messages"
)

// ── Workflow Bus Command Handlers ──
// All delegate to runtime/dispatch.js via Kernel.callJS.

func handleWorkflowStart(ctx context.Context, kernel *Kernel, req messages.WorkflowStartMsg) (*messages.WorkflowStartResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.start", map[string]any{
		"name":      req.Name,
		"inputData": jsonOrNull(req.InputData),
	})
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowStartResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowStartAsync(ctx context.Context, kernel *Kernel, req messages.WorkflowStartAsyncMsg) (*messages.WorkflowStartAsyncResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.startAsync", map[string]any{
		"name":      req.Name,
		"inputData": jsonOrNull(req.InputData),
	})
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowStartAsyncResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowStatus(ctx context.Context, kernel *Kernel, req messages.WorkflowStatusMsg) (*messages.WorkflowStatusResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.status", req)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowStatusResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowResume(ctx context.Context, kernel *Kernel, req messages.WorkflowResumeMsg) (*messages.WorkflowResumeResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.resume", map[string]any{
		"name":       req.Name,
		"runId":      req.RunID,
		"step":       req.Step,
		"resumeData": jsonOrNull(req.ResumeData),
	})
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowResumeResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowCancel(ctx context.Context, kernel *Kernel, req messages.WorkflowCancelMsg) (*messages.WorkflowCancelResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.cancel", req)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowCancelResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowList(ctx context.Context, kernel *Kernel, _ messages.WorkflowListMsg) (*messages.WorkflowListResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.list", nil)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowListResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowRuns(ctx context.Context, kernel *Kernel, req messages.WorkflowRunsMsg) (*messages.WorkflowRunsResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.runs", req)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowRunsResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowRestart(ctx context.Context, kernel *Kernel, req messages.WorkflowRestartMsg) (*messages.WorkflowRestartResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.restart", req)
	if err != nil {
		return nil, err
	}
	var resp messages.WorkflowRestartResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// jsonOrNull returns the raw JSON if non-empty, or nil for JSON null.
func jsonOrNull(data json.RawMessage) any {
	if len(data) == 0 {
		return nil
	}
	return json.RawMessage(data)
}
