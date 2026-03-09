package program

import (
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// --- test helpers ---

func newTestIdent(name string) *ast.IdentifierExpression {
	return ast.NewIdentifierExpression(name, nativeRange(), false)
}

func newTestTypeName(name string) *ast.TypeName {
	return ast.NewSimpleTypeName(name, nativeRange())
}

func newTestNamedTypeNode(name string) *ast.NamedTypeNode {
	return ast.NewNamedTypeNode(newTestTypeName(name), nil, false, nativeRange())
}

func newTestFuncDecl(name string, flags int32, sig *ast.FunctionTypeNode) *ast.FunctionDeclaration {
	return ast.NewFunctionDeclaration(
		newTestIdent(name),
		nil,
		flags,
		nil,
		sig,
		nil,
		ast.ArrowKindNone,
		nativeRange(),
	)
}

func newTestClassDecl(name string, flags int32) *ast.ClassDeclaration {
	return ast.NewClassDeclaration(
		newTestIdent(name),
		nil,
		flags,
		nil, nil, nil, nil,
		nativeRange(),
	)
}

func newTestSig(retType ast.Node) *ast.FunctionTypeNode {
	return ast.NewFunctionTypeNode(nil, retType, nil, false, nativeRange())
}

func newTestSigWithParams(params []*ast.ParameterNode, retType ast.Node) *ast.FunctionTypeNode {
	return ast.NewFunctionTypeNode(params, retType, nil, false, nativeRange())
}

func newTestParam(name string, typNode ast.Node) *ast.ParameterNode {
	return ast.NewParameterNode(ast.ParameterKindDefault, newTestIdent(name), typNode, nil, nativeRange())
}

func newVoidTypeDefinition(prog *Program) *TypeDefinition {
	typeDecl := prog.MakeNativeTypeDeclaration("void", 0)
	td := NewTypeDefinition("void", prog.NativeFile, typeDecl, 0)
	td.SetType(types.TypeVoid)
	return td
}

func newI32TypeDefinition(prog *Program) *TypeDefinition {
	typeDecl := prog.MakeNativeTypeDeclaration("i32", 0)
	td := NewTypeDefinition("i32", prog.NativeFile, typeDecl, 0)
	td.SetType(types.TypeI32)
	return td
}

// registerType registers a type definition in the program for name resolution.
func registerType(prog *Program, name string, td *TypeDefinition) {
	prog.EnsureGlobal(name, td)
}

// --- tests ---

func TestNewResolver(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	if resolver.GetProgram() != prog {
		t.Error("resolver program mismatch")
	}
	if resolver.DiscoveredOverride {
		t.Error("DiscoveredOverride should be false initially")
	}
	if resolver.CurrentThisExpression != nil {
		t.Error("CurrentThisExpression should be nil initially")
	}
	if resolver.CurrentElementExpression != nil {
		t.Error("CurrentElementExpression should be nil initially")
	}
}

func TestReportMode(t *testing.T) {
	if ReportModeReport != 0 {
		t.Errorf("ReportModeReport should be 0, got %d", ReportModeReport)
	}
	if ReportModeSwallow != 1 {
		t.Errorf("ReportModeSwallow should be 1, got %d", ReportModeSwallow)
	}
}

func TestResolveTypeUnsupported(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	// Use a node type that isn't NamedTypeNode or FunctionTypeNode
	node := &ast.NodeBase{}
	result := resolver.ResolveType(node, nil, prog.NativeFile, nil, ReportModeSwallow)
	if result != nil {
		t.Error("ResolveType should return nil for unsupported node type")
	}
}

func TestResolveTypeRecursiveNamed(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	node := newTestNamedTypeNode("i32")
	node.CurrentlyResolving = true

	result := resolver.ResolveType(node, nil, prog.NativeFile, nil, ReportModeSwallow)
	if result != nil {
		t.Error("ResolveType should return nil for recursive type")
	}
}

func TestResolveTypeRecursiveFunction(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	retType := newTestNamedTypeNode("i32")
	node := ast.NewFunctionTypeNode(nil, retType, nil, false, nativeRange())
	node.CurrentlyResolving = true

	result := resolver.ResolveType(node, nil, prog.NativeFile, nil, ReportModeSwallow)
	if result != nil {
		t.Error("ResolveType should return nil for recursive function type")
	}
}

func TestResolveTypeNameNotFound(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	node := newTestNamedTypeNode("NonExistent")
	result := resolver.ResolveType(node, nil, prog.NativeFile, nil, ReportModeSwallow)
	if result != nil {
		t.Error("ResolveType should return nil for unknown type name")
	}
}

func TestResolveTypeNameNotFoundWithReport(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	node := newTestNamedTypeNode("NonExistent")
	resolver.ResolveType(node, nil, prog.NativeFile, nil, ReportModeReport)

	// Should have emitted an error
	diags := prog.DiagnosticEmitter.Diagnostics
	if len(diags) == 0 {
		t.Error("expected an error diagnostic for unknown type name")
	}
}

func TestResolveFunctionPrototype(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	// Register "i32" type
	registerType(prog, "i32", newI32TypeDefinition(prog))

	// Create a simple function: func foo(): i32
	sig := newTestSig(newTestNamedTypeNode("i32"))
	decl := newTestFuncDecl("foo", 0, sig)
	prototype := NewFunctionPrototype("foo", prog.NativeFile, decl, 0)

	instance := resolver.ResolveFunction(prototype, nil, nil, ReportModeReport)
	if instance == nil {
		t.Fatal("ResolveFunction returned nil")
	}
	if instance.GetName() != "foo" {
		t.Errorf("expected name 'foo', got '%s'", instance.GetName())
	}
	if instance.Prototype != prototype {
		t.Error("instance prototype should match original prototype")
	}
	if instance.Signature == nil {
		t.Error("function should have a signature")
	}
	if instance.Signature.ReturnType != types.TypeI32 {
		t.Error("function return type should be i32")
	}
}

func TestResolveFunctionPrototypeCached(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)
	registerType(prog, "i32", newI32TypeDefinition(prog))

	sig := newTestSig(newTestNamedTypeNode("i32"))
	decl := newTestFuncDecl("bar", 0, sig)
	prototype := NewFunctionPrototype("bar", prog.NativeFile, decl, 0)

	first := resolver.ResolveFunction(prototype, nil, nil, ReportModeSwallow)
	second := resolver.ResolveFunction(prototype, nil, nil, ReportModeSwallow)

	if first != second {
		t.Error("resolving the same prototype twice should return the same instance")
	}
}

