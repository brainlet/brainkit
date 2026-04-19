# bus/ Fixtures

Tests the platform message bus: pub/sub, request/reply, streaming, mailbox routing, and adversarial edge cases.

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| emit-fire-and-forget | no | none | `bus.emit()` sends messages with no replyTo; subscriber receives both payloads |
| mailbox-on | no | none | `bus.on()` subscribes to deployment-namespaced mailbox topic (`ts.<source>.<local>`), handler receives payload and `msg.reply()` delivers response |
| publish-reply | no | none | `bus.publish()` returns `replyTo` + `correlationId`, subscriber uses `msg.reply()` to send response back |
| send-to-service | no | none | `bus.sendTo()` routes to a deployed .ts service by name, resolving `<name>.ts` + topic to `ts.<name>.<topic>` |
| streaming-send-reply | no | none | `msg.send()` delivers 3 intermediate streaming chunks then `msg.reply()` delivers final, subscriber collects all 4 messages |
| subscribe-basic | no | none | `bus.subscribe()` receives published messages, `bus.unsubscribe()` stops delivery, confirms post-unsubscribe messages are dropped |

### errors/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| concurrent-publish | no | none | Rapidly publishes 50 messages in a loop to verify no crash under concurrent load |
| large-payload | no | none | Publishes a 50KB string payload, verifies publish succeeds and replyTo is returned |
| schedule-unschedule | no | none | Schedules 5 future messages via `bus.schedule()`, unschedules all via `bus.unschedule()`, confirms valid IDs |
| send-no-heartbeat-adv | no | none | Verifies `msg.send()` intermediate chunks work correctly without triggering heartbeat goroutine interference, then `msg.reply()` delivers final |
| sendto | no | none | `bus.sendTo()` to a service with no receiver deployed; verifies publish succeeds and returns replyTo |
| streaming-protocol-adv | no | none | Exercises every `msg.stream.*` method: `text()`, `progress()`, `object()`, `event()`, `end()` to prove wire format works |
