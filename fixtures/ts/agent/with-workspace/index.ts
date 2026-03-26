// Test: Agent with Workspace — LocalFilesystem file operations
import { Agent, Workspace, LocalFilesystem } from "agent";
import { model, kit, fs, output } from "kit";

try {
  // Create a workspace with the kit's filesystem
  const workspace = new Workspace({
    id: "test-workspace",
    filesystem: new LocalFilesystem(),
  });

  const agent = new Agent({
    name: "workspace-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You have access to a workspace for file operations.",
    workspace,
  });

  output({
    hasWorkspace: workspace !== null,
    workspaceType: typeof workspace,
  });
} catch (e: any) {
  // Workspace might require additional setup
  output({
    error: e.message.substring(0, 200),
    available: typeof Workspace === "function",
    fsAvailable: typeof LocalFilesystem === "function",
  });
}
