# modules/metrics - stable

Runtime metrics command surface. The module exposes the Kit metrics snapshot over
the bus for clients that do not use process metrics directly.

## Bus commands

- `metrics.get` - return the JSON-encoded runtime metrics payload.

The payload includes stable runtime counters plus low-cardinality bus and
transport health:

- `bus.published`, `bus.handled`, and `bus.errors` are keyed by logical topic.
- `bus.handleDuration*` records handler duration count, total, and max by
  logical topic.
- `transport` reports transport kind, active raw subscriptions, router handler
  counts, and active stream heartbeat goroutines.

Detailed lifecycle/debug snapshots remain on the control/debug surface rather
than being folded into metrics.

## Capabilities

- Requires: `brainkit.core.metrics_snapshot`.
- Provides: none.

## Runtime resources

None. The module owns only the `metrics.get` command handler.

## Hot unmount

Unmounting unregisters `metrics.get` and drops the snapshot callback.
