// approval.js — Bus-based HITL (Human-in-the-Loop) tool approval.
// Outputs: globalThis.__kit_generateWithApproval
// Depends on: __go_brainkit_await_approval bridge function

(function() {
  "use strict";

  // generateWithApproval: thin layer over Agent.generate that routes
  // tool approval through the bus. Any surface (Go, .ts, plugin, gateway)
  // can approve or decline by replying to the approval topic.
  //
  // The agent passed in MUST be reachable through a Mastra instance (either
  // created with `new Mastra({ agents: { name: agent }, storage })` and
  // fetched via `mastra.getAgent(name)`, or otherwise attached). Without
  // that, the agentic-loop workflow cannot persist a resume snapshot, so
  // approveToolCallGenerate / declineToolCallGenerate silently return the
  // same suspended state and the agent never advances past the pending
  // tool call. See fixtures/ts/agent/hitl/bus-approval for the reference
  // wiring.
  globalThis.__kit_generateWithApproval = async function(agent, promptOrMessages, options) {
    if (!options || !options.approvalTopic) {
      throw new Error("generateWithApproval: approvalTopic is required");
    }

    var approvalTopic = options.approvalTopic;
    var timeout = options.timeout || 30000;

    // Strip brainkit-specific options, pass rest to Mastra
    var agentOptions = {};
    for (var key in options) {
      if (key !== "approvalTopic" && key !== "timeout") {
        agentOptions[key] = options[key];
      }
    }
    agentOptions.requireToolApproval = true;

    // Phase 1: agent.generate — may suspend on tool call needing approval
    var result = await agent.generate(promptOrMessages, agentOptions);

    if (result.finishReason !== "suspended" || !result.runId) {
      return result; // Not suspended — tool wasn't called or no approval needed
    }

    // Phase 2: Go bridge handles the full bus lifecycle
    var approvalPayload = JSON.stringify({
      runId: result.runId,
      toolCallId: result.suspendPayload && result.suspendPayload.toolCallId,
      toolName: result.suspendPayload && result.suspendPayload.toolName,
      args: result.suspendPayload && result.suspendPayload.args,
    });

    var responseJSON = await __go_brainkit_await_approval(approvalTopic, approvalPayload, timeout);
    var response = JSON.parse(responseJSON);

    // Phase 3: resume agent based on approval decision
    var approved = response.approved !== false;
    var resumeOpts = {
      runId: result.runId,
      toolCallId: result.suspendPayload && result.suspendPayload.toolCallId,
    };

    if (approved) {
      return await agent.approveToolCallGenerate(resumeOpts);
    } else {
      return await agent.declineToolCallGenerate(resumeOpts);
    }
  };
})();
