// Ported from: assemblyscript/src/bindings/js.ts
package bindings

import (
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// RuntimeFunctions are functions to export if --exportRuntime is set.
// Ported from: assemblyscript/src/compiler.ts runtimeFunctions
var RuntimeFunctions = []string{"__new", "__pin", "__unpin", "__collect"}

// RuntimeGlobals are globals to export if --exportRuntime is set.
// Ported from: assemblyscript/src/compiler.ts runtimeGlobals
var RuntimeGlobals = []string{"__rtti_base"}

// jsMode distinguishes import vs export context for code generation.
type jsMode int

const (
	jsModeImport jsMode = iota
	jsModeExport
)

// importToModule maps special imports to their actual modules.
func importToModule(moduleName string) string {
	if moduleName == "rtrace" {
		return "#rtrace"
	}
	return moduleName
}

// shouldInstrument determines whether a module's imports should be instrumented.
func shouldInstrument(moduleName string) bool {
	return moduleName != "rtrace"
}

// JSBuilder generates JavaScript bindings for WebAssembly modules.
// Ported from: assemblyscript/src/bindings/js.ts JSBuilder
type JSBuilder struct {
	ExportsWalker
	esm         bool
	sb          []string
	indentLevel int

	needsLiftBuffer      bool
	needsLowerBuffer     bool
	needsLiftString      bool
	needsLowerString     bool
	needsLiftArray       bool
	needsLowerArray      bool
	needsLiftTypedArray  bool
	needsLowerTypedArray bool
	needsLiftStaticArray bool
	needsLowerStaticArray bool
	needsLiftInternref   bool
	needsLowerInternref  bool
	needsRetain          bool
	needsRelease         bool
	needsNotNull         bool
	needsSetU8           bool
	needsSetU16          bool
	needsSetU32          bool
	needsSetU64          bool
	needsSetF32          bool
	needsSetF64          bool
	needsGetI8           bool
	needsGetU8           bool
	needsGetI16          bool
	needsGetU16          bool
	needsGetI32          bool
	needsGetU32          bool
	needsGetI64          bool
	needsGetU64          bool
	needsGetF32          bool
	needsGetF64          bool

	deferredLifts  map[program.Element]struct{}
	deferredLowers map[program.Element]struct{}
	deferredCode   []string

	exports        []string
	importMappings map[string]int32
}

// BuildJS builds JavaScript bindings for the specified program.
func BuildJS(prog *program.Program, esm bool) string {
	b := NewJSBuilder(prog, esm, false)
	return b.Build()
}

// NewJSBuilder constructs a new JavaScript bindings builder.
func NewJSBuilder(prog *program.Program, esm bool, includePrivate bool) *JSBuilder {
	b := &JSBuilder{
		ExportsWalker:  NewExportsWalker(prog, includePrivate),
		esm:            esm,
		sb:             make([]string, 0),
		deferredLifts:  make(map[program.Element]struct{}),
		deferredLowers: make(map[program.Element]struct{}),
		deferredCode:   make([]string, 0),
		exports:        make([]string, 0),
		importMappings: make(map[string]int32),
	}
	b.OnVisitGlobal = b.visitGlobal
	b.OnVisitEnum = b.visitEnum
	b.OnVisitFunction = b.visitFunction
	b.OnVisitClass = b.visitClass
	b.OnVisitInterface = b.visitInterface
	b.OnVisitNamespace = b.visitNamespace
	b.OnVisitAlias = b.visitAlias
	return b
}

// indentSB writes indentation to the string slice.
func indentSB(sb *[]string, level int) {
	*sb = append(*sb, strings.Repeat("  ", level))
}

func (b *JSBuilder) visitGlobal(name string, element *program.Global) {
	sb := &b.sb
	typ := element.GetResolvedType()
	b.exports = append(b.exports, name)
	if !isPlainValue(typ, jsModeExport) {
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, name)
		*sb = append(*sb, ": {\n")
		b.indentLevel++
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "// ")
		*sb = append(*sb, element.GetInternalName())
		*sb = append(*sb, ": ")
		*sb = append(*sb, typ.String())
		*sb = append(*sb, "\n")
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "valueOf() { return this.value; },\n")
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "get value() {\n")
		b.indentLevel++
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "return ")
		b.makeLiftFromValue("exports."+name+".value", typ, sb)
		*sb = append(*sb, ";\n")
		b.indentLevel--
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "}")
		if !element.Is(common.CommonFlagsConst) {
			*sb = append(*sb, ",\n")
			indentSB(sb, b.indentLevel)
			*sb = append(*sb, "set value(value) {\n")
			b.indentLevel++
			indentSB(sb, b.indentLevel)
			*sb = append(*sb, "exports.")
			*sb = append(*sb, name)
			*sb = append(*sb, ".value = ")
			b.makeLowerToValue("value", typ, sb)
			*sb = append(*sb, ";\n")
			b.indentLevel--
			indentSB(sb, b.indentLevel)
			*sb = append(*sb, "}")
		}
		*sb = append(*sb, "\n")
		b.indentLevel--
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "},\n")
	}
	b.visitNamespace(name, element)
}

func (b *JSBuilder) visitEnum(name string, element *program.Enum) {
	sb := &b.sb
	b.exports = append(b.exports, name)
	indentSB(sb, b.indentLevel)
	*sb = append(*sb, name)
	*sb = append(*sb, ": (values => (\n")
	b.indentLevel++
	indentSB(sb, b.indentLevel)
	*sb = append(*sb, "// ")
	*sb = append(*sb, element.GetInternalName())
	*sb = append(*sb, "\n")
	members := element.GetMembers()
	if members != nil {
		for _, value := range members {
			if value.GetElementKind() != program.ElementKindEnumValue {
				continue
			}
			indentSB(sb, b.indentLevel)
			*sb = append(*sb, "values[values.")
			*sb = append(*sb, value.GetName())
			ev := value.(*program.EnumValue)
			if ev.Is(common.CommonFlagsInlined) {
				*sb = append(*sb, " = ")
				*sb = append(*sb, fmt.Sprintf("%d", int32(ev.GetConstantIntegerValue())))
			} else {
				*sb = append(*sb, " = exports[\"")
				*sb = append(*sb, util.EscapeString(name+"."+value.GetName(), util.CharCodeDoubleQuote))
				*sb = append(*sb, "\"].valueOf()")
			}
			*sb = append(*sb, "] = \"")
			*sb = append(*sb, util.EscapeString(value.GetName(), util.CharCodeDoubleQuote))
			*sb = append(*sb, "\",\n")
		}
	}
	indentSB(sb, b.indentLevel)
	*sb = append(*sb, "values\n")
	b.indentLevel--
	indentSB(sb, b.indentLevel)
	*sb = append(*sb, "))({}),\n")
	b.visitNamespace(name, element)
}

func (b *JSBuilder) makeGlobalImport(moduleName string, name string, element *program.Global) {
	sb := &b.sb
	typ := element.GetResolvedType()
	indentSB(sb, b.indentLevel)
	if util.IsIdentifier(name) {
		*sb = append(*sb, name)
	} else {
		*sb = append(*sb, "\"")
		*sb = append(*sb, util.EscapeString(name, util.CharCodeDoubleQuote))
		*sb = append(*sb, "\": ")
	}
	moduleId := b.ensureModuleId(moduleName)
	if isPlainValue(typ, jsModeImport) {
		*sb = append(*sb, "(\n")
		indentSB(sb, b.indentLevel+1)
		*sb = append(*sb, "// ")
		*sb = append(*sb, element.GetInternalName())
		*sb = append(*sb, ": ")
		*sb = append(*sb, element.GetResolvedType().String())
		*sb = append(*sb, "\n")
		indentSB(sb, b.indentLevel+1)
		if moduleName != "env" {
			*sb = append(*sb, "__module")
			*sb = append(*sb, fmt.Sprintf("%d", moduleId))
			*sb = append(*sb, ".")
		}
		*sb = append(*sb, name)
		*sb = append(*sb, "\n")
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, ")")
	} else {
		*sb = append(*sb, "{\n")
		b.indentLevel++
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "// ")
		*sb = append(*sb, element.GetInternalName())
		*sb = append(*sb, ": ")
		*sb = append(*sb, element.GetResolvedType().String())
		*sb = append(*sb, "\n")
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "// not supported: cannot lower before instantiate completes\n")
		b.indentLevel--
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "}")
	}
	*sb = append(*sb, ",\n")
}

