import { callAgent, parseResult, log } from "wasm";

export function run(): i32 {
  // Stage 1: initial agent call
  const raw1 = callAgent("test-helper", "first prompt");
  if (raw1.length == 0) return 1;
  log("stage1: " + raw1);

  const parsed1 = parseResult(raw1);
  if (parsed1.isNull()) return 2;

  // Check for text field in first result
  const obj1 = parsed1.asObject();
  const text1 = obj1.getString("text");
  if (text1.length == 0) return 3;

  // Stage 2: follow-up call using first result
  const followUp = "based on: " + text1 + " - continue";
  const raw2 = callAgent("test-helper", followUp);
  if (raw2.length == 0) return 4;
  log("stage2: " + raw2);

  return 0;
}
