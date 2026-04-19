// Test: InMemoryStore direct API — no Agent, no LLM. Exercises the
// StorageInstance surface: construction, `getStore("memory")` domain
// access, and thread CRUD on the memory sub-store.
import { InMemoryStore } from "agent";
import { output } from "kit";

const store = new InMemoryStore({ id: "storage-inmem-basic" });

const memDomain: any = await store.getStore("memory");
const hasMemoryDomain = memDomain !== undefined && memDomain !== null;

let threadWritten = false;
let threadReadBack = false;
let threadDeleted = false;
let error = "";
try {
  if (hasMemoryDomain) {
    const threadId = "t-" + Date.now();
    await memDomain.saveThread({
      thread: {
        id: threadId,
        resourceId: "test-resource",
        title: "Storage fixture thread",
        metadata: {},
        createdAt: new Date(),
        updatedAt: new Date(),
      },
    });
    threadWritten = true;

    const read = await memDomain.getThreadById({ threadId });
    threadReadBack = read?.id === threadId;

    await memDomain.deleteThread({ threadId });
    const readAfter = await memDomain.getThreadById({ threadId });
    threadDeleted = readAfter === null || readAfter === undefined;
  }
} catch (e: any) {
  error = String(e?.message || e).substring(0, 200);
}

output({
  constructed: store !== null && store !== undefined,
  hasMemoryDomain,
  threadWritten,
  threadReadBack,
  threadDeleted,
  errorIfAny: error,
});
