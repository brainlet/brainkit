# modules/secrets - stable

Secret management command surface. The Kit secret store can exist without this
module; mount this module only when `secrets.*` bus administration should be
available.

## Bus commands

- `secrets.set`
- `secrets.get`
- `secrets.delete`
- `secrets.list`
- `secrets.rotate`

## Events

- `secrets.stored`
- `secrets.accessed`
- `secrets.deleted`
- `secrets.rotated`

## Capabilities

- Requires: `brainkit.core.secret_store`.
- Uses when present: `brainkit.core.audit_recorder`,
  `brainkit.core.caller_id`, `brainkit.core.refresh_provider_secret`, and
  `brainkit.core.plugin_restarter`.
- Provides: none.

## Runtime resources

None. The secret store is owned by the Kit; this module owns secret admin
command handlers and emits secret lifecycle events. Provider key rotation uses
the typed `brainkit.core.refresh_provider_secret` capability when available; if
provider registry or runtime cache refresh fails, `secrets.rotate` returns that
error instead of reporting a successful rotation with stale runtime state. When
`Restart` is requested, matching plugin restarts are all attempted and any
failed restarts are returned as a joined error with plugin names.

## Hot unmount

Unmounting unregisters `secrets.*` commands and drops optional audit/provider
refresh/plugin restarter references.
