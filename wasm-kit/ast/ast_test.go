package ast

import (
	"testing"

	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
	"github.com/brainlet/brainkit/wasm-kit/tokenizer"
)

func makeRange(start, end int32) diagnostics.Range {
	return diagnostics.Range{Start: start, End: end}
}

func makeSourceRange(src *Source, start, end int32) diagnostics.Range {
	return diagnostics.Range{Start: start, End: end, Source: src}
}

// --- NodeKind enum ---

func TestNodeKindValues(t *testing.T) {
	if NodeKindSource != 0 {
		t.Errorf("NodeKindSource = %d, want 0", NodeKindSource)
	}
	if NodeKindIdentifier != 6 {
		t.Errorf("NodeKindIdentifier = %d, want 6", NodeKindIdentifier)
	}
	// Verify sequential ordering
	if NodeKindNamedType != 1 {
		t.Errorf("NodeKindNamedType = %d, want 1", NodeKindNamedType)
	}
	// Parameter is last in types block
	if NodeKindParameter != 5 {
		t.Errorf("NodeKindParameter = %d, want 5", NodeKindParameter)
	}
}

// --- Small enums ---

func TestParameterKind(t *testing.T) {
	if ParameterKindDefault != 0 || ParameterKindOptional != 1 || ParameterKindRest != 2 {
		t.Error("ParameterKind values incorrect")
	}
}

func TestLiteralKind(t *testing.T) {
	if LiteralKindFloat != 0 || LiteralKindInteger != 1 || LiteralKindString != 2 {
		t.Error("LiteralKind values incorrect")
	}
}

func TestAssertionKind(t *testing.T) {
	if AssertionKindPrefix != 0 || AssertionKindAs != 1 || AssertionKindNonNull != 2 || AssertionKindConst != 3 {
		t.Error("AssertionKind values incorrect")
	}
}

func TestArrowKind(t *testing.T) {
	if ArrowKindNone != 0 || ArrowKindParenthesized != 1 || ArrowKindSingle != 2 {
		t.Error("ArrowKind values incorrect")
	}
}

func TestSourceKind(t *testing.T) {
	if SourceKindUser != 0 || SourceKindUserEntry != 1 || SourceKindLibrary != 2 || SourceKindLibraryEntry != 3 {
		t.Error("SourceKind values incorrect")
	}
}

func TestDecoratorKind(t *testing.T) {
	if DecoratorKindCustom != 0 || DecoratorKindGlobal != 1 {
		t.Error("DecoratorKind values incorrect")
	}
}

// --- Node interface ---

func TestNodeBaseImplementsNode(t *testing.T) {
	var n Node = &IdentifierExpression{
		NodeBase: NodeBase{Kind: NodeKindIdentifier, Range: makeRange(0, 3)},
		Text:     "foo",
	}
	if n.GetKind() != NodeKindIdentifier {
		t.Errorf("GetKind() = %d, want %d", n.GetKind(), NodeKindIdentifier)
	}
	r := n.GetRange()
	if r.Start != 0 || r.End != 3 {
		t.Errorf("GetRange() = {%d,%d}, want {0,3}", r.Start, r.End)
	}
}

// --- Identifier expressions ---

func TestNewIdentifierExpression(t *testing.T) {
	rng := makeRange(0, 3)
	id := NewIdentifierExpression("foo", rng, false)
	if id.GetKind() != NodeKindIdentifier {
		t.Error("wrong kind")
	}
	if id.Text != "foo" {
		t.Errorf("Text = %q, want %q", id.Text, "foo")
	}
	if id.IsQuoted {
		t.Error("should not be quoted")
	}
}

func TestNewConstructorExpression(t *testing.T) {
	rng := makeRange(0, 11)
	c := NewConstructorExpression(rng)
	if c.GetKind() != NodeKindConstructor {
		t.Errorf("kind = %d, want %d", c.GetKind(), NodeKindConstructor)
	}
	if c.Text != "constructor" {
		t.Errorf("Text = %q, want %q", c.Text, "constructor")
	}
}

func TestNewNullExpression(t *testing.T) {
	n := NewNullExpression(makeRange(0, 4))
	if n.GetKind() != NodeKindNull {
		t.Error("wrong kind")
	}
	if n.Text != "null" {
		t.Error("wrong text")
	}
}

func TestNewSuperExpression(t *testing.T) {
	s := NewSuperExpression(makeRange(0, 5))
	if s.GetKind() != NodeKindSuper {
		t.Error("wrong kind")
	}
	if s.Text != "super" {
		t.Error("wrong text")
	}
}

func TestNewThisExpression(t *testing.T) {
	th := NewThisExpression(makeRange(0, 4))
	if th.GetKind() != NodeKindThis {
		t.Error("wrong kind")
	}
}

func TestNewTrueExpression(t *testing.T) {
	tr := NewTrueExpression(makeRange(0, 4))
	if tr.GetKind() != NodeKindTrue {
		t.Error("wrong kind")
	}
}

func TestNewFalseExpression(t *testing.T) {
	f := NewFalseExpression(makeRange(0, 5))
	if f.GetKind() != NodeKindFalse {
		t.Error("wrong kind")
	}
}

// --- Literal expressions ---

