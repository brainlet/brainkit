// Test: createTool schema-driven input inference. No explicit
// type annotations on the execute callback — tsc infers the
// argument shape from the canonical 7-generic signature once
// generics are provided. The outputSchema narrows the return
// type so Mastra validates the tool's output end-to-end.
import { Agent, createTool, z } from "agent";
import type { ToolExecutionContext } from "agent";
import { model, output } from "kit";

// The three explicit generics (id, input shape, output shape)
// feed directly into Tool<TIn, TOut, ...> and into execute's
// (inputData, context) parameters — no destructure coercion.
const calculator = createTool<
  "calculator",
  { a: number; b: number },
  { result: number }
>({
  id: "calculator",
  description: "Adds two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  outputSchema: z.object({ result: z.number() }),
  // ↓ No `:any`, no type annotation. {a, b} is typed
  //   as `{ a: number; b: number }` via the generic slot.
  execute: async ({ a, b }) => ({ result: a + b }),
});

// Compile-time probe: the execute callback's first argument is
// typed exactly `{ a: number; b: number }`, the return type
// flows through as `{ result: number }`.
const _checkInput: (input: { a: number; b: number }, ctx?: ToolExecutionContext) => Promise<{ result: number } | unknown> =
  calculator.execute!;
void _checkInput;

const a = new Agent({
  name: "fixture",
  model: model("openai", "gpt-4o-mini"),
  instructions: "Use the calculator tool to add 17 and 25. Return ONLY the number.",
  tools: { calculator },
});

const result = await a.generate("What is 17 + 25?");

output({
  text: result.text,
  hasAnswer: result.text.includes("42"),
  toolCalls: result.toolCalls.length,
});
