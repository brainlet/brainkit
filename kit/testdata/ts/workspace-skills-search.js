// Test: Workspace skills (SKILL.md) + BM25 search
import { Agent, Workspace, LocalFilesystem } from "agent";
import { model, output } from "kit";

try {
  var results = {};
  var tmpDir = globalThis.process?.env?.TEST_TMPDIR;
  if (!tmpDir) throw new Error("TEST_TMPDIR not set");

  // Create a skills directory — skill name must match directory name
  await __go_fs_mkdir(tmpDir + "/skills/code-review", true);
  await __go_fs_writeFile(tmpDir + "/skills/code-review/SKILL.md", `---
name: code-review
description: Review code for quality and bugs
category: development
---

# Code Review Skill

When reviewing code, follow these steps:

1. Check for obvious bugs and logic errors
2. Verify error handling is complete
3. Look for performance issues
4. Ensure naming conventions are followed
5. Check test coverage

## Style Guide
- Use descriptive variable names
- Keep functions under 50 lines
- Add comments for non-obvious logic
`);

  // Create some content files for search
  await __go_fs_mkdir(tmpDir + "/docs", true);
  await __go_fs_writeFile(tmpDir + "/docs/guide.md", "# Getting Started\n\nBrainlet is an Agent OS for deploying AI teams.\n\n## Installation\n\nRun: go install brainlet\n");
  await __go_fs_writeFile(tmpDir + "/docs/api.md", "# API Reference\n\nThe API provides endpoints for agent management, tool registration, and workflow execution.\n");

  // Create workspace with skills + BM25 search
  var workspace = new Workspace({
    filesystem: new LocalFilesystem({ basePath: tmpDir }),
    skills: ["skills"],
    bm25: true,
    autoIndexPaths: ["docs"],
  });
  await workspace.init();

  // Test 1: Agent with skills — should have skill, skill_read, skill_search tools
  var a = new Agent({
    name: "fixture",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You have workspace tools including skill tools. Use them as requested. Be concise.",
    workspace: workspace,
    maxSteps: 5,
  });

  // Test skill discovery
  var r1 = await a.generate("List available skills using the skill tool.");
  results.skillList = { text: r1.text.substring(0, 200), hasCodeReview: r1.text.toLowerCase().includes("code-review") || r1.text.toLowerCase().includes("code review"), tools: r1.toolCalls.length };

  // Test search
  var r2 = await a.generate('Search for "Agent OS" in the workspace.');
  results.search = { text: r2.text.substring(0, 200), hasResult: r2.text.toLowerCase().includes("agent os") || r2.text.toLowerCase().includes("brainlet"), tools: r2.toolCalls.length };

  output(results);
} catch(e) {
  output({ error: e ? (e.message || String(e)) : "null", stack: (e?.stack || "").substring(0, 2000) });
}