func (b *JSBuilder) makeFunctionImport(moduleName string, name string, element *program.Function, code string) {
	sb := &b.sb
	signature := element.Signature
	indentSB(sb, b.indentLevel)
	if util.IsIdentifier(name) {
		*sb = append(*sb, name)
	} else {
		*sb = append(*sb, "\"")
		*sb = append(*sb, util.EscapeString(name, util.CharCodeDoubleQuote))
		*sb = append(*sb, "\"")
	}
	if isPlainFunction(signature, jsModeImport) && code == "" && util.IsIdentifier(name) {
		*sb = append(*sb, ": (\n")
		indentSB(sb, b.indentLevel+1)
		*sb = append(*sb, "// ")
		*sb = append(*sb, element.GetInternalName())
		*sb = append(*sb, signature.String())
		*sb = append(*sb, "\n")
		indentSB(sb, b.indentLevel+1)
		if moduleName != "env" {
			*sb = append(*sb, moduleName)
			*sb = append(*sb, ".")
		}
		*sb = append(*sb, name)
		*sb = append(*sb, "\n")
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, ")")
	} else {
		*sb = append(*sb, "(")
		parameterTypes := signature.ParameterTypes
		parameterNames := make([]string, 0, len(parameterTypes))
		for i := range parameterTypes {
			parameterNames = append(parameterNames, element.GetParameterName(int32(i)))
		}
		*sb = append(*sb, strings.Join(parameterNames, ", "))
		*sb = append(*sb, ") {\n")
		b.indentLevel++
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "// ")
		*sb = append(*sb, element.GetInternalName())
		*sb = append(*sb, signature.String())
		*sb = append(*sb, "\n")
		for i, ptype := range parameterTypes {
			if !isPlainValue(ptype, jsModeExport) {
				pname := element.GetParameterName(int32(i))
				indentSB(sb, b.indentLevel)
				*sb = append(*sb, pname)
				*sb = append(*sb, " = ")
				b.makeLiftFromValue(pname, ptype, sb)
				*sb = append(*sb, ";\n")
			}
		}
		expr := make([]string, 0)
		moduleId := b.ensureModuleId(moduleName)
		if code != "" {
			expr = append(expr, "(() => {\n")
			indentSBSlice(&expr, 1)
			expr = append(expr, "// @external.js\n")
			indentText(code, 1, &expr, false)
			expr = append(expr, "\n})()")
		} else {
			if moduleName != "env" {
				expr = append(expr, "__module")
				expr = append(expr, fmt.Sprintf("%d", moduleId))
				expr = append(expr, ".")
			}
			expr = append(expr, name)
			expr = append(expr, "(")
			expr = append(expr, strings.Join(parameterNames, ", "))
			expr = append(expr, ")")
		}
		codeStr := strings.Join(expr, "")
		expr = expr[:0]
		indentText(codeStr, b.indentLevel, &expr, true)
		codeStr = strings.Join(expr, "")
		indentSB(sb, b.indentLevel)
		if signature.ReturnType != types.TypeVoid {
			*sb = append(*sb, "return ")
			b.makeLowerToValue(codeStr, signature.ReturnType, sb)
			*sb = append(*sb, ";\n")
		} else {
			*sb = append(*sb, codeStr)
			*sb = append(*sb, ";\n")
		}
		b.indentLevel--
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "}")
	}
	*sb = append(*sb, ",\n")
}

func (b *JSBuilder) visitFunction(name string, element *program.Function) {
	if element.Is(common.CommonFlagsPrivate) {
		return
	}
	sb := &b.sb
	signature := element.Signature
	b.exports = append(b.exports, name)
	if !isPlainFunction(signature, jsModeExport) {
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, name)
		*sb = append(*sb, "(")
		parameterTypes := signature.ParameterTypes
		numReferences := 0
		for i, ptype := range parameterTypes {
			if ptype.IsInternalReference() {
				numReferences++
			}
			if i > 0 {
				*sb = append(*sb, ", ")
			}
			*sb = append(*sb, element.GetParameterName(int32(i)))
		}
		*sb = append(*sb, ") {\n")
		b.indentLevel++
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "// ")
		*sb = append(*sb, element.GetInternalName())
		*sb = append(*sb, signature.String())
		*sb = append(*sb, "\n")
		releases := make([]string, 0)
		for i, ptype := range parameterTypes {
			if !isPlainValue(ptype, jsModeImport) {
				pname := element.GetParameterName(int32(i))
				indentSB(sb, b.indentLevel)
				*sb = append(*sb, pname)
				*sb = append(*sb, " = ")
				needsRetainRelease := ptype.IsInternalReference() && numReferences > 1
				if needsRetainRelease {
					numReferences--
					b.needsRetain = true
					b.needsRelease = true
					*sb = append(*sb, "__retain(")
					releases = append(releases, pname)
				}
				b.makeLowerToValue(pname, ptype, sb)
				if needsRetainRelease {
					*sb = append(*sb, ")")
				}
				*sb = append(*sb, ";\n")
			}
		}
		if len(releases) > 0 {
			indentSB(sb, b.indentLevel)
			b.indentLevel++
			*sb = append(*sb, "try {\n")
		}
		if signature.RequiredParameters < int32(len(parameterTypes)) {
			indentSB(sb, b.indentLevel)
			*sb = append(*sb, "exports.__setArgumentsLength(arguments.length);\n")
		}
		expr := make([]string, 0)
		expr = append(expr, "exports.")
		expr = append(expr, name)
		expr = append(expr, "(")
		for i := range parameterTypes {
			if i > 0 {
				expr = append(expr, ", ")
			}
			expr = append(expr, element.GetParameterName(int32(i)))
		}
		expr = append(expr, ")")
		if signature.ReturnType != types.TypeVoid {
			indentSB(sb, b.indentLevel)
			*sb = append(*sb, "return ")
			b.makeLiftFromValue(strings.Join(expr, ""), signature.ReturnType, sb)
		} else {
			indentSB(sb, b.indentLevel)
			*sb = append(*sb, strings.Join(expr, ""))
		}
		*sb = append(*sb, ";\n")
		if len(releases) > 0 {
			indentSB(sb, b.indentLevel-1)
			*sb = append(*sb, "} finally {\n")
			for _, relName := range releases {
				indentSB(sb, b.indentLevel)
				*sb = append(*sb, "__release(")
				*sb = append(*sb, relName)
				*sb = append(*sb, ");\n")
			}
			b.indentLevel--
			indentSB(sb, b.indentLevel)
			*sb = append(*sb, "}\n")
		}
		b.indentLevel--
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "},\n")
	}
	b.visitNamespace(name, element)
}

func (b *JSBuilder) visitClass(name string, element *program.Class) {
	// not implemented
}

func (b *JSBuilder) visitInterface(name string, element *program.Interface) {
	b.visitClass(name, &element.Class)
}

