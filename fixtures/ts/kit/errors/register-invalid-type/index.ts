import { kit, output } from "kit";
let caught = "none";
try {
  kit.register("banana" as any, "test", {});
} catch (e: any) {
  caught = e.message || "error";
}
output({ caught, hasError: caught !== "none" });
