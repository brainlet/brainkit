// Test: Agent.network — supervisor delegates to sub-agents. Mastra
// requires a memory module on the supervisor for multi-agent routing.
import { Agent, Memory, InMemoryStore } from "agent";
import { model, output } from "kit";

const writer = new Agent({
  name: "writer",
  description: "Writes short technical summaries.",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Produce a one-sentence summary when asked.",
});

const editor = new Agent({
  name: "editor",
  description: "Polishes prose for clarity.",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Rewrite input so it's crisp and clear.",
});

const supervisor = new Agent({
  name: "supervisor",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Delegate to writer or editor as needed; return a final answer.",
  agents: { writer, editor },
  memory: new Memory({ storage: new InMemoryStore(), options: { lastMessages: 5 } }),
});

const stream: any = await supervisor.network(
  "Write a 1-sentence summary of what brainkit is, then polish it.",
  { memory: { thread: { id: "net-basic-" + Date.now() }, resource: "test" } } as any,
);

let chunks = 0;
for await (const _chunk of stream) {
  chunks += 1;
  if (chunks > 500) break;
}
const status = await stream.status;

output({
  hasRunId: typeof stream.runId === "string" && stream.runId.length > 0,
  hadChunks: chunks > 0,
  status: typeof status === "string" ? status : "",
});
