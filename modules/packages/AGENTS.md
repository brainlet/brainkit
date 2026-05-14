# Packages Module Notes

This module owns package deployment commands and source package preparation.
It is intentionally not broad npm support.

## Current Resolver Profile

The standard esbuild builder is `source-relative`:

- relative `.ts`, `.js`, `.mjs`, and `.json` files in the deployed package are
  supported;
- package-local index files are supported;
- the only accepted bare imports are Brainkit endowments: `kit`, `ai`,
  `agent`, and `compiler`;
- any other bare import must fail with
  `PACKAGE_RESOLVER_UNSUPPORTED_IMPORT`.

The resolver diagnostic must include:

- requested specifier;
- importer and source package;
- profile: `source-relative`;
- allowed bare imports;
- suggested future owner: opt-in npm ecosystem resolver profile.

## Future Resolver Profile

Do not silently add arbitrary npm resolution here. A future profile needs its
own design for install/cache policy, resolver traces, package patch ownership,
native/worker rejection, lifecycle cleanup, and validation.

## Tests

Useful gates:

```sh
go test ./modules/packages ./modules/packages/bundlers/esbuild ./modules/packages/source -run 'Test.*Bundle|Test.*PackageDeploy|Test.*TypeScript|Test.*BareImport|Test.*Resolver|TestBuilder' -count=1 -timeout=600s
go test ./test/fixtures -run 'TestFixtures/ecosystem' -count=1 -timeout=600s
```