func TestResolveFunctionWithParams(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	registerType(prog, "i32", newI32TypeDefinition(prog))

	params := []*ast.ParameterNode{
		newTestParam("x", newTestNamedTypeNode("i32")),
		newTestParam("y", newTestNamedTypeNode("i32")),
	}
	sig := newTestSigWithParams(params, newTestNamedTypeNode("i32"))
	decl := newTestFuncDecl("add", 0, sig)
	prototype := NewFunctionPrototype("add", prog.NativeFile, decl, 0)

	instance := resolver.ResolveFunction(prototype, nil, nil, ReportModeReport)
	if instance == nil {
		t.Fatal("ResolveFunction returned nil for function with params")
	}
	if len(instance.Signature.ParameterTypes) != 2 {
		t.Errorf("expected 2 parameter types, got %d", len(instance.Signature.ParameterTypes))
	}
	if instance.Signature.RequiredParameters != 2 {
		t.Errorf("expected 2 required parameters, got %d", instance.Signature.RequiredParameters)
	}
}

func TestResolveClassPrototype(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	decl := newTestClassDecl("Foo", 0)
	prototype := NewClassPrototype("Foo", prog.NativeFile, decl, 0, false)

	instance := resolver.ResolveClass(prototype, nil, nil, ReportModeReport)
	if instance == nil {
		t.Fatal("ResolveClass returned nil")
	}
	if instance.GetName() != "Foo" {
		t.Errorf("expected name 'Foo', got '%s'", instance.GetName())
	}
	if instance.Prototype != prototype {
		t.Error("instance prototype should match original prototype")
	}
	if instance.IsInterface() {
		t.Error("class should not be an interface")
	}
}

func TestResolveClassPrototypeCached(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	decl := newTestClassDecl("Bar", 0)
	prototype := NewClassPrototype("Bar", prog.NativeFile, decl, 0, false)

	first := resolver.ResolveClass(prototype, nil, nil, ReportModeSwallow)
	second := resolver.ResolveClass(prototype, nil, nil, ReportModeSwallow)

	if first != second {
		t.Error("resolving the same prototype twice should return the same instance")
	}
}

