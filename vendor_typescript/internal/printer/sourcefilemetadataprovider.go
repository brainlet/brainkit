package printer

import (
	"github.com/brainlet/brainkit/vendor_typescript/internal/ast"
	"github.com/brainlet/brainkit/vendor_typescript/internal/tspath"
)

type SourceFileMetaDataProvider interface {
	GetSourceFileMetaData(path tspath.Path) *ast.SourceFileMetaData
}
