// Test: call a Go-registered tool directly from .ts
// The "uppercase" tool is registered in Go before this runs.
import { tools, output } from "kit";

const result = await tools.call("uppercase", { text: "hello brainlet" });

output(result);
