// Test: Full Memory thread management API
// Verifies: saveThread, getThreadById, listThreads, updateThread, deleteThread, deleteMessages
import { Memory, LibSQLStore } from "agent";
import { output } from "kit";

const store = new LibSQLStore({ id: "thread-mgmt" });
const mem = new Memory({ storage: store, options: { lastMessages: 10 } });
const now = new Date().toISOString();
const results: any = {};

// 1. saveThread — create two threads
try {
  await mem.saveThread({ thread: { id: "t1", title: "Thread One", resourceId: "u1" } });
  await mem.saveThread({ thread: { id: "t2", title: "Thread Two", resourceId: "u1" } });
  results.saveThread = "ok";
} catch(e: any) { results.saveThread = "error: " + e.message; }

// 2. getThreadById
try {
  const t = await mem.getThreadById({ threadId: "t1" });
  results.getThreadById = t ? t.id : "null";
  results.getThreadTitle = t?.title || "null";
} catch(e: any) { results.getThreadById = "error: " + e.message; }

// 3. listThreads
try {
  const list = await mem.listThreads({ resourceId: "u1" });
  results.listThreads = list?.length || 0;
} catch(e: any) { results.listThreads = "error: " + e.message; }

// 4. updateThread
try {
  await mem.updateThread({ threadId: "t1", title: "Updated Title" });
  const updated = await mem.getThreadById({ threadId: "t1" });
  results.updateThread = updated?.title || "null";
} catch(e: any) { results.updateThread = "error: " + e.message; }

// 5. saveMessages — add messages to t1
try {
  await mem.saveMessages({ threadId: "t1", messages: [
    { role: "user", content: "Hello" },
    { role: "assistant", content: "Hi there" },
  ] });
  results.saveMessages = "ok";
} catch(e: any) { results.saveMessages = "error: " + e.message; }

// 6. recall — get messages back
try {
  const recalled = await mem.recall({ threadId: "t1", resourceId: "u1" });
  results.recallCount = recalled?.messages?.length || 0;
} catch(e: any) { results.recallCount = "error: " + e.message; }

// 7. deleteMessages
try {
  await mem.deleteMessages({ threadId: "t1" });
  const afterDelete = await mem.recall({ threadId: "t1", resourceId: "u1" });
  results.afterDeleteCount = afterDelete?.messages?.length || 0;
} catch(e: any) { results.afterDeleteCount = "error: " + e.message; }

// 8. deleteThread
try {
  await mem.deleteThread("t2");
  const deleted = await mem.getThreadById({ threadId: "t2" });
  results.deleteThread = (deleted === null || deleted === undefined) ? "deleted" : "still exists";
} catch(e: any) { results.deleteThread = "error: " + e.message; }

// 9. listThreads after delete — should be 1
try {
  const list2 = await mem.listThreads({ resourceId: "u1" });
  results.listAfterDelete = list2?.length || 0;
} catch(e: any) { results.listAfterDelete = "error: " + e.message; }

output(results);
