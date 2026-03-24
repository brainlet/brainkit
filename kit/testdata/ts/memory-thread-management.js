// Test: Full Memory thread management API
// Verifies: saveThread, getThreadById, listThreads, updateThread, deleteThread, deleteMessages
import { Memory, LibSQLStore } from "agent";
import { output } from "kit";

const store = new LibSQLStore({ id: "thread-mgmt" });
const mem = new Memory({ storage: store, options: { lastMessages: 10 } });
const now = new Date().toISOString();
const results = {};

// 1. saveThread — create two threads
try {
  await mem.saveThread({ thread: { id: "t1", title: "Thread One", resourceId: "u1", createdAt: now, updatedAt: now } });
  await mem.saveThread({ thread: { id: "t2", title: "Thread Two", resourceId: "u1", createdAt: now, updatedAt: now } });
  results.saveThread = "ok";
} catch(e) { results.saveThread = "error: " + e.message; }

// 2. getThreadById
try {
  const t = await mem.getThreadById({ threadId: "t1" });
  results.getThreadById = t ? t.id : "null";
  results.getThreadTitle = t?.title || "null";
} catch(e) { results.getThreadById = "error: " + e.message; }

// 3. listThreads
try {
  const list = await mem.listThreads({ resourceId: "u1" });
  results.listThreads = list?.threads?.length || 0;
} catch(e) { results.listThreads = "error: " + e.message; }

// 4. updateThread — Mastra signature: updateThread({ id, title, metadata })
try {
  await mem.updateThread({ id: "t1", title: "Updated Title", metadata: {} });
  const updated = await mem.getThreadById({ threadId: "t1" });
  results.updateThread = updated?.title || "null";
} catch(e) { results.updateThread = "error: " + e.message; }

// 5. saveMessages — add messages to t1
try {
  await mem.saveMessages({ messages: [
    { id: "m1", threadId: "t1", content: JSON.stringify({ content: "Hello" }), role: "user", type: "v2", createdAt: now, resourceId: "u1" },
    { id: "m2", threadId: "t1", content: JSON.stringify({ content: "Hi there" }), role: "assistant", type: "v2", createdAt: now, resourceId: "u1" },
  ] });
  results.saveMessages = "ok";
} catch(e) { results.saveMessages = "error: " + e.message; }

// 6. recall — get messages back
try {
  const recalled = await mem.recall({ threadId: "t1", resourceId: "u1" });
  results.recallCount = recalled?.messages?.length || 0;
} catch(e) { results.recallCount = "error: " + e.message; }

// 7. deleteMessages
try {
  await mem.deleteMessages(["m1"]);
  const afterDelete = await mem.recall({ threadId: "t1", resourceId: "u1" });
  results.afterDeleteCount = afterDelete?.messages?.length || 0;
} catch(e) { results.afterDeleteCount = "error: " + e.message; }

// 8. deleteThread — Mastra signature: deleteThread(threadId: string)
try {
  await mem.deleteThread("t2");
  const deleted = await mem.getThreadById({ threadId: "t2" });
  results.deleteThread = (deleted === null || deleted === undefined) ? "deleted" : "still exists";
} catch(e) { results.deleteThread = "error: " + e.message; }

// 9. listThreads after delete — should be 1
try {
  const list2 = await mem.listThreads({ resourceId: "u1" });
  results.listAfterDelete = list2?.threads?.length || 0;
} catch(e) { results.listAfterDelete = "error: " + e.message; }

output(results);
