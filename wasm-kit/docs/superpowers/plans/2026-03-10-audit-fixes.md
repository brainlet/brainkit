# Audit Fixes Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all 26 verified bugs found by the 30-agent codebase audit comparing the Go port against the original AssemblyScript TypeScript source.

**Architecture:** Each fix is a faithful 1:1 port correction — read the TS source, match the Go code to it exactly. No invention, no optimization, no abstraction. Every change must trace back to a TS line.

**Tech Stack:** Go, TypeScript (reference only), Binaryen (CGo module interface)

**Source Locations:**
- **TS Original**: `/Users/davidroman/Documents/code/clones/assemblyscript/src/`
- **Go Port**: `/Users/davidroman/Documents/code/brainlet/brainkit/wasm-kit/`
- **Audit Reports**: `/Users/davidroman/Documents/code/brainlet/brainkit-maps/binaryen-to-wasm-kit/.audits/`

**Mandatory Pre-Reading (for every task):**
- `/Users/davidroman/Documents/code/brainlet/brainkit-maps/binaryen-to-wasm-kit/PORTING_RULES.md`
- `/Users/davidroman/Documents/code/brainlet/brainkit-maps/binaryen-to-wasm-kit/FILE_MAP.md`
- `/Users/davidroman/Documents/code/brainlet/brainkit-maps/binaryen-to-wasm-kit/DEPENDENCY_GRAPH.md`

---

## Chunk 1: Showstopper + Critical Compiler Infrastructure

These 4 fixes unblock the entire builtin system and fix the most-called compiler function.

---

### Task 1: S1 — Port `compileCallExpressionBuiltin` dispatch (SHOWSTOPPER)

**Impact:** ALL ~549 builtin handlers are dead code. Every builtin call (sizeof, assert, memory ops, SIMD, atomics) hits "Not implemented".

**Files:**
- Modify: `compiler/compile.go:1954-1967`
- Reference: TS `compiler.ts:6215-6268`

- [ ] **Step 1: Read the TS source**

Read `compiler.ts:6215-6268`. The function:
1. Checks `prototype.hasDecorator(DecoratorFlags.Unsafe)` → calls `this.checkUnsafe(expression)`
2. Resolves type arguments if present (lines 6222-6243)
3. Builds a `BuiltinFunctionContext` (lines 6245-6256)
4. Computes `internalName` — for instance builtins: `parent.prototype.internalName + "#" + prototype.name`; otherwise: `prototype.internalName` (lines 6258-6264)
5. Asserts handler exists in `builtinFunctions` map, calls it (lines 6265-6267)

- [ ] **Step 2: Read existing Go infrastructure**

Verify these exist (they do, confirmed by audit):
- `BuiltinFunctionContext` struct in `compiler/builtins_context.go`
- `builtinFunctions` map in `compiler/builtins_types.go:37`
- `GetBuiltinHandler()` in `compiler/builtins.go:32`
- `Compiler.checkUnsafe()` — search for it in compile.go

- [ ] **Step 3: Replace the stub**

Replace `compiler/compile.go:1954-1967` with the faithful port:

```go
func (c *Compiler) compileCallExpressionBuiltin(
	prototype *program.FunctionPrototype,
	expression *ast.CallExpression,
	contextualType *types.Type,
) module.ExpressionRef {
	// Check @unsafe decorator
	if prototype.HasDecorator(common.DecoratorFlagsUnsafe) {
		c.checkUnsafe(expression)
	}

	var typeArguments []*types.Type

	// Builtins handle omitted type arguments on their own. If present,
	// resolve them here and pass them to the builtin.
	typeParameterNodes := prototype.TypeParameterNodes()
	typeArgumentNodes := expression.TypeArguments()
	if typeArgumentNodes != nil {
		if !prototype.Is(common.CommonFlagsGeneric) {
			c.Error(
				diagnostics.DiagnosticCodeType0IsNotGeneric,
				expression.GetRange(),
				prototype.InternalName(), "", "",
			)
		}
		ctxTypeArgs := make(map[string]*types.Type)
		for k, v := range c.CurrentFlow.ContextualTypeArguments() {
			ctxTypeArgs[k] = v
		}
		typeArguments = c.Program.Resolver.ResolveTypeArguments(
			typeParameterNodes,
			typeArgumentNodes,
			c.CurrentFlow,
			c.CurrentFlow.SourceFunction().Parent(),
			ctxTypeArgs,
			expression,
		)
	}

	// Build the context
	callee := expression.Expression()
	var thisExpression ast.Expression
	if callee.Kind() == ast.NodeKindPropertyAccess {
		thisExpression = callee.(*ast.PropertyAccessExpression).Expression()
	}
	ctx := &BuiltinFunctionContext{
		Compiler:       c,
		Prototype:      prototype,
		TypeArguments:  typeArguments,
		Operands:       expression.Args(),
		ThisOperand:    thisExpression,
		ContextualType: contextualType,
		ReportNode:     expression,
		IsCompile:      false,
	}

	// Compute internal name for dispatch
	var internalName string
	if prototype.Is(common.CommonFlagsInstance) {
		parent := prototype.GetBoundClassOrInterface()
		internalName = parent.Prototype().InternalName() + "#" + prototype.Name()
	} else {
		internalName = prototype.InternalName()
	}

	// Dispatch to handler
	fn := GetBuiltinHandler(internalName)
	if fn == nil {
		panic("missing builtin handler for: " + internalName)
	}
	return fn(ctx)
}
```

**IMPORTANT:** The exact field names and method signatures above are approximations. You MUST:
1. Check the actual `BuiltinFunctionContext` struct fields in `builtins_context.go` / `builtins_types.go`
2. Check the actual `FunctionPrototype` methods in `program/`
3. Check the actual `CallExpression` methods in `ast/`
4. Adjust field/method names to match the existing Go codebase exactly

- [ ] **Step 4: Verify it compiles**

Run: `cd /Users/davidroman/Documents/code/brainlet/brainkit/wasm-kit && go build ./...`
Expected: No compilation errors.

- [ ] **Step 5: Commit**

```bash
git add compiler/compile.go
git commit -m "fix(S1): port compileCallExpressionBuiltin dispatch from TS

Replaces the stub with faithful port of compiler.ts:6215-6268.
Unlocks all ~549 builtin handlers that were previously dead code."
```

---

### Task 2: C1 — Fix `CompileFile` slice aliasing bug

**Impact:** Expressions added by `CompileGlobal`/`CompileEnum` to start function body are silently lost when `append` reallocates.

**Files:**
- Modify: `compiler/compile_file.go:45-72`
- Modify: `compiler/compile_file.go:243` (any `c.CurrentBody = append(...)` usage)
- Modify: `compiler/compile_global.go:283` (any `c.CurrentBody = append(...)` usage)
- Modify: `compiler/compiler.go:29` (field declaration)
- Modify: `compiler/compiler.go:198` (field initialization)
- Reference: TS `compiler.ts:1097-1138` — in TS, both variables reference the same JS array object

**Root cause:** Go slices are value types. When `c.CurrentBody = append(c.CurrentBody, expr)` causes reallocation, the local `startFunctionBody` variable still points to the old backing array. The TS equivalent uses JS arrays which are reference types — both variables always point to the same object.

- [ ] **Step 1: Change `CurrentBody` field to pointer-to-slice**

In `compiler/compiler.go:29`, change:
```go
// FROM:
CurrentBody []module.ExpressionRef
// TO:
CurrentBody *[]module.ExpressionRef
```

- [ ] **Step 2: Update initialization in compiler.go**

In `compiler/compiler.go:198`, change:
```go
// FROM:
c.CurrentBody = make([]module.ExpressionRef, 0)
// TO:
body := make([]module.ExpressionRef, 0)
c.CurrentBody = &body
```

- [ ] **Step 3: Update CompileFile in compile_file.go**

