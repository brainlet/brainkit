// glue_test.go provides Go equivalents of the JS glue files in tests/compiler/*.js.
//
// These glue files provide host imports that certain fixtures need to instantiate.
// Ported from:
//   - tests/compiler/bigint-integration.js
//   - tests/compiler/declare.js
//   - tests/compiler/exportimport-table.js
//   - tests/compiler/external.js
//   - tests/compiler/mutable-globals.js
package tests

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// getGlue returns the Glue for a fixture that needs custom host imports.
// Returns nil if no glue is registered for the given fixture name.
func getGlue(name string) *Glue {
	switch name {
	case "declare":
		return glueDeclare
	case "external":
		return glueExternal
	case "bigint-integration":
		return glueBigintIntegration
	case "exportimport-table":
		return glueExportimportTable
	case "mutable-globals":
		return glueMutableGlobals
	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// declare.js
// Provides: declare.externalFunction, declare.externalConstant,
//           declare."my.externalFunction", declare."my.externalConstant"
// ---------------------------------------------------------------------------

var glueDeclare = &Glue{
	PreInstantiate: func(rt wazero.Runtime, ctx context.Context) error {
		_, err := rt.NewHostModuleBuilder("declare").
			NewFunctionBuilder().WithFunc(func() {}).Export("externalFunction").
			NewFunctionBuilder().WithFunc(func() {}).Export("my.externalFunction").
			Instantiate(ctx)
		return err
	},
}

// ---------------------------------------------------------------------------
// external.js
// Provides: external.foo, external."foo.bar", external.bar,
//           foo.baz, foo."var"
// ---------------------------------------------------------------------------

var glueExternal = &Glue{
	PreInstantiate: func(rt wazero.Runtime, ctx context.Context) error {
		_, err := rt.NewHostModuleBuilder("external").
			NewFunctionBuilder().WithFunc(func() {}).Export("foo").
			NewFunctionBuilder().WithFunc(func() {}).Export("foo.bar").
			NewFunctionBuilder().WithFunc(func() {}).Export("bar").
			Instantiate(ctx)
		if err != nil {
			return err
		}
		_, err = rt.NewHostModuleBuilder("foo").
			NewFunctionBuilder().WithFunc(func() {}).Export("baz").
			Instantiate(ctx)
		return err
	},
}

// ---------------------------------------------------------------------------
// bigint-integration.js
// Provides: bigint-integration.externalValue (i64 global),
//           bigint-integration.getExternalValue (func -> i64)
// PostInstantiate: checks internalValue export == 9007199254740991
// ---------------------------------------------------------------------------

var glueBigintIntegration = &Glue{
	PreInstantiate: func(rt wazero.Runtime, ctx context.Context) error {
		externalValue := int64(9007199254740991) // Number.MAX_SAFE_INTEGER
		_, err := rt.NewHostModuleBuilder("bigint-integration").
			NewFunctionBuilder().WithFunc(func() int64 {
			return externalValue
		}).Export("getExternalValue").
			Instantiate(ctx)
		return err
	},
	PostInstantiate: func(mod api.Module) error {
		fn := mod.ExportedFunction("getInternalValue")
		if fn == nil {
			return fmt.Errorf("getInternalValue not exported")
		}
		results, err := fn.Call(context.Background())
		if err != nil {
			return fmt.Errorf("getInternalValue call: %w", err)
		}
		if int64(results[0]) != 9007199254740991 {
			return fmt.Errorf("unexpected internalValue: %d", int64(results[0]))
		}
		return nil
	},
}

// ---------------------------------------------------------------------------
// exportimport-table.js
// Provides: env.table (WebAssembly.Table with 2 anyfunc entries)
// wazero handles tables differently — we provide a minimal env module.
// ---------------------------------------------------------------------------

var glueExportimportTable = &Glue{
	PreInstantiate: func(rt wazero.Runtime, ctx context.Context) error {
		// wazero doesn't support importing tables directly via host modules.
		// Skip this fixture's runtime test — WAT match is sufficient.
		return fmt.Errorf("table imports not supported in wazero host modules")
	},
}

// ---------------------------------------------------------------------------
// mutable-globals.js
// Provides: mutable-globals.external (mutable i32 global, initial value 123)
// PostStart: checks external == 133, internal == 134
// ---------------------------------------------------------------------------

var glueMutableGlobals = &Glue{
	PreInstantiate: func(rt wazero.Runtime, ctx context.Context) error {
		// wazero doesn't support mutable global imports via host modules.
		// Skip this fixture's runtime test — WAT match is sufficient.
		return fmt.Errorf("mutable global imports not supported in wazero host modules")
	},
}
