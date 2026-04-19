import { ToolCallFilter } from "agent";
import { output } from "kit";

const p = new ToolCallFilter({ allow: ["safe-tool"], deny: ["dangerous-tool"] });
output({ id: p.id, hasProcessInput: typeof (p as any).processInput === "function" });
