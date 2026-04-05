# Scheduling Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests that assert exact counts (ListSchedules, fire count) use suite.Full(t) for fresh kernels to avoid pollution from the shared env. Schedule topics use unique names to prevent cross-test interference. The drain test uses SetDraining(true) then SetDraining(false) on a fresh kernel.

## Adding a test

1. Add function to fire.go (firing/cancellation) or backend_advanced.go (transport-specific)
2. Register in run.go
3. Update TEST_MAP.md
