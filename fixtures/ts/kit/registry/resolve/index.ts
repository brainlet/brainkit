// Test: registry.resolve() returns config or null
import { registry, output } from "kit";
const missing = registry.resolve("provider", "nonexistent");
output({ missingIsNull: missing === null });
