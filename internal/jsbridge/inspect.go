package jsbridge

import quickjs "github.com/buke/quickjs-go"

// InspectPolyfill provides globalThis.__util_inspect — a Node.js-style value
// formatter for console.log and friends. Handles circular references, errors,
// dates, regex, arrays, and nested objects with depth limiting.
type InspectPolyfill struct{}

func Inspect() *InspectPolyfill { return &InspectPolyfill{} }

func (p *InspectPolyfill) Name() string { return "inspect" }

func (p *InspectPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, inspectJS)
}

const inspectJS = `(function() {
  "use strict";

  var maxDepth = 4;
  var maxArrayLen = 100;
  var maxStringLen = 10000;

  function inspect(value, depth, seen) {
    if (depth === undefined) depth = 0;
    if (!seen) seen = [];

    // Primitives
    if (value === null) return "null";
    if (value === undefined) return "undefined";
    if (typeof value === "string") return depth > 0 ? JSON.stringify(value) : value;
    if (typeof value === "number" || typeof value === "boolean") return String(value);
    if (typeof value === "bigint") return String(value) + "n";
    if (typeof value === "symbol") return String(value);
    if (typeof value === "function") return "[Function: " + (value.name || "anonymous") + "]";

    // Depth limit
    if (depth > maxDepth) return "[...]";

    // Circular reference detection
    for (var i = 0; i < seen.length; i++) {
      if (seen[i] === value) return "[Circular]";
    }
    seen = seen.concat([value]);

    // Error
    if (value instanceof Error) {
      var errStr = value.name ? value.name + ": " : "Error: ";
      errStr += value.message || "";
      if (value.code) errStr += " [" + value.code + "]";
      if (value.stack && depth === 0) errStr += "\n" + value.stack;
      return errStr;
    }

    // Date
    if (value instanceof Date) return value.toISOString();

    // RegExp
    if (value instanceof RegExp) return String(value);

    // Array
    if (Array.isArray(value)) {
      if (value.length === 0) return "[]";
      var items = [];
      var len = Math.min(value.length, maxArrayLen);
      for (var j = 0; j < len; j++) {
        items.push(inspect(value[j], depth + 1, seen));
      }
      if (value.length > maxArrayLen) items.push("... " + (value.length - maxArrayLen) + " more");
      return "[ " + items.join(", ") + " ]";
    }

    // Map
    if (typeof Map !== "undefined" && value instanceof Map) {
      var mapItems = [];
      value.forEach(function(v, k) {
        mapItems.push(inspect(k, depth + 1, seen) + " => " + inspect(v, depth + 1, seen));
      });
      return "Map(" + value.size + ") { " + mapItems.join(", ") + " }";
    }

    // Set
    if (typeof Set !== "undefined" && value instanceof Set) {
      var setItems = [];
      value.forEach(function(v) {
        setItems.push(inspect(v, depth + 1, seen));
      });
      return "Set(" + value.size + ") { " + setItems.join(", ") + " }";
    }

    // Plain object
    var keys = Object.keys(value);
    if (keys.length === 0) return "{}";
    var props = [];
    for (var k = 0; k < keys.length; k++) {
      var key = keys[k];
      try {
        props.push(key + ": " + inspect(value[key], depth + 1, seen));
      } catch(e) {
        props.push(key + ": [Getter Error]");
      }
    }
    return "{ " + props.join(", ") + " }";
  }

  function formatArgs(args) {
    var parts = [];
    for (var i = 0; i < args.length; i++) {
      parts.push(inspect(args[i], 0));
    }
    return parts.join(" ");
  }

  // Truncate if extremely long (prevent log flooding)
  function format(args) {
    var s = formatArgs(args);
    if (s.length > maxStringLen) return s.substring(0, maxStringLen) + "... (truncated)";
    return s;
  }

  globalThis.__util_inspect = inspect;
  globalThis.__util_format = format;
})();`
