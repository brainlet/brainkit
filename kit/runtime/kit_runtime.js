// kit_runtime.js — Bootstrap for brainkit .ts services.
// Sets up: bus API, resource registry, model resolution, observability, compartments.
// AI SDK and Mastra are accessed via "ai" and "agent" modules — NOT wrapped here.

(function() {
  "use strict";

  var embed = globalThis.__agent_embed;
  if (!embed) return;

  // ─── QuickJS Workarounds ──────────────────────────────────────
  // QuickJS bug: obj?.method() does NOT short-circuit when method is undefined.
  // Mastra workflows and agents use this.#mastra?.generateId() internally.
  // Fix: patch prototypes to inject storage shim before any method call.

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

  // Patch Workflow.commit to inject storage shim
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

  // Patch Agent.generate/stream to inject storage shim
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

  // ─── Model / Provider Resolution ──────────────────────────────

  var providerFactories = {
    openai: "createOpenAI", anthropic: "createAnthropic",
    google: "createGoogleGenerativeAI", mistral: "createMistral",
    xai: "createXai", groq: "createGroq", deepseek: "createDeepSeek",
    cerebras: "createCerebras", perplexity: "createPerplexity",
    togetherai: "createTogetherAI", fireworks: "createFireworks",
    cohere: "createCohere",
  };

  function resolveModel(providerName, modelId) {
    var providers = globalThis.__kit_providers || {};
    var pc = providers[providerName];
    if (!pc) return providerName + "/" + modelId;
    var factoryName = providerFactories[providerName];
    if (!factoryName || !embed[factoryName]) return providerName + "/" + modelId;
    var opts = { apiKey: pc.APIKey || pc.apiKey };
    if (pc.BaseURL || pc.baseURL) opts.baseURL = pc.BaseURL || pc.baseURL;
    return embed[factoryName](opts)(modelId);
  }

  function resolveEmbeddingModel(providerName, modelId) {
    var providers = globalThis.__kit_providers || {};
    var pc = providers[providerName];
    if (!pc) throw new Error("embeddingModel: provider '" + providerName + "' not registered");
    var factoryName = providerFactories[providerName];
    if (!factoryName || !embed[factoryName]) throw new Error("embeddingModel: provider '" + providerName + "' not available");
    var opts = { apiKey: pc.APIKey || pc.apiKey };
    if (pc.BaseURL || pc.baseURL) opts.baseURL = pc.BaseURL || pc.baseURL;
    var prov = embed[factoryName](opts);
    if (typeof prov.embedding === "function") return prov.embedding(modelId);
    if (typeof prov.textEmbeddingModel === "function") return prov.textEmbeddingModel(modelId);
    throw new Error("embeddingModel: provider '" + providerName + "' does not support embeddings");
  }

  var _providerCache = {};
  function resolveProvider(name) {
    if (_providerCache[name]) return _providerCache[name];
    var configJSON = __go_registry_resolve("provider", name);
    if (!configJSON) throw new Error("AI provider '" + name + "' not registered");
    var parsed = JSON.parse(configJSON);
    var cfg = parsed.config || {};
    var factoryName = providerFactories[parsed.type];
    if (!factoryName || !embed[factoryName]) throw new Error("AI provider '" + parsed.type + "' not available");
    var opts = {};
    if (cfg.APIKey) opts.apiKey = cfg.APIKey;
    if (cfg.BaseURL) opts.baseURL = cfg.BaseURL;
    var instance = embed[factoryName](opts);
    _providerCache[name] = instance;
    return instance;
  }

  // ─── Storage / Vector Resolution (IIFE closure caching) ──────

  var _storageCache = {};
  function resolveStorage(name) {
    if (_storageCache[name]) return _storageCache[name];
    var configJSON = __go_registry_resolve("storage", name);
    if (!configJSON) throw new Error("storage '" + name + "' not registered");
    var parsed = JSON.parse(configJSON);
    var cfg = parsed.config || {};
    var instance;
    switch (parsed.type) {
      case "memory": instance = new embed.InMemoryStore(); break;
      case "libsql": instance = new embed.LibSQLStore({ id: name, url: cfg.URL, authToken: cfg.AuthToken }); break;
      case "postgres": instance = new embed.PostgresStore({ id: name, connectionString: cfg.ConnectionString }); break;
      case "mongodb": instance = new embed.MongoDBStore({ id: name, uri: cfg.URI, dbName: cfg.DBName }); break;
      case "upstash": instance = new embed.UpstashStore({ id: name, url: cfg.URL, token: cfg.Token }); break;
      default: throw new Error("storage type '" + parsed.type + "' not available");
    }
    _storageCache[name] = instance;
    return instance;
  }

  var _vectorStoreCache = {};
  function resolveVectorStore(name) {
    if (_vectorStoreCache[name]) return _vectorStoreCache[name];
    var configJSON = __go_registry_resolve("vectorStore", name);
    if (!configJSON) throw new Error("vector store '" + name + "' not registered");
    var parsed = JSON.parse(configJSON);
    var cfg = parsed.config || {};
    var instance;
    switch (parsed.type) {
      case "libsql": instance = new embed.LibSQLVector({ id: name, connectionUrl: cfg.URL, authToken: cfg.AuthToken }); break;
      case "pgvector": instance = new embed.PgVector({ id: name, connectionString: cfg.ConnectionString }); break;
      case "mongodb": instance = new embed.MongoDBVector({ id: name, uri: cfg.URI, dbName: cfg.DBName }); break;
      default: throw new Error("vector store type '" + parsed.type + "' not available");
    }
    _vectorStoreCache[name] = instance;
    return instance;
  }

  // ─── Resource Registry ────────────────────────────────────────

  var _currentSource = "";
  var _resourceRegistry = {
    entries: {},
    cleanups: {},
    register: function(type, id, name, ref, cleanupFn) {
      var key = type + ":" + id;
      if (this.cleanups[key]) { try { this.cleanups[key](); } catch(e) {} }
      this.entries[key] = {
        type: type, id: id, name: name || id,
        source: _currentSource || "unknown",
        createdAt: Date.now(), ref: ref,
      };
      if (typeof cleanupFn === "function") this.cleanups[key] = cleanupFn;
    },
    unregister: function(type, id) {
      var key = type + ":" + id;
      var entry = this.entries[key];
      if (entry) {
        if (this.cleanups[key]) { try { this.cleanups[key](); } catch(e) {} delete this.cleanups[key]; }
        delete this.entries[key];
        return entry;
      }
      return null;
    },
    list: function(type) {
      var result = [];
      for (var key in this.entries) {
        var entry = this.entries[key];
        if (!type || entry.type === type) {
          result.push({ type: entry.type, id: entry.id, name: entry.name, source: entry.source, createdAt: entry.createdAt });
        }
      }
      return result;
    },
    listBySource: function(source) {
      var result = [];
      for (var key in this.entries) {
        var entry = this.entries[key];
        if (entry.source === source) result.push({ type: entry.type, id: entry.id, name: entry.name, source: entry.source, createdAt: entry.createdAt });
      }
      return result;
    },
    get: function(type, id) { return this.entries[type + ":" + id] || null; },
  };
  Object.defineProperty(globalThis, '__kit_registry', {
    value: _resourceRegistry, writable: false, enumerable: false, configurable: true
  });

  // ─── Bridge Helpers ───────────────────────────────────────────

  function bridgeRequest(topic, payload) {
    if (typeof __go_brainkit_request === "function") {
      return __go_brainkit_request(topic, typeof payload === "string" ? payload : JSON.stringify(payload));
    }
    throw new Error("brainkit: platform bridge not available (topic: " + topic + ")");
  }

  function bridgeRequestAsync(topic, payload) {
    if (typeof __go_brainkit_request_async === "function") {
      return __go_brainkit_request_async(topic, typeof payload === "string" ? payload : JSON.stringify(payload));
    }
    return Promise.resolve(bridgeRequest(topic, payload));
  }

  function bridgeControl(action, payload) {
    if (typeof __go_brainkit_control === "function") {
      return __go_brainkit_control(action, typeof payload === "string" ? payload : JSON.stringify(payload));
    }
    throw new Error("brainkit: platform control bridge not available (action: " + action + ")");
  }

  function parseBridgeResponse(raw) {
    var result = JSON.parse(raw);
    if (result && result.error) throw new Error("brainkit: " + result.error);
    return result;
  }

  // ─── Bus API ──────────────────────────────────────────────────

  Object.defineProperty(globalThis, '__bus_subs', {
    value: {}, writable: false, enumerable: false, configurable: true
  });

  function wrapMsg(rawMsg) {
    return {
      payload: rawMsg.payload,
      replyTo: rawMsg.replyTo || "",
      correlationId: rawMsg.correlationId || "",
      topic: rawMsg.topic || "",
      callerId: rawMsg.callerId || "",
      reply: function(data) {
        if (this.replyTo) {
          __go_brainkit_bus_reply(this.replyTo, JSON.stringify(data), this.correlationId, true);
        }
      },
      send: function(data) {
        if (this.replyTo) {
          __go_brainkit_bus_reply(this.replyTo, JSON.stringify(data), this.correlationId, false);
        }
      },
    };
  }

  var bus = {
    publish: function(topic, data) {
      var result = __go_brainkit_bus_publish(topic, JSON.stringify(data || null));
      return JSON.parse(result);
    },
    emit: function(topic, data) {
      __go_brainkit_bus_emit(topic, JSON.stringify(data || null));
    },
    subscribe: function(topic, handler) {
      var subId = __go_brainkit_subscribe(topic);
      globalThis.__bus_subs[subId] = function(rawMsg) {
        handler(wrapMsg(rawMsg));
      };
      _resourceRegistry.register("subscription", subId, subId, null, function() {
        __go_brainkit_unsubscribe(subId);
        delete globalThis.__bus_subs[subId];
      });
      return subId;
    },
    on: function(localTopic, handler) {
      // bus.on requires a deployment namespace — set during deploy via __kitRunWithSource
      if (!globalThis.__kit_deployment_namespace) {
        throw new Error("bus.on() can only be used inside a deployed .ts file");
      }
      return bus.subscribe(globalThis.__kit_deployment_namespace + "." + localTopic, handler);
    },
    unsubscribe: function(subId) {
      __go_brainkit_unsubscribe(subId);
      delete globalThis.__bus_subs[subId];
      _resourceRegistry.unregister("subscription", subId);
    },
  };

  // ─── kit.register ─────────────────────────────────────────────

  var _validTypes = { "tool": true, "agent": true, "workflow": true, "memory": true };

  var kit = {
    register: function(type, name, ref) {
      if (!_validTypes[type]) {
        throw new Error("kit.register: invalid type '" + type + "' (must be tool, agent, workflow, or memory)");
      }
      if (!name || typeof name !== "string") {
        throw new Error("kit.register: name is required and must be a string");
      }

      // Idempotent: if already registered with same type+name, skip
      var existing = _resourceRegistry.get(type, name);
      if (existing) {
        return; // already registered — no-op
      }

      var cleanupFn = null;
      if (type === "tool") {
        try {
          bridgeControl("tools.register", { name: name, description: (ref && ref.description) || "", inputSchema: {} });
        } catch(e) {}
        cleanupFn = function() { try { bridgeControl("tools.unregister", { name: name }); } catch(e) {} };
      } else if (type === "agent") {
        try {
          bridgeControl("agents.register", { name: name, capabilities: [], model: "", kit: globalThis.__brainkit_sandbox_id || "" });
        } catch(e) {}
        cleanupFn = function() { try { bridgeControl("agents.unregister", { name: name }); } catch(e) {} };
      }
      _resourceRegistry.register(type, name, name, ref, cleanupFn);
    },
    unregister: function(type, name) {
      _resourceRegistry.unregister(type, name);
    },
    list: function(type) {
      return _resourceRegistry.list(type);
    },
    get source() { return _currentSource; },
    get namespace() { return globalThis.__brainkit_sandbox_namespace || ""; },
    get callerId() { return globalThis.__brainkit_sandbox_callerID || ""; },
  };

  // ─── Infrastructure APIs ──────────────────────────────────────

  var tools = {
    call: async function(name, input) {
      var raw = await bridgeRequestAsync("tools.call", { name: name, input: input });
      return parseBridgeResponse(raw).result;
    },
    list: function(namespace) {
      var raw = bridgeRequest("tools.list", { namespace: namespace || "" });
      return parseBridgeResponse(raw).tools || [];
    },
    resolve: function(name) {
      var raw = bridgeRequest("tools.resolve", { name: name });
      return parseBridgeResponse(raw);
    },
  };

  var fs = {
    read: async function(path) { return parseBridgeResponse(await bridgeRequestAsync("fs.read", { path: path })); },
    write: async function(path, data) { return parseBridgeResponse(await bridgeRequestAsync("fs.write", { path: path, data: data })); },
    list: async function(path, pattern) { return parseBridgeResponse(await bridgeRequestAsync("fs.list", { path: path || ".", pattern: pattern || "" })); },
    stat: async function(path) { return parseBridgeResponse(await bridgeRequestAsync("fs.stat", { path: path })); },
    delete: async function(path) { return parseBridgeResponse(await bridgeRequestAsync("fs.delete", { path: path })); },
    mkdir: async function(path) { return parseBridgeResponse(await bridgeRequestAsync("fs.mkdir", { path: path })); },
  };

  var mcp = {
    listTools: function(server) { return parseBridgeResponse(bridgeRequest("mcp.listTools", { server: server || "" })).tools || []; },
    callTool: async function(server, tool, args) { return parseBridgeResponse(await bridgeRequestAsync("mcp.callTool", { server: server, tool: tool, args: args || {} })); },
  };

  var registry = {
    has: function(category, name) { return __go_registry_has(category, name) === "true"; },
    list: function(category) { return JSON.parse(__go_registry_list(category)); },
    resolve: function(category, name) { var r = __go_registry_resolve(category, name); return r ? JSON.parse(r) : null; },
    register: function(category, name, config) { bridgeControl("registry.register", { category: category, name: name, config: config }); },
    unregister: function(category, name) { bridgeControl("registry.unregister", { category: category, name: name }); },
  };

  // ─── Output ───────────────────────────────────────────────────

  Object.defineProperty(globalThis, '__module_result', {
    value: undefined, writable: true, enumerable: false, configurable: true
  });
  function output(value) {
    globalThis.__module_result = typeof value === "string" ? value : JSON.stringify(value);
  }

  // ─── Export to globalThis.__kit ───────────────────────────────

  globalThis.__kit = {
    bus: bus,
    kit: kit,
    model: resolveModel,
    embeddingModel: resolveEmbeddingModel,
    provider: resolveProvider,
    storage: resolveStorage,
    vectorStore: resolveVectorStore,
    registry: registry,
    tools: tools,
    fs: fs,
    mcp: mcp,
    output: output,
  };

  // ─── Compartment Endowments ───────────────────────────────────

  globalThis.__kit_compartments = {};

  function __withSource(fn, source) {
    return function() {
      var prev = _currentSource;
      _currentSource = source;
      try { return fn.apply(this, arguments); }
      finally { _currentSource = prev; }
    };
  }

  globalThis.__kitRunWithSource = async function(source, fn) {
    var prev = _currentSource;
    _currentSource = source;
    var prevNs = globalThis.__kit_deployment_namespace;
    globalThis.__kit_deployment_namespace = "ts." + source.replace(/\.ts$/, "").replace(/\//g, ".");
    try { return await fn(); }
    finally {
      _currentSource = prev;
      globalThis.__kit_deployment_namespace = prevNs;
    }
  };

  var _kitObj = globalThis.__kit;

  globalThis.__kitEndowments = function(source) {
    var ns = "ts." + source.replace(/\.ts$/, "").replace(/\//g, ".");
    var ws = function(fn) { return __withSource(fn, source); };

    var scopedBus = {
      publish: _kitObj.bus.publish,
      emit: _kitObj.bus.emit,
      subscribe: ws(_kitObj.bus.subscribe),
      on: function(localTopic, handler) {
        return scopedBus.subscribe(ns + "." + localTopic, handler);
      },
      unsubscribe: _kitObj.bus.unsubscribe,
    };

    var scopedKit = {
      register: ws(_kitObj.kit.register),
      unregister: ws(_kitObj.kit.unregister),
      list: _kitObj.kit.list,
      get source() { return source; },
      get namespace() { return globalThis.__brainkit_sandbox_namespace || ""; },
      get callerId() { return globalThis.__brainkit_sandbox_callerID || ""; },
    };

    // ── Compartment Endowments ─────────────────────────────────────
    // Everything a deployed .ts file can access as a global variable.
    // SES Compartments don't inherit globals — they ONLY see what's here.
    //
    // For SES-tamed builtins (Date, Math), we use pre-lockdown captures
    // stored in globalThis.__brainkit_pre_lockdown by agent-embed-setup.js.
    // This runs BEFORE lockdown() so the real implementations are captured
    // before SES freezes them as "ambient authority".
    //
    // For bridge functions (__go_process_env, etc.), the jsbridge polyfills
    // capture references in closures before SES can remove them from globalThis.
    var endowments = {
      // brainkit infrastructure ("kit" module)
      bus: scopedBus,
      kit: scopedKit,
      model: _kitObj.model,
      embeddingModel: _kitObj.embeddingModel,
      provider: _kitObj.provider,
      storage: _kitObj.storage,
      vectorStore: _kitObj.vectorStore,
      registry: _kitObj.registry,
      tools: _kitObj.tools,
      // tool() resolver — wraps a Go-registered tool as a Mastra-compatible tool object
      // so Agent can use it: const t = tool("name"); new Agent({ tools: { name: t } })
      tool: function(name) {
        var info = _kitObj.tools.resolve(name);
        if (!info) throw new Error("tool '" + name + "' not found");
        return embed.createTool({
          id: info.shortName || name,
          description: info.description || "",
          inputSchema: info.inputSchema ? embed.z.object(info.inputSchema) : embed.z.any(),
          execute: async function(input) {
            return await _kitObj.tools.call(name, input);
          },
        });
      },
      fs: _kitObj.fs,
      mcp: _kitObj.mcp,
      output: _kitObj.output,
      // AI SDK ("ai" module) — also available as endowments for Compartment code
      generateText: embed.generateText,
      streamText: embed.streamText,
      generateObject: embed.generateObject,
      streamObject: embed.streamObject,
      embed: embed.embed,
      embedMany: embed.embedMany,
      z: embed.z,
      // Mastra ("agent" module) — also available as endowments for Compartment code
      Agent: embed.Agent,
      createTool: ws(embed.createTool),
      createWorkflow: ws(embed.createWorkflow),
      createStep: embed.createStep,
      Memory: embed.Memory,
      InMemoryStore: embed.InMemoryStore,
      LibSQLStore: embed.LibSQLStore,
      UpstashStore: embed.UpstashStore,
      PostgresStore: embed.PostgresStore,
      MongoDBStore: embed.MongoDBStore,
      LibSQLVector: embed.LibSQLVector,
      PgVector: embed.PgVector,
      MongoDBVector: embed.MongoDBVector,
      ModelRouterEmbeddingModel: embed.ModelRouterEmbeddingModel,
      RequestContext: embed.RequestContext,
      Workspace: embed.Workspace,
      LocalFilesystem: embed.LocalFilesystem,
      LocalSandbox: embed.LocalSandbox,
      MDocument: embed.MDocument,
      GraphRAG: embed.GraphRAG,
      createVectorQueryTool: embed.createVectorQueryTool,
      createDocumentChunkerTool: embed.createDocumentChunkerTool,
      createGraphRAGTool: embed.createGraphRAGTool,
      rerank: embed.rerank,
      rerankWithScorer: embed.rerankWithScorer,
      Observability: embed.Observability,
      DefaultExporter: embed.DefaultExporter,
      createScorer: embed.createScorer,
      runEvals: embed.runEvals,
      // Compiler ("compiler" module)
      compile: async function(source, opts) {
        var raw = await (typeof __go_brainkit_request_async === "function"
          ? __go_brainkit_request_async("wasm.compile", JSON.stringify({ source: source, options: opts || {} }))
          : __go_brainkit_request("wasm.compile", JSON.stringify({ source: source, options: opts || {} })));
        var result = JSON.parse(raw);
        if (result && result.error) throw new Error("compiler: " + result.error);
        result.run = async function(input) {
          var runRaw = await (typeof __go_brainkit_request_async === "function"
            ? __go_brainkit_request_async("wasm.run", JSON.stringify({ moduleId: result.moduleId, input: input || null }))
            : __go_brainkit_request("wasm.run", JSON.stringify({ moduleId: result.moduleId, input: input || null })));
          var runResult = JSON.parse(runRaw);
          if (runResult && runResult.error) throw new Error("wasm.run: " + runResult.error);
          return runResult;
        };
        return result;
      },
      // JS built-ins — per-source tagged console
      console: {
        log:   function() { __go_console_log_tagged(source, "log", Array.prototype.slice.call(arguments).map(String).join(' ')); },
        warn:  function() { __go_console_log_tagged(source, "warn", Array.prototype.slice.call(arguments).map(String).join(' ')); },
        error: function() { __go_console_log_tagged(source, "error", Array.prototype.slice.call(arguments).map(String).join(' ')); },
        info:  function() { __go_console_log_tagged(source, "info", Array.prototype.slice.call(arguments).map(String).join(' ')); },
        debug: function() { __go_console_log_tagged(source, "debug", Array.prototype.slice.call(arguments).map(String).join(' ')); },
      },
      JSON: JSON,
      Promise: globalThis.Promise,
      setTimeout: ws(globalThis.setTimeout),
      setInterval: ws(globalThis.setInterval),
      clearTimeout: globalThis.clearTimeout,
      clearInterval: globalThis.clearInterval,
      queueMicrotask: globalThis.queueMicrotask,
      // Web APIs — fetch, streams, encoding, URL, crypto
      fetch: globalThis.fetch,
      Request: globalThis.Request,
      Response: globalThis.Response,
      Headers: globalThis.Headers,
      URL: globalThis.URL,
      URLSearchParams: globalThis.URLSearchParams,
      AbortController: globalThis.AbortController,
      AbortSignal: globalThis.AbortSignal,
      TextEncoder: globalThis.TextEncoder,
      TextDecoder: globalThis.TextDecoder,
      ReadableStream: globalThis.ReadableStream,
      WritableStream: globalThis.WritableStream,
      TransformStream: globalThis.TransformStream,
      TextDecoderStream: globalThis.TextDecoderStream,
      TextEncoderStream: globalThis.TextEncoderStream,
      atob: globalThis.atob,
      btoa: globalThis.btoa,
      crypto: globalThis.crypto,
      structuredClone: globalThis.structuredClone,
      // Date — SES blocks Date.now() and new Date() in Compartments.
      // Uses pre-lockdown capture from __brainkit_pre_lockdown (set before SES runs).
      Date: (function() {
        var _pre = globalThis.__brainkit_pre_lockdown || {};
        var _realDateNow = _pre.dateNow || Date.now.bind(Date);
        var _RealDate = _pre.Date || Date;
        function BrainkitDate() {
          if (arguments.length === 0) return new _RealDate(_realDateNow());
          return new (Function.prototype.bind.apply(_RealDate, [null].concat(Array.prototype.slice.call(arguments))))();
        }
        BrainkitDate.now = _realDateNow;
        BrainkitDate.parse = _RealDate.parse;
        BrainkitDate.UTC = _RealDate.UTC;
        BrainkitDate.prototype = _RealDate.prototype;
        return BrainkitDate;
      })(),
      // Math — SES blocks Math.random() in Compartments (ambient authority).
      // Uses pre-lockdown capture from __brainkit_pre_lockdown.
      Math: (function() {
        var _pre = globalThis.__brainkit_pre_lockdown || {};
        var _realRandom = _pre.mathRandom;
        var wrapper = {};
        // Copy all Math properties (floor, ceil, abs, PI, etc.)
        var names = Object.getOwnPropertyNames(Math);
        for (var i = 0; i < names.length; i++) {
          var k = names[i];
          try {
            var v = Math[k];
            wrapper[k] = typeof v === "function" ? v : v;
          } catch(e) {}
        }
        // Override random with pre-lockdown capture
        if (_realRandom) wrapper.random = _realRandom;
        return wrapper;
      })(),
      // Node.js compat — Buffer, EventEmitter, process, child_process, fs, path
      process: globalThis.process,
      Buffer: globalThis.Buffer,
      EventEmitter: globalThis.EventEmitter,
    };
    return typeof globalThis.harden === "function" ? globalThis.harden(endowments) : endowments;
  };
})();
