# Concept Docs Notes

These docs are part of the compatibility contract. Keep them aligned with code
and checked artifacts when changing runtime behavior.

## Required Alignment

- `bundle-and-bytecode.md` should describe the curated agent bundle, bytecode,
  compatibility artifacts, resolver products, and upgrade workflow.
- `jsbridge-polyfills.md` should describe Node/Web runtime surfaces,
  unsupported boundaries, diagnostics, resource accounting, and dependency
  failure triage.
- `deployment-pipeline.md` should stay aligned with runtime source/artifact
  deployment behavior.
- `bus-and-messaging.md` should stay aligned with request/reply and streaming
  caller semantics.

## Documentation Standard

Do not document aspiration as support. If behavior is partial, unsupported, or
package-patched, say so directly and point to the owner and validation gate.

When docs mention package resolution, preserve the three-product split:

- agent embed curated bundle;
- default/source-relative package deployment;
- future opt-in npm ecosystem resolver profile.
