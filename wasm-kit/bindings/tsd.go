// Ported from: assemblyscript/src/bindings/tsd.ts
package bindings

import (
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/program"
	"github.com/brainlet/brainkit/wasm-kit/types"
	"github.com/brainlet/brainkit/wasm-kit/util"
)

// tsdMode distinguishes import vs export context for type generation.
type tsdMode int

const (
	tsdModeImport tsdMode = iota
	tsdModeExport
)

// TSDBuilder generates TypeScript definition files (.d.ts).
// Ported from: assemblyscript/src/bindings/tsd.ts TSDBuilder
type TSDBuilder struct {
	ExportsWalker
	esm             bool
	sb              strings.Builder
	indentLevel     int
	seenObjectTypes map[*program.Class]string
	deferredTypings []string
}

// BuildTSD builds TypeScript definitions for the specified program.
func BuildTSD(prog *program.Program, esm bool) string {
	b := NewTSDBuilder(prog, esm, false)
	return b.Build()
}

// NewTSDBuilder constructs a new TypeScript definitions builder.
func NewTSDBuilder(prog *program.Program, esm bool, includePrivate bool) *TSDBuilder {
	b := &TSDBuilder{
		ExportsWalker:   NewExportsWalker(prog, includePrivate),
		esm:             esm,
		seenObjectTypes: make(map[*program.Class]string),
		deferredTypings: make([]string, 0),
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

func (b *TSDBuilder) visitGlobal(name string, element *program.Global) {
	sb := &b.sb
	typ := element.GetResolvedType()
	tsType := b.toTypeScriptType(typ, tsdModeExport)
	util.Indent(sb, b.indentLevel)
	sb.WriteString("/** ")
	sb.WriteString(element.GetInternalName())
	sb.WriteString(" */\n")
	util.Indent(sb, b.indentLevel)
	sb.WriteString("export ")
	if b.esm {
		sb.WriteString("declare ")
	}
	sb.WriteString("const ")
	sb.WriteString(name)
	sb.WriteString(": {\n")
	b.indentLevel++
	util.Indent(sb, b.indentLevel)
	sb.WriteString("/** @type `")
	sb.WriteString(typ.String())
	sb.WriteString("` */\n")
	util.Indent(sb, b.indentLevel)
	sb.WriteString("get value(): ")
	sb.WriteString(tsType)
	if !element.Is(common.CommonFlagsConst) {
		sb.WriteString(";\n")
		util.Indent(sb, b.indentLevel)
		sb.WriteString("set value(value: ")
		sb.WriteString(tsType)
		sb.WriteString(");\n")
	} else {
		sb.WriteString("\n")
	}
	b.indentLevel--
	util.Indent(sb, b.indentLevel)
	sb.WriteString("};\n")
}

func (b *TSDBuilder) visitEnum(name string, element *program.Enum) {
	sb := &b.sb
	util.Indent(sb, b.indentLevel)
	sb.WriteString("/** ")
	sb.WriteString(element.GetInternalName())
	sb.WriteString(" */\n")
	util.Indent(sb, b.indentLevel)
	b.indentLevel++
	sb.WriteString("export ")
	if b.esm {
		sb.WriteString("declare ")
	}
	sb.WriteString("enum ")
	sb.WriteString(name)
	sb.WriteString(" {\n")
	members := element.GetMembers()
	if members != nil {
		for memberName, member := range members {
			if member.GetElementKind() != program.ElementKindEnumValue {
				continue
			}
			util.Indent(sb, b.indentLevel)
			sb.WriteString("/** @type `i32` */\n")
			util.Indent(sb, b.indentLevel)
			sb.WriteString(memberName)
			sb.WriteString(",\n")
		}
	}
	b.indentLevel--
	util.Indent(sb, b.indentLevel)
	sb.WriteString("}\n")
}

func (b *TSDBuilder) visitFunction(name string, element *program.Function) {
	sb := &b.sb
	signature := element.Signature
	util.Indent(sb, b.indentLevel)
	sb.WriteString("/**\n")
	util.Indent(sb, b.indentLevel)
	sb.WriteString(" * ")
	sb.WriteString(element.GetInternalName())
	sb.WriteString("\n")
	parameterTypes := signature.ParameterTypes
	numParameters := len(parameterTypes)
	for i := 0; i < numParameters; i++ {
		util.Indent(sb, b.indentLevel)
		sb.WriteString(" * @param ")
		sb.WriteString(element.GetParameterName(int32(i)))
		sb.WriteString(" `")
		sb.WriteString(parameterTypes[i].String())
		sb.WriteString("`\n")
	}
	returnType := signature.ReturnType
	if returnType != types.TypeVoid {
		util.Indent(sb, b.indentLevel)
		sb.WriteString(" * @returns `")
		sb.WriteString(returnType.String())
		sb.WriteString("`\n")
	}
	util.Indent(sb, b.indentLevel)
	sb.WriteString(" */\n")
	util.Indent(sb, b.indentLevel)
	sb.WriteString("export ")
	if b.esm {
		sb.WriteString("declare ")
	}
	sb.WriteString("function ")
	sb.WriteString(name)
	sb.WriteString("(")
	requiredParameters := signature.RequiredParameters
	for i := 0; i < numParameters; i++ {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(element.GetParameterName(int32(i)))
		if int32(i) >= requiredParameters {
			sb.WriteString("?")
		}
		sb.WriteString(": ")
		sb.WriteString(b.toTypeScriptType(parameterTypes[i], tsdModeImport))
	}
	sb.WriteString("): ")
	sb.WriteString(b.toTypeScriptType(returnType, tsdModeExport))
	sb.WriteString(";\n")
}

func (b *TSDBuilder) visitClass(name string, element *program.Class) {
	// not implemented
}

func (b *TSDBuilder) visitInterface(name string, element *program.Interface) {
	// not implemented
}

func (b *TSDBuilder) visitNamespace(name string, element program.Element) {
	// not implemented
}

func (b *TSDBuilder) visitAlias(name string, element program.Element, originalName string) {
	// not implemented
}

// Build builds the TypeScript definitions string.
func (b *TSDBuilder) Build() string {
	sb := &b.sb
	prog := b.Program
	if !b.esm {
		sb.WriteString("declare namespace __AdaptedExports {\n")
		b.indentLevel++
	}
	declarePrefix := ""
	if b.esm {
		declarePrefix = "declare "
	}
	if prog.Options.ExportMemory {
		util.Indent(sb, b.indentLevel)
		sb.WriteString("/** Exported memory */\n")
		util.Indent(sb, b.indentLevel)
		sb.WriteString("export ")
		sb.WriteString(declarePrefix)
		sb.WriteString("const memory: WebAssembly.Memory;\n")
	}
	if prog.Options.ExportTable {
		util.Indent(sb, b.indentLevel)
		sb.WriteString("/** Exported table */\n")
		util.Indent(sb, b.indentLevel)
		sb.WriteString("export ")
		sb.WriteString(declarePrefix)
		sb.WriteString("const table: WebAssembly.Table;\n")
	}
	if prog.Options.ExportRuntime {
		util.Indent(sb, b.indentLevel)
		sb.WriteString("// Exported runtime interface\n")
		util.Indent(sb, b.indentLevel)
		sb.WriteString(fmt.Sprintf("export %sfunction __new(size: number, id: number): number;\n", declarePrefix))
		util.Indent(sb, b.indentLevel)
		sb.WriteString(fmt.Sprintf("export %sfunction __pin(ptr: number): number;\n", declarePrefix))
		util.Indent(sb, b.indentLevel)
		sb.WriteString(fmt.Sprintf("export %sfunction __unpin(ptr: number): void;\n", declarePrefix))
		util.Indent(sb, b.indentLevel)
		sb.WriteString(fmt.Sprintf("export %sfunction __collect(): void;\n", declarePrefix))
		util.Indent(sb, b.indentLevel)
		sb.WriteString(fmt.Sprintf("export %sconst __rtti_base: number;\n", declarePrefix))
	}
	b.Walk()
	if !b.esm {
		b.indentLevel--
		sb.WriteString("}\n")
	}
	for _, dt := range b.deferredTypings {
		sb.WriteString(dt)
	}
	if !b.esm {
		sb.WriteString("/** Instantiates the compiled WebAssembly module with the given imports. */\n")
		sb.WriteString("export declare function instantiate(module: WebAssembly.Module, imports: {\n")
		moduleImports := prog.ModuleImports
		for moduleName := range moduleImports {
			sb.WriteString("  ")
			if util.IsIdentifier(moduleName) {
				sb.WriteString(moduleName)
			} else {
				sb.WriteString("\"")
				sb.WriteString(util.EscapeString(moduleName, util.CharCodeDoubleQuote))
				sb.WriteString("\"")
			}
			sb.WriteString(": unknown,\n")
		}
		sb.WriteString("}): Promise<typeof __AdaptedExports>;\n")
	}
	return sb.String()
}

func (b *TSDBuilder) isPlainObject(clazz *program.Class) bool {
	// A plain object does not inherit and does not have a constructor or private properties
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
				// a generated constructor is ok
				decl := member.GetDeclaration()
				if decl != nil && decl.GetRange() != nil {
					nativeRange := b.Program.NativeFile.Source.GetRange()
					if nativeRange != nil && decl.GetRange() != nativeRange {
						return false
					}
				}
			}
		}
	}
	return true
}

