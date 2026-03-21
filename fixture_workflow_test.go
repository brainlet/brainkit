//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"testing"
)

func TestFixture_TS_WorkflowBasic(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/workflow-basic.js")

	result, err := kit.EvalModule(context.Background(), "workflow-basic.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Status   string `json:"status"`
		Result   any    `json:"result"`
		Expected string `json:"expected"`
		Match    bool   `json:"match"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.Match {
		t.Errorf("workflow result mismatch: status=%s result=%v expected=%s", out.Status, out.Result, out.Expected)
	}
	t.Logf("workflow-basic: status=%s match=%v", out.Status, out.Match)
}

func TestFixture_TS_WorkflowWithAgent(t *testing.T) {
	kit := newTestKit(t)
	code := loadFixture(t, "testdata/ts/workflow-with-agent.js")

	result, err := kit.EvalModule(context.Background(), "workflow-with-agent.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Status    string `json:"status"`
		Result    any    `json:"result"`
		HasAnswer bool   `json:"hasAnswer"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.HasAnswer {
		t.Errorf("workflow+agent: status=%s result=%v", out.Status, out.Result)
	}
	t.Logf("workflow-with-agent: status=%s hasAnswer=%v result=%v", out.Status, out.HasAnswer, out.Result)
}

func TestFixture_TS_WorkflowSuspendResume(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/workflow-suspend-resume.js")

	result, err := kit.EvalModule(context.Background(), "workflow-suspend-resume.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Phase          string `json:"phase"`
		Status         string `json:"status"`
		Result         any    `json:"result"`
		SuspendPayload any    `json:"suspendPayload"`
		RunId          string `json:"runId"`
		Approved       bool   `json:"approved"`
		Error          string `json:"error"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Error != "" {
		t.Logf("full output: %s", result)
		t.Fatalf("fixture error: %s", out.Error)
	}
	if out.Phase != "complete" {
		t.Errorf("expected phase=complete, got %s", out.Phase)
	}
	if out.Status != "success" {
		t.Errorf("expected status=success, got %s", out.Status)
	}
	if !out.Approved {
		t.Errorf("result should contain approver David: %v", out.Result)
	}
	if out.RunId == "" {
		t.Error("expected non-empty runId")
	}
	t.Logf("fixture workflow-suspend-resume: phase=%s status=%s runId=%s result=%v suspendPayload=%v",
		out.Phase, out.Status, out.RunId, out.Result, out.SuspendPayload)
}

func TestFixture_TS_WorkflowState(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/workflow-state.js")

	result, err := kit.EvalModule(context.Background(), "workflow-state.js", code)
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Status    string   `json:"status"`
		Result    any      `json:"result"`
		HasItems  bool     `json:"hasItems"`
		HasCount  bool     `json:"hasCount"`
		Items     []string `json:"items"`
		FirstItem bool     `json:"firstItem"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Status != "success" {
		t.Errorf("expected success, got %s: %s", out.Status, result)
	}
	if !out.HasItems {
		t.Errorf("expected 3 items, got %v", out.Items)
	}
	if !out.HasCount {
		t.Errorf("expected count=3, got %v", out.Result)
	}
	if !out.FirstItem {
		t.Errorf("first item should be 'test-first', got %v", out.Items)
	}
	t.Logf("fixture workflow-state: status=%s items=%v", out.Status, out.Items)
}

func TestFixture_TS_WorkflowParallel(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/workflow-parallel.js")
	result, err := kit.EvalModule(context.Background(), "workflow-parallel.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Status  string `json:"status"`
		Result  any    `json:"result"`
		Correct bool   `json:"correct"`
	}
	json.Unmarshal([]byte(result), &out)
	if !out.Correct {
		t.Errorf("parallel incorrect: status=%s result=%v raw=%s", out.Status, out.Result, result)
	}
	t.Logf("workflow-parallel: status=%s result=%v correct=%v", out.Status, out.Result, out.Correct)
}

func TestFixture_TS_WorkflowBranch(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/workflow-branch.js")
	result, err := kit.EvalModule(context.Background(), "workflow-branch.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		High    struct{ Status, Label string } `json:"high"`
		Low     struct{ Status, Label string } `json:"low"`
		Correct bool                           `json:"correct"`
	}
	json.Unmarshal([]byte(result), &out)
	if !out.Correct {
		t.Errorf("branch incorrect: high=%v low=%v raw=%s", out.High, out.Low, result)
	}
	t.Logf("workflow-branch: high=%s low=%s correct=%v", out.High.Label, out.Low.Label, out.Correct)
}

func TestFixture_TS_WorkflowForeach(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/workflow-foreach.js")
	result, err := kit.EvalModule(context.Background(), "workflow-foreach.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Status  string `json:"status"`
		Result  any    `json:"result"`
		IsArray bool   `json:"isArray"`
	}
	json.Unmarshal([]byte(result), &out)
	if out.Status != "success" {
		t.Errorf("foreach: status=%s result=%v raw=%s", out.Status, out.Result, result)
	}
	t.Logf("workflow-foreach: status=%s isArray=%v result=%v", out.Status, out.IsArray, out.Result)
}

func TestFixture_TS_WorkflowLoop(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/workflow-loop.js")
	result, err := kit.EvalModule(context.Background(), "workflow-loop.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Status          string `json:"status"`
		Result          any    `json:"result"`
		LoopedCorrectly bool   `json:"loopedCorrectly"`
	}
	json.Unmarshal([]byte(result), &out)
	if !out.LoopedCorrectly {
		t.Errorf("loop incorrect: status=%s result=%v raw=%s", out.Status, out.Result, result)
	}
	t.Logf("workflow-loop: status=%s loopedCorrectly=%v result=%v", out.Status, out.Result, out.LoopedCorrectly)
}

func TestFixture_TS_WorkflowSleep(t *testing.T) {
	kit := newTestKitNoKey(t)
	code := loadFixture(t, "testdata/ts/workflow-sleep.js")
	result, err := kit.EvalModule(context.Background(), "workflow-sleep.js", code)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Status      string `json:"status"`
		Elapsed     int    `json:"elapsed"`
		SleptEnough bool   `json:"sleptEnough"`
	}
	json.Unmarshal([]byte(result), &out)
	if !out.SleptEnough {
		t.Errorf("sleep too short: elapsed=%dms raw=%s", out.Elapsed, result)
	}
	t.Logf("workflow-sleep: status=%s elapsed=%dms sleptEnough=%v", out.Status, out.Elapsed, out.SleptEnough)
}

func TestKit_ResumeWorkflow(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Create a workflow with a suspend step via EvalTS
	// Note: EvalTS already destructures __kit, so we use those names directly
	setupCode := `
var step1 = createStep({
  id: "greet",
  inputSchema: z.object({ name: z.string() }),
  outputSchema: z.object({ greeting: z.string() }),
  execute: async ({ inputData, resumeData, suspend }) => {
    if (!resumeData) {
      return suspend({ draft: "Hello " + inputData.name });
    }
    if (resumeData.confirmed) {
      return { greeting: "Hello " + inputData.name + "!" };
    }
    return { greeting: "Cancelled" };
  },
});

var wf = createWorkflow({
  id: "greet-wf",
  inputSchema: z.object({ name: z.string() }),
  outputSchema: z.object({ greeting: z.string() }),
}).then(step1).commit();

var run = await createWorkflowRun(wf);
var result = await run.start({ inputData: { name: "Alice" } });
globalThis.__test_runId = run.runId;
globalThis.__test_status = result.status;
`
	_, err := kit.EvalTS(context.Background(), "setup.js", setupCode)
	if err != nil {
		t.Fatal(err)
	}

	// Check it's suspended
	statusVal, _ := kit.bridge.Eval("check.js", `globalThis.__test_status`)
	defer statusVal.Free()
	if statusVal.String() != "suspended" {
		t.Fatalf("expected suspended, got %s", statusVal.String())
	}

	runIdVal, _ := kit.bridge.Eval("get-runid.js", `globalThis.__test_runId`)
	defer runIdVal.Free()
	runId := runIdVal.String()
	t.Logf("Workflow suspended, runId=%s", runId)

	// Resume from Go
	result, err := kit.ResumeWorkflow(context.Background(), runId, "greet", `{"confirmed": true}`)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Resume result: %s", result)

	var out struct {
		Status string `json:"status"`
		Result any    `json:"result"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Status != "success" {
		t.Errorf("expected success, got %s", out.Status)
	}
}
