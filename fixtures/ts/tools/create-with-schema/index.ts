// Test: canonical schema-driven inference on createTool.
//
// No explicit generics on createTool and NO type annotation on
// the `execute` parameter — the canonical schema-typed generics
// (`TInputSchema` / `TOutputSchema`) + the `InferSchema<...>`
// helper derive `execute`'s first-arg shape from the schema
// captured at the call site, matching
// `@mastra/core/tools/tool.ts` in clones/mastra.
import { Agent, createTool, z } from "agent";
import { model, output } from "kit";

const calculator = createTool({
  id: "calculator",
  description: "Adds two numbers",
  inputSchema: z.object({ a: z.number(), b: z.number() }),
  outputSchema: z.object({ result: z.number() }),
  // ↓ No `:any`, no type annotation. `{ a, b }` is typed as
  //   `{ a: number; b: number }` via inference through
  //   `InferSchema<TInputSchema>` — schema drives the payload
  //   type, not a hand-written generic.
  execute: async ({ a, b }) => ({ result: a + b }),
});

// ── Compile-time probes ────────────────────────────────────────
//
// (1) Assignability probe — mirrors VAL-TOOLS-006's evidence
// contract: the inferred `execute` parameter MUST match
// `{ a: number; b: number }`. A mis-shaped probe (e.g.
// `{ a: string; b: number }`) raises TS2322 thanks to the
// exact-shape equality check below.
const _checkInput: (input: { a: number; b: number }) => Promise<
  { result: number } | { error: true; message: string; [key: string]: unknown }
> = calculator.execute!;
void _checkInput;

// (2) Exact-shape equality probe — verifies that the inferred
// `execute` parameter is EXACTLY `{ a: number; b: number }`
// (not widened to `any` / `unknown` / `Record<string, unknown>`
// and not broadened to a superset like `{ a: number; b: number;
// c?: number }`). The `Equals<A, B>` alias uses the standard
// "assignability in both directions via identical thunk shapes"
// pattern; it resolves to `true` only when A and B are the same
// type per TypeScript's structural equivalence.
type Equals<A, B> = (<T>() => T extends A ? 1 : 2) extends (<T>() => T extends B ? 1 : 2) ? true : false;
type ExecuteArg = Parameters<NonNullable<typeof calculator.execute>>[0];
const _exactShape: Equals<ExecuteArg, { a: number; b: number }> = true;
void _exactShape;

// ── Runtime behavior ───────────────────────────────────────────
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
