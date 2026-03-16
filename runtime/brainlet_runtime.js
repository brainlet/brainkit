// brainlet-runtime.js — The "brainlet" module.
// Loaded into every sandbox before user code.
// LOCAL imports wrap Mastra directly. PLATFORM imports call Go bridges.

(function() {
  "use strict";

  var embed = globalThis.__agent_embed;
  if (!embed) {
    // Not in an agent-embed sandbox — skip runtime setup
    return;
  }

  // ─── LOCAL (intra-sandbox, direct JS, no bus) ──────────────────

  // resolveModel converts "provider/model-id" to an AI SDK model instance
  // using the provider configs injected by the sandbox.
  var providerFactories = {
    openai: "createOpenAI",
    anthropic: "createAnthropic",
    google: "createGoogleGenerativeAI",
    mistral: "createMistral",
    xai: "createXai",
    groq: "createGroq",
    deepseek: "createDeepSeek",
    cerebras: "createCerebras",
    perplexity: "createPerplexity",
    togetherai: "createTogetherAI",
    fireworks: "createFireworks",
    cohere: "createCohere",
  };

  function resolveModel(modelStr) {
    if (!modelStr || typeof modelStr !== "string") return modelStr;
    var slashIdx = modelStr.indexOf("/");
    if (slashIdx < 0) return modelStr;

    var providerName = modelStr.substring(0, slashIdx);
    var modelId = modelStr.substring(slashIdx + 1);

    var providers = globalThis.__brainlet_providers || {};
    var pc = providers[providerName];
    if (!pc) return modelStr;

    var factoryName = providerFactories[providerName];
    if (!factoryName || !embed[factoryName]) return modelStr;

    var opts = { apiKey: pc.APIKey || pc.apiKey };
    if (pc.BaseURL || pc.baseURL) opts.baseURL = pc.BaseURL || pc.baseURL;
    return embed[factoryName](opts)(modelId);
  }

  // ─── Shared result wrappers ─────────────────────────────────────
  // THE brainlet response contract. Maps from Mastra/AI SDK internals to our stable shape.
  // Both agent().generate() and ai.generate() return this exact shape.
  function wrapGenerateResult(result) {
    return {
      text: result.text || "",
      object: result.object || undefined,
      reasoning: result.reasoningText || "",
      usage: {
        promptTokens: result.usage?.inputTokens || result.usage?.promptTokens || 0,
        completionTokens: result.usage?.outputTokens || result.usage?.completionTokens || 0,
        totalTokens: result.usage?.totalTokens || 0,
      },
      totalUsage: {
        promptTokens: result.totalUsage?.inputTokens || 0,
        completionTokens: result.totalUsage?.outputTokens || 0,
        totalTokens: result.totalUsage?.totalTokens || 0,
      },
      finishReason: result.finishReason || "stop",
      toolCalls: result.toolCalls || [],
      toolResults: result.toolResults || [],
      steps: result.steps || [],
      sources: result.sources || [],
      files: result.files || [],
      warnings: result.warnings || [],
      response: {
        id: result.response?.id || "",
        modelId: result.response?.modelId || "",
        timestamp: result.response?.timestamp?.toISOString?.() || "",
      },
      traceId: result.traceId || undefined,
      runId: result.runId || undefined,
      providerMetadata: result.providerMetadata || undefined,
      suspendPayload: result.suspendPayload || undefined,
    };
  }

  // Both agent().stream() and ai.stream() return this exact shape.
  // Wraps the internal stream object with stable field names.
  function wrapStreamResult(rawStream) {
    return {
      textStream: rawStream.textStream,
      fullStream: rawStream.fullStream,
      text: rawStream.text,
      object: rawStream.object,
      usage: rawStream.usage,
      totalUsage: rawStream.totalUsage,
      finishReason: rawStream.finishReason,
      reasoning: rawStream.reasoning || rawStream.reasoningText,
      toolCalls: rawStream.toolCalls,
      toolResults: rawStream.toolResults,
      steps: rawStream.steps,
      sources: rawStream.sources,
      files: rawStream.files,
      warnings: rawStream.warnings,
      response: rawStream.response,
      traceId: rawStream.traceId,
      runId: rawStream.runId,
      error: rawStream.error,
      tripwire: rawStream.tripwire,
      scoringData: rawStream.scoringData,
      providerMetadata: rawStream.providerMetadata,
    };
  }

  // Shared default store for the Kit (in-memory, persists across agent calls within this Kit)
  var _defaultStore = new embed.InMemoryStore();
  // Expose for observability span queries (internal — not part of public API)
  globalThis.__brainlet_internal_store = _defaultStore;
  globalThis.__brainlet_internal_observability = null; // set after creation

  // Observability — auto-tracing for agents, tools, workflows, LLM calls.
  // Config comes from Kit.Config.Observability (injected as globalThis.__brainkit_obs_config).
  var _obsConfig = globalThis.__brainkit_obs_config || { enabled: true, strategy: "realtime", serviceName: "brainkit" };
  var _observability = null;
  if (_obsConfig.enabled !== false) {
    try {
      if (embed.Observability && embed.DefaultExporter) {
        _observability = new embed.Observability({
          configs: {
            default: {
              serviceName: _obsConfig.serviceName || "brainkit",
              exporters: [new embed.DefaultExporter({
                storage: _defaultStore,
                strategy: _obsConfig.strategy || "realtime",
              })],
            },
          },
        });
        globalThis.__brainlet_internal_observability = _observability;
      }
    } catch(e) {
      // Observability init failed — continue without tracing
    }
  }

  // Storage shim — injected into workflows, scorers, and agents via __registerMastra().
  // Provides getStorage() for snapshot persistence and observability for auto-tracing.
  // Satisfies the Mastra interface without creating a full Mastra instance.
  var _workflowStorageShim = {
    getStorage: function() { return _defaultStore; },
    getLogger: function() { return undefined; },
    generateId: function() { return crypto.randomUUID(); },
    get observability() { return _observability; },
    // Agent-specific: methods accessed via this.#mastra?.method()
    // Must exist as functions (even no-ops) due to QuickJS optional chaining bug
    addWorkspace: function() {},
    getWorkspace: function() { return undefined; },
    getScorerById: function() { return undefined; },
    listGateways: function() { return undefined; },
  };

  // Initialize observability exporters — they need the storage shim to persist spans.
  // This replicates what Mastra.constructor does: this.#observability.setMastraContext({ mastra: this })
  if (_observability && typeof _observability.setMastraContext === "function") {
    try { _observability.setMastraContext({ mastra: _workflowStorageShim }); } catch(e) {}
  }

  // Registry of active workflow runs — needed for resume after suspend.
  // Key: runId, Value: { run, workflow }
  var _pendingRuns = {};

  // Wrap createWorkflow to inject storage for snapshot persistence and track runs.
  // Without this, workflow suspend/resume cannot persist snapshots.
  // QuickJS bug workaround: obj?.method() does NOT short-circuit when method is undefined.
  // Mastra workflows call this.#mastra?.generateId() in createRun(). Without a mastra instance,
  // QuickJS calls undefined() instead of returning undefined.
  //
  // Fix: patch the Workflow class prototype to inject our storage shim into every workflow
  // instance via __registerMastra. This catches ALL workflows — user-created, scorer internals,
  // processor workflows — regardless of how they're constructed.
  (function() {
    var probe = embed.createWorkflow({ id: "__probe", inputSchema: embed.z.any(), outputSchema: embed.z.any() });
    var WorkflowProto = Object.getPrototypeOf(probe);
    var _origCommitProto = WorkflowProto.commit;
    if (_origCommitProto) {
      WorkflowProto.commit = function() {
        // Inject storage shim before commit so createRun() has a valid #mastra
        if (typeof this.__registerMastra === "function") {
          try { this.__registerMastra(_workflowStorageShim); } catch(e) {}
        }
        return _origCommitProto.apply(this, arguments);
      };
    }
  })();

  // Patch Agent constructor to auto-inject storage shim on every new Agent.
  // This catches internally-created Agents (like observational memory's observer/reflector)
  // that bypass our agent() wrapper. Without this, the internal agents hit QuickJS's
  // optional chaining bug on this.#mastra?.generateId() etc.
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

  // wrappedCreateWorkflow — the public API that developers use.
  function wrappedCreateWorkflow(config) {
    return embed.createWorkflow(config);
  }

  // createWorkflowRun — create a run from a committed workflow, with storage + tracking.
  // This is our own API that wraps workflow.createRun() to enable suspend/resume.
  async function createWorkflowRun(workflow, opts) {
    // Inject storage shim for snapshot persistence
    if (typeof workflow.__registerMastra === "function") {
      try { workflow.__registerMastra(_workflowStorageShim); } catch(e) {}
    }
    var run = await workflow.createRun(opts);
    var origRunId = run.runId;
    _pendingRuns[origRunId] = { run: run, workflow: workflow };

    // Wrap start to track suspension
    var _origStart = run.start.bind(run);
    run.start = async function(params) {
      var result = await _origStart(params);
      if (result.status !== "suspended") {
        delete _pendingRuns[origRunId];
      }
      return result;
    };

    return run;
  }

  // resumeWorkflow — resume a suspended workflow run.
  // runId: the run's ID (from run.runId)
  // stepId: (optional) which step to resume, auto-detected if omitted
  // resumeData: the data to pass to the resumed step
  async function resumeWorkflow(runId, stepId, resumeData) {
    var entry = _pendingRuns[runId];
    if (!entry) {
      throw new Error("brainlet: no pending workflow run with id " + runId);
    }

    var resumeOpts = { resumeData: resumeData };
    if (stepId) {
      resumeOpts.step = stepId;
    }

    var result = await entry.run.resume(resumeOpts);

    // Clean up if completed
    if (result.status !== "suspended") {
      delete _pendingRuns[runId];
    }

    return result;
  }

  // createMemory() — create a Memory instance using @mastra/memory
  // Developers can pass their own storage or use the Kit's default in-memory store.
  // Supports vector store + embedder for semantic recall.
  // Supports observationalMemory for 3-tier memory compression.
  function createMemory(memoryConfig) {
    var storage = memoryConfig.storage || _defaultStore;
    var opts = {};
    if (typeof memoryConfig.lastMessages === "number") opts.lastMessages = memoryConfig.lastMessages;
    if (memoryConfig.semanticRecall !== undefined) opts.semanticRecall = memoryConfig.semanticRecall;
    if (memoryConfig.workingMemory !== undefined) opts.workingMemory = memoryConfig.workingMemory;
    if (memoryConfig.generateTitle !== undefined) opts.generateTitle = memoryConfig.generateTitle;

    // Observational memory — 3-tier compression (messages → observations → reflections)
    if (memoryConfig.observationalMemory !== undefined) {
      if (memoryConfig.observationalMemory === true) {
        // Simple mode: use defaults (google/gemini-2.5-flash, 30K/40K thresholds)
        opts.observationalMemory = true;
      } else if (typeof memoryConfig.observationalMemory === "object") {
        // Detailed config: resolve model strings to real model instances
        var omCfg = { ...memoryConfig.observationalMemory };
        if (omCfg.model && typeof omCfg.model === "string") {
          omCfg.model = resolveModel(omCfg.model);
        }
        if (omCfg.observation && typeof omCfg.observation === "object") {
          omCfg.observation = { ...omCfg.observation };
          if (omCfg.observation.model && typeof omCfg.observation.model === "string") {
            omCfg.observation.model = resolveModel(omCfg.observation.model);
          }
        }
        if (omCfg.reflection && typeof omCfg.reflection === "object") {
          omCfg.reflection = { ...omCfg.reflection };
          if (omCfg.reflection.model && typeof omCfg.reflection.model === "string") {
            omCfg.reflection.model = resolveModel(omCfg.reflection.model);
          }
        }
        opts.observationalMemory = omCfg;
      }
    }

    var memConfig = {
      storage: storage,
      options: opts,
    };

    // Vector store for semantic recall
    if (memoryConfig.vector) {
      memConfig.vector = memoryConfig.vector;
    } else {
      memConfig.vector = false;
    }

    // Embedder model for semantic recall (string like "openai/text-embedding-3-small")
    if (memoryConfig.embedder) {
      memConfig.embedder = memoryConfig.embedder;
    }

    var mem = new embed.Memory(memConfig);
    // Inject storage shim into Memory — needed for observational memory's observer/reflector
    // agents which access this.#mastra internally.
    if (typeof mem.__registerMastra === "function") {
      try { mem.__registerMastra(_workflowStorageShim); } catch(e) {}
    }
    return mem;
  }

  // agent() — create a persistent agent in THIS Kit
  function agent(config) {
    var agentOpts = {
      name: config.name || "unnamed",
      id: config.id || undefined,
      description: config.description || "",
      instructions: config.instructions || "",
      model: resolveModel(config.model),
      tools: config.tools || {},
    };

    // Processors — middleware for input/output transformation and safety guardrails
    if (config.inputProcessors) agentOpts.inputProcessors = config.inputProcessors;
    if (config.outputProcessors) agentOpts.outputProcessors = config.outputProcessors;
    if (config.maxProcessorRetries !== undefined) agentOpts.maxProcessorRetries = config.maxProcessorRetries;

    // Scorers — auto-evaluate agent responses (fire-and-forget via hook system)
    if (config.scorers) agentOpts.scorers = config.scorers;

    // Memory: accept a Memory instance directly (Mastra-style) or a config object
    var memoryOpts = null;
    if (config.memory) {
      if (config.memory instanceof embed.Memory || (config.memory.constructor && config.memory.constructor.name === "Memory")) {
        // Already a Memory instance — pass through (Mastra API)
        agentOpts.memory = config.memory;
      } else {
        // Config object — create Memory from it
        agentOpts.memory = createMemory(config.memory);
        memoryOpts = {
          thread: typeof config.memory.thread === "string" ? { id: config.memory.thread } : config.memory.thread || { id: "default" },
          resource: config.memory.resource || "default",
        };
      }
    }

    var a = new embed.Agent(agentOpts);

    // Inject storage shim (includes observability) for auto-tracing.
    // Wrapped in try-catch: if the shim is missing methods that Agent needs,
    // the agent still works (just without tracing).
    if (typeof a.__registerMastra === "function") {
      try {
        a.__registerMastra(_workflowStorageShim);
      } catch(e) {
        // Agent's __registerMastra may need more than workflows — fall back silently
      }
    }

    return {
      generate: async function(promptOrMessages, options) {
        var opts = options || {};
        if (memoryOpts && !opts.memory) {
          opts.memory = memoryOpts;
        }
        var result = await a.generate(
          typeof promptOrMessages === "string" ? promptOrMessages : promptOrMessages,
          opts
        );
        return wrapGenerateResult(result);
      },
      stream: async function(promptOrMessages, options) {
        var opts = options || {};
        if (memoryOpts && !opts.memory) {
          opts.memory = memoryOpts;
        }
        var result = await a.stream(
          typeof promptOrMessages === "string" ? promptOrMessages : promptOrMessages,
          opts
        );
        return wrapStreamResult(result);
      },
    };
  }

  // createTool() — define a tool in THIS sandbox
  // Spread passthrough: forwards ALL Mastra fields (outputSchema, suspendSchema, resumeSchema,
  // requireApproval, toModelOutput, providerOptions, lifecycle hooks, etc.)
  // Backward-compat: accepts `name` as alias for `id`, `schema` as alias for `inputSchema`.
  function createTool(config) {
    return embed.createTool({
      ...config,
      id: config.id || config.name,
      inputSchema: config.inputSchema || config.schema || embed.z.object({}),
    });
  }

  // z — Zod schemas
  var z = embed.z;

  // ─── PLATFORM (cross-sandbox, through Go bridges) ──────────────

  // Synchronous bridge request — blocks QuickJS thread. For quick ops (tools.resolve).
  function bridgeRequest(topic, payload) {
    if (typeof __go_brainkit_request === "function") {
      return __go_brainkit_request(topic, typeof payload === "string" ? payload : JSON.stringify(payload));
    }
    throw new Error("brainlet: platform bridge not available (topic: " + topic + ")");
  }

  // Async bridge request — returns Promise, frees QuickJS thread. For I/O ops (tools.call).
  function bridgeRequestAsync(topic, payload) {
    if (typeof __go_brainkit_request_async === "function") {
      return __go_brainkit_request_async(topic, typeof payload === "string" ? payload : JSON.stringify(payload));
    }
    // Fallback to sync if async not available
    return Promise.resolve(bridgeRequest(topic, payload));
  }

  // Parse bridge response, throwing if it contains an error.
  function parseBridgeResponse(raw) {
    var result = JSON.parse(raw);
    if (result && result.error) {
      throw new Error("brainlet: " + result.error);
    }
    return result;
  }

  // ai.* — LOCAL: direct LLM calls via AI SDK (generateText/streamText).
  // No Agent creation, no bus round-trip.
  var ai = {
    generate: async function(params) {
      var model = resolveModel(params.model);
      var opts = { model: model };
      if (params.prompt) opts.prompt = params.prompt;
      if (params.system) opts.system = params.system;
      if (params.messages) opts.messages = params.messages;
      var result = await embed.generateText(opts);
      return wrapGenerateResult(result);
    },
    stream: function(params) {
      var model = resolveModel(params.model);
      var opts = { model: model };
      if (params.prompt) opts.prompt = params.prompt;
      if (params.system) opts.system = params.system;
      if (params.messages) opts.messages = params.messages;
      var result = embed.streamText(opts);
      return wrapStreamResult(result);
    },
    embed: async function(params) {
      // LOCAL: uses AI SDK embed() + Mastra's ModelRouterEmbeddingModel
      var embeddingModel = new embed.ModelRouterEmbeddingModel(params.model);
      var result = await embed.embed({ model: embeddingModel, value: params.value });
      return {
        embedding: result.embedding,
        usage: {
          promptTokens: result.usage?.tokens || 0,
          completionTokens: 0,
          totalTokens: result.usage?.tokens || 0,
        },
      };
    },
    embedMany: async function(params) {
      var embeddingModel = new embed.ModelRouterEmbeddingModel(params.model);
      var result = await embed.embedMany({ model: embeddingModel, values: params.values });
      return {
        embeddings: result.embeddings,
        usage: {
          promptTokens: result.usage?.tokens || 0,
          completionTokens: 0,
          totalTokens: result.usage?.tokens || 0,
        },
      };
    },
    generateObject: async function(params) {
      var model = resolveModel(params.model);
      var opts = { model: model };
      if (params.prompt) opts.prompt = params.prompt;
      if (params.system) opts.system = params.system;
      if (params.messages) opts.messages = params.messages;
      if (params.schema) opts.schema = params.schema;
      if (params.schemaName) opts.schemaName = params.schemaName;
      if (params.schemaDescription) opts.schemaDescription = params.schemaDescription;
      if (params.mode) opts.mode = params.mode;
      if (params.output) opts.output = params.output;
      if (params.enum) opts.enum = params.enum;
      var result = await embed.generateObject(opts);
      return {
        object: result.object,
        usage: {
          promptTokens: result.usage?.promptTokens || 0,
          completionTokens: result.usage?.completionTokens || 0,
          totalTokens: result.usage?.totalTokens || 0,
        },
        finishReason: result.finishReason || "stop",
        warnings: result.warnings || [],
        response: {
          id: result.response?.id || "",
          modelId: result.response?.modelId || "",
          timestamp: result.response?.timestamp?.toISOString?.() || "",
        },
      };
    },
    streamObject: function(params) {
      var model = resolveModel(params.model);
      var opts = { model: model };
      if (params.prompt) opts.prompt = params.prompt;
      if (params.system) opts.system = params.system;
      if (params.messages) opts.messages = params.messages;
      if (params.schema) opts.schema = params.schema;
      if (params.schemaName) opts.schemaName = params.schemaName;
      if (params.schemaDescription) opts.schemaDescription = params.schemaDescription;
      if (params.mode) opts.mode = params.mode;
      if (params.output) opts.output = params.output;
      if (params.enum) opts.enum = params.enum;
      var result = embed.streamObject(opts);
      return {
        partialObjectStream: result.partialObjectStream,
        object: result.object,
        usage: result.usage,
        finishReason: result.finishReason,
        warnings: result.warnings,
        response: result.response,
      };
    },
  };

  // wasm.* — compile/run via as-embed + wazero
  var wasm = {
    compile: async function(source, opts) {
      var raw = bridgeRequest("wasm.compile", { source: source, options: opts || {} });
      return JSON.parse(raw);
    },
    run: async function(module, input) {
      var raw = bridgeRequest("wasm.run", { module: module, input: input });
      return JSON.parse(raw);
    },
    validate: async function(module) {
      var raw = bridgeRequest("wasm.validate", { module: module });
      return JSON.parse(raw);
    },
  };

  // tools.* — tool registry
  var tools = {
    call: async function(name, input) {
      // ASYNC — may hit plugin gRPC, external APIs
      var raw = await bridgeRequestAsync("tools.call", { name: name, input: input });
      return parseBridgeResponse(raw);
    },
    list: async function(namespace) {
      var raw = bridgeRequest("tools.list", { namespace: namespace || "" });
      return JSON.parse(raw);
    },
    register: async function(name, config) {
      bridgeRequest("tools.register", { name: name, description: config.description, inputSchema: config.inputSchema });
    },
  };

  // buildZodFromJsonSchema — converts JSON Schema to Zod object for Mastra tools
  function buildZodFromJsonSchema(schema) {
    if (!schema || typeof schema !== "object") return embed.z.object({});
    var props = schema.properties;
    if (!props) return embed.z.object({});

    var required = {};
    if (Array.isArray(schema.required)) {
      for (var i = 0; i < schema.required.length; i++) {
        required[schema.required[i]] = true;
      }
    }

    var shape = {};
    for (var key in props) {
      var prop = props[key];
      var typ = prop.type || "string";
      var field;
      switch (typ) {
        case "number": case "integer": field = embed.z.number(); break;
        case "boolean": field = embed.z.boolean(); break;
        case "array": field = embed.z.array(embed.z.any()); break;
        case "object": field = buildZodFromJsonSchema(prop); break;
        default: field = embed.z.string(); break;
      }
      if (prop.description) field = field.describe(prop.description);
      if (!required[key]) field = field.optional();
      shape[key] = field;
    }
    return embed.z.object(shape);
  }

  // tool() — namespace-aware tool lookup, returns Mastra-compatible tool
  function tool(name) {
    if (typeof __go_brainkit_request !== "function") {
      throw new Error("brainlet: platform bridge not available for tool resolution");
    }
    var raw = bridgeRequest("tools.resolve", { name: name });
    var info = parseBridgeResponse(raw);

    // Build Zod schema from JSON Schema if available
    var schema = embed.z.object({});
    if (info.inputSchema) {
      try {
        var jsonSchema = typeof info.inputSchema === "string" ? JSON.parse(info.inputSchema) : info.inputSchema;
        schema = buildZodFromJsonSchema(jsonSchema);
      } catch(e) { /* fallback to empty schema */ }
    }

    var t = embed.createTool({
      id: info.shortName || name,
      description: info.description || "",
      inputSchema: schema,
      execute: async function(input) {
        return await tools.call(info.name || name, input);
      },
    });
    t._registryTool = true;
    return t;
  }

  // bus.* — platform bus
  // __bus_subs stores JS callback functions keyed by subscription ID
  globalThis.__bus_subs = {};

  var busMod = {
    send: function(topic, payload) {
      __go_brainkit_bus_send(topic, JSON.stringify(payload || null));
    },
    publish: function(topic, payload) {
      __go_brainkit_bus_send(topic, JSON.stringify(payload || null));
    },
    subscribe: function(topic, handler) {
      var subId = __go_brainkit_subscribe(topic);
      globalThis.__bus_subs[subId] = handler;
      return subId;
    },
    unsubscribe: function(subId) {
      __go_brainkit_unsubscribe(subId);
      delete globalThis.__bus_subs[subId];
    },
    request: async function(topic, payload) {
      var raw = await bridgeRequestAsync(topic, payload);
      return parseBridgeResponse(raw);
    },
  };

  // sandbox context
  var sandboxCtx = {
    id: globalThis.__brainkit_sandbox_id || "",
    namespace: globalThis.__brainkit_sandbox_namespace || "",
    callerID: globalThis.__brainkit_sandbox_callerID || "",
  };

  // output() — set the module's output value (read by Go after execution)
  function output(value) {
    globalThis.__module_result = typeof value === "string" ? value : JSON.stringify(value);
  }

  // Helper: convert a plain string to a MastraDBMessage (format used by type:'agent' scorers)
  function toMastraDBMessage(text, role) {
    return {
      id: "msg-" + Math.random().toString(36).slice(2),
      role: role || "user",
      content: {
        format: 2,
        parts: [{ type: "text", text: text }],
        content: text,
      },
      createdAt: new Date(),
    };
  }

  // Wrap a pre-built scorer factory so .run() accepts plain strings
  // Pre-built scorers use type:'agent' and expect MastraDBMessage arrays.
  // We wrap .run() to convert plain string input/output to the agent format.
  function wrapPrebuiltScorer(factoryFn) {
    return function(opts) {
      var scorer = factoryFn(opts);
      // Inject storage shim (scorer creates workflow internally)
      if (typeof scorer.__registerMastra === "function") {
        try { scorer.__registerMastra(_workflowStorageShim); } catch(e) {}
      }
      var origRun = scorer.run.bind(scorer);
      scorer.run = async function(input) {
        // If input/output are plain strings, convert to MastraDBMessage format
        var converted = {};
        if (typeof input.input === "string") {
          converted.input = {
            inputMessages: [toMastraDBMessage(input.input, "user")],
            rememberedMessages: [],
            systemMessages: [],
            taggedSystemMessages: {},
          };
        } else {
          converted.input = input.input;
        }
        if (typeof input.output === "string") {
          converted.output = [toMastraDBMessage(input.output, "assistant")];
        } else {
          converted.output = input.output;
        }
        if (input.runId) converted.runId = input.runId;
        if (input.groundTruth !== undefined) converted.groundTruth = input.groundTruth;
        return origRun(converted);
      };
      return scorer;
    };
  }

  // Wrap an LLM-based pre-built scorer factory.
  // Same as wrapPrebuiltScorer but also resolves the model string for the judge.
  function wrapLLMScorer(factoryFn) {
    return function(opts) {
      // Resolve model string for the judge
      if (opts && typeof opts.model === "string") {
        opts = { ...opts, model: resolveModel(opts.model) };
      }
      var scorer = factoryFn(opts);
      if (typeof scorer.__registerMastra === "function") {
        try { scorer.__registerMastra(_workflowStorageShim); } catch(e) {}
      }
      // Wrap .run() to convert plain strings to MastraDBMessage format
      var origRun = scorer.run.bind(scorer);
      scorer.run = async function(input) {
        var converted = {};
        if (typeof input.input === "string") {
          converted.input = {
            inputMessages: [toMastraDBMessage(input.input, "user")],
            rememberedMessages: [],
            systemMessages: [],
            taggedSystemMessages: {},
          };
        } else {
          converted.input = input.input;
        }
        if (typeof input.output === "string") {
          converted.output = [toMastraDBMessage(input.output, "assistant")];
        } else {
          converted.output = input.output;
        }
        if (input.runId) converted.runId = input.runId;
        if (input.groundTruth !== undefined) converted.groundTruth = input.groundTruth;
        return origRun(converted);
      };
      return scorer;
    };
  }

  // ─── EXPORT ────────────────────────────────────────────────────

  globalThis.__brainlet = {
    // LOCAL
    agent: agent,
    createTool: createTool,
    createWorkflow: wrappedCreateWorkflow,
    createStep: embed.createStep,
    createMemory: createMemory,
    z: z,

    // STORAGE (for custom memory configs)
    Memory: embed.Memory,
    InMemoryStore: embed.InMemoryStore,
    LibSQLStore: embed.LibSQLStore,
    UpstashStore: embed.UpstashStore,
    PostgresStore: embed.PostgresStore,
    MongoDBStore: embed.MongoDBStore,

    // VECTOR STORES (for semantic recall)
    LibSQLVector: embed.LibSQLVector,
    PgVector: embed.PgVector,
    MongoDBVector: embed.MongoDBVector,

    // AI SDK (for advanced use — most developers use ai.generate/ai.stream instead)
    generateText: embed.generateText,
    streamText: embed.streamText,
    generateObject: embed.generateObject,
    streamObject: embed.streamObject,

    // PLATFORM
    ai: ai,
    wasm: wasm,
    tools: tools,
    tool: tool,
    bus: busMod,

    // CONTEXT
    sandbox: sandboxCtx,
    RequestContext: embed.RequestContext,

    // WORKFLOWS
    createWorkflowRun: createWorkflowRun,
    resumeWorkflow: resumeWorkflow,

    // EVALS
    createScorer: function(config) {
      // Pre-resolve judge model string to a real model instance
      var resolvedConfig = config;
      if (config.judge && typeof config.judge.model === "string") {
        var resolved = resolveModel(config.judge.model);
        if (resolved !== config.judge.model) {
          resolvedConfig = { ...config, judge: { ...config.judge, model: resolved } };
        }
      }
      var scorer = embed.createScorer(resolvedConfig);
      // Inject storage shim — scorers internally create workflows that need storage
      if (typeof scorer.__registerMastra === "function") {
        try { scorer.__registerMastra(_workflowStorageShim); } catch(e) {}
      }
      return scorer;
    },
    runEvals: embed.runEvals,
    scorers: {
      // Rule-based (no LLM)
      completeness: wrapPrebuiltScorer(embed.createCompletenessScorer),
      textualDifference: wrapPrebuiltScorer(embed.createTextualDifferenceScorer),
      keywordCoverage: wrapPrebuiltScorer(embed.createKeywordCoverageScorer),
      contentSimilarity: wrapPrebuiltScorer(embed.createContentSimilarityScorer),
      tone: wrapPrebuiltScorer(embed.createToneScorer),
      // LLM-based (require judge model)
      hallucination: wrapLLMScorer(embed.createHallucinationScorer),
      faithfulness: wrapLLMScorer(embed.createFaithfulnessScorer),
      answerRelevancy: wrapLLMScorer(embed.createAnswerRelevancyScorer),
      answerSimilarity: wrapLLMScorer(embed.createAnswerSimilarityScorer),
      bias: wrapLLMScorer(embed.createBiasScorer),
      toxicity: wrapLLMScorer(embed.createToxicityScorer),
      contextPrecision: wrapLLMScorer(embed.createContextPrecisionScorer),
      contextRelevance: wrapLLMScorer(embed.createContextRelevanceScorerLLM),
      noiseSensitivity: wrapLLMScorer(embed.createNoiseSensitivityScorerLLM),
      promptAlignment: wrapLLMScorer(embed.createPromptAlignmentScorerLLM),
      toolCallAccuracy: wrapLLMScorer(embed.createToolCallAccuracyScorerLLM),
    },

    // MCP
    mcp: {
      listTools: async function(serverName) {
        var raw = bridgeRequest("mcp.listTools", { server: serverName || "" });
        return JSON.parse(raw);
      },
      callTool: async function(serverName, toolName, args) {
        var raw = await bridgeRequestAsync("mcp.callTool", {
          server: serverName,
          tool: toolName,
          args: args || {},
        });
        return parseBridgeResponse(raw);
      },
    },

    // RAG
    MDocument: embed.MDocument,
    GraphRAG: embed.GraphRAG,
    createVectorQueryTool: embed.createVectorQueryTool,
    createDocumentChunkerTool: embed.createDocumentChunkerTool,
    createGraphRAGTool: embed.createGraphRAGTool,
    rerank: embed.rerank,
    rerankWithScorer: embed.rerankWithScorer,

    // MODULE
    output: output,
  };
})();
