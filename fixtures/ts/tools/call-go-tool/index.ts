// Test: call Go-registered tools from .ts
import { tools, output } from "kit";

const echoResult = await tools.call("echo", { message: "from typescript" });
const addResult = await tools.call("add", { a: 17, b: 25 });

output({
  echoed: (echoResult as any).echoed,
  sum: (addResult as any).sum,
});
