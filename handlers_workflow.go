package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/workflow"
	"github.com/brainlet/brainkit/sdk/messages"
)

// kernelAIGenerator adapts the Kernel's EvalTS to the workflow.AIGenerator interface.
type kernelAIGenerator struct {
	kernel *Kernel
}

func (g *kernelAIGenerator) GenerateText(ctx context.Context, prompt string) (string, error) {
	script := fmt.Sprintf(
		`var __r = await globalThis.__agent_embed.generateText({model: model("openai","gpt-4o-mini"), prompt: %q}); return __r.text;`,
		prompt)
	return g.kernel.EvalTS(ctx, "__wf_ai_generate.ts", script)
}

func (g *kernelAIGenerator) EmbedText(ctx context.Context, text string) (string, error) {
	script := fmt.Sprintf(
		`var __r = await globalThis.__agent_embed.embed({model: embeddingModel("openai","text-embedding-3-small"), value: %q}); return JSON.stringify(__r.embedding);`,
		text)
	return g.kernel.EvalTS(ctx, "__wf_ai_embed.ts", script)
}

// WorkflowDomain handles workflow.run/status/cancel/list/history bus commands.
// Note: kernelAIGenerator (also in this file) stays on *Kernel for EvalTS.
type WorkflowDomain struct {
	engine *workflow.Engine
}

func newWorkflowDomain(engine *workflow.Engine) *WorkflowDomain {
	return &WorkflowDomain{engine: engine}
}

func (d *WorkflowDomain) Run(ctx context.Context, req messages.WorkflowRunMsg) (*messages.WorkflowRunResp, error) {
	var opts []workflow.RunOption
	if len(req.HostResults) > 0 {
		opts = append(opts, workflow.WithHostResults(req.HostResults))
	}
	runID, err := d.engine.Run(ctx, req.WorkflowID, req.Input, opts...)
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
