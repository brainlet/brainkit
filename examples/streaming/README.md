# streaming

Every streaming surface the Kit exposes, wired onto one gateway:

- **Bus `CallStream`** — Go caller consumes ordered chunks from a
  `.ts` handler that emits via `msg.send`, then the terminal
  reply.
- **Gateway SSE route** — HTTP GET streams Server-Sent Events.
- **Gateway WebSocket route** — bidirectional WS.
- **Gateway Webhook route** — fire-and-forget (no reply).

## Run

```sh
go run ./examples/streaming
```

Expected output:

```
listening on http://127.0.0.1:XXXX
  SSE:     curl -N 'http://127.0.0.1:XXXX/sse/count?n=5'
  WS:      wscat -c 'ws://127.0.0.1:XXXX/ws/count?n=5'
  Webhook: curl -X POST 'http://127.0.0.1:XXXX/webhook/log' -d '{"msg":"hi"}'

bus CallStream round trip (Go):
  chunk: map[tick:1]
  chunk: map[tick:2]
  …
  terminal: done=true total=5
```

Keep the process running and run the curl / wscat commands in
another shell.

## SSE

```sh
curl -N 'http://127.0.0.1:PORT/sse/count?n=5'
```

You'll see five `data:` frames followed by a done event.

## WebSocket

```sh
wscat -c 'ws://127.0.0.1:PORT/ws/count?n=5'
```

(Install with `npm install -g wscat` if missing. Send any initial
JSON to trigger the stream; the gateway maps the request to
`ts.streaming-demo.count`.)

## Webhook

```sh
curl -X POST 'http://127.0.0.1:PORT/webhook/log' -d '{"msg":"hi"}'
```

Returns 202 Accepted immediately. The handler runs
asynchronously and logs the payload to the server stdout.

## When to pick which

| | Bus CallStream | Gateway SSE | Gateway WS | Webhook |
|---|---|---|---|---|
| Direction | Go caller → Kit | HTTP client ← Kit | bidirectional | HTTP client → Kit |
| Transport | in-process / NATS | HTTP (unidirectional) | HTTP → WS upgrade | HTTP (fire-and-forget) |
| Typed | yes (`Chunk` generic) | no (raw JSON frames) | no | no |
| Latency | fastest | low | lowest | lowest |
| Use case | Go→Kit streaming | Browser LLM chat | Interactive protocols | Webhooks from SaaS |