func (b *JSBuilder) visitNamespace(name string, element program.Element) {
	// not implemented
}

func (b *JSBuilder) visitAlias(name string, element program.Element, originalName string) {
	// not implemented
}

func (b *JSBuilder) getExternalCode(element *program.Function) string {
	decorator := ast.FindDecorator(ast.DecoratorKindExternalJs, element.DecoratorNodes())
	if decorator != nil {
		args := decorator.Args
		if args != nil && len(args) == 1 {
			codeArg := args[0]
			if codeArg.GetKind() == ast.NodeKindLiteral {
				if strLit, ok := codeArg.(*ast.StringLiteralExpression); ok {
					return strLit.Value
				}
				if tmplLit, ok := codeArg.(*ast.TemplateLiteralExpression); ok {
					if len(tmplLit.Parts) == 1 {
						return tmplLit.Parts[0]
					}
				}
			}
		}
	}
	return ""
}

// Build builds the JavaScript bindings string.
func (b *JSBuilder) Build() string {
	exports := b.exports
	_ = exports // used later
	moduleImports := b.Program.ModuleImports
	prog := b.Program
	options := prog.Options
	sb := &b.sb

	*sb = append(*sb, "") // placeholder [0]
	indentSB(sb, b.indentLevel)
	b.indentLevel++
	if !b.esm {
		*sb = append(*sb, "export ")
	}
	*sb = append(*sb, "async function instantiate(module, imports = {}) {\n")
	insertPos := len(*sb)
	*sb = append(*sb, "") // placeholder for module mappings

	// Instrument module imports
	indentSB(sb, b.indentLevel)
	b.indentLevel++
	*sb = append(*sb, "const adaptedImports = {\n")
	sbLengthBefore := len(*sb)
	for moduleName, moduleElements := range moduleImports {
		moduleId := b.ensureModuleId(moduleName)
		indentSB(sb, b.indentLevel)
		if util.IsIdentifier(moduleName) {
			*sb = append(*sb, moduleName)
		} else {
			*sb = append(*sb, "\"")
			*sb = append(*sb, util.EscapeString(moduleName, util.CharCodeDoubleQuote))
			*sb = append(*sb, "\"")
		}
		if !shouldInstrument(moduleName) {
			*sb = append(*sb, ": __module")
			*sb = append(*sb, fmt.Sprintf("%d", moduleId))
			*sb = append(*sb, ",\n")
			continue
		}
		resetPos := len(*sb)

		*sb = append(*sb, ": Object.setPrototypeOf({\n")
		b.indentLevel++
		numInstrumented := 0
		for elemName, elem := range moduleElements {
			if elem.GetElementKind() == program.ElementKindFunction {
				fn := elem.(*program.Function)
				code := b.getExternalCode(fn)
				if !isPlainFunction(fn.Signature, jsModeImport) || !util.IsIdentifier(elemName) || code != "" {
					b.makeFunctionImport(moduleName, elemName, fn, code)
					numInstrumented++
				}
			} else if elem.GetElementKind() == program.ElementKindGlobal {
				g := elem.(*program.Global)
				if !isPlainValue(g.GetResolvedType(), jsModeImport) || !util.IsIdentifier(elemName) {
					b.makeGlobalImport(moduleName, elemName, g)
					numInstrumented++
				}
			}
		}
		b.indentLevel--
		if numInstrumented == 0 {
			*sb = (*sb)[:resetPos]
			if moduleName == "env" {
				*sb = append(*sb, ": Object.assign(Object.create(globalThis), imports.env || {})")
			} else {
				*sb = append(*sb, ": __module")
				*sb = append(*sb, fmt.Sprintf("%d", moduleId))
			}
			*sb = append(*sb, ",\n")
		} else {
			indentSB(sb, b.indentLevel)
			*sb = append(*sb, "}, ")
			if moduleName == "env" {
				*sb = append(*sb, "Object.assign(Object.create(globalThis), imports.env || {})")
			} else {
				*sb = append(*sb, "__module")
				*sb = append(*sb, fmt.Sprintf("%d", moduleId))
			}
			*sb = append(*sb, "),\n")
		}
	}
	b.indentLevel--
	hasAdaptedImports := len(*sb) > sbLengthBefore
	if hasAdaptedImports {
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "};\n")
	} else {
		*sb = (*sb)[:sbLengthBefore-2] // incl. indent
	}

	// Build module mappings
	mappings := b.importMappings
	var mapSb strings.Builder
	for moduleName, moduleId := range mappings {
		if moduleName == "env" {
			mapSb.WriteString("  const env = imports.env;\n")
		} else {
			if moduleName == "rtrace" {
				mapSb.WriteString("  ((rtrace) => {\n")
				mapSb.WriteString("    delete imports.rtrace;\n")
				mapSb.WriteString("    new rtrace.Rtrace({ getMemory() { return memory; }, onerror(err) { console.log(`RTRACE: ${err.stack}`); } }).install(imports);\n")
				mapSb.WriteString("  })(imports.rtrace);\n")
			}
			mapSb.WriteString("  const __module")
			mapSb.WriteString(fmt.Sprintf("%d", moduleId))
			mapSb.WriteString(" = imports")
			if util.IsIdentifier(moduleName) {
				mapSb.WriteString(".")
				mapSb.WriteString(moduleName)
			} else {
				mapSb.WriteString("[\"")
				mapSb.WriteString(util.EscapeString(moduleName, util.CharCodeDoubleQuote))
				mapSb.WriteString("\"]")
			}
			mapSb.WriteString(";\n")
		}
	}
	(*sb)[insertPos] = mapSb.String()

	indentSB(sb, b.indentLevel)
	*sb = append(*sb, "const { exports } = await WebAssembly.instantiate(module")
	if hasAdaptedImports {
		*sb = append(*sb, ", adaptedImports);\n")
	} else {
		*sb = append(*sb, ", imports);\n")
	}
	indentSB(sb, b.indentLevel)
	*sb = append(*sb, "const memory = exports.memory || imports.env.memory;\n")
	indentSB(sb, b.indentLevel)
	b.indentLevel++
	*sb = append(*sb, "const adaptedExports = Object.setPrototypeOf({\n")
	sbLengthBefore = len(*sb)

	// Instrument module exports
	b.Walk()
	b.indentLevel--
	hasAdaptedExports := len(*sb) > sbLengthBefore
	if hasAdaptedExports {
		indentSB(sb, b.indentLevel)
		*sb = append(*sb, "}, exports);\n")
	} else {
		if b.needsLiftBuffer || b.needsLowerBuffer ||
			b.needsLiftString || b.needsLowerString ||
			b.needsLiftArray || b.needsLowerArray ||
			b.needsLiftTypedArray || b.needsLowerTypedArray ||
			b.needsLiftStaticArray {
			*sb = (*sb)[:sbLengthBefore-2] // skip adaptedExports + 1x indent
		} else {
			*sb = (*sb)[:sbLengthBefore-4] // skip memory and adaptedExports + 2x indent
		}
	}

	// Add deferred code fragments
	for _, code := range b.deferredCode {
		*sb = append(*sb, code)
	}

	// Add lifting and lowering adapters
	if b.needsLiftBuffer {
		objectInstance := prog.ObjectInstance()
		rtSizeOffset := objectInstance.Offsetof("rtSize") - objectInstance.NextMemoryOffset
		*sb = append(*sb, fmt.Sprintf(`  function __liftBuffer(pointer) {
    if (!pointer) return null;
    return memory.buffer.slice(pointer, pointer + new Uint32Array(memory.buffer)[pointer - %d >>> 2]);
  }
`, -int32(rtSizeOffset)))
	}
	if b.needsLowerBuffer {
		arrayBufferId := prog.ArrayBufferInstance().Id()
		*sb = append(*sb, fmt.Sprintf(`  function __lowerBuffer(value) {
    if (value == null) return 0;
    const pointer = exports.__new(value.byteLength, %d) >>> 0;
    new Uint8Array(memory.buffer).set(new Uint8Array(value), pointer);
    return pointer;
  }
`, arrayBufferId))
	}
	if b.needsLiftString {
		objectInstance := prog.ObjectInstance()
		rtSizeOffset := objectInstance.Offsetof("rtSize") - objectInstance.NextMemoryOffset
		chunkSize := 1024
		*sb = append(*sb, fmt.Sprintf(`  function __liftString(pointer) {
    if (!pointer) return null;
    const
      end = pointer + new Uint32Array(memory.buffer)[pointer - %d >>> 2] >>> 1,
      memoryU16 = new Uint16Array(memory.buffer);
    let
      start = pointer >>> 1,
      string = "";
    while (end - start > %d) string += String.fromCharCode(...memoryU16.subarray(start, start += %d));
    return string + String.fromCharCode(...memoryU16.subarray(start, end));
  }
`, -int32(rtSizeOffset), chunkSize, chunkSize))
	}
	if b.needsLowerString {
		stringId := prog.StringInstance().Id()
		*sb = append(*sb, fmt.Sprintf(`  function __lowerString(value) {
    if (value == null) return 0;
    const
      length = value.length,
      pointer = exports.__new(length << 1, %d) >>> 0,
      memoryU16 = new Uint16Array(memory.buffer);
    for (let i = 0; i < length; ++i) memoryU16[(pointer >>> 1) + i] = value.charCodeAt(i);
    return pointer;
  }
`, stringId))
	}
	if b.needsLiftArray {
		abvInstance := prog.ArrayBufferViewInstance()
		dataStartOffset := abvInstance.Offsetof("dataStart")
		lengthOffset := abvInstance.NextMemoryOffset
		b.needsGetU32 = true
		*sb = append(*sb, fmt.Sprintf(`  function __liftArray(liftElement, align, pointer) {
    if (!pointer) return null;
    const
      dataStart = __getU32(pointer + %d),
      length = __dataview.getUint32(pointer + %d, true),
      values = new Array(length);
    for (let i = 0; i < length; ++i) values[i] = liftElement(dataStart + (i << align >>> 0));
    return values;
  }
`, dataStartOffset, lengthOffset))
	}
	if b.needsLowerArray {
		arrayBufferId := prog.ArrayBufferInstance().Id()
		abvInstance := prog.ArrayBufferViewInstance()
		arraySize := abvInstance.NextMemoryOffset + 4 // + length
		bufferOffset := abvInstance.Offsetof("buffer")
		dataStartOffset := abvInstance.Offsetof("dataStart")
		byteLengthOffset := abvInstance.Offsetof("byteLength")
		lengthOffset := byteLengthOffset + 4
		b.needsSetU32 = true
		*sb = append(*sb, fmt.Sprintf(`  function __lowerArray(lowerElement, id, align, values) {
    if (values == null) return 0;
    const
      length = values.length,
      buffer = exports.__pin(exports.__new(length << align, %d)) >>> 0,
      header = exports.__pin(exports.__new(%d, id)) >>> 0;
    __setU32(header + %d, buffer);
    __dataview.setUint32(header + %d, buffer, true);
    __dataview.setUint32(header + %d, length << align, true);
    __dataview.setUint32(header + %d, length, true);
    for (let i = 0; i < length; ++i) lowerElement(buffer + (i << align >>> 0), values[i]);
    exports.__unpin(buffer);
    exports.__unpin(header);
    return header;
  }
`, arrayBufferId, arraySize, bufferOffset, dataStartOffset, byteLengthOffset, lengthOffset))
	}
	if b.needsLiftTypedArray {
		abvInstance := prog.ArrayBufferViewInstance()
		dataStartOffset := abvInstance.Offsetof("dataStart")
		byteLengthOffset := abvInstance.Offsetof("byteLength")
		b.needsGetU32 = true
		*sb = append(*sb, fmt.Sprintf(`  function __liftTypedArray(constructor, pointer) {
    if (!pointer) return null;
    return new constructor(
      memory.buffer,
      __getU32(pointer + %d),
      __dataview.getUint32(pointer + %d, true) / constructor.BYTES_PER_ELEMENT
    ).slice();
  }
`, dataStartOffset, byteLengthOffset))
	}
	if b.needsLowerTypedArray {
		arrayBufferId := prog.ArrayBufferInstance().Id()
		abvInstance := prog.ArrayBufferViewInstance()
		size := abvInstance.NextMemoryOffset
		bufferOffset := abvInstance.Offsetof("buffer")
		dataStartOffset := abvInstance.Offsetof("dataStart")
		byteLengthOffset := abvInstance.Offsetof("byteLength")
		b.needsSetU32 = true
		*sb = append(*sb, fmt.Sprintf(`  function __lowerTypedArray(constructor, id, align, values) {
    if (values == null) return 0;
    const
      length = values.length,
      buffer = exports.__pin(exports.__new(length << align, %d)) >>> 0,
      header = exports.__new(%d, id) >>> 0;
    __setU32(header + %d, buffer);
    __dataview.setUint32(header + %d, buffer, true);
    __dataview.setUint32(header + %d, length << align, true);
    new constructor(memory.buffer, buffer, length).set(values);
    exports.__unpin(buffer);
    return header;
  }
`, arrayBufferId, size, bufferOffset, dataStartOffset, byteLengthOffset))
	}
	if b.needsLiftStaticArray {
		objectInstance := prog.ObjectInstance()
		rtSizeOffset := objectInstance.Offsetof("rtSize") - objectInstance.NextMemoryOffset
		b.needsGetU32 = true
		*sb = append(*sb, fmt.Sprintf(`  function __liftStaticArray(liftElement, align, pointer) {
    if (!pointer) return null;
    const
      length = __getU32(pointer - %d) >>> align,
      values = new Array(length);
    for (let i = 0; i < length; ++i) values[i] = liftElement(pointer + (i << align >>> 0));
    return values;
  }
`, -int32(rtSizeOffset)))
	}
	if b.needsLowerStaticArray {
		*sb = append(*sb, `  function __lowerStaticArray(lowerElement, id, align, values, typedConstructor) {
    if (values == null) return 0;
    const
      length = values.length,
      buffer = exports.__pin(exports.__new(length << align, id)) >>> 0;
    if (typedConstructor) {
      new typedConstructor(memory.buffer, buffer, length).set(values);
    } else {
      for (let i = 0; i < length; i++) lowerElement(buffer + (i << align >>> 0), values[i]);
    }
    exports.__unpin(buffer);
    return buffer;
  }
`)
	}
	if b.needsLiftInternref || b.needsLowerInternref {
		*sb = append(*sb, "  class Internref extends Number {}\n")
	}
	if b.needsLiftInternref {
		b.needsRetain = true
		b.needsRelease = true
		*sb = append(*sb, `  const registry = new FinalizationRegistry(__release);
  function __liftInternref(pointer) {
    if (!pointer) return null;
    const sentinel = new Internref(__retain(pointer));
    registry.register(sentinel, pointer);
    return sentinel;
  }
`)
	}
	if b.needsLowerInternref {
		*sb = append(*sb, `  function __lowerInternref(value) {
    if (value == null) return 0;
    if (value instanceof Internref) return value.valueOf();
    throw TypeError("internref expected");
  }
`)
	}
	if b.needsRetain || b.needsRelease {
		*sb = append(*sb, "  const refcounts = new Map();\n")
	}
	if b.needsRetain {
		*sb = append(*sb, `  function __retain(pointer) {
    if (pointer) {
      const refcount = refcounts.get(pointer);
      if (refcount) refcounts.set(pointer, refcount + 1);
      else refcounts.set(exports.__pin(pointer), 1);
    }
    return pointer;
  }
`)
	}
	if b.needsRelease {
		*sb = append(*sb, "  function __release(pointer) {\n")
		*sb = append(*sb, "    if (pointer) {\n")
		*sb = append(*sb, "      const refcount = refcounts.get(pointer);\n")
		*sb = append(*sb, "      if (refcount === 1) exports.__unpin(pointer), refcounts.delete(pointer);\n")
		*sb = append(*sb, "      else if (refcount) refcounts.set(pointer, refcount - 1);\n")
		*sb = append(*sb, "      else throw Error(`invalid refcount '${refcount}' for reference '${pointer}'`);\n")
		*sb = append(*sb, "    }\n")
		*sb = append(*sb, "  }\n")
	}
	if b.needsNotNull {
		*sb = append(*sb, `  function __notnull() {
    throw TypeError("value must not be null");
  }
`)
	}
	if b.needsSetU8 || b.needsSetU16 || b.needsSetU32 || b.needsSetU64 ||
		b.needsSetF32 || b.needsSetF64 ||
		b.needsGetI8 || b.needsGetU8 || b.needsGetI16 || b.needsGetU16 ||
		b.needsGetI32 || b.needsGetU32 || b.needsGetI64 || b.needsGetU64 ||
		b.needsGetF32 || b.needsGetF64 {
		*sb = append(*sb, "  let __dataview = new DataView(memory.buffer);\n")
	}
	if b.needsSetU8 {
		*sb = append(*sb, makeCheckedSetter("U8", "setUint8"))
	}
	if b.needsSetU16 {
		*sb = append(*sb, makeCheckedSetter("U16", "setUint16"))
	}
	if b.needsSetU32 {
		*sb = append(*sb, makeCheckedSetter("U32", "setUint32"))
	}
	if b.needsSetU64 {
		*sb = append(*sb, makeCheckedSetter("U64", "setBigUint64"))
	}
	if b.needsSetF32 {
		*sb = append(*sb, makeCheckedSetter("F32", "setFloat32"))
	}
	if b.needsSetF64 {
		*sb = append(*sb, makeCheckedSetter("F64", "setFloat64"))
	}
	if b.needsGetI8 {
		*sb = append(*sb, makeCheckedGetter("I8", "getInt8"))
	}
	if b.needsGetU8 {
		*sb = append(*sb, makeCheckedGetter("U8", "getUint8"))
	}
	if b.needsGetI16 {
		*sb = append(*sb, makeCheckedGetter("I16", "getInt16"))
	}
	if b.needsGetU16 {
		*sb = append(*sb, makeCheckedGetter("U16", "getUint16"))
	}
	if b.needsGetI32 {
		*sb = append(*sb, makeCheckedGetter("I32", "getInt32"))
	}
	if b.needsGetU32 {
		*sb = append(*sb, makeCheckedGetter("U32", "getUint32"))
	}
	if b.needsGetI64 {
		*sb = append(*sb, makeCheckedGetter("I64", "getBigInt64"))
	}
	if b.needsGetU64 {
		*sb = append(*sb, makeCheckedGetter("U64", "getBigUint64"))
	}
	if b.needsGetF32 {
		*sb = append(*sb, makeCheckedGetter("F32", "getFloat32"))
	}
	if b.needsGetF64 {
		*sb = append(*sb, makeCheckedGetter("F64", "getFloat64"))
	}

	exportStart := options.ExportStart
	if exportStart != "" {
		*sb = append(*sb, fmt.Sprintf("  exports.%s();\n", exportStart))
	}

	if hasAdaptedExports {
		*sb = append(*sb, "  return adaptedExports;\n}\n")
	} else {
		*sb = append(*sb, "  return exports;\n}\n")
	}
	b.indentLevel--
	if b.indentLevel != 0 {
		panic("indent level mismatch")
	}

	if b.esm {
		*sb = append(*sb, "export const {\n")
		if prog.Options.ExportMemory {
			*sb = append(*sb, "  memory,\n")
		}
		if prog.Options.ExportTable {
			*sb = append(*sb, "  table,\n")
		}
		if prog.Options.ExportRuntime {
			for _, name := range RuntimeFunctions {
				*sb = append(*sb, "  ")
				*sb = append(*sb, name)
				*sb = append(*sb, ",\n")
			}
			for _, name := range RuntimeGlobals {
				*sb = append(*sb, "  ")
				*sb = append(*sb, name)
				*sb = append(*sb, ",\n")
			}
		}
		for _, name := range b.exports {
			*sb = append(*sb, "  ")
			*sb = append(*sb, name)
			*sb = append(*sb, ",\n")
		}
		*sb = append(*sb, "} = await (async url => instantiate(\n")
		*sb = append(*sb, "  await (async () => {\n")
		*sb = append(*sb, "    const isNodeOrBun = typeof process != \"undefined\" && process.versions != null && (process.versions.node != null || process.versions.bun != null);\n")
		*sb = append(*sb, "    if (isNodeOrBun) { return globalThis.WebAssembly.compile(await (await import(\"node:fs/promises\")).readFile(url)); }\n")
		*sb = append(*sb, "    else { return await globalThis.WebAssembly.compileStreaming(globalThis.fetch(url)); }\n")
		*sb = append(*sb, "  })(), {\n")

		needsMaybeDefault := false
		importExpr := make([]string, 0)
		for moduleName := range mappings {
			if moduleName == "env" {
				indentSB(sb, 2)
				*sb = append(*sb, "env: globalThis,\n")
			} else {
				moduleId := b.ensureModuleId(moduleName)
				indentSB(sb, 2)
				if util.IsIdentifier(moduleName) {
					*sb = append(*sb, moduleName)
				} else {
					*sb = append(*sb, "\"")
					*sb = append(*sb, util.EscapeString(moduleName, util.CharCodeDoubleQuote))
					*sb = append(*sb, "\"")
				}
				*sb = append(*sb, ": __maybeDefault(__import")
				*sb = append(*sb, fmt.Sprintf("%d", moduleId))
				*sb = append(*sb, "),\n")
				importExpr = append(importExpr, "import * as __import")
				importExpr = append(importExpr, fmt.Sprintf("%d", moduleId))
				importExpr = append(importExpr, " from \"")
				importExpr = append(importExpr, util.EscapeString(importToModule(moduleName), util.CharCodeDoubleQuote))
				importExpr = append(importExpr, "\";\n")
				needsMaybeDefault = true
			}
		}
		(*sb)[0] = strings.Join(importExpr, "")
		*sb = append(*sb, "  }\n")
		*sb = append(*sb, fmt.Sprintf("))(new URL(\"%s.wasm\", import.meta.url));\n",
			util.EscapeString(options.BasenameHint, util.CharCodeDoubleQuote)))
		if needsMaybeDefault {
			*sb = append(*sb, "function __maybeDefault(module) {\n")
			*sb = append(*sb, "  return typeof module.default === \"object\" && Object.keys(module).length == 1\n")
			*sb = append(*sb, "    ? module.default\n")
			*sb = append(*sb, "    : module;\n")
			*sb = append(*sb, "}\n")
		}
	}
	return strings.Join(*sb, "")
}