In `compiler/compile_file.go:45-72`, change to use pointer semantics:
```go
previousBody := c.CurrentBody
startFunctionBody := make([]module.ExpressionRef, 0)
c.CurrentBody = &startFunctionBody
// ... compilation ...
c.CurrentBody = previousBody

// Check length via dereferencing:
if len(*c.CurrentBody) > 0 { ... }
// When passing to Flatten:
mod.Flatten(*c.CurrentBody, module.TypeRefNone)
```

Wait — actually there's a simpler approach that matches TS semantics more closely. The TS uses `startFunctionBody` as a local alias that stays in sync with `this.currentBody`. We can achieve the same by always going through the pointer:

```go
previousBody := c.CurrentBody
startFunctionBody := make([]module.ExpressionRef, 0)
c.CurrentBody = &startFunctionBody

// ... compilation happens, appending via *c.CurrentBody ...

c.CurrentBody = previousBody

// Now startFunctionBody has all appended expressions because
// c.CurrentBody was a pointer to startFunctionBody
if len(startFunctionBody) > 0 {
    // ...
}
```

- [ ] **Step 4: Update ALL append sites**

Search for every `c.CurrentBody = append(c.CurrentBody,` and change to:
```go
*c.CurrentBody = append(*c.CurrentBody, expr)
```

Files to check:
- `compiler/compile_file.go:243`
- `compiler/compile_global.go:283`
- Any other file doing `c.CurrentBody = append(...)`

Also search for every read of `c.CurrentBody` (e.g., `len(c.CurrentBody)`, passing it to functions) and dereference:
```go
len(*c.CurrentBody)
// or
mod.Flatten(*c.CurrentBody, ...)
```

- [ ] **Step 5: Verify it compiles**

Run: `cd /Users/davidroman/Documents/code/brainlet/brainkit/wasm-kit && go build ./...`

- [ ] **Step 6: Commit**

```bash
git add compiler/compiler.go compiler/compile_file.go compiler/compile_global.go
git commit -m "fix(C1): change CurrentBody to *[]ExpressionRef to prevent slice aliasing

In Go, append() can reallocate the backing array, causing local variables
that alias the slice to diverge. TS arrays are reference types so both
variables always point to the same object. Using a pointer-to-slice
matches TS reference semantics."
```

---

### Task 3: C2 — Fix `CompileExpression` missing initialization and post-processing

**Impact:** Most-called function in compiler. Missing 4 critical behaviors.

**Files:**
- Modify: `compiler/compile_expression.go:24-38`
- Reference: TS `compiler.ts:3431-3542`

- [ ] **Step 1: Read the full TS `compileExpression`**

Read `compiler.ts:3431-3542`. Key behaviors:
- Line 3436-3438: While-loop to unwrap Parenthesized nodes
- Line 3439: `this.currentType = contextualType`
- Line 3440: `if (contextualType == Type.void) constraints |= Constraints.WillDrop`
- Line 3442-3524: Switch on expression kind
- Line 3526-3527: `let currentType = this.currentType; let wrap = (constraints & Constraints.MustWrap) != 0;`
- Line 3528: `if (currentType != contextualType.nonNullableType)` — nullable-aware comparison
- Lines 3529-3535: ConvExplicit / ConvImplicit conversion
- Line 3537: `if (wrap) expr = this.ensureSmallIntegerWrap(expr, currentType);`
- Line 3540: Source map debug location

- [ ] **Step 2: Read current Go `CompileExpression`**

Read `compiler/compile_expression.go:24-38`. Current Go code:
```go
func (c *Compiler) CompileExpression(expression ast.Expression, contextualType *types.Type, constraints int32) module.ExpressionRef {
    expr := c.compileExpressionInner(expression, contextualType, constraints)
    ct := c.CurrentType
    if ct != contextualType && contextualType != types.TypeVoid {
        if constraints&(ConstraintsConvImplicit|ConstraintsConvExplicit) != 0 {
            expr = c.convertExpression(expr, ct, contextualType, constraints&ConstraintsConvExplicit != 0, expression)
            c.CurrentType = contextualType
        }
    }
    return expr
}
```

- [ ] **Step 3: Add pre-dispatch initialization**

Add before the `compileExpressionInner` call:
```go
// Skip parenthesized wrappers (TS lines 3436-3438)
for expression.Kind() == ast.NodeKindParenthesized {
    expression = expression.(*ast.ParenthesizedExpression).Expression()
}

// Set currentType default (TS line 3439)
c.CurrentType = contextualType

// Auto-add WillDrop for void context (TS line 3440)
if contextualType == types.TypeVoid {
    constraints |= ConstraintsWillDrop
}
```

**Note:** If a `compileParenthesizedExpression` case exists in `compileExpressionInner`, it should be removed or made to just extract the inner expression (since the while-loop now handles unwrapping before dispatch).

- [ ] **Step 4: Fix post-dispatch nullable comparison and MustWrap**

Replace the post-dispatch logic with:
```go
ct := c.CurrentType
wrap := constraints&ConstraintsMustWrap != 0

// Allow assigning non-nullable to nullable (TS line 3528)
if ct != contextualType.NonNullableType() {
    if constraints&ConstraintsConvExplicit != 0 {
        expr = c.convertExpression(expr, ct, contextualType, true, expression)
        ct = c.CurrentType
    } else if constraints&ConstraintsConvImplicit != 0 {
        expr = c.convertExpression(expr, ct, contextualType, false, expression)
        ct = c.CurrentType
    }
}

// Ensure small integer wrapping (TS line 3537)
if wrap {
    expr = c.ensureSmallIntegerWrap(expr, ct)
}
```

**IMPORTANT:** Verify that `types.Type` has a `NonNullableType()` method. If it's named differently (e.g., `NonNullable()`), use the correct name.

- [ ] **Step 5: Verify `ensureSmallIntegerWrap` exists**

Confirm `ensureSmallIntegerWrap` is at `compiler/compile.go:2062` and has signature:
```go
func (c *Compiler) ensureSmallIntegerWrap(expr module.ExpressionRef, typ *types.Type) module.ExpressionRef
```

- [ ] **Step 6: Verify it compiles**

Run: `go build ./...`

- [ ] **Step 7: Commit**

```bash
git add compiler/compile_expression.go
git commit -m "fix(C2): add CompileExpression pre-init and post-processing

Port 4 missing behaviors from compiler.ts:3431-3542:
1. currentType = contextualType before dispatch
2. WillDrop constraint when contextualType is void
3. MustWrap post-switch ensureSmallIntegerWrap
4. Nullable-aware type comparison (NonNullableType)"
```

---

### Task 4: C3 — Fix boolean condition constraints in all control flow

**Impact:** Every do/for/if/while passes wrong constraints for condition compilation.

**Files:**
- Modify: `compiler/compile_statement.go` — lines 231, 310, 427, 924
- Reference: TS `compiler.ts` — condition compilation uses `Type.bool` with no third argument (defaults to `Constraints.None`)

- [ ] **Step 1: Verify the 4 sites**

In `compiler/compile_statement.go`, find:
1. `compileDoStatement` ~line 231: condition compilation
2. `compileForStatement` ~line 310: condition compilation
3. `compileIfStatement` ~line 427: condition compilation
4. `compileWhileStatement` ~line 924: condition compilation

Each currently passes `ConstraintsConvImplicit` (value = 1) as the third argument to `CompileExpression` for the condition.

- [ ] **Step 2: Read TS condition compilation**

In TS, all 4 sites compile conditions like:
```typescript
let condExpr = this.compileExpression(statement.condition, Type.bool);
```
The third parameter `constraints` defaults to `Constraints.None` (value = 0).

- [ ] **Step 3: Change all 4 sites**

At each site, change:
```go
// FROM:
c.CompileExpression(condition, types.TypeBool, ConstraintsConvImplicit)
// TO:
c.CompileExpression(condition, types.TypeBool, ConstraintsNone)
```

- [ ] **Step 4: Verify it compiles**

Run: `go build ./...`

- [ ] **Step 5: Commit**

