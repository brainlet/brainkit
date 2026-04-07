// bridges.js — Bridge helper functions for Go ↔ JS communication.
// Wraps __go_brainkit_request, __go_brainkit_request_async, __go_brainkit_control.
// Outputs: globalThis.BrainkitError, __kit_bridgeRequest, __kit_bridgeRequestAsync,
//          __kit_bridgeControl, __kit_parseBridgeResponse

(function() {
  "use strict";

  // ─── BrainkitError class ──────────────────────────────────────
  // Defined on globalThis so it's available everywhere (global context + endowments).
  // Extends Error with code and details as first-class constructor parameters.
  // SES harden() preserves these because they're defined in the prototype chain,
  // not monkey-patched onto an instance after construction.
  function BrainkitError(message, code, details) {
    var instance = new Error(message);
    Object.setPrototypeOf(instance, BrainkitError.prototype);
    // Define code and details as own properties on the instance (not prototype).
    // This survives SES harden() — own data properties on a new object are writable
    // during construction even if the constructor itself is hardened.
    Object.defineProperty(instance, 'code', {
      value: code || "INTERNAL_ERROR", writable: false, enumerable: true, configurable: false
    });
    Object.defineProperty(instance, 'details', {
      value: details || {}, writable: false, enumerable: true, configurable: false
    });
    return instance;
  }
  BrainkitError.prototype = Object.create(Error.prototype, {
    constructor: { value: BrainkitError, writable: true, configurable: true },
    name: { value: "BrainkitError", writable: true, configurable: true },
  });

  globalThis.BrainkitError = BrainkitError;

  // ─── Bridge functions ──────────────────────────────────────────

  globalThis.__kit_bridgeRequest = function(topic, payload) {
    if (typeof __go_brainkit_request === "function") {
      return __go_brainkit_request(topic, typeof payload === "string" ? payload : JSON.stringify(payload));
    }
    throw new BrainkitError("platform bridge not available (topic: " + topic + ")", "BRIDGE_ERROR");
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
    throw new BrainkitError("platform control bridge not available (action: " + action + ")", "BRIDGE_ERROR");
  };

  globalThis.__kit_parseBridgeResponse = function(raw) {
    var result = JSON.parse(raw);
    if (result && result.error) {
      throw new BrainkitError(result.error, result.code || "INTERNAL_ERROR", result.details || {});
    }
    return result;
  };
})();
