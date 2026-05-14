# Ecosystem Fixture Notes

This category proves the boundary between current source package deployment and
future broad npm support.

## Purpose

- Keep representative Node ecosystem package behavior visible through checked
  canaries.
- Keep default/source package deployment honest: arbitrary bare npm imports are
  rejected until a named resolver profile exists.

## Negative Fixtures

`bare-npm-rejected` intentionally imports `uuid` as a bare npm package. It must
fail during deploy with `PACKAGE_RESOLVER_UNSUPPORTED_IMPORT`.

Because that import is intentionally unsupported by the current source-relative
profile, the fixture is excluded from the global TypeScript fixture type-check.
Do not "fix" the type-check by adding a global `uuid` path. That would hide the
resolver boundary.

## Validation

```sh
go test ./test/fixtures -run 'TestFixtures/ecosystem' -count=1 -timeout=600s
go test ./internal/embed/agent -run 'Test.*EcosystemCanary|Test.*Canary' -count=1 -timeout=600s
```
