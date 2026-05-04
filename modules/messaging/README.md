# modules/messaging - stable

Request/reply bridge module. It exposes a small bus command for issuing a nested
shared-inbox call from one topic handler to another topic.

## Bus commands

- `kit.send` - call a target topic with a JSON payload and return its terminal
  reply payload.

## Capabilities

- Requires: `brainkit.core.request_caller`.
- Provides: none.

## Runtime resources

None. The module owns only the `kit.send` command handler.

## Hot unmount

Unmounting unregisters `kit.send` and drops the request caller reference.

The implementation uses the SDK shared caller path, so replies are correlated by
request ID without creating a subscription per call.
