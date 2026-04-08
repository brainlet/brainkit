package engine

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/sdk"
)

// ── Workflow Bus Command Handlers ──
// All delegate to runtime/dispatch.js via Kernel.callJS.

func handleWorkflowStart(ctx context.Context, kernel *Kernel, req sdk.WorkflowStartMsg) (*sdk.WorkflowStartResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.start", map[string]any{
		"name":      req.Name,
		"inputData": jsonOrNull(req.InputData),
	})
	if err != nil {
		return nil, err
	}
	var resp sdk.WorkflowStartResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowStartAsync(ctx context.Context, kernel *Kernel, req sdk.WorkflowStartAsyncMsg) (*sdk.WorkflowStartAsyncResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.startAsync", map[string]any{
		"name":      req.Name,
		"inputData": jsonOrNull(req.InputData),
	})
	if err != nil {
		return nil, err
	}
	var resp sdk.WorkflowStartAsyncResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowStatus(ctx context.Context, kernel *Kernel, req sdk.WorkflowStatusMsg) (*sdk.WorkflowStatusResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.status", req)
	if err != nil {
		return nil, err
	}
	var resp sdk.WorkflowStatusResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowResume(ctx context.Context, kernel *Kernel, req sdk.WorkflowResumeMsg) (*sdk.WorkflowResumeResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.resume", map[string]any{
		"name":       req.Name,
		"runId":      req.RunID,
		"step":       req.Step,
		"resumeData": jsonOrNull(req.ResumeData),
	})
	if err != nil {
		return nil, err
	}
	var resp sdk.WorkflowResumeResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowCancel(ctx context.Context, kernel *Kernel, req sdk.WorkflowCancelMsg) (*sdk.WorkflowCancelResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.cancel", req)
	if err != nil {
		return nil, err
	}
	var resp sdk.WorkflowCancelResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowList(ctx context.Context, kernel *Kernel, _ sdk.WorkflowListMsg) (*sdk.WorkflowListResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.list", nil)
	if err != nil {
		return nil, err
	}
	var resp sdk.WorkflowListResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowRuns(ctx context.Context, kernel *Kernel, req sdk.WorkflowRunsMsg) (*sdk.WorkflowRunsResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.runs", req)
	if err != nil {
		return nil, err
	}
	var resp sdk.WorkflowRunsResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func handleWorkflowRestart(ctx context.Context, kernel *Kernel, req sdk.WorkflowRestartMsg) (*sdk.WorkflowRestartResp, error) {
	raw, err := kernel.callJS(ctx, "__brainkit.workflow.restart", req)
	if err != nil {
		return nil, err
	}
	var resp sdk.WorkflowRestartResp
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