```bash
git add compiler/compile_statement.go
git commit -m "fix(C3): use ConstraintsNone for boolean conditions

All 4 control flow condition sites (do/for/if/while) were passing
ConstraintsConvImplicit instead of ConstraintsNone. TS defaults the
constraints parameter to Constraints.None for condition compilation."
```

---

## Chunk 2: Critical Resolver + Expression Operator Fixes

---

### Task 5: C4 — Fix `GetElementOfType` to delegate to `GetClassOrWrapper`

**Impact:** Returns different elements than TS due to inlined/duplicated logic.

**Files:**
- Modify: `program/resolver.go:3448-3468`
- Reference: TS `resolver.ts:886-890`

- [ ] **Step 1: Read TS source**

```typescript
getElementOfType(type: Type): Element | null {
    let classReference = type.getClassOrWrapper(this.program);
    if (classReference) return classReference;
    return null;
}
```

- [ ] **Step 2: Replace the Go function**

Replace `resolver.go:3448-3468` with:
```go
func (r *Resolver) GetElementOfType(typ *types.Type) Element {
	if typ == nil {
		return nil
	}
	classRef := typ.GetClassOrWrapper(r.program)
	if classRef != nil {
		return classRef
	}
	return nil
}
```

**Note:** Check if `GetClassOrWrapper` returns a `*Class` or an `Element`. If it returns `*Class`, you may need a cast. The TS `getClassOrWrapper` returns `Class | null` and `Class` extends `Element`, so a direct return works.

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add program/resolver.go
git commit -m "fix(C4): delegate GetElementOfType to GetClassOrWrapper

Remove inlined/duplicated logic. TS resolver.ts:886-890 is a 3-line
delegator to type.getClassOrWrapper(this.program)."
```

---

### Task 6: C5 — Remove invented fallbacks from `findConstructorPrototype`

**Impact:** Can find constructors the TS would not, affecting generic class type inference.

**Files:**
- Modify: `program/resolver.go:1309-1326`
- Reference: TS `resolver.ts:3591-3594`

- [ ] **Step 1: Read TS source**

```typescript
let constructorPrototype: FunctionPrototype | null = null;
for (let p: ClassPrototype | null = prototype; p && !constructorPrototype; p = p.basePrototype) {
    constructorPrototype = p.constructorPrototype;
}
```

- [ ] **Step 2: Replace the Go function**

Replace `resolver.go:1309-1326` with:
```go
func (r *Resolver) findConstructorPrototype(prototype *ClassPrototype) *FunctionPrototype {
	for current := prototype; current != nil; current = current.BasePrototype {
		if current.ConstructorPrototype != nil {
			return current.ConstructorPrototype
		}
	}
	return nil
}
```

This removes the two invented fallback paths that searched `InstanceMembers` and `GetMembers()`.

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`

- [ ] **Step 4: Commit**

```bash
git add program/resolver.go
git commit -m "fix(C5): remove invented fallbacks from findConstructorPrototype

TS resolver.ts:3591-3594 only checks p.constructorPrototype in a loop.
Go had two extra InstanceMembers/GetMembers lookups not in TS."
```

---

### Task 7: C6 — Add missing types to `makeBinaryEq` / `makeBinaryNe`

**Impact:** V128, Struct, Array, I31, String equality comparisons produce unreachable.

**Files:**
- Modify: `compiler/compile_expression.go` — `makeBinaryEq` (~line 3895) and `makeBinaryNe` (~line 3920)
- Reference: TS `compiler.ts` — `makeEq` (lines 4913-4961) and `makeNe` (lines 4963-5019)

- [ ] **Step 1: Add `reportNode` parameter to both functions**

TS signatures: `makeEq(leftExpr, rightExpr, type, reportNode)` / `makeNe(leftExpr, rightExpr, type, reportNode)`.

Add `reportNode ast.Node` as a fourth parameter to both Go functions. Update all call sites.

- [ ] **Step 2: Add missing cases to `makeBinaryEq`**

After the F64 case and before the default, add:
```go
case types.TypeKindV128:
    return mod.Unary(module.UnaryOpAllTrueI8x16,
        mod.Binary(module.BinaryOpEqI8x16, left, right))

case types.TypeKindEq, types.TypeKindStruct, types.TypeKindArray, types.TypeKindI31:
    return mod.RefEq(left, right)

case types.TypeKindString:
    return mod.StringEq(left, right)

case types.TypeKindStringviewWTF8, types.TypeKindStringviewWTF16,
     types.TypeKindStringviewIter, types.TypeKindFunc,
     types.TypeKindExtern, types.TypeKindAny:
    c.Error(
        diagnostics.DiagnosticCodeOperationNotSupported,
        reportNode.GetRange(),
        "ref.eq", typ.String(), "",
    )
    return mod.Unreachable()
```

**IMPORTANT:** Verify the exact type kind constants, module method names (`RefEq`, `StringEq`), and diagnostic code name in the existing codebase.

- [ ] **Step 3: Add missing cases to `makeBinaryNe`**

Same cases but with inverted logic:
```go
case types.TypeKindV128:
    return mod.Unary(module.UnaryOpAnyTrueV128,
        mod.Binary(module.BinaryOpNeI8x16, left, right))

case types.TypeKindEq, types.TypeKindStruct, types.TypeKindArray, types.TypeKindI31:
    return mod.Unary(module.UnaryOpEqzI32, mod.RefEq(left, right))

case types.TypeKindString:
    return mod.Unary(module.UnaryOpEqzI32, mod.StringEq(left, right))

case types.TypeKindStringviewWTF8, types.TypeKindStringviewWTF16,
     types.TypeKindStringviewIter, types.TypeKindFunc,
     types.TypeKindExtern, types.TypeKindAny:
    c.Error(
        diagnostics.DiagnosticCodeOperationNotSupported,
        reportNode.GetRange(),
        "ref.eq", typ.String(), "",
    )
    return mod.Unreachable()
```

- [ ] **Step 4: Verify it compiles**

Run: `go build ./...`

- [ ] **Step 5: Commit**

```bash
git add compiler/compile_expression.go
git commit -m "fix(C6): add V128, ref types, String to makeBinaryEq/Ne

Port missing type cases from compiler.ts:4913-5019.
Adds V128 (lane-wise eq), Struct/Array/I31/Eq (ref_eq),
String (string_eq), and error diagnostics for unsupported types."
```

---

### Task 8: C7 — Add float remainder via `Math.mod` / `Mathf.mod`

**Impact:** Float remainder operations produce unreachable.

**Files:**
- Modify: `compiler/compile_expression.go` — `makeBinaryRem` (~line 3722)
- Modify: `compiler/compiler.go` — add cached fields `f32ModInstance` and `f64ModInstance`
- Reference: TS `compiler.ts:5376-5429`

- [ ] **Step 1: Add `reportNode` parameter to `makeBinaryRem`**

Add `reportNode ast.Node` parameter. Update all call sites.

- [ ] **Step 2: Add cached fields to Compiler struct**

In `compiler/compiler.go`, add:
```go
f32ModInstance *program.Function
f64ModInstance *program.Function
```

- [ ] **Step 3: Add F32 and F64 cases**

Before the default case in `makeBinaryRem`, add (adapted from TS lines 5376-5429):

