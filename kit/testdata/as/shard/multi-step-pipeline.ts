// Persistent shard: multi-step pipeline — tool call → AI analysis → store.
// Tests: chained typed async calls (tool → AI), multi-callback flow,
//        persistent state tracking pipeline stages
import { setMode, on, reply, setState, getState, log, JSONValue, tools, ToolCallMsg, ai, AiGenerateMsg } from "brainkit";
import { bus } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("pipeline.run", "handleRun");
  on("pipeline.status", "handleStatus");
}

export function handleRun(topic: string, payload: string): void {
  setState("stage", "fetching");
  log("pipeline: stage 1 — fetching data");

  // Stage 1: call a tool to fetch data
  tools.call(new ToolCallMsg("data_fetch", JSONValue.parse(payload).toString()), "onDataFetched");
}

export function onDataFetched(topic: string, payload: string): void {
  setState("stage", "analyzing");
  setState("fetchedData", payload);
  log("pipeline: stage 2 — analyzing with AI");

  // Stage 2: send fetched data to AI for analysis
  ai.generate(new AiGenerateMsg("openai/gpt-4o-mini", "Analyze this data: " + payload), "onAnalysisComplete");
}

export function onAnalysisComplete(topic: string, payload: string): void {
  setState("stage", "complete");
  log("pipeline: stage 3 — complete");

  const parsed = JSONValue.parse(payload);
  if (parsed.isObject()) {
    const analysis = parsed.asObject().getString("text");
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

  bus.sendRaw("pipeline.completed", '{"run":' + count.toString() + '}');
}

export function handleStatus(topic: string, payload: string): void {
  var stage = getState("stage");
  var runs = getState("runs");
  var analysis = getState("analysis");
  reply('{"stage":"' + stage + '","runs":' + (runs.length > 0 ? runs : '0') + ',"analysis":"' + analysis + '"}');
}
