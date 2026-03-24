// Persistent shard: multi-step pipeline — tool call → .ts AI service → store.
// Tests: chained async calls (tool → .ts service), multi-callback flow,
//        persistent state tracking pipeline stages.
import { setMode, on, reply, setState, getState, log, JSONValue, JSONObject, publish, emit } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("pipeline.run", "handleRun");
  on("pipeline.status", "handleStatus");
}

export function handleRun(topic: string, payload: string): void {
  setState("stage", "fetching");
  log("pipeline: stage 1 — fetching data");

  // Stage 1: call a tool to fetch data via bus publish
  const toolPayload = new JSONObject()
    .setString("name", "data_fetch")
    .set("input", JSONValue.parse(payload));
  publish("tools.call", toolPayload.toString(), "onDataFetched");
}

export function onDataFetched(topic: string, payload: string): void {
  setState("stage", "analyzing");
  setState("fetchedData", payload);
  log("pipeline: stage 2 — analyzing via .ts AI service");

  // Stage 2: publish to a .ts AI service for analysis
  publish("ts.ai-service.analyze", '{"data":' + payload + '}', "onAnalysisComplete");
}

export function onAnalysisComplete(topic: string, payload: string): void {
  setState("stage", "complete");
  log("pipeline: stage 3 — complete");

  const parsed = JSONValue.parse(payload);
  if (parsed.isObject()) {
    const analysis = parsed.asObject().getString("analysis");
    setState("analysis", analysis);
  } else {
    setState("analysis", payload);
  }

  // Increment run counter
  var count: i32 = 0;
  var raw = getState("runs");
  if (raw.length > 0) count = I32.parseInt(raw);
  count++;
  setState("runs", count.toString());

  emit("pipeline.completed", '{"run":' + count.toString() + '}');
}

export function handleStatus(topic: string, payload: string): void {
  var stage = getState("stage");
  var runs = getState("runs");
  var analysis = getState("analysis");
  reply('{"stage":"' + stage + '","runs":' + (runs.length > 0 ? runs : '0') + ',"analysis":"' + analysis + '"}');
}
