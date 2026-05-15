// kit_runtime.js — Export and Compartment endowments.
// All APIs read from globalThis (set by: patches.js, bridges.js, resolve.js, bus.js, infrastructure.js, approval.js).
// This file assembles globalThis.__kit and defines __kitEndowments + __kitRunWithSource.

(function() {
  "use strict";

  var embed = globalThis.__agent_embed;
  if (!embed) return;

  // Read extracted APIs from globalThis
  var bus = globalThis.__kit_bus;
  var kit = globalThis.__kit_kitAPI;
  var tools = globalThis.__kit_tools;
  var fs = globalThis.fs;
  var mcp = globalThis.__kit_mcp;
  var registry = globalThis.__kit_registry_api;
  var secretsAPI = globalThis.__kit_secrets;
  var output = globalThis.__kit_output;
  var generateWithApproval = globalThis.__kit_generateWithApproval;

  function harnessErrorMessage(error) {
    if (error && typeof error.message === "string") return error.message;
    return String(error);
  }

  function sanitizeHarnessValue(value, depth) {
    if (depth > 8) return "[MaxDepth]";
    if (value instanceof Error) {
      return { name: value.name || "Error", message: value.message || "", stack: value.stack || "" };
    }
    if (Array.isArray(value)) {
      return value.map(function(item) { return sanitizeHarnessValue(item, depth + 1); });
    }
    if (value && typeof value === "object") {
      var out = {};
      for (var key in value) {
        out[key] = sanitizeHarnessValue(value[key], depth + 1);
      }
      return out;
    }
    if (typeof value === "bigint") return String(value);
    return value;
  }

  function sanitizeHarnessEvent(event) {
    var out = sanitizeHarnessValue(event || { type: "error", error: "empty harness event" }, 0);
    if (out && typeof out.error === "object" && typeof out.error.message === "string") {
      out.error = out.error.message;
    }
    return out;
  }

  function resolveHarnessModel(modelId) {
    if (!modelId || typeof modelId !== "string") return undefined;
    var slash = modelId.indexOf("/");
    if (slash < 0) return globalThis.__kit_resolveModel("openai", modelId);
    return globalThis.__kit_resolveModel(modelId.slice(0, slash), modelId.slice(slash + 1));
  }

  var browserAPI = {
    launch: async function(req) {
      var resp = await bus.call("browser.session.launch", req || {}, { timeoutMs: 60000 });
      return resp && resp.session;
    },
    close: async function(idOrSession) {
      var id = typeof idOrSession === "string" ? idOrSession : (idOrSession && idOrSession.id);
      var resp = await bus.call("browser.session.close", { id: id || "" }, { timeoutMs: 30000 });
      return !!(resp && resp.closed);
    },
    list: async function() {
      var resp = await bus.call("browser.session.list", {}, { timeoutMs: 10000 });
      return (resp && resp.sessions) || [];
    },
  };

  function buildHarnessTool(toolName) {
    var info = tools.resolve(toolName);
    if (!info) {
      throw new Error("harness.createHarness: tool '" + toolName + "' is not registered");
    }
    var parsedSchema = null;
    if (info.inputSchema) {
      if (typeof info.inputSchema === "string") {
        try { parsedSchema = JSON.parse(info.inputSchema); } catch(e) { parsedSchema = null; }
      } else if (typeof info.inputSchema === "object") {
        parsedSchema = info.inputSchema;
      }
    }
    return embed.createTool({
      id: info.shortName || toolName,
      description: info.description || "",
      inputSchema: parsedSchema || embed.z.any(),
      execute: async function(input) {
        var args = (input && input.context !== undefined) ? input.context : input;
        return await tools.call(toolName, args);
      },
    });
  }

  function normalizeHarnessOMConfig(omConfig) {
    if (!omConfig || typeof omConfig !== "object") return undefined;
    var out = {};
    var observerModel = omConfig.defaultObserverModelId || omConfig.defaultObserverModel;
    var reflectorModel = omConfig.defaultReflectorModelId || omConfig.defaultReflectorModel;
    var observationThreshold = omConfig.defaultObservationThreshold !== undefined
      ? omConfig.defaultObservationThreshold
      : omConfig.observationThreshold;
    var reflectionThreshold = omConfig.defaultReflectionThreshold !== undefined
      ? omConfig.defaultReflectionThreshold
      : omConfig.reflectionThreshold;
    if (observerModel !== undefined && observerModel !== "") out.defaultObserverModelId = observerModel;
    if (reflectorModel !== undefined && reflectorModel !== "") out.defaultReflectorModelId = reflectorModel;
    if (observationThreshold !== undefined && observationThreshold !== 0) out.defaultObservationThreshold = observationThreshold;
    if (reflectionThreshold !== undefined && reflectionThreshold !== 0) out.defaultReflectionThreshold = reflectionThreshold;
    return out;
  }

  async function destroyHarnessFromGo() {
    var unsubscribe = globalThis.__brainkit_harness_unsubscribe;
    globalThis.__brainkit_harness_unsubscribe = undefined;
    if (typeof unsubscribe === "function") {
      try { unsubscribe(); } catch(e) {}
    }
    var harness = globalThis.__brainkit_harness;
    if (harness && typeof harness.destroy === "function") {
      await harness.destroy();
    }
    globalThis.__brainkit_harness = undefined;
    return { ok: true };
  }

  async function createHarnessFromGo(rawConfig) {
    if (!embed.Harness) {
      throw new Error("harness.createHarness: Harness export is unavailable");
    }
    var cfg = typeof rawConfig === "string" ? JSON.parse(rawConfig) : (rawConfig || {});
    var registryRefs = globalThis.__kit_registry;
    var storage = cfg.storage || new embed.InMemoryStore();
    var memory = cfg.memory || new embed.Memory({ storage: storage, options: { lastMessages: 10 } });
    var modes = (cfg.modes || []).map(function(mode) {
      var next = {};
      for (var key in mode) next[key] = mode[key];
      if (!next.agent && next.agentName) {
        var entry = registryRefs && registryRefs.get("agent", next.agentName);
        next.agent = entry && entry.ref;
        if (!next.agent) {
          throw new Error("harness.createHarness: agent '" + next.agentName + "' is not registered");
        }
      }
      if (!next.agent) {
        throw new Error("harness.createHarness: mode '" + (next.id || "?") + "' has no agent");
      }
      delete next.agentName;
      return next;
    });
    if (modes.length === 0) {
      throw new Error("harness.createHarness: at least one mode is required");
    }

    var harnessConfig = {};
    for (var cfgKey in cfg) harnessConfig[cfgKey] = cfg[cfgKey];
    harnessConfig.storage = storage;
    harnessConfig.memory = memory;
    harnessConfig.modes = modes;

    if (Array.isArray(cfg.toolNames) && !harnessConfig.tools) {
      var resolvedTools = {};
      cfg.toolNames.forEach(function(toolName) {
        resolvedTools[toolName] = buildHarnessTool(toolName);
      });
      harnessConfig.tools = resolvedTools;
    }
    delete harnessConfig.toolNames;

    if (Array.isArray(cfg.subagents)) {
      harnessConfig.subagents = cfg.subagents.map(function(subagent) {
        var next = {};
        for (var key in subagent) next[key] = subagent[key];
        if (!next.name) next.name = next.id;
        if (!next.description) next.description = String(next.instructions || next.id || "Harness subagent");
        if (!next.allowedHarnessTools && next.allowedTools) next.allowedHarnessTools = next.allowedTools;
        delete next.allowedTools;
        return next;
      });
    }

    var omConfig = normalizeHarnessOMConfig(cfg.omConfig);
    if (omConfig) harnessConfig.omConfig = omConfig;
    if (!harnessConfig.resolveModel) harnessConfig.resolveModel = resolveHarnessModel;

    if (!harnessConfig.threadLock &&
        typeof globalThis.__go_harness_lock_acquire === "function" &&
        typeof globalThis.__go_harness_lock_release === "function") {
      harnessConfig.threadLock = {
        acquire: async function(threadId) {
          var err = globalThis.__go_harness_lock_acquire(String(threadId || ""));
          if (err) throw new Error(String(err));
        },
        release: async function(threadId) {
          var err = globalThis.__go_harness_lock_release(String(threadId || ""));
          if (err) throw new Error(String(err));
        },
      };
    }

    if (harnessConfig.toolCategories && !harnessConfig.toolCategoryResolver) {
      var categoryByTool = harnessConfig.toolCategories;
      harnessConfig.toolCategoryResolver = function(toolName) {
        return categoryByTool[toolName] || null;
      };
      delete harnessConfig.toolCategories;
    }

    if (harnessConfig.defaultPermissions || harnessConfig.alwaysAllowTools) {
      var initialState = harnessConfig.initialState || {};
      var permissionRules = initialState.permissionRules || { categories: {}, tools: {} };
      if (harnessConfig.defaultPermissions) {
        for (var category in harnessConfig.defaultPermissions) {
          permissionRules.categories[category] = harnessConfig.defaultPermissions[category];
        }
      }
      if (Array.isArray(harnessConfig.alwaysAllowTools)) {
        harnessConfig.alwaysAllowTools.forEach(function(toolName) {
          permissionRules.tools[toolName] = "allow";
        });
      }
      initialState.permissionRules = permissionRules;
      harnessConfig.initialState = initialState;
      delete harnessConfig.defaultPermissions;
      delete harnessConfig.alwaysAllowTools;
    }

    if (harnessConfig.workspace && harnessConfig.workspace.rootDir && embed.Workspace && embed.LocalFilesystem) {
      harnessConfig.workspace = new embed.Workspace({
        id: harnessConfig.workspace.id || harnessConfig.id + "-workspace",
        name: harnessConfig.workspace.name || harnessConfig.workspace.id || harnessConfig.id + " workspace",
        filesystem: new embed.LocalFilesystem({ basePath: harnessConfig.workspace.rootDir }),
      });
    }

    if (globalThis.__brainkit_harness) {
      await destroyHarnessFromGo();
    }

    var harness = new embed.Harness(harnessConfig);
    var unsubscribe = typeof harness.subscribe === "function" ? harness.subscribe(function(event) {
      var bridge = globalThis.__go_harness_event;
      if (typeof bridge !== "function") return;
      try {
        bridge(JSON.stringify(sanitizeHarnessEvent(event)));
      } catch (error) {
        bridge(JSON.stringify({ type: "error", error: "harness event serialization failed: " + harnessErrorMessage(error), fatal: false }));
      }
    }) : function() {};
    globalThis.__brainkit_harness = harness;
    globalThis.__brainkit_harness_unsubscribe = unsubscribe;
    return { ok: true, id: cfg.id || "" };
  }

  // ─── Export to globalThis.__kit ───────────────────────────────

  globalThis.__kit = {
    bus: bus,
    kit: kit,
    model: globalThis.__kit_resolveModel,
    embeddingModel: globalThis.__kit_resolveEmbeddingModel,
    provider: globalThis.__kit_resolveProvider,
    __clearProviderCache: globalThis.__kit_clearProviderCache,
    __clearRegistryCache: globalThis.__kit_clearRegistryCache,
    storage: globalThis.__kit_resolveStorage,
    vectorStore: globalThis.__kit_resolveVectorStore,
    registry: registry,
    tools: tools,
    browser: browserAPI,
    fs: fs,
    mcp: mcp,
    output: output,
    secrets: secretsAPI,
    generateWithApproval: generateWithApproval,
    createHarness: createHarnessFromGo,
    destroyHarness: destroyHarnessFromGo,
  };

  // ─── Compartment Endowments ───────────────────────────────────

  globalThis.__kit_compartments = {};

  function __withSource(fn, source) {
    return function() {
      var prev = globalThis.__kit_currentSource;
      globalThis.__kit_currentSource = source;
      try { return fn.apply(this, arguments); }
      finally { globalThis.__kit_currentSource = prev; }
    };
  }

  globalThis.__kitRunWithSource = async function(source, fn) {
    var prev = globalThis.__kit_currentSource;
    globalThis.__kit_currentSource = source;
    var prevNs = globalThis.__kit_deployment_namespace;
    globalThis.__kit_deployment_namespace = "ts." + source.replace(/\.ts$/, "").replace(/\//g, ".");
    try { return await fn(); }
    finally {
      globalThis.__kit_currentSource = prev;
      globalThis.__kit_deployment_namespace = prevNs;
    }
  };

  var _kitObj = globalThis.__kit;

  // Wrap detector processors (PromptInjectionDetector, PIIDetector) so
  // their internal detection agent uses an InMemoryStore-backed Mastra
  // parent instead of inheriting the kit's LibSQLStore via patches.js.
  // Without this, both the outer agent's agentic-loop workflow AND the
  // detection agent's workflow compete for the same SQLite write lock,
  // causing deadlock → timeout → InternalError: interrupted → fail-open.
  function _wrapDetectorProcessor(Base, detectorId) {
    if (!Base) return undefined;
    function Wrapped(opts) {
      var instance = new Base(opts);
      if (instance.detectionAgent && typeof instance.detectionAgent.getMastraInstance === "function" && !instance.detectionAgent.getMastraInstance()) {
        var agents = {};
        agents[detectorId] = instance.detectionAgent;
        var wrapper = new embed.Mastra({
          agents: agents,
          storage: new embed.InMemoryStore(),
        });
        instance.detectionAgent = wrapper.getAgent(detectorId);
      }
      return instance;
    }
    Wrapped.prototype = Base.prototype;
    Wrapped.DEFAULT_DETECTION_TYPES = Base.DEFAULT_DETECTION_TYPES;
    return Wrapped;
  }

  globalThis.__kitEndowments = function(source) {
    var ns = "ts." + source.replace(/\.ts$/, "").replace(/\//g, ".");
    var ws = function(fn) { return __withSource(fn, source); };
    var _reg = globalThis.__kit_registry;
    var _BKE = globalThis.BrainkitError;

    var scopedBus = {
      publish: _kitObj.bus.publish,
      emit: _kitObj.bus.emit,
      subscribe: ws(_kitObj.bus.subscribe),
      on: function(localTopic, handler) {
        var fullTopic = ns + "." + localTopic;
        var existing = _reg.get("topic", fullTopic);
        if (existing) {
          throw new _BKE(
            "topic '" + localTopic + "' already subscribed in this package",
            "TOPIC_COLLISION"
          );
        }
        // Pass source explicitly — the closure var `source` is the deployment
        // source, while __kit_currentSource may be an internal eval filename.
        _reg.register("topic", fullTopic, localTopic, null, null, source);
        return scopedBus.subscribe(fullTopic, handler);
      },
      unsubscribe: _kitObj.bus.unsubscribe,
      sendTo: _kitObj.bus.sendTo,
      call: _kitObj.bus.call,
      callStream: _kitObj.bus.callStream,
      callService: _kitObj.bus.callService,
      callServiceStream: _kitObj.bus.callServiceStream,
      callTo: _kitObj.bus.callTo,
      callToStream: _kitObj.bus.callToStream,
      schedule: ws(function(expression, topic, data) {
        return _kitObj.bus.schedule(expression, ns + "." + topic, data);
      }),
      unschedule: _kitObj.bus.unschedule,
      onCancel: _kitObj.bus.onCancel,
      withCancelController: _kitObj.bus.withCancelController,
    };

    var scopedKit = {
      register: ws(_kitObj.kit.register),
      unregister: ws(_kitObj.kit.unregister),
      list: _kitObj.kit.list,
      get source() { return source; },
      get namespace() { return globalThis.__brainkit_sandbox_namespace || ""; },
      get callerId() { return globalThis.__brainkit_sandbox_callerID || ""; },
    };

    // Wrap bridge-thrown errors as BrainkitError instances inside the
    // Compartment. throwBrainkitError in bridges_util.go sets real
    // `.code`/`.details` properties on the JS error; we just promote them
    // to the Compartment-visible BrainkitError class so `instanceof`
    // checks work in user code.
    function rewrapErrors(fn) {
      return function() {
        try { return fn.apply(this, arguments); }
        catch(e) {
          if (e && e.code && !(e instanceof _BKE)) {
            throw new _BKE(e.message, e.code, e.details || {});
          }
          throw e;
        }
      };
    }
    function rewrapErrorsAsync(fn) {
      return async function() {
        try { return await fn.apply(this, arguments); }
        catch(e) {
          if (e && e.code && !(e instanceof _BKE)) {
            throw new _BKE(e.message, e.code, e.details || {});
          }
          throw e;
        }
      };
    }

    var endowments = {
      // Error class — must be in endowments so Compartment code can catch with instanceof
      BrainkitError: _BKE,
      Error: globalThis.Error,
      // brainkit infrastructure ("kit" module)
      bus: {
        publish: rewrapErrors(scopedBus.publish),
        emit: rewrapErrors(scopedBus.emit),
        subscribe: scopedBus.subscribe,
        on: scopedBus.on,
        unsubscribe: scopedBus.unsubscribe,
        sendTo: rewrapErrors(scopedBus.sendTo),
        call: rewrapErrorsAsync(scopedBus.call),
        callStream: rewrapErrorsAsync(scopedBus.callStream),
        callService: rewrapErrorsAsync(scopedBus.callService),
        callServiceStream: rewrapErrorsAsync(scopedBus.callServiceStream),
        callTo: rewrapErrorsAsync(scopedBus.callTo),
        callToStream: rewrapErrorsAsync(scopedBus.callToStream),
        schedule: rewrapErrors(scopedBus.schedule),
        unschedule: scopedBus.unschedule,
        onCancel: scopedBus.onCancel,
        withCancelController: scopedBus.withCancelController,
      },
      kit: scopedKit,
      model: _kitObj.model,
      embeddingModel: _kitObj.embeddingModel,
      provider: _kitObj.provider,
      storage: _kitObj.storage,
      vectorStore: _kitObj.vectorStore,
      registry: _kitObj.registry,
      tools: {
        call: rewrapErrorsAsync(_kitObj.tools.call),
        list: rewrapErrors(_kitObj.tools.list),
        resolve: rewrapErrors(_kitObj.tools.resolve),
      },
      browser: {
        launch: rewrapErrorsAsync(_kitObj.browser.launch),
        close: rewrapErrorsAsync(_kitObj.browser.close),
        list: rewrapErrorsAsync(_kitObj.browser.list),
      },
      // Unified `tool` endowment — single identifier serving both surfaces.
      //
      // Because free identifiers in the deployed .ts resolve through the
      // Compartment's endowments (imports are stripped at transpile time),
      // the `kit` module's `tool(name: string)` and the AI SDK's
      // `tool(definition: object)` share a name. Discriminate by argument
      // shape:
      //
      //   • string → kit surface: resolve a Go-registered tool by name and
      //     wrap it in `embed.createTool(...)` so Agents / generateText
      //     receive a real Mastra Tool instance (passes isMastraTool,
      //     avoids the `'parameters' in tool` TypeError in
      //     ensureToolProperties → isVercelTool).
      //   • anything else → AI SDK surface: identity-ish pass-through via
      //     the bundle's `embed.tool(config)` helper, which is typed to
      //     aid AI SDK tool-config inference.
      //
      // Without this discrimination, the literal `tool: embed.tool`
      // endowment below would silently shadow the kit resolver and
      // `tool("multiply")` would just return the string "multiply",
      // producing a non-object tool reference that ensureToolProperties
      // then crashed on inside `prepare-tools-step`.
      tool: function(nameOrDefinition) {
        if (typeof nameOrDefinition === "string") {
          var name = nameOrDefinition;
          var info = _kitObj.tools.resolve(name);
          if (!info) throw new Error("tool '" + name + "' not found");
          // info.inputSchema arrives as a JSON-schema STRING (from Go's
          // json.RawMessage serialization). Parse it into a proper schema
          // object, then convert via z.toJSONSchema's inverse — Mastra's
          // createTool accepts a raw JSON-schema Record<string, unknown>
          // or a ZodType, so the parsed object is the safe path. Without
          // this, the LLM only sees an empty schema and calls the tool
          // with no arguments.
          var parsedSchema = null;
          if (info.inputSchema) {
            if (typeof info.inputSchema === "string") {
              try { parsedSchema = JSON.parse(info.inputSchema); } catch(e) { parsedSchema = null; }
            } else if (typeof info.inputSchema === "object") {
              parsedSchema = info.inputSchema;
            }
          }
          return embed.createTool({
            id: info.shortName || name,
            description: info.description || "",
            inputSchema: parsedSchema || embed.z.any(),
            execute: async function(input) {
              // Mastra v1 passes { context: <args>, runtimeContext } in v5
              // execute surface. Unwrap if present so the Go bridge gets
              // the raw user args.
              var args = (input && input.context !== undefined) ? input.context : input;
              return await _kitObj.tools.call(name, args);
            },
          });
        }
        // AI SDK authoring surface: `tool({ description, inputSchema, execute, ... })`.
        // `embed.tool` in ai-sdk v6 is the identity function used purely for
        // compile-time inference. Fall back to identity if the bundle's
        // helper is missing.
        return typeof embed.tool === "function" ? embed.tool(nameOrDefinition) : nameOrDefinition;
      },
      fs: globalThis.fs,
      mcp: _kitObj.mcp,
      output: _kitObj.output,
      secrets: {
        get: rewrapErrors(_kitObj.secrets.get),
      },
      // Embedded LLM-reference corpus. Deployments that build
      // architect-style agents use reference.get(name) to pull a
      // pack or raw doc and drop it into a system prompt, so the
      // underlying LLM writes code against the actual brainkit
      // surface instead of guessing. Lists every available name
      // + size via reference.list().
      reference: {
        get: async function(name) {
          if (!name || typeof name !== "string") {
            throw new BrainkitError("reference.get: name is required", "VALIDATION_ERROR", { field: "name" });
          }
          var resp = await globalThis.__kit_bus.call("kit.reference", { name: name }, { timeoutMs: 5000 });
          return (resp && resp.content) || "";
        },
        list: async function() {
          var resp = await globalThis.__kit_bus.call("kit.reference.list", null, { timeoutMs: 5000 });
          return (resp && resp.references) || [];
        },
      },
      generateWithApproval: _kitObj.generateWithApproval,
      // AI SDK
      generateText: embed.generateText,
      streamText: embed.streamText,
      generateObject: embed.generateObject,
      streamObject: embed.streamObject,
      embed: embed.embed,
      embedMany: embed.embedMany,
      z: embed.z,
      // Mastra
      Agent: embed.Agent,
      createTool: ws(embed.createTool),
      createWorkflow: ws(embed.createWorkflow),
      createStep: embed.createStep,
      Memory: embed.Memory,
      InMemoryStore: embed.InMemoryStore,
      LibSQLStore: function(opts) {
        if (opts && opts.url && /^file:/i.test(opts.url)) {
          throw new _BKE("file: URLs not supported — use storage('name') to access configured backends", "VALIDATION_ERROR");
        }
        return new embed.LibSQLStore(opts);
      },
      UpstashStore: embed.UpstashStore,
      PostgresStore: embed.PostgresStore,
      MongoDBStore: embed.MongoDBStore,
      LibSQLVector: function(opts) {
        if (opts && opts.connectionUrl && /^file:/i.test(opts.connectionUrl)) {
          throw new _BKE("file: URLs not supported — use vectorStore('name') to access configured backends", "VALIDATION_ERROR");
        }
        return new embed.LibSQLVector(opts);
      },
      PgVector: embed.PgVector,
      MongoDBVector: embed.MongoDBVector,
      PineconeVector: embed.PineconeVector,
      ChromaVector: embed.ChromaVector,
      QdrantVector: embed.QdrantVector,
      ModelRouterEmbeddingModel: embed.ModelRouterEmbeddingModel,
      RequestContext: embed.RequestContext,
      RuntimeContext: embed.RuntimeContext || embed.RequestContext,
      Workspace: embed.Workspace,
      LocalFilesystem: embed.LocalFilesystem,
      LocalSandbox: embed.LocalSandbox,
      WORKSPACE_TOOLS_PREFIX: embed.WORKSPACE_TOOLS_PREFIX,
      WORKSPACE_TOOLS: embed.WORKSPACE_TOOLS,
      // Voice — OpenAIVoice handles whisper-1 (STT) + tts-1 (TTS);
      // CompositeVoice routes different providers for STT vs TTS.
      //
      // Patch: @mastra/voice-openai@0.11 calls `this.traced(fn, name)()`
      // in speak/listen, but MastraVoice doesn't provide `traced`.
      // The Mastra build pipeline wraps this via a decorator at
      // publish time in their managed deployment; the raw
      // published package leaves the call unresolved, so
      // speak/listen crash with `TypeError: not a function`.
      // Polyfill `traced` to an identity wrapper —
      // `traced(fn, name) => fn`. brainkit's own tracing module
      // still produces spans; we just lose the voice-specific
      // span names that Mastra's decorator would have added.
      OpenAIVoice: (function() {
        const Base = embed.OpenAIVoice;
        if (Base && Base.prototype && typeof Base.prototype.traced !== "function") {
          Base.prototype.traced = function(fn, _name) { return fn; };
        }
        return Base;
      })(),
      CompositeVoice: embed.CompositeVoice,
      MastraVoice: embed.MastraVoice,
      // Mastra's realtime voice surface — wraps the OpenAI
      // Realtime WebSocket API. Needs a globalThis WebSocket
      // polyfill OR a Go-side bridge; the pre-flight smoke
      // test in session 12 determines which.
      OpenAIRealtimeVoice: embed.OpenAIRealtimeVoice,
      // Additional voice providers. Each extends MastraVoice
      // with its own .speak / .listen; identical polyfill
      // story — fetch for request/response providers,
      // globalThis.WebSocket for realtime ones
      // (GeminiLiveVoice).
      AzureVoice: embed.AzureVoice,
      ElevenLabsVoice: embed.ElevenLabsVoice,
      // GoogleVoice (non-realtime) needs grpc-over-http2;
      // not bundled. Gemini Live (below) covers Google.
      CloudflareVoice: embed.CloudflareVoice,
      DeepgramVoice: embed.DeepgramVoice,
      PlayAIVoice: embed.PlayAIVoice,
      PLAYAI_VOICES: embed.PLAYAI_VOICES,
      SpeechifyVoice: embed.SpeechifyVoice,
      SarvamVoice: embed.SarvamVoice,
      MurfVoice: embed.MurfVoice,
      // GeminiLiveVoice removed pending SES compatibility
      // investigation (see entry.mjs comment).
      MDocument: embed.MDocument,
      GraphRAG: embed.GraphRAG,
      createVectorQueryTool: embed.createVectorQueryTool,
      createDocumentChunkerTool: embed.createDocumentChunkerTool,
      createGraphRAGTool: embed.createGraphRAGTool,
      rerank: embed.rerank,
      rerankWithScorer: embed.rerankWithScorer,
      Observability: embed.Observability,
      DefaultExporter: embed.DefaultExporter,
      SensitiveDataFilter: embed.SensitiveDataFilter,
      // OpenTelemetry sdk-trace-base primitives. Needed so deployed
      // fixtures can wire BasicTracerProvider + InMemorySpanExporter
      // + BatchSpanProcessor / SimpleSpanProcessor end-to-end.
      BatchSpanProcessor: embed.BatchSpanProcessor,
      SimpleSpanProcessor: embed.SimpleSpanProcessor,
      NoopSpanProcessor: embed.NoopSpanProcessor,
      ConsoleSpanExporter: embed.ConsoleSpanExporter,
      InMemorySpanExporter: embed.InMemorySpanExporter,
      BasicTracerProvider: embed.BasicTracerProvider,
      AlwaysOnSampler: embed.AlwaysOnSampler,
      AlwaysOffSampler: embed.AlwaysOffSampler,
      ParentBasedSampler: embed.ParentBasedSampler,
      TraceIdRatioBasedSampler: embed.TraceIdRatioBasedSampler,
      createScorer: embed.createScorer,
      runEvals: embed.runEvals,
      // Prebuilt scorer factories (`@mastra/evals/scorers/prebuilt`).
      // Available inside a deployed .ts as plain globals, e.g.
      // `createAnswerRelevancyScorer({ model: model("openai", ...) })`.
      createCompletenessScorer: embed.createCompletenessScorer,
      createTextualDifferenceScorer: embed.createTextualDifferenceScorer,
      createKeywordCoverageScorer: embed.createKeywordCoverageScorer,
      createContentSimilarityScorer: embed.createContentSimilarityScorer,
      createToneScorer: embed.createToneScorer,
      createAnswerRelevancyScorer: embed.createAnswerRelevancyScorer,
      createAnswerSimilarityScorer: embed.createAnswerSimilarityScorer,
      createFaithfulnessScorer: embed.createFaithfulnessScorer,
      createHallucinationScorer: embed.createHallucinationScorer,
      createBiasScorer: embed.createBiasScorer,
      createToxicityScorer: embed.createToxicityScorer,
      createContextPrecisionScorer: embed.createContextPrecisionScorer,
      createContextRelevanceScorerLLM: embed.createContextRelevanceScorerLLM,
      createNoiseSensitivityScorerLLM: embed.createNoiseSensitivityScorerLLM,
      createPromptAlignmentScorerLLM: embed.createPromptAlignmentScorerLLM,
      createToolCallAccuracyScorerLLM: embed.createToolCallAccuracyScorerLLM,
      // Processors — built-in input/output middleware for Agents.
      // Used inside an Agent config as `inputProcessors` /
      // `outputProcessors` for safety and shaping.
      ModerationProcessor: embed.ModerationProcessor,
      PromptInjectionDetector: _wrapDetectorProcessor(embed.PromptInjectionDetector, "prompt-injection-detector"),
      PIIDetector: _wrapDetectorProcessor(embed.PIIDetector, "pii-detector"),
      SystemPromptScrubber: embed.SystemPromptScrubber,
      UnicodeNormalizer: embed.UnicodeNormalizer,
      LanguageDetector: embed.LanguageDetector,
      TokenLimiterProcessor: embed.TokenLimiterProcessor,
      BatchPartsProcessor: embed.BatchPartsProcessor,
      StructuredOutputProcessor: embed.StructuredOutputProcessor,
      ToolCallFilter: embed.ToolCallFilter,
      ToolSearchProcessor: embed.ToolSearchProcessor,
      AgentsMDInjector: embed.AgentsMDInjector,
      SkillsProcessor: embed.SkillsProcessor,
      SkillSearchProcessor: embed.SkillSearchProcessor,
      WorkspaceInstructionsProcessor: embed.WorkspaceInstructionsProcessor,
      ResponseCache: embed.ResponseCache,
      DEFAULT_RESPONSE_CACHE_TTL_SECONDS: embed.DEFAULT_RESPONSE_CACHE_TTL_SECONDS,
      RESPONSE_CACHE_CONTEXT_KEY: embed.RESPONSE_CACHE_CONTEXT_KEY,
      buildResponseCacheKey: embed.buildResponseCacheKey,
      InMemoryServerCache: embed.InMemoryServerCache,
      MastraServerCache: embed.MastraServerCache,
      // JS built-ins
      console: {
        log:   function() { __go_console_log_tagged(source, "log", __util_format(Array.prototype.slice.call(arguments))); },
        warn:  function() { __go_console_log_tagged(source, "warn", __util_format(Array.prototype.slice.call(arguments))); },
        error: function() { __go_console_log_tagged(source, "error", __util_format(Array.prototype.slice.call(arguments))); },
        info:  function() { __go_console_log_tagged(source, "info", __util_format(Array.prototype.slice.call(arguments))); },
        debug: function() { __go_console_log_tagged(source, "debug", __util_format(Array.prototype.slice.call(arguments))); },
      },
      JSON: JSON,
      Promise: globalThis.Promise,
      setTimeout: ws(globalThis.setTimeout),
      setInterval: ws(globalThis.setInterval),
      clearTimeout: globalThis.clearTimeout,
      clearInterval: globalThis.clearInterval,
      queueMicrotask: globalThis.queueMicrotask,
      setImmediate: globalThis.setImmediate,
      clearImmediate: globalThis.clearImmediate,
      // Web APIs
      fetch: globalThis.fetch,
      Request: globalThis.Request,
      Response: globalThis.Response,
      Headers: globalThis.Headers,
      FormData: globalThis.FormData,
      Blob: globalThis.Blob,
      File: globalThis.File,
      Audio: globalThis.Audio,
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
      Intl: globalThis.Intl,
      WebSocket: globalThis.WebSocket,
      structuredClone: globalThis.structuredClone,
      // Date — SES tamed
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
      // Math — SES tamed
      Math: (function() {
        var _pre = globalThis.__brainkit_pre_lockdown || {};
        var _realRandom = _pre.mathRandom;
        var wrapper = {};
        var names = Object.getOwnPropertyNames(Math);
        for (var i = 0; i < names.length; i++) {
          var k = names[i];
          try { var v = Math[k]; wrapper[k] = typeof v === "function" ? v : v; } catch(e) {}
        }
        if (_realRandom) wrapper.random = _realRandom;
        return wrapper;
      })(),
      // Node.js compat
      GoSocket: globalThis.GoSocket,
      process: globalThis.process,
      Buffer: globalThis.Buffer,
      EventEmitter: globalThis.EventEmitter,
      stream: globalThis.stream,
      Readable: globalThis.stream && globalThis.stream.Readable,
      Writable: globalThis.stream && globalThis.stream.Writable,
      Duplex: globalThis.stream && globalThis.stream.Duplex,
      Transform: globalThis.stream && globalThis.stream.Transform,
      PassThrough: globalThis.stream && globalThis.stream.PassThrough,
      net: globalThis.net,
      tls: globalThis.tls,
      os: globalThis.os,
      path: globalThis.path,
      dns: globalThis.dns,
      zlib: globalThis.zlib,
      child_process: globalThis.child_process,
      http: globalThis.http,
      https: globalThis.https,
      node_url: globalThis.node_url,
      node_module: globalThis.node_module,
      require: globalThis.node_module && globalThis.node_module.createRequire("brainkit:package"),
      util: globalThis.util,
      utilTypes: globalThis.utilTypes,
      assert: globalThis.assert,
      querystring: globalThis.querystring,
      StringDecoder: globalThis.StringDecoder,
      perf_hooks: globalThis.perf_hooks,
      timersPromises: globalThis.timersPromises,
      async_hooks: globalThis.async_hooks,
      diagnostics_channel: globalThis.diagnostics_channel,
      worker_threads: globalThis.worker_threads,

      // ── Gap 12: Mastra + AI SDK surface completion ───────────────
      // These landed after the initial endowment pass to close surface
      // gaps flagged in the brainkit-maps/brainkit/plans-03 audit.

      // Mastra core — agent internals
      Mastra: embed.Mastra,
      TripWire: embed.TripWire,
      MessageList: embed.MessageList,
      convertMessages: embed.convertMessages,
      TypeDetector: embed.TypeDetector,

      // Mastra core — workflow class + helpers
      Workflow: embed.Workflow,
      cloneWorkflow: embed.cloneWorkflow,
      cloneStep: embed.cloneStep,
      mapVariable: embed.mapVariable,

      // Mastra core — background tasks
      BackgroundTaskManager: embed.BackgroundTaskManager,
      createBackgroundTask: embed.createBackgroundTask,
      generateBackgroundTaskSystemPrompt: embed.generateBackgroundTaskSystemPrompt,
      __brainkitMastraBackgroundTaskDebug: embed.__brainkitMastraBackgroundTaskDebug,

      // Mastra core — Harness
      Harness: embed.Harness,
      assignTaskIds: embed.assignTaskIds,
      askUserTool: embed.askUserTool,
      defaultDisplayState: embed.defaultDisplayState,
      defaultOMProgressState: embed.defaultOMProgressState,
      parseSubagentMeta: embed.parseSubagentMeta,
      submitPlanTool: embed.submitPlanTool,
      taskWriteTool: embed.taskWriteTool,
      taskUpdateTool: embed.taskUpdateTool,
      taskCompleteTool: embed.taskCompleteTool,
      taskCheckTool: embed.taskCheckTool,

      // Mastra core — logger
      ConsoleLogger: embed.ConsoleLogger,
      MultiLogger: embed.MultiLogger,
      DualLogger: embed.DualLogger,

      // Mastra core — evals infrastructure
      MastraScorer: embed.MastraScorer,
      registerHook: embed.registerHook,
      executeHook: embed.executeHook,
      AvailableHooks: embed.AvailableHooks,

      // Mastra evals — trajectory + tool-call-accuracy code variants
      createTrajectoryAccuracyScorerLLM: embed.createTrajectoryAccuracyScorerLLM,
      createToolCallAccuracyScorerCode: embed.createToolCallAccuracyScorerCode,
      createTrajectoryAccuracyScorerCode: embed.createTrajectoryAccuracyScorerCode,
      createTrajectoryScorerCode: embed.createTrajectoryScorerCode,

      // Mastra rag — relevance scorers for rerankWithScorer
      CohereRelevanceScorer: embed.CohereRelevanceScorer,
      MastraAgentRelevanceScorer: embed.MastraAgentRelevanceScorer,
      ZeroEntropyRelevanceScorer: embed.ZeroEntropyRelevanceScorer,

      // Mastra workspace — extended filesystem + tool factory + individual tools
      CompositeFilesystem: embed.CompositeFilesystem,
      createWorkspaceTools: embed.createWorkspaceTools,
      resolveToolConfig: embed.resolveToolConfig,
      readFileTool: embed.readFileTool,
      writeFileTool: embed.writeFileTool,
      editFileTool: embed.editFileTool,
      listFilesTool: embed.listFilesTool,
      deleteFileTool: embed.deleteFileTool,
      fileStatTool: embed.fileStatTool,
      mkdirTool: embed.mkdirTool,
      searchTool: embed.searchTool,
      indexContentTool: embed.indexContentTool,
      executeCommandTool: embed.executeCommandTool,
      requireWorkspace: embed.requireWorkspace,
      requireFilesystem: embed.requireFilesystem,
      requireSandbox: embed.requireSandbox,

      // Mastra browser — core provider contract + context processor
      MastraBrowser: embed.MastraBrowser,
      BrowserContextProcessor: embed.BrowserContextProcessor,

      // Mastra channels — core AgentChannels orchestration surface.
      // Concrete Slack/Discord/Telegram adapters still need explicit module /
      // gateway ownership before they count as provider support.
      AgentChannels: embed.AgentChannels,
      ChatChannelProcessor: embed.ChatChannelProcessor,
      MastraStateAdapter: embed.MastraStateAdapter,

      // Mastra voice — extended defaults
      DefaultVoice: embed.DefaultVoice,
      AISDKSpeech: embed.AISDKSpeech,
      AISDKTranscription: embed.AISDKTranscription,

      // Mastra observability — extended exporters
      BaseExporter: embed.BaseExporter,
      CloudExporter: embed.CloudExporter,
      ConsoleExporter: embed.ConsoleExporter,
      TestExporter: embed.TestExporter,
      TrackingExporter: embed.TrackingExporter,
      chainFormatters: embed.chainFormatters,

      // AI SDK — tool authoring
      //
      // NOTE: `tool` is intentionally NOT re-endowed here. The unified
      // `tool` entry above (paired with the kit surface) already
      // forwards object-shaped calls to `embed.tool` so AI SDK fixtures
      // (e.g. `tool({ description, inputSchema, execute })`) keep
      // compiling and running unchanged. A second `tool: embed.tool`
      // entry at this position would shadow that dispatcher and
      // silently break the `kit`-side `tool("name")` resolver — the
      // original regression surfaced by `agent/tools/with-registered-tool`.
      dynamicTool: embed.dynamicTool,
      jsonSchema: embed.jsonSchema,
      zodSchema: embed.zodSchema,
      asSchema: embed.asSchema,
      generateId: embed.generateId,
      createIdGenerator: embed.createIdGenerator,
      hasToolCall: embed.hasToolCall,
      stepCountIs: embed.stepCountIs,
      isLoopFinished: embed.isLoopFinished,

      // AI SDK — middleware
      wrapLanguageModel: embed.wrapLanguageModel,
      wrapEmbeddingModel: embed.wrapEmbeddingModel,
      wrapImageModel: embed.wrapImageModel,
      wrapProvider: embed.wrapProvider,
      extractReasoningMiddleware: embed.extractReasoningMiddleware,
      extractJsonMiddleware: embed.extractJsonMiddleware,
      defaultSettingsMiddleware: embed.defaultSettingsMiddleware,
      defaultEmbeddingSettingsMiddleware: embed.defaultEmbeddingSettingsMiddleware,
      simulateStreamingMiddleware: embed.simulateStreamingMiddleware,
      smoothStream: embed.smoothStream,
      addToolInputExamplesMiddleware: embed.addToolInputExamplesMiddleware,

      // AI SDK — provider registry
      createProviderRegistry: embed.createProviderRegistry,
      customProvider: embed.customProvider,
      experimental_createProviderRegistry: embed.experimental_createProviderRegistry,
      experimental_customProvider: embed.experimental_customProvider,

      // AI SDK — message utilities
      convertToModelMessages: embed.convertToModelMessages,
      pruneMessages: embed.pruneMessages,
      validateUIMessages: embed.validateUIMessages,
      safeValidateUIMessages: embed.safeValidateUIMessages,
      readUIMessageStream: embed.readUIMessageStream,
      consumeStream: embed.consumeStream,
      convertFileListToFileUIParts: embed.convertFileListToFileUIParts,

      // AI SDK — media
      generateImage: embed.generateImage,
      experimental_generateImage: embed.experimental_generateImage,
      experimental_generateVideo: embed.experimental_generateVideo,
      experimental_transcribe: embed.experimental_transcribe,
      experimental_generateSpeech: embed.experimental_generateSpeech,

      // AI SDK — misc
      cosineSimilarity: embed.cosineSimilarity,
      simulateReadableStream: embed.simulateReadableStream,
      parsePartialJson: embed.parsePartialJson,
      parseJsonEventStream: embed.parseJsonEventStream,

      // AI SDK — gateway
      gateway: embed.gateway,
      createGateway: embed.createGateway,

      // AI SDK — error classes (for instanceof in catch blocks)
      AISDKError: embed.AISDKError,
      APICallError: embed.APICallError,
      NoObjectGeneratedError: embed.NoObjectGeneratedError,
      NoSuchModelError: embed.NoSuchModelError,
      NoSuchToolError: embed.NoSuchToolError,
      InvalidArgumentError: embed.InvalidArgumentError,
      InvalidDataContentError: embed.InvalidDataContentError,
      InvalidPromptError: embed.InvalidPromptError,
      InvalidToolInputError: embed.InvalidToolInputError,
      NoContentGeneratedError: embed.NoContentGeneratedError,
      NoSpeechGeneratedError: embed.NoSpeechGeneratedError,
      NoTranscriptGeneratedError: embed.NoTranscriptGeneratedError,
      NoVideoGeneratedError: embed.NoVideoGeneratedError,
      RetryError: embed.RetryError,
      ToolCallRepairError: embed.ToolCallRepairError,
      TypeValidationError: embed.TypeValidationError,
      MessageConversionError: embed.MessageConversionError,
      MissingToolResultsError: embed.MissingToolResultsError,
      LoadAPIKeyError: embed.LoadAPIKeyError,
      InvalidToolApprovalError: embed.InvalidToolApprovalError,
      ToolCallNotFoundForApprovalError: embed.ToolCallNotFoundForApprovalError,
    };
    endowments.global = endowments;
    return typeof globalThis.harden === "function" ? globalThis.harden(endowments) : endowments;
  };
})();