func (b *JSBuilder) ensureModuleId(moduleName string) int32 {
	if moduleName == "env" {
		return -1
	}
	if id, ok := b.importMappings[moduleName]; ok {
		return id
	}
	id := int32(len(b.importMappings))
	b.importMappings[moduleName] = id
	return id
}

// makeLiftFromValue lifts a WebAssembly value to a JavaScript value, as an expression.
func (b *JSBuilder) makeLiftFromValue(valueExpr string, typ *types.Type, sb *[]string) {
	if typ.IsInternalReference() {
		clazz := b.getClassOrWrapper(typ)
		if clazz == nil {
			*sb = append(*sb, valueExpr)
			return
		}
		prog := b.Program
		if clazz.ExtendsPrototype(prog.ArrayBufferInstance().Prototype) {
			*sb = append(*sb, "__liftBuffer(")
			b.needsLiftBuffer = true
		} else if clazz.ExtendsPrototype(prog.StringInstance().Prototype) {
			*sb = append(*sb, "__liftString(")
			b.needsLiftString = true
		} else if clazz.ExtendsPrototype(prog.ArrayPrototype()) {
			valueType := clazz.GetArrayValueType()
			*sb = append(*sb, "__liftArray(")
			b.makeLiftFromMemoryFunc(valueType, sb)
			*sb = append(*sb, ", ")
			*sb = append(*sb, fmt.Sprintf("%d", valueType.AlignLog2()))
			*sb = append(*sb, ", ")
			b.needsLiftArray = true
		} else if clazz.ExtendsPrototype(prog.StaticArrayPrototype()) {
			valueType := clazz.GetArrayValueType()
			*sb = append(*sb, "__liftStaticArray(")
			b.makeLiftFromMemoryFunc(valueType, sb)
			*sb = append(*sb, ", ")
			*sb = append(*sb, fmt.Sprintf("%d", valueType.AlignLog2()))
			*sb = append(*sb, ", ")
			b.needsLiftStaticArray = true
		} else if clazz.ExtendsPrototype(prog.ArrayBufferViewInstance().Prototype) {
			*sb = append(*sb, "__liftTypedArray(")
			className := clazz.GetName()
			if className == "Uint64Array" {
				*sb = append(*sb, "BigUint64Array")
			} else if className == "Int64Array" {
				*sb = append(*sb, "BigInt64Array")
			} else {
				*sb = append(*sb, className)
			}
			*sb = append(*sb, ", ")
			b.needsLiftTypedArray = true
		} else if isPlainObject(clazz, prog) {
			*sb = append(*sb, "__liftRecord")
			*sb = append(*sb, fmt.Sprintf("%d", clazz.Id()))
			*sb = append(*sb, "(")
			if _, ok := b.deferredLifts[clazz]; !ok {
				b.deferredLifts[clazz] = struct{}{}
				prevIndentLevel := b.indentLevel
				b.indentLevel = 1
				b.deferredCode = append(b.deferredCode, b.makeLiftRecord(clazz))
				b.indentLevel = prevIndentLevel
			}
		} else {
			*sb = append(*sb, "__liftInternref(")
			b.needsLiftInternref = true
		}
		*sb = append(*sb, valueExpr)
		if !strings.HasPrefix(valueExpr, "__get") {
			*sb = append(*sb, " >>> 0")
		}
		*sb = append(*sb, ")")
	} else {
		if typ == types.TypeBool {
			*sb = append(*sb, valueExpr+" != 0")
		} else if typ.IsUnsignedIntegerValue() && typ.Size >= 32 {
			if typ.Size == 64 {
				*sb = append(*sb, fmt.Sprintf("BigInt.asUintN(64, %s)", valueExpr))
			} else {
				*sb = append(*sb, valueExpr+" >>> 0")
			}
		} else {
			*sb = append(*sb, valueExpr)
		}
	}
}

