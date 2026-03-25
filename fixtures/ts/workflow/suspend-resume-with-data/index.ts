// Test: workflow suspend with payload, resume with resumeData
import { createWorkflow, createStep, z } from "agent";
import { output } from "kit";

const approvalStep = createStep({
  id: "approval",
  inputSchema: z.object({ item: z.string() }),
  outputSchema: z.object({ approved: z.boolean(), approver: z.string() }),
  execute: async ({ inputData, suspend, resumeData }) => {
    if (!resumeData) {
      await suspend({ needsApproval: true, item: inputData.item });
      return { approved: false, approver: "" };
    }
    return { approved: resumeData.approved, approver: resumeData.approver };
  },
});

const wf = createWorkflow({
  id: "suspend-data-test",
  inputSchema: z.object({ item: z.string() }),
  outputSchema: z.object({ approved: z.boolean(), approver: z.string() }),
}).then(approvalStep).commit();

const run = await wf.createRun();
const suspended = await run.start({ inputData: { item: "deploy v2" } });

let resumeResult: any = null;
if (suspended.status === "suspended") {
  resumeResult = await run.resume({
    step: "approval",
    resumeData: { approved: true, approver: "david" },
  });
}

output({
  suspendedFirst: suspended.status === "suspended",
  finalStatus: resumeResult?.status || "none",
  approved: resumeResult?.steps?.approval?.output?.approved,
  approver: resumeResult?.steps?.approval?.output?.approver,
});