func TestNewFloatLiteralExpression(t *testing.T) {
	f := NewFloatLiteralExpression(3.14, makeRange(0, 4))
	if f.GetKind() != NodeKindLiteral {
		t.Error("wrong kind")
	}
	if f.LiteralKind != LiteralKindFloat {
		t.Error("wrong literal kind")
	}
	if f.Value != 3.14 {
		t.Errorf("Value = %f, want 3.14", f.Value)
	}
}

func TestNewIntegerLiteralExpression(t *testing.T) {
	i := NewIntegerLiteralExpression(42, makeRange(0, 2))
	if i.LiteralKind != LiteralKindInteger {
		t.Error("wrong literal kind")
	}
	if i.Value != 42 {
		t.Errorf("Value = %d, want 42", i.Value)
	}
}

func TestNewStringLiteralExpression(t *testing.T) {
	s := NewStringLiteralExpression("hello", makeRange(0, 7))
	if s.LiteralKind != LiteralKindString {
		t.Error("wrong literal kind")
	}
	if s.Value != "hello" {
		t.Errorf("Value = %q, want %q", s.Value, "hello")
	}
}

func TestNewArrayLiteralExpression(t *testing.T) {
	a := NewArrayLiteralExpression(nil, makeRange(0, 2))
	if a.LiteralKind != LiteralKindArray {
		t.Error("wrong literal kind")
	}
}

func TestNewObjectLiteralExpression(t *testing.T) {
	o := NewObjectLiteralExpression(nil, nil, makeRange(0, 2))
	if o.LiteralKind != LiteralKindObject {
		t.Error("wrong literal kind")
	}
}

func TestNewRegexpLiteralExpression(t *testing.T) {
	r := NewRegexpLiteralExpression("abc", "gi", makeRange(0, 8))
	if r.LiteralKind != LiteralKindRegExp {
		t.Error("wrong literal kind")
	}
	if r.Pattern != "abc" || r.PatternFlags != "gi" {
		t.Error("wrong pattern/flags")
	}
}

func TestNewTemplateLiteralExpression(t *testing.T) {
	tl := NewTemplateLiteralExpression(nil, []string{"hello ", " world"}, []string{"hello ", " world"}, nil, makeRange(0, 20))
	if tl.LiteralKind != LiteralKindTemplate {
		t.Error("wrong literal kind")
	}
	if len(tl.Parts) != 2 {
		t.Error("wrong parts count")
	}
}

// --- Other expressions ---

func TestNewBinaryExpression(t *testing.T) {
	left := NewIdentifierExpression("a", makeRange(0, 1), false)
	right := NewIdentifierExpression("b", makeRange(4, 5), false)
	b := NewBinaryExpression(tokenizer.TokenPlus, left, right, makeRange(0, 5))
	if b.GetKind() != NodeKindBinary {
		t.Error("wrong kind")
	}
	if b.Operator != tokenizer.TokenPlus {
		t.Error("wrong operator")
	}
}

func TestNewCallExpression(t *testing.T) {
	fn := NewIdentifierExpression("foo", makeRange(0, 3), false)
	c := NewCallExpression(fn, nil, nil, makeRange(0, 5))
	if c.GetKind() != NodeKindCall {
		t.Error("wrong kind")
	}
}

func TestNewUnaryExpressions(t *testing.T) {
	operand := NewIdentifierExpression("x", makeRange(0, 1), false)
	post := NewUnaryPostfixExpression(tokenizer.TokenPlusPlus, operand, makeRange(0, 3))
	if post.GetKind() != NodeKindUnaryPostfix {
		t.Error("wrong kind for postfix")
	}
	pre := NewUnaryPrefixExpression(tokenizer.TokenMinus, operand, makeRange(0, 2))
	if pre.GetKind() != NodeKindUnaryPrefix {
		t.Error("wrong kind for prefix")
	}
}

func TestNewTernaryExpression(t *testing.T) {
	cond := NewTrueExpression(makeRange(0, 4))
	then := NewIntegerLiteralExpression(1, makeRange(7, 8))
	els := NewIntegerLiteralExpression(2, makeRange(11, 12))
	ter := NewTernaryExpression(cond, then, els, makeRange(0, 12))
	if ter.GetKind() != NodeKindTernary {
		t.Error("wrong kind")
	}
}

func TestNewAssertionExpression(t *testing.T) {
	expr := NewIdentifierExpression("x", makeRange(0, 1), false)
	a := NewAssertionExpression(AssertionKindNonNull, expr, nil, makeRange(0, 2))
	if a.GetKind() != NodeKindAssertion {
		t.Error("wrong kind")
	}
	if a.AssertionKind != AssertionKindNonNull {
		t.Error("wrong assertion kind")
	}
}

func TestNewPropertyAccessExpression(t *testing.T) {
	obj := NewIdentifierExpression("obj", makeRange(0, 3), false)
	prop := NewIdentifierExpression("field", makeRange(4, 9), false)
	pa := NewPropertyAccessExpression(obj, prop, makeRange(0, 9))
	if pa.GetKind() != NodeKindPropertyAccess {
		t.Error("wrong kind")
	}
}

