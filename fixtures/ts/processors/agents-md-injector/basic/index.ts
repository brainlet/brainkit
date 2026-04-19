import { AgentsMDInjector } from "agent";
import { output } from "kit";

const p = new AgentsMDInjector({});
output({ id: p.id, hasProcessInputStep: typeof (p as any).processInputStep === "function" });
