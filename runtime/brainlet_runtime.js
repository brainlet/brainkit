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
    };
  }

  // Both agent().stream() and ai.stream() return this exact shape.
  // Wraps the internal stream object with stable field names.
  function wrapStreamResult(rawStream) {
    return {
      textStream: rawStream.textStream,
      fullStream: rawStream.fullStream,
      text: rawStream.text,
      usage: rawStream.usage,
      finishReason: rawStream.finishReason,
      reasoning: rawStream.reasoning || rawStream.reasoningText,
      toolCalls: rawStream.toolCalls,
      toolResults: rawStream.toolResults,
      steps: rawStream.steps,
      sources: rawStream.sources,
      response: rawStream.response,
    };
  }

  // Shared default store for the Kit (in-memory, persists across agent calls within this Kit)
  var _defaultStore = new embed.InMemoryStore();

  // createMemory() — create a Memory instance using @mastra/memory
  // Developers can pass their own storage or use the Kit's default in-memory store.
  // Supports vector store + embedder for semantic recall.
  function createMemory(memoryConfig) {
    var storage = memoryConfig.storage || _defaultStore;
    var opts = {};
    if (typeof memoryConfig.lastMessages === "number") opts.lastMessages = memoryConfig.lastMessages;
    if (memoryConfig.semanticRecall !== undefined) opts.semanticRecall = memoryConfig.semanticRecall;
    if (memoryConfig.workingMemory !== undefined) opts.workingMemory = memoryConfig.workingMemory;
    if (memoryConfig.generateTitle !== undefined) opts.generateTitle = memoryConfig.generateTitle;

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

    return new embed.Memory(memConfig);
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
  // Field names match Mastra: id, inputSchema, description, execute
  function createTool(config) {
    return embed.createTool({
      id: config.id || config.name,
      description: config.description || "",
      inputSchema: config.inputSchema || config.schema || embed.z.object({}),
      execute: config.execute,
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

  // ─── EXPORT ────────────────────────────────────────────────────

  globalThis.__brainlet = {
    // LOCAL
    agent: agent,
    createTool: createTool,
    createWorkflow: embed.createWorkflow,
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

    // MODULE
    output: output,
  };
})();