func TestResolveInterfacePrototype(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	decl := newTestClassDecl("IFoo", 0)
	prototype := NewInterfacePrototype("IFoo", prog.NativeFile, decl, 0)

	instance := resolver.ResolveClass(&prototype.ClassPrototype, nil, nil, ReportModeReport)
	if instance == nil {
		t.Fatal("ResolveClass returned nil for interface")
	}
	if !instance.IsInterface() {
		t.Error("interface should be marked as interface")
	}
	iface := instance.AsInterface()
	if iface == nil {
		t.Error("AsInterface should return non-nil for interface")
	}
}

func TestResolveClassWithBase(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	baseDecl := newTestClassDecl("Base", 0)
	baseProto := NewClassPrototype("Base", prog.NativeFile, baseDecl, 0, false)

	derivedDecl := newTestClassDecl("Derived", 0)
	derivedProto := NewClassPrototype("Derived", prog.NativeFile, derivedDecl, 0, false)
	derivedProto.BasePrototype = baseProto

	derived := resolver.ResolveClass(derivedProto, nil, nil, ReportModeReport)
	if derived == nil {
		t.Fatal("ResolveClass returned nil for derived class")
	}
	if derived.Base == nil {
		t.Error("derived class should have a base")
	}
	if derived.Base.GetName() != "Base" {
		t.Errorf("base class name should be 'Base', got '%s'", derived.Base.GetName())
	}
}

func TestResolveClassCircularInheritance(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	declA := newTestClassDecl("A", 0)
	protoA := NewClassPrototype("A", prog.NativeFile, declA, 0, false)

	declB := newTestClassDecl("B", 0)
	protoB := NewClassPrototype("B", prog.NativeFile, declB, 0, false)

	// Circular: A extends B, B extends A
	protoA.BasePrototype = protoB
	protoB.BasePrototype = protoA

	result := resolver.ResolveClass(protoA, nil, nil, ReportModeSwallow)
	if result != nil {
		t.Error("ResolveClass should return nil for circular inheritance")
	}
}

func TestResolveProperty(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)
	registerType(prog, "void", newVoidTypeDefinition(prog))

	classDecl := newTestClassDecl("MyClass", 0)
	classProto := NewClassPrototype("MyClass", prog.NativeFile, classDecl, 0, false)
	classInstance := resolver.ResolveClass(classProto, nil, nil, ReportModeReport)
	if classInstance == nil {
		t.Fatal("ResolveClass returned nil")
	}

	// Create getter prototype
	getterSig := newTestSig(newTestNamedTypeNode("void"))
	getterDecl := newTestFuncDecl("value", int32(common.CommonFlagsInstance|common.CommonFlagsGet), getterSig)

	pp := NewPropertyPrototype("value", classInstance, getterDecl)
	pp.GetterPrototype = NewFunctionPrototype(common.GETTER_PREFIX+"value", classInstance, getterDecl, 0)

	property := resolver.ResolveProperty(pp, ReportModeReport)
	if property == nil {
		t.Fatal("ResolveProperty returned nil")
	}
	if property.Prototype != pp {
		t.Error("property prototype should match")
	}
}

func TestResolvePropertyCached(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)
	registerType(prog, "void", newVoidTypeDefinition(prog))

	classDecl := newTestClassDecl("MyClass", 0)
	classProto := NewClassPrototype("MyClass", prog.NativeFile, classDecl, 0, false)
	classInstance := resolver.ResolveClass(classProto, nil, nil, ReportModeReport)
	if classInstance == nil {
		t.Fatal("ResolveClass returned nil")
	}

	getterSig := newTestSig(newTestNamedTypeNode("void"))
	getterDecl := newTestFuncDecl("val", int32(common.CommonFlagsInstance|common.CommonFlagsGet), getterSig)
	pp := NewPropertyPrototype("val", classInstance, getterDecl)

	first := resolver.ResolveProperty(pp, ReportModeSwallow)
	second := resolver.ResolveProperty(pp, ReportModeSwallow)

	if first != second {
		t.Error("resolving the same property prototype twice should return the same instance")
	}
}

