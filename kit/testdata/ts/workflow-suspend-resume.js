// Test: workflow suspend and resume — HITL approval pattern
import { createWorkflow, createStep, createWorkflowRun, resumeWorkflow, z, output } from "kit";

var log = [];
try {
  const approvalStep = createStep({
    id: "approval",
    inputSchema: z.object({ amount: z.number() }),
    outputSchema: z.object({ approved: z.boolean(), approver: z.string() }),
    execute: async ({ inputData, resumeData, suspend }) => {
      if (!resumeData) {
        return suspend({ question: "Approve?", amount: inputData.amount });
      }
      return { approved: resumeData.approved, approver: resumeData.approver };
    },
  });
  log.push("step-created");

  const workflow = createWorkflow({
    id: "approval-wf",
    inputSchema: z.object({ amount: z.number() }),
    outputSchema: z.object({ approved: z.boolean(), approver: z.string() }),
  }).then(approvalStep).commit();
  log.push("workflow-committed");

  const run = await createWorkflowRun(workflow);
  log.push("run-created:" + run.runId);

  const result1 = await run.start({ inputData: { amount: 500 } });
  log.push("started:" + result1.status);

  if (result1.status === "suspended") {
    log.push("resuming");
    const result2 = await resumeWorkflow(run.runId, "approval", {
      approved: true,
      approver: "David",
    });
    log.push("resumed:" + result2.status);

    output({
      phase: "complete",
      status: result2.status,
      result: result2.result,
      runId: run.runId,
      approved: result2.result?.approved === true && result2.result?.approver === "David",
      log: log,
    });
  } else {
    output({ error: "Expected suspended, got: " + result1.status, log: log });
  }
} catch(e) {
  output({ error: e.message, stack: (e.stack || "").substring(0, 500), log: log });
}
