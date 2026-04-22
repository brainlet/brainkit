// approval.js — Bus-based HITL (Human-in-the-Loop) tool approval.
// Outputs: globalThis.__kit_generateWithApproval
// Depends on: __go_brainkit_await_approval bridge function

(function() {
  "use strict";

  // generateWithApproval: thin layer over Agent.generate that routes
  // tool approval through the bus. Any surface (Go, .ts, plugin, gateway)
  // can approve or decline by replying to the approval topic.
  //
  // The agent MUST be reachable through a Mastra instance — Mastra's
  // resumeGenerate loads the workflow snapshot via `#mastra.getStorage()`,
  // and without that snapshot the approve/decline resume path silently
  // half-completes (execute may run, but the agent ends in a confused
  // suspended state with empty text). We fail loudly here instead of
  // shipping a half-broken HITL flow.
  globalThis.__kit_generateWithApproval = async function(agent, promptOrMessages, options) {
    if (!options || !options.approvalTopic) {
      throw new Error("generateWithApproval: approvalTopic is required");
    }
    if (!agent || typeof agent.generate !== "function") {
      throw new Error("generateWithApproval: agent is required and must expose .generate()");
    }
    if (typeof agent.getMastraInstance === "function" && !agent.getMastraInstance()) {
      throw new Error(
        "generateWithApproval: agent has no Mastra parent. " +
        "Wrap it with `new Mastra({ agents: { '" + (agent.name || "name") + "': agent }, storage: new InMemoryStore() })` " +
        "and pass `mastra.getAgent('" + (agent.name || "name") + "')` to generateWithApproval. " +
        "Without a Mastra parent the resume path cannot load the workflow snapshot."
      );
    }

    var approvalTopic = options.approvalTopic;
    var timeout = options.timeout || 30000;
    // Cap on suspend/resume cycles per call. Models that retry after a
    // decline can chain several requireApproval calls within one
    // generateWithApproval invocation (the user's writer agent in
    // /Users/davidroman/brainkit/packages/writer is exactly this shape).
    // Without a loop the function returns a half-finished suspended
    // result — agent text empty, finishReason="suspended" — which the
    // caller has no way to act on.
    var maxApprovalCycles = options.maxApprovalCycles || 8;

    // Strip brainkit-specific options, pass rest to Mastra
    var agentOptions = {};
    for (var key in options) {
      if (key !== "approvalTopic" && key !== "timeout" && key !== "maxApprovalCycles") {
        agentOptions[key] = options[key];
      }
    }
    agentOptions.requireToolApproval = true;

    // Phase 1: agent.generate — may suspend on tool call needing approval
    var result = await agent.generate(promptOrMessages, agentOptions);

    var cycles = 0;
    while (result && result.finishReason === "suspended" && result.runId) {
      if (cycles >= maxApprovalCycles) {
        throw new Error(
          "generateWithApproval: exceeded maxApprovalCycles (" + maxApprovalCycles +
          "). The agent kept suspending after each approval reply — raise the limit or " +
          "tighten the agent's instructions so it stops retrying."
        );
      }
      cycles++;

      // Phase 2: Go bridge handles the full bus lifecycle for this cycle
      var approvalPayload = JSON.stringify({
        runId: result.runId,
        toolCallId: result.suspendPayload && result.suspendPayload.toolCallId,
        toolName: result.suspendPayload && result.suspendPayload.toolName,
        args: result.suspendPayload && result.suspendPayload.args,
        cycle: cycles,
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
        result = await agent.approveToolCallGenerate(resumeOpts);
      } else {
        result = await agent.declineToolCallGenerate(resumeOpts);
      }
    }

    // Normalize toolResults to the flat shape declared in agent.d.ts.
    // Mastra's resume path returns an envelope shape with {type, from,
    // runId, payload: {toolCallId, toolName, args, result}} instead of
    // the flat {toolCallId, toolName, args, result}. Unwrap so callers
    // don't need to know about both shapes.
    if (result && Array.isArray(result.toolResults)) {
      result.toolResults = result.toolResults.map(function(tr) {
        if (tr && tr.payload && tr.payload.toolName !== undefined) {
          return {
            toolCallId: tr.payload.toolCallId || tr.toolCallId,
            toolName: tr.payload.toolName,
            args: tr.payload.args,
            result: tr.payload.result,
          };
        }
        return tr;
      });
    }
    return result;
  };
})();
