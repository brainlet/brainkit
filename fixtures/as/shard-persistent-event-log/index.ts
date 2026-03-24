// Persistent shard: logs events and tracks count.
// Tests: persistent state across events, send() from handler, reply with state
import { setMode, on, reply, getState, setState, log } from "brainkit";
import { bus } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("audit.event", "handleAudit");
  on("audit.stats", "handleStats");
}

export function handleAudit(topic: string, payload: string): void {
  var count: i32 = 0;
  var raw = getState("eventCount");
  if (raw.length > 0) count = I32.parseInt(raw);
  count++;
  setState("eventCount", count.toString());
  setState("lastPayload", payload);

  // Forward a notification
  bus.sendRaw("audit.logged", '{"count":' + count.toString() + '}');
  log("audit event #" + count.toString());
}

export function handleStats(topic: string, payload: string): void {
  var count = getState("eventCount");
  var last = getState("lastPayload");
  reply('{"eventCount":' + (count.length > 0 ? count : '0') + ',"lastPayload":' + (last.length > 0 ? last : '""') + '}');
}
