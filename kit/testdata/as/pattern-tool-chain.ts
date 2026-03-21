import { bus, setState, log, JSONObject, JSONValue } from "brainkit";

export function run(): i32 {
  // Step 1: call echo with step=1
  bus.askAsyncRaw("tools.call", '{"name":"echo","input":{"step":1}}', "onStep1");
  return 0;
}

export function onStep1(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "step1 empty");
    return;
  }
  log("step1 result: " + payload);

  const parsed = JSONValue.parse(payload);
  if (parsed.isNull()) {
    setState("error", "step1 null");
    return;
  }

  // Step 2: chain from step 1 result
  const step2Input = new JSONObject()
    .setInt("step", 2)
    .setString("prev", payload);
  const step2Payload = new JSONObject()
    .setString("name", "echo")
    .set("input", step2Input);
  bus.askAsyncRaw("tools.call", step2Payload.toString(), "onStep2");
}

export function onStep2(topic: string, payload: string): void {
  if (payload.length == 0) {
    setState("error", "step2 empty");
    return;
  }
  log("step2 result: " + payload);

  if (!payload.includes("2")) {
    setState("error", "step2 missing '2'");
    return;
  }
  setState("ok", "true");
}
