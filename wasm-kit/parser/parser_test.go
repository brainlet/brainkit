package parser

import (
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
)

// --- helpers ---

func parseSource(text string) (*ast.Source, []*diagnostics.DiagnosticMessage) {
	p := NewParser(nil)
	p.ParseFile(text, "test.ts", true)
	// Drain backlog (imports add files to backlog)
	for {
		next := p.NextFile()
		if next == "" {
			break
		}
		// Parse empty stubs for imported files
		p.ParseFile("", next, false)
	}
	p.Finish()
	sources := p.Sources()
	if len(sources) == 0 {
		return nil, p.Diagnostics
	}
	return sources[0], p.Diagnostics
}

func stmts(t *testing.T, src *ast.Source) []ast.Node {
	t.Helper()
	if src == nil {
		t.Fatal("source is nil")
	}
	return src.Statements
}

func expectStmtCount(t *testing.T, src *ast.Source, n int) []ast.Node {
	t.Helper()
	s := stmts(t, src)
	if len(s) != n {
		t.Fatalf("expected %d statements, got %d", n, len(s))
	}
	return s
}

func expectNoDiagnostics(t *testing.T, diags []*diagnostics.DiagnosticMessage) {
	t.Helper()
	if len(diags) > 0 {
		for _, d := range diags {
			t.Errorf("unexpected diagnostic: code=%d message=%q", d.Code, d.Message)
		}
		t.FailNow()
	}
}

func expectDiagnosticCount(t *testing.T, diags []*diagnostics.DiagnosticMessage, n int) {
	t.Helper()
	if len(diags) != n {
		for _, d := range diags {
			t.Logf("  diagnostic: code=%d message=%q", d.Code, d.Message)
		}
		t.Fatalf("expected %d diagnostics, got %d", n, len(diags))
	}
}

func expectDiagnosticCode(t *testing.T, diags []*diagnostics.DiagnosticMessage, idx int, code int32) {
	t.Helper()
	if idx >= len(diags) {
		t.Fatalf("diagnostic index %d out of range (have %d)", idx, len(diags))
	}
	if diags[idx].Code != code {
		t.Errorf("diagnostic[%d] code = %d, want %d (message: %q)", idx, diags[idx].Code, code, diags[idx].Message)
	}
}

func asExprStmt(t *testing.T, node ast.Node) *ast.ExpressionStatement {
	t.Helper()
	es, ok := node.(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T (kind=%d)", node, node.GetKind())
	}
	return es
}

// --- Empty source ---

func TestParseEmpty(t *testing.T) {
	src, diags := parseSource("")
	expectNoDiagnostics(t, diags)
	expectStmtCount(t, src, 0)
}

// --- Literals ---

func TestParseIntegerLiteral(t *testing.T) {
	src, diags := parseSource("42;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	if es.Expression.GetKind() != ast.NodeKindLiteral {
		t.Fatalf("expected Literal, got kind=%d", es.Expression.GetKind())
	}
	lit := es.Expression.(*ast.IntegerLiteralExpression)
	if lit.LiteralKind != ast.LiteralKindInteger {
		t.Errorf("LiteralKind = %d, want Integer", lit.LiteralKind)
	}
	if lit.Value != 42 {
		t.Errorf("Value = %d, want 42", lit.Value)
	}
}

