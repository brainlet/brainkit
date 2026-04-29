// dispatch.js — Go-callable functions for bus command handlers.
// Replaces inline JS in Go handlers with named, testable functions.
// Loaded last (after bus.js, kit_runtime.js) — all globalThis APIs available.
// Go calls these via Kernel.callJS(ctx, "__brainkit.<domain>.<method>", args).

(function() {
  "use strict";

  var refs = globalThis.__kit_refs;
  var registry = globalThis.__kit_registry;
  if (!registry) return;

  function getWorkflow(name) {
    var entry = refs["workflow:" + name];
    if (!entry || !entry.ref) throw new BrainkitError("workflow not found: " + name, "NOT_FOUND");
    return entry.ref;
  }

  function workflowStatus(runState) {
    return (runState && runState.status) || "unknown";
  }

  function isTerminalWorkflowStatus(status) {
    return status === "success" || status === "failed" || status === "canceled" || status === "cancelled";
  }

  async function getWorkflowRunState(wf, runId) {
    var runState = await wf.getWorkflowRunById(runId);
    if (!runState) {
      throw new BrainkitError("workflow run not found: " + runId, "NOT_FOUND", { runId: runId });
    }
    return runState;
  }

  function wrapWorkflowError(op, err, code) {
    if (typeof BrainkitError === "function" && err instanceof BrainkitError) {
      throw err;
    }
    var message = err && err.message ? err.message : String(err);
    throw new BrainkitError("workflow." + op + " failed: " + message, code || "INTERNAL_ERROR");
  }

  globalThis.__brainkit = {

    // ── Workflow ──────────────────────────────────────────────────

    workflow: {
      start: async function(args) {
        var wf = getWorkflow(args.name);
        var run = await wf.createRun();
        var result = await run.start({ inputData: args.inputData || null });
        return {
          runId: run.runId || "",
          status: result.status || "unknown",
          steps: result.steps || null,
          result: result.result !== undefined ? result.result : null,
        };
      },

      startAsync: async function(args) {
        var wf = getWorkflow(args.name);
        var run = await wf.createRun();
        var runId = run.runId || "";
        var name = args.name;
        run.start({ inputData: args.inputData || null }).then(function(result) {
          __go_brainkit_bus_emit("workflow.completed." + runId, JSON.stringify({
            runId: runId, name: name, status: result.status || "unknown", steps: result.steps || null,
            result: result.result !== undefined ? result.result : null,
          }));
        }).catch(function(err) {
          __go_brainkit_bus_emit("workflow.completed." + runId, JSON.stringify({
            runId: runId, name: name, status: "failed", error: err.message || String(err),
          }));
        });
        return { runId: runId };
      },

      status: async function(args) {
        var wf = getWorkflow(args.name);
        var runState = await getWorkflowRunState(wf, args.runId);
        return {
          runId: args.runId,
          status: workflowStatus(runState),
          steps: runState.steps || null,
        };
      },

      resume: async function(args) {
        var wf = getWorkflow(args.name);
        var runState = await getWorkflowRunState(wf, args.runId);
        var status = workflowStatus(runState);
        if (isTerminalWorkflowStatus(status)) {
          throw new BrainkitError("workflow run is already complete: " + args.runId, "VALIDATION_ERROR", {
            runId: args.runId,
            status: status,
          });
        }
        if (status !== "suspended") {
          throw new BrainkitError("workflow run is not suspended: " + args.runId, "VALIDATION_ERROR", {
            runId: args.runId,
            status: status,
          });
        }
        var run = await wf.createRun({ runId: args.runId });
        var opts = { resumeData: args.resumeData || null };
        if (args.step) opts.step = args.step;
        var result;
        try {
          result = await run.resume(opts);
        } catch (err) {
          wrapWorkflowError("resume", err, "VALIDATION_ERROR");
        }
        return {
          status: workflowStatus(result),
          steps: result.steps || null,
        };
      },

      cancel: async function(args) {
        var wf = getWorkflow(args.name);
        var runState = await getWorkflowRunState(wf, args.runId);
        var status = workflowStatus(runState);
        if (isTerminalWorkflowStatus(status)) {
          throw new BrainkitError("workflow run is already complete: " + args.runId, "VALIDATION_ERROR", {
            runId: args.runId,
            status: status,
          });
        }
        var run = await wf.createRun({ runId: args.runId });
        try {
          await run.cancel();
        } catch (err) {
          wrapWorkflowError("cancel", err);
        }
        return { cancelled: true };
      },

      list: function() {
        var entries = registry.list("workflow");
        var result = [];
        for (var i = 0; i < entries.length; i++) {
          var e = entries[i];
          var ref = registry.get("workflow", e.name);
          result.push({
            name: e.name,
            source: e.source || "",
            hasInput: !!(ref && ref.ref && ref.ref.inputSchema),
            hasOutput: !!(ref && ref.ref && ref.ref.outputSchema),
          });
        }
        return { workflows: result };
      },

      runs: async function(args) {
        var wf = getWorkflow(args.name);
        var opts = {};
        if (args.status) opts.status = args.status;
        var result = await wf.listWorkflowRuns(opts);
        return {
          runs: result.runs || [],
          total: result.total || 0,
        };
      },

      restart: async function(args) {
        var wf = getWorkflow(args.name);
        await getWorkflowRunState(wf, args.runId);
        var run = await wf.createRun({ runId: args.runId });
        var result;
        try {
          result = await run.restart();
        } catch (err) {
          wrapWorkflowError("restart", err);
        }
        return {
          status: workflowStatus(result),
          steps: result.steps || null,
        };
      },
    },

    // ── Tools ─────────────────────────────────────────────────────

    tools: {
      execute: async function(args) {
        var entry = registry.get("tool", args.name);
        var execFn = entry && entry.ref && typeof entry.ref.__brainkit_execute === "function"
          ? entry.ref.__brainkit_execute
          : entry && entry.ref && typeof entry.ref.execute === "function"
            ? entry.ref.execute
            : null;
        if (!execFn) {
          throw new Error("tool not found in JS registry: " + args.name);
        }
        var input = args.input;
        var wrapped;
        if (input && typeof input === "object" && !Array.isArray(input)) {
          wrapped = {};
          for (var key in input) wrapped[key] = input[key];
          wrapped.context = input;
        } else {
          wrapped = { context: input };
        }
        var result = await execFn(wrapped, { requestContext: null });
        return result === undefined ? null : result;
      },
    },

    // ── Secrets ───────────────────────────────────────────────────

    secrets: {
      refreshProvider: function(args) {
        var providers = globalThis.__kit_providers;
        if (!providers || !providers[args.provider]) return;
        providers[args.provider].APIKey = args.apiKey;
        providers[args.provider].apiKey = args.apiKey;
        if (globalThis.__kit && globalThis.__kit.__clearProviderCache) {
          globalThis.__kit.__clearProviderCache(args.provider);
        }
      },
    },

    // ── Storage ───────────────────────────────────────────────────

    storage: {
      upgrade: async function() {
        var storageNames = globalThis.__kit_registry_api.list("storage");
        var storageName = null;
        for (var i = 0; i < storageNames.length; i++) {
          if (storageNames[i].name === "default") { storageName = "default"; break; }
        }
        if (!storageName && storageNames.length > 0) storageName = storageNames[0].name;
        if (storageName) {
          var configuredStorage = globalThis.__kit_resolveStorage(storageName);
          await configuredStorage.init();
          globalThis.__kit_store_holder.store = configuredStorage;
          return { upgraded: true, storage: storageName };
        }
        return { upgraded: false };
      },

      restartWorkflows: async function() {
        var restarted = 0;
        var errors = [];
        for (var key in refs) {
          if (key.indexOf("workflow:") !== 0) continue;
          var wfName = key.substring(9);
          var entry = refs[key];
          if (entry && entry.ref && typeof entry.ref.restartAllActiveWorkflowRuns === "function") {
            try {
              await entry.ref.restartAllActiveWorkflowRuns();
              restarted++;
            } catch(e) {
              errors.push({ workflow: wfName, error: e.message || String(e) });
            }
          }
        }
        return { restarted: restarted, errors: errors };
      },
    },
  };
})();
