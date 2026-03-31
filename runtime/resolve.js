// resolve.js — Model, provider, storage, and vector store resolution.
// Outputs: globalThis.__kit_resolveModel, __kit_resolveEmbeddingModel, __kit_resolveProvider,
//          __kit_resolveStorage, __kit_resolveVectorStore, __kit_clearProviderCache
// Depends on: globalThis.__agent_embed, __go_registry_resolve

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

  globalThis.__kit_resolveModel = function(providerName, modelId) {
    var providers = globalThis.__kit_providers || {};
    var pc = providers[providerName];
    if (!pc) return providerName + "/" + modelId;
    var factoryName = providerFactories[providerName];
    if (!factoryName || !embed[factoryName]) return providerName + "/" + modelId;
    var opts = { apiKey: pc.APIKey || pc.apiKey };
    if (pc.BaseURL || pc.baseURL) opts.baseURL = pc.BaseURL || pc.baseURL;
    return embed[factoryName](opts)(modelId);
  };

  globalThis.__kit_resolveEmbeddingModel = function(providerName, modelId) {
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
  };

  var _providerCache = {};
  globalThis.__kit_resolveProvider = function(name) {
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
  };

  var _storageCache = {};
  globalThis.__kit_resolveStorage = function(name) {
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
  };

  var _vectorStoreCache = {};
  globalThis.__kit_resolveVectorStore = function(name) {
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
  };

  globalThis.__kit_clearProviderCache = function(name) {
    if (name) { delete _providerCache[name]; }
    else { for (var k in _providerCache) delete _providerCache[k]; }
  };
})();
