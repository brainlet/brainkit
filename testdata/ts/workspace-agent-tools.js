// Test: All workspace auto-tools
import { agent, Workspace, LocalFilesystem, LocalSandbox, MDocument, output } from "brainlet";

try {
  var results = {};
  var tmpDir = globalThis.process?.env?.TEST_TMPDIR;
  if (!tmpDir) throw new Error("TEST_TMPDIR not set");

  // RAG regression check
  var doc = MDocument.fromText("Hello world, this is a test for chunking.");
  var chunks = await doc.chunk({ strategy: "recursive", maxSize: 500, overlap: 50 });
  results.ragChunks = chunks.length;

  var workspace = new Workspace({
    filesystem: new LocalFilesystem({ basePath: tmpDir }),
    sandbox: new LocalSandbox({ workingDirectory: tmpDir }),
  });
  await workspace.init();

  // Pre-create files
  await __go_fs_writeFile(tmpDir + "/readme.txt", "This project is called Brainlet.\nIt is an Agent OS.\nBuilt with Go and TypeScript.");
  await __go_fs_writeFile(tmpDir + "/config.json", '{"name":"brainlet","version":"1.0"}');

  var a = agent({
    model: "openai/gpt-4o-mini",
    instructions: `You have workspace tools. Use them as requested. Paths are relative to workspace root. Be concise.`,
    workspace: workspace,
    maxSteps: 5,
  });

  // Test 1: read_file
  var r1 = await a.generate('Read "readme.txt" and tell me the project name.');
  results.read = { has: r1.text.includes("Brainlet"), tools: r1.toolCalls.length };

  // Test 2: write_file
  var r2 = await a.generate('Write a file "output.txt" with content "Hello from agent"');
  var written = await workspace.filesystem.readFile("output.txt", { encoding: "utf-8" });
  results.write = { ok: String(written).includes("Hello from agent"), tools: r2.toolCalls.length };

  // Test 3: list_files
  var r3 = await a.generate('List files in the root directory.');
  results.list = { hasReadme: r3.text.toLowerCase().includes("readme"), tools: r3.toolCalls.length };

  // Test 4: grep
  var r4 = await a.generate('Search for "Agent OS" in the workspace.');
  results.grep = { found: r4.text.toLowerCase().includes("readme"), tools: r4.toolCalls.length };

  // Test 5: execute_command
  var r5 = await a.generate('Execute the command: echo brainlet-works');
  results.exec = { has: r5.text.includes("brainlet-works"), tools: r5.toolCalls.length };

  output(results);
} catch(e) {
  output({ error: e ? (e.message || String(e)) : "null", stack: (e?.stack || "").substring(0, 2000) });
}
