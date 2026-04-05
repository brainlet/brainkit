// Test: kit.register, kit.list, kit.unregister lifecycle
import { kit, output } from "kit";
kit.register("memory", "test-mem", {});
kit.register("workflow", "test-wf", {});
const before = kit.list();
const memBefore = before.filter((r: any) => r.type === "memory").length;
kit.unregister("memory", "test-mem");
const after = kit.list();
const memAfter = after.filter((r: any) => r.type === "memory").length;
output({ memBefore, memAfter, removed: memBefore > memAfter });