// makeLowerToValue lowers a JavaScript value to a WebAssembly value, as an expression.
func (b *JSBuilder) makeLowerToValue(valueExpr string, typ *types.Type, sb *[]string) {
	if typ.IsInternalReference() {
		clazz := b.getClassOrWrapper(typ)
		if clazz == nil {
			*sb = append(*sb, valueExpr)
			return
		}
		prog := b.Program
		if clazz.ExtendsPrototype(prog.ArrayBufferInstance().Prototype) {
			*sb = append(*sb, "__lowerBuffer(")
			b.needsLowerBuffer = true
		} else if clazz.ExtendsPrototype(prog.StringInstance().Prototype) {
			*sb = append(*sb, "__lowerString(")
			b.needsLowerString = true
		} else if clazz.ExtendsPrototype(prog.ArrayPrototype()) {
			valueType := clazz.GetArrayValueType()
			*sb = append(*sb, "__lowerArray(")
			b.makeLowerToMemoryFunc(valueType, sb)
			*sb = append(*sb, ", ")
			*sb = append(*sb, fmt.Sprintf("%d", clazz.Id()))
			*sb = append(*sb, ", ")
			*sb = append(*sb, fmt.Sprintf("%d", clazz.GetArrayValueType().AlignLog2()))
			*sb = append(*sb, ", ")
			b.needsLowerArray = true
		} else if clazz.ExtendsPrototype(prog.StaticArrayPrototype()) {
			valueType := clazz.GetArrayValueType()
			*sb = append(*sb, "__lowerStaticArray(")
			b.makeLowerToMemoryFunc(valueType, sb)
			*sb = append(*sb, ", ")
			*sb = append(*sb, fmt.Sprintf("%d", clazz.Id()))
			*sb = append(*sb, ", ")
			*sb = append(*sb, fmt.Sprintf("%d", valueType.AlignLog2()))
			*sb = append(*sb, ", ")
			b.needsLowerStaticArray = true
		} else if clazz.ExtendsPrototype(prog.ArrayBufferViewInstance().Prototype) {
			valueType := clazz.GetArrayValueType()
			*sb = append(*sb, "__lowerTypedArray(")
			if valueType == types.TypeU64 {
				*sb = append(*sb, "BigUint64Array")
			} else if valueType == types.TypeI64 {
				*sb = append(*sb, "BigInt64Array")
			} else {
				*sb = append(*sb, clazz.GetName())
			}
			*sb = append(*sb, ", ")
			*sb = append(*sb, fmt.Sprintf("%d", clazz.Id()))
			*sb = append(*sb, ", ")
			*sb = append(*sb, fmt.Sprintf("%d", clazz.GetArrayValueType().AlignLog2()))
			*sb = append(*sb, ", ")
			b.needsLowerTypedArray = true
		} else if isPlainObject(clazz, prog) {
			*sb = append(*sb, "__lowerRecord")
			*sb = append(*sb, fmt.Sprintf("%d", clazz.Id()))
			*sb = append(*sb, "(")
			if _, ok := b.deferredLowers[clazz]; !ok {
				b.deferredLowers[clazz] = struct{}{}
				prevIndentLevel := b.indentLevel
				b.indentLevel = 1
				b.deferredCode = append(b.deferredCode, b.makeLowerRecord(clazz))
				b.indentLevel = prevIndentLevel
			}
		} else {
			*sb = append(*sb, "__lowerInternref(")
			b.needsLowerInternref = true
		}
		*sb = append(*sb, valueExpr)
		if clazz.ExtendsPrototype(prog.StaticArrayPrototype()) {
			// optional last argument for __lowerStaticArray
			valueType := clazz.GetArrayValueType()
			if valueType.IsNumericValue() {
				*sb = append(*sb, ", ")
				switch {
				case valueType == types.TypeU8 || valueType == types.TypeBool:
					*sb = append(*sb, "Uint8Array")
				case valueType == types.TypeI8:
					*sb = append(*sb, "Int8Array")
				case valueType == types.TypeU16:
					*sb = append(*sb, "Uint16Array")
				case valueType == types.TypeI16:
					*sb = append(*sb, "Int16Array")
				case valueType == types.TypeU32 || valueType == types.TypeUsize32:
					*sb = append(*sb, "Uint32Array")
				case valueType == types.TypeI32 || valueType == types.TypeIsize32:
					*sb = append(*sb, "Int32Array")
				case valueType == types.TypeU64 || valueType == types.TypeUsize64:
					*sb = append(*sb, "BigUint64Array")
				case valueType == types.TypeI64 || valueType == types.TypeIsize64:
					*sb = append(*sb, "BigInt64Array")
				case valueType == types.TypeF32:
					*sb = append(*sb, "Float32Array")
				case valueType == types.TypeF64:
					*sb = append(*sb, "Float64Array")
				default:
					panic("unreachable")
				}
			}
		}
		*sb = append(*sb, ")")
		if !typ.Is(types.TypeFlagNullable) {
			b.needsNotNull = true
			*sb = append(*sb, " || __notnull()")
		}
	} else {
		*sb = append(*sb, valueExpr)
		if typ.IsIntegerValue() && typ.Size == 64 {
			*sb = append(*sb, " || 0n")
		} else if typ == types.TypeBool {
			*sb = append(*sb, " ? 1 : 0")
		}
	}
}

