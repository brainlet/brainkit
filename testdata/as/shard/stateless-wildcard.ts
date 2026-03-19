// Stateless shard subscribing to a wildcard topic pattern.
// Tests: on("prefix.*", ...) matches subtopics, handler sees actual topic
import { setMode, on, reply } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("events.*", "handleEvent");
}

export function handleEvent(topic: string, payload: string): void {
  reply('{"matchedTopic":"' + topic + '","payload":' + payload + '}');
}