func TestNewElementAccessExpression(t *testing.T) {
	arr := NewIdentifierExpression("arr", makeRange(0, 3), false)
	idx := NewIntegerLiteralExpression(0, makeRange(4, 5))
	ea := NewElementAccessExpression(arr, idx, makeRange(0, 6))
	if ea.GetKind() != NodeKindElementAccess {
		t.Error("wrong kind")
	}
}

func TestNewParenthesizedExpression(t *testing.T) {
	inner := NewIdentifierExpression("x", makeRange(1, 2), false)
	p := NewParenthesizedExpression(inner, makeRange(0, 3))
	if p.GetKind() != NodeKindParenthesized {
		t.Error("wrong kind")
	}
}

func TestNewCommaExpression(t *testing.T) {
	a := NewIdentifierExpression("a", makeRange(0, 1), false)
	b := NewIdentifierExpression("b", makeRange(3, 4), false)
	c := NewCommaExpression([]Node{a, b}, makeRange(0, 4))
	if c.GetKind() != NodeKindComma {
		t.Error("wrong kind")
	}
	if len(c.Expressions) != 2 {
		t.Error("wrong expression count")
	}
}

func TestNewInstanceOfExpression(t *testing.T) {
	expr := NewIdentifierExpression("x", makeRange(0, 1), false)
	typ := NewNamedTypeNode(NewSimpleTypeName("Foo", makeRange(13, 16)), nil, false, makeRange(13, 16))
	io := NewInstanceOfExpression(expr, typ, makeRange(0, 16))
	if io.GetKind() != NodeKindInstanceOf {
		t.Error("wrong kind")
	}
}

func TestNewNewExpression(t *testing.T) {
	tn := NewSimpleTypeName("Foo", makeRange(4, 7))
	ne := NewNewExpression(tn, nil, nil, makeRange(0, 9))
	if ne.GetKind() != NodeKindNew {
		t.Error("wrong kind")
	}
}

// --- Node helpers ---

func TestIsLiteralKind(t *testing.T) {
	f := NewFloatLiteralExpression(1.0, makeRange(0, 3))
	if !IsLiteralKind(f, LiteralKindFloat) {
		t.Error("expected float literal")
	}
	if IsLiteralKind(f, LiteralKindInteger) {
		t.Error("should not match integer")
	}
}

func TestIsNumericLiteral(t *testing.T) {
	f := NewFloatLiteralExpression(1.0, makeRange(0, 3))
	i := NewIntegerLiteralExpression(42, makeRange(0, 2))
	s := NewStringLiteralExpression("x", makeRange(0, 3))
	id := NewIdentifierExpression("x", makeRange(0, 1), false)

	if !IsNumericLiteral(f) {
		t.Error("float should be numeric")
	}
	if !IsNumericLiteral(i) {
		t.Error("integer should be numeric")
	}
	if IsNumericLiteral(s) {
		t.Error("string should not be numeric")
	}
	if IsNumericLiteral(id) {
		t.Error("identifier should not be numeric")
	}
}

func TestCompilesToConst(t *testing.T) {
	if !CompilesToConst(NewFloatLiteralExpression(1.0, makeRange(0, 1))) {
		t.Error("float should be const")
	}
	if !CompilesToConst(NewIntegerLiteralExpression(1, makeRange(0, 1))) {
		t.Error("integer should be const")
	}
	if !CompilesToConst(NewStringLiteralExpression("x", makeRange(0, 1))) {
		t.Error("string should be const")
	}
	if !CompilesToConst(NewNullExpression(makeRange(0, 4))) {
		t.Error("null should be const")
	}
	if !CompilesToConst(NewTrueExpression(makeRange(0, 4))) {
		t.Error("true should be const")
	}
	if !CompilesToConst(NewFalseExpression(makeRange(0, 5))) {
		t.Error("false should be const")
	}
	if CompilesToConst(NewIdentifierExpression("x", makeRange(0, 1), false)) {
		t.Error("identifier should not be const")
	}
}

func TestIsAccessOnThis(t *testing.T) {
	this := NewThisExpression(makeRange(0, 4))
	prop := NewIdentifierExpression("x", makeRange(5, 6), false)
	access := NewPropertyAccessExpression(this, prop, makeRange(0, 6))
	if !IsAccessOnThis(access) {
		t.Error("property access on 'this' should return true")
	}

	// Call on this.x()
	call := NewCallExpression(access, nil, nil, makeRange(0, 8))
	if !IsAccessOnThis(call) {
		t.Error("call on 'this' property should return true")
	}

	other := NewIdentifierExpression("obj", makeRange(0, 3), false)
	otherAccess := NewPropertyAccessExpression(other, prop, makeRange(0, 6))
	if IsAccessOnThis(otherAccess) {
		t.Error("property access on non-this should return false")
	}
}

func TestIsAccessOnSuper(t *testing.T) {
	sup := NewSuperExpression(makeRange(0, 5))
	prop := NewIdentifierExpression("method", makeRange(6, 12), false)
	access := NewPropertyAccessExpression(sup, prop, makeRange(0, 12))
	if !IsAccessOnSuper(access) {
		t.Error("property access on 'super' should return true")
	}
}

func TestIsEmpty(t *testing.T) {
	if !IsEmpty(NewEmptyStatement(makeRange(0, 1))) {
		t.Error("empty statement should be empty")
	}
	if IsEmpty(NewIdentifierExpression("x", makeRange(0, 1), false)) {
		t.Error("identifier should not be empty")
	}
}