func (b *JSBuilder) ensureLiftFromMemoryFn(valueType *types.Type) string {
	if valueType.IsInternalReference() {
		if b.Program.Options.IsWasm64() {
			b.needsGetU64 = true
			return "__getU64"
		}
		b.needsGetU32 = true
		return "__getU32"
	}
	switch {
	case valueType == types.TypeI8:
		b.needsGetI8 = true
		return "__getI8"
	case valueType == types.TypeU8 || valueType == types.TypeBool:
		b.needsGetU8 = true
		return "__getU8"
	case valueType == types.TypeI16:
		b.needsGetI16 = true
		return "__getI16"
	case valueType == types.TypeU16:
		b.needsGetU16 = true
		return "__getU16"
	case valueType == types.TypeI32 || valueType == types.TypeIsize32:
		b.needsGetI32 = true
		return "__getI32"
	case valueType == types.TypeU32 || valueType == types.TypeUsize32:
		b.needsGetU32 = true
		return "__getU32"
	case valueType == types.TypeI64 || valueType == types.TypeIsize64:
		b.needsGetI64 = true
		return "__getI64"
	case valueType == types.TypeU64 || valueType == types.TypeUsize64:
		b.needsGetU64 = true
		return "__getU64"
	case valueType == types.TypeF32:
		b.needsGetF32 = true
		return "__getF32"
	case valueType == types.TypeF64:
		b.needsGetF64 = true
		return "__getF64"
	}
	return "(() => { throw Error(\"unsupported type\"); })"
}

