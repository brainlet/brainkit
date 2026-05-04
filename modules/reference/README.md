# modules/reference - stable

Embedded reference corpus command surface. This module exposes compiled-in
reference documents over the bus without exposing the catalog implementation.

## Bus commands

- `kit.reference` - fetch one reference by name.
- `kit.reference.list` - list reference entries and metadata.

## Capabilities

- Requires: `brainkit.core.reference_catalog`.
- Provides: none.

## Runtime resources

None. The reference catalog is owned by the Kit; this module owns only command
handlers.

## Hot unmount

Unmounting unregisters both handlers and drops the catalog reference.
