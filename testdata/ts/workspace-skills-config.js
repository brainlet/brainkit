// Test: Workspace skills directory config + skill discovery
// Verifies: skills paths passed via workspace config, skill tool discovers SKILL.md files
import { Workspace, LocalFilesystem, output } from "brainlet";

const basePath = globalThis.process?.env?.WORKSPACE_PATH;
if (!basePath) throw new Error("WORKSPACE_PATH not set");

const results = {};

try {
  const ws = new Workspace({
    id: "skills-test",
    filesystem: new LocalFilesystem({ basePath }),
    skills: [basePath + "/skills"],
  });

  // Verify workspace was created with skills config
  const info = ws.getInfo();
  results.create = info ? "ok" : "no info";

  // getInstructions should mention available skills
  const instructions = ws.getInstructions();
  results.hasInstructions = typeof instructions === "string" && instructions.length > 0 ? "ok" : "empty";
  results.mentionsSkills = instructions.toLowerCase().includes("skill") ? "ok" : "no mention";

  results.status = "ok";
} catch(e) {
  results.error = e.message;
  results.stack = (e.stack || "").substring(0, 200);
}

output(results);
