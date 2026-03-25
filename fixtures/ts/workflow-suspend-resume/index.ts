// Test: workflow suspend and resume — HITL approval pattern
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

var log = [];
try {
  const approvalStep = createStep({
    id: "approval",
    inputSchema: z.object({ amount: z.number() }),
    outputSchema: z.object({ approved: z.boolean(), approver: z.string() }),
    execute: async ({ inputData, resumeData, suspend }: any) => {
      if (!resumeData) {
        return await suspend({ question: "Approve?", amount: inputData.amount });
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

  const run = await workflow.createRun();
  log.push("run-created:" + run.runId);

  const result1 = await run.start({ inputData: { amount: 500 } });
  log.push("started:" + result1.status);

  if (result1.status === "suspended") {
    log.push("resuming");
    const result2 = await run.resume({
      step: "approval",
      resumeData: { approved: true, approver: "David" },
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
