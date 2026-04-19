import { BatchPartsProcessor } from "agent";
import { output } from "kit";

const p = new BatchPartsProcessor({ batchSize: 10, flushIntervalMs: 100 });
output({ id: p.id, hasProcessOutputStream: typeof (p as any).processOutputStream === "function" });
