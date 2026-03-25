// Test: registry.has() and registry.list()
import { registry, output } from "kit";
const hasNonExistent = registry.has("provider", "nonexistent");
const providers = registry.list("provider");
const storages = registry.list("storage");
output({
  hasNonExistent: hasNonExistent,
  providersIsArray: Array.isArray(providers),
  storagesIsArray: Array.isArray(storages),
});
