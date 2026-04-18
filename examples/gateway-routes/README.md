# gateway-routes

HTTP gateway on top of a Kit. Registers `GET /hello` → bus topic
`ts.greeter.hello`, deploys a `.ts` handler behind it, serves
until `SIGINT`.

```sh
go run ./examples/gateway-routes
# prints:
#   listening on http://127.0.0.1:<port>
#     curl 'http://127.0.0.1:<port>/hello?name=world'
```

In another shell:

```sh
curl 'http://127.0.0.1:<port>/hello?name=world'
# {"greeting":"hello, world"}
```

## What it shows

- `modules/gateway` composes onto a bare Kit — no `server` package
  needed for small experiments.
- `gw.Handle(method, path, topic)` maps HTTP → bus. The same API
  is available from `.ts` via the `gateway.http.route.*` bus
  commands when you want dynamic routing from deployed code.
- `bus.on(topic, handler)` in the `.ts` receives the HTTP payload
  as `msg.payload` and calls `msg.reply` with the response body.
