// Stress test: Heavy concurrent async I/O during agent tool calls
// Tests: file I/O, exec, timers, multiple tool calls, streaming — all interleaved
import { Agent, createTool, Workspace, LocalFilesystem, LocalSandbox, z } from "agent";
import { model, output } from "kit";

try {
  var results = {};
  var tmpDir = globalThis.process?.env?.TEST_TMPDIR;
  if (!tmpDir) throw new Error("TEST_TMPDIR not set");

  // Tool 1: Heavy file I/O — write + read multiple files concurrently
  const fileIOTool = createTool({
    id: "file-io-stress",
    description: "Write and read multiple files concurrently",
    inputSchema: z.object({ count: z.number(), prefix: z.string() }),
    execute: async (input) => {
      var promises = [];
      for (var i = 0; i < input.count; i++) {
        var path = tmpDir + "/" + input.prefix + "-" + i + ".txt";
        var content = "File " + i + " content: " + "x".repeat(100);
        promises.push(
          __go_fs_writeFile(path, content).then(() => __go_fs_readFile(path, "utf8"))
        );
      }
      var results = await Promise.all(promises);
      return { filesWritten: results.length, firstFile: results[0].substring(0, 30) };
    },
  });

  // Tool 2: Exec multiple commands concurrently
  const execStressTool = createTool({
    id: "exec-stress",
    description: "Run multiple shell commands concurrently",
    inputSchema: z.object({ count: z.number() }),
    execute: async (input) => {
      var promises = [];
      for (var i = 0; i < input.count; i++) {
        promises.push(globalThis.child_process.exec("echo result-" + i));
      }
      var results = await Promise.all(promises);
      return {
        commandsRun: results.length,
        outputs: results.map(r => r.stdout.trim()),
      };
    },
  });

  // Tool 3: Mixed I/O — file + exec + timer all at once
  const mixedIOTool = createTool({
    id: "mixed-io",
    description: "Do file I/O, exec, and timer simultaneously",
    inputSchema: z.object({ label: z.string() }),
    execute: async (input) => {
      var [fileResult, execResult, timerResult] = await Promise.all([
        // File I/O
        (async () => {
          var p = tmpDir + "/mixed-" + input.label + ".txt";
          await __go_fs_writeFile(p, "mixed content for " + input.label);
          return await __go_fs_readFile(p, "utf8");
        })(),
        // Exec
        globalThis.child_process.exec("echo mixed-" + input.label),
        // Timer
        new Promise(resolve => setTimeout(() => resolve("timer-done-" + input.label), 50)),
      ]);
      return {
        file: fileResult.substring(0, 30),
        exec: execResult.stdout.trim(),
        timer: timerResult,
      };
    },
  });

  var a = new Agent({
    name: "fixture",
    model: model("openai", "gpt-4o-mini"),
    instructions: `You have stress test tools. When asked, call them with the specified parameters. Be concise.`,
    tools: {
      "file-io-stress": fileIOTool,
      "exec-stress": execStressTool,
      "mixed-io": mixedIOTool,
    },
    maxSteps: 10,
  });

  // Test 1: Heavy file I/O during generate
  var r1 = await a.generate('Call file-io-stress with count 10 and prefix "gen"');
  results.fileIO = {
    text: r1.text.substring(0, 100),
    toolCalls: r1.toolCalls.length,
  };

  // Test 2: Concurrent exec during generate
  var r2 = await a.generate('Call exec-stress with count 5');
  results.exec = {
    text: r2.text.substring(0, 100),
    toolCalls: r2.toolCalls.length,
  };

  // Test 3: Mixed I/O during generate
  var r3 = await a.generate('Call mixed-io with label "alpha"');
  results.mixed = {
    text: r3.text.substring(0, 100),
    toolCalls: r3.toolCalls.length,
  };

  // Test 4: Stream with tool call (file I/O during streaming)
  var streamResult = await a.stream('Call file-io-stress with count 5 and prefix "stream"');
  var streamText = "";
  for await (var chunk of streamResult.textStream) {
    streamText += chunk;
  }
  results.streamFileIO = {
    text: streamText.substring(0, 100),
    hasContent: streamText.length > 0,
  };

  // Test 5: Stream with mixed I/O tool
  var streamResult2 = await a.stream('Call mixed-io with label "beta"');
  var streamText2 = "";
  for await (var chunk2 of streamResult2.textStream) {
    streamText2 += chunk2;
  }
  results.streamMixed = {
    text: streamText2.substring(0, 100),
    hasContent: streamText2.length > 0,
  };

  // Test 6: Multiple sequential tool calls in one generate
  var r6 = await a.generate('First call file-io-stress with count 3 and prefix "multi1", then call exec-stress with count 3, then call mixed-io with label "gamma"');
  results.multiTool = {
    text: r6.text.substring(0, 150),
    toolCalls: r6.toolCalls.length,
    steps: r6.steps?.length,
  };

  // Test 7: Workspace auto-tools during streaming (reads + lists)
  var workspace = new Workspace({
    filesystem: new LocalFilesystem({ basePath: tmpDir }),
    sandbox: new LocalSandbox({ workingDirectory: tmpDir }),
  });
  await workspace.init();
  await __go_fs_writeFile(tmpDir + "/stress-data.txt", "stress test content here");

  var wsAgent = new Agent({
    name: "ws-fixture",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Use workspace tools. Be concise.",
    workspace: workspace,
    maxSteps: 5,
  });
  var wsStream = await wsAgent.stream('Read "stress-data.txt" and list root directory files.');
  var wsText = "";
  for await (var wc of wsStream.textStream) {
    wsText += wc;
  }
  results.workspaceStream = {
    text: wsText.substring(0, 150),
    hasContent: wsText.length > 0,
    hasStressData: wsText.includes("stress"),
  };

  // Verify files were actually created
  var files = JSON.parse(await __go_fs_readdir(tmpDir));
  results.totalFiles = files.length;

  results.success = true;
  output(results);
} catch(e) {
  output({ error: e ? (e.message || String(e)) : "null", stack: (e?.stack || "").substring(0, 2000) });
}
