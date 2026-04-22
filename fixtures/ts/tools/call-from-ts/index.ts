// Test: call a Go-registered tool directly from .ts
// The "uppercase" tool is registered in Go before this runs.
// The type probe on ToolsInput proves the canonical heterogeneous
// shape (ToolAction | VercelTool | VercelToolV5 | ProviderDefinedTool)
// is imported and usable from "agent".
import type { ToolsInput } from "agent";
import { tools, output } from "kit";

// Compile-time probe — heterogeneous tool map.
const _toolsInputShape: ToolsInput | undefined = undefined;
void _toolsInputShape;

const result = await tools.call("uppercase", { text: "hello brainlet" });

output(result);