// --- Type nodes ---

func TestNewSimpleTypeName(t *testing.T) {
	tn := NewSimpleTypeName("i32", makeRange(0, 3))
	if tn.GetKind() != NodeKindTypeName {
		t.Error("wrong kind")
	}
	if tn.Identifier.Text != "i32" {
		t.Error("wrong text")
	}
	if tn.Next != nil {
		t.Error("next should be nil")
	}
}

func TestNewNamedTypeNode(t *testing.T) {
	name := NewSimpleTypeName("Array", makeRange(0, 5))
	arg := NewNamedTypeNode(NewSimpleTypeName("i32", makeRange(6, 9)), nil, false, makeRange(6, 9))
	nt := NewNamedTypeNode(name, []Node{arg}, false, makeRange(0, 10))
	if !nt.HasTypeArguments() {
		t.Error("should have type arguments")
	}
	if nt.IsNullable {
		t.Error("should not be nullable")
	}
	if nt.IsNull() {
		t.Error("should not be null type")
	}
}

func TestNamedTypeNodeIsNull(t *testing.T) {
	name := NewSimpleTypeName("null", makeRange(0, 4))
	nt := NewNamedTypeNode(name, nil, false, makeRange(0, 4))
	if !nt.IsNull() {
		t.Error("should be null type")
	}
}

func TestNewOmittedType(t *testing.T) {
	ot := NewOmittedType(makeRange(0, 0))
	if !IsTypeOmitted(ot) {
		t.Error("should be omitted")
	}
}

func TestIsTypeOmitted(t *testing.T) {
	named := NewNamedTypeNode(NewSimpleTypeName("i32", makeRange(0, 3)), nil, false, makeRange(0, 3))
	if IsTypeOmitted(named) {
		t.Error("i32 type should not be omitted")
	}
}

func TestNewFunctionTypeNode(t *testing.T) {
	ret := NewNamedTypeNode(NewSimpleTypeName("void", makeRange(10, 14)), nil, false, makeRange(10, 14))
	ft := NewFunctionTypeNode(nil, ret, nil, false, makeRange(0, 14))
	if ft.GetKind() != NodeKindFunctionType {
		t.Error("wrong kind")
	}
}

func TestNewTypeParameterNode(t *testing.T) {
	name := NewIdentifierExpression("T", makeRange(0, 1), false)
	tp := NewTypeParameterNode(name, nil, nil, makeRange(0, 1))
	if tp.GetKind() != NodeKindTypeParameter {
		t.Error("wrong kind")
	}
	if tp.Name.Text != "T" {
		t.Error("wrong name")
	}
}

func TestNewParameterNode(t *testing.T) {
	name := NewIdentifierExpression("x", makeRange(0, 1), false)
	typ := NewNamedTypeNode(NewSimpleTypeName("i32", makeRange(3, 6)), nil, false, makeRange(3, 6))
	p := NewParameterNode(ParameterKindDefault, name, typ, nil, makeRange(0, 6))
	if p.GetKind() != NodeKindParameter {
		t.Error("wrong kind")
	}
	if p.ParameterKind != ParameterKindDefault {
		t.Error("wrong parameter kind")
	}
}

func TestParameterNodeFlags(t *testing.T) {
	name := NewIdentifierExpression("x", makeRange(0, 1), false)
	typ := NewNamedTypeNode(NewSimpleTypeName("i32", makeRange(3, 6)), nil, false, makeRange(3, 6))
	p := NewParameterNode(ParameterKindDefault, name, typ, nil, makeRange(0, 6))
	p.Set(0x01) // some flag
	if !p.Is(0x01) {
		t.Error("flag should be set")
	}
	if !p.IsAny(0x01) {
		t.Error("flag should be set")
	}
	if p.Is(0x02) {
		t.Error("flag 0x02 should not be set")
	}
}

func TestHasGenericComponent(t *testing.T) {
	tpName := NewIdentifierExpression("T", makeRange(0, 1), false)
	tp := NewTypeParameterNode(tpName, nil, nil, makeRange(0, 1))

	// Named type "T" should match type parameter "T"
	tNode := NewNamedTypeNode(NewSimpleTypeName("T", makeRange(0, 1)), nil, false, makeRange(0, 1))
	if !HasGenericComponent(tNode, []*TypeParameterNode{tp}) {
		t.Error("T should have generic component")
	}

	// Named type "i32" should not match
	i32Node := NewNamedTypeNode(NewSimpleTypeName("i32", makeRange(0, 3)), nil, false, makeRange(0, 3))
	if HasGenericComponent(i32Node, []*TypeParameterNode{tp}) {
		t.Error("i32 should not have generic component")
	}
}

// --- Decorator ---

