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
          // Terminal reply: wrap in wire envelope {ok:true, data} + set
          // envelope=true metadata so the Caller + SubscribeTo unwrap
          // cleanly. msg.send stays raw because its chunk/partial-reply
          // semantics overlap with msg.stream.* and we don't want to
          // claim envelope contract for both shapes.
          var env = JSON.stringify({ ok: true, data: data === undefined ? null : data });
          __go_brainkit_bus_reply(msg.replyTo, env, msg.correlationId, true, true);
        }
      },
      send: function(data) {
        if (msg.replyTo) {
          // Intentionally raw — see reply() note above.
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
            // Keep the legacy typed-stream-error shape here — the SSE
            // gateway depends on it. Envelope wrapping for streams will
            // land alongside the gateway stream rewrite.
            if (msg.replyTo) {
              __go_brainkit_bus_reply(msg.replyTo,
                JSON.stringify({ type: "error", seq: _seq, total: _seq, data: { message: typeof message === "string" ? message : String(message) } }),
                msg.correlationId, true);
            }
          },
          end: function(finalData) {
            // Keep the legacy typed-stream-end shape — same reason as
            // stream.error above.
            if (msg.replyTo) {
              __go_brainkit_bus_reply(msg.replyTo,
                JSON.stringify({ type: "end", seq: _seq, total: _seq, data: finalData || null }),
                msg.correlationId, true);
            }
          },
        },
      onCancel: function(handler) {
        if (!msg.correlationId) return function() {};
        return globalThis.__kit_bus.onCancel(msg.correlationId, handler);
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
        // Wrap user handler so BrainkitError throws (sync OR async) leak
        // their .code/.details into globalThis.__pending_handler_err so the
        // Go dispatcher can surface them as a typed envelope error.
        function _capture(e) {
          if (e && e.code) {
            globalThis.__pending_handler_err = {
              code: e.code, message: e.message || "", details: e.details || null,
            };
          }
        }
        try {
          var r = handler(wrapMsg(rawMsg));
          if (r && typeof r.then === "function") {
            return r.catch(function(e) { _capture(e); throw e; });
          }
          return r;
        } catch (e) { _capture(e); throw e; }
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
    // call(topic, data, { timeoutMs }) → Promise<responseData>
    // Publishes a request-reply command; waits for the envelope terminal;
    // throws BrainkitError on ok=false or the configured timeout.
    // timeoutMs is REQUIRED (mirrors Go's deadline rule).
    call: function(topic, data, opts) {
      opts = opts || {};
      if (!opts.timeoutMs || typeof opts.timeoutMs !== "number") {
        return Promise.reject(new BrainkitError("bus.call: timeoutMs is required", "VALIDATION_ERROR", { field: "timeoutMs" }));
      }
      return __go_brainkit_bus_call(topic, JSON.stringify(data === undefined ? null : data), "", opts.timeoutMs).then(function(raw) {
        if (raw === "" || raw === "null") return null;
        return JSON.parse(raw);
      });
    },
    // callTo(namespace, topic, data, { timeoutMs }) → same as call, cross-kit.
    callTo: function(namespace, topic, data, opts) {
      opts = opts || {};
      if (!opts.timeoutMs || typeof opts.timeoutMs !== "number") {
        return Promise.reject(new BrainkitError("bus.callTo: timeoutMs is required", "VALIDATION_ERROR", { field: "timeoutMs" }));
      }
      if (!namespace || typeof namespace !== "string") {
        return Promise.reject(new BrainkitError("bus.callTo: namespace is required", "VALIDATION_ERROR", { field: "namespace" }));
      }
      return __go_brainkit_bus_call(topic, JSON.stringify(data === undefined ? null : data), namespace, opts.timeoutMs).then(function(raw) {
        if (raw === "" || raw === "null") return null;
        return JSON.parse(raw);
      });
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
    // onCancel(correlationId, handler) → subscribes to `_brainkit.cancel`,
    // filters by correlationId, invokes handler when the upstream caller
    // signals cancellation. Returns an unsubscribe function.
    onCancel: function(correlationId, handler) {
      if (!correlationId) throw new BrainkitError("bus.onCancel: correlationId is required", "VALIDATION_ERROR", { field: "correlationId" });
      if (typeof handler !== "function") throw new BrainkitError("bus.onCancel: handler must be a function", "VALIDATION_ERROR", { field: "handler" });
      var subId = globalThis.__kit_bus.subscribe("_brainkit.cancel", function(msg) {
        var body = msg && msg.payload;
        if (body && body.correlationId === correlationId) {
          try { handler(body); } catch (_) {}
        }
      });
      return function() { globalThis.__kit_bus.unsubscribe(subId); };
    },
    // withCancelController(msg) → { signal, cleanup }. Handlers pass
    // signal to fetch / AbortController-aware APIs and call cleanup
    // before they return so the cancel subscription is torn down.
    withCancelController: function(msg) {
      var controller = new AbortController();
      var unsubscribe = function() {};
      var correlationId = msg && msg.correlationId;
      if (correlationId) {
        unsubscribe = globalThis.__kit_bus.onCancel(correlationId, function() {
          controller.abort();
        });
      }
      return { signal: controller.signal, cleanup: unsubscribe };
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
