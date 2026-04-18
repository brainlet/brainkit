# modules/gateway — stable

HTTP gateway for a brainkit Kit. Routes map incoming requests onto
bus topics; responses flow back through the shared-inbox Caller.

## Usage

```go
gw := gateway.New(gateway.Config{Listen: ":8080"})
gw.Handle("GET", "/hello", "ts.greeter.hello")

brainkit.New(brainkit.Config{Modules: []brainkit.Module{gw}})
```

## Route types

- `Handle(method, path, topic)` — request/response bus call.
- `HandleStream(method, path, topic)` — SSE stream.
- `HandleWebSocket(path, topic)` — bidirectional WS.
- `HandleWebhook(method, path, topic)` — fire-and-forget.

Dynamic routing is available over `gateway.http.route.add` /
`.remove` / `.list` / `gateway.http.status` bus commands so `.ts`
packages can own their HTTP surface.

## Config highlights

- `NoHealth` — skip the built-in `/health` / `/ready` endpoints.
- `CORS`, `Middleware`, `RateLimit`, `Stream` — optional knobs; see
  `gateway.Config` fields.
