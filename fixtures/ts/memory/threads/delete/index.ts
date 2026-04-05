import { Memory, InMemoryStore } from "agent";
import { output } from "kit";
const store = new InMemoryStore();
const mem = new Memory({ storage: store });
const t = await mem.createThread({ resourceId: "u1" });
await mem.deleteThread(t.id);
const afterDelete = await mem.getThreadById({ threadId: t.id });
output({ deleted: afterDelete === null });
