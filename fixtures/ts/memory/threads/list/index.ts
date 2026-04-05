import { Memory, InMemoryStore } from "agent";
import { output } from "kit";
const store = new InMemoryStore();
const mem = new Memory({ storage: store });
const t1 = await mem.createThread({ resourceId: "user-a" });
const t2 = await mem.createThread({ resourceId: "user-a" });
const t3 = await mem.createThread({ resourceId: "user-b" });
// Verify all created and retrievable
const found1 = await mem.getThreadById({ threadId: t1.id });
const found2 = await mem.getThreadById({ threadId: t2.id });
const found3 = await mem.getThreadById({ threadId: t3.id });
output({
  created: 3,
  allFound: found1 !== null && found2 !== null && found3 !== null,
  distinctIds: t1.id !== t2.id && t2.id !== t3.id,
});
