// Test: Workspace tool name remapping and enable/disable
// Verifies: custom tool names via tools config, disabled tools, setToolsConfig()
import { Workspace, LocalFilesystem, LocalSandbox } from "agent";
import { output } from "kit";

const basePath = globalThis.process?.env?.WORKSPACE_PATH;
if (!basePath) throw new Error("WORKSPACE_PATH not set");

const results = {};

// 1. Create workspace with tool remapping
try {
  const ws = new Workspace({
    id: "remap-test",
    filesystem: new LocalFilesystem({ basePath }),
    sandbox: new LocalSandbox({ workingDirectory: basePath }),
    tools: {
      mastra_workspace_read_file: { name: "view" },
      mastra_workspace_write_file: { name: "write", requireApproval: true },
      mastra_workspace_edit_file: { enabled: false },
    },
  });

  // Verify workspace was created
  const info = ws.getInfo();
  results.create = info ? "ok" : "no info";

  // Verify tools config
  const toolsConfig = ws.getToolsConfig();
  results.hasToolsConfig = toolsConfig ? "ok" : "null";
  results.readFileRenamed = toolsConfig?.mastra_workspace_read_file?.name || "not set";
  results.editFileDisabled = toolsConfig?.mastra_workspace_edit_file?.enabled === false ? "ok" : "not disabled";

  // 2. Test setToolsConfig — switch to "plan mode" (disable write tools)
  ws.setToolsConfig({
    mastra_workspace_read_file: { name: "view" },
    mastra_workspace_write_file: { enabled: false },
    mastra_workspace_edit_file: { enabled: false },
  });

  const planConfig = ws.getToolsConfig();
  results.planModeWrite = planConfig?.mastra_workspace_write_file?.enabled === false ? "disabled" : "enabled";
  results.planModeEdit = planConfig?.mastra_workspace_edit_file?.enabled === false ? "disabled" : "enabled";

  // 3. Switch back to "build mode" (enable everything)
  ws.setToolsConfig({
    mastra_workspace_read_file: { name: "view" },
    mastra_workspace_write_file: { name: "write" },
    mastra_workspace_edit_file: { name: "edit" },
  });

  const buildConfig = ws.getToolsConfig();
  results.buildModeWrite = buildConfig?.mastra_workspace_write_file?.enabled !== false ? "enabled" : "disabled";

} catch(e) {
  results.error = e.message;
}

output(results);
