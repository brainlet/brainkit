// Persistent shard: generic key-value store via state.
// Tests: dynamic key usage with getState/setState/hasState, JSON parsing
import { setMode, on, reply, getState, setState, hasState } from "brainkit";

export function init(): void {
  setMode("persistent");
  on("kv.set", "handleSet");
  on("kv.get", "handleGet");
  on("kv.has", "handleHas");
}

export function handleSet(topic: string, payload: string): void {
  // payload: {"key":"...", "value":"..."}
  const keyStart = payload.indexOf('"key":"') + 6;
  const keyEnd = payload.indexOf('"', keyStart + 1);
  const key = payload.substring(keyStart + 1, keyEnd);

  const valStart = payload.indexOf('"value":"') + 8;
  const valEnd = payload.indexOf('"', valStart + 1);
  const value = payload.substring(valStart + 1, valEnd);

  setState(key, value);
  reply('{"ok":true}');
}

export function handleGet(topic: string, payload: string): void {
  const keyStart = payload.indexOf('"key":"') + 6;
  const keyEnd = payload.indexOf('"', keyStart + 1);
  const key = payload.substring(keyStart + 1, keyEnd);

  const value = getState(key);
  reply('{"value":"' + value + '"}');
}

export function handleHas(topic: string, payload: string): void {
  const keyStart = payload.indexOf('"key":"') + 6;
  const keyEnd = payload.indexOf('"', keyStart + 1);
  const key = payload.substring(keyStart + 1, keyEnd);

  const exists = hasState(key);
  reply('{"exists":' + (exists ? 'true' : 'false') + '}');
}
