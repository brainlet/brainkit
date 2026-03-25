import { Memory, InMemoryStore } from "agent";
import { output } from "kit";
const store = new InMemoryStore();
const mem = new Memory({ storage: store });
const thread = await mem.createThread({ resourceId: "user-1" });
const fetched = await mem.getThreadById({ threadId: thread.id });
output({ created: thread.id.length > 0, fetched: fetched !== null, idMatch: thread.id === fetched?.id });
