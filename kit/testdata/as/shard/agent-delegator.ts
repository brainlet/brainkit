// Persistent shard: receives work, publishes to a .ts agent service, stores result.
// Tests: bus_publish to .ts agent service, callback with response parsing,
//        persistent state accumulating completed task count.
// Note: Agent calls are now handled by .ts services, not WASM catalog commands.
import { setMode, on, reply, setState, getState, log, JSONValue, publish, emit } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("delegate.task", "handleTask");
  on("delegate.results", "handleResults");
}

export function handleTask(topic: string, payload: string): void {
  const parsed = JSONValue.parse(payload);
  if (parsed.isNull()) return;
  const obj = parsed.asObject();
  const agentName = obj.getString("agent");
  const prompt = obj.getString("prompt");

  // Publish to a .ts agent service via bus
  publish("ts.agent-service.ask", '{"agent":"' + agentName + '","prompt":"' + prompt + '"}', "onAgentResponse");
  log("delegated to agent service: " + agentName);
}

export function onAgentResponse(topic: string, payload: string): void {
  const parsed = JSONValue.parse(payload);
  if (parsed.isNull()) {
    setState("lastError", "null agent response");
    return;
  }
  if (parsed.isObject()) {
    const text = parsed.asObject().getString("text");
    setState("lastResult", text);

    var count: i32 = 0;
    var raw = getState("completedTasks");
    if (raw.length > 0) count = I32.parseInt(raw);
    count++;
    setState("completedTasks", count.toString());

    emit("delegate.completed", '{"result":"' + text + '","taskNum":' + count.toString() + '}');
    log("agent task #" + count.toString() + " completed");
  }
}

export function handleResults(topic: string, payload: string): void {
  var lastResult = getState("lastResult");
  var completed = getState("completedTasks");
  reply('{"lastResult":"' + lastResult + '","completedTasks":' + (completed.length > 0 ? completed : '0') + '}');
}
