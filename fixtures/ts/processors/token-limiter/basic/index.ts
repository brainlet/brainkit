import { TokenLimiterProcessor } from "agent";
import { output } from "kit";

const p = new TokenLimiterProcessor({ maxTokens: 500 });
output({ id: p.id, hasProcessOutputStream: typeof (p as any).processOutputStream === "function" });
