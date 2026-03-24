// Persistent shard: handler emits event that triggers another handler on the same shard.
// Tests: intra-shard event chain, persistent state across chained handlers
import { setMode, on, reply, getState, setState, log } from "brainkit";
import { bus } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("chain.start", "onStart");
  on("chain.step2", "onStep2");
}

export function onStart(topic: string, payload: string): void {
  setState("step", "1");
  log("chain: step 1 done");
  bus.sendRaw("chain.step2", '{"from":"step1"}');
  reply('{"started":true,"step":"1"}');
}

export function onStep2(topic: string, payload: string): void {
  let prev = getState("step");
  setState("step", prev + "->2");
  log("chain: step 2 done, state=" + getState("step"));
  reply('{"chain":"' + getState("step") + '"}');
}
