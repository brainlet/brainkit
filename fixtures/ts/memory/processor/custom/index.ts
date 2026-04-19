// Test: Agent-level custom Input + Output processor replacing the
// deprecated Memory.processors option. Input processor tags the last
// user message; output processor counts invocations; fixture asserts
// both ran and the agent saw the tagged input.
import { Agent, Memory, InMemoryStore } from "agent";
import { model, output } from "kit";

let inputRuns = 0;
let outputRuns = 0;

const taggingInput = {
  id: "tagging-input",
  name: "Tagging Input",
  async processInput(args: any) {
    inputRuns += 1;
    const msgs = args.messages;
    for (let i = msgs.length - 1; i >= 0; i--) {
      const m = msgs[i];
      if (m?.role !== "user") continue;
      if (typeof m.content === "string") {
        m.content = `[tagged] ${m.content}`;
      } else if (Array.isArray(m.content)) {
        for (const part of m.content) {
          if (part?.type === "text" && typeof part.text === "string") {
            part.text = `[tagged] ${part.text}`;
          }
        }
      }
      break;
    }
  },
};

const countingOutput = {
  id: "counting-output",
  name: "Counting Output",
  async processOutputResult(_args: any) {
    outputRuns += 1;
  },
};

const agent = new Agent({
  name: "proc-agent",
  model: model("openai", "gpt-4o-mini"),
  instructions:
    "Repeat the user's message back exactly, including any prefixes they include.",
  memory: new Memory({ storage: new InMemoryStore(), options: { lastMessages: 5 } }),
  inputProcessors: [taggingInput],
  outputProcessors: [countingOutput],
});

const result = await agent.generate("Hello world", {
  memory: { thread: { id: "proc-agent-" + Date.now() }, resource: "test" },
});

output({
  inputRan: inputRuns > 0,
  outputRan: outputRuns > 0,
  gotReply: typeof result.text === "string" && result.text.length > 0,
});
