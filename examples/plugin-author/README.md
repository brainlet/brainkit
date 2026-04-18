# plugin-author

Minimal brainkit plugin — registers one tool and one bus
subscription, then connects back to its host Kit over WebSocket.
Real plugins live as standalone Go binaries outside the brainkit
module; this directory carries its own `go.mod` with replace
directives back to the repo root.

## Build

```sh
cd examples/plugin-author && go mod tidy && go build .
```

Produces `./examples/plugin-author/plugin-author`.

## Run under a host Kit

```go
import (
    "github.com/brainlet/brainkit"
    pluginsmod "github.com/brainlet/brainkit/modules/plugins"
)

kit, _ := brainkit.New(brainkit.Config{
    Namespace: "demo",
    Transport: brainkit.EmbeddedNATS(),
    Modules: []brainkit.Module{
        pluginsmod.NewModule(pluginsmod.Config{
            Plugins: []brainkit.PluginConfig{{
                Name:   "demo",
                Binary: "./examples/plugin-author/plugin-author",
            }},
        }),
    },
})
```

After `kit.Deploy` or a bus `tools.call` for `echo`, the host
dispatches to this plugin over the WS control plane.

## Run + verify end-to-end

A runnable host lives at [`examples/plugin-host/`](../plugin-host/)
alongside this directory. It builds this plugin from source into a
temp directory, boots a Kit, waits for the plugin to register,
calls the `echo` tool, and prints the reply:

```sh
go run ./examples/plugin-host
# building plugin …
# plugin registered: test/plugin-author@0.1.0
# echo reply: {"echoed":"ping"}
```

The same flow runs under `go test`:

```sh
go test ./examples/plugin-host
```

The test is skipped under `go test -short` because it shells out
to `go build`.

## What it shows

- `bkplugin.New(owner, name, version, opts...)` is the manifest
  boilerplate; `owner/name@version` shows up in the host's
  `plugin.registered` event.
- `bkplugin.Tool[In, Out](p, name, desc, fn)` auto-generates the
  JSON schema from `In` and routes `tools.call {name: "echo"}` to
  `fn`.
- `bkplugin.On[E](p, topic, fn)` subscribes over WS; the host
  forwards matching bus events.
- `p.Run()` owns the WS handshake, manifest send, tool loop, and
  shutdown signals — plugin authors never touch the protocol.
