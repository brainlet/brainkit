import { callTool, parseResult, log, busSend, JSONObject } from "wasm";

export function run(): i32 {
  // 1. Build input data
  const input = new JSONObject()
    .setString("action", "process")
    .setInt("value", 100);

  // 2. Call echo tool with the input
  const raw = callTool("echo", input);
  if (raw.length == 0) return 1;

  // 3. Parse the tool result
  const parsed = parseResult(raw);
  if (parsed.isNull()) return 2;

  // 4. Log the result
  log("processed: " + raw);

  // 5. Forward result via bus
  const output = new JSONObject().setString("status", "processed").setString("raw", raw);
  busSend("data.processed", output);

  return 0;
}
