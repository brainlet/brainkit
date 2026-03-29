// Test: storage("name") resolves from the Kernel's resource pool
import { output } from "kit";

const store = storage("default");

output({
    resolved: store !== null && store !== undefined,
    hasName: typeof store.name === "string",
    type: "pool",
});
