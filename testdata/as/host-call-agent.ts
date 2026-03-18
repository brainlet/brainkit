import { callAgent, parseResult } from "wasm";

export function run(): i32 {
  const raw = callAgent("test-helper", "say hello");
  if (raw.length == 0) return 1;

  const parsed = parseResult(raw);
  if (parsed.isNull()) return 2;

  // Verify the result has a "text" field
  const obj = parsed.asObject();
  const text = obj.getString("text");
  if (text.length == 0) return 3;

  return 0;
}
