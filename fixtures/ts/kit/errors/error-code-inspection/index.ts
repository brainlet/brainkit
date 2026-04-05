import { bus, tools, output } from "kit";

const results: Record<string, string> = {};

// Test 1: Call nonexistent tool — should get error with message
try {
  await tools.call("absolutely-does-not-exist", {});
  results.toolCall = "NO_ERROR";
} catch (e: any) {
  results.toolCall = e.message || "error";
}

// Test 2: bus.publish returns replyTo
const pubResult = bus.publish("incoming.error-test", { test: true });
results.hasReplyTo = pubResult.replyTo ? "yes" : "no";

// Test 3: bus.emit doesn't throw on valid event topic
try {
  bus.emit("events.error-test", { test: true });
  results.emit = "ok";
} catch (e: any) {
  results.emit = "error: " + e.message;
}

output(results);
