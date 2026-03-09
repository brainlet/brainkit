package program

import (
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/wasm-kit/common"
	"github.com/brainlet/brainkit/wasm-kit/diagnostics"
)

// MangleInternalName produces a unique internal name for an element.
func MangleInternalName(name string, parent Element, isInstance bool, asGlobal bool) string {
	if parent == nil {
		return name
	}
	switch parent.GetElementKind() {
	case ElementKindFile:
		if asGlobal {
			return name
		}
		return parent.GetInternalName() + common.PATH_DELIMITER + name
	case ElementKindFunction:
		if asGlobal {
			return name
		}
		return parent.GetInternalName() + common.INNER_DELIMITER + name
	case ElementKindPropertyPrototype, ElementKindProperty:
		parent = parent.GetParent()
		fallthrough
	default:
		delim := common.STATIC_DELIMITER
		if isInstance {
			delim = common.INSTANCE_DELIMITER
		}
		return MangleInternalName(parent.GetName(), parent.GetParent(), parent.Is(common.CommonFlagsInstance), asGlobal) + delim + name
	}
}

// RegisterConcreteElement registers a concrete element instance with a program.
func RegisterConcreteElement(program *Program, element Element) {
	program.InstancesByNameMap[element.GetInternalName()] = element
}

// TryMerge attempts to merge two elements. Returns the merged element on success.
func TryMerge(older Element, newer Element) DeclaredElement {
	if newer.GetMembers() != nil {
		return nil
	}
	var merged DeclaredElement
	switch older.GetElementKind() {
	case ElementKindFunctionPrototype:
		switch newer.GetElementKind() {
		case ElementKindNamespace:
			CopyMembers(newer, older)
			merged = older.(DeclaredElement)
		case ElementKindTypeDefinition:
			if older.GetShadowType() == nil {
				older.SetShadowType(newer.(*TypeDefinition))
				CopyMembers(newer, older)
				merged = older.(DeclaredElement)
			}
		}
	case ElementKindClassPrototype, ElementKindEnum:
		if newer.GetElementKind() == ElementKindNamespace {
			CopyMembers(newer, older)
			merged = older.(DeclaredElement)
		}
	case ElementKindNamespace:
		switch newer.GetElementKind() {
		case ElementKindEnum, ElementKindClassPrototype, ElementKindFunctionPrototype:
			CopyMembers(older, newer)
			merged = newer.(DeclaredElement)
		case ElementKindNamespace:
			CopyMembers(newer, older)
			merged = older.(DeclaredElement)
		case ElementKindTypeDefinition:
			if older.GetShadowType() == nil {
				older.SetShadowType(newer.(*TypeDefinition))
				CopyMembers(newer, older)
				merged = older.(DeclaredElement)
			}
		}
	case ElementKindGlobal:
		if newer.GetElementKind() == ElementKindTypeDefinition {
			if older.GetShadowType() == nil {
				older.SetShadowType(newer.(*TypeDefinition))
				CopyMembers(newer, older)
				merged = older.(DeclaredElement)
			}
		}
	case ElementKindTypeDefinition:
		switch newer.GetElementKind() {
		case ElementKindGlobal, ElementKindFunctionPrototype, ElementKindNamespace:
			if newer.GetShadowType() == nil {
				newer.SetShadowType(older.(*TypeDefinition))
				CopyMembers(older, newer)
				merged = newer.(DeclaredElement)
			}
		}
	}
	if merged != nil {
		olderIsExport := older.Is(common.CommonFlagsExport) || older.HasDecorator(DecoratorFlagsGlobal)
		newerIsExport := newer.Is(common.CommonFlagsExport) || newer.HasDecorator(DecoratorFlagsGlobal)
		if olderIsExport != newerIsExport {
			older.GetProgram().Error(
				diagnostics.DiagnosticCodeIndividualDeclarationsInMergedDeclaration0MustBeAllExportedOrAllLocal,
				merged.IdentifierNode().GetRange(),
				merged.IdentifierNode().Text,
			)
		}
	}
	return merged
}

// CopyMembers copies the members of src to dest.
func CopyMembers(src Element, dest Element) {
	srcMembers := src.GetMembers()
	if srcMembers == nil {
		return
	}
	destMembers := dest.GetMembers()
	if destMembers == nil {
		destMembers = make(map[string]DeclaredElement)
		dest.SetMembers(destMembers)
	}
	for name, member := range srcMembers {
		destMembers[name] = member
	}
}

// cachedDefaultParameterNames caches "$0", "$1", etc.
var (
	cachedDefaultParameterNames     []string
	cachedDefaultParameterNamesLock sync.Mutex
)

// GetDefaultParameterName returns the cached default parameter name for an index.
func GetDefaultParameterName(index int32) string {
	cachedDefaultParameterNamesLock.Lock()
	defer cachedDefaultParameterNamesLock.Unlock()
	for i := int32(len(cachedDefaultParameterNames)); i <= index; i++ {
		cachedDefaultParameterNames = append(cachedDefaultParameterNames, fmt.Sprintf("$%d", i))
	}
	return cachedDefaultParameterNames[index]
}

// Memory manager constants
const (
	AlSize = 16
	AlMask = AlSize - 1
)