```go
case types.TypeKindF32:
    instance := c.f32ModInstance
    if instance == nil {
        mathf := c.Program.Lookup(common.CommonNameMathf)
        if mathf == nil {
            c.Error(diagnostics.DiagnosticCodeCannotFind0, reportNode.GetRange(),
                "Mathf", "", "")
            return mod.Unreachable()
        }
        modMember := mathf.GetMember("mod")
        if modMember == nil {
            c.Error(diagnostics.DiagnosticCodeCannotFind0, reportNode.GetRange(),
                "Mathf.mod", "", "")
            return mod.Unreachable()
        }
        modProto := modMember.(*program.FunctionPrototype)
        instance = c.Program.Resolver.ResolveFunction(modProto, nil, ReportModeReport)
        if instance == nil {
            return mod.Unreachable()
        }
        c.f32ModInstance = instance
    }
    return c.makeCallDirect(instance, []module.ExpressionRef{left, right}, reportNode)

case types.TypeKindF64:
    instance := c.f64ModInstance
    if instance == nil {
        math := c.Program.Lookup(common.CommonNameMath)
        if math == nil {
            c.Error(diagnostics.DiagnosticCodeCannotFind0, reportNode.GetRange(),
                "Math", "", "")
            return mod.Unreachable()
        }
        modMember := math.GetMember("mod")
        if modMember == nil {
            c.Error(diagnostics.DiagnosticCodeCannotFind0, reportNode.GetRange(),
                "Math.mod", "", "")
            return mod.Unreachable()
        }
        modProto := modMember.(*program.FunctionPrototype)
        instance = c.Program.Resolver.ResolveFunction(modProto, nil, ReportModeReport)
        if instance == nil {
            return mod.Unreachable()
        }
        c.f64ModInstance = instance
    }
    return c.makeCallDirect(instance, []module.ExpressionRef{left, right}, reportNode)
```

**IMPORTANT:** Verify the exact method names (`Lookup`, `GetMember`, `ResolveFunction`, `makeCallDirect`) against the existing codebase. The TS code at lines 5376-5429 is the authoritative reference.

- [ ] **Step 4: Verify it compiles**

Run: `go build ./...`

- [ ] **Step 5: Commit**

```bash
git add compiler/compile_expression.go compiler/compiler.go
git commit -m "fix(C7): add F32/F64 float remainder via Mathf.mod/Math.mod

Port from compiler.ts:5376-5429. Float % operator requires calling
runtime Mathf.mod (f32) or Math.mod (f64) functions. Adds cached
instance fields to avoid re-resolving every time."
```

---

## Chunk 3: HIGH Priority — Compiler Core Bugs (H1, H2, H6, H7)

---

### Task 9: H1 — Fix `makeCallInline` nonnull flow check

**Impact:** Checks caller's flow instead of inlined function's flow for nonnull.

**Files:**
- Modify: `compiler/compile.go:1261`
- Reference: TS `compiler.ts:6482`

- [ ] **Step 1: Read the context**

Read `compiler/compile.go` around line 1258-1265. You'll see:
```go
if previousFlow.CanOverflow(paramExpr, paramType) {  // correct — uses previousFlow
    ...
}
if previousFlow.IsNonnull(paramExpr, paramType) {    // BUG — should be fl
    ...
}
```

- [ ] **Step 2: Fix the check**

Change line 1261 from:
```go
if previousFlow.IsNonnull(paramExpr, paramType) {
```
to:
```go
if fl.IsNonnull(paramExpr, paramType) {
```

Where `fl` is the inline flow variable (should already exist in scope — verify the variable name).

- [ ] **Step 3: Verify and commit**

```bash
go build ./... && git add compiler/compile.go && git commit -m "fix(H1): use inline flow for IsNonnull in makeCallInline

compiler.ts:6482 uses flow.isNonnull (inline flow), not previousFlow.
The canOverflow check correctly uses previousFlow (TS line 6481),
but IsNonnull must use the new inline flow."
```

---

### Task 10: H2 — Fix `makeRuntimeDowncastCheck` class lookup

**Impact:** Uses `GetClassOrWrapper` which can wrap basic types; TS uses direct `classReference`.

**Files:**
- Modify: `compiler/compile.go:2159`
- Reference: TS `compiler.ts:10602`

- [ ] **Step 1: Fix the lookup**

Change line 2159 from:
```go
classRef := toType.GetClassOrWrapper(c.Program)
```
to:
```go
classRef := toType.GetClass()
```

The TS uses `toType.classReference!` which is the direct class field, equivalent to Go's `GetClass()`.

- [ ] **Step 2: Verify and commit**

```bash
go build ./... && git add compiler/compile.go && git commit -m "fix(H2): use GetClass instead of GetClassOrWrapper in downcast check

compiler.ts:10602 uses toType.classReference (direct class). Using
GetClassOrWrapper could return wrapper classes for basic types."
```

---

### Task 11: H6 — Port `ensureRuntimeFunction` properly

**Impact:** Returns table index instead of memory address. Missing all runtime function resolution logic.

**Files:**
- Modify: `compiler/compile_global.go:481-492`
- Reference: TS `compiler.ts:2124-2145`

- [ ] **Step 1: Read the TS implementation**

Read `compiler.ts:2124-2145`. The TS:
1. Asserts function is compiled and not a stub
2. Checks `instance.memorySegment` cache — returns if already created
3. Gets table base offset, adds function to table
4. Resolves runtime `Function` class prototype
5. Creates a buffer and writes `_index` and `_env` fields
6. Creates a memory segment from the buffer
7. Returns `memorySegment.offset + program.totalOverhead` (i64 pointer)

- [ ] **Step 2: Port faithfully**

This is a complex function. Read the exact TS carefully. The result type should be `int64` (memory address), not `int32` (table index). Key dependencies:
- `program.functionPrototype` — the `Function` class prototype
- `resolver.resolveClass()` — to instantiate Function with type args
- `program.totalOverhead` — runtime object header size
- Memory segment creation infrastructure

**Note:** If memory segment infrastructure doesn't exist yet in the Go port, this may need to be deferred or stubbed with a TODO that documents exactly what's missing. Check `compiler/` for existing `MemorySegment` or `writeStaticBuffer` usage.

- [ ] **Step 3: Verify and commit**

```bash
go build ./... && git add compiler/compile_global.go && git commit -m "fix(H6): port ensureRuntimeFunction from compiler.ts:2124-2145

Returns memory address (i64) to runtime Function object, not raw
table index (int32). Creates proper memory segment with _index
and _env fields."
```

---

### Task 12: H7 — Fix `CompileEnum` map iteration order

**Impact:** Non-deterministic enum value auto-increment due to Go map random iteration.

**Files:**
- Modify: `compiler/compile_file.go:177` (and surrounding enum compilation)
- Reference: TS `compiler.ts:1436`

- [ ] **Step 1: Determine iteration order source**

In TS, `Map_values(members)` iterates in insertion order. Check how `members` is populated in the Go `Enum` type. If the Go enum stores members in a `map[string]DeclaredElement`, you need an ordered alternative.

Options:
1. If the enum has an ordered member list (e.g., `OrderedMembers []string` or similar), use it
2. If not, add one — when members are added during initialization, maintain insertion order

- [ ] **Step 2: Check existing ordered member infrastructure**

Search for `OrderedMembers`, `MemberNames`, or similar in `program/element.go` or `program/enum.go`. The Go port may already have ordered iteration support added for this exact reason.

- [ ] **Step 3: Replace map range with ordered iteration**

Change from:
```go
for _, member := range members {
```
to iteration over the ordered member list (exact syntax depends on what infrastructure exists).

- [ ] **Step 4: Verify and commit**

```bash
go build ./... && git add compiler/compile_file.go && git commit -m "fix(H7): use ordered iteration for enum member compilation

Go map iteration is non-deterministic. Enum auto-increment depends
on insertion-ordered iteration (TS Map preserves insertion order)."
```

---

## Chunk 4: HIGH Priority — Program Package Bugs (H8, H9, H10)

---

### Task 13: H8 — Fix `Element.Add` parent comparison

**Impact:** Incorrect duplicate detection — compares grandparent instead of container.

**Files:**
- Modify: `program/element.go:149`
- Reference: TS `program.ts:3039`

- [ ] **Step 1: Understand the TS logic**

TS line 3039: `if (existing.parent != this)` — compares existing element's parent to `this` (the Element on which `add` is being called — the container).

- [ ] **Step 2: Fix the Go comparison**

The challenge is that `Add` is on `*ElementBase` and needs to compare against the containing Element (not `e.parent`). This requires the Add method to receive or know its owning Element.