// makeLiftFromMemoryFunc lifts a WebAssembly memory address to a JavaScript value, as a function.
func (b *JSBuilder) makeLiftFromMemoryFunc(valueType *types.Type, sb *[]string) {
	fn := b.ensureLiftFromMemoryFn(valueType)
	if valueType.IsInternalReference() ||
		valueType == types.TypeBool ||
		(valueType.IsUnsignedIntegerValue() && valueType.Size >= 32) {
		*sb = append(*sb, "pointer => ")
		b.makeLiftFromValue(fn+"(pointer)", valueType, sb)
	} else {
		*sb = append(*sb, fn)
	}
}

// makeLiftFromMemoryCall lifts a WebAssembly memory address to a JavaScript value, as a call.
func (b *JSBuilder) makeLiftFromMemoryCall(valueType *types.Type, sb *[]string, pointerExpr string) {
	fn := b.ensureLiftFromMemoryFn(valueType)
	if valueType.IsInternalReference() {
		b.makeLiftFromValue(fn+"("+pointerExpr+")", valueType, sb)
	} else {
		*sb = append(*sb, fn)
		*sb = append(*sb, "(")
		*sb = append(*sb, pointerExpr)
		*sb = append(*sb, ")")
		if valueType == types.TypeBool {
			*sb = append(*sb, " != 0")
		}
	}
}

func (b *JSBuilder) ensureLowerToMemoryFn(valueType *types.Type) string {
	if valueType.IsInternalReference() {
		if b.Program.Options.IsWasm64() {
			b.needsSetU64 = true
			return "__setU64"
		}
		b.needsSetU32 = true
		return "__setU32"
	}
	switch {
	case valueType == types.TypeI8 || valueType == types.TypeU8 || valueType == types.TypeBool:
		b.needsSetU8 = true
		return "__setU8"
	case valueType == types.TypeI16 || valueType == types.TypeU16:
		b.needsSetU16 = true
		return "__setU16"
	case valueType == types.TypeI32 || valueType == types.TypeU32 ||
		valueType == types.TypeIsize32 || valueType == types.TypeUsize32:
		b.needsSetU32 = true
		return "__setU32"
	case valueType == types.TypeI64 || valueType == types.TypeU64 ||
		valueType == types.TypeIsize64 || valueType == types.TypeUsize64:
		b.needsSetU64 = true
		return "__setU64"
	case valueType == types.TypeF32:
		b.needsSetF32 = true
		return "__setF32"
	case valueType == types.TypeF64:
		b.needsSetF64 = true
		return "__setF64"
	}
	return "(() => { throw Error(\"unsupported type\") })"
}

// makeLowerToMemoryFunc lowers a JavaScript value to a WebAssembly memory address, as a function.
func (b *JSBuilder) makeLowerToMemoryFunc(valueType *types.Type, sb *[]string) {
	fn := b.ensureLowerToMemoryFn(valueType)
	if valueType.IsInternalReference() {
		*sb = append(*sb, "(pointer, value) => { ")
		*sb = append(*sb, fn)
		*sb = append(*sb, "(pointer, ")
		b.makeLowerToValue("value", valueType, sb)
		*sb = append(*sb, "); }")
	} else {
		*sb = append(*sb, fn)
	}
}

// makeLowerToMemoryCall lowers a JavaScript value to a WebAssembly memory address, as a call.
func (b *JSBuilder) makeLowerToMemoryCall(valueType *types.Type, sb *[]string, pointerExpr string, valueExpr string) {
	fn := b.ensureLowerToMemoryFn(valueType)
	*sb = append(*sb, fn)
	*sb = append(*sb, "(")
	*sb = append(*sb, pointerExpr)
	*sb = append(*sb, ", ")
	b.makeLowerToValue(valueExpr, valueType, sb)
	*sb = append(*sb, ")")
}

func (b *JSBuilder) makeLiftRecord(clazz *program.Class) string {
	sb := make([]string, 0)
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "function __liftRecord")
	sb = append(sb, fmt.Sprintf("%d", clazz.Id()))
	sb = append(sb, "(pointer) {\n")
	b.indentLevel++
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "// ")
	sb = append(sb, clazz.GetResolvedType().String())
	sb = append(sb, "\n")
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "// Hint: Opt-out from lifting as a record by providing an empty constructor\n")
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "if (!pointer) return null;\n")
	indentSB(&sb, b.indentLevel)
	b.indentLevel++
	sb = append(sb, "return {\n")
	members := clazz.GetMembers()
	if members != nil {
		for memberName, member := range members {
			_ = memberName
			if member.GetElementKind() != program.ElementKindPropertyPrototype {
				continue
			}
			pp := member.(*program.PropertyPrototype)
			property := pp.PropertyInstance
			if property == nil || !property.IsField() {
				continue
			}
			if property.MemoryOffset < 0 {
				panic("expected non-negative memory offset")
			}
			indentSB(&sb, b.indentLevel)
			sb = append(sb, property.GetName())
			sb = append(sb, ": ")
			b.makeLiftFromMemoryCall(property.GetResolvedType(), &sb, fmt.Sprintf("pointer + %d", property.MemoryOffset))
			sb = append(sb, ",\n")
		}
	}
	b.indentLevel--
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "};\n")
	b.indentLevel--
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "}\n")
	return strings.Join(sb, "")
}

