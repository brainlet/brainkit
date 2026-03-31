// bridges.js — Bridge helper functions for Go ↔ JS communication.
// Wraps __go_brainkit_request, __go_brainkit_request_async, __go_brainkit_control.
// Outputs: globalThis.__kit_bridgeRequest, __kit_bridgeRequestAsync, __kit_bridgeControl, __kit_parseBridgeResponse

(function() {
  "use strict";

  globalThis.__kit_bridgeRequest = function(topic, payload) {
    if (typeof __go_brainkit_request === "function") {
      return __go_brainkit_request(topic, typeof payload === "string" ? payload : JSON.stringify(payload));
    }
    throw new Error("brainkit: platform bridge not available (topic: " + topic + ")");
  };

  globalThis.__kit_bridgeRequestAsync = function(topic, payload) {
    if (typeof __go_brainkit_request_async === "function") {
      return __go_brainkit_request_async(topic, typeof payload === "string" ? payload : JSON.stringify(payload));
    }
    return Promise.resolve(globalThis.__kit_bridgeRequest(topic, payload));
  };

  globalThis.__kit_bridgeControl = function(action, payload) {
    if (typeof __go_brainkit_control === "function") {
      return __go_brainkit_control(action, typeof payload === "string" ? payload : JSON.stringify(payload));
    }
    throw new Error("brainkit: platform control bridge not available (action: " + action + ")");
  };

  globalThis.__kit_parseBridgeResponse = function(raw) {
    var result = JSON.parse(raw);
    if (result && result.error) throw new Error("brainkit: " + result.error);
    return result;
  };
})();
