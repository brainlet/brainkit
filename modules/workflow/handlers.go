package workflow

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/brainlet/brainkit/modules/workflow/workflowmsg"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// All handlers delegate to runtime/dispatch.js via Kit.CallJS.

func (m *Module) handleStart(ctx context.Context, req workflowmsg.WorkflowStartMsg) (*workflowmsg.WorkflowStartResp, error) {
	raw, err := m.call(ctx, "__brainkit.workflow.start", map[string]any{
		"name":      req.Name,
		"inputData": jsonOrNull(req.InputData),
	})
	if err != nil {
		return nil, mapWorkflowError(err)
	}
	var resp workflowmsg.WorkflowStartResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *Module) handleStartAsync(ctx context.Context, req workflowmsg.WorkflowStartAsyncMsg) (*workflowmsg.WorkflowStartAsyncResp, error) {
	raw, err := m.call(ctx, "__brainkit.workflow.startAsync", map[string]any{
		"name":      req.Name,
		"inputData": jsonOrNull(req.InputData),
	})
	if err != nil {
		return nil, mapWorkflowError(err)
	}
	var resp workflowmsg.WorkflowStartAsyncResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *Module) handleStatus(ctx context.Context, req workflowmsg.WorkflowStatusMsg) (*workflowmsg.WorkflowStatusResp, error) {
	raw, err := m.call(ctx, "__brainkit.workflow.status", req)
	if err != nil {
		return nil, mapWorkflowError(err)
	}
	var resp workflowmsg.WorkflowStatusResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *Module) handleResume(ctx context.Context, req workflowmsg.WorkflowResumeMsg) (*workflowmsg.WorkflowResumeResp, error) {
	raw, err := m.call(ctx, "__brainkit.workflow.resume", map[string]any{
		"name":       req.Name,
		"runId":      req.RunID,
		"step":       req.Step,
		"resumeData": jsonOrNull(req.ResumeData),
	})
	if err != nil {
		return nil, mapWorkflowError(err)
	}
	var resp workflowmsg.WorkflowResumeResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *Module) handleCancel(ctx context.Context, req workflowmsg.WorkflowCancelMsg) (*workflowmsg.WorkflowCancelResp, error) {
	raw, err := m.call(ctx, "__brainkit.workflow.cancel", req)
	if err != nil {
		return nil, mapWorkflowError(err)
	}
	var resp workflowmsg.WorkflowCancelResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *Module) handleList(ctx context.Context, _ workflowmsg.WorkflowListMsg) (*workflowmsg.WorkflowListResp, error) {
	raw, err := m.call(ctx, "__brainkit.workflow.list", nil)
	if err != nil {
		return nil, mapWorkflowError(err)
	}
	var resp workflowmsg.WorkflowListResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *Module) handleRuns(ctx context.Context, req workflowmsg.WorkflowRunsMsg) (*workflowmsg.WorkflowRunsResp, error) {
	raw, err := m.call(ctx, "__brainkit.workflow.runs", req)
	if err != nil {
		return nil, mapWorkflowError(err)
	}
	var resp workflowmsg.WorkflowRunsResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (m *Module) handleRestart(ctx context.Context, req workflowmsg.WorkflowRestartMsg) (*workflowmsg.WorkflowRestartResp, error) {
	raw, err := m.call(ctx, "__brainkit.workflow.restart", req)
	if err != nil {
		return nil, mapWorkflowError(err)
	}
	var resp workflowmsg.WorkflowRestartResp
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

func mapWorkflowError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(err.Error())
	msg = strings.TrimPrefix(msg, "BrainkitError: ")

	switch {
	case strings.Contains(msg, "workflow run not found:"):
		return &sdkerrors.NotFoundError{
			Resource: "workflow run",
			Name:     extractWorkflowErrorValue(msg, "workflow run not found:"),
		}
	case strings.Contains(msg, "workflow not found:"):
		return &sdkerrors.NotFoundError{
			Resource: "workflow",
			Name:     extractWorkflowErrorValue(msg, "workflow not found:"),
		}
	case strings.Contains(msg, "workflow run is already complete:"),
		strings.Contains(msg, "workflow run is not suspended:"),
		strings.Contains(msg, "workflow.resume failed:"),
		strings.Contains(msg, "workflow.cancel failed:"),
		strings.Contains(msg, "workflow.restart failed:"):
		return &sdkerrors.ValidationError{Field: "workflow", Message: msg}
	default:
		return err
	}
}

func extractWorkflowErrorValue(msg, marker string) string {
	idx := strings.Index(msg, marker)
	if idx < 0 {
		return ""
	}
	value := strings.TrimSpace(msg[idx+len(marker):])
	if nl := strings.IndexByte(value, '\n'); nl >= 0 {
		value = value[:nl]
	}
	return strings.TrimSpace(value)
}
