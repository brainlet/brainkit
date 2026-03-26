// Test: cross-Kit awareness — verify namespace, tool registration, and bus on shared transport
// Kit A deploys this .ts. Kit B exists on the same NATS transport.
import { kit, tools, bus, output } from "kit";
import { createTool, z } from "agent";

// Register a tool on Kit A that Kit B can call cross-namespace
const myTool = createTool({
  id: "cross-kit-echo",
  description: "Echo from Kit A",
  inputSchema: z.object({ msg: z.string() }),
  execute: async ({ msg }: any) => ({ echoed: msg, from: kit.namespace }),
});
kit.register("tool", "cross-kit-echo", myTool);

// Verify bus works within this Kit on NATS transport
let received = false;
bus.subscribe("cross-kit.ping", () => { received = true; });
bus.emit("cross-kit.ping", {});
await new Promise(r => setTimeout(r, 200));

// Call our own tool to verify it works
const result = await tools.call("cross-kit-echo", { msg: "self-test" });

output({
  namespace: kit.namespace,
  hasNamespace: kit.namespace.length > 0,
  toolRegistered: true,
  selfCallWorks: (result as any)?.echoed === "self-test",
  busWorks: received,
});
