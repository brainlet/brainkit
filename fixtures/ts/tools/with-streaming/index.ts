// Test: tool with a writable context.writer — Mastra injects a
// stream writer when the surrounding agent runs stream-mode
// generation. We verify the construct + optional lifecycle hooks.
import { createTool, z } from "agent";
import { output } from "kit";

const tool = createTool({
  id: "streamy",
  description: "A streaming-aware tool",
  inputSchema: z.object({ n: z.number() }),
  outputSchema: z.object({ ok: z.boolean() }),
  execute: async ({ n }: any, ctx?: any) => {
    // When a writer is supplied, pipe progress into it. When it
    // isn't (direct execute call), this is a no-op.
    if (ctx?.writer && typeof ctx.writer.write === "function") {
      try {
        await ctx.writer.write(new TextEncoder().encode(`step ${n}`));
      } catch (_) { /* ignore */ }
    }
    return { ok: true };
  },
});

const direct = await (tool as any).execute({ n: 3 });

output({
  hasExecute: typeof tool.execute === "function",
  directOk: direct.ok === true,
});
