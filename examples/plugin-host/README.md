# plugin-host

Live round-trip driver for the sibling
[`examples/plugin-author/`](../plugin-author/) plugin.

```sh
go run ./examples/plugin-host
# building plugin …
# plugin registered: test/plugin-author@0.1.0
# echo reply: {"echoed":"ping"}
```

The same flow runs under `go test` — this is how the CI smoke
check gates plugin SDK regressions:

```sh
go test ./examples/plugin-host
```

Skipped under `go test -short` because the test shells out to
`go build` to compile the plugin binary.

## What it shows

- How to wire `modules/plugins.NewModule` onto an embedded-NATS
  Kit with one `brainkit.PluginConfig`.
- How to wait for the plugin to be ready (poll `plugin.list`
  instead of subscribing to `plugin.registered` — the event
  emits before the caller can subscribe in practice).
- How to invoke a plugin tool through the generated
  `brainkit.CallToolCall` wrapper.

## Contrast with `test/suite/plugins/`

The suite tests write the plugin source inline into a temp dir,
run `go mod tidy` with replace directives, then build. This host
example instead uses the shipped `examples/plugin-author/` source
— so it doubles as a smoke test for the scaffolder output.