func (b *TSDBuilder) toTypeScriptType(typ *types.Type, mode tsdMode) string {
	if typ.IsInternalReference() {
		var tsb strings.Builder
		clazz := b.getClassOrWrapper(typ)
		if clazz == nil {
			return "unknown"
		}
		prog := b.Program
		if clazz.ExtendsPrototype(prog.ArrayBufferInstance().Prototype) {
			tsb.WriteString("ArrayBuffer")
		} else if clazz.ExtendsPrototype(prog.StringInstance().Prototype) {
			tsb.WriteString("string")
		} else if clazz.ExtendsPrototype(prog.ArrayPrototype()) {
			valueType := clazz.GetArrayValueType()
			tsb.WriteString("Array<")
			tsb.WriteString(b.toTypeScriptType(valueType, mode))
			tsb.WriteString(">")
		} else if clazz.ExtendsPrototype(prog.StaticArrayPrototype()) {
			valueType := clazz.GetArrayValueType()
			tsb.WriteString("ArrayLike<")
			tsb.WriteString(b.toTypeScriptType(valueType, mode))
			tsb.WriteString(">")
		} else if clazz.ExtendsPrototype(prog.ArrayBufferViewInstance().Prototype) {
			valueType := clazz.GetArrayValueType()
			if valueType == types.TypeI8 {
				tsb.WriteString("Int8Array")
			} else if valueType == types.TypeU8 {
				if clazz.ExtendsPrototype(prog.Uint8ClampedArrayPrototype()) {
					tsb.WriteString("Uint8ClampedArray")
				} else {
					tsb.WriteString("Uint8Array")
				}
			} else if valueType == types.TypeI16 {
				tsb.WriteString("Int16Array")
			} else if valueType == types.TypeU16 {
				tsb.WriteString("Uint16Array")
			} else if valueType == types.TypeI32 {
				tsb.WriteString("Int32Array")
			} else if valueType == types.TypeU32 {
				tsb.WriteString("Uint32Array")
			} else if valueType == types.TypeI64 {
				tsb.WriteString("BigInt64Array")
			} else if valueType == types.TypeU64 {
				tsb.WriteString("BigUint64Array")
			} else if valueType == types.TypeF32 {
				tsb.WriteString("Float32Array")
			} else if valueType == types.TypeF64 {
				tsb.WriteString("Float64Array")
			} else {
				tsb.WriteString("unknown")
			}
		} else {
			seenObjectTypes := b.seenObjectTypes
			if typeName, ok := seenObjectTypes[clazz]; ok {
				tsb.WriteString(typeName)
				if b.isPlainObject(clazz) {
					if mode == tsdModeExport {
						tsb.WriteString("<never>")
					} else {
						tsb.WriteString("<undefined>")
					}
				}
			} else {
				isPlain := b.isPlainObject(clazz)
				var typeName string
				if isPlain {
					typeName = fmt.Sprintf("__Record%d", clazz.Id())
				} else {
					typeName = fmt.Sprintf("__Internref%d", clazz.Id())
				}
				tsb.WriteString(typeName)
				seenObjectTypes[clazz] = typeName
				if isPlain {
					if mode == tsdModeExport {
						tsb.WriteString("<never>")
					} else {
						tsb.WriteString("<undefined>")
					}
					b.deferredTypings = append(b.deferredTypings, b.makeRecordType(clazz, mode))
				} else {
					b.deferredTypings = append(b.deferredTypings, b.makeInternrefType(clazz))
				}
			}
		}
		if typ.Is(types.TypeFlagNullable) {
			tsb.WriteString(" | null")
		}
		return tsb.String()
	}
	if typ == types.TypeBool {
		return "boolean"
	}
	if typ == types.TypeVoid {
		return "void"
	}
	if typ.IsNumericValue() {
		if typ.IsLongIntegerValue() {
			return "bigint"
		}
		return "number"
	}
	return "unknown"
}

