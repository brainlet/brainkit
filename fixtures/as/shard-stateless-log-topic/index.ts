// Stateless shard: logs the topic it received.
// Tests: handler receives correct topic string
import { setMode, on, reply, log } from "brainkit";

export function init(): void {
  setMode("stateless");
  on("test.topic-check", "handleTopicCheck");
}

export function handleTopicCheck(topic: string, payload: string): void {
  log("topic=" + topic);
  reply('{"receivedTopic":"' + topic + '"}');
}