Check how `Add` is called. If the caller is always the embedding type (e.g., `myClass.Add(member)`), then `Add` needs access to the embedding struct. Options:
1. Add a parameter: `func (e *ElementBase) Add(name string, element DeclaredElement, container Element) bool`
2. Store a self-reference: `e.self` pointing to the embedding Element

Read the existing `Add` method and its call sites to determine the best approach that maintains existing patterns.

Change the comparison from:
```go
if existing.GetParent() == e.parent {  // wrong: grandparent
```
to the equivalent of:
```go
if existing.GetParent() == container {  // correct: this element
```

Also note the logic inversion is intentional — TS has override in `if` branch, Go has it in `else`. Just fix the operand.

- [ ] **Step 3: Update all call sites if signature changed**

- [ ] **Step 4: Verify and commit**

```bash
go build ./... && git add program/element.go && git commit -m "fix(H8): compare against container (this) not grandparent in Add

TS program.ts:3039 compares existing.parent to this (the container).
Go was comparing to e.parent (the container's parent = grandparent)."
```

---

### Task 14: H9 — Add missing diagnostics to `EnsureGlobal`

**Impact:** Silently overwrites on merge failure instead of reporting duplicate identifier error.

**Files:**
- Modify: `program/program.go:825-841`
- Reference: TS `program.ts:1812-1844`

- [ ] **Step 1: Read TS merge failure handling**

TS lines 1823-1837: When merge fails, reports `Duplicate_identifier_0` error with source locations for both elements. Does NOT overwrite the map — returns the new element but keeps the old one in the map.

- [ ] **Step 2: Fix the Go merge failure path**

Change the merge failure block (around line 834) from:
```go
// If merge fails, the newer element wins.
p.ElementsByNameMap[name] = element
return element
```
to:
```go
// Report duplicate error (TS lines 1823-1837)
existingDecl, isExistingDeclared := existing.(DeclaredElement)
if isExistingDeclared {
    p.ErrorRelated(
        diagnostics.DiagnosticCodeDuplicateIdentifier0,
        element.IdentifierNode().GetRange(),
        existingDecl.Declaration().Name().GetRange(),
        name,
    )
} else {
    p.Error(
        diagnostics.DiagnosticCodeDuplicateIdentifier0,
        element.IdentifierNode().GetRange(),
        name, "", "",
    )
}
// TS does NOT overwrite the map on merge failure
return element
```

**IMPORTANT:** Verify the exact method names (`ErrorRelated`, `IdentifierNode`, `Declaration`, etc.) exist in the Go codebase. Remove the `p.ElementsByNameMap[name] = element` line — TS keeps the existing element in the map.

- [ ] **Step 3: Verify and commit**

```bash
go build ./... && git add program/program.go && git commit -m "fix(H9): add Duplicate_identifier_0 diagnostic to EnsureGlobal

TS program.ts:1823-1837 reports an error on merge failure and does
NOT overwrite the map. Go was silently overwriting."
```

---

### Task 15: H10 — Remove caching from `AbortInstance`

**Impact:** TS intentionally does NOT cache abort instance. Caching could return stale values if abort is disabled.

**Files:**
- Modify: `program/program.go:1190-1196` (method)
- Modify: `program/program.go:57` (cached field)
- Reference: TS `program.ts:672-676`

- [ ] **Step 1: Read TS abort getter**

```typescript
get abortInstance(): Function | null {
    let prototype = this.lookup(CommonNames.abort);
    if (!prototype || prototype.kind != ElementKind.FunctionPrototype) return null;
    return this.resolver.resolveFunction(<FunctionPrototype>prototype, null);
}
```

No caching. Can return nil. Soft lookup.

- [ ] **Step 2: Rewrite Go method**

Replace `AbortInstance()` with:
```go
func (p *Program) AbortInstance() *Function {
    prototype := p.Lookup(common.CommonNameAbort)
    if prototype == nil {
        return nil
    }
    funcProto, ok := prototype.(*FunctionPrototype)
    if !ok {
        return nil
    }
    return p.Resolver.ResolveFunction(funcProto, nil, ReportModeSwallow)
}
```

Remove the `cachedAbortInstance` field from the struct.

- [ ] **Step 3: Verify and commit**

```bash
go build ./... && git add program/program.go && git commit -m "fix(H10): remove caching from AbortInstance

TS program.ts:672-676 intentionally does not cache abortInstance.
It performs a fresh lookup+resolve each time, returning nil if abort
is disabled."
```

---

## Chunk 5: HIGH Priority — Parser Bugs (H12, H13, H14)

---

### Task 16: H12 — Fix `declare` in ambient context flag handling

**Impact:** Sets `Declare|Ambient` flags even in error cases where TS does not.

**Files:**
- Modify: `parser/parse_statements.go:53-62`
- Reference: TS `parser.ts:235-249`

- [ ] **Step 1: Read TS structure**

```typescript
if (tn.skip(Token.Declare)) {
    if (contextIsAmbient) {
        this.error(...); // report error, DON'T set flags
    } else {
        // Only set flags in non-error case
        if (startPos < 0) startPos = tn.tokenPos;
        declareStart = startPos;
        declareEnd = tn.pos;
        flags |= CommonFlags.Declare | CommonFlags.Ambient;
    }
} else if (contextIsAmbient) {
    flags |= CommonFlags.Ambient;
}
```

- [ ] **Step 2: Restructure Go code**

Change `parse_statements.go:53-62` to match the TS `if/else` structure:
```go
if tn.Skip(tokenizer.TokenDeclare, tokenizer.IdentifierHandlingDefault) {
    if namespace != nil && namespace.Is(cf(common.CommonFlagsAmbient)) {
        // Error case — do NOT set flags or positions
        p.error(
            diagnostics.DiagnosticCodeADeclareModifierCannotBeUsedInAnAlreadyAmbientContext,
            tn.Range(tn.TokenPos, tn.Pos),
        )
    } else {
        // Non-error case — set flags and positions
        if startPos < 0 {
            startPos = tn.TokenPos
        }
        declareStart = tn.TokenPos
        declareEnd = tn.Pos
        flags |= cf(common.CommonFlagsDeclare | common.CommonFlagsAmbient)
    }
} else if namespace != nil && namespace.Is(cf(common.CommonFlagsAmbient)) {
    flags |= cf(common.CommonFlagsAmbient)
}
```

- [ ] **Step 3: Verify and commit**

```bash
go build ./... && git add parser/parse_statements.go && git commit -m "fix(H12): only set Declare|Ambient flags in non-error case

TS parser.ts:235-249 has an if/else structure where flags are only
set in the else branch (non-ambient context). Go was unconditionally
setting flags after the error report."
```

---

### Task 17: H13 — Fix `SkipIdentifier` handling across parser

**Impact:** Many call sites pass `IdentifierHandlingDefault` instead of `IdentifierHandlingPrefer`, rejecting contextual keywords as identifiers.

**Files:**
- Modify: `parser/parse_expressions.go` — lines 57, 239
- Modify: `parser/parse_statements.go` — lines 659, 762
- Modify: `parser/parse_declarations.go` — lines 13, 78, 187, 327, 394, 498, 597, 704, 1202, 1249, 1435, 1495, 1516, 1519
- Reference: TS `tokenizer.ts:1008` — default parameter is `IdentifierHandling.Prefer`

- [ ] **Step 1: Identify all affected sites**

Search for `SkipIdentifier(tokenizer.IdentifierHandlingDefault)` across the parser package. For each hit, check the corresponding TS call. If the TS uses bare `tn.skipIdentifier()` (no argument), the Go should use `IdentifierHandlingPrefer`.

- [ ] **Step 2: Change all sites where TS defaults to Prefer**

For each confirmed site, change:
```go
tn.SkipIdentifier(tokenizer.IdentifierHandlingDefault)
```
to:
```go
tn.SkipIdentifier(tokenizer.IdentifierHandlingPrefer)
```