func TestResolveTypeArguments(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	registerType(prog, "i32", newI32TypeDefinition(prog))

	// Create type parameters: <T>
	typeParams := []*ast.TypeParameterNode{
		{
			NodeBase: ast.NodeBase{Kind: ast.NodeKindTypeParameter, Range: nativeRange()},
			Name:     newTestIdent("T"),
		},
	}

	// Create type argument nodes: <i32>
	typeArgNodes := []ast.Node{
		newTestNamedTypeNode("i32"),
	}

	ctxTypes := make(map[string]*types.Type)
	result := resolver.ResolveTypeArguments(typeParams, typeArgNodes, nil, prog.NativeFile, ctxTypes, nil, ReportModeReport)
	if result == nil {
		t.Fatal("ResolveTypeArguments returned nil")
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 type argument, got %d", len(result))
	}
	if result[0] != types.TypeI32 {
		t.Error("type argument should be i32")
	}
}

func TestResolveTypeArgumentsWrongCount(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	// Create type parameters: <T, U>
	typeParams := []*ast.TypeParameterNode{
		{
			NodeBase: ast.NodeBase{Kind: ast.NodeKindTypeParameter, Range: nativeRange()},
			Name:     newTestIdent("T"),
		},
		{
			NodeBase: ast.NodeBase{Kind: ast.NodeKindTypeParameter, Range: nativeRange()},
			Name:     newTestIdent("U"),
		},
	}

	// Only 1 type argument for 2 type parameters
	typeArgNodes := []ast.Node{
		newTestNamedTypeNode("i32"),
	}

	ctxTypes := make(map[string]*types.Type)
	result := resolver.ResolveTypeArguments(typeParams, typeArgNodes, nil, prog.NativeFile, ctxTypes, nil, ReportModeSwallow)
	if result != nil {
		t.Error("ResolveTypeArguments should return nil for wrong count")
	}
}

func TestEnsureOneTypeArgument(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	registerType(prog, "i32", newI32TypeDefinition(prog))

	ctxTypes := make(map[string]*types.Type)
	result := resolver.EnsureOneTypeArgument(
		[]ast.Node{newTestNamedTypeNode("i32")},
		nil, prog.NativeFile, ctxTypes, nil, ReportModeReport,
	)
	if result == nil {
		t.Fatal("EnsureOneTypeArgument returned nil")
	}
	if result != types.TypeI32 {
		t.Error("expected i32 type")
	}
}

func TestEnsureOneTypeArgumentWrongCount(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	ctxTypes := make(map[string]*types.Type)

	// Zero type arguments
	result := resolver.EnsureOneTypeArgument(nil, nil, prog.NativeFile, ctxTypes, nil, ReportModeSwallow)
	if result != nil {
		t.Error("expected nil for 0 type arguments")
	}

	// Two type arguments
	result = resolver.EnsureOneTypeArgument(
		[]ast.Node{newTestNamedTypeNode("i32"), newTestNamedTypeNode("i32")},
		nil, prog.NativeFile, ctxTypes, nil, ReportModeSwallow,
	)
	if result != nil {
		t.Error("expected nil for 2 type arguments")
	}
}

func TestResolveOverridesEmpty(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)
	registerType(prog, "i32", newI32TypeDefinition(prog))

	sig := newTestSig(newTestNamedTypeNode("i32"))
	decl := newTestFuncDecl("fn", 0, sig)
	prototype := NewFunctionPrototype("fn", prog.NativeFile, decl, 0)

	instance := resolver.ResolveFunction(prototype, nil, nil, ReportModeSwallow)
	if instance == nil {
		t.Fatal("ResolveFunction returned nil")
	}

	overrides := resolver.ResolveOverrides(instance)
	if overrides != nil {
		t.Error("expected nil overrides for function without UnboundOverrides")
	}
}

func TestGetTypeOfElement(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	// Create a class and check its type
	classDecl := newTestClassDecl("Foo", 0)
	classProto := NewClassPrototype("Foo", prog.NativeFile, classDecl, 0, false)
	instance := resolver.ResolveClass(classProto, nil, nil, ReportModeReport)
	if instance == nil {
		t.Fatal("ResolveClass returned nil")
	}

	typ := resolver.GetTypeOfElement(instance)
	if typ == nil {
		t.Error("GetTypeOfElement should return non-nil for class")
	}
}

func TestLookupExpressionStub(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	result := resolver.LookupExpression(nil, nil, nil, ReportModeSwallow)
	if result != nil {
		t.Error("LookupExpression stub should return nil")
	}
}

