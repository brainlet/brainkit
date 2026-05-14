# Brainkit Agent Notes

Read `CLAUDE.md` first. This file records the current Mastra / Node ecosystem
compatibility contract for agents working in this repo.

## Compatibility Shape

Brainkit is not pretending QuickJS is Node. The accepted standard is:

- every imported package runs on a declared Brainkit-owned compatibility
  surface;
- or it has a version-scoped package patch with tests;
- or it fails with an explicit typed unsupported diagnostic;
- or it is moved behind an intentional external/native/runtime boundary.

The main ownership split is:

- `internal/jsbridge`: generic Node/Web runtime behavior.
- `internal/embed/agent`: curated Mastra / AI SDK bundle, package patches,
  dynamic require policy, bytecode, and compatibility metadata.
- `internal/jsruntime`: SES Compartment deployment, raw TypeScript source
  preparation, normalized JS artifacts, JS bus bridge behavior.
- `modules/packages`: package command surface and source-relative package
  bundling.
- `fixtures/ts` and `test/fixtures`: runtime proof for deployed TS packages.
- `examples`: user-facing smoke proof, including live/provider and
  external-service paths.

## Hard Boundaries

- Do not add hidden empty-object fallbacks for unknown dynamic require.
- Do not turn `modules/packages/bundlers/esbuild` into accidental arbitrary npm
  support.
- Do not put Node/Web implementation logic in agent bundle stubs. Stubs are
  mechanical re-exports from `globalThis`.
- Do not treat native addons, workers, or server listeners as solved by shape
  stubs. They need explicit owners or typed unsupported boundaries.
- Do not remove raw `.ts` deployment support. It is a Brainkit feature and is
  separate from the embedded Mastra bundle.

## Current Caveats

- Default/source package deployment is `source-relative`, not broad npm
  resolution.
- Async context and diagnostics semantics cover Brainkit-owned callback
  boundaries, not full Node promise-hook semantics.
- OTel active span propagation is still not complete runtime behavior.
- Native addons, workers, and server listeners remain explicit boundaries.
- The next major design item is a future opt-in npm ecosystem resolver profile.

## Validation

For compatibility work, prefer these gates by scope:

- `make agent-embed-rebuild-check`
- `make jsbridge-compat-check`
- `make jsbridge-lifecycle-check`
- `make deps-profile-check`
- `go test ./... -run '^$' -count=1 -timeout=600s`
- `make type-check`
- `make examples-smoke-all`

The nested SDK module is not covered by root `go test ./...`; run this when
SDK files change:

```sh
cd sdk && go test ./... -count=1 -timeout=600s
```

Use the root `.env` for live/provider examples and the `brainkit` Podman
machine for external-service examples.
