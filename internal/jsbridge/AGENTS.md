# JS Bridge Notes

This subtree owns generic Node/Web compatibility for QuickJS. If a dependency
fails because a runtime API is missing or semantically wrong, start here.

## Runtime Contract

- Polyfills are Go-backed and installed before the agent bundle runs.
- Clean globals are preferred: `stream`, `crypto`, `net`, `dns`, `zlib`, `fs`,
  `process`, and similar Node/Web names.
- `crypto` merges WebCrypto and Node-style crypto on the same object.
- Long-lived work must be bridge-tracked with cancellation and resource
  counters.
- Diagnostics should include owner, phase, source, safe bridge snapshot, and JS
  stack/cause when available.

## Boundaries

- Client-side `http`/`https` request/get is supported over fetch.
- Server listeners are not generic jsbridge behavior. They belong to
  `modules/gateway` or a future server-listener profile.
- Workers are an explicit unsupported boundary unless a real worker owner is
  designed.
- Native addons are unsupported or package-patched. Do not fake them with
  empty objects.
- Async context support covers tested Brainkit-owned callback boundaries. Do
  not claim full Node promise-hook semantics without adding focused proof.

## Tests

For changed polyfills, add focused Go tests and update conformance packs when
the surface is general. For resource-owning behavior, add close/cancel or
resource-drain coverage.

Useful gates:

```sh
go test ./internal/jsbridge -count=1 -timeout=600s
make jsbridge-compat-check
make jsbridge-lifecycle-check
```