func TestDecoratorKindFromNode(t *testing.T) {
	tests := []struct {
		name string
		want DecoratorKind
	}{
		{"global", DecoratorKindGlobal},
		{"inline", DecoratorKindInline},
		{"external", DecoratorKindExternal},
		{"builtin", DecoratorKindBuiltin},
		{"lazy", DecoratorKindLazy},
		{"unsafe", DecoratorKindUnsafe},
		{"unmanaged", DecoratorKindUnmanaged},
		{"operator", DecoratorKindOperator},
		{"final", DecoratorKindFinal},
		{"custom", DecoratorKindCustom},
	}
	for _, tt := range tests {
		id := NewIdentifierExpression(tt.name, makeRange(0, int32(len(tt.name))), false)
		got := DecoratorKindFromNode(id)
		if got != tt.want {
			t.Errorf("DecoratorKindFromNode(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestDecoratorKindFromNodePropertyAccess(t *testing.T) {
	// operator.binary
	opExpr := NewIdentifierExpression("operator", makeRange(0, 8), false)
	binProp := NewIdentifierExpression("binary", makeRange(9, 15), false)
	pa := NewPropertyAccessExpression(opExpr, binProp, makeRange(0, 15))
	if DecoratorKindFromNode(pa) != DecoratorKindOperatorBinary {
		t.Error("operator.binary should return OperatorBinary")
	}

	// operator.prefix
	prefProp := NewIdentifierExpression("prefix", makeRange(9, 15), false)
	pa2 := NewPropertyAccessExpression(opExpr, prefProp, makeRange(0, 15))
	if DecoratorKindFromNode(pa2) != DecoratorKindOperatorPrefix {
		t.Error("operator.prefix should return OperatorPrefix")
	}

	// external.js
	extExpr := NewIdentifierExpression("external", makeRange(0, 8), false)
	jsProp := NewIdentifierExpression("js", makeRange(9, 11), false)
	pa3 := NewPropertyAccessExpression(extExpr, jsProp, makeRange(0, 11))
	if DecoratorKindFromNode(pa3) != DecoratorKindExternalJs {
		t.Error("external.js should return ExternalJs")
	}
}

func TestNewDecoratorNode(t *testing.T) {
	name := NewIdentifierExpression("inline", makeRange(1, 7), false)
	d := NewDecoratorNode(name, nil, makeRange(0, 7))
	if d.GetKind() != NodeKindDecorator {
		t.Error("wrong kind")
	}
	if d.DecoratorKind != DecoratorKindInline {
		t.Error("wrong decorator kind")
	}
}

// --- Comment ---

func TestNewCommentNode(t *testing.T) {
	c := NewCommentNode(tokenizer.CommentKindLine, "// hello", makeRange(0, 8))
	if c.GetKind() != NodeKindComment {
		t.Error("wrong kind")
	}
	if c.CommentKind != tokenizer.CommentKindLine {
		t.Error("wrong comment kind")
	}
	if c.Text != "// hello" {
		t.Error("wrong text")
	}
}

// --- Statements ---

func TestNewBlockStatement(t *testing.T) {
	b := NewBlockStatement(nil, makeRange(0, 2))
	if b.GetKind() != NodeKindBlock {
		t.Error("wrong kind")
	}
}

func TestNewBreakStatement(t *testing.T) {
	b := NewBreakStatement(nil, makeRange(0, 5))
	if b.GetKind() != NodeKindBreak {
		t.Error("wrong kind")
	}
	if b.Label != nil {
		t.Error("label should be nil")
	}
}

func TestNewContinueStatement(t *testing.T) {
	c := NewContinueStatement(nil, makeRange(0, 8))
	if c.GetKind() != NodeKindContinue {
		t.Error("wrong kind")
	}
}

func TestNewDoStatement(t *testing.T) {
	body := NewEmptyStatement(makeRange(3, 4))
	cond := NewTrueExpression(makeRange(12, 16))
	d := NewDoStatement(body, cond, makeRange(0, 17))
	if d.GetKind() != NodeKindDo {
		t.Error("wrong kind")
	}
}

func TestNewEmptyStatement(t *testing.T) {
	e := NewEmptyStatement(makeRange(0, 1))
	if e.GetKind() != NodeKindEmpty {
		t.Error("wrong kind")
	}
}

func TestNewExpressionStatement(t *testing.T) {
	expr := NewIdentifierExpression("foo", makeRange(0, 3), false)
	es := NewExpressionStatement(expr)
	if es.GetKind() != NodeKindExpression {
		t.Error("wrong kind")
	}
	// Range should match the expression's range
	if es.Range.Start != 0 || es.Range.End != 3 {
		t.Error("wrong range")
	}
}

func TestNewForStatement(t *testing.T) {
	body := NewEmptyStatement(makeRange(10, 11))
	f := NewForStatement(nil, nil, nil, body, makeRange(0, 11))
	if f.GetKind() != NodeKindFor {
		t.Error("wrong kind")
	}
}

func TestNewIfStatement(t *testing.T) {
	cond := NewTrueExpression(makeRange(4, 8))
	ifTrue := NewEmptyStatement(makeRange(10, 11))
	i := NewIfStatement(cond, ifTrue, nil, makeRange(0, 11))
	if i.GetKind() != NodeKindIf {
		t.Error("wrong kind")
	}
	if i.IfFalse != nil {
		t.Error("ifFalse should be nil")
	}
}

func TestNewReturnStatement(t *testing.T) {
	r := NewReturnStatement(nil, makeRange(0, 6))
	if r.GetKind() != NodeKindReturn {
		t.Error("wrong kind")
	}
}

func TestNewSwitchStatement(t *testing.T) {
	cond := NewIdentifierExpression("x", makeRange(8, 9), false)
	defCase := NewSwitchCase(nil, nil, makeRange(12, 20))
	if !defCase.IsDefault() {
		t.Error("should be default case")
	}
	sw := NewSwitchStatement(cond, []*SwitchCase{defCase}, makeRange(0, 22))
	if sw.GetKind() != NodeKindSwitch {
		t.Error("wrong kind")
	}
}

func TestNewThrowStatement(t *testing.T) {
	val := NewIdentifierExpression("err", makeRange(6, 9), false)
	th := NewThrowStatement(val, makeRange(0, 9))
	if th.GetKind() != NodeKindThrow {
		t.Error("wrong kind")
	}
}

func TestNewTryStatement(t *testing.T) {
	ts := NewTryStatement(nil, nil, nil, nil, makeRange(0, 10))
	if ts.GetKind() != NodeKindTry {
		t.Error("wrong kind")
	}
}

func TestNewVoidStatement(t *testing.T) {
	expr := NewIdentifierExpression("x", makeRange(5, 6), false)
	v := NewVoidStatement(expr, makeRange(0, 6))
	if v.GetKind() != NodeKindVoid {
		t.Error("wrong kind")
	}
}

func TestNewWhileStatement(t *testing.T) {
	cond := NewTrueExpression(makeRange(7, 11))
	body := NewEmptyStatement(makeRange(13, 14))
	w := NewWhileStatement(cond, body, makeRange(0, 14))
	if w.GetKind() != NodeKindWhile {
		t.Error("wrong kind")
	}
}

func TestNewModuleDeclaration(t *testing.T) {
	m := NewModuleDeclaration("mymod", 0, makeRange(0, 15))
	if m.GetKind() != NodeKindModule {
		t.Error("wrong kind")
	}
	if m.ModuleName != "mymod" {
		t.Error("wrong module name")
	}
}

// --- Declaration statements ---

func TestNewClassDeclaration(t *testing.T) {
	name := NewIdentifierExpression("Foo", makeRange(6, 9), false)
	c := NewClassDeclaration(name, nil, 0, nil, nil, nil, nil, makeRange(0, 12))
	if c.GetKind() != NodeKindClassDeclaration {
		t.Error("wrong kind")
	}
	if c.Name.Text != "Foo" {
		t.Error("wrong name")
	}
	if c.IsGeneric() {
		t.Error("should not be generic")
	}
}

func TestNewInterfaceDeclaration(t *testing.T) {
	name := NewIdentifierExpression("IFoo", makeRange(10, 14), false)
	i := NewInterfaceDeclaration(name, nil, 0, nil, nil, nil, nil, makeRange(0, 17))
	if i.GetKind() != NodeKindInterfaceDeclaration {
		t.Error("wrong kind")
	}
	// Should still be a *ClassDeclaration
	if i.Name.Text != "IFoo" {
		t.Error("wrong name")
	}
}

func TestNewEnumDeclaration(t *testing.T) {
	name := NewIdentifierExpression("Color", makeRange(5, 10), false)
	e := NewEnumDeclaration(name, nil, 0, nil, makeRange(0, 15))
	if e.GetKind() != NodeKindEnumDeclaration {
		t.Error("wrong kind")
	}
}

func TestNewEnumValueDeclaration(t *testing.T) {
	name := NewIdentifierExpression("Red", makeRange(0, 3), false)
	v := NewEnumValueDeclaration(name, 0, nil, makeRange(0, 3))
	if v.GetKind() != NodeKindEnumValueDeclaration {
		t.Error("wrong kind")
	}
}

func TestNewFieldDeclaration(t *testing.T) {
	name := NewIdentifierExpression("x", makeRange(0, 1), false)
	f := NewFieldDeclaration(name, nil, 0, nil, nil, makeRange(0, 5))
	if f.GetKind() != NodeKindFieldDeclaration {
		t.Error("wrong kind")
	}
	if f.ParameterIndex != -1 {
		t.Errorf("ParameterIndex = %d, want -1", f.ParameterIndex)
	}
}

func TestNewFunctionDeclaration(t *testing.T) {
	name := NewIdentifierExpression("foo", makeRange(9, 12), false)
	ret := NewNamedTypeNode(NewSimpleTypeName("void", makeRange(16, 20)), nil, false, makeRange(16, 20))
	sig := NewFunctionTypeNode(nil, ret, nil, false, makeRange(12, 20))
	f := NewFunctionDeclaration(name, nil, 0, nil, sig, nil, ArrowKindNone, makeRange(0, 22))
	if f.GetKind() != NodeKindFunctionDeclaration {
		t.Error("wrong kind")
	}
	if f.IsGeneric() {
		t.Error("should not be generic")
	}
}

func TestFunctionDeclarationClone(t *testing.T) {
	name := NewIdentifierExpression("foo", makeRange(0, 3), false)
	ret := NewNamedTypeNode(NewSimpleTypeName("void", makeRange(5, 9)), nil, false, makeRange(5, 9))
	sig := NewFunctionTypeNode(nil, ret, nil, false, makeRange(3, 9))
	f := NewFunctionDeclaration(name, nil, 0, nil, sig, nil, ArrowKindNone, makeRange(0, 9))
	clone := f.Clone()
	if clone.Name != f.Name {
		t.Error("cloned name should be same reference")
	}
	if clone.Signature != f.Signature {
		t.Error("cloned signature should be same reference")
	}
}

func TestNewMethodDeclaration(t *testing.T) {
	name := NewIdentifierExpression("bar", makeRange(0, 3), false)
	ret := NewNamedTypeNode(NewSimpleTypeName("void", makeRange(5, 9)), nil, false, makeRange(5, 9))
	sig := NewFunctionTypeNode(nil, ret, nil, false, makeRange(3, 9))
	m := NewMethodDeclaration(name, nil, 0, nil, sig, nil, makeRange(0, 9))
	if m.GetKind() != NodeKindMethodDeclaration {
		t.Error("wrong kind")
	}
	if m.ArrowKind != ArrowKindNone {
		t.Error("method should not be arrow")
	}
}

func TestNewImportDeclaration(t *testing.T) {
	foreign := NewIdentifierExpression("foo", makeRange(10, 13), false)
	d := NewImportDeclaration(foreign, nil, makeRange(10, 13))
	if d.GetKind() != NodeKindImportDeclaration {
		t.Error("wrong kind")
	}
	// name should default to foreignName
	if d.Name.Text != "foo" {
		t.Error("name should default to foreign name")
	}
}

func TestNewVariableDeclaration(t *testing.T) {
	name := NewIdentifierExpression("x", makeRange(4, 5), false)
	v := NewVariableDeclaration(name, nil, 0, nil, nil, makeRange(4, 5))
	if v.GetKind() != NodeKindVariableDeclaration {
		t.Error("wrong kind")
	}
}

func TestNewVariableStatement(t *testing.T) {
	name := NewIdentifierExpression("x", makeRange(4, 5), false)
	decl := NewVariableDeclaration(name, nil, 0, nil, nil, makeRange(4, 5))
	vs := NewVariableStatement(nil, []*VariableDeclaration{decl}, makeRange(0, 5))
	if vs.GetKind() != NodeKindVariable {
		t.Error("wrong kind")
	}
	if len(vs.Declarations) != 1 {
		t.Error("wrong declaration count")
	}
}

func TestNewNamespaceDeclaration(t *testing.T) {
	name := NewIdentifierExpression("ns", makeRange(10, 12), false)
	ns := NewNamespaceDeclaration(name, nil, 0, nil, makeRange(0, 15))
	if ns.GetKind() != NodeKindNamespaceDeclaration {
		t.Error("wrong kind")
	}
}

func TestNewTypeDeclaration(t *testing.T) {
	name := NewIdentifierExpression("MyType", makeRange(5, 11), false)
	typ := NewNamedTypeNode(NewSimpleTypeName("i32", makeRange(14, 17)), nil, false, makeRange(14, 17))
	td := NewTypeDeclaration(name, nil, 0, nil, typ, makeRange(0, 17))
	if td.GetKind() != NodeKindTypeDeclaration {
		t.Error("wrong kind")
	}
}

func TestDeclarationBaseFlags(t *testing.T) {
	name := NewIdentifierExpression("x", makeRange(0, 1), false)
	c := NewClassDeclaration(name, nil, 0, nil, nil, nil, nil, makeRange(0, 5))
	c.Set(0x04) // some flag
	if !c.Is(0x04) {
		t.Error("flag should be set")
	}
	if !c.IsAny(0x04) {
		t.Error("flag should match any")
	}
	if c.Is(0x08) {
		t.Error("flag 0x08 should not be set")
	}
}

func TestNewExportMember(t *testing.T) {
	local := NewIdentifierExpression("foo", makeRange(0, 3), false)
	em := NewExportMember(local, nil, makeRange(0, 3))
	if em.GetKind() != NodeKindExportMember {
		t.Error("wrong kind")
	}
	// exportedName defaults to localName
	if em.ExportedName.Text != "foo" {
		t.Error("exported name should default to local name")
	}
}

func TestNewExportImportStatement(t *testing.T) {
	name := NewIdentifierExpression("foo", makeRange(0, 3), false)
	ext := NewIdentifierExpression("bar", makeRange(4, 7), false)
	ei := NewExportImportStatement(name, ext, makeRange(0, 7))
	if ei.GetKind() != NodeKindExportImport {
		t.Error("wrong kind")
	}
}

func TestNewExportDefaultStatement(t *testing.T) {
	name := NewIdentifierExpression("Foo", makeRange(22, 25), false)
	decl := NewClassDeclaration(name, nil, 0, nil, nil, nil, nil, makeRange(15, 28))
	ed := NewExportDefaultStatement(decl, makeRange(0, 28))
	if ed.GetKind() != NodeKindExportDefault {
		t.Error("wrong kind")
	}
}

func TestNewIndexSignatureNode(t *testing.T) {
	keyType := NewNamedTypeNode(NewSimpleTypeName("string", makeRange(1, 7)), nil, false, makeRange(1, 7))
	valueType := NewNamedTypeNode(NewSimpleTypeName("i32", makeRange(10, 13)), nil, false, makeRange(10, 13))
	is := NewIndexSignatureNode(keyType, valueType, 0, makeRange(0, 14))
	if is.GetKind() != NodeKindIndexSignature {
		t.Error("wrong kind")
	}
}

// --- Source ---

func TestNewSource(t *testing.T) {
	src := NewSource(SourceKindUser, "test.ts", "let x = 5;")
	if src.GetKind() != NodeKindSource {
		t.Error("wrong kind")
	}
	if src.SourceKind != SourceKindUser {
		t.Error("wrong source kind")
	}
	if src.NormalizedPath != "test.ts" {
		t.Error("wrong normalized path")
	}
	if src.InternalPath != "test" {
		t.Errorf("InternalPath = %q, want %q", src.InternalPath, "test")
	}
	if src.SimplePath != "test" {
		t.Errorf("SimplePath = %q, want %q", src.SimplePath, "test")
	}
	if src.Range.Start != 0 || src.Range.End != 10 {
		t.Errorf("Range = {%d,%d}, want {0,10}", src.Range.Start, src.Range.End)
	}
	if src.Range.Source != src {
		t.Error("source range should reference self")
	}
	if src.DebugInfoIndex != -1 {
		t.Error("debug info index should be -1")
	}
}

func TestSourceImplementsDiagnosticsSource(t *testing.T) {
	src := NewSource(SourceKindUser, "test.ts", "let x = 5;")
	var ds diagnostics.Source = src
	if ds.SourceText() != "let x = 5;" {
		t.Error("wrong source text")
	}
	if ds.SourceNormalizedPath() != "test.ts" {
		t.Error("wrong normalized path")
	}
}

func TestSourceLineAt(t *testing.T) {
	src := NewSource(SourceKindUser, "test.ts", "line1\nline2\nline3")
	if src.LineAt(0) != 1 {
		t.Error("pos 0 should be line 1")
	}
	if src.ColumnAt() != 1 {
		t.Error("col at pos 0 should be 1")
	}
	if src.LineAt(6) != 2 {
		t.Errorf("pos 6 should be line 2, got %d", src.LineAt(6))
	}
	if src.ColumnAt() != 1 {
		t.Errorf("col at pos 6 should be 1, got %d", src.ColumnAt())
	}
	if src.LineAt(8) != 2 {
		t.Errorf("pos 8 should be line 2, got %d", src.LineAt(8))
	}
	if src.ColumnAt() != 3 {
		t.Errorf("col at pos 8 should be 3, got %d", src.ColumnAt())
	}
	if src.LineAt(12) != 3 {
		t.Errorf("pos 12 should be line 3, got %d", src.LineAt(12))
	}
}

func TestSourceIsLibrary(t *testing.T) {
	user := NewSource(SourceKindUser, "test.ts", "")
	if user.IsLibrary() {
		t.Error("user source should not be library")
	}
	lib := NewSource(SourceKindLibrary, "~lib/test.ts", "")
	if !lib.IsLibrary() {
		t.Error("library source should be library")
	}
	libEntry := NewSource(SourceKindLibraryEntry, "~lib/entry.ts", "")
	if !libEntry.IsLibrary() {
		t.Error("library entry source should be library")
	}
}

func TestNativeSource(t *testing.T) {
	n := NativeSource()
	if n.SourceKind != SourceKindLibraryEntry {
		t.Error("native source should be library entry")
	}
	if !n.IsNative() {
		t.Error("native source should report as native")
	}
	// Should be singleton
	n2 := NativeSource()
	if n != n2 {
		t.Error("native source should be singleton")
	}
}

func TestSourceInternalPathWithSlash(t *testing.T) {
	src := NewSource(SourceKindUser, "src/lib/", "")
	if src.InternalPath != "src/lib/index" {
		t.Errorf("InternalPath = %q, want %q", src.InternalPath, "src/lib/index")
	}
}

func TestSourceSimplePathWithDelimiter(t *testing.T) {
	src := NewSource(SourceKindUser, "src/lib/foo.ts", "")
	if src.SimplePath != "foo" {
		t.Errorf("SimplePath = %q, want %q", src.SimplePath, "foo")
	}
}

// --- Helpers ---

func TestMangleInternalPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"foo.ts", "foo"},
		{"src/bar.ts", "src/bar"},
		{"path/to/", "path/to/index"},
		{"already", "already"},
		{"", ""},
	}
	for _, tt := range tests {
		got := MangleInternalPath(tt.input)
		if got != tt.want {
			t.Errorf("MangleInternalPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFindDecorator(t *testing.T) {
	name := NewIdentifierExpression("inline", makeRange(1, 7), false)
	d := NewDecoratorNode(name, nil, makeRange(0, 7))

	found := FindDecorator(DecoratorKindInline, []*DecoratorNode{d})
	if found != d {
		t.Error("should find inline decorator")
	}

	notFound := FindDecorator(DecoratorKindGlobal, []*DecoratorNode{d})
	if notFound != nil {
		t.Error("should not find global decorator")
	}

	nilResult := FindDecorator(DecoratorKindInline, nil)
	if nilResult != nil {
		t.Error("should return nil for nil decorators")
	}
}

func TestCompiledExpression(t *testing.T) {
	ce := NewCompiledExpression(42, "i32", makeRange(0, 1))
	if ce.GetKind() != NodeKindCompiled {
		t.Error("wrong kind")
	}
	if ce.Expr != 42 {
		t.Error("wrong expr")
	}
}
