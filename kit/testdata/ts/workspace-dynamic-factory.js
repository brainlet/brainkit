// Test: Dynamic workspace factory — workspace resolved per generate() call via requestContext
import { agent, Workspace, LocalFilesystem, RequestContext, output } from "kit";

const basePath = globalThis.process?.env?.WORKSPACE_PATH;
if (!basePath) throw new Error("WORKSPACE_PATH not set");

const results = {};

try {
  // Create agent with dynamic workspace (function resolver)
  let factoryCalled = false;
  let factoryPath = null;

  const a = agent({
    model: "openai/gpt-4o-mini",
    instructions: "You are a helpful assistant with workspace access. List files in the workspace.",
    workspace: ({ requestContext }) => {
      factoryCalled = true;
      // Could read from requestContext to customize per-request
      factoryPath = basePath;
      return new Workspace({
        id: "dynamic-ws",
        filesystem: new LocalFilesystem({ basePath }),
      });
    },
  });

  // Generate — this should trigger the workspace factory
  const r = await a.generate("List the files you can see.", {
    maxSteps: 3,
  });

  results.factoryCalled = factoryCalled;
  results.factoryPath = factoryPath;
  results.hasResponse = r.text.length > 0;
  results.status = "ok";

} catch(e) {
  results.error = e.message;
  results.stack = (e.stack || "").substring(0, 200);
}

output(results);
