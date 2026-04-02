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

  // ─── Export to globalThis.__kit ───────────────────────────────

  globalThis.__kit = {
    bus: bus,
    kit: kit,
    model: globalThis.__kit_resolveModel,
    embeddingModel: globalThis.__kit_resolveEmbeddingModel,
    provider: globalThis.__kit_resolveProvider,
    __clearProviderCache: globalThis.__kit_clearProviderCache,
    storage: globalThis.__kit_resolveStorage,
    vectorStore: globalThis.__kit_resolveVectorStore,
    registry: registry,
    tools: tools,
    fs: fs,
    mcp: mcp,
    output: output,
    secrets: secretsAPI,
    generateWithApproval: generateWithApproval,
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
      sendTo: _kitObj.bus.sendTo,
      sendToShard: _kitObj.bus.sendToShard,
      schedule: ws(function(expression, topic, data) {
        return _kitObj.bus.schedule(expression, ns + "." + topic, data);
      }),
      unschedule: _kitObj.bus.unschedule,
    };

    var scopedKit = {
      register: ws(_kitObj.kit.register),
      unregister: ws(_kitObj.kit.unregister),
      list: _kitObj.kit.list,
      get source() { return source; },
      get namespace() { return globalThis.__brainkit_sandbox_namespace || ""; },
      get callerId() { return globalThis.__brainkit_sandbox_callerID || ""; },
    };

    // Parse "[CODE] message {{details_json}}" format from error messages.
    // This format is set by throwBrainkitError in bridges.go to survive
    // QuickJS error wrapping and SES Compartment boundaries.
    var _BKE = globalThis.BrainkitError;
    var _codeRe = /^\[([A-Z_]+)\]\s/;
    var _detailsRe = /\s\{\{(.+)\}\}$/;
    function _parseError(e) {
      var msg = e && e.message ? e.message : String(e);
      var codeMatch = _codeRe.exec(msg);
      if (!codeMatch) return null;
      var code = codeMatch[1];
      var cleanMsg = msg.replace(_codeRe, "");
      var details = {};
      var detailsMatch = _detailsRe.exec(cleanMsg);
      if (detailsMatch) {
        try { details = JSON.parse(detailsMatch[1]); } catch(x) {}
        cleanMsg = cleanMsg.replace(_detailsRe, "");
      }
      return { message: cleanMsg, code: code, details: details };
    }

    function rewrapErrors(fn) {
      return function() {
        try { return fn.apply(this, arguments); }
        catch(e) {
          var parsed = _parseError(e);
          if (parsed) throw new _BKE(parsed.message, parsed.code, parsed.details);
          if (e && e.code) throw new _BKE(e.message, e.code, e.details || {});
          throw e;
        }
      };
    }
    function rewrapErrorsAsync(fn) {
      return async function() {
        try { return await fn.apply(this, arguments); }
        catch(e) {
          var parsed = _parseError(e);
          if (parsed) throw new _BKE(parsed.message, parsed.code, parsed.details);
          if (e && e.code) throw new _BKE(e.message, e.code, e.details || {});
          throw e;
        }
      };
    }

    var endowments = {
      // Error class — must be in endowments so Compartment code can catch with instanceof
      BrainkitError: _BKE,
      // brainkit infrastructure ("kit" module)
      bus: {
        publish: rewrapErrors(scopedBus.publish),
        emit: rewrapErrors(scopedBus.emit),
        subscribe: scopedBus.subscribe,
        on: scopedBus.on,
        unsubscribe: scopedBus.unsubscribe,
        sendTo: rewrapErrors(scopedBus.sendTo),
        sendToShard: rewrapErrors(scopedBus.sendToShard),
        schedule: rewrapErrors(scopedBus.schedule),
        unschedule: scopedBus.unschedule,
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
      fs: globalThis.fs,
      mcp: _kitObj.mcp,
      output: _kitObj.output,
      secrets: {
        get: rewrapErrors(_kitObj.secrets.get),
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
      // Compiler
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
      // JS built-ins
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
      // Web APIs
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
      net: globalThis.net,
      os: globalThis.os,
      dns: globalThis.dns,
      zlib: globalThis.zlib,
      child_process: globalThis.child_process,
    };
    return typeof globalThis.harden === "function" ? globalThis.harden(endowments) : endowments;
  };
})();
