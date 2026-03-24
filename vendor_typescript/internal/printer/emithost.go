package printer

import (
	"github.com/brainlet/brainkit/vendor_typescript/internal/ast"
	"github.com/brainlet/brainkit/vendor_typescript/internal/core"
	"github.com/brainlet/brainkit/vendor_typescript/internal/tsoptions"
	"github.com/brainlet/brainkit/vendor_typescript/internal/tspath"
)

// NOTE: EmitHost operations must be thread-safe
type EmitHost interface {
	Options() *core.CompilerOptions
	SourceFiles() []*ast.SourceFile
	UseCaseSensitiveFileNames() bool
	GetCurrentDirectory() string
	CommonSourceDirectory() string
	IsEmitBlocked(file string) bool
	WriteFile(fileName string, text string) error
	GetEmitModuleFormatOfFile(file ast.HasFileName) core.ModuleKind
	GetEmitResolver() EmitResolver
	GetProjectReferenceFromSource(path tspath.Path) *tsoptions.SourceOutputAndProjectReference
	IsSourceFileFromExternalLibrary(file *ast.SourceFile) bool
}
