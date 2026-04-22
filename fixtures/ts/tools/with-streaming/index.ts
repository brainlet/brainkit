// Test: streaming-aware tool — context.writer is typed as the
// canonical ToolStream (not `any`). When Mastra runs a tool in
// a streaming agent run, it injects a ToolStream through
// context.writer; direct execute calls leave it undefined.
import { createTool, z } from "agent";
import type { MCPToolExecutionContext, ToolExecutionContext, ToolStream } from "agent";
import { output } from "kit";

const streamy = createTool<
  "streamy",
  { n: number },
  { ok: boolean; written: number }
>({
  id: "streamy",
  description: "A streaming-aware tool",
  inputSchema: z.object({ n: z.number() }),
  outputSchema: z.object({ ok: z.boolean(), written: z.number() }),
  execute: async ({ n }, ctx) => {
    // Compile-time probe: ctx.writer is typed `ToolStream | undefined`.
    const writer: ToolStream | undefined = ctx?.writer;
    let written = 0;
    if (writer) {
      try {
        await writer.write({ step: n });
        written = 1;
      } catch {
        /* noop — stream may not be attached in direct-execute mode */
      }
    }
    return { ok: true, written };
  },
});

// Type probes that the canonical ToolExecutionContext exposes
// the `writer` + `mcp` slots we claim in agent.d.ts.
const _writerProbe: ToolExecutionContext['writer'] = undefined;
const _mcpProbe: MCPToolExecutionContext | undefined = undefined;
void _writerProbe;
void _mcpProbe;

// Direct execute — no writer injected → `written` stays 0.
const direct = await streamy.execute!({ n: 3 });

output({
  hasExecute: typeof streamy.execute === "function",
  directOk: (direct as { ok: boolean }).ok === true,
  written: (direct as { written: number }).written,
});
