// Test: Memory thread management API
import { Memory, LibSQLStore, output } from "brainlet";

const store = new LibSQLStore({ id: "thread-mgmt" });

const mem = new Memory({
  storage: store,
  options: { lastMessages: 10 },
});

const results = {};

// saveThread
try {
  await mem.saveThread({
    thread: { id: "t1", title: "Test", resourceId: "u1", createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
  });
  results.saveThread = "ok";
} catch(e) {
  results.saveThread = "error: " + e.message;
}

// getThreadById
try {
  const t = await mem.getThreadById({ threadId: "t1" });
  results.getThreadById = t ? t.id : "null";
} catch(e) {
  results.getThreadById = "error: " + e.message;
}

// listThreads
try {
  const list = await mem.listThreads({ resourceId: "u1" });
  results.listThreads = list?.threads?.length || 0;
} catch(e) {
  results.listThreads = "error: " + e.message;
}

output(results);
