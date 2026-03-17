// Test: Two threads have separate observations — Thread B doesn't see Thread A's facts.
import { agent, output, Memory, LibSQLStore } from "kit";

try {
  const url = globalThis.process?.env?.LIBSQL_URL;
  if (!url) throw new Error("LIBSQL_URL not set");

  const store = new LibSQLStore({ id: "om-isolation", url: url });

  function makeAgent(threadId) {
    const memory = new Memory({
      storage: store,
      options: {
        lastMessages: 2,
        observationalMemory: {
          model: "openai/gpt-4o-mini",
          observation: { messageTokens: 200, bufferTokens: false },
          reflection: { observationTokens: 50000 },
          scope: "thread",
        },
      },
      vector: false,
    });
    return {
      a: agent({
        model: "openai/gpt-4o-mini",
        instructions: "Answer questions concisely. If you don't know something, say 'I don't know'.",
        memory: memory,
      }),
      memOpts: { memory: { thread: { id: threadId }, resource: "isolation-user" } },
    };
  }

  const { a: agentA, memOpts: optsA } = makeAgent("thread-A");
  const { a: agentB, memOpts: optsB } = makeAgent("thread-B");

  // Thread A: plant a unique fact
  await agentA.generate("My secret code is ALPHA-7742.", optsA);
  await agentA.generate("Remember that code, it's important.", optsA);
  await agentA.generate("What is the weather like?", optsA);
  await agentA.generate("Tell me about quantum computing.", optsA);

  // Thread B: plant a different fact
  await agentB.generate("My secret code is BETA-9901.", optsB);
  await agentB.generate("Remember that code, it's important.", optsB);
  await agentB.generate("What is 10 times 10?", optsB);
  await agentB.generate("Tell me about machine learning.", optsB);

  // Ask each thread about its code
  const recallA = await agentA.generate("What is my secret code?", optsA);
  const recallB = await agentB.generate("What is my secret code?", optsB);

  output({
    recallA: recallA.text,
    recallB: recallB.text,
    aHasAlpha: recallA.text.includes("ALPHA-7742"),
    aHasBeta: recallA.text.includes("BETA-9901"),
    bHasBeta: recallB.text.includes("BETA-9901"),
    bHasAlpha: recallB.text.includes("ALPHA-7742"),
  });
} catch(e) {
  output({ error: e.message, stack: (e.stack || "").substring(0, 1000) });
}
