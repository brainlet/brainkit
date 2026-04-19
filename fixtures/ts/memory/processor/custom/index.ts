// Test: Memory `processors` is deprecated + removed in Mastra. We
// assert the construction throws a clear deprecation error pointing
// users to the Agent-level Input/Output processor system.
import { Memory, InMemoryStore } from "agent";
import { output } from "kit";

const dummyProcessor = {
  name: "dummy",
  process(messages: any[]) {
    return messages;
  },
};

let errorMsg = "";
try {
  new Memory({
    storage: new InMemoryStore(),
    options: { lastMessages: 5 },
    processors: [dummyProcessor],
  } as any);
} catch (e: any) {
  errorMsg = String(e?.message || e).substring(0, 200);
}

output({
  rejected: errorMsg.length > 0,
  deprecationExplained: errorMsg.toLowerCase().includes("deprecat")
    || errorMsg.toLowerCase().includes("removed")
    || errorMsg.toLowerCase().includes("input/output processor"),
});
