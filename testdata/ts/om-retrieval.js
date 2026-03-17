// Test: Observational memory retrieval — agent recalls facts from observations.
// Uses LibSQLStore because InMemoryStore has a threadId=null issue with message persistence.
import { agent, output, Memory, LibSQLStore } from "kit";

try {
  const url = globalThis.process?.env?.LIBSQL_URL;
  if (!url) throw new Error("LIBSQL_URL not set");

  const store = new LibSQLStore({ id: "om-retrieval", url: url });

  const memory = new Memory({
    storage: store,
    options: {
      lastMessages: 3,  // Only 3 recent messages — forces reliance on observations
      observationalMemory: {
        model: "openai/gpt-4o-mini",
        observation: {
          messageTokens: 500,
          bufferTokens: false,
        },
        reflection: {
          observationTokens: 50000,
        },
        scope: "thread",
      },
    },
    vector: false,
  });

  const a = agent({
    model: "openai/gpt-4o-mini",
    instructions: "You are a helpful assistant. Answer questions concisely. Use information the user has told you previously.",
    memory: memory,
  });

  const memOpts = { memory: { thread: { id: "om-retrieval-1" }, resource: "om-user-1" } };

  // Phase 1: Plant unique facts
  await a.generate("My dog's name is Biscuit and he is a golden retriever.", memOpts);
  await a.generate("I was born on March 15, 1990 in Lyon, France.", memOpts);
  await a.generate("My favorite movie is Inception directed by Christopher Nolan.", memOpts);
  await a.generate("I work at a company called NeuralForge as CTO.", memOpts);
  await a.generate("The office is located at 42 Rue de la Paix in Paris.", memOpts);

  // Phase 2: Padding messages to push facts out of lastMessages window
  await a.generate("What is 2 + 2?", memOpts);
  await a.generate("Tell me a joke.", memOpts);
  await a.generate("What color is the sky?", memOpts);

  // Phase 3: Recall — agent must retrieve from observations
  const recall1 = await a.generate("What is my dog's name and what breed is he?", memOpts);
  const recall2 = await a.generate("Where was I born and when?", memOpts);
  const recall3 = await a.generate("What company do I work for and what is my role?", memOpts);

  output({
    recall1: recall1.text,
    recall2: recall2.text,
    recall3: recall3.text,
    hasDogName: recall1.text.toLowerCase().includes("biscuit"),
    hasBirthCity: recall2.text.toLowerCase().includes("lyon"),
    hasCompany: recall3.text.toLowerCase().includes("neuralforge"),
  });
} catch(e) {
  output({ error: e.message, stack: (e.stack || "").substring(0, 1500) });
}
