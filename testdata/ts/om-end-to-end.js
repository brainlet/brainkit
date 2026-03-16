// Test: Long conversation where agent proves it remembers facts from the very beginning.
// This is the ultimate empirical proof that observational memory works end-to-end.
import { agent, output, Memory, LibSQLStore } from "brainlet";

try {
  const url = globalThis.process?.env?.LIBSQL_URL;
  if (!url) throw new Error("LIBSQL_URL not set");

  const store = new LibSQLStore({ id: "om-e2e", url: url });
  const memory = new Memory({
    storage: store,
    options: {
      lastMessages: 3,
      observationalMemory: {
        model: "openai/gpt-4o-mini",
        observation: { messageTokens: 300, bufferTokens: false },
        reflection: { observationTokens: 50000 },
        scope: "thread",
      },
    },
    vector: false,
  });

  const a = agent({
    model: "openai/gpt-4o-mini",
    instructions: "You are a personal assistant. Remember everything the user tells you. Answer concisely.",
    memory: memory,
  });

  const memOpts = { memory: { thread: { id: "om-e2e-1" }, resource: "om-e2e-user" } };

  // Early facts (will be observed and fall out of recent window)
  await a.generate("I'm planning a trip to Tokyo next April.", memOpts);
  await a.generate("My budget is $5000 for the trip.", memOpts);
  await a.generate("I'm allergic to shellfish, so no sushi with shrimp.", memOpts);

  // Middle conversation (padding)
  await a.generate("What's a good book about Japanese history?", memOpts);
  await a.generate("Can you explain the concept of wabi-sabi?", memOpts);
  await a.generate("What are some common Japanese phrases for tourists?", memOpts);
  await a.generate("Tell me about the Tokyo Metro system.", memOpts);
  await a.generate("What's the weather like in Tokyo in April?", memOpts);
  await a.generate("Are there any festivals in April?", memOpts);
  await a.generate("Tell me about Shibuya crossing.", memOpts);

  // Late facts (should still be in recent messages or fresh observations)
  await a.generate("I just booked my hotel at the Park Hyatt in Shinjuku.", memOpts);
  await a.generate("My flight is on April 3rd, arriving at Narita.", memOpts);

  // Recall test — asks about EARLY facts that are only in observations
  const tripRecall = await a.generate(
    "Summarize everything you know about my trip: destination, budget, allergies, hotel, and flight date.",
    memOpts
  );

  const text = tripRecall.text.toLowerCase();

  output({
    summary: tripRecall.text,
    hasTokyo: text.includes("tokyo"),
    hasBudget: text.includes("5000") || text.includes("5,000"),
    hasAllergy: text.includes("shellfish") || text.includes("shrimp"),
    hasHotel: text.includes("park hyatt") || text.includes("shinjuku"),
    hasFlight: text.includes("april 3") || text.includes("narita"),
  });
} catch(e) {
  output({ error: e.message, stack: (e.stack || "").substring(0, 1000) });
}
