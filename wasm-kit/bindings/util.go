// Package bindings provides builders for various definitions describing a module.
//
// Ported from: assemblyscript/src/bindings/util.ts
package bindings

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/program"
)

// ExportsWalker is the base type for walking exported elements.
// Ported from: assemblyscript/src/bindings/util.ts ExportsWalker
type ExportsWalker struct {
	Program        *program.Program
	IncludePrivate bool
	Seen           map[program.Element]string

	// Visitor callbacks (set by the concrete builder)
	OnVisitGlobal    func(name string, element *program.Global)
	OnVisitEnum      func(name string, element *program.Enum)
	OnVisitFunction  func(name string, element *program.Function)
	OnVisitClass     func(name string, element *program.Class)
	OnVisitInterface func(name string, element *program.Interface)
	OnVisitNamespace func(name string, element program.Element)
	OnVisitAlias     func(name string, element program.Element, originalName string)
}

// NewExportsWalker creates a new ExportsWalker.
func NewExportsWalker(prog *program.Program, includePrivate bool) ExportsWalker {
	return ExportsWalker{
		Program:        prog,
		IncludePrivate: includePrivate,
		Seen:           make(map[program.Element]string),
	}
}

// Walk walks all elements and calls the respective handlers.
func (w *ExportsWalker) Walk() {
	for _, file := range w.Program.FilesByName {
		if file.Source.SourceKind == ast.SourceKindUserEntry {
			w.VisitFile(file)
		}
	}
}

// VisitFile visits all exported elements of a file.
func (w *ExportsWalker) VisitFile(file *program.File) {
	exports := file.Exports
	if exports != nil {
		for memberName, member := range exports {
			w.VisitElement(memberName, member)
		}
	}
	exportsStar := file.ExportsStar
	if exportsStar != nil {
		for _, exportStar := range exportsStar {
			w.VisitFile(exportStar)
		}
	}
}

// VisitElement visits an element.
func (w *ExportsWalker) VisitElement(name string, element program.Element) {
	if element.Is(common.CommonFlagsPrivate) && !w.IncludePrivate {
		return
	}
	seen := w.Seen
	if !element.Is(common.CommonFlagsInstance) {
		if origName, ok := seen[element]; ok {
			if w.OnVisitAlias != nil {
				w.OnVisitAlias(name, element, origName)
			}
			return
		}
	}
	seen[element] = name

	switch element.GetElementKind() {
	case program.ElementKindGlobal:
		if element.Is(common.CommonFlagsCompiled) {
			if w.OnVisitGlobal != nil {
				w.OnVisitGlobal(name, element.(*program.Global))
			}
		}
	case program.ElementKindEnum:
		if element.Is(common.CommonFlagsCompiled) {
			if w.OnVisitEnum != nil {
				w.OnVisitEnum(name, element.(*program.Enum))
			}
		}
	case program.ElementKindEnumValue:
		// handled by visitEnum
	case program.ElementKindFunctionPrototype:
		w.visitFunctionInstances(name, element.(*program.FunctionPrototype))
	case program.ElementKindClassPrototype:
		w.visitClassInstances(name, element.(*program.ClassPrototype))
	case program.ElementKindInterfacePrototype:
		w.visitInterfaceInstances(name, element.(*program.InterfacePrototype))
	case program.ElementKindPropertyPrototype:
		pp := element.(*program.PropertyPrototype)
		propertyInstance := pp.PropertyInstance
		if propertyInstance == nil {
			break
		}
		element = propertyInstance
		// fall-through
		fallthrough
	case program.ElementKindProperty:
		propertyInstance := element.(*program.Property)
		getterInstance := propertyInstance.GetterInstance
		if getterInstance != nil && w.OnVisitFunction != nil {
			w.OnVisitFunction(name, getterInstance)
		}
		setterInstance := propertyInstance.SetterInstance
		if setterInstance != nil && w.OnVisitFunction != nil {
			w.OnVisitFunction(name, setterInstance)
		}
	case program.ElementKindNamespace:
		if HasCompiledMember(element) {
			if w.OnVisitNamespace != nil {
				w.OnVisitNamespace(name, element)
			}
		}
	case program.ElementKindTypeDefinition, program.ElementKindIndexSignature:
		// skip
	default:
		// Not (directly) reachable exports:
		// File, Local, Function, Class, Interface
		panic("unexpected element kind in exports walker")
	}
}

func (w *ExportsWalker) visitFunctionInstances(name string, element *program.FunctionPrototype) {
	instances := element.Instances
	if instances != nil {
		for _, instance := range instances {
			if instance.Is(common.CommonFlagsCompiled) {
				if w.OnVisitFunction != nil {
					w.OnVisitFunction(name, instance)
				}
			}
		}
	}
}

func (w *ExportsWalker) visitClassInstances(name string, element *program.ClassPrototype) {
	instances := element.Instances
	if instances != nil {
		for _, instance := range instances {
			if instance.GetElementKind() != program.ElementKindClass {
				panic("expected class instance")
			}
			if instance.Is(common.CommonFlagsCompiled) {
				if w.OnVisitClass != nil {
					w.OnVisitClass(name, instance)
				}
			}
		}
	}
}

func (w *ExportsWalker) visitInterfaceInstances(name string, element *program.InterfacePrototype) {
	instances := element.Instances
	if instances != nil {
		for _, instance := range instances {
			if instance.GetElementKind() != program.ElementKindInterface {
				panic("expected interface instance")
			}
			if instance.Is(common.CommonFlagsCompiled) {
				if w.OnVisitInterface != nil {
					iface := instance.AsInterface()
					if iface != nil {
						w.OnVisitInterface(name, iface)
					}
				}
			}
		}
	}
}

// HasCompiledMember tests if a namespace-like element has at least one compiled member.
// Ported from: assemblyscript/src/bindings/util.ts hasCompiledMember
func HasCompiledMember(element program.Element) bool {
	members := element.GetMembers()
	if members != nil {
		for _, member := range members {
			switch member.GetElementKind() {
			case program.ElementKindFunctionPrototype:
				fp := member.(*program.FunctionPrototype)
				instances := fp.Instances
				if instances != nil {
					for _, instance := range instances {
						if instance.Is(common.CommonFlagsCompiled) {
							return true
						}
					}
				}
			case program.ElementKindClassPrototype:
				cp := member.(*program.ClassPrototype)
				instances := cp.Instances
				if instances != nil {
					for _, instance := range instances {
						if instance.Is(common.CommonFlagsCompiled) {
							return true
						}
					}
				}
			default:
				if member.Is(common.CommonFlagsCompiled) || HasCompiledMember(member) {
					return true
				}
			}
		}
	}
	return false
}
