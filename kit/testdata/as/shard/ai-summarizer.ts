// Persistent shard: receives text, asks AI to summarize, stores result.
// Tests: typed AI async wrapper, callback processing AI response,
//        persistent state accumulating request count
import { setMode, on, reply, setState, getState, log, JSONValue, ai, AiGenerateMsg } from "brainkit";
import { bus } from "brainkit";

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
  const model = obj.getString("model");

  // Ask AI to generate a summary
  ai.generate(new AiGenerateMsg(model, "Summarize: " + text), "onAiResponse");

  // Track request count
  var count: i32 = 0;
  var raw = getState("requestCount");
  if (raw.length > 0) count = I32.parseInt(raw);
  count++;
  setState("requestCount", count.toString());
  log("AI summary request #" + count.toString());
}

export function onAiResponse(topic: string, payload: string): void {
  const parsed = JSONValue.parse(payload);
  if (parsed.isNull()) {
    setState("lastError", "null ai response");
    return;
  }
  if (parsed.isObject()) {
    const text = parsed.asObject().getString("text");
    setState("lastSummary", text);
    bus.sendRaw("summarize.completed", '{"summary":"' + text + '"}');
  }
}

export function handleLast(topic: string, payload: string): void {
  var summary = getState("lastSummary");
  var count = getState("requestCount");
  reply('{"lastSummary":"' + summary + '","requestCount":' + (count.length > 0 ? count : '0') + '}');
}
