import { registry, output } from "kit";

const hasProv = registry.has("provider", "nonexistent-provider");
const providers = registry.list("provider");
const storages = registry.list("storage");

output({
  hasNonexistent: hasProv,
  hasProviders: Array.isArray(providers),
  hasStorages: Array.isArray(storages)
});