func (b *TSDBuilder) getClassOrWrapper(typ *types.Type) *program.Class {
	classRef := typ.GetClassOrWrapper(b.Program)
	if classRef == nil {
		return nil
	}
	if c, ok := classRef.(*program.Class); ok {
		return c
	}
	return nil
}

func (b *TSDBuilder) makeRecordType(clazz *program.Class, mode tsdMode) string {
	var rsb strings.Builder
	members := clazz.GetMembers()
	rsb.WriteString("/** ")
	rsb.WriteString(clazz.GetInternalName())
	rsb.WriteString(" */\ndeclare interface __Record")
	rsb.WriteString(fmt.Sprintf("%d", clazz.Id()))
	rsb.WriteString("<TOmittable> {\n")
	if members != nil {
		for _, member := range members {
			if member.GetElementKind() != program.ElementKindPropertyPrototype {
				continue
			}
			pp := member.(*program.PropertyPrototype)
			property := pp.PropertyInstance
			if property == nil || !property.IsField() {
				continue
			}
			rsb.WriteString("  /** @type `")
			rsb.WriteString(property.GetResolvedType().String())
			rsb.WriteString("` */\n  ")
			rsb.WriteString(property.GetName())
			rsb.WriteString(": ")
			rsb.WriteString(b.toTypeScriptType(property.GetResolvedType(), mode))
			if b.fieldAcceptsUndefined(property.GetResolvedType()) {
				rsb.WriteString(" | TOmittable")
			}
			rsb.WriteString(";\n")
		}
	}
	rsb.WriteString("}\n")
	return rsb.String()
}

func (b *TSDBuilder) fieldAcceptsUndefined(typ *types.Type) bool {
	if typ.IsInternalReference() {
		return typ.Is(types.TypeFlagNullable)
	}
	return true
}

func (b *TSDBuilder) makeInternrefType(clazz *program.Class) string {
	var rsb strings.Builder
	rsb.WriteString("/** ")
	rsb.WriteString(clazz.GetInternalName())
	rsb.WriteString(" */\n")
	rsb.WriteString("declare class __Internref")
	rsb.WriteString(fmt.Sprintf("%d", clazz.Id()))
	rsb.WriteString(" extends Number {\n")
	base := clazz
	for base != nil {
		rsb.WriteString("  private __nominal")
		rsb.WriteString(fmt.Sprintf("%d", base.Id()))
		rsb.WriteString(": symbol;\n")
		base = base.Base
	}
	rsb.WriteString("}\n")
	return rsb.String()
}
