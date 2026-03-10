package compiler

import (
	"encoding/binary"
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/flow"
	"github.com/brainlet/brainkit/wasm-kit/module"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

func compilerTestSource() *ast.Source {
	return ast.NewSource(ast.SourceKindUserEntry, "compiler-test.ts", "class Test {}")
}

func compilerTestRange(src *ast.Source, start, end int32) diagnostics.Range {
	return diagnostics.Range{Start: start, End: end, Source: src}
}

func compilerTestNamedType(name string, rng diagnostics.Range) *ast.NamedTypeNode {
	return ast.NewNamedTypeNode(ast.NewSimpleTypeName(name, rng), nil, false, rng)
}

func compilerTestProgram() (*program.Program, *Compiler) {
	opts := program.NewOptions()
	prog := program.NewProgram(opts, nil)
	prog.Initialize()
	// Clear any diagnostics emitted during initialization so tests
	// can check only their own diagnostic output.
	prog.DiagnosticEmitter.Diagnostics = prog.DiagnosticEmitter.Diagnostics[:0]
	c := NewCompiler(prog)
	c.DiagnosticEmitter.Diagnostics = c.DiagnosticEmitter.Diagnostics[:0]
	return prog, c
}

func compilerTestClass(prog *program.Program, src *ast.Source, name string) (*program.ClassPrototype, *program.Class) {
	rng := compilerTestRange(src, 0, int32(len(name)))
	classDecl := ast.NewClassDeclaration(
		ast.NewIdentifierExpression(name, rng, false),
		nil,
		0,
		nil,
		nil,
		nil,
		nil,
		rng,
	)
	classProto := program.NewClassPrototype(name, prog.NativeFile, classDecl, 0, false)
	classInstance := program.NewClass(name, classProto, nil, false)
	return classProto, classInstance
}

func compilerTestField(
	prog *program.Program,
	classProto *program.ClassPrototype,
	classInstance *program.Class,
	name string,
	fieldType *types.Type,
	flags common.CommonFlags,
	parameterIndex int32,
	initializer ast.Node,
	offset int32,
	rng diagnostics.Range,
) (*program.PropertyPrototype, *program.Property) {
	fieldDecl := ast.NewFieldDeclaration(
		ast.NewIdentifierExpression(name, rng, false),
		nil,
		int32(flags|common.CommonFlagsInstance),
		compilerTestNamedType(fieldType.KindToString(), rng),
		initializer,
		rng,
	)
	fieldDecl.ParameterIndex = parameterIndex

	prototype := program.PropertyPrototypeForField(name, classProto, fieldDecl, 0)
	classProto.AddInstance(name, prototype)

	boundPrototype := prototype.ToBound(classInstance)
	property := program.NewProperty(boundPrototype, classInstance)
	property.SetType(fieldType)
	property.MemoryOffset = offset
	boundPrototype.PropertyInstance = property

	setterSignature := types.CreateSignature(
		prog,
		[]*types.Type{fieldType},
		types.TypeVoid,
		classInstance.GetResolvedType(),
		1,
		false,
	)
	property.SetterInstance = program.NewFunction(common.SETTER_PREFIX+name, boundPrototype.SetterPrototype, nil, setterSignature, nil)
	property.GetterInstance = program.NewFunction(common.GETTER_PREFIX+name, boundPrototype.GetterPrototype, nil, types.CreateSignature(
		prog,
		nil,
		fieldType,
		classInstance.GetResolvedType(),
		0,
		false,
	), nil)

	return boundPrototype, property
}

func compilerTestConstructor(
	prog *program.Program,
	classInstance *program.Class,
	paramNames []string,
	paramTypes []*types.Type,
	rng diagnostics.Range,
) *program.Function {
	params := make([]*ast.ParameterNode, len(paramNames))
	for i, name := range paramNames {
		params[i] = ast.NewParameterNode(ast.ParameterKindDefault, ast.NewIdentifierExpression(name, rng, false), compilerTestNamedType(paramTypes[i].KindToString(), rng), nil, rng)
	}
	signatureNode := ast.NewFunctionTypeNode(params, ast.NewOmittedType(rng), nil, false, rng)
	ctorDecl := ast.NewMethodDeclaration(
		ast.NewConstructorExpression(rng),
		nil,
		int32(common.CommonFlagsConstructor|common.CommonFlagsInstance),
		nil,
		signatureNode,
		nil,
		rng,
	)
	ctorPrototype := program.NewFunctionPrototype(common.CommonNameConstructor, classInstance, ctorDecl, 0)
	ctorSignature := types.CreateSignature(
		prog,
		paramTypes,
		classInstance.GetResolvedType(),
		classInstance.GetResolvedType(),
		int32(len(paramTypes)),
		false,
	)
	ctor := program.NewFunction(common.CommonNameConstructor, ctorPrototype, nil, ctorSignature, nil)
	classInstance.ConstructorInstance = ctor
	return ctor
}

func TestCheckFieldInitializationReportsMissingOwnField(t *testing.T) {
	prog, c := compilerTestProgram()
	src := compilerTestSource()
	classProto, classInstance := compilerTestClass(prog, src, "MissingInit")
	_, property := compilerTestField(
		prog,
		classProto,
		classInstance,
		"value",
		classInstance.GetResolvedType(),
		0,
		-1,
		nil,
		0,
		compilerTestRange(src, 0, 5),
	)
	classInstance.SetMembers(map[string]program.DeclaredElement{
		property.GetName(): property.Prototype,
	})
	ctor := compilerTestConstructor(prog, classInstance, nil, nil, compilerTestRange(src, 10, 20))
	ctor.Flow.InitThisFieldFlags()
	c.CurrentFlow = ctor.Flow

	reportNode := ast.NewIntegerLiteralExpression(1, compilerTestRange(src, 20, 21))
	c.checkFieldInitialization(classInstance, reportNode)
	c.checkFieldInitialization(classInstance, reportNode)

	if got := len(c.Diagnostics); got != 1 {
		t.Fatalf("len(c.Diagnostics) = %d, want 1", got)
	}
	diag := c.Diagnostics[0]
	if diag.Code != int32(diagnostics.DiagnosticCodeProperty0HasNoInitializerAndIsNotAssignedInTheConstructorBeforeThisIsUsedOrReturned) {
		t.Fatalf("diag.Code = %d, want %d", diag.Code, diagnostics.DiagnosticCodeProperty0HasNoInitializerAndIsNotAssignedInTheConstructorBeforeThisIsUsedOrReturned)
	}
	if diag.Category != diagnostics.DiagnosticCategoryError {
		t.Fatalf("diag.Category = %v, want error", diag.Category)
	}
	if diag.Range == nil || !diag.Range.Equals(property.IdentifierNode().GetRange()) {
		t.Fatalf("diag.Range = %#v, want property identifier range", diag.Range)
	}
	if diag.RelatedRange == nil || !diag.RelatedRange.Equals(reportNode.GetRange()) {
		t.Fatalf("diag.RelatedRange = %#v, want report node range", diag.RelatedRange)
	}
}

func TestCheckFieldInitializationWarnsForRedundantDefiniteAssignmentReference(t *testing.T) {
	prog, c := compilerTestProgram()
	src := compilerTestSource()
	classProto, classInstance := compilerTestClass(prog, src, "WarnInit")
	_, property := compilerTestField(
		prog,
		classProto,
		classInstance,
		"ref",
		classInstance.GetResolvedType(),
		common.CommonFlagsDefinitelyAssigned,
		-1,
		nil,
		0,
		compilerTestRange(src, 0, 3),
	)
	classInstance.SetMembers(map[string]program.DeclaredElement{
		property.GetName(): property.Prototype,
	})
	ctor := compilerTestConstructor(prog, classInstance, nil, nil, compilerTestRange(src, 10, 20))
	ctor.Flow.InitThisFieldFlags()
	ctor.Flow.SetThisFieldFlag(property, flow.FieldFlagInitialized)
	c.CurrentFlow = ctor.Flow

	c.checkFieldInitialization(classInstance, nil)

	if got := len(c.Diagnostics); got != 1 {
		t.Fatalf("len(c.Diagnostics) = %d, want 1", got)
	}
	diag := c.Diagnostics[0]
	if diag.Code != int32(diagnostics.DiagnosticCodeProperty0IsAlwaysAssignedBeforeBeingUsed) {
		t.Fatalf("diag.Code = %d, want %d", diag.Code, diagnostics.DiagnosticCodeProperty0IsAlwaysAssignedBeforeBeingUsed)
	}
	if diag.Category != diagnostics.DiagnosticCategoryWarning {
		t.Fatalf("diag.Category = %v, want warning", diag.Category)
	}
}

func TestCheckFieldInitializationPedanticForRedundantDefiniteAssignmentValue(t *testing.T) {
	prog, c := compilerTestProgram()
	src := compilerTestSource()
	classProto, classInstance := compilerTestClass(prog, src, "PedanticInit")
	_, property := compilerTestField(
		prog,
		classProto,
		classInstance,
		"count",
		types.TypeI32,
		common.CommonFlagsDefinitelyAssigned,
		-1,
		nil,
		0,
		compilerTestRange(src, 0, 5),
	)
	classInstance.SetMembers(map[string]program.DeclaredElement{
		property.GetName(): property.Prototype,
	})
	ctor := compilerTestConstructor(prog, classInstance, nil, nil, compilerTestRange(src, 10, 20))
	ctor.Flow.InitThisFieldFlags()
	c.CurrentFlow = ctor.Flow

	c.checkFieldInitialization(classInstance, nil)

	if got := len(c.Diagnostics); got != 1 {
		t.Fatalf("len(c.Diagnostics) = %d, want 1", got)
	}
	diag := c.Diagnostics[0]
	if diag.Code != int32(diagnostics.DiagnosticCodeUnnecessaryDefiniteAssignment) {
		t.Fatalf("diag.Code = %d, want %d", diag.Code, diagnostics.DiagnosticCodeUnnecessaryDefiniteAssignment)
	}
	if diag.Category != diagnostics.DiagnosticCategoryPedantic {
		t.Fatalf("diag.Category = %v, want pedantic", diag.Category)
	}
}

func TestMakeFieldInitializationInConstructorOrdersParameterFieldsFirst(t *testing.T) {
	prog, c := compilerTestProgram()
	src := compilerTestSource()
	classProto, classInstance := compilerTestClass(prog, src, "CtorInit")

	firstRange := compilerTestRange(src, 0, 5)
	secondRange := compilerTestRange(src, 6, 15)
	thirdRange := compilerTestRange(src, 16, 21)

	firstProto, firstProperty := compilerTestField(
		prog,
		classProto,
		classInstance,
		"later",
		types.TypeI32,
		0,
		-1,
		ast.NewIntegerLiteralExpression(7, firstRange),
		0,
		firstRange,
	)
	paramProto, paramProperty := compilerTestField(
		prog,
		classProto,
		classInstance,
		"paramField",
		types.TypeI32,
		0,
		0,
		nil,
		4,
		secondRange,
	)
	thirdProto, thirdProperty := compilerTestField(
		prog,
		classProto,
		classInstance,
		"zeroed",
		types.TypeI32,
		0,
		-1,
		nil,
		8,
		thirdRange,
	)

	classInstance.SetMembers(map[string]program.DeclaredElement{
		firstProperty.GetName(): firstProto,
		paramProperty.GetName(): paramProto,
		thirdProperty.GetName(): thirdProto,
	})

	ctor := compilerTestConstructor(prog, classInstance, []string{"paramField"}, []*types.Type{types.TypeI32}, compilerTestRange(src, 22, 32))
	c.CurrentFlow = ctor.Flow

	var stmts []module.ExpressionRef
	c.makeFieldInitializationInConstructor(classInstance, &stmts)

	if got := len(stmts); got != 3 {
		t.Fatalf("len(stmts) = %d, want 3", got)
	}

	wantTargets := []string{
		paramProperty.SetterInstance.GetInternalName(),
		firstProperty.SetterInstance.GetInternalName(),
		thirdProperty.SetterInstance.GetInternalName(),
	}
	for i, stmt := range stmts {
		if got := module.GetExpressionId(stmt); got != module.ExpressionIdCall {
			t.Fatalf("stmt %d expression id = %v, want call", i, got)
		}
		if got := module.GetCallTarget(stmt); got != wantTargets[i] {
			t.Fatalf("stmt %d call target = %q, want %q", i, got, wantTargets[i])
		}
	}

	paramValue := module.GetCallOperandAt(stmts[0], 1)
	if got := module.GetExpressionId(paramValue); got != module.ExpressionIdLocalGet {
		t.Fatalf("parameter field value expression id = %v, want local.get", got)
	}
	if got := module.GetLocalGetIndex(paramValue); got != 1 {
		t.Fatalf("parameter field local index = %d, want 1", got)
	}

	initValue := module.GetCallOperandAt(stmts[1], 1)
	if got := module.GetExpressionId(initValue); got != module.ExpressionIdConst {
		t.Fatalf("initialized field expression id = %v, want const", got)
	}
	if got := module.GetConstValueI32(initValue); got != 7 {
		t.Fatalf("initialized field const = %d, want 7", got)
	}

	zeroValue := module.GetCallOperandAt(stmts[2], 1)
	if got := module.GetExpressionId(zeroValue); got != module.ExpressionIdConst {
		t.Fatalf("zero field expression id = %v, want const", got)
	}
	if got := module.GetConstValueI32(zeroValue); got != 0 {
		t.Fatalf("zero field const = %d, want 0", got)
	}
}

func TestCompileRTTICreatesRuntimeTypeInfoSegment(t *testing.T) {
	prog, c := compilerTestProgram()
	src := compilerTestSource()

	_, pointerfreeClass := compilerTestClass(prog, src, "Pointerfree")

	holderProto, holderClass := compilerTestClass(prog, src, "Holder")
	holderFieldProto, holderField := compilerTestField(
		prog,
		holderProto,
		holderClass,
		"ref",
		pointerfreeClass.GetResolvedType(),
		0,
		-1,
		nil,
		0,
		compilerTestRange(src, 0, 3),
	)
	holderClass.SetMembers(map[string]program.DeclaredElement{
		holderField.GetName(): holderFieldProto,
	})

	compileRTTI(c)

	if got := len(c.MemorySegments); got != 1 {
		t.Fatalf("len(c.MemorySegments) = %d, want 1", got)
	}

	segment := c.MemorySegments[0]
	if got := binary.LittleEndian.Uint32(segment.Buffer[0:4]); got != 2 {
		t.Fatalf("rtti count = %d, want 2", got)
	}

	firstFlags := common.TypeinfoFlags(binary.LittleEndian.Uint32(segment.Buffer[4:8]))
	secondFlags := common.TypeinfoFlags(binary.LittleEndian.Uint32(segment.Buffer[8:12]))
	if firstFlags&common.TypeinfoFlagsPOINTERFREE == 0 {
		t.Fatalf("first class flags = %v, want POINTERFREE", firstFlags)
	}
	if secondFlags&common.TypeinfoFlagsPOINTERFREE != 0 {
		t.Fatalf("second class flags = %v, want non-pointerfree", secondFlags)
	}
	if got := pointerfreeClass.RttiFlags; got != uint32(firstFlags) {
		t.Fatalf("pointerfreeClass.RttiFlags = %d, want %d", got, firstFlags)
	}
	if got := holderClass.RttiFlags; got != uint32(secondFlags) {
		t.Fatalf("holderClass.RttiFlags = %d, want %d", got, secondFlags)
	}

	rttiBase := c.Module().GetGlobal(common.BuiltinNameRttiBase)
	if rttiBase == 0 {
		t.Fatal("missing __rtti_base global")
	}
	initExpr := module.GetGlobalInit(rttiBase)
	if got := module.GetConstValueInteger(initExpr, c.Options().IsWasm64()); got != module.GetConstValueInteger(segment.Offset, c.Options().IsWasm64()) {
		t.Fatalf("__rtti_base = %d, want %d", got, module.GetConstValueInteger(segment.Offset, c.Options().IsWasm64()))
	}
}

func compilerTestRegisterVisitRuntime(prog *program.Program) *program.Function {
	signature := types.CreateSignature(prog, []*types.Type{types.TypeU32}, types.TypeVoid, prog.Options.UsizeType(), 1, false)
	fn := prog.MakeNativeFunction(common.CommonNameVisit, signature, nil, common.CommonFlagsAmbient, 0)
	fn.SetInternalName(common.CommonNameVisit)
	fn.Prototype.SetInternalName(common.CommonNameVisit)
	prog.ElementsByNameMap[common.CommonNameVisit] = fn.Prototype
	prog.InstancesByNameMap[common.CommonNameVisit] = fn
	return fn
}

func hasCallTarget(expr module.ExpressionRef, target string) bool {
	switch module.GetExpressionId(expr) {
	case module.ExpressionIdCall:
		return module.GetCallTarget(expr) == target
	case module.ExpressionIdBlock:
		for i := module.Index(0); i < module.GetBlockChildCount(expr); i++ {
			if hasCallTarget(module.GetBlockChildAt(expr, i), target) {
				return true
			}
		}
	case module.ExpressionIdIf:
		if hasCallTarget(module.GetIfCondition(expr), target) {
			return true
		}
		if hasCallTarget(module.GetIfTrue(expr), target) {
			return true
		}
		if falseExpr := module.GetIfFalse(expr); falseExpr != 0 && hasCallTarget(falseExpr, target) {
			return true
		}
	}
	return false
}

func TestCompileVisitGlobalsEmitsManagedGlobalVisitorCalls(t *testing.T) {
	prog, c := compilerTestProgram()
	src := compilerTestSource()
	visitRuntime := compilerTestRegisterVisitRuntime(prog)

	_, managedClass := compilerTestClass(prog, src, "ManagedGlobal")

	globalDecl := ast.NewVariableDeclaration(
		ast.NewIdentifierExpression("g", compilerTestRange(src, 0, 1), false),
		nil,
		0,
		compilerTestNamedType("ManagedGlobal", compilerTestRange(src, 0, 1)),
		nil,
		compilerTestRange(src, 0, 1),
	)
	global := program.NewGlobal("g", prog.NativeFile, 0, globalDecl)
	global.SetType(managedClass.GetResolvedType())
	global.Set(common.CommonFlagsCompiled)
	prog.ElementsByNameMap[global.GetInternalName()] = global

	compileVisitGlobals(c)

	fn := c.Module().GetFunction(common.BuiltinNameVisitGlobals)
	if fn == 0 {
		t.Fatal("missing __visit_globals helper")
	}
	body := c.Module().GetFunctionBody(fn)
	if !hasCallTarget(body, visitRuntime.GetInternalName()) {
		t.Fatalf("__visit_globals body does not call %q", visitRuntime.GetInternalName())
	}
}

func TestCompileVisitMembersCreatesVisitorHelpers(t *testing.T) {
	prog, c := compilerTestProgram()
	src := compilerTestSource()
	visitRuntime := compilerTestRegisterVisitRuntime(prog)

	_, refClass := compilerTestClass(prog, src, "RefValue")
	holderProto, holderClass := compilerTestClass(prog, src, "HolderVisit")
	fieldProto, field := compilerTestField(
		prog,
		holderProto,
		holderClass,
		"ref",
		refClass.GetResolvedType(),
		0,
		-1,
		nil,
		0,
		compilerTestRange(src, 0, 3),
	)
	holderClass.SetMembers(map[string]program.DeclaredElement{
		field.GetName(): fieldProto,
	})

	compileVisitMembers(c)

	if holderClass.VisitRef == 0 {
		t.Fatal("expected class visitor helper to be emitted")
	}

	visitFn := c.Module().GetFunction(common.BuiltinNameVisitMembers)
	if visitFn == 0 {
		t.Fatal("missing __visit_members helper")
	}
	if !hasCallTarget(c.Module().GetFunctionBody(visitFn), holderClass.GetInternalName()+"~visit") {
		t.Fatalf("__visit_members does not dispatch to %q", holderClass.GetInternalName()+"~visit")
	}
	if !hasCallTarget(c.Module().GetFunctionBody(holderClass.VisitRef), visitRuntime.GetInternalName()) {
		t.Fatalf("%q helper does not call %q", holderClass.GetInternalName()+"~visit", visitRuntime.GetInternalName())
	}
}
