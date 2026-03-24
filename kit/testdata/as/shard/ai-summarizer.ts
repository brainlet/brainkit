// Persistent shard: receives text, publishes to a .ts AI service, stores result.
// Tests: bus_publish to .ts service, callback processing response,
//        persistent state accumulating request count.
// Note: AI calls are now handled by .ts services, not WASM catalog commands.
import { setMode, on, reply, setState, getState, log, JSONValue, publish, emit } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("summarize.request", "handleRequest");
  on("summarize.last", "handleLast");
}

export function handleRequest(topic: string, payload: string): void {
  const parsed = JSONValue.parse(payload);
  if (parsed.isNull()) {
    reply('{"error":"invalid payload"}');
    return;
  }
  const obj = parsed.asObject();
  const text = obj.getString("text");

  // Publish to a .ts AI service via bus (the .ts service calls generateText internally)
  publish("ts.ai-service.summarize", '{"text":"' + text + '"}', "onServiceResponse");

  // Track request count
  var count: i32 = 0;
  var raw = getState("requestCount");
  if (raw.length > 0) count = I32.parseInt(raw);
  count++;
  setState("requestCount", count.toString());
  log("summary request #" + count.toString());
}

export function onServiceResponse(topic: string, payload: string): void {
  const parsed = JSONValue.parse(payload);
  if (parsed.isNull()) {
    setState("lastError", "null service response");
    return;
  }
  if (parsed.isObject()) {
    const summary = parsed.asObject().getString("summary");
    setState("lastSummary", summary);
    emit("summarize.completed", '{"summary":"' + summary + '"}');
  }
}

export function handleLast(topic: string, payload: string): void {
  var summary = getState("lastSummary");
  var count = getState("requestCount");
  reply('{"lastSummary":"' + summary + '","requestCount":' + (count.length > 0 ? count : '0') + '}');
}
