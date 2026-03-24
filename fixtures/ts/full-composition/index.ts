// Test: full composition — uses Agent, generateText, tools, compile, z, createTool
// This is what a real developer .ts file looks like.
import { Agent, createTool, z } from "agent";
import { generateText } from "ai";
import { compile } from "compiler";
import { model, tools, output } from "kit";

// 1. Direct AI call (LOCAL)
const aiResult = await generateText({
  model: model("openai", "gpt-4o-mini"),
  prompt: "What is 2+2? Reply with just the number.",
});

// 2. Call a Go tool through the bus (PLATFORM)
const reversed = await tools.call("reverse", { text: "brainlet" });

// 3. Create a local tool with Zod schema
const concatTool = createTool({
  id: "concat",
  description: "Concatenates two strings",
  inputSchema: z.object({
    a: z.string().describe("first string"),
    b: z.string().describe("second string"),
  }),
  execute: async ({ a, b }) => ({ result: a + b }),
});

// 4. Compile and run WASM
const wasmModule = await compile('export function run(): i32 { return 99; }');
const wasmResult = await wasmModule.run({});

output({
  aiText: aiResult.text,
  reversed: reversed.result,
  hasLocalTool: !!concatTool,
  wasmExitCode: wasmResult.exitCode,
});
