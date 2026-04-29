// test_runtime.js — Test framework for brainkit .ts test files.
// Sets up globalThis.__test for the "test" module exports.
// Hardhat-inspired: deploy services, send bus messages, assert responses.
// Depends on: globalThis.__kit (bus.publish, bus.subscribe, bus.unsubscribe)

(function() {
  "use strict";

  var _kitBus = globalThis.__kit && globalThis.__kit.bus;
  if (!_kitBus) return; // kit not loaded yet

  var _testResults = [];
  var _testHooks = { beforeAll: [], afterAll: [], beforeEach: [], afterEach: [] };
  var _testDeployments = [];       // test-scoped deployments (torn down after each test)
  var _beforeAllDeployments = [];  // beforeAll deployments (persist across tests, torn down in afterAll)
  var _testTimeout = 30000;        // default per-test timeout: 30s

  globalThis.__test = {
    test: function(name, fn) { _testResults.push({ name: name, fn: fn, timeout: _testTimeout }); },
    describe: function(name, fn) { fn(); },
    beforeAll: function(fn) { _testHooks.beforeAll.push(fn); },
    afterAll: function(fn) { _testHooks.afterAll.push(fn); },
    beforeEach: function(fn) { _testHooks.beforeEach.push(fn); },
    afterEach: function(fn) { _testHooks.afterEach.push(fn); },
    timeout: function(ms) { _testTimeout = ms; },
    expect: function(value) {
      var _not = false;
      var _check = function(pass, msg) {
        if (_not) { pass = !pass; msg = "NOT: " + msg; }
        if (!pass) throw new Error(msg);
      };
      var api = {
        toBe: function(expected) { _check(value === expected, "Expected " + JSON.stringify(value) + " to be " + JSON.stringify(expected)); },
        toEqual: function(expected) { _check(JSON.stringify(value) === JSON.stringify(expected), "Expected deep equal failed"); },
        toContain: function(sub) { var s = typeof value === "string" ? value : JSON.stringify(value); _check(s.indexOf(sub) !== -1, "Expected to contain " + JSON.stringify(sub)); },
        toMatch: function(pat) { _check(new RegExp(pat).test(String(value)), "Expected to match " + pat); },
        toBeTruthy: function() { _check(!!value, "Expected truthy, got " + JSON.stringify(value)); },
        toBeFalsy: function() { _check(!value, "Expected falsy, got " + JSON.stringify(value)); },
        toBeDefined: function() { _check(value !== undefined && value !== null, "Expected defined"); },
        toBeNull: function() { _check(value === null || value === undefined, "Expected null, got " + JSON.stringify(value)); },
        toBeGreaterThan: function(n) { _check(value > n, value + " not > " + n); },
        toBeLessThan: function(n) { _check(value < n, value + " not < " + n); },
        toHaveLength: function(n) { var l = value && value.length !== undefined ? value.length : 0; _check(l === n, "Expected length " + n + ", got " + l); },
        toHaveProperty: function(key) { _check(value && typeof value === "object" && key in value, "Expected property " + JSON.stringify(key)); },
        toThrow: function(msg) {
          if (typeof value !== "function") throw new Error("toThrow: expected a function");
          var threw = false, threwMsg = "";
          try { value(); } catch(e) { threw = true; threwMsg = e.message || String(e); }
          _check(threw, "Expected function to throw");
          if (msg) _check(threwMsg.indexOf(msg) !== -1, "Expected throw message to contain " + JSON.stringify(msg) + ", got " + JSON.stringify(threwMsg));
        },
        get not() { _not = !_not; return api; },
      };
      return api;
    },
    deploy: async function(source, code) {
      var name = source.replace(/\.ts$/, "");
      var manifest = { name: name, entry: source };
      var files = {}; files[source] = code;
      var raw = await __go_brainkit_request_async("package.deploy", JSON.stringify({ manifest: manifest, files: files }));
      var result = JSON.parse(raw);
      if (result && result.error) throw new Error("deploy: " + result.error);
      _testDeployments.push(name);
      return result;
    },
    deployFile: async function(path) {
      var raw = await __go_brainkit_request_async("package.deploy", JSON.stringify({ path: path }));
      var result = JSON.parse(raw);
      if (result && result.error) throw new Error("deployFile: " + result.error);
      if (result && result.name) _testDeployments.push(result.name);
      return result;
    },
    sleep: function(ms) { return new Promise(function(r) { setTimeout(r, ms); }); },
    // Promise-based sendTo — publishes to service mailbox, subscribes to replyTo, waits for reply
    sendTo: function(service, topic, data, timeoutMs) {
      var name = service.replace(/\.ts$/, "").replace(/\//g, ".");
      var fullTopic = "ts." + name + "." + topic;
      var pubResult = _kitBus.publish(fullTopic, data);
      return new Promise(function(resolve, reject) {
        var timer = setTimeout(function() {
          _kitBus.unsubscribe(subId);
          reject(new Error("sendTo timeout after " + (timeoutMs || 10000) + "ms"));
        }, timeoutMs || 10000);
        var subId = _kitBus.subscribe(pubResult.replyTo, function(msg) {
          clearTimeout(timer);
          _kitBus.unsubscribe(subId);
          resolve(msg.payload);
        });
      });
    },
    evaluate: async function(service, topic, cases) {
      var results = { total: cases.length, passed: 0, failed: 0, items: [] };
      var start = Date.now();
      for (var i = 0; i < cases.length; i++) {
        var c = cases[i];
        var caseStart = Date.now();
        try {
          var resp = await globalThis.__test.sendTo(service, topic, c.input, c.timeout || 10000);
          var passed = true;
          if (c.expect) {
            for (var key in c.expect) {
              var expected = c.expect[key];
              if (expected instanceof RegExp) {
                if (!expected.test(String(resp[key]))) passed = false;
              } else if (resp[key] !== expected) {
                passed = false;
              }
            }
          }
          if (passed) results.passed++; else results.failed++;
          results.items.push({ input: c.input, output: resp, expected: c.expect, passed: passed, duration: Date.now() - caseStart });
        } catch(e) {
          results.failed++;
          results.items.push({ input: c.input, passed: false, error: e.message, duration: Date.now() - caseStart });
        }
      }
      results.accuracy = results.total > 0 ? results.passed / results.total : 0;
      results.totalDuration = Date.now() - start;
      return results;
    },
    runWorkflow: async function(workflowId, opts) {
      var input = opts && opts.input ? JSON.stringify(opts.input) : "{}";
      var runPayload = { workflowId: workflowId, input: JSON.parse(input) };
      if (opts && opts.hostResults) { runPayload.hostResults = opts.hostResults; }
      var raw = await __go_brainkit_request_async("workflow.run", JSON.stringify(runPayload));
      var resp = JSON.parse(raw);
      if (resp && resp.error) throw new Error("runWorkflow: " + resp.error);
      var runId = resp.runId;
      for (var i = 0; i < 300; i++) {
        await globalThis.__test.sleep(100);
        var statusRaw = await __go_brainkit_request_async("workflow.status", JSON.stringify({ runId: runId }));
        var status = JSON.parse(statusRaw);
        if (status.status === "completed" || status.status === "failed") {
          var histRaw = await __go_brainkit_request_async("workflow.history", JSON.stringify({ runId: runId }));
          var hist = JSON.parse(histRaw);
          return { status: status.status, output: status.output, steps: hist.entries || [], journal: hist.entries || [], runId: runId };
        }
      }
      return { status: "timeout", runId: runId };
    },
  };

  globalThis.__runTests = async function() {
    var results = [];
    // beforeAll hooks — deployments here persist across tests
    var prevDeployments = _testDeployments.slice();
    for (var i = 0; i < _testHooks.beforeAll.length; i++) await _testHooks.beforeAll[i]();
    _beforeAllDeployments = _testDeployments.filter(function(d) { return prevDeployments.indexOf(d) === -1; });
    _testDeployments = [];

    for (var t = 0; t < _testResults.length; t++) {
      for (var j = 0; j < _testHooks.beforeEach.length; j++) await _testHooks.beforeEach[j]();
      var testEntry = _testResults[t];
      var start = Date.now();
      var testTimeoutMs = testEntry.timeout || 30000;
      try {
        var result = await Promise.race([
          testEntry.fn(),
          new Promise(function(_, rej) { setTimeout(function() { rej(new Error("test timeout (" + testTimeoutMs + "ms)")); }, testTimeoutMs); }),
        ]);
        results.push({ name: testEntry.name, passed: true, duration: Date.now() - start });
      } catch(e) {
        results.push({ name: testEntry.name, passed: false, error: e.message || String(e), duration: Date.now() - start });
      }
      // Teardown test-scoped deployments only (not beforeAll ones)
      for (var d = _testDeployments.length - 1; d >= 0; d--) {
        try { __go_brainkit_request("package.teardown", JSON.stringify({ name: _testDeployments[d] })); } catch(e) {}
      }
      _testDeployments = [];
      for (var k = 0; k < _testHooks.afterEach.length; k++) await _testHooks.afterEach[k]();
    }

    // afterAll hooks
    for (var m = 0; m < _testHooks.afterAll.length; m++) await _testHooks.afterAll[m]();
    // Teardown beforeAll deployments
    for (var b = _beforeAllDeployments.length - 1; b >= 0; b--) {
      try { __go_brainkit_request("package.teardown", JSON.stringify({ name: _beforeAllDeployments[b] })); } catch(e) {}
    }
    _beforeAllDeployments = [];
    _testResults = [];
    _testHooks = { beforeAll: [], afterAll: [], beforeEach: [], afterEach: [] };
    _testTimeout = 30000;
    return JSON.stringify(results);
  };
})();
