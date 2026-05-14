# Agent Embed Notes

This subtree owns the curated Mastra / AI SDK runtime bundle. It is not the
user package resolver.

## Ownership

- `bundle/package.json` and `bundle/pnpm-lock.yaml` are the curated npm input.
- `bundle/build.mjs` builds the single agent embed bundle.
- `agent_embed_bundle.js` and `agent_embed_bundle.bc` are checked runtime
  artifacts. Bytecode is loaded before JS source.
- `bundle/compat/manifest.json` declares Node/Web/package surfaces.
- `bundle/compat/report.json` and `bundle/compat/inventory.json` are generated
  review artifacts.
- `bundle/compat/node-api-target.json` records supported and unsupported
  runtime semantics.
- `bundle/compat/capability-matrix.json` records Mastra feature proof levels.

## Rules

- Unknown dynamic require must throw `BrainkitUnsupportedDynamicRequireError`.
- Bundle stubs must be thin re-exports from `globalThis`. Put behavior in
  `internal/jsbridge`, not in `build.mjs`.
- Package-specific quirks belong in manifest package-patch rows with version
  ranges, tests, and removal conditions.
- Native addons, workers, server listeners, optional native peers, and external
  service paths need explicit boundary metadata.
- If `build.mjs`, dependencies, package patches, or entry exports change,
  rebuild both JS and bytecode.

## Rebuild And Check

Use:

```sh
make agent-embed-rebuild
make jsbridge-compat-report-save
make jsbridge-compat-inventory-save
make agent-embed-rebuild-check
```

Before accepting a dependency upgrade, inspect:

- direct package version changes;
- new transitive packages;
- new Node/Web APIs;
- new dynamic requires;
- package patch hits and misses;
- bundle and bytecode size changes.

No `package-lock.json`, `npm-shrinkwrap.json`, or `yarn.lock` belongs in
`internal/embed/agent/bundle`.
