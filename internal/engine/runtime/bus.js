// bus.js — Resource registry, bus API, and kit.register.
// Shares _currentSource (globalThis.__kit_currentSource) between registry and bus.
// Outputs: globalThis.__kit_registry (resource registry), __kit_bus, __kit_kitAPI,
//          __bus_subs, __kit_currentSource
// Depends on: globalThis.__kit_bridgeRequest, __kit_bridgeRequestAsync, __kit_bridgeControl

(function() {
  "use strict";

  var bridgeControl = globalThis.__kit_bridgeControl;
  if (!bridgeControl) return;

  // ─── Shared source tracking ──────────────────────────────────
  // _currentSource is the active .ts deployment source, set by __kitRunWithSource.
  // Used by registry (tracks which source created a resource) and bus.schedule.
  Object.defineProperty(globalThis, '__kit_currentSource', {
    value: "", writable: true, enumerable: false, configurable: true
  });

  // ─── JS Refs — object references that can't cross Go/JS boundary ──
  // Keyed by "type:id". dispatch.js reads workflow/agent refs from here.
  var _refs = {};
  Object.defineProperty(globalThis, '__kit_refs', {
    value: _refs, writable: false, enumerable: false, configurable: true
  });

  var goResourceRegister = globalThis.__go_resource_register;

  // ─── Resource Registry ────────────────────────────────────────
  // Go owns metadata tracking (via __go_resource_register).
  // JS keeps: object refs (_refs), cleanup callbacks (cleanups).
  // Go TeardownFile runs Go-side cleanup and sweeps JS refs+cleanups.
  // On re-register, old cleanups are NOT fired — Go already ran cleanup during teardown.
  var _resourceRegistry = {
    cleanups: {},
    register: function(type, id, name, ref, cleanupFn, source) {
      var key = type + ":" + id;
      var resolvedSource = source || globalThis.__kit_currentSource || "unknown";
      // Track metadata in Go registry (source of truth for List/ListBySource/TeardownFile)
      if (goResourceRegister) {
        goResourceRegister(type, id, name || id, resolvedSource);
      }
      // Store JS object ref + source locally (Go can't hold JS heap objects)
      _refs[key] = { ref: ref || null, source: resolvedSource, name: name || id };
      // Replace cleanup — don't fire the old one (Go TeardownFile handles cleanup)
      if (typeof cleanupFn === "function") {
        this.cleanups[key] = cleanupFn;
      } else {
        delete this.cleanups[key];
      }
    },
    unregister: function(type, id) {
      var key = type + ":" + id;
      if (this.cleanups[key]) { try { this.cleanups[key](); } catch(e) {} delete this.cleanups[key]; }
      var entry = _refs[key];
      delete _refs[key];
      return entry ? { type: type, id: id, ref: entry.ref, source: entry.source } : null;
    },
    list: function(type) {
      var result = [];
      for (var key in _refs) {
        var parts = key.split(":");
        var t = parts[0];
        var id = parts.slice(1).join(":");
        if (!type || t === type) {
          var entry = _refs[key];
          result.push({ type: t, id: id, name: entry.name || id, source: entry.source || "" });
        }
      }
      return result;
    },
    get: function(type, id) {
      var entry = _refs[type + ":" + id];
      return entry ? { type: type, id: id, ref: entry.ref, source: entry.source, name: entry.name } : null;
    },
  };
  Object.defineProperty(globalThis, '__kit_registry', {
    value: _resourceRegistry, writable: false, enumerable: false, configurable: true
  });

  // ─── Bus Subscriptions Map ────────────────────────────────────
  Object.defineProperty(globalThis, '__bus_subs', {
    value: {}, writable: false, enumerable: false, configurable: true
  });

  // ─── Message Wrapper ──────────────────────────────────────────
  function wrapMsg(rawMsg) {
    var _seq = 0; // monotonic sequence number for stream events
    var msg = {
      payload: rawMsg.payload,
      replyTo: rawMsg.replyTo || "",
      correlationId: rawMsg.correlationId || "",
      topic: rawMsg.topic || "",
      callerId: rawMsg.callerId || "",
      reply: function(data) {
        if (msg.replyTo) {
          __go_brainkit_bus_reply(msg.replyTo, JSON.stringify(data), msg.correlationId, true);
        }
      },
      send: function(data) {
        if (msg.replyTo) {
          __go_brainkit_bus_reply(msg.replyTo, JSON.stringify(data), msg.correlationId, false);
        }
      },
      stream: {
          text: function(chunk) {
            if (msg.replyTo) {
              __go_brainkit_bus_reply(msg.replyTo,
                JSON.stringify({ type: "text", seq: _seq++, data: chunk }),
                msg.correlationId, false);
            }
          },
          progress: function(value, message) {
            if (msg.replyTo) {
              __go_brainkit_bus_reply(msg.replyTo,
                JSON.stringify({ type: "progress", seq: _seq++, data: { value: value, message: message || "" } }),
                msg.correlationId, false);
            }
          },
          object: function(partial) {
            if (msg.replyTo) {
              __go_brainkit_bus_reply(msg.replyTo,
                JSON.stringify({ type: "object", seq: _seq++, data: partial }),
                msg.correlationId, false);
            }
          },
          event: function(name, data) {
            if (msg.replyTo) {
              __go_brainkit_bus_reply(msg.replyTo,
                JSON.stringify({ type: "event", seq: _seq++, event: name, data: data || null }),
                msg.correlationId, false);
            }
          },
          error: function(message) {
            if (msg.replyTo) {
              __go_brainkit_bus_reply(msg.replyTo,
                JSON.stringify({ type: "error", seq: _seq, total: _seq, data: { message: typeof message === "string" ? message : String(message) } }),
                msg.correlationId, true);
            }
          },
          end: function(finalData) {
            if (msg.replyTo) {
              __go_brainkit_bus_reply(msg.replyTo,
                JSON.stringify({ type: "end", seq: _seq, total: _seq, data: finalData || null }),
                msg.correlationId, true);
            }
          },
        },
    };
    return msg;
  }

  // ─── Bus API ──────────────────────────────────────────────────
  globalThis.__kit_bus = {
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
        return handler(wrapMsg(rawMsg));
      };
      _resourceRegistry.register("subscription", subId, subId, null, function() {
        __go_brainkit_unsubscribe(subId);
        delete globalThis.__bus_subs[subId];
      });
      return subId;
    },
    on: function(localTopic, handler) {
      if (!globalThis.__kit_deployment_namespace) {
        throw new Error("bus.on() can only be used inside a deployed .ts file");
      }
      return globalThis.__kit_bus.subscribe(globalThis.__kit_deployment_namespace + "." + localTopic, handler);
    },
    unsubscribe: function(subId) {
      __go_brainkit_unsubscribe(subId);
      delete globalThis.__bus_subs[subId];
      _resourceRegistry.unregister("subscription", subId);
    },
    sendTo: function(service, localTopic, data) {
      var name = service.replace(/\.ts$/, "").replace(/\//g, ".");
      return globalThis.__kit_bus.publish("ts." + name + "." + localTopic, data);
    },
    schedule: function(expression, topic, data) {
      var id = __go_brainkit_bus_schedule(expression, topic, JSON.stringify(data || null), globalThis.__kit_currentSource || "go");
      _resourceRegistry.register("schedule", id, id, null, function() {
        __go_brainkit_bus_unschedule(id);
      });
      return id;
    },
    unschedule: function(scheduleId) {
      __go_brainkit_bus_unschedule(scheduleId);
      _resourceRegistry.unregister("schedule", scheduleId);
    },
  };

  // ─── kit.register ─────────────────────────────────────────────
  var _validTypes = { "tool": true, "agent": true, "workflow": true, "memory": true };

  globalThis.__kit_kitAPI = {
    register: function(type, name, ref) {
      if (!_validTypes[type]) {
        throw new Error("kit.register: invalid type '" + type + "' (must be tool, agent, workflow, or memory)");
      }
      if (!name || typeof name !== "string") {
        throw new Error("kit.register: name is required and must be a string");
      }
      var existing = _resourceRegistry.get(type, name);
      if (existing) return;
      var cleanupFn = null;
      if (type === "tool") {
        bridgeControl("tools.register", { name: name, description: (ref && ref.description) || "", inputSchema: {} });
        cleanupFn = function() { try { bridgeControl("tools.unregister", { name: name }); } catch(e) {} };
      } else if (type === "agent") {
        bridgeControl("agents.register", { name: name, capabilities: [], model: "", kit: globalThis.__brainkit_sandbox_id || "" });
        cleanupFn = function() { try { bridgeControl("agents.unregister", { name: name }); } catch(e) {} };
      }
      _resourceRegistry.register(type, name, name, ref, cleanupFn);
    },
    unregister: function(type, name) {
      // Only allow unregistering resources owned by the current deployment
      var entry = _resourceRegistry.get(type, name);
      if (entry && entry.source !== globalThis.__kit_currentSource && globalThis.__kit_currentSource !== "") {
        throw new Error("kit.unregister: cannot unregister " + type + " '" + name + "' owned by " + entry.source);
      }
      _resourceRegistry.unregister(type, name);
    },
    list: function(type) {
      return _resourceRegistry.list(type);
    },
    get source() { return globalThis.__kit_currentSource; },
    get namespace() { return globalThis.__brainkit_sandbox_namespace || ""; },
    get callerId() { return globalThis.__brainkit_sandbox_callerID || ""; },
  };
})();
