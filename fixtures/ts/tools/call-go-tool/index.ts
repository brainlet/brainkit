// Test: call Go-registered tools from .ts.
// Exercises ProviderDefinedTool and VercelToolV5 via type
// probes — the canonical tool surface Mastra accepts alongside
// native ToolAction instances.
import type { ProviderDefinedTool, VercelToolV5 } from "agent";
import { tools, output } from "kit";

// Compile-time probes — shapes importable from "agent".
const _providerDefined: ProviderDefinedTool | undefined = undefined;
const _vercelV5: VercelToolV5 | undefined = undefined;
void _providerDefined;
void _vercelV5;

const echoResult = await tools.call("echo", { message: "from typescript" });
const addResult = await tools.call("add", { a: 17, b: 25 });

output({
  echoed: (echoResult as any).echoed,
  sum: (addResult as any).sum,
});
