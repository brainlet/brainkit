// Ported from: assemblyscript/src/compiler.ts (lines 226-425)
package compiler

import "github.com/brainlet/brainkit/wasm-kit/common"

// DefaultFeatures are the features enabled by default.
const DefaultFeatures = common.FeatureMutableGlobals |
	common.FeatureSignExtension |
	common.FeatureNontrappingF2I |
	common.FeatureBulkMemory

// UncheckedBehavior controls how unchecked operations are handled.
type UncheckedBehavior int32

const (
	// UncheckedBehaviorDefault only uses unchecked operations inside unchecked().
	UncheckedBehaviorDefault UncheckedBehavior = 0
	// UncheckedBehaviorNever never uses unchecked operations.
	UncheckedBehaviorNever UncheckedBehavior = 1
	// UncheckedBehaviorAlways always uses unchecked operations if possible.
	UncheckedBehaviorAlways UncheckedBehavior = 2
)

// Constraints are various constraints in expression compilation.
type Constraints int32

const (
	ConstraintsNone        Constraints = 0
	ConstraintsConvImplicit Constraints = 1 << 0 // Must implicitly convert to the target type.
	ConstraintsConvExplicit Constraints = 1 << 1 // Must explicitly convert to the target type.
	ConstraintsMustWrap     Constraints = 1 << 2 // Must wrap small integer values to match the target type.
	ConstraintsWillDrop     Constraints = 1 << 3 // Indicates that the value will be dropped immediately.
	ConstraintsPreferStatic Constraints = 1 << 4 // Indicates that static data is preferred.
	ConstraintsIsThis       Constraints = 1 << 5 // Indicates the value will become `this` of a property access or instance call.
)

// RuntimeFeatures are runtime features to be activated by the compiler.
type RuntimeFeatures int32

const (
	RuntimeFeaturesNone               RuntimeFeatures = 0
	RuntimeFeaturesData               RuntimeFeatures = 1 << 0 // Requires data setup.
	RuntimeFeaturesStack              RuntimeFeatures = 1 << 1 // Requires a stack.
	RuntimeFeaturesHeap               RuntimeFeatures = 1 << 2 // Requires heap setup.
	RuntimeFeaturesRtti               RuntimeFeatures = 1 << 3 // Requires runtime type information setup.
	RuntimeFeaturesVisitGlobals       RuntimeFeatures = 1 << 4 // Requires the built-in globals visitor.
	RuntimeFeaturesVisitMembers       RuntimeFeatures = 1 << 5 // Requires the built-in members visitor.
	RuntimeFeaturesSetArgumentsLength RuntimeFeatures = 1 << 6 // Requires the setArgumentsLength export.
)

// ImportNames are imported default names of compiler-generated elements.
const (
	ImportNameDefaultNamespace = "env"
	ImportNameMemory           = "memory"
	ImportNameTable            = "table"
)

// ExportNames are exported names of compiler-generated elements.
const (
	ExportNameMemory             = "memory"
	ExportNameTable              = "table"
	ExportNameArgumentsLength    = "__argumentsLength"
	ExportNameSetArgumentsLength = "__setArgumentsLength"
)

// RuntimeFunctionNames are functions to export if --exportRuntime is set.
var RuntimeFunctionNames = []string{"__new", "__pin", "__unpin", "__collect"}

// RuntimeGlobalNames are globals to export if --exportRuntime is set.
var RuntimeGlobalNames = []string{"__rtti_base"}