**IMPORTANT:** Not ALL sites should be changed. Some TS call sites explicitly pass `IdentifierHandling.Default`. Only change sites where the TS uses the bare `tn.skipIdentifier()` call (no argument = Prefer default).

- [ ] **Step 3: Verify and commit**

```bash
go build ./... && git add parser/ && git commit -m "fix(H13): use IdentifierHandlingPrefer where TS defaults to Prefer

TS skipIdentifier() defaults to IdentifierHandling.Prefer. Many Go
call sites were passing IdentifierHandlingDefault instead."
```

---

### Task 18: H14 — Fix `parseType` default for `acceptParenthesized`

**Impact:** 3 call sites reject parenthesized types.

**Files:**
- Modify: `parser/parse_expressions.go` — lines 275, 487, 510
- Reference: TS `parser.ts:498-502` — default for `acceptParenthesized` is `true`

- [ ] **Step 1: Fix the 3 call sites**

At each line, change:
```go
p.parseType(tn, false, false)
```
to:
```go
p.parseType(tn, true, false)
```

The three sites are:
1. Line 275: Prefix assertion expression (`<Type>expr`)
2. Line 487: `as` assertion
3. Line 510: `instanceof` expression

- [ ] **Step 2: Verify and commit**

```bash
go build ./... && git add parser/parse_expressions.go && git commit -m "fix(H14): pass acceptParenthesized=true to parseType

TS parseType() defaults acceptParenthesized to true. Three expression
parsing sites were passing false."
```

---

## Chunk 6: HIGH Priority — Compiler Statement + Builtins Bugs (H15, H16, H17, H18)

---

### Task 19: H15 — Fix `makeIsTrueish` default case

**Impact:** Default case returns expression silently instead of reporting error.

**Files:**
- Modify: `compiler/compile_statement.go` — `makeIsTrueish` default case (~line 1066)
- Reference: TS `compiler.ts:10275-10282`

- [ ] **Step 1: Fix the default case**

Change from:
```go
default:
    return expr
```
to:
```go
case types.TypeKindVoid:
    fallthrough
default:
    c.Error(
        diagnostics.DiagnosticCodeAnExpressionOfType0CannotBeTestedForTruthiness,
        reportNode.GetRange(),
        typ.String(), "", "",
    )
    return mod.I32(0)
```

**IMPORTANT:** Verify the exact diagnostic code name exists. It should be something like `DiagnosticCodeAnExpressionOfType0CannotBeTestedForTruthiness` or similar.

- [ ] **Step 2: Verify and commit**

```bash
go build ./... && git add compiler/compile_statement.go && git commit -m "fix(H15): report error in makeIsTrueish default case

TS compiler.ts:10275-10282 reports an error and returns i32(0) for
Void and unsupported types. Go was silently returning the expression."
```

---

### Task 20: H16 — Add missing `compileTypeDeclaration`

**Impact:** Local `type X = Y` aliases produce `unreachable`.

**Files:**
- Modify: `compiler/compile_statement.go` — `CompileStatement` switch (~line 20)
- Reference: TS `compiler.ts:2306-2308` (switch case) and `compiler.ts:2368-2384` (implementation)

- [ ] **Step 1: Read TS implementation**

TS `compileTypeDeclaration` (lines 2368-2384):
- Checks for duplicate type names in current scope
- Creates a `TypeDefinition` from the declaration
- Adds it to the current flow's scoped type aliases
- Returns `module.nop()`

- [ ] **Step 2: Add the switch case**

In `CompileStatement` switch, add:
```go
case ast.NodeKindTypeDeclaration:
    return c.compileTypeDeclaration(statement.(*ast.TypeDeclaration))
```

- [ ] **Step 3: Implement `compileTypeDeclaration`**

Port from TS lines 2368-2384:
```go
func (c *Compiler) compileTypeDeclaration(declaration *ast.TypeDeclaration) module.ExpressionRef {
    // Port from compiler.ts:2368-2384
    // Check for duplicates, register type alias, return nop
    return c.Module().Nop()
}
```

Read the full TS implementation and port faithfully. Key operations involve checking existing type aliases in the flow and registering the new one.

Also add `NodeKindModule` case:
```go
case ast.NodeKindModule:
    return mod.Nop()
```

- [ ] **Step 4: Verify and commit**

```bash
go build ./... && git add compiler/compile_statement.go && git commit -m "fix(H16): add compileTypeDeclaration and Module cases

Local type aliases (type X = Y) were hitting default case and
producing unreachable. Port from compiler.ts:2368-2384."
```

---

### Task 21: H17 — Register 7 generic operator builtins

**Impact:** `add<T>`, `sub<T>`, `mul<T>`, `div<T>`, `rem<T>`, `eq<T>`, `ne<T>` handlers exist but are never registered.

**Files:**
- Modify: `compiler/builtins_simd.go` — `registerSIMDBuiltins()` function (or appropriate init)
- Reference: TS `builtins.ts:2601-2927`

- [ ] **Step 1: Add registrations**

In the appropriate `init()` function (likely `builtins_simd.go` or a new `builtins_operators.go`), add:
```go
builtinFunctions[common.BuiltinNameRem] = builtinRem
builtinFunctions[common.BuiltinNameAdd] = builtinAdd
builtinFunctions[common.BuiltinNameSub] = builtinSub
builtinFunctions[common.BuiltinNameMul] = builtinMul
builtinFunctions[common.BuiltinNameDiv] = builtinDiv
builtinFunctions[common.BuiltinNameEq] = builtinEq
builtinFunctions[common.BuiltinNameNe] = builtinNe
```

Verify the constant names exist in `common/builtinnames.go` (they do — lines 35-57).

- [ ] **Step 2: Verify and commit**

```bash
go build ./... && git add compiler/builtins_simd.go && git commit -m "fix(H17): register 7 generic operator builtins in dispatch map

Handler functions existed but were never registered. Adds add<T>,
sub<T>, mul<T>, div<T>, rem<T>, eq<T>, ne<T> to builtinFunctions."
```

---

### Task 22: H18 — Port 7 v128 constructor builtins

**Impact:** `v128()`, `i8x16()`, `i16x8()`, `i32x4()`, `i64x2()`, `f32x4()`, `f64x2()` are missing.

**Files:**
- Modify: `compiler/builtins_simd.go`
- Reference: TS `builtins.ts:4031-4385`

- [ ] **Step 1: Read TS implementations**

Each constructor creates a v128 value from lane arguments. Example `builtin_i8x16` (TS lines 4083-4132):
- Expects 16 operands (i8 values)
- For each operand: `runExpression(operands[i], i32)` for constant folding
- If constant: store in byte array
- If non-constant: use `simd_replace` to insert lane at runtime
- Returns the v128 const or a chain of simd_replace ops

- [ ] **Step 2: Port all 7 constructors**

Port each constructor faithfully from the TS. Register in `init()`:
```go
builtinFunctions[common.BuiltinNameV128] = builtinV128
builtinFunctions[common.BuiltinNameI8x16] = builtinI8x16
builtinFunctions[common.BuiltinNameI16x8] = builtinI16x8
builtinFunctions[common.BuiltinNameI32x4] = builtinI32x4
builtinFunctions[common.BuiltinNameI64x2] = builtinI64x2
builtinFunctions[common.BuiltinNameF32x4] = builtinF32x4
builtinFunctions[common.BuiltinNameF64x2] = builtinF64x2
```

Where `builtinV128` delegates to `builtinI8x16` (TS line 4031: `builtin_v128 = builtin_i8x16`).

- [ ] **Step 3: Verify and commit**

```bash
go build ./... && git add compiler/builtins_simd.go && git commit -m "fix(H18): port 7 v128 constructor builtins

Port v128(), i8x16(), i16x8(), i32x4(), i64x2(), f32x4(), f64x2()
from builtins.ts:4031-4385. Each creates a v128 constant from lane
values using runExpression for constant folding."
```

---

## Chunk 7: HIGH Priority — Compiler Function/Global Bugs (H4, H5)

