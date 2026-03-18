import { callToolRaw, parseResult, JSONObject, log } from "wasm";

export function run(): i32 {
  // Step 1: call echo with step=1
  const raw1 = callToolRaw("echo", '{"step":1}');
  if (raw1.length == 0) return 1;
  log("step1 result: " + raw1);

  const parsed1 = parseResult(raw1);
  if (parsed1.isNull()) return 2;

  // Step 2: build new payload with step=2 and chain from step 1
  const step2Input = new JSONObject()
    .setInt("step", 2)
    .setString("prev", raw1);
  const raw2 = callToolRaw("echo", step2Input.toString());
  if (raw2.length == 0) return 3;
  log("step2 result: " + raw2);

  const parsed2 = parseResult(raw2);
  if (parsed2.isNull()) return 4;

  // Verify step 2 is present in the final result
  if (!raw2.includes("2")) return 5;

  return 0;
}
