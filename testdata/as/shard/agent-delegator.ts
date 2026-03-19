// Persistent shard: receives work, delegates to an agent, stores result.
// Tests: agents.request via askAsync, callback with response parsing,
//        persistent state accumulating completed task count
import { setMode, on, reply, setState, getState, log, JSONValue, JSONObject } from "brainkit";
import { bus } from "brainkit";

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

  // Ask the agent
  const agentPayload = new JSONObject()
    .setString("name", agentName)
    .setString("prompt", prompt);
  bus.askAsyncRaw("agents.request", agentPayload.toString(), "onAgentResponse");
  log("delegated to agent: " + agentName);
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

    bus.sendRaw("delegate.completed", '{"result":"' + text + '","taskNum":' + count.toString() + '}');
    log("agent task #" + count.toString() + " completed");
  }
}

export function handleResults(topic: string, payload: string): void {
  var lastResult = getState("lastResult");
  var completed = getState("completedTasks");
  reply('{"lastResult":"' + lastResult + '","completedTasks":' + (completed.length > 0 ? completed : '0') + '}');
}
