import { Memory, InMemoryStore } from "agent";
import { output } from "kit";
const store = new InMemoryStore();
const mem = new Memory({ storage: store, options: { lastMessages: 10 } });
const t = await mem.createThread({ resourceId: "u1" });
await mem.saveMessages({
  threadId: t.id,
  messages: [
    { role: "user", content: "Hello" },
    { role: "assistant", content: "Hi there!" },
    { role: "user", content: "How are you?" },
  ],
});
const recalled = await mem.recall({ threadId: t.id });
output({ messageCount: recalled.messages.length, hasMessages: recalled.messages.length > 0 });
