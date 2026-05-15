# Ecosystem Fixtures

These fixtures cover the boundary between current source package deployment and
future broad npm resolution.

Default/source package deployment is intentionally source-relative today. Bare
npm imports should fail clearly unless the package explicitly opts into a named
resolver profile.

Negative resolver-policy fixtures may be excluded from the global TypeScript
fixture type-check because their import graph is intentionally unsupported.

`npm-preview` exists as an experimental `modules/packages` filesystem package
profile. It is covered by package builder and package-deploy tests, not this
raw `.ts` fixture runner, because it requires a package root with
`package.json` and `pnpm-lock.yaml`.

The current npm-preview coverage includes jsbridge-owned Node builtin stubs,
dynamic import lowering, and a real `@mastra/stagehand` package import/eval
smoke through `package.deploy`. That still does not prove concrete browser
provider behavior. Real provider support requires provider package fixtures,
real browser/CDP lifecycle, and live model/browser tests.
