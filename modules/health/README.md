# modules/health - stable

Runtime health command surface. The module converts the Kit health snapshot into
a bus reply for clients that do not use the HTTP gateway.

## Bus commands

- `kit.health` - return the JSON-encoded runtime health snapshot.

## Capabilities

- Requires: `brainkit.core.health_snapshot`.
- Provides: none.

## Runtime resources

None. The module owns only the `kit.health` command handler.

## Hot unmount

Unmounting unregisters `kit.health` and drops the snapshot callback.
