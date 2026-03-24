// Persistent shard: counter that survives across handler invocations.
// Tests: setMode("persistent"), getState/setState across calls, reply with state
import { setMode, on, reply, getState, setState } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("counter.inc", "handleInc");
  on("counter.get", "handleGet");
  on("counter.reset", "handleReset");
}

export function handleInc(topic: string, payload: string): void {
  var count: i32 = 0;
  var raw = getState("count");
  if (raw.length > 0) count = I32.parseInt(raw);
  count++;
  setState("count", count.toString());
  reply('{"count":' + count.toString() + '}');
}

export function handleGet(topic: string, payload: string): void {
  var raw = getState("count");
  reply('{"count":' + (raw.length > 0 ? raw : '0') + '}');
}

export function handleReset(topic: string, payload: string): void {
  setState("count", "0");
  reply('{"count":0}');
}
