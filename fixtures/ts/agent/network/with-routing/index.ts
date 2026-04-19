// Test: Agent.network — routing + completion scorer options flow.
// Supervisor needs memory wired in for network mode.
import { Agent, Memory, InMemoryStore } from "agent";
import { model, output } from "kit";

const researcher = new Agent({
  name: "researcher",
  description: "Finds factual answers.",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Answer factual questions in 1-2 sentences.",
});

const supervisor = new Agent({
  name: "router",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Route every question to the researcher and return their answer.",
  agents: { researcher },
  memory: new Memory({ storage: new InMemoryStore(), options: { lastMessages: 5 } }),
});

const stream: any = await supervisor.network("What year was Go 1.0 released?", {
  routing: { strategy: "model" },
  memory: { thread: { id: "net-routing-" + Date.now() }, resource: "test" },
} as any);

let chunks = 0;
for await (const _chunk of stream) {
  chunks += 1;
  if (chunks > 500) break;
}

output({
  hasRunId: typeof stream.runId === "string" && stream.runId.length > 0,
  hadChunks: chunks > 0,
});