---

### Task 23: H4 + H5 — Port `liftRequiresExportRuntime` and `lowerRequiresExportRuntime`

**Impact:** Both are stubs that over-report runtime requirements.

**Files:**
- Modify: `compiler/compile_function.go:451-475`
- Reference: TS `bindings/js.ts:1525-1580`

**Note:** These functions are in `bindings/js.ts` in the TS, not `compiler.ts`. Check if the Go port has a bindings package. If so, these may belong there instead.

- [ ] **Step 1: Read TS `liftRequiresExportRuntime`**

TS `bindings/js.ts:1525-1554`:
1. If no classReference but has signatureReference → `true`
2. If extends ArrayBuffer → `false`
3. If extends String → `false`
4. If extends ArrayBufferView → `false`
5. If extends Array or StaticArray → recurse on `clazz.getArrayValueType()`
6. Otherwise → `true`

- [ ] **Step 2: Port `liftRequiresExportRuntime`**

```go
func liftRequiresExportRuntime(typ *types.Type, prog *program.Program) bool {
    if !typ.IsInternalReference() {
        return false
    }
    clazz := typ.GetClass()
    if clazz == nil {
        // Function type (has signature) — needs runtime
        return true
    }
    if clazz.ExtendsPrototype(prog.ArrayBufferPrototype()) {
        return false
    }
    if clazz.ExtendsPrototype(prog.StringPrototype()) {
        return false
    }
    if clazz.ExtendsPrototype(prog.ArrayBufferViewPrototype()) {
        return false
    }
    if clazz.ExtendsPrototype(prog.ArrayPrototype()) ||
       clazz.ExtendsPrototype(prog.StaticArrayPrototype()) {
        elemType := clazz.GetArrayValueType()
        return liftRequiresExportRuntime(elemType, prog)
    }
    return true
}
```

**IMPORTANT:** Verify the exact method names (`ExtendsPrototype`, `ArrayBufferPrototype`, etc.) exist. These are program-level accessors that may have different names in the Go port.

- [ ] **Step 3: Read TS `lowerRequiresExportRuntime`**

TS `bindings/js.ts:1556-1580`:
1. Same initial checks as lift
2. If extends ArrayBuffer/String/ArrayBufferView/Array/StaticArray → `true`
3. Otherwise → `isPlainObject(clazz)`

- [ ] **Step 4: Port `lowerRequiresExportRuntime`**

```go
func lowerRequiresExportRuntime(typ *types.Type, prog *program.Program) bool {
    if !typ.IsInternalReference() {
        return false
    }
    clazz := typ.GetClass()
    if clazz == nil {
        return false // function types lower by reference
    }
    if clazz.ExtendsPrototype(prog.ArrayBufferPrototype()) ||
       clazz.ExtendsPrototype(prog.StringPrototype()) ||
       clazz.ExtendsPrototype(prog.ArrayBufferViewPrototype()) ||
       clazz.ExtendsPrototype(prog.ArrayPrototype()) ||
       clazz.ExtendsPrototype(prog.StaticArrayPrototype()) {
        return true
    }
    return isPlainObject(clazz)
}
```

- [ ] **Step 5: Verify `isPlainObject` exists or port it**

Search for `isPlainObject` in the codebase. If missing, port from `bindings/js.ts`.

- [ ] **Step 6: Verify and commit**

```bash
go build ./... && git add compiler/compile_function.go && git commit -m "fix(H4,H5): port lift/lower RequiresExportRuntime class checks

Both were stubs returning true for all references. TS bindings/js.ts
checks class hierarchies: ArrayBuffer/String return false for lift,
and lower checks isPlainObject for non-collection types."
```

---

## Chunk 8: MEDIUM Priority Fixes

---

### Task 24: M1 — Fix sign extension in `GetConstValueInteger`

**Files:**
- Modify: `module/helpers.go:120`

- [ ] **Step 1: Fix the cast**

Change from:
```go
return int64(GetConstValueI32(expr))
```
to:
```go
return int64(uint32(GetConstValueI32(expr)))
```

This zero-extends instead of sign-extending, matching the TS `i64_new(lo, 0)` which casts through `<u32>`.

- [ ] **Step 2: Verify and commit**

```bash
go build ./... && git add module/helpers.go && git commit -m "fix(M1): zero-extend i32 to i64 in GetConstValueInteger

TS uses <u32> cast before widening to i64, which zero-extends.
Go int64(int32) sign-extends. Use uint32 intermediate cast."
```

---

### Task 25: M3 — Add missing `ModuleImport` flag in `MarkModuleImport`

**Files:**
- Modify: `program/program.go:857-865`
- Reference: TS `program.ts:1749-1760`

- [ ] **Step 1: Add the flag set**

Add as the first line of `MarkModuleImport`:
```go
element.Set(common.CommonFlagsModuleImport)
```

Verify `CommonFlagsModuleImport` exists in `common/`. If the element interface doesn't have `Set()`, check what method is used (e.g., `SetFlag()`).

- [ ] **Step 2: Verify and commit**

```bash
go build ./... && git add program/program.go && git commit -m "fix(M3): set CommonFlagsModuleImport in MarkModuleImport

TS program.ts:1749 calls element.set(CommonFlags.ModuleImport)
as the first operation. Go was missing this."
```

---

### Task 26: M5 — Add missing runtime accessor methods + `computeBlockSize`

**Files:**
- Modify: `program/program.go`
- Reference: TS `program.ts:689-753` (runtime accessors), `program.ts:836` (computeBlockSize)

**Note:** Verification found 3 confirmed missing (not 7): `reallocInstance`, `freeInstance`, and `computeBlockSize`. Others like `renewInstance`, `collectInstance`, `newBufferInstance`, `newArrayInstance` should also be checked — add if missing.

- [ ] **Step 1: Add missing accessor methods**

Following the pattern of existing accessors (e.g., `AllocInstance()`), add at minimum:
```go
func (p *Program) ReallocInstance() *Function { ... }
func (p *Program) FreeInstance() *Function { ... }
func (p *Program) ComputeBlockSize(payloadSize int) int { ... }
```

Each function accessor follows the TS pattern: lookup by `CommonNames.xxx`, resolve as function prototype, cache result. `computeBlockSize` is a computation method. Read each TS definition for exact semantics.

Also check and add if missing: `RenewInstance`, `CollectInstance`, `NewBufferInstance`, `NewArrayInstance`.

- [ ] **Step 2: Verify and commit**

```bash
go build ./... && git add program/program.go && git commit -m "fix(M5): add missing runtime accessor methods

Port reallocInstance, freeInstance, computeBlockSize and others
from program.ts:689-753, 836."
```

---

### Task 27: M6 + M7 — Fix Ambient flag in `MakeNativeFunctionDeclaration` and start function

**Files:**
- Modify: `program/program.go:959` (M6)
- Modify: `compiler/compiler.go:189-194` (M7)
- Reference: TS `program.ts:907-910` and `compiler.ts:522`

- [ ] **Step 1: Fix M6 — Remove forced Ambient**

In `program/program.go:959`, change:
```go
// FROM:
int32(flags | common.CommonFlagsAmbient),
// TO:
int32(flags),
```

- [ ] **Step 2: Fix M7 — Use None for start function flags**

In `compiler/compiler.go` around line 189-194, change:
```go
// FROM:
common.CommonFlagsAmbient,
// TO:
common.CommonFlagsNone,
```

Or just `0` if `CommonFlagsNone` doesn't exist.

- [ ] **Step 3: Verify and commit**

```bash
go build ./... && git add program/program.go compiler/compiler.go && git commit -m "fix(M6,M7): remove spurious Ambient flag from native functions

M6: MakeNativeFunctionDeclaration was forcing Ambient on all functions.
TS program.ts:907-910 passes flags as-is.
M7: Start function was created with Ambient flag. TS compiler.ts:522
uses the default (CommonFlags.None)."
```

---

### Task 28: M9 — Fix parser `startPos` computation

