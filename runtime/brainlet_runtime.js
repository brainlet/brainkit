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

  // agent() — create a persistent agent in THIS sandbox
  function agent(config) {
    var a = new embed.Agent({
      name: config.name || "unnamed",
      id: config.id || undefined,
      description: config.description || "",
      instructions: config.instructions || "",
      model: resolveModel(config.model),
      tools: config.tools || {},
    });

    return {
      _mastraAgent: a,
      generate: async function(promptOrMessages, options) {
        var result = await a.generate(
          typeof promptOrMessages === "string" ? promptOrMessages : promptOrMessages,
          options || {}
        );
        return {
          text: result.text || "",
          reasoning: result.reasoningText || "",
          usage: {
            promptTokens: result.usage?.inputTokens || result.usage?.promptTokens || 0,
            completionTokens: result.usage?.outputTokens || result.usage?.completionTokens || 0,
            totalTokens: result.usage?.totalTokens || 0,
          },
          finishReason: result.finishReason || "stop",
          toolCalls: result.toolCalls || [],
          toolResults: result.toolResults || [],
          steps: result.steps || [],
        };
      },
      stream: async function(promptOrMessages, options) {
        var result = await a.stream(
          typeof promptOrMessages === "string" ? promptOrMessages : promptOrMessages,
          options || {}
        );
        return result;
      },
    };
  }

  // createTool() — define a tool in THIS sandbox
  function createTool(config) {
    return embed.createTool({
      id: config.name || config.id,
      description: config.description || "",
      inputSchema: config.schema || embed.z.object({}),
      execute: config.execute,
    });
  }

  // z — Zod schemas
  var z = embed.z;

  // ─── PLATFORM (cross-sandbox, through Go bridges) ──────────────

  // Generic bridge request — calls Go function if available, otherwise no-op
  function bridgeRequest(topic, payload) {
    if (typeof __go_brainkit_request === "function") {
      return __go_brainkit_request(topic, typeof payload === "string" ? payload : JSON.stringify(payload));
    }
    throw new Error("brainlet: platform bridge not available (topic: " + topic + ")");
  }

  // Parse bridge response, throwing if it contains an error.
  function parseBridgeResponse(raw) {
    var result = JSON.parse(raw);
    if (result && result.error) {
      throw new Error("brainlet: " + result.error);
    }
    return result;
  }

  // ai.* — direct LLM calls via ai-embed
  var ai = {
    generate: async function(params) {
      var raw = bridgeRequest("ai.generate", params);
      return parseBridgeResponse(raw);
    },
    stream: async function(params) {
      var raw = bridgeRequest("ai.stream", params);
      return parseBridgeResponse(raw);
    },
    embed: async function(params) {
      var raw = bridgeRequest("ai.embed", params);
      return parseBridgeResponse(raw);
    },
    embedMany: async function(params) {
      var raw = bridgeRequest("ai.embedMany", params);
      return parseBridgeResponse(raw);
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
      var raw = bridgeRequest("tools.call", { name: name, input: input });
      return JSON.parse(raw);
    },
    list: async function(namespace) {
      var raw = bridgeRequest("tools.list", { namespace: namespace || "" });
      return JSON.parse(raw);
    },
    register: async function(name, config) {
      bridgeRequest("tools.register", { name: name, description: config.description, inputSchema: config.inputSchema });
    },
  };

  // tool() — namespace-aware tool lookup, returns Mastra-compatible tool
  function tool(name) {
    if (typeof __go_brainkit_request !== "function") {
      throw new Error("brainlet: platform bridge not available for tool resolution");
    }
    var raw = bridgeRequest("tools.resolve", { name: name });
    var info = JSON.parse(raw);

    var t = embed.createTool({
      id: info.shortName || name,
      description: info.description || "",
      inputSchema: embed.z.object({}),
      execute: async function(input) {
        return await tools.call(info.name || name, input);
      },
    });
    t._registryTool = true;
    return t;
  }

  // bus.* — platform bus
  var busMod = {
    send: async function(topic, payload) {
      bridgeRequest("bus.send", { topic: topic, payload: payload });
    },
    publish: async function(topic, payload) {
      bridgeRequest("bus.send", { topic: topic, payload: payload });
    },
    request: async function(topic, payload) {
      var raw = bridgeRequest(topic, payload);
      return JSON.parse(raw);
    },
  };

  // sandbox context
  var sandboxCtx = {
    id: globalThis.__brainkit_sandbox_id || "",
    namespace: globalThis.__brainkit_sandbox_namespace || "",
    callerID: globalThis.__brainkit_sandbox_callerID || "",
  };

  // ─── EXPORT ────────────────────────────────────────────────────

  globalThis.__brainlet = {
    // LOCAL
    agent: agent,
    createTool: createTool,
    z: z,

    // PLATFORM
    ai: ai,
    wasm: wasm,
    tools: tools,
    tool: tool,
    bus: busMod,

    // CONTEXT
    sandbox: sandboxCtx,
  };
})();
