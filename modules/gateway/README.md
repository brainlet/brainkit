# modules/gateway — stable

HTTP gateway for a brainkit Kit. Routes map incoming requests onto
bus topics; responses flow back through the shared-inbox Caller.

## Usage

```go
gw := gateway.New(gateway.Config{Listen: ":8080"})
gw.Handle("GET", "/hello", "ts.greeter.hello")

brainkit.New(brainkit.Config{Modules: []module.Module{gw}})
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

## Capabilities

- Requires: `brainkit.core.request_caller`.
- Uses when present: `brainkit.core.health_probes`,
  `brainkit.core.lifecycle_debug_registry`,
  `brainkit.core.runtime_control`.
- Provides: none.

## Runtime resources

Owns `gateway.listener`, the runtime route table (`gateway.routes`), dynamic
route command subscriptions, active HTTP/SSE/WebSocket request handlers, SSE
stream sessions, stream reply-topic subscription leases, and the stream-session
sweep loop.

## Hot unmount

Unmounting closes the HTTP server, unregisters dynamic route command
subscriptions, removes the `brainkit.core.lifecycle_debug_registry` snapshot,
clears the caller reference, cancels active HTTP/SSE/WebSocket handlers, closes
stream sessions, stops and waits for the session sweep loop, and releases the
listener resource. If graceful HTTP shutdown reaches the caller's deadline,
gateway force-closes the HTTP server and keeps the server handle until active
request bookkeeping drains, so a later close can finish cleanup. Lifecycle debug
reports `closing`, `serverAttached`, `activeConnections`, `streamSessions`,
`streamSubscriptions`, and `sessionSweepRunning` so teardown can be inspected.
Dynamic route and stream subscription close failures are retained for retry
instead of being dropped, and `Start` refuses to create duplicate resources
while a previous listener, route subscription, stream session, or stream
subscription is still active.