func TestParseFloatLiteral(t *testing.T) {
	src, diags := parseSource("3.14;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	lit := es.Expression.(*ast.FloatLiteralExpression)
	if lit.LiteralKind != ast.LiteralKindFloat {
		t.Errorf("LiteralKind = %d, want Float", lit.LiteralKind)
	}
	if lit.Value != 3.14 {
		t.Errorf("Value = %f, want 3.14", lit.Value)
	}
}

func TestParseStringLiteral(t *testing.T) {
	src, diags := parseSource(`"hello";`)
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	lit := es.Expression.(*ast.StringLiteralExpression)
	if lit.LiteralKind != ast.LiteralKindString {
		t.Errorf("LiteralKind = %d, want String", lit.LiteralKind)
	}
	if lit.Value != "hello" {
		t.Errorf("Value = %q, want %q", lit.Value, "hello")
	}
}

// --- Identifiers ---

func TestParseIdentifier(t *testing.T) {
	src, diags := parseSource("foo;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	ident := es.Expression.(*ast.IdentifierExpression)
	if ident.Text != "foo" {
		t.Errorf("Text = %q, want %q", ident.Text, "foo")
	}
}

// --- Boolean and null literals ---

func TestParseTrueFalseNull(t *testing.T) {
	src, diags := parseSource("true;\nfalse;\nnull;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 3)

	if s[0].(*ast.ExpressionStatement).Expression.GetKind() != ast.NodeKindTrue {
		t.Error("expected true expression")
	}
	if s[1].(*ast.ExpressionStatement).Expression.GetKind() != ast.NodeKindFalse {
		t.Error("expected false expression")
	}
	if s[2].(*ast.ExpressionStatement).Expression.GetKind() != ast.NodeKindNull {
		t.Error("expected null expression")
	}
}

// --- this / super ---

func TestParseThis(t *testing.T) {
	src, diags := parseSource("this;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	if es.Expression.GetKind() != ast.NodeKindThis {
		t.Errorf("expected This, got kind=%d", es.Expression.GetKind())
	}
}

// --- Binary expressions ---

func TestParseBinaryExpression(t *testing.T) {
	src, diags := parseSource("a + b;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	bin := es.Expression.(*ast.BinaryExpression)
	if bin.Operator != tokenizer.TokenPlus {
		t.Errorf("Operator = %d, want TokenPlus", bin.Operator)
	}
	left := bin.Left.(*ast.IdentifierExpression)
	if left.Text != "a" {
		t.Errorf("Left.Text = %q, want %q", left.Text, "a")
	}
	right := bin.Right.(*ast.IdentifierExpression)
	if right.Text != "b" {
		t.Errorf("Right.Text = %q, want %q", right.Text, "b")
	}
}

func TestParseBinaryPrecedence(t *testing.T) {
	// a + b * c should parse as a + (b * c)
	src, diags := parseSource("a + b * c;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	bin := es.Expression.(*ast.BinaryExpression)
	if bin.Operator != tokenizer.TokenPlus {
		t.Fatalf("top operator = %d, want TokenPlus", bin.Operator)
	}
	right := bin.Right.(*ast.BinaryExpression)
	if right.Operator != tokenizer.TokenAsterisk {
		t.Errorf("right operator = %d, want TokenAsterisk", right.Operator)
	}
}

// --- Unary expressions ---

func TestParseUnaryPrefix(t *testing.T) {
	src, diags := parseSource("!x;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	unary := es.Expression.(*ast.UnaryPrefixExpression)
	if unary.Operator != tokenizer.TokenExclamation {
		t.Errorf("Operator = %d, want TokenExclamation", unary.Operator)
	}
	operand := unary.Operand.(*ast.IdentifierExpression)
	if operand.Text != "x" {
		t.Errorf("Operand.Text = %q, want %q", operand.Text, "x")
	}
}

func TestParseUnaryPostfix(t *testing.T) {
	src, diags := parseSource("x++;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	unary := es.Expression.(*ast.UnaryPostfixExpression)
	if unary.Operator != tokenizer.TokenPlusPlus {
		t.Errorf("Operator = %d, want TokenPlusPlus", unary.Operator)
	}
}

// --- Call expressions ---

func TestParseCallExpression(t *testing.T) {
	src, diags := parseSource("foo();")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	call := es.Expression.(*ast.CallExpression)
	callee := call.Expression.(*ast.IdentifierExpression)
	if callee.Text != "foo" {
		t.Errorf("callee = %q, want %q", callee.Text, "foo")
	}
	if len(call.Args) != 0 {
		t.Errorf("expected 0 args, got %d", len(call.Args))
	}
}

func TestParseCallWithArgs(t *testing.T) {
	src, diags := parseSource("foo(a, b);")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	call := es.Expression.(*ast.CallExpression)
	if len(call.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(call.Args))
	}
	arg0 := call.Args[0].(*ast.IdentifierExpression)
	if arg0.Text != "a" {
		t.Errorf("arg0 = %q, want %q", arg0.Text, "a")
	}
}

func TestParseChainedCalls(t *testing.T) {
	src, diags := parseSource("foo()();")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	outer := es.Expression.(*ast.CallExpression)
	inner := outer.Expression.(*ast.CallExpression)
	callee := inner.Expression.(*ast.IdentifierExpression)
	if callee.Text != "foo" {
		t.Errorf("callee = %q, want %q", callee.Text, "foo")
	}
}

// --- Property access ---

func TestParsePropertyAccess(t *testing.T) {
	src, diags := parseSource("obj.prop;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	prop := es.Expression.(*ast.PropertyAccessExpression)
	obj := prop.Expression.(*ast.IdentifierExpression)
	if obj.Text != "obj" {
		t.Errorf("object = %q, want %q", obj.Text, "obj")
	}
	if prop.Property.Text != "prop" {
		t.Errorf("property = %q, want %q", prop.Property.Text, "prop")
	}
}

// --- Element access ---

func TestParseElementAccess(t *testing.T) {
	src, diags := parseSource("arr[0];")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	elem := es.Expression.(*ast.ElementAccessExpression)
	arr := elem.Expression.(*ast.IdentifierExpression)
	if arr.Text != "arr" {
		t.Errorf("expression = %q, want %q", arr.Text, "arr")
	}
	idx := elem.ElementExpression.(*ast.IntegerLiteralExpression)
	if idx.Value != 0 {
		t.Errorf("index = %d, want 0", idx.Value)
	}
}

// --- Parenthesized expression ---

func TestParseParenthesized(t *testing.T) {
	src, diags := parseSource("(a + b);")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	paren := es.Expression.(*ast.ParenthesizedExpression)
	bin := paren.Expression.(*ast.BinaryExpression)
	if bin.Operator != tokenizer.TokenPlus {
		t.Errorf("operator = %d, want TokenPlus", bin.Operator)
	}
}

// --- Ternary expression ---

func TestParseTernary(t *testing.T) {
	src, diags := parseSource("a ? b : c;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	tern := es.Expression.(*ast.TernaryExpression)
	cond := tern.Condition.(*ast.IdentifierExpression)
	if cond.Text != "a" {
		t.Errorf("condition = %q, want %q", cond.Text, "a")
	}
	ifTrue := tern.IfThen.(*ast.IdentifierExpression)
	if ifTrue.Text != "b" {
		t.Errorf("ifThen = %q, want %q", ifTrue.Text, "b")
	}
	ifFalse := tern.IfElse.(*ast.IdentifierExpression)
	if ifFalse.Text != "c" {
		t.Errorf("ifElse = %q, want %q", ifFalse.Text, "c")
	}
}

// --- New expression ---

func TestParseNewExpression(t *testing.T) {
	src, diags := parseSource("new Foo();")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	newExpr := es.Expression.(*ast.NewExpression)
	if newExpr.TypeName.Identifier.Text != "Foo" {
		t.Errorf("typeName = %q, want %q", newExpr.TypeName.Identifier.Text, "Foo")
	}
}

// --- Assignment ---

func TestParseAssignment(t *testing.T) {
	src, diags := parseSource("x = 5;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	bin := es.Expression.(*ast.BinaryExpression)
	if bin.Operator != tokenizer.TokenEquals {
		t.Errorf("operator = %d, want TokenEquals", bin.Operator)
	}
}

// --- Variable declarations ---

func TestParseLetDeclaration(t *testing.T) {
	src, diags := parseSource("let x: i32 = 5;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	varStmt := s[0].(*ast.VariableStatement)
	if len(varStmt.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(varStmt.Declarations))
	}
	decl := varStmt.Declarations[0]
	if decl.Name.Text != "x" {
		t.Errorf("name = %q, want %q", decl.Name.Text, "x")
	}
	if decl.Type == nil {
		t.Error("expected type annotation")
	}
	if decl.Initializer == nil {
		t.Error("expected initializer")
	}
}

func TestParseConstDeclaration(t *testing.T) {
	src, diags := parseSource("const y = 10;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	varStmt := s[0].(*ast.VariableStatement)
	if len(varStmt.Declarations) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(varStmt.Declarations))
	}
}

func TestParseMultipleVarDeclarations(t *testing.T) {
	src, diags := parseSource("let a = 1, b = 2;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	varStmt := s[0].(*ast.VariableStatement)
	if len(varStmt.Declarations) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(varStmt.Declarations))
	}
	if varStmt.Declarations[0].Name.Text != "a" {
		t.Errorf("first decl name = %q, want %q", varStmt.Declarations[0].Name.Text, "a")
	}
	if varStmt.Declarations[1].Name.Text != "b" {
		t.Errorf("second decl name = %q, want %q", varStmt.Declarations[1].Name.Text, "b")
	}
}

// --- Function declarations ---

func TestParseFunctionDeclaration(t *testing.T) {
	src, diags := parseSource("function add(a: i32, b: i32): i32 { return a + b; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	fn := s[0].(*ast.FunctionDeclaration)
	if fn.Name.Text != "add" {
		t.Errorf("name = %q, want %q", fn.Name.Text, "add")
	}
	if fn.Signature == nil {
		t.Fatal("expected signature")
	}
	if len(fn.Signature.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fn.Signature.Parameters))
	}
	if fn.Body == nil {
		t.Error("expected body")
	}
}

func TestParseFunctionNoReturnType(t *testing.T) {
	src, diags := parseSource("function foo(): void {}")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	fn := s[0].(*ast.FunctionDeclaration)
	if fn.Name.Text != "foo" {
		t.Errorf("name = %q, want %q", fn.Name.Text, "foo")
	}
}

// --- Class declarations ---

func TestParseClassDeclaration(t *testing.T) {
	src, diags := parseSource("class Foo { x: i32; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	cls := s[0].(*ast.ClassDeclaration)
	if cls.Name.Text != "Foo" {
		t.Errorf("name = %q, want %q", cls.Name.Text, "Foo")
	}
	if len(cls.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(cls.Members))
	}
}

func TestParseClassWithMethod(t *testing.T) {
	src, diags := parseSource("class Bar { greet(): void {} }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	cls := s[0].(*ast.ClassDeclaration)
	if len(cls.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(cls.Members))
	}
	method := cls.Members[0].(*ast.FunctionDeclaration)
	if method.GetKind() != ast.NodeKindMethodDeclaration {
		t.Errorf("expected MethodDeclaration kind, got %d", method.GetKind())
	}
	if method.Name.Text != "greet" {
		t.Errorf("method name = %q, want %q", method.Name.Text, "greet")
	}
}

func TestParseClassWithTypeParameters(t *testing.T) {
	src, diags := parseSource("class Box<T> { value: T; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	cls := s[0].(*ast.ClassDeclaration)
	if len(cls.TypeParameters) != 1 {
		t.Fatalf("expected 1 type parameter, got %d", len(cls.TypeParameters))
	}
	if cls.TypeParameters[0].Name.Text != "T" {
		t.Errorf("type param = %q, want %q", cls.TypeParameters[0].Name.Text, "T")
	}
}

func TestParseClassExtends(t *testing.T) {
	src, diags := parseSource("class Child extends Parent {}")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	cls := s[0].(*ast.ClassDeclaration)
	if cls.ExtendsType == nil {
		t.Fatal("expected extends type")
	}
}

// --- Interface declarations ---

func TestParseInterfaceDeclaration(t *testing.T) {
	src, diags := parseSource("interface IFoo { bar(): void; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	iface := s[0].(*ast.ClassDeclaration) // interfaces use ClassDeclaration
	if iface.Name.Text != "IFoo" {
		t.Errorf("name = %q, want %q", iface.Name.Text, "IFoo")
	}
}

// --- Enum declarations ---

func TestParseEnumDeclaration(t *testing.T) {
	src, diags := parseSource("enum Color { Red, Green, Blue }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	enumDecl := s[0].(*ast.EnumDeclaration)
	if enumDecl.Name.Text != "Color" {
		t.Errorf("name = %q, want %q", enumDecl.Name.Text, "Color")
	}
	if len(enumDecl.Values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(enumDecl.Values))
	}
}

func TestParseEnumWithValues(t *testing.T) {
	src, diags := parseSource("enum E { A = 1, B = 2 }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	enumDecl := s[0].(*ast.EnumDeclaration)
	if len(enumDecl.Values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(enumDecl.Values))
	}
	if enumDecl.Values[0].Initializer == nil {
		t.Error("expected initializer for first value")
	}
}

// --- Statements ---

func TestParseIfStatement(t *testing.T) {
	src, diags := parseSource("if (true) { x; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	ifStmt := s[0].(*ast.IfStatement)
	if ifStmt.Condition.GetKind() != ast.NodeKindTrue {
		t.Error("expected true condition")
	}
	if ifStmt.IfFalse != nil {
		t.Error("expected no else branch")
	}
}

func TestParseIfElse(t *testing.T) {
	src, diags := parseSource("if (x) { a; } else { b; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	ifStmt := s[0].(*ast.IfStatement)
	if ifStmt.IfFalse == nil {
		t.Error("expected else branch")
	}
}

func TestParseWhileStatement(t *testing.T) {
	src, diags := parseSource("while (true) { break; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	whileStmt := s[0].(*ast.WhileStatement)
	if whileStmt.Condition.GetKind() != ast.NodeKindTrue {
		t.Error("expected true condition")
	}
}

func TestParseDoWhile(t *testing.T) {
	src, diags := parseSource("do { x; } while (y);")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	doStmt := s[0].(*ast.DoStatement)
	if doStmt.Condition == nil {
		t.Error("expected condition")
	}
}

func TestParseForStatement(t *testing.T) {
	src, diags := parseSource("for (let i = 0; i < 10; i++) {}")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	forStmt := s[0].(*ast.ForStatement)
	if forStmt.Initializer == nil {
		t.Error("expected initializer")
	}
	if forStmt.Condition == nil {
		t.Error("expected condition")
	}
	if forStmt.Incrementor == nil {
		t.Error("expected incrementor")
	}
}

func TestParseReturnStatement(t *testing.T) {
	src, diags := parseSource("function f(): i32 { return 42; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	fn := s[0].(*ast.FunctionDeclaration)
	body := fn.Body.(*ast.BlockStatement)
	if len(body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(body.Statements))
	}
	ret := body.Statements[0].(*ast.ReturnStatement)
	if ret.Value == nil {
		t.Error("expected return value")
	}
}

func TestParseBreakContinue(t *testing.T) {
	src, diags := parseSource("while (true) { break; continue; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	whileStmt := s[0].(*ast.WhileStatement)
	block := whileStmt.Body.(*ast.BlockStatement)
	if len(block.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(block.Statements))
	}
	if block.Statements[0].GetKind() != ast.NodeKindBreak {
		t.Error("expected break")
	}
	if block.Statements[1].GetKind() != ast.NodeKindContinue {
		t.Error("expected continue")
	}
}

func TestParseSwitchStatement(t *testing.T) {
	src, diags := parseSource("switch (x) { case 1: break; default: break; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	sw := s[0].(*ast.SwitchStatement)
	if len(sw.Cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(sw.Cases))
	}
}

func TestParseThrowStatement(t *testing.T) {
	src, diags := parseSource("throw new Error();")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	throwStmt := s[0].(*ast.ThrowStatement)
	if throwStmt.Value == nil {
		t.Error("expected throw value")
	}
}

func TestParseTryCatch(t *testing.T) {
	src, diags := parseSource("try { x; } catch (e) { y; }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	tryStmt := s[0].(*ast.TryStatement)
	if tryStmt.CatchVariable == nil {
		t.Error("expected catch variable")
	}
	if tryStmt.CatchStatements == nil {
		t.Error("expected catch body")
	}
}

// --- Export / Import ---

func TestParseExportFunction(t *testing.T) {
	src, diags := parseSource("export function foo(): void {}")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	fn := s[0].(*ast.FunctionDeclaration)
	if fn.Name.Text != "foo" {
		t.Errorf("name = %q, want %q", fn.Name.Text, "foo")
	}
	// Export flag should be set on the declaration
	if fn.Flags&int32(common.CommonFlagsExport) == 0 {
		t.Error("expected CommonFlagsExport to be set")
	}
}

func TestParseImport(t *testing.T) {
	src, diags := parseSource(`import { foo } from "./bar";`)
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	if s[0].GetKind() != ast.NodeKindImport {
		t.Errorf("expected import statement, got kind=%d", s[0].GetKind())
	}
}

// --- Namespace ---

func TestParseNamespace(t *testing.T) {
	src, diags := parseSource("namespace Foo { export function bar(): void {} }")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	ns := s[0].(*ast.NamespaceDeclaration)
	if ns.Name.Text != "Foo" {
		t.Errorf("name = %q, want %q", ns.Name.Text, "Foo")
	}
	if len(ns.Members) == 0 {
		t.Error("expected members")
	}
}

// --- Type declarations ---

func TestParseTypeDeclaration(t *testing.T) {
	src, diags := parseSource("type Num = i32;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	typeDecl := s[0].(*ast.TypeDeclaration)
	if typeDecl.Name.Text != "Num" {
		t.Errorf("name = %q, want %q", typeDecl.Name.Text, "Num")
	}
}

// --- Arrow functions ---

func TestParseArrowFunction(t *testing.T) {
	src, diags := parseSource("let f = (x: i32): i32 => x;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	varStmt := s[0].(*ast.VariableStatement)
	decl := varStmt.Declarations[0]
	fnExpr := decl.Initializer.(*ast.FunctionExpression)
	if fnExpr.Declaration.ArrowKind != ast.ArrowKindParenthesized {
		t.Errorf("ArrowKind = %d, want ArrowKindParenthesized", fnExpr.Declaration.ArrowKind)
	}
}

// --- Array literal ---

func TestParseArrayLiteral(t *testing.T) {
	src, diags := parseSource("[1, 2, 3];")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	arr := es.Expression.(*ast.ArrayLiteralExpression)
	if len(arr.ElementExpressions) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr.ElementExpressions))
	}
}

// --- Multiple statements ---

func TestParseMultipleStatements(t *testing.T) {
	src, diags := parseSource("let x = 1;\nlet y = 2;\nx + y;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 3)
	if s[0].GetKind() != ast.NodeKindVariable {
		t.Errorf("stmt[0] kind = %d, want Variable", s[0].GetKind())
	}
	if s[1].GetKind() != ast.NodeKindVariable {
		t.Errorf("stmt[1] kind = %d, want Variable", s[1].GetKind())
	}
	if s[2].GetKind() != ast.NodeKindExpression {
		t.Errorf("stmt[2] kind = %d, want Expression", s[2].GetKind())
	}
}

// --- ASI (Automatic Semicolon Insertion) ---

func TestParseASI(t *testing.T) {
	src, diags := parseSource("let x = 1\nlet y = 2\n")
	expectNoDiagnostics(t, diags)
	expectStmtCount(t, src, 2)
}

// --- Error recovery ---

func TestParseErrorRecovery(t *testing.T) {
	// Missing semicolons and invalid tokens; parser should recover
	src, diags := parseSource("let x = ;")
	if len(diags) == 0 {
		t.Error("expected diagnostics for invalid source")
	}
	// Parser should still produce some output
	if src == nil {
		t.Fatal("expected source even on error")
	}
}

// --- Decorator ---

func TestParseDecorator(t *testing.T) {
	src, diags := parseSource("@inline\nexport function foo(): void {}")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	fn := s[0].(*ast.FunctionDeclaration)
	if len(fn.Decorators) != 1 {
		t.Fatalf("expected 1 decorator, got %d", len(fn.Decorators))
	}
	decorName := fn.Decorators[0].Name.(*ast.IdentifierExpression)
	if decorName.Text != "inline" {
		t.Errorf("decorator name = %q, want %q", decorName.Text, "inline")
	}
}

// --- Comma expression ---

func TestParseCommaExpression(t *testing.T) {
	src, diags := parseSource("(a, b, c);")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	paren := es.Expression.(*ast.ParenthesizedExpression)
	comma := paren.Expression.(*ast.CommaExpression)
	if len(comma.Expressions) != 3 {
		t.Errorf("expected 3 comma expressions, got %d", len(comma.Expressions))
	}
}

// --- Template literal ---

func TestParseTemplateLiteral(t *testing.T) {
	src, diags := parseSource("`hello`;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	tmpl := es.Expression.(*ast.TemplateLiteralExpression)
	if len(tmpl.Parts) != 1 {
		t.Errorf("expected 1 part, got %d", len(tmpl.Parts))
	}
}

func TestParseTemplateLiteralWithExpression(t *testing.T) {
	src, diags := parseSource("`hello ${name}`;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	tmpl := es.Expression.(*ast.TemplateLiteralExpression)
	if len(tmpl.Parts) != 2 {
		t.Errorf("expected 2 parts, got %d", len(tmpl.Parts))
	}
	if len(tmpl.Expressions) != 1 {
		t.Errorf("expected 1 expression, got %d", len(tmpl.Expressions))
	}
}

// --- as-assertion ---

func TestParseAsAssertion(t *testing.T) {
	src, diags := parseSource("x as i32;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	assertion := es.Expression.(*ast.AssertionExpression)
	if assertion.AssertionKind != ast.AssertionKindAs {
		t.Errorf("kind = %d, want AssertionKindAs", assertion.AssertionKind)
	}
}

// --- instanceof ---

func TestParseInstanceOf(t *testing.T) {
	src, diags := parseSource("x instanceof Foo;")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	if es.Expression.GetKind() != ast.NodeKindInstanceOf {
		t.Errorf("expected InstanceOf, got kind=%d", es.Expression.GetKind())
	}
}

// --- for-of ---

func TestParseForOf(t *testing.T) {
	src, diags := parseSource("for (let x of arr) {}")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	forOf := s[0].(*ast.ForOfStatement)
	if forOf.Variable == nil {
		t.Error("expected variable")
	}
	if forOf.Iterable == nil {
		t.Error("expected iterable")
	}
}

// --- Diagnostic codes ---

func TestDiagnosticOnInvalidExpression(t *testing.T) {
	_, diags := parseSource("let x = ;")
	if len(diags) == 0 {
		t.Fatal("expected diagnostics")
	}
	// Should get "Expression expected" (1109)
	found := false
	for _, d := range diags {
		if d.Code == 1109 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected diagnostic code 1109 (Expression expected), got codes:")
		for _, d := range diags {
			t.Logf("  code=%d message=%q", d.Code, d.Message)
		}
	}
}

// --- Complex chained expression ---

func TestParseComplexChainedExpression(t *testing.T) {
	// id(a)[0](b)(c).a(d)
	src, diags := parseSource("id(a)[0](b)(c).a(d);")
	expectNoDiagnostics(t, diags)
	s := expectStmtCount(t, src, 1)
	es := asExprStmt(t, s[0])
	// The outermost should be a call: .a(d)
	outerCall := es.Expression.(*ast.CallExpression)
	if len(outerCall.Args) != 1 {
		t.Errorf("outer call args = %d, want 1", len(outerCall.Args))
	}
	// Its expression should be a property access: .a
	propAccess := outerCall.Expression.(*ast.PropertyAccessExpression)
	if propAccess.Property.Text != "a" {
		t.Errorf("property = %q, want %q", propAccess.Property.Text, "a")
	}
}

// --- Parser lifecycle ---

func TestParserMultipleFiles(t *testing.T) {
	p := NewParser(nil)
	p.ParseFile("let a = 1;", "file1.ts", true)
	p.ParseFile("let b = 2;", "file2.ts", false)
	p.Finish()
	if len(p.Sources()) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(p.Sources()))
	}
}

func TestParserDuplicateFile(t *testing.T) {
	p := NewParser(nil)
	p.ParseFile("let a = 1;", "file.ts", true)
	p.ParseFile("let b = 2;", "file.ts", false) // same path
	p.Finish()
	// Should only have parsed once
	if len(p.Sources()) != 1 {
		t.Fatalf("expected 1 source (deduped), got %d", len(p.Sources()))
	}
}

// --- Comment handler ---

func TestParserCommentHandler(t *testing.T) {
	p := NewParser(nil)
	var kinds []tokenizer.CommentKind
	p.OnComment = func(kind tokenizer.CommentKind, text string, rng *diagnostics.Range) {
		kinds = append(kinds, kind)
	}
	p.ParseFile("// line comment\nlet x = 1; /* block */", "test.ts", true)
	p.Finish()
	if len(kinds) == 0 {
		t.Fatal("expected at least one comment callback")
	}
	hasLine, hasBlock := false, false
	for _, k := range kinds {
		if k == tokenizer.CommentKindLine || k == tokenizer.CommentKindTriple {
			hasLine = true
		}
		if k == tokenizer.CommentKindBlock {
			hasBlock = true
		}
	}
	if !hasLine {
		t.Error("expected line comment kind")
	}
	if !hasBlock {
		t.Error("expected block comment kind")
	}
}
