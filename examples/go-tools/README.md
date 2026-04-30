# go-tools

Mount typed Go functions as first-class brainkit tools.
Deployed `.ts` code and Go callers both invoke the tools over
the bus at `tools.call`.

## Run

```sh
go run ./examples/go-tools
```

Expected output:

```
typed Go tools invoked from .ts:
{"weather":{"city":"Paris","tempC":18,"condition":"cloudy"},"sum":{"sum":5}}
same tool invoked directly from Go (math.add):
{"sum":42}
```

## What it shows

- `toolsmod.GoTool(name, toolsmod.TypedTool[In]{…})`
  returns a scoped module that registers a typed Go function as a bus-addressable tool. The
  framework generates a JSON Schema from the `In` struct tags and
  serves it through `tools.list`.
- Deployed `.ts` code calls the tool through
  `bus.call("tools.call", {name, input}, {timeoutMs})` — same
  surface used for plugin tools and MCP tools.
- Go callers can invoke the same tool through the generated
  `toolmsg.CallToolCall` wrapper — no type-parameter guessing.

## When to use typed Go tools vs plugins

| | Typed Go tool | Subprocess plugin |
|---|---|---|
| Isolation | None (runs in the Kit's Go process) | Full (separate process, WS control) |
| Crash impact | Kills the Kit | Contained; supervisor restarts it |
| Restart cost | Cheap (function pointer) | Process restart |
| Use case | Helpers written by the Kit author | Capabilities written by third parties |

For author-owned helpers the typed path is simpler and faster;
for untrusted or separately-versioned capabilities, reach for
the plugin SDK (`sdk/plugin` + `examples/plugin-author`).