func TestResolveExpressionStub(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)

	result := resolver.ResolveExpression(nil, nil, nil, ReportModeSwallow)
	if result != nil {
		t.Error("ResolveExpression stub should return nil")
	}
}

func TestResolveFunctionSetterReturnVoid(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)
	registerType(prog, "i32", newI32TypeDefinition(prog))

	params := []*ast.ParameterNode{
		newTestParam("value", newTestNamedTypeNode("i32")),
	}
	sig := newTestSigWithParams(params, newTestNamedTypeNode("i32")) // return type ignored for setter
	decl := newTestFuncDecl("myProp", int32(common.CommonFlagsSet), sig)
	prototype := NewFunctionPrototype("myProp", prog.NativeFile, decl, 0)
	prototype.Set(common.CommonFlagsSet)

	instance := resolver.ResolveFunction(prototype, nil, nil, ReportModeReport)
	if instance == nil {
		t.Fatal("ResolveFunction returned nil for setter")
	}
	if instance.Signature.ReturnType != types.TypeVoid {
		t.Error("setter return type should be void")
	}
}

func TestResolveFunctionConstructorReturnsClassType(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)
	registerType(prog, "void", newVoidTypeDefinition(prog))

	classDecl := newTestClassDecl("Obj", 0)
	classProto := NewClassPrototype("Obj", prog.NativeFile, classDecl, 0, false)
	classInstance := resolver.ResolveClass(classProto, nil, nil, ReportModeReport)
	if classInstance == nil {
		t.Fatal("ResolveClass returned nil")
	}

	ctorSig := newTestSig(newTestNamedTypeNode("void"))
	ctorDecl := newTestFuncDecl("constructor", int32(common.CommonFlagsConstructor|common.CommonFlagsInstance), ctorSig)
	ctorProto := NewFunctionPrototype("constructor", classInstance, ctorDecl, 0)
	ctorProto.Set(common.CommonFlagsConstructor | common.CommonFlagsInstance)

	boundProto := ctorProto.ToBound(classInstance)
	ctorInstance := resolver.ResolveFunction(boundProto, nil, nil, ReportModeReport)
	if ctorInstance == nil {
		t.Fatal("ResolveFunction returned nil for constructor")
	}
	if ctorInstance.Signature.ReturnType != classInstance.GetResolvedType() {
		t.Error("constructor return type should be the class type")
	}
}

func TestResolveFunctionTypeNode(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)
	registerType(prog, "i32", newI32TypeDefinition(prog))

	params := []*ast.ParameterNode{
		newTestParam("a", newTestNamedTypeNode("i32")),
	}
	retType := newTestNamedTypeNode("i32")
	ftNode := ast.NewFunctionTypeNode(params, retType, nil, false, nativeRange())

	result := resolver.ResolveType(ftNode, nil, prog.NativeFile, nil, ReportModeReport)
	if result == nil {
		t.Fatal("ResolveType returned nil for function type node")
	}
}

func TestResolveNamedTypeReturnsCorrectType(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)
	registerType(prog, "i32", newI32TypeDefinition(prog))

	node := newTestNamedTypeNode("i32")
	result := resolver.ResolveType(node, nil, prog.NativeFile, nil, ReportModeReport)
	if result == nil {
		t.Fatal("ResolveType returned nil for i32")
	}
	if result != types.TypeI32 {
		t.Error("expected i32 type")
	}
}

func TestResolveClassMemberFunction(t *testing.T) {
	prog := newTestProgram()
	resolver := NewResolver(prog)
	registerType(prog, "void", newVoidTypeDefinition(prog))

	classDecl := newTestClassDecl("MyClass", 0)
	classProto := NewClassPrototype("MyClass", prog.NativeFile, classDecl, 0, false)

	// Add a method to the class prototype
	methodSig := newTestSig(newTestNamedTypeNode("void"))
	methodDecl := newTestFuncDecl("doStuff", int32(common.CommonFlagsInstance), methodSig)
	methodProto := NewFunctionPrototype("doStuff", classProto, methodDecl, 0)
	classProto.Add("doStuff", methodProto, nil)

	classInstance := resolver.ResolveClass(classProto, nil, nil, ReportModeReport)
	if classInstance == nil {
		t.Fatal("ResolveClass returned nil")
	}
	// finishResolveClass should have resolved the method
}
