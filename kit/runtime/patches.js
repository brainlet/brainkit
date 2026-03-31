// patches.js — QuickJS workarounds and prototype patches for Mastra.
// Must load FIRST — before anything that touches Mastra classes.
// Outputs: globalThis.__kit_internal_store, __kit_internal_observability, prototype patches on Workflow/Agent

(function() {
  "use strict";

  var embed = globalThis.__agent_embed;
  if (!embed) return;

  // ─── Storage shim + Observability ────────────────────────────
  var _defaultStore = new embed.InMemoryStore();
  Object.defineProperty(globalThis, '__kit_internal_store', {
    value: _defaultStore, writable: false, enumerable: false, configurable: true
  });
  Object.defineProperty(globalThis, '__kit_internal_observability', {
    value: null, writable: true, enumerable: false, configurable: true
  });

  var _obsConfig = globalThis.__brainkit_obs_config || { enabled: true, strategy: "realtime", serviceName: "brainkit" };
  var _observability = null;
  if (_obsConfig.enabled !== false) {
    try {
      if (embed.Observability && embed.DefaultExporter) {
        _observability = new embed.Observability({
          configs: { default: {
            serviceName: _obsConfig.serviceName || "brainkit",
            exporters: [new embed.DefaultExporter({ storage: _defaultStore, strategy: _obsConfig.strategy || "realtime" })],
          }},
        });
        globalThis.__kit_internal_observability = _observability;
      }
    } catch(e) {}
  }

  var _workflowStorageShim = {
    getStorage: function() { return _defaultStore; },
    getLogger: function() { return undefined; },
    generateId: function() { return crypto.randomUUID(); },
    get observability() { return _observability; },
    addWorkspace: function() {},
    getWorkspace: function() { return undefined; },
    getScorerById: function() { return undefined; },
    listGateways: function() { return undefined; },
  };

  if (_observability && typeof _observability.setMastraContext === "function") {
    try { _observability.setMastraContext({ mastra: _workflowStorageShim }); } catch(e) {}
  }

  // ─── Prototype patches ───────────────────────────────────────
  // QuickJS bug: obj?.method() does NOT short-circuit when method is undefined.
  // Mastra workflows and agents use this.#mastra?.generateId() internally.
  // Fix: patch prototypes to inject storage shim before any method call.

  // Patch Workflow.commit
  (function() {
    var probe = embed.createWorkflow({ id: "__probe", inputSchema: embed.z.any(), outputSchema: embed.z.any() });
    var WorkflowProto = Object.getPrototypeOf(probe);
    var _origCommit = WorkflowProto.commit;
    if (_origCommit) {
      WorkflowProto.commit = function() {
        if (typeof this.__registerMastra === "function") {
          try { this.__registerMastra(_workflowStorageShim); } catch(e) {}
        }
        return _origCommit.apply(this, arguments);
      };
    }
  })();

  // Patch Agent.generate/stream
  (function() {
    var AgentProto = embed.Agent.prototype;
    var _origGenerate = AgentProto.generate;
    if (_origGenerate) {
      AgentProto.generate = function() {
        if (typeof this.__registerMastra === "function") {
          try { this.__registerMastra(_workflowStorageShim); } catch(e) {}
        }
        return _origGenerate.apply(this, arguments);
      };
    }
    var _origStream = AgentProto.stream;
    if (_origStream) {
      AgentProto.stream = function() {
        if (typeof this.__registerMastra === "function") {
          try { this.__registerMastra(_workflowStorageShim); } catch(e) {}
        }
        return _origStream.apply(this, arguments);
      };
    }
  })();
})();
