// Test: sandbox context is available
import { sandbox, output } from "brainlet";

output({
  id: sandbox.id,
  namespace: sandbox.namespace,
  callerID: sandbox.callerID,
});