**Files:**
- Modify: `parser/parse_statements.go:72`
- Reference: TS `parser.ts:256`

- [ ] **Step 1: Fix the position source**

Change from:
```go
startPos = tn.TokenPos + 1
```
to:
```go
startPos = tn.NextTokenPos()
```

Verify that `NextTokenPos()` exists on the tokenizer (it does — confirmed at `tokenizer/tokenizer.go:1295`).

- [ ] **Step 2: Verify and commit**

```bash
go build ./... && git add parser/parse_statements.go && git commit -m "fix(M9): use NextTokenPos() instead of TokenPos+1 for startPos

TS parser.ts:256 uses tn.nextTokenPos (start position of peeked token).
Go was using tn.TokenPos+1 (one past current token start), which is
a different value."
```

---

### Task 29: M4 — Simplify `resolveLiteralExpression` to delegate

**Files:**
- Modify: `program/resolver.go:3008-3093`
- Reference: TS `resolver.ts:2377-2399`

- [ ] **Step 1: Read TS implementation**

TS `resolveLiteralExpression` is a simple delegator:
```typescript
let element = this.lookupLiteralExpression(node, ctxFlow, ctxType, reportMode);
if (!element) return null;
let type = this.getTypeOfElement(element);
if (!type && reportMode == ReportMode.Report) {
    this.error(DiagnosticCode.Expression_cannot_be_represented_by_a_type, node.range);
}
return type;
```

- [ ] **Step 2: Replace with faithful port**

Replace the entire inline dispatch (lines 3008-3093) with:
```go
func (r *Resolver) resolveLiteralExpression(
    node ast.Node,
    ctxFlow *flow.Flow,
    ctxType *types.Type,
    reportMode ReportMode,
) *types.Type {
    element := r.lookupLiteralExpression(node, ctxFlow, ctxType, reportMode)
    if element == nil {
        return nil
    }
    typ := r.GetTypeOfElement(element)
    if typ == nil && reportMode == ReportModeReport {
        r.program.Error(
            diagnostics.DiagnosticCodeExpressionCannotBeRepresentedByAType,
            node.GetRange(),
            "", "", "",
        )
    }
    return typ
}
```

- [ ] **Step 3: Verify and commit**

```bash
go build ./... && git add program/resolver.go && git commit -m "fix(M4): simplify resolveLiteralExpression to delegate

TS resolver.ts:2377-2399 is a 3-line delegator to
lookupLiteralExpression + getTypeOfElement. Go had a full inline
type dispatch duplicating lookupLiteralExpression logic."
```

---

### Task 30: M2 — Add missing table size update in `AddFunctionTable`

**Files:**
- Modify: `module/module.go:1117-1123`
- Reference: TS `module.ts:2396-2418`

- [ ] **Step 1: Add the else branch**

The TS checks if the table exists and updates its sizes:
```typescript
if (!tableRef) {
    tableRef = binaryen._BinaryenAddTable(this.ref, cStr, initial, maximum, TypeRef.Funcref);
} else {
    binaryen._BinaryenTableSetInitial(tableRef, initial);
    binaryen._BinaryenTableSetMax(tableRef, maximum);
}
```

Add the `else` branch to the Go:
```go
tableRef := m.bmod.GetTable(name)
if tableRef == 0 {
    m.bmod.AddTable(name, initial, maximum, binaryen.TypeFuncref())
} else {
    m.bmod.TableSetInitial(tableRef, initial)
    m.bmod.TableSetMax(tableRef, maximum)
}
```

**IMPORTANT:** Verify that `TableSetInitial` and `TableSetMax` exist in the binaryen Go bindings. If not, they may need to be added to the CGo wrapper.

- [ ] **Step 2: Verify and commit**

```bash
go build ./... && git add module/module.go && git commit -m "fix(M2): update table sizes when table already exists

TS module.ts:2410-2416 updates initial/maximum when table exists.
Go was skipping the update entirely."
```

---

### Task 31: M8 — Remove invented `StringSegmentOffsets` cache

**Files:**
- Modify: `compiler/compiler.go:37` (field declaration)
- Modify: `compiler/compiler.go:111` (initialization)
- Reference: TS `compiler.ts:440-458` — no such map exists

- [ ] **Step 1: Verify the cache is unused or find usages**

Search for `StringSegmentOffsets` across the compiler package. If it's used, understand what it does and replace with the TS approach (computing offsets inline via `stringSegments` map). If unused, remove it.

- [ ] **Step 2: Remove or replace**

If the map is not used meaningfully:
- Remove the field from the Compiler struct
- Remove the initialization

If it IS used, refactor to match TS — the TS uses `stringSegments: Map<string, MemorySegment>` and accesses `.offset` on the segment directly.

- [ ] **Step 3: Verify and commit**

```bash
go build ./... && git add compiler/compiler.go && git commit -m "fix(M8): remove invented StringSegmentOffsets cache

This map does not exist in TS compiler.ts. The TS computes string
segment offsets directly from the MemorySegment objects."
```

---

## Verification

After all tasks are complete:

- [ ] **Full build check**: `cd /Users/davidroman/Documents/code/brainlet/brainkit/wasm-kit && go build ./...`
- [ ] **Run existing tests**: `go test ./...`
- [ ] **Update PROGRESS.md** with corrected completeness percentages
- [ ] **Update audit SUMMARY.md** marking each issue as FIXED

---

## Issue Reference Table

| ID | Task | Priority | File | Status |
|----|------|----------|------|--------|
| S1 | 1 | SHOWSTOPPER | compile.go:1954 | |
| C1 | 2 | CRITICAL | compile_file.go:45, compiler.go:29 | |
| C2 | 3 | CRITICAL | compile_expression.go:24 | |
| C3 | 4 | CRITICAL | compile_statement.go:231,310,427,924 | |
| C4 | 5 | CRITICAL | resolver.go:3448 | |
| C5 | 6 | CRITICAL | resolver.go:1309 | |
| C6 | 7 | CRITICAL | compile_expression.go:3895,3920 | |
| C7 | 8 | CRITICAL | compile_expression.go:3722, compiler.go | |
| H1 | 9 | HIGH | compile.go:1261 | |
| H2 | 10 | HIGH | compile.go:2159 | |
| H3 | — | FALSE POSITIVE | — | N/A |
| H4 | 23 | HIGH | compile_function.go:451 | |
| H5 | 23 | HIGH | compile_function.go:463 | |
| H6 | 11 | HIGH | compile_global.go:481 | |
| H7 | 12 | HIGH | compile_file.go:177 | |
| H8 | 13 | HIGH | element.go:149 | |
| H9 | 14 | HIGH | program.go:825 | |
| H10 | 15 | HIGH | program.go:1190 | |
| H11 | — | FALSE POSITIVE | — | N/A |
| H12 | 16 | HIGH | parse_statements.go:53 | |
| H13 | 17 | HIGH | parse_expressions.go + parse_statements.go + parse_declarations.go | |
| H14 | 18 | HIGH | parse_expressions.go:275,487,510 | |
| H15 | 19 | HIGH | compile_statement.go:1066 | |
| H16 | 20 | HIGH | compile_statement.go:20 | |
| H17 | 21 | HIGH | builtins_simd.go | |
| H18 | 22 | HIGH | builtins_simd.go | |
| M1 | 24 | MEDIUM | helpers.go:120 | |
| M3 | 25 | MEDIUM | program.go:857 | |
| M4 | 29 | MEDIUM | resolver.go:3008 | |
| M5 | 26 | MEDIUM | program.go | |
| M6 | 27 | MEDIUM | program.go:959 | |
| M7 | 27 | MEDIUM | compiler.go:189 | |
| M8 | 31 | MEDIUM | compiler.go:37 | |
| M9 | 28 | MEDIUM | parse_statements.go:72 | |
| M2 | 30 | MEDIUM | module.go:1117 | |
| M10 | — | FALSE POSITIVE | — | N/A |
