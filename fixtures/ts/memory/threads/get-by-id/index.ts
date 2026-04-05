import { Memory, InMemoryStore } from "agent";
import { output } from "kit";
const store = new InMemoryStore();
const mem = new Memory({ storage: store });
const t = await mem.createThread({ resourceId: "u1" });
const found = await mem.getThreadById({ threadId: t.id });
const missing = await mem.getThreadById({ threadId: "nonexistent-id" });
output({ found: found !== null, missing: missing === null, correctId: found?.id === t.id });
