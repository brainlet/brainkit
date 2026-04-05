# Gateway Test Map

**Purpose:** Verifies the HTTP gateway: routing, request/response, SSE streaming, WebSocket, health endpoints, CORS, error mapping, rate limiting, and attack resilience
**Tests:** 57 functions across 5 files
**Entry point:** `gateway_test.go` → `Run(t, env)`
**Campaigns:** transport (all 5), fullstack (all 3)

## Files

### routes.go — Core gateway routes and features

| Function | Purpose |
|----------|---------|
| testRequestResponseE2E | Deploys .ts handler, registers POST route, sends HTTP request, verifies JSON response |
| testTimeout504 | Registers route to handler that never replies, asserts 504 timeout |
| testWebhook | Registers webhook route, sends POST, asserts 202 accepted (fire-and-forget) |
| testDrainReturns503 | Sets kernel draining, sends request, asserts 503 |
| testDrainAllowsWebhooks | Sets kernel draining, sends webhook, asserts it still goes through |
| testNotFound | Sends request to unregistered path, asserts 404 |
| testCORSPreflight | Sends OPTIONS request, asserts CORS headers present |
| testWithHTTPContext | Registers route with WithHTTPContext, verifies handler receives HTTP method and path in payload |
| testPathParams | Registers route with path parameters, verifies params are passed to handler |
| testRouteTable | Registers multiple routes, fetches route table, verifies all registered |
| testBusRouteAdd | Adds a route via bus command (gateway.route.add), verifies it works |
| testBusRouteRemoveByOwner | Adds routes owned by a source, removes by owner, verifies routes are gone |
| testHealthEndpoints | Checks /healthz, /readyz, /livez return 200 |
| testReadyzDuringDrain | Sets draining, checks /readyz returns 503 |
| testSSEStreaming | Registers SSE route, sends request, verifies text/event-stream content type and events |
| testSSEProgressAndEvents | Registers SSE route with progress events, verifies progress and end events arrive |
| testSSEErrorTerminates | Registers SSE route to handler that errors, verifies stream terminates with error event |
| testErrorResponse500 | Registers route to handler that returns error, verifies 500 with error body |
| testHealthJSON | Checks /healthz with Accept: application/json, verifies JSON health response |
| testRouteReplacement | Registers a route then replaces it with new handler, verifies new handler serves |
| testBusRouteList | Adds routes, lists via bus command, verifies route entries |
| testBusStatus | Queries gateway status via bus command, verifies running state |
| testWebSocket | Registers WebSocket route, connects via ws://, sends message, verifies response |
| testRateLimiting | Configures gateway rate limiting, sends rapid requests, verifies some get 429 |

### stream.go — SSE streaming configuration tests

| Function | Purpose |
|----------|---------|
| testStreamHeartbeatTimeout | Deploys handler that stalls, configures short heartbeat timeout, verifies stream terminates with heartbeat_timeout reason |
| testStreamMaxDuration | Configures short max stream duration, verifies stream terminates when duration exceeded |
| testStreamMaxEvents | Configures max event count, verifies stream terminates after limit |
| testStreamKeepaliveComments | Configures keepalive interval, verifies SSE comment heartbeats arrive |
| testStreamReconnection | Verifies stream reconnection with Last-Event-ID header |
| testStreamSessionExpired | Attempts reconnection with expired session, verifies appropriate error |
| testStreamConcurrent | Opens multiple concurrent SSE streams, verifies all complete |
| testStreamGatewayShutdown | Opens SSE stream, stops gateway, verifies stream terminates cleanly |

### advanced.go — Adversarial advanced gateway tests

| Function | Purpose |
|----------|---------|
| testAdvSSEStreaming | Deploys handler using msg.stream.text/progress/end, verifies typed SSE events arrive correctly |
| testAdvWebhookDelivery | Registers webhook, sends POST, verifies 202 and bus event delivery |
| testAdvMultipleRoutes | Registers multiple routes pointing to different handlers, verifies each route dispatches correctly |
| testAdvRouteReplacement | Registers route, replaces it, verifies new handler serves and old is gone |
| testAdvConcurrentRequests | Sends multiple concurrent HTTP requests, verifies all get responses without deadlock |
| testAdvLargeResponse | Handler returns large response body, verifies full delivery |
| testAdvHealthDuringRequests | Checks /healthz while concurrent requests are in-flight, verifies 200 |

### errors.go — Gateway error handling

| Function | Purpose |
|----------|---------|
| testErrNotFound | Routes to handler that calls nonexistent tool, verifies 404 with NOT_FOUND error |
| testErrTimeout | Routes to handler that never replies, verifies 504 |
| testErrValidResponse | Routes to working handler, verifies 200 with correct body |
| testErrNoRoute | Requests unregistered path, verifies 404 |
| testErrHealthEndpoints | Checks /healthz, /readyz, /livez all return 200 |
| testErrCORS | Sends OPTIONS preflight, verifies CORS headers |
| testErrHandlerError | Routes to handler that returns an error object, verifies error in response |
| testErrLargePayload | Sends large request body, verifies handled without crash |
| testErrPathParams | Registers route with path params, verifies params extracted correctly |
| testErrGatewayStatusMapping | Verifies different error codes (NOT_FOUND, VALIDATION_ERROR, etc.) map to correct HTTP status codes |

### attacks.go — Gateway attack resilience

| Function | Purpose |
|----------|---------|
| testAttackRequestBodyBomb | Sends 10MB request body, verifies kernel survives |
| testAttackMethodConfusion | Sends wrong HTTP method to POST-only route, verifies 405 or 404 |
| testAttackConcurrentFlood | Floods with 100 concurrent requests, verifies kernel stays alive |
| testAttackSSEClientDisconnect | Opens SSE stream then disconnects client, verifies server cleans up |
| testAttackCORSBypass | Sends request with malicious Origin header, verifies CORS not bypassed |
| testAttackErrorInfoLeak | Triggers errors, verifies responses don't leak internal details |
| testAttackSlowloris | Opens connection with slow body, verifies timeout handling |
| testAttackRouteRemovalViaBus | Removes a route via bus while requests are in-flight, verifies no crash |

## Cross-references

- **Campaigns:** `transport/{sqlite,nats,postgres,redis,amqp}_test.go`, `fullstack/{redis_mongodb,amqp_postgres_vector,nats_postgres_rbac}_test.go`
- **Related domains:** bus (bus.on handlers), deploy (deploying handlers), health (health endpoints)
- **Fixtures:** none
