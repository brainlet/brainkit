package messages

import "testing"

func TestBusTopics(t *testing.T) {
	tests := []struct {
		name  string
		msg   BusMessage
		topic string
	}{
		// AI
		{"AiGenerateMsg", AiGenerateMsg{}, "ai.generate"},
		{"AiStreamMsg", AiStreamMsg{}, "ai.stream"},
		{"AiEmbedMsg", AiEmbedMsg{}, "ai.embed"},
		{"AiEmbedManyMsg", AiEmbedManyMsg{}, "ai.embedMany"},
		{"AiGenerateObjectMsg", AiGenerateObjectMsg{}, "ai.generateObject"},

		// Tools
		{"ToolCallMsg", ToolCallMsg{}, "tools.call"},
		{"ToolListMsg", ToolListMsg{}, "tools.list"},
		{"ToolResolveMsg", ToolResolveMsg{}, "tools.resolve"},
		{"ToolRegisterMsg", ToolRegisterMsg{}, "tools.register"},

		// Agents
		{"AgentRequestMsg", AgentRequestMsg{}, "agents.request"},
		{"AgentStreamMsg", AgentStreamMsg{}, "agents.stream"},
		{"AgentMessageMsg", AgentMessageMsg{}, "agents.message"},
		{"AgentListMsg", AgentListMsg{}, "agents.list"},
		{"AgentDiscoverMsg", AgentDiscoverMsg{}, "agents.discover"},
		{"AgentRegisterMsg", AgentRegisterMsg{}, "agents.register"},
		{"AgentUnregisterMsg", AgentUnregisterMsg{}, "agents.unregister"},
		{"AgentGetStatusMsg", AgentGetStatusMsg{}, "agents.get-status"},
		{"AgentSetStatusMsg", AgentSetStatusMsg{}, "agents.set-status"},

		// WASM
		{"WasmCompileMsg", WasmCompileMsg{}, "wasm.compile"},
		{"WasmRunMsg", WasmRunMsg{}, "wasm.run"},
		{"WasmDeployMsg", WasmDeployMsg{}, "wasm.deploy"},
		{"WasmUndeployMsg", WasmUndeployMsg{}, "wasm.undeploy"},
		{"WasmListMsg", WasmListMsg{}, "wasm.list"},
		{"WasmGetMsg", WasmGetMsg{}, "wasm.get"},
		{"WasmRemoveMsg", WasmRemoveMsg{}, "wasm.remove"},
		{"WasmDescribeMsg", WasmDescribeMsg{}, "wasm.describe"},

		// Memory
		{"MemoryCreateThreadMsg", MemoryCreateThreadMsg{}, "memory.createThread"},
		{"MemoryGetThreadMsg", MemoryGetThreadMsg{}, "memory.getThread"},
		{"MemoryListThreadsMsg", MemoryListThreadsMsg{}, "memory.listThreads"},
		{"MemorySaveMsg", MemorySaveMsg{}, "memory.save"},
		{"MemoryRecallMsg", MemoryRecallMsg{}, "memory.recall"},
		{"MemoryDeleteThreadMsg", MemoryDeleteThreadMsg{}, "memory.deleteThread"},

		// Workflows
		{"WorkflowRunMsg", WorkflowRunMsg{}, "workflows.run"},
		{"WorkflowResumeMsg", WorkflowResumeMsg{}, "workflows.resume"},
		{"WorkflowCancelMsg", WorkflowCancelMsg{}, "workflows.cancel"},
		{"WorkflowStatusMsg", WorkflowStatusMsg{}, "workflows.status"},

		// Vectors
		{"VectorUpsertMsg", VectorUpsertMsg{}, "vectors.upsert"},
		{"VectorQueryMsg", VectorQueryMsg{}, "vectors.query"},
		{"VectorCreateIndexMsg", VectorCreateIndexMsg{}, "vectors.createIndex"},
		{"VectorDeleteIndexMsg", VectorDeleteIndexMsg{}, "vectors.deleteIndex"},
		{"VectorListIndexesMsg", VectorListIndexesMsg{}, "vectors.listIndexes"},

		// Filesystem
		{"FsReadMsg", FsReadMsg{}, "fs.read"},
		{"FsWriteMsg", FsWriteMsg{}, "fs.write"},
		{"FsListMsg", FsListMsg{}, "fs.list"},
		{"FsStatMsg", FsStatMsg{}, "fs.stat"},
		{"FsDeleteMsg", FsDeleteMsg{}, "fs.delete"},
		{"FsMkdirMsg", FsMkdirMsg{}, "fs.mkdir"},

		// MCP
		{"McpListToolsMsg", McpListToolsMsg{}, "mcp.listTools"},
		{"McpCallToolMsg", McpCallToolMsg{}, "mcp.callTool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.BusTopic(); got != tt.topic {
				t.Errorf("BusTopic() = %q, want %q", got, tt.topic)
			}
		})
	}
}
