// Test: Agent with auto-injected workspace tools + RAG chunking
import { agent, Workspace, LocalFilesystem, MDocument, output } from "brainlet";

try {
  var results = {};

  // Test 1: RAG chunking (was broken by z.function() issue)
  var doc = MDocument.fromText("Hello world, this is a longer text that should be split into multiple chunks for testing purposes.");
  var chunks = await doc.chunk({ strategy: "recursive", maxSize: 500, overlap: 50 });
  results.ragChunks = chunks.length + " chunks";

  // Test 2: Workspace auto-tools
  var tmpDir = globalThis.process?.env?.TEST_TMPDIR;
  if (tmpDir) {
    var workspace = new Workspace({
      filesystem: new LocalFilesystem({ basePath: tmpDir }),
    });
    await workspace.init();
    await __go_fs_writeFile(tmpDir + "/data.txt", "The secret answer is 42.");

    var a = agent({
      model: "openai/gpt-4o-mini",
      instructions: "Use read_file to read files. Be concise.",
      workspace: workspace,
      maxSteps: 3,
    });

    var result = await a.generate('Read "data.txt" and tell me the answer.');
    results.agentText = result.text;
    results.has42 = result.text.includes("42");
    results.toolCalls = result.toolCalls.length;
  }

  output(results);
} catch(e) {
  output({ error: e ? (e.message || String(e)) : "null", stack: (e?.stack || "").substring(0, 1000) });
}
