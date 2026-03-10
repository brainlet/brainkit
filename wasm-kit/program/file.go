package program

import (
	"github.com/brainlet/brainkit/wasm-kit/ast"
	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/types"
)

// File represents a source file in the program.
type File struct {
	ElementBase
	Source          *ast.Source
	Exports         map[string]DeclaredElement
	ExportsStar     []*File
	StartFunction   *Function
	AliasNamespaces []*Namespace
}

// NewFile creates a new file element.
func NewFile(prog *Program, source *ast.Source) *File {
	f := &File{}
	InitElementBase(&f.ElementBase, ElementKindFile, source.NormalizedPath, source.InternalPath, prog, nil)
	f.Source = source
	f.parent = f // File is its own parent (special case)
	prog.FilesByName[f.internalName] = f

	startFunction := prog.MakeNativeFunction(
		"start:"+f.internalName,
		types.CreateSignature(prog, nil, types.TypeVoid, nil, 0, false),
		f,
		0,
		DecoratorFlagsNone,
	)
	startFunction.SetInternalName(startFunction.GetName())
	f.StartFunction = startFunction

	return f
}

// Add adds an element as a member, handling @global decorators and exports.
func (f *File) Add(name string, element DeclaredElement, localIdentifierIfImport *ast.IdentifierExpression) bool {
	if element.HasDecorator(DecoratorFlagsGlobal) {
		element = f.program.EnsureGlobal(name, element)
	}
	if !f.ElementBase.Add(name, element, localIdentifierIfImport) {
		return false
	}
	element = f.GetMember(name) // possibly merged locally
	if element == nil {
		return true
	}
	if element.Is(common.CommonFlagsExport) && localIdentifierIfImport == nil {
		f.EnsureExport(element.GetName(), element)
	}
	return true
}

// GetMember looks up a member by name, including re-exports.
func (f *File) GetMember(name string) DeclaredElement {
	if member := f.ElementBase.GetMember(name); member != nil {
		return member
	}
	if f.ExportsStar != nil {
		for _, file := range f.ExportsStar {
			if member := file.GetMember(name); member != nil {
				return member
			}
		}
	}
	return nil
}

// Lookup looks up an element by name.
func (f *File) Lookup(name string, isType bool) Element {
	if member := f.GetMember(name); member != nil {
		return member
	}
	return f.program.Lookup(name)
}

// EnsureExport ensures an element is an export of this file.
func (f *File) EnsureExport(name string, element DeclaredElement) {
	if f.Exports == nil {
		f.Exports = make(map[string]DeclaredElement)
	}
	f.Exports[name] = element
	if f.Source.SourceKind == ast.SourceKindLibraryEntry {
		f.program.EnsureGlobal(name, element)
	}
	for _, ns := range f.AliasNamespaces {
		ns.Add(name, element, nil)
	}
}

// EnsureExportStar ensures another file is a re-export of this file.
func (f *File) EnsureExportStar(file *File) {
	if f.ExportsStar != nil {
		for _, existing := range f.ExportsStar {
			if existing == file {
				return
			}
		}
	}
	f.ExportsStar = append(f.ExportsStar, file)
}

// LookupExport looks up an export by name.
func (f *File) LookupExport(name string) DeclaredElement {
	if f.Exports != nil {
		if elem, ok := f.Exports[name]; ok {
			return elem
		}
	}
	if f.ExportsStar != nil {
		for _, file := range f.ExportsStar {
			if elem := file.LookupExport(name); elem != nil {
				return elem
			}
		}
	}
	return nil
}

// AsAliasNamespace creates an imported namespace from this file.
func (f *File) AsAliasNamespace(name string, parent Element, localIdentifier *ast.IdentifierExpression) *Namespace {
	declaration := f.program.MakeNativeNamespaceDeclaration(name, 0)
	declaration.Name = localIdentifier
	ns := NewNamespace(name, parent, declaration, DecoratorFlagsNone)
	ns.Set(common.CommonFlagsScoped)
	f.copyExportsToNamespace(ns)
	f.AliasNamespaces = append(f.AliasNamespaces, ns)
	return ns
}

func (f *File) copyExportsToNamespace(ns *Namespace) {
	if f.Exports != nil {
		for name, member := range f.Exports {
			ns.Add(name, member, nil)
		}
	}
	if f.ExportsStar != nil {
		for _, file := range f.ExportsStar {
			file.copyExportsToNamespace(ns)
		}
	}
}

// File returns itself, since a File is its own file context.
func (f *File) File() *File {
	return f
}

// String returns a string representation.
func (f *File) String() string {
	return f.internalName
}
