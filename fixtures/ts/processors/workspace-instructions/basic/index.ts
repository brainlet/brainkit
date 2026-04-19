import { WorkspaceInstructionsProcessor } from "agent";
import { output } from "kit";

const p = new WorkspaceInstructionsProcessor({});
output({ id: p.id, hasProcessInputStep: typeof (p as any).processInputStep === "function" });
