# Environment

This file documents the runtime and development environment for brainkit mission workers.

## Podman

brainkit uses a **dedicated Podman machine named `brainkit`** that is isolated from any other project's podman machines. This prevents container/port conflicts and keeps brainkit's testcontainers (pgvector, mongodb, libsql) separate from whatever the user runs on their default machine.

### Resource caps
- **CPUs:** 4
- **Memory:** 8 GiB
- **Disk:** 60 GB

### Lifecycle (Makefile targets)
- `make podman-ensure` — idempotent init + start; sets `brainkit` as the default podman connection. This is the default invocation for workers and is automatically pulled in by `make test`.
- `make podman-down` — stop the `brainkit` machine if it is running.
- `make podman-status` — show the `brainkit` machine state and current default connection.
- `make podman-reset CONFIRM=1` — destroy and recreate the `brainkit` machine. Requires `CONFIRM=1` to prevent accidental data loss.

### Container management
- Workers **do NOT** manage pgvector, mongodb, or libsql containers manually.
- `testcontainers-go` lazy-spawns these containers on top of the `brainkit` machine during `go test ./test/fixtures/...`.
- The only podman surface workers should touch is the machine itself via the `make podman-*` targets.
