// Test: Workspace filesystem operations directly + agent
import { agent, Workspace, LocalFilesystem, output } from "brainlet";

try {
  const tmpDir = globalThis.process?.env?.TEST_TMPDIR;
  if (!tmpDir) throw new Error("TEST_TMPDIR not set");

  // Create the workspace
  const filesystem = new LocalFilesystem({ basePath: tmpDir });
  const workspace = new Workspace({ filesystem: filesystem });
  await workspace.init();

  // Pre-create a file via Go bridge
  await __go_fs_writeFile(tmpDir + "/data.txt", "The secret answer is 42.");

  // Test 1: Read the file directly via the workspace's filesystem
  var directRead = null;
  var directReadErr = null;
  try {
    directRead = await filesystem.readFile("data.txt");
  } catch(e) {
    directReadErr = e.message + " | " + (e.stack || "").substring(0, 500);
  }

  // Test 2: List files
  var files = null;
  var filesErr = null;
  try {
    files = await filesystem.readdir("/");
  } catch(e) {
    filesErr = e.message + " | " + (e.stack || "").substring(0, 500);
  }

  // Test 3: Stat
  var statResult = null;
  var statErr = null;
  try {
    statResult = await filesystem.stat("data.txt");
  } catch(e) {
    statErr = e.message + " | " + (e.stack || "").substring(0, 500);
  }

  // If direct filesystem works, test with agent
  var agentResult = null;
  var agentErr = null;
  if (directRead) {
    try {
      const a = agent({
        model: "openai/gpt-4o-mini",
        instructions: "You have filesystem tools. Use read_file to read files. Be concise.",
        workspace: workspace,
      });
      agentResult = await a.generate("Read the file data.txt and tell me the secret answer.");
    } catch(e) {
      agentErr = e.message;
    }
  }

  output({
    directRead: directRead,
    directReadErr: directReadErr,
    files: files ? files.map(f => typeof f === "string" ? f : f.name) : null,
    filesErr: filesErr,
    statResult: statResult ? { size: statResult.size, isFile: typeof statResult.isFile === "function" ? statResult.isFile() : statResult.isFile } : null,
    statErr: statErr,
    agentText: agentResult?.text,
    agentHas42: agentResult?.text?.includes("42"),
    agentToolCalls: agentResult?.toolCalls?.length,
    agentErr: agentErr,
  });
} catch(e) {
  output({ error: e.message, stack: (e.stack || "").substring(0, 2000) });
}
