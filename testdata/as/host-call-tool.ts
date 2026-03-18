import { callTool, parseResult, JSONObject } from "wasm";

export function run(): i32 {
  const args = new JSONObject().setString("key", "val");
  const raw = callTool("echo", args);
  if (raw.length == 0) return 1;

  const parsed = parseResult(raw);
  if (parsed.isNull()) return 2;

  return 0;
}
