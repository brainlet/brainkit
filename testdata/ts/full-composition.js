// Test: full composition — uses agent, ai, tools, wasm, sandbox, z, createTool
// This is what a real developer .ts file looks like.
import { agent, ai, tools, wasm, sandbox, z, createTool, output } from "brainlet";

// 1. Check sandbox context
const ctx = { ns: sandbox.namespace, id: sandbox.id };

// 2. Direct AI call (LOCAL)
const aiResult = await ai.generate({
  model: "openai/gpt-4o-mini",
  prompt: "What is 2+2? Reply with just the number.",
});

// 3. Call a Go tool through the bus (PLATFORM)
const reversed = await tools.call("reverse", { text: "brainlet" });

// 4. Create a local tool with Zod schema
const concatTool = createTool({
  name: "concat",
  description: "Concatenates two strings",
  schema: z.object({
    a: z.string().describe("first string"),
    b: z.string().describe("second string"),
  }),
  execute: async ({ a, b }) => ({ result: a + b }),
});

// 5. Compile and run WASM
const wasmModule = await wasm.compile('export function run(): i32 { return 99; }');
const wasmResult = await wasm.run(wasmModule, {});

output({
  sandbox: ctx,
  aiText: aiResult.text,
  reversed: reversed.result,
  hasLocalTool: !!concatTool,
  wasmExitCode: wasmResult.exitCode,
});
