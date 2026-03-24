// Test: Observational memory creates observations after threshold
import { Agent, Memory, InMemoryStore } from "agent";
import { model, output } from "kit";

try {
  const memory = new Memory({
    storage: new InMemoryStore(),
    options: {
      lastMessages: 40,
      observationalMemory: {
        model: "openai/gpt-4o-mini",
        observation: {
          messageTokens: 300,        // Very low threshold for testing
          bufferTokens: false,       // Synchronous — no async buffering
        },
        reflection: {
          observationTokens: 50000,  // High — don't trigger reflection in this test
        },
        scope: "thread",
      },
    },
  });

  const a = new Agent({
    name: "fixture",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are a helpful assistant. Keep your replies to 1-2 sentences.",
    memory: memory,
  });

  const memOpts = { memory: { thread: { id: "om-test-thread-1" }, resource: "om-test-user" } };

  // Send enough messages to exceed the 300 token threshold
  // Each message pair (user + assistant) is ~50-80 tokens
  const messages = [
    "My name is David and I live in Montreal, Canada.",
    "I work as a software engineer building an Agent OS called Brainlet.",
    "My favorite programming language is Go, but I also use TypeScript.",
    "I have a cat named Pixel who likes to sit on my keyboard.",
    "Today is a sunny day and I'm working from my home office.",
    "I enjoy hiking in the Laurentian mountains on weekends.",
    "My favorite food is poutine, especially from La Banquise.",
    "I've been coding for about 15 years, starting with PHP.",
  ];

  var lastResult;
  for (var i = 0; i < messages.length; i++) {
    lastResult = await a.generate(messages[i], memOpts);
  }

  output({
    messagesCount: messages.length,
    lastResponse: lastResult.text.substring(0, 100),
    success: true,
  });
} catch(e) {
  output({
    error: e.message,
    stack: (e.stack || "").substring(0, 1000),
  });
}
