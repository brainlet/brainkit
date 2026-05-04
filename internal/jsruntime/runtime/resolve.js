// resolve.js — Model, provider, storage, and vector store resolution.
// Outputs: globalThis.__kit_resolveModel, __kit_resolveEmbeddingModel, __kit_resolveProvider,
//          __kit_resolveStorage, __kit_resolveVectorStore, __kit_clearRegistryCache
// Depends on: globalThis.__agent_embed, __go_registry_resolve, __go_registry_runtime_resolve

(function() {
  "use strict";

  var embed = globalThis.__agent_embed;
  if (!embed) return;

  var providerFactories = {
    openai: "createOpenAI", anthropic: "createAnthropic",
    google: "createGoogleGenerativeAI", mistral: "createMistral",
    xai: "createXai", groq: "createGroq", deepseek: "createDeepSeek",
    cerebras: "createCerebras", perplexity: "createPerplexity",
    togetherai: "createTogetherAI", fireworks: "createFireworks",
    cohere: "createCohere",
  };

  function registryRuntimeResolve(category, name) {
    var resolver = typeof __go_registry_runtime_resolve === "function"
      ? __go_registry_runtime_resolve
      : __go_registry_resolve;
    var configJSON = resolver(category, name);
    return configJSON ? JSON.parse(configJSON) : null;
  }

  function startupProviderConfig(name) {
    var providers = globalThis.__kit_providers || {};
    var pc = providers[name];
    if (!pc) return null;
    return { type: pc.type || pc.Type || name, name: name, config: pc };
  }

  function providerConfig(name) {
    return registryRuntimeResolve("provider", name) || startupProviderConfig(name);
  }

  function providerType(parsed, fallbackName) {
    var cfg = parsed && parsed.config || {};
    return parsed && parsed.type || cfg.type || cfg.Type || fallbackName;
  }

  function providerOptions(cfg) {
    cfg = cfg || {};
    var opts = {};
    var apiKey = cfg.APIKey || cfg.apiKey || cfg.api_key;
    var authToken = cfg.AuthToken || cfg.authToken || cfg.auth_token;
    var baseURL = cfg.BaseURL || cfg.baseURL || cfg.base_url;
    var headers = cfg.Headers || cfg.headers;
    if (apiKey) opts.apiKey = apiKey;
    if (authToken) opts.authToken = authToken;
    if (baseURL) opts.baseURL = baseURL;
    if (headers) opts.headers = headers;
    if (cfg.Organization || cfg.organization) opts.organization = cfg.Organization || cfg.organization;
    if (cfg.Project || cfg.project) opts.project = cfg.Project || cfg.project;
    return opts;
  }

  function buildProvider(name, missingOK) {
    var parsed = providerConfig(name);
    if (!parsed) {
      if (missingOK) return null;
      throw new Error("AI provider '" + name + "' not registered");
    }
    var typ = providerType(parsed, name);
    var factoryName = providerFactories[typ];
    if (!factoryName || !embed[factoryName]) {
      if (missingOK) return null;
      throw new Error("AI provider '" + typ + "' not available");
    }
    return embed[factoryName](providerOptions(parsed.config || {}));
  }

  globalThis.__kit_resolveModel = function(providerName, modelId) {
    var prov = buildProvider(providerName, true);
    if (!prov) return providerName + "/" + modelId;
    return prov(modelId);
  };

  globalThis.__kit_resolveEmbeddingModel = function(providerName, modelId) {
    var prov = buildProvider(providerName, false);
    if (typeof prov.embedding === "function") return prov.embedding(modelId);
    if (typeof prov.textEmbeddingModel === "function") return prov.textEmbeddingModel(modelId);
    throw new Error("embeddingModel: provider '" + providerName + "' does not support embeddings");
  };

  var _providerCache = {};
  globalThis.__kit_resolveProvider = function(name) {
    if (_providerCache[name]) return _providerCache[name];
    var instance = buildProvider(name, false);
    _providerCache[name] = instance;
    return instance;
  };

  var _storageCache = {};
  globalThis.__kit_resolveStorage = function(name) {
    if (_storageCache[name]) return _storageCache[name];
    var configJSON = __go_registry_runtime_resolve("storage", name);
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
  };

  var _vectorStoreCache = {};
  globalThis.__kit_resolveVectorStore = function(name) {
    if (_vectorStoreCache[name]) return _vectorStoreCache[name];
    var configJSON = __go_registry_runtime_resolve("vectorStore", name);
    if (!configJSON) throw new Error("vector store '" + name + "' not registered");
    var parsed = JSON.parse(configJSON);
    var cfg = parsed.config || {};
    var instance;
    switch (parsed.type) {
      case "libsql": instance = new embed.LibSQLVector({ id: name, url: cfg.URL, authToken: cfg.AuthToken }); break;
      case "pgvector": instance = new embed.PgVector({ id: name, connectionString: cfg.ConnectionString }); break;
      case "mongodb": instance = new embed.MongoDBVector({ id: name, uri: cfg.URI, dbName: cfg.DBName }); break;
      default: throw new Error("vector store type '" + parsed.type + "' not available");
    }
    _vectorStoreCache[name] = instance;
    return instance;
  };

  globalThis.__kit_clearProviderCache = function(name) {
    if (name) { delete _providerCache[name]; }
    else { for (var k in _providerCache) delete _providerCache[k]; }
  };

  globalThis.__kit_clearStorageCache = function(name) {
    if (name) { delete _storageCache[name]; }
    else { for (var k in _storageCache) delete _storageCache[k]; }
  };

  globalThis.__kit_clearVectorStoreCache = function(name) {
    if (name) { delete _vectorStoreCache[name]; }
    else { for (var k in _vectorStoreCache) delete _vectorStoreCache[k]; }
  };

  globalThis.__kit_clearRegistryCache = function(category, name) {
    switch (category) {
      case "provider":
        globalThis.__kit_clearProviderCache(name);
        return true;
      case "storage":
        globalThis.__kit_clearStorageCache(name);
        return true;
      case "vectorStore":
        globalThis.__kit_clearVectorStoreCache(name);
        return true;
      default:
        return false;
    }
  };
})();
