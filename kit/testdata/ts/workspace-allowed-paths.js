// Test: LocalFilesystem.setAllowedPaths() runtime update
// Verifies: initial containment, setAllowedPaths expands access, containment still works for disallowed paths
import { Workspace, LocalFilesystem, output } from "kit";

const basePath = globalThis.process?.env?.WORKSPACE_PATH;
const extraPath = globalThis.process?.env?.EXTRA_PATH;
if (!basePath) throw new Error("WORKSPACE_PATH not set");
if (!extraPath) throw new Error("EXTRA_PATH not set");

const results = {};

try {
  const fs = new LocalFilesystem({ basePath, contained: true });
  const ws = new Workspace({
    id: "allowed-paths-test",
    filesystem: fs,
  });

  // 1. Can read from basePath
  try {
    const content = await fs.readFile("test.txt");
    results.readBase = "ok: " + content.substring(0, 20);
  } catch(e) {
    results.readBase = "error: " + e.message;
  }

  // 2. Cannot read from extraPath (not allowed yet)
  try {
    const content = await fs.readFile(extraPath + "/extra.txt");
    results.readExtraBefore = "ok (should have failed)";
  } catch(e) {
    results.readExtraBefore = "blocked: " + (e.message || "").substring(0, 50);
  }

  // 3. setAllowedPaths — expand access to include extraPath
  if (typeof fs.setAllowedPaths === "function") {
    fs.setAllowedPaths([basePath, extraPath]);
    results.setAllowedPaths = "ok";

    // 4. Now can read from extraPath
    try {
      const content = await fs.readFile(extraPath + "/extra.txt");
      results.readExtraAfter = "ok: " + content.substring(0, 20);
    } catch(e) {
      results.readExtraAfter = "error: " + e.message;
    }
  } else {
    results.setAllowedPaths = "method not found";
  }

  results.status = "ok";
} catch(e) {
  results.error = e.message;
}

output(results);