func (b *JSBuilder) makeLowerRecord(clazz *program.Class) string {
	sb := make([]string, 0)
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "function __lowerRecord")
	sb = append(sb, fmt.Sprintf("%d", clazz.Id()))
	sb = append(sb, "(value) {\n")
	b.indentLevel++
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "// ")
	sb = append(sb, clazz.GetResolvedType().String())
	sb = append(sb, "\n")
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "// Hint: Opt-out from lowering as a record by providing an empty constructor\n")
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "if (value == null) return 0;\n")
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "const pointer = exports.__pin(exports.__new(")
	sb = append(sb, fmt.Sprintf("%d", clazz.NextMemoryOffset))
	sb = append(sb, ", ")
	sb = append(sb, fmt.Sprintf("%d", clazz.Id()))
	sb = append(sb, "));\n")
	members := clazz.GetMembers()
	if members != nil {
		for memberName, member := range members {
			if member.GetElementKind() != program.ElementKindPropertyPrototype {
				continue
			}
			pp := member.(*program.PropertyPrototype)
			property := pp.PropertyInstance
			if property == nil || !property.IsField() {
				continue
			}
			if property.MemoryOffset < 0 {
				panic("expected non-negative memory offset")
			}
			indentSB(&sb, b.indentLevel)
			b.makeLowerToMemoryCall(property.GetResolvedType(), &sb,
				fmt.Sprintf("pointer + %d", property.MemoryOffset),
				fmt.Sprintf("value.%s", memberName))
			sb = append(sb, ";\n")
		}
	}
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "exports.__unpin(pointer);\n")
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "return pointer;\n")
	b.indentLevel--
	indentSB(&sb, b.indentLevel)
	sb = append(sb, "}\n")
	return strings.Join(sb, "")
}

func (b *JSBuilder) getClassOrWrapper(typ *types.Type) *program.Class {
	classRef := typ.GetClassOrWrapper(b.Program)
	if classRef == nil {
		return nil
	}
	if c, ok := classRef.(*program.Class); ok {
		return c
	}
	return nil
}

// --- Helper functions ---

func isPlainValue(typ *types.Type, mode jsMode) bool {
	if mode == jsModeImport {
		if typ == types.TypeBool {
			return false
		}
		if typ.IsIntegerValue() && typ.Size == 64 {
			return false
		}
	} else {
		if typ == types.TypeBool {
			return false
		}
		if typ.IsUnsignedIntegerValue() && typ.Size >= 32 {
			return false
		}
	}
	return !typ.IsInternalReference()
}

func isPlainFunction(signature *types.Signature, mode jsMode) bool {
	parameterTypes := signature.ParameterTypes
	var inverseMode jsMode
	if mode == jsModeImport {
		inverseMode = jsModeExport
	} else {
		inverseMode = jsModeImport
	}
	if !isPlainValue(signature.ReturnType, mode) {
		return false
	}
	for _, ptype := range parameterTypes {
		if !isPlainValue(ptype, inverseMode) {
			return false
		}
	}
	return true
}

func isPlainObject(clazz *program.Class, prog *program.Program) bool {
	if clazz.Base != nil && !clazz.Prototype.ImplicitlyExtendsObject {
		return false
	}
	members := clazz.GetMembers()
	if members != nil {
		for _, member := range members {
			if member.IsAny(common.CommonFlagsPrivate | common.CommonFlagsProtected) {
				return false
			}
			if member.Is(common.CommonFlagsConstructor) {
				decl := member.GetDeclaration()
				if decl != nil && decl.GetRange() != nil {
					nativeRange := prog.NativeFile.Source.GetRange()
					if nativeRange != nil && decl.GetRange() != nativeRange {
						return false
					}
				}
			}
		}
	}
	return true
}

func indentSBSlice(sb *[]string, level int) {
	*sb = append(*sb, strings.Repeat("  ", level))
}

func indentText(text string, indentLevel int, sb *[]string, butFirst bool) {
	lineStart := 0
	length := len(text)
	pos := 0
	for pos < length {
		if text[pos] == '\n' {
			if butFirst {
				butFirst = false
			} else {
				indentSBSlice(sb, indentLevel)
			}
			*sb = append(*sb, text[lineStart:pos+1])
			lineStart = pos + 1
		}
		pos++
	}
	if lineStart < length {
		if !butFirst {
			indentSBSlice(sb, indentLevel)
		}
		*sb = append(*sb, text[lineStart:])
	}
}

// LiftRequiresExportRuntime tests if lifting the given type requires export runtime.
func LiftRequiresExportRuntime(typ *types.Type) bool {
	if !typ.IsInternalReference() {
		return false
	}
	classRef := typ.GetClass()
	if classRef == nil {
		// functions lift as internref using __pin
		if typ.GetSignature() != nil {
			return true
		}
		return false
	}
	clazz, ok := classRef.(*program.Class)
	if !ok {
		return true
	}
	prog := clazz.GetProgram()
	// flat collections lift via memory copy
	if clazz.ExtendsPrototype(prog.ArrayBufferInstance().Prototype) ||
		clazz.ExtendsPrototype(prog.StringInstance().Prototype) ||
		clazz.ExtendsPrototype(prog.ArrayBufferViewInstance().Prototype) {
		return false
	}
	// nested collections lift depending on element type
	if clazz.ExtendsPrototype(prog.ArrayPrototype()) ||
		clazz.ExtendsPrototype(prog.StaticArrayPrototype()) {
		return LiftRequiresExportRuntime(clazz.GetArrayValueType())
	}
	return true
}

// LowerRequiresExportRuntime tests if lowering the given type requires export runtime.
func LowerRequiresExportRuntime(typ *types.Type) bool {
	if !typ.IsInternalReference() {
		return false
	}
	classRef := typ.GetClass()
	if classRef == nil {
		// lowers by reference
		if typ.GetSignature() != nil {
			return false
		}
		return false
	}
	clazz, ok := classRef.(*program.Class)
	if !ok {
		return false
	}
	prog := clazz.GetProgram()
	// lowers using __new
	if clazz.ExtendsPrototype(prog.ArrayBufferInstance().Prototype) ||
		clazz.ExtendsPrototype(prog.StringInstance().Prototype) ||
		clazz.ExtendsPrototype(prog.ArrayBufferViewInstance().Prototype) ||
		clazz.ExtendsPrototype(prog.ArrayPrototype()) ||
		clazz.ExtendsPrototype(prog.StaticArrayPrototype()) {
		return true
	}
	return isPlainObject(clazz, prog)
}

// makeCheckedSetter makes a checked setter function to memory for the given basic type.
func makeCheckedSetter(typeName string, fn string) string {
	return fmt.Sprintf(`  function __set%s(pointer, value) {
    try {
      __dataview.%s(pointer, value, true);
    } catch {
      __dataview = new DataView(memory.buffer);
      __dataview.%s(pointer, value, true);
    }
  }
`, typeName, fn, fn)
}

// makeCheckedGetter makes a checked getter function from memory for the given basic type.
func makeCheckedGetter(typeName string, fn string) string {
	return fmt.Sprintf(`  function __get%s(pointer) {
    try {
      return __dataview.%s(pointer, true);
    } catch {
      __dataview = new DataView(memory.buffer);
      return __dataview.%s(pointer, true);
    }
  }
`, typeName, fn, fn)
}
