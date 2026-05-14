# Ecosystem Fixtures

These fixtures cover the boundary between current source package deployment and
future broad npm resolution.

Default/source package deployment is intentionally source-relative today. Bare
npm imports should fail clearly until Brainkit adds a named resolver profile.

Negative resolver-policy fixtures may be excluded from the global TypeScript
fixture type-check because their import graph is intentionally unsupported.
