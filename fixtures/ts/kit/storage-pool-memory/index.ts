// Test: storage("mem") resolves InMemoryStore from the resource pool
import { output } from "kit";

const store = storage("mem");

output({
    resolved: store !== null && store !== undefined,
    hasName: typeof store.name === "string",
    type: "memory-pool",
});
