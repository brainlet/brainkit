// Test: LSP diagnostics — create a TS file with a type error, verify diagnostics
import { Workspace, LocalFilesystem, LocalSandbox, output } from "kit";

try {
  var results = {};
  var tmpDir = globalThis.process?.env?.TEST_TMPDIR;
  if (!tmpDir) throw new Error("TEST_TMPDIR not set");

  // Create a minimal TypeScript project
  await __go_fs_mkdir(tmpDir, true);
  await __go_fs_writeFile(tmpDir + "/tsconfig.json", JSON.stringify({
    compilerOptions: {
      target: "ES2020",
      module: "commonjs",
      strict: true,
      noEmit: true,
    },
  }));

  // Create a TS file with a deliberate type error
  var tsContent = `
const name: string = 42;  // Type error: number assigned to string
console.log(name);
`;
  await __go_fs_writeFile(tmpDir + "/app.ts", tsContent);

  // Create workspace with LSP
  var workspace = new Workspace({
    filesystem: new LocalFilesystem({ basePath: tmpDir }),
    sandbox: new LocalSandbox({ workingDirectory: tmpDir }),
    lsp: {
      diagnosticTimeout: 10000,
      initTimeout: 15000,
      binaryOverrides: {
        typescript: "typescript-language-server --stdio",
      },
    },
  });
  await workspace.init();

  results.hasLSP = !!workspace.lsp;
  results.workspaceReady = workspace.status === "ready";

  // Get diagnostics for the file with the type error
  if (workspace.lsp) {
    try {
      var diagnostics = await workspace.lsp.getDiagnostics(
        tmpDir + "/app.ts",
        tsContent
      );
      results.diagnosticsCount = diagnostics ? diagnostics.length : 0;
      results.diagnostics = diagnostics ? diagnostics.map(d => ({
        severity: d.severity,
        message: d.message,
        line: d.line,
      })) : [];
      results.hasTypeError = diagnostics?.some(d => d.message?.includes("not assignable")) || false;
    } catch(e) {
      results.diagnosticError = e ? (e.message || String(e)) : "null";
      results.diagnosticStack = (e?.stack || "").substring(0, 500);
    }
  }

  // Shutdown LSP
  try {
    await workspace.lsp?.shutdownAll();
  } catch(e) {}

  output(results);
} catch(e) {
  output({ error: e ? (e.message || String(e)) : "null", stack: (e?.stack || "").substring(0, 2000) });
}
