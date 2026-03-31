// infrastructure.js — Tool, FS, MCP, Registry, Secrets, and Output APIs.
// Outputs: globalThis.__kit_tools, __kit_fs, __kit_mcp, __kit_registry_api, __kit_secrets, __kit_output
// Depends on: globalThis.__kit_bridgeRequest, __kit_bridgeRequestAsync, __kit_bridgeControl, __kit_parseBridgeResponse

(function() {
  "use strict";

  var bridgeRequest = globalThis.__kit_bridgeRequest;
  var bridgeRequestAsync = globalThis.__kit_bridgeRequestAsync;
  var bridgeControl = globalThis.__kit_bridgeControl;
  var parseBridgeResponse = globalThis.__kit_parseBridgeResponse;
  if (!bridgeRequest) return;

  globalThis.__kit_tools = {
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

  globalThis.__kit_fs = {
    read: async function(path) { return parseBridgeResponse(await bridgeRequestAsync("fs.read", { path: path })); },
    write: async function(path, data) { return parseBridgeResponse(await bridgeRequestAsync("fs.write", { path: path, data: data })); },
    list: async function(path, pattern) { return parseBridgeResponse(await bridgeRequestAsync("fs.list", { path: path || ".", pattern: pattern || "" })); },
    stat: async function(path) { return parseBridgeResponse(await bridgeRequestAsync("fs.stat", { path: path })); },
    delete: async function(path) { return parseBridgeResponse(await bridgeRequestAsync("fs.delete", { path: path })); },
    mkdir: async function(path) { return parseBridgeResponse(await bridgeRequestAsync("fs.mkdir", { path: path })); },
  };

  globalThis.__kit_mcp = {
    listTools: function(server) { return parseBridgeResponse(bridgeRequest("mcp.listTools", { server: server || "" })).tools || []; },
    callTool: async function(server, tool, args) { return parseBridgeResponse(await bridgeRequestAsync("mcp.callTool", { server: server, tool: tool, args: args || {} })); },
  };

  globalThis.__kit_registry_api = {
    has: function(category, name) { return __go_registry_has(category, name) === "true"; },
    list: function(category) { return JSON.parse(__go_registry_list(category)); },
    resolve: function(category, name) { var r = __go_registry_resolve(category, name); return r ? JSON.parse(r) : null; },
    register: function(category, name, config) { bridgeControl("registry.register", { category: category, name: name, config: config }); },
    unregister: function(category, name) { bridgeControl("registry.unregister", { category: category, name: name }); },
  };

  globalThis.__kit_secrets = {
    get: function(name) {
      if (typeof __go_brainkit_secret_get === "function") {
        return __go_brainkit_secret_get(name);
      }
      return "";
    },
  };

  Object.defineProperty(globalThis, '__module_result', {
    value: undefined, writable: true, enumerable: false, configurable: true
  });
  globalThis.__kit_output = function(value) {
    globalThis.__module_result = typeof value === "string" ? value : JSON.stringify(value);
  };
})();
