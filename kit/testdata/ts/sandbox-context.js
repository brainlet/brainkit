// Test: sandbox context is available
// NOTE: sandbox export is removed. Context info is now available via kit.
import { output } from "kit";

// The sandbox context (id, namespace, callerID) is now accessed differently.
// TODO: Update once the new kit context API is defined.
output({
  id: globalThis.__kit_sandbox_id || "unavailable",
  namespace: globalThis.__kit_namespace || "unavailable",
  callerID: globalThis.__kit_caller_id || "unavailable",
});
