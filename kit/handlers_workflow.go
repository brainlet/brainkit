package kit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/brainlet/brainkit/kit/workflow"
	"github.com/brainlet/brainkit/sdk/messages"
)

// WorkflowDomain handles workflow.run/status/cancel/list/history bus commands.
type WorkflowDomain struct {
	kit    *Kernel
	engine *workflow.Engine
}

func newWorkflowDomain(k *Kernel, engine *workflow.Engine) *WorkflowDomain {
	return &WorkflowDomain{kit: k, engine: engine}
}

func (d *WorkflowDomain) Run(ctx context.Context, req messages.WorkflowRunMsg) (*messages.WorkflowRunResp, error) {
	runID, err := d.engine.Run(ctx, req.WorkflowID, req.Input)
	if err != nil {
		return nil, err
	}
	return &messages.WorkflowRunResp{RunID: runID, Status: "running"}, nil
}

func (d *WorkflowDomain) Status(_ context.Context, req messages.WorkflowStatusMsg) (*messages.WorkflowStatusResp, error) {
	run, err := d.engine.GetRun(req.RunID)
	if err != nil {
		return nil, err
	}
	resp := &messages.WorkflowStatusResp{
		RunID:       run.RunID,
		WorkflowID:  run.WorkflowID,
		Status:      string(run.Status),
		CurrentStep: run.CurrentStep,
		StartedAt:   run.StartedAt.Format(time.RFC3339),
		Error:       run.Error,
		Output:      run.Output,
	}
	return resp, nil
}

func (d *WorkflowDomain) Cancel(_ context.Context, req messages.WorkflowCancelMsg) (*messages.WorkflowCancelResp, error) {
	if err := d.engine.CancelRun(req.RunID); err != nil {
		return nil, err
	}
	return &messages.WorkflowCancelResp{Cancelled: true}, nil
}

func (d *WorkflowDomain) List(_ context.Context, _ messages.WorkflowListMsg) (*messages.WorkflowListResp, error) {
	runs := d.engine.ListRuns()
	infos := make([]messages.WorkflowRunInfo, len(runs))
	for i, r := range runs {
		infos[i] = messages.WorkflowRunInfo{
			RunID:       r.RunID,
			WorkflowID:  r.WorkflowID,
			Status:      r.Status,
			CurrentStep: r.CurrentStep,
			StartedAt:   r.StartedAt,
			Error:       r.Error,
		}
	}
	return &messages.WorkflowListResp{Runs: infos}, nil
}

func (d *WorkflowDomain) History(_ context.Context, req messages.WorkflowHistoryMsg) (*messages.WorkflowHistoryResp, error) {
	entries, err := d.engine.GetJournal(req.RunID)
	if err != nil {
		return nil, err
	}
	data, _ := json.Marshal(entries)
	return &messages.WorkflowHistoryResp{Entries: data}, nil
}
